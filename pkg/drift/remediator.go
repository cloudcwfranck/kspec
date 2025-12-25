package drift

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudcwfranck/kspec/pkg/enforcer"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Remediator automatically fixes detected drift.
type Remediator struct {
	client        kubernetes.Interface
	dynamicClient dynamic.Interface
	enforcer      *enforcer.Enforcer
}

// NewRemediator creates a new drift remediator.
func NewRemediator(client kubernetes.Interface, dynamicClient dynamic.Interface) *Remediator {
	return &Remediator{
		client:        client,
		dynamicClient: dynamicClient,
		enforcer:      enforcer.NewEnforcer(client, dynamicClient),
	}
}

// Remediate remediates drift detected in a drift report.
func (r *Remediator) Remediate(ctx context.Context, clusterSpec *spec.ClusterSpecification, report *DriftReport, opts RemediateOptions) error {
	remediatedCount := 0
	failedCount := 0

	for i := range report.Events {
		event := &report.Events[i]

		// Skip if type not enabled for remediation
		if !r.isTypeEnabled(event.Type, opts.Types) {
			continue
		}

		// Skip if already remediated
		if event.Remediation != nil && event.Remediation.Status == DriftStatusRemediated {
			continue
		}

		// Perform remediation based on drift type
		var err error
		switch event.Type {
		case DriftTypePolicy:
			err = r.remediatePolicyDrift(ctx, clusterSpec, event, opts)
		case DriftTypeCompliance:
			// Compliance drift requires manual remediation
			event.Remediation = &RemediationResult{
				Action:    "manual-required",
				Status:    DriftStatusManualRequired,
				Timestamp: time.Now(),
				Details:   "Compliance drift requires manual intervention",
			}
		default:
			event.Remediation = &RemediationResult{
				Action:    "skipped",
				Status:    DriftStatusManualRequired,
				Timestamp: time.Now(),
				Details:   fmt.Sprintf("Remediation not supported for type %s", event.Type),
			}
		}

		// Update counters
		if err != nil {
			failedCount++
			if event.Remediation != nil {
				event.Remediation.Status = DriftStatusFailed
				event.Remediation.Error = err.Error()
			}
		} else if event.Remediation != nil && event.Remediation.Status == DriftStatusRemediated {
			remediatedCount++
		}
	}

	if failedCount > 0 {
		return fmt.Errorf("remediation completed with %d failures (%d succeeded)", failedCount, remediatedCount)
	}

	return nil
}

// remediatePolicyDrift remediates policy drift.
func (r *Remediator) remediatePolicyDrift(ctx context.Context, clusterSpec *spec.ClusterSpecification, event *DriftEvent, opts RemediateOptions) error {
	switch event.DriftKind {
	case "missing":
		return r.remediateMissingPolicy(ctx, event, opts)
	case "modified":
		return r.remediateModifiedPolicy(ctx, event, opts)
	case "extra":
		return r.remediateExtraPolicy(ctx, event, opts)
	default:
		return fmt.Errorf("unknown drift kind: %s", event.DriftKind)
	}
}

// remediateMissingPolicy creates a missing policy.
func (r *Remediator) remediateMissingPolicy(ctx context.Context, event *DriftEvent, opts RemediateOptions) error {
	if event.Expected == nil {
		return fmt.Errorf("no expected policy to create")
	}

	// Convert to unstructured for dynamic client
	unstructuredPolicy, err := runtime.DefaultUnstructuredConverter.ToUnstructured(event.Expected)
	if err != nil {
		return fmt.Errorf("failed to convert policy: %w", err)
	}

	u := &unstructured.Unstructured{Object: unstructuredPolicy}
	u.SetAPIVersion("kyverno.io/v1")
	u.SetKind("ClusterPolicy")

	policyName := u.GetName()

	// Dry-run mode
	if opts.DryRun {
		event.Remediation = &RemediationResult{
			Action:    "create",
			Status:    DriftStatusDetected,
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Would create ClusterPolicy '%s' (dry-run)", policyName),
		}
		return nil
	}

	// Create the policy
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	_, err = r.dynamicClient.Resource(gvr).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		event.Remediation = &RemediationResult{
			Action:    "create",
			Status:    DriftStatusFailed,
			Timestamp: time.Now(),
			Error:     err.Error(),
		}
		return fmt.Errorf("failed to create policy: %w", err)
	}

	event.Remediation = &RemediationResult{
		Action:    "create",
		Status:    DriftStatusRemediated,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Created ClusterPolicy '%s'", policyName),
	}

	return nil
}

