/*
Copyright 2025 kspec contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/alerts"
	"github.com/cloudcwfranck/kspec/pkg/audit"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
	"github.com/cloudcwfranck/kspec/pkg/drift"
	"github.com/cloudcwfranck/kspec/pkg/enforcer/kyverno"
	"github.com/cloudcwfranck/kspec/pkg/metrics"
	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/scanner/checks"
	"github.com/cloudcwfranck/kspec/pkg/spec"
)

const (
	// FinalizerName is the finalizer added to ClusterSpecifications
	FinalizerName = "kspec.io/finalizer"

	// DefaultRequeueAfter is the default reconciliation interval
	DefaultRequeueAfter = 5 * time.Minute

	// ReportNamespace is the namespace where reports are created
	ReportNamespace = "kspec-system"

	// MaxReportsToKeep is the maximum number of reports to retain per ClusterSpec
	MaxReportsToKeep = 30
)

// ClusterSpecReconciler reconciles a ClusterSpecification object
type ClusterSpecReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	LocalConfig   *rest.Config
	ClientFactory *clientpkg.ClusterClientFactory
	AlertManager  *alerts.Manager
}

// +kubebuilder:rbac:groups=kspec.io,resources=clusterspecifications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kspec.io,resources=clusterspecifications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kspec.io,resources=clusterspecifications/finalizers,verbs=update
// +kubebuilder:rbac:groups=kspec.io,resources=compliancereports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kspec.io,resources=driftreports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kyverno.io,resources=clusterpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates/status,verbs=get
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces;pods;serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("clusterspec", req.NamespacedName)
	auditLog := audit.NewLogger(ctx)

	// Track reconciliation duration
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordReconcileDuration("clusterspec", req.Name, duration)
	}()

	// Fetch the ClusterSpecification instance
	var clusterSpec kspecv1alpha1.ClusterSpecification
	if err := r.Get(ctx, req.NamespacedName, &clusterSpec); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			log.Info("ClusterSpecification resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ClusterSpecification")
		metrics.RecordReconcileError("clusterspec", req.Name, "get_resource_failed")
		return ctrl.Result{}, err
	}

	// Record reconciliation attempt
	metrics.RecordReconcile("clusterspec", clusterSpec.Name)

	// Handle deletion
	if !clusterSpec.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &clusterSpec)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&clusterSpec, FinalizerName) {
		controllerutil.AddFinalizer(&clusterSpec, FinalizerName)
		if err := r.Update(ctx, &clusterSpec); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Update status to indicate reconciliation is in progress
	if clusterSpec.Status.Phase == "" {
		clusterSpec.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, &clusterSpec); err != nil {
			log.Error(err, "Failed to update status to Pending")
			return ctrl.Result{}, err
		}
	}

	// NEW: Create clients for target cluster (local or remote)
	kubeClient, dynamicClient, clusterInfo, err := r.ClientFactory.CreateClientsForClusterSpec(ctx, &clusterSpec)
	if err != nil {
		log.Error(err, "Failed to create cluster clients", "clusterRef", clusterSpec.Spec.ClusterRef)
		r.updateStatusFailed(ctx, &clusterSpec, fmt.Errorf("cluster unreachable: %w", err))
		return ctrl.Result{RequeueAfter: DefaultRequeueAfter}, err
	}

	log.Info("Reconciling cluster",
		"cluster", clusterInfo.Name,
		"isLocal", clusterInfo.IsLocal,
		"allowEnforcement", clusterInfo.AllowEnforcement)

	// Step 1: Run compliance scan using existing pkg/scanner
	log.Info("Running compliance scan")
	scanStartTime := time.Now()
	scanResult, err := r.runComplianceScan(ctx, &clusterSpec, kubeClient)
	scanDuration := time.Since(scanStartTime).Seconds()

	// Record scan metrics and audit log
	if err != nil {
		log.Error(err, "Failed to run compliance scan")
		auditLog.LogComplianceScan(clusterInfo.Name, clusterInfo.UID, clusterSpec.Name, 0, 0, 0, err)
		metrics.RecordReconcileError("clusterspec", clusterSpec.Name, "scan_failed")
		r.updateStatusFailed(ctx, &clusterSpec, err)
		return ctrl.Result{RequeueAfter: DefaultRequeueAfter}, err
	}

	// Record successful scan metrics
	metrics.RecordScanDuration(clusterInfo.Name, clusterSpec.Name, scanDuration)
	metrics.RecordComplianceMetrics(
		clusterInfo.Name,
		clusterInfo.UID,
		clusterSpec.Name,
		scanResult.Summary.TotalChecks,
		scanResult.Summary.Passed,
		scanResult.Summary.Failed,
	)
	auditLog.LogComplianceScan(
		clusterInfo.Name,
		clusterInfo.UID,
		clusterSpec.Name,
		scanResult.Summary.TotalChecks,
		scanResult.Summary.Passed,
		scanResult.Summary.Failed,
		nil,
	)

	// Step 2: Create ComplianceReport CR
	log.Info("Creating ComplianceReport", "passRate", calculatePassRate(scanResult.Summary))
	if err := r.createComplianceReport(ctx, &clusterSpec, scanResult, clusterInfo); err != nil {
		log.Error(err, "Failed to create ComplianceReport")
		auditLog.LogReportGeneration("ComplianceReport", "", clusterInfo.Name, err)
		// Don't fail reconciliation if report creation fails
	}

	// Send compliance alert if score is below threshold (default: 80%)
	complianceScore := calculatePassRate(scanResult.Summary)
	complianceThreshold := 80
	if complianceScore < complianceThreshold {
		r.sendComplianceAlert(ctx, &clusterSpec, clusterInfo, scanResult, complianceScore)
	}

	// Step 3: Detect drift using existing pkg/drift
	log.Info("Detecting drift")
	driftReport, err := r.detectDrift(ctx, &clusterSpec, kubeClient, dynamicClient)
	if err != nil {
		log.Error(err, "Failed to detect drift")
		auditLog.LogDriftDetection(clusterInfo.Name, clusterInfo.UID, clusterSpec.Name, false, 0, err)
		// Continue even if drift detection fails
	} else if driftReport != nil {
		// Record drift metrics
		eventCount := len(driftReport.Events)
		eventsByType := make(map[string]int)
		for _, event := range driftReport.Events {
			eventsByType[event.DriftKind]++
		}
		metrics.RecordDriftMetrics(
			clusterInfo.Name,
			clusterInfo.UID,
			clusterSpec.Name,
			driftReport.Drift.Detected,
			eventCount,
			eventsByType,
		)
		auditLog.LogDriftDetection(
			clusterInfo.Name,
			clusterInfo.UID,
			clusterSpec.Name,
			driftReport.Drift.Detected,
			eventCount,
			nil,
		)

		if driftReport.Drift.Detected {
			// Step 4: Create DriftReport CR
			log.Info("Drift detected, creating DriftReport", "events", len(driftReport.Events))
			if err := r.createDriftReport(ctx, &clusterSpec, driftReport, clusterInfo); err != nil {
				log.Error(err, "Failed to create DriftReport")
				auditLog.LogReportGeneration("DriftReport", "", clusterInfo.Name, err)
			}

			// Send drift detection alert
			r.sendDriftAlert(ctx, &clusterSpec, clusterInfo, driftReport)

			// Step 5: Remediate drift (only if allowed by cluster policy)
			if clusterInfo.AllowEnforcement {
				log.Info("Remediating drift")
				if err := r.remediateDrift(ctx, &clusterSpec, driftReport, kubeClient, dynamicClient, clusterInfo, auditLog); err != nil {
					log.Error(err, "Failed to remediate drift")
					// Continue even if remediation fails
				} else {
					// Send remediation success alert
					r.sendRemediationAlert(ctx, &clusterSpec, clusterInfo, driftReport)
				}
			} else {
				log.Info("Skipping drift remediation (enforcement not allowed on this cluster)")
			}
		}
	}

	// Step 5.5: Manage policy enforcement (v0.3.0)
	policiesGenerated := 0
	if clusterInfo.AllowEnforcement {
		log.Info("Managing policy enforcement")
		if err := r.managePolicyEnforcement(ctx, &clusterSpec, dynamicClient); err != nil {
			log.Error(err, "Failed to manage policy enforcement")
			// Continue even if policy enforcement fails (non-fatal)
		} else {
			// Count generated policies for status
			if clusterSpec.Spec.Enforcement != nil && clusterSpec.Spec.Enforcement.Enabled {
				generator := kyverno.NewGenerator()
				specForCounting := &spec.ClusterSpecification{
					Metadata: spec.Metadata{Name: clusterSpec.Name},
					Spec:     clusterSpec.Spec.SpecFields,
				}
				policies, _ := generator.GeneratePolicies(specForCounting)
				policiesGenerated = len(policies)
			}
		}
	} else {
		log.Info("Skipping policy enforcement (enforcement not allowed on this cluster)")
	}

	// Update enforcement status
	r.updateEnforcementStatus(ctx, &clusterSpec, policiesGenerated)

	// Step 5.6: Manage webhook certificates (v0.3.0 Phase 2)
	certificateReady := false
	if clusterInfo.AllowEnforcement {
		log.Info("Managing webhook certificates")
		certReady, err := r.manageCertificate(ctx, &clusterSpec, dynamicClient)
		if err != nil {
			log.Error(err, "Failed to manage certificate")
			// Continue even if certificate management fails (non-fatal)
		}
		certificateReady = certReady
	} else {
		log.Info("Skipping certificate management (enforcement not allowed on this cluster)")
	}

	// Update webhook status
	r.updateWebhookStatus(ctx, &clusterSpec, certificateReady)

	// Step 5.7: Manage ValidatingWebhookConfiguration (v0.3.0 Phase 3)
	if clusterInfo.AllowEnforcement {
		log.Info("Managing ValidatingWebhookConfiguration")
		if err := r.manageValidatingWebhook(ctx, &clusterSpec); err != nil {
			log.Error(err, "Failed to manage ValidatingWebhookConfiguration")
			// Continue even if webhook config management fails (non-fatal)
		}
	} else {
		log.Info("Skipping webhook configuration (enforcement not allowed on this cluster)")
	}

	// Step 6: Update ClusterSpecification status
	if err := r.updateStatus(ctx, &clusterSpec, scanResult, driftReport); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Step 7: Clean up old reports
	if err := r.cleanupOldReports(ctx, &clusterSpec, clusterInfo); err != nil {
		log.Error(err, "Failed to cleanup old reports")
		// Don't fail reconciliation if cleanup fails
	}

	log.Info("Reconciliation complete",
		"cluster", clusterInfo.Name,
		"phase", clusterSpec.Status.Phase,
		"score", clusterSpec.Status.ComplianceScore)

	// Requeue after configured interval for continuous monitoring
	return ctrl.Result{RequeueAfter: DefaultRequeueAfter}, nil
}

// handleDeletion handles cleanup when ClusterSpecification is deleted
func (r *ClusterSpecReconciler) handleDeletion(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(clusterSpec, FinalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("Handling deletion, cleaning up resources")

	// Clean up ComplianceReports
	// Note: We can't use owner references because ClusterSpecification is cluster-scoped
	// while reports are namespaced. We use labels instead.
	var complianceReports kspecv1alpha1.ComplianceReportList
	if err := r.List(ctx, &complianceReports,
		client.InNamespace(ReportNamespace),
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpec.Name,
		},
	); err != nil {
		log.Error(err, "Failed to list ComplianceReports for cleanup")
	} else {
		for i := range complianceReports.Items {
			if err := r.Delete(ctx, &complianceReports.Items[i]); err != nil {
				log.Error(err, "Failed to delete ComplianceReport", "name", complianceReports.Items[i].Name)
			}
		}
		log.Info("Cleaned up ComplianceReports", "count", len(complianceReports.Items))
	}

	// Clean up DriftReports
	var driftReports kspecv1alpha1.DriftReportList
	if err := r.List(ctx, &driftReports,
		client.InNamespace(ReportNamespace),
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpec.Name,
		},
	); err != nil {
		log.Error(err, "Failed to list DriftReports for cleanup")
	} else {
		for i := range driftReports.Items {
			if err := r.Delete(ctx, &driftReports.Items[i]); err != nil {
				log.Error(err, "Failed to delete DriftReport", "name", driftReports.Items[i].Name)
			}
		}
		log.Info("Cleaned up DriftReports", "count", len(driftReports.Items))
	}

	// Clean up policies and certificates (v0.3.0)
	// Create clients for cleanup
	_, dynamicClient, _, err := r.ClientFactory.CreateClientsForClusterSpec(ctx, clusterSpec)
	if err != nil {
		log.Error(err, "Failed to create clients for cleanup")
		// Continue even if we can't clean up policies/certificates
	} else {
		// Clean up policies
		if err := r.cleanupPolicies(ctx, clusterSpec, dynamicClient); err != nil {
			log.Error(err, "Failed to cleanup policies")
			// Continue even if cleanup fails
		}

		// Clean up certificate (Phase 2)
		if err := r.cleanupCertificate(ctx, dynamicClient); err != nil {
			log.Error(err, "Failed to cleanup certificate")
			// Continue even if cleanup fails
		}
	}

	// Clean up ValidatingWebhookConfiguration (Phase 3)
	if err := r.cleanupValidatingWebhook(ctx); err != nil {
		log.Error(err, "Failed to cleanup ValidatingWebhookConfiguration")
		// Continue even if cleanup fails
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(clusterSpec, FinalizerName)
	if err := r.Update(ctx, clusterSpec); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Finalizer removed, deletion complete")
	return ctrl.Result{}, nil
}

// runComplianceScan runs a compliance scan using the existing scanner
func (r *ClusterSpecReconciler) runComplianceScan(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, kubeClient kubernetes.Interface) (*scanner.ScanResult, error) {
	// Convert ClusterSpecification to spec.ClusterSpecification
	specToScan := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    clusterSpec.Name,
			Version: clusterSpec.ResourceVersion,
		},
		Spec: clusterSpec.Spec.SpecFields,
	}

	// Create scanner with all checks
	checkList := []scanner.Check{
		&checks.KubernetesVersionCheck{},
		&checks.PodSecurityStandardsCheck{},
		&checks.NetworkPolicyCheck{},
		&checks.WorkloadSecurityCheck{},
		&checks.RBACCheck{},
		&checks.AdmissionCheck{},
		&checks.ObservabilityCheck{},
	}

	scannerInstance := scanner.NewScanner(kubeClient, checkList)

	// Run scan using scanner
	result, err := scannerInstance.Scan(ctx, specToScan)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return result, nil
}

// detectDrift detects drift using the existing drift detector
func (r *ClusterSpecReconciler) detectDrift(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) (*drift.DriftReport, error) {
	// Convert ClusterSpecification to spec.ClusterSpecification
	specToCheck := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    clusterSpec.Name,
			Version: clusterSpec.ResourceVersion,
		},
		Spec: clusterSpec.Spec.SpecFields,
	}

	// Create drift detector
	driftDetector := drift.NewDetector(kubeClient, dynamicClient)

	// Detect drift using detector
	opts := drift.DetectOptions{
		EnabledTypes: []drift.DriftType{
			drift.DriftTypePolicy,
			drift.DriftTypeCompliance,
		},
	}

	driftReport, err := driftDetector.Detect(ctx, specToCheck, opts)
	if err != nil {
		return nil, fmt.Errorf("drift detection failed: %w", err)
	}

	return driftReport, nil
}

// remediateDrift remediates detected drift
func (r *ClusterSpecReconciler) remediateDrift(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, driftReport *drift.DriftReport, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, clusterInfo *clientpkg.ClusterInfo, auditLog *audit.Logger) error {
	// Convert to spec.ClusterSpecification
	specToRemediate := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    clusterSpec.Name,
			Version: clusterSpec.ResourceVersion,
		},
		Spec: clusterSpec.Spec.SpecFields,
	}

	// Remediate using existing drift.RemediateAll
	remediateOpts := drift.RemediateOptions{
		DryRun: false,
		Types:  []drift.DriftType{drift.DriftTypePolicy}, // Only auto-remediate policy drift
	}

	_, err := drift.RemediateAll(ctx, kubeClient, dynamicClient, specToRemediate, remediateOpts)
	if err != nil {
		metrics.RecordRemediationError(clusterInfo.Name, clusterInfo.UID, clusterSpec.Name, "remediation_failed")
		return fmt.Errorf("drift remediation failed: %w", err)
	}

	// Record remediation metrics for each event
	for _, event := range driftReport.Events {
		if event.Resource.Kind != "" {
			action := "remediate_" + event.DriftKind
			metrics.RecordRemediationAction(clusterInfo.Name, clusterInfo.UID, clusterSpec.Name, action)
			auditLog.LogRemediation(
				clusterInfo.Name,
				clusterInfo.UID,
				clusterSpec.Name,
				event.Resource.Kind,
				event.Resource.Name,
				action,
				nil,
			)
		}
	}

	// Log summary
	auditLog.LogRemediation(
		clusterInfo.Name,
		clusterInfo.UID,
		clusterSpec.Name,
		"drift",
		"all",
		"remediate_all",
		nil,
	)

	return nil
}

// sendComplianceAlert sends an alert when compliance score is below threshold
func (r *ClusterSpecReconciler) sendComplianceAlert(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, clusterInfo *clientpkg.ClusterInfo, scanResult *scanner.ScanResult, score int) {
	if r.AlertManager == nil {
		return
	}

	log := log.FromContext(ctx)
	alert := alerts.Alert{
		Level:       alerts.AlertLevelWarning,
		Title:       "Compliance score below threshold",
		Description: fmt.Sprintf("Cluster %s compliance score is %d%% (threshold: 80%%)", clusterInfo.Name, score),
		Source:      fmt.Sprintf("ClusterSpec/%s", clusterSpec.Name),
		EventType:   "ComplianceFailure",
		Labels: map[string]string{
			"cluster":      clusterInfo.Name,
			"cluster_uid":  clusterInfo.UID,
			"spec":         clusterSpec.Name,
			"platform":     clusterInfo.Platform,
		},
		Metadata: map[string]interface{}{
			"score":        score,
			"total_checks": scanResult.Summary.TotalChecks,
			"passed":       scanResult.Summary.Passed,
			"failed":       scanResult.Summary.Failed,
			"cluster":      clusterInfo.Name,
		},
	}

	if err := r.AlertManager.Send(ctx, alert); err != nil {
		log.Error(err, "Failed to send compliance alert", "cluster", clusterInfo.Name, "score", score)
	}
}

// sendDriftAlert sends an alert when drift is detected
func (r *ClusterSpecReconciler) sendDriftAlert(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, clusterInfo *clientpkg.ClusterInfo, driftReport *drift.DriftReport) {
	if r.AlertManager == nil {
		return
	}

	log := log.FromContext(ctx)
	eventCount := len(driftReport.Events)

	// Build description with drift details
	description := fmt.Sprintf("Detected %d drift event(s) in cluster %s", eventCount, clusterInfo.Name)
	if eventCount > 0 {
		description += "\n\nDrift events:"
		for i, event := range driftReport.Events {
			if i >= 5 {
				description += fmt.Sprintf("\n... and %d more", eventCount-5)
				break
			}
			description += fmt.Sprintf("\n- %s: %s/%s", event.DriftKind, event.Resource.Kind, event.Resource.Name)
		}
	}

	alert := alerts.Alert{
		Level:       alerts.AlertLevelCritical,
		Title:       "Configuration drift detected",
		Description: description,
		Source:      fmt.Sprintf("ClusterSpec/%s", clusterSpec.Name),
		EventType:   "DriftDetected",
		Labels: map[string]string{
			"cluster":     clusterInfo.Name,
			"cluster_uid": clusterInfo.UID,
			"spec":        clusterSpec.Name,
			"platform":    clusterInfo.Platform,
		},
		Metadata: map[string]interface{}{
			"event_count": eventCount,
			"cluster":     clusterInfo.Name,
		},
	}

	if err := r.AlertManager.Send(ctx, alert); err != nil {
		log.Error(err, "Failed to send drift alert", "cluster", clusterInfo.Name, "events", eventCount)
	}
}

// sendRemediationAlert sends an alert when drift remediation is performed
func (r *ClusterSpecReconciler) sendRemediationAlert(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, clusterInfo *clientpkg.ClusterInfo, driftReport *drift.DriftReport) {
	if r.AlertManager == nil {
		return
	}

	log := log.FromContext(ctx)
	eventCount := len(driftReport.Events)

	alert := alerts.Alert{
		Level:       alerts.AlertLevelInfo,
		Title:       "Drift remediation performed",
		Description: fmt.Sprintf("Successfully remediated %d drift event(s) in cluster %s", eventCount, clusterInfo.Name),
		Source:      fmt.Sprintf("ClusterSpec/%s", clusterSpec.Name),
		EventType:   "RemediationPerformed",
		Labels: map[string]string{
			"cluster":     clusterInfo.Name,
			"cluster_uid": clusterInfo.UID,
			"spec":        clusterSpec.Name,
			"platform":    clusterInfo.Platform,
		},
		Metadata: map[string]interface{}{
			"event_count": eventCount,
			"cluster":     clusterInfo.Name,
		},
	}

	if err := r.AlertManager.Send(ctx, alert); err != nil {
		log.Error(err, "Failed to send remediation alert", "cluster", clusterInfo.Name)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterSpecReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Note: We don't use Owns() for reports because ClusterSpecification is cluster-scoped
	// while reports are namespaced, so owner references don't work. Cleanup is handled via finalizers.
	return ctrl.NewControllerManagedBy(mgr).
		For(&kspecv1alpha1.ClusterSpecification{}).
		Complete(r)
}

// NewClusterSpecReconciler creates a new ClusterSpecReconciler
func NewClusterSpecReconciler(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	localConfig *rest.Config,
	clientFactory *clientpkg.ClusterClientFactory,
	alertManager *alerts.Manager,
) *ClusterSpecReconciler {
	return &ClusterSpecReconciler{
		Client:        k8sClient,
		Scheme:        scheme,
		LocalConfig:   localConfig,
		ClientFactory: clientFactory,
		AlertManager:  alertManager,
	}
}
