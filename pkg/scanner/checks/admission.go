package checks

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AdmissionCheck validates admission controller requirements.
type AdmissionCheck struct{}

// Name returns the check name.
func (c *AdmissionCheck) Name() string {
	return "admission.controllers"
}

// Run executes the admission controller check.
func (c *AdmissionCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Skip if not specified
	if clusterSpec.Spec.Admission == nil {
		return &scanner.CheckResult{
			Name:    c.Name(),
			Status:  scanner.StatusSkip,
			Message: "Admission controller requirements not specified in cluster spec",
		}, nil
	}

	violations := []string{}
	evidence := make(map[string]interface{})

	// Check required admission webhooks
	if len(clusterSpec.Spec.Admission.Required) > 0 {
		webhookViolations, err := c.checkRequiredWebhooks(ctx, client, clusterSpec.Spec.Admission.Required)
		if err != nil {
			return nil, err
		}
		violations = append(violations, webhookViolations...)
	}

	// Check policy requirements (Kyverno)
	if clusterSpec.Spec.Admission.Policies != nil {
		policyViolations, policyEvidence, err := c.checkPolicies(ctx, client, clusterSpec.Spec.Admission.Policies)
		if err != nil {
			// Policies might not be available if Kyverno is not installed
			// Treat this as a violation, not an error
			violations = append(violations, fmt.Sprintf("Unable to check policies: %v (Kyverno may not be installed)", err))
		} else {
			violations = append(violations, policyViolations...)
			for k, v := range policyEvidence {
				evidence[k] = v
			}
		}
	}

	if len(violations) > 0 {
		evidence["violations"] = violations
		evidence["violation_count"] = len(violations)

		severity := scanner.SeverityHigh
		// If only policy violations and Kyverno is not installed, reduce severity
		if len(violations) == 1 && len(clusterSpec.Spec.Admission.Required) == 0 {
			severity = scanner.SeverityMedium
		}

		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: severity,
			Message:  fmt.Sprintf("Found %d admission controller violations", len(violations)),
			Evidence: evidence,
			Remediation: `Review and fix admission controller violations:
1. Install required admission controllers (e.g., Kyverno)
   helm install kyverno kyverno/kyverno --namespace kyverno --create-namespace
2. Ensure required ValidatingWebhookConfigurations exist
3. Deploy required policies (ClusterPolicy resources)
4. Verify minimum policy count is met

Example: Create a Kyverno policy:
kubectl apply -f https://raw.githubusercontent.com/kyverno/policies/main/pod-security/baseline/disallow-privileged-containers/disallow-privileged-containers.yaml`,
		}, nil
	}

	return &scanner.CheckResult{
		Name:    c.Name(),
		Status:  scanner.StatusPass,
		Message: "Admission controller requirements satisfied",
		Evidence: evidence,
	}, nil
}

// checkRequiredWebhooks validates required admission webhooks exist.
func (c *AdmissionCheck) checkRequiredWebhooks(ctx context.Context, client kubernetes.Interface, requirements []spec.AdmissionRequirement) ([]string, error) {
	violations := []string{}

	// Get ValidatingWebhookConfigurations
	validatingWebhooks, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list validating webhooks: %w", err)
	}

	// Get MutatingWebhookConfigurations
	mutatingWebhooks, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list mutating webhooks: %w", err)
	}

	// Check each requirement
	for _, req := range requirements {
		count := 0

		switch req.Type {
		case "ValidatingWebhookConfiguration":
			count = c.countMatchingWebhooks(validatingWebhooks.Items, req.NamePattern)
		case "MutatingWebhookConfiguration":
			count = c.countMatchingMutatingWebhooks(mutatingWebhooks.Items, req.NamePattern)
		default:
			violations = append(violations, fmt.Sprintf("Unknown webhook type: %s", req.Type))
			continue
		}

		if count < req.MinCount {
			violations = append(violations, fmt.Sprintf("Required %s with pattern '%s': found %d, need %d",
				req.Type, req.NamePattern, count, req.MinCount))
		}
	}

	return violations, nil
}

// countMatchingWebhooks counts ValidatingWebhookConfigurations matching a pattern.
func (c *AdmissionCheck) countMatchingWebhooks(webhooks []admissionv1.ValidatingWebhookConfiguration, pattern string) int {
	count := 0
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0
	}

	for _, webhook := range webhooks {
		if re.MatchString(webhook.Name) {
			count++
		}
	}
	return count
}

// countMatchingMutatingWebhooks counts MutatingWebhookConfigurations matching a pattern.
func (c *AdmissionCheck) countMatchingMutatingWebhooks(webhooks []admissionv1.MutatingWebhookConfiguration, pattern string) int {
	count := 0
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0
	}

	for _, webhook := range webhooks {
		if re.MatchString(webhook.Name) {
			count++
		}
	}
	return count
}

// checkPolicies validates Kyverno policy requirements.
func (c *AdmissionCheck) checkPolicies(ctx context.Context, client kubernetes.Interface, policySpec *spec.PolicySpec) ([]string, map[string]interface{}, error) {
	violations := []string{}
	evidence := make(map[string]interface{})

	// Get dynamic client for Kyverno CRDs
	config, err := rest.InClusterConfig()
	if err != nil {
		// Try kubeconfig if not in-cluster
		// For testing, this will fail gracefully
		return violations, evidence, fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return violations, evidence, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define Kyverno ClusterPolicy GVR
	clusterPolicyGVR := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	// List ClusterPolicies
	policies, err := dynamicClient.Resource(clusterPolicyGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return violations, evidence, fmt.Errorf("failed to list cluster policies: %w", err)
	}

	policyCount := len(policies.Items)
	evidence["policy_count"] = policyCount

	// Check minimum count
	if policyCount < policySpec.MinCount {
		violations = append(violations, fmt.Sprintf("Cluster has %d policies, requires at least %d", policyCount, policySpec.MinCount))
	}

	// Check required policies exist
	if len(policySpec.RequiredPolicies) > 0 {
		policyNames := make(map[string]bool)
		for _, policy := range policies.Items {
			policyNames[policy.GetName()] = true
		}

		missingPolicies := []string{}
		for _, required := range policySpec.RequiredPolicies {
			if !policyNames[required.Name] {
				missingPolicies = append(missingPolicies, required.Name)
			}
		}

		if len(missingPolicies) > 0 {
			violations = append(violations, fmt.Sprintf("Missing required policies: %v", missingPolicies))
			evidence["missing_policies"] = missingPolicies
		}
	}

	return violations, evidence, nil
}