// remediateModifiedPolicy updates a modified policy.
func (r *Remediator) remediateModifiedPolicy(ctx context.Context, event *DriftEvent, opts RemediateOptions) error {
	if event.Expected == nil {
		return fmt.Errorf("no expected policy to update to")
	}

	// Convert to unstructured for dynamic client
	unstructuredPolicy, err := runtime.DefaultUnstructuredConverter.ToUnstructured(event.Expected)
	if err != nil {
		return fmt.Errorf("failed to convert policy: %w", err)
	}

	u := &unstructured.Unstructured{Object: unstructuredPolicy}
	u.SetAPIVersion("kyverno.io/v1")
	u.SetKind("ClusterPolicy")

	policyName := u.GetName()

	// Dry-run mode
	if opts.DryRun {
		event.Remediation = &RemediationResult{
			Action:    "update",
			Status:    DriftStatusDetected,
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Would update ClusterPolicy '%s' (dry-run)", policyName),
		}
		return nil
	}

	// Get current policy to retrieve resourceVersion
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	existing, err := r.dynamicClient.Resource(gvr).Get(ctx, policyName, metav1.GetOptions{})
	if err != nil {
		event.Remediation = &RemediationResult{
			Action:    "update",
			Status:    DriftStatusFailed,
			Timestamp: time.Now(),
			Error:     err.Error(),
		}
		return fmt.Errorf("failed to get existing policy: %w", err)
	}

	// Set resourceVersion for update
	u.SetResourceVersion(existing.GetResourceVersion())

	// Update the policy
	_, err = r.dynamicClient.Resource(gvr).Update(ctx, u, metav1.UpdateOptions{})
	if err != nil {
		event.Remediation = &RemediationResult{
			Action:    "update",
			Status:    DriftStatusFailed,
			Timestamp: time.Now(),
			Error:     err.Error(),
		}
		return fmt.Errorf("failed to update policy: %w", err)
	}

	event.Remediation = &RemediationResult{
		Action:    "update",
		Status:    DriftStatusRemediated,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Updated ClusterPolicy '%s'", policyName),
	}

	return nil
}

// remediateExtraPolicy handles extra policies.
func (r *Remediator) remediateExtraPolicy(ctx context.Context, event *DriftEvent, opts RemediateOptions) error {
	policyName := event.Resource.Name

	// By default, we don't delete extra policies (conservative approach)
	// Only delete if Force flag is set
	if !opts.Force {
		event.Remediation = &RemediationResult{
			Action:    "skip",
			Status:    DriftStatusManualRequired,
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Extra policy '%s' not deleted (use --force to delete)", policyName),
		}
		return nil
	}

	// Dry-run mode
	if opts.DryRun {
		event.Remediation = &RemediationResult{
			Action:    "delete",
			Status:    DriftStatusDetected,
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("Would delete ClusterPolicy '%s' (dry-run)", policyName),
		}
		return nil
	}

	// Delete the policy
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	err := r.dynamicClient.Resource(gvr).Delete(ctx, policyName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		event.Remediation = &RemediationResult{
			Action:    "delete",
			Status:    DriftStatusFailed,
			Timestamp: time.Now(),
			Error:     err.Error(),
		}
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	event.Remediation = &RemediationResult{
		Action:    "delete",
		Status:    DriftStatusRemediated,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Deleted ClusterPolicy '%s'", policyName),
	}

	return nil
}

// isTypeEnabled checks if a drift type is enabled for remediation.
func (r *Remediator) isTypeEnabled(driftType DriftType, enabledTypes []DriftType) bool {
	if len(enabledTypes) == 0 {
		// By default, only auto-remediate policy drift
		return driftType == DriftTypePolicy
	}
	for _, t := range enabledTypes {
		if t == driftType {
			return true
		}
	}
	return false
}

// RemediateAll is a convenience function that detects and remediates drift in one call.
func RemediateAll(ctx context.Context, client kubernetes.Interface, dynamicClient dynamic.Interface, clusterSpec *spec.ClusterSpecification, opts RemediateOptions) (*DriftReport, error) {
	// Detect drift
	detector := NewDetector(client, dynamicClient)
	report, err := detector.Detect(ctx, clusterSpec, DetectOptions{
		EnabledTypes: opts.Types,
	})
	if err != nil {
		return nil, fmt.Errorf("drift detection failed: %w", err)
	}

	// Remediate if drift detected
	if report.Drift.Detected {
		remediator := NewRemediator(client, dynamicClient)
		if err := remediator.Remediate(ctx, clusterSpec, report, opts); err != nil {
			return report, fmt.Errorf("remediation failed: %w", err)
		}
	}

	return report, nil
}
