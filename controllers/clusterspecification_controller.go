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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/drift"
	"github.com/cloudcwfranck/kspec/pkg/enforcer"
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
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface

	// Reused kspec components
	Scanner       *scanner.Scanner
	DriftDetector *drift.Detector
	Enforcer      *enforcer.Enforcer
}

// +kubebuilder:rbac:groups=kspec.io,resources=clusterspecifications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kspec.io,resources=clusterspecifications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kspec.io,resources=clusterspecifications/finalizers,verbs=update
// +kubebuilder:rbac:groups=kspec.io,resources=compliancereports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kspec.io,resources=driftreports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kyverno.io,resources=clusterpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces;pods;serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("clusterspec", req.NamespacedName)

	// Fetch the ClusterSpecification instance
	var clusterSpec kspecv1alpha1.ClusterSpecification
	if err := r.Get(ctx, req.NamespacedName, &clusterSpec); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			log.Info("ClusterSpecification resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ClusterSpecification")
		return ctrl.Result{}, err
	}

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

	// Step 1: Run compliance scan using existing pkg/scanner
	log.Info("Running compliance scan")
	scanResult, err := r.runComplianceScan(ctx, &clusterSpec)
	if err != nil {
		log.Error(err, "Failed to run compliance scan")
		r.updateStatusFailed(ctx, &clusterSpec, err)
		return ctrl.Result{RequeueAfter: DefaultRequeueAfter}, err
	}

	// Step 2: Create ComplianceReport CR
	log.Info("Creating ComplianceReport", "passRate", calculatePassRate(scanResult.Summary))
	if err := r.createComplianceReport(ctx, &clusterSpec, scanResult); err != nil {
		log.Error(err, "Failed to create ComplianceReport")
		// Don't fail reconciliation if report creation fails
	}

	// Step 3: Detect drift using existing pkg/drift
	log.Info("Detecting drift")
	driftReport, err := r.detectDrift(ctx, &clusterSpec)
	if err != nil {
		log.Error(err, "Failed to detect drift")
		// Continue even if drift detection fails
	} else if driftReport != nil && driftReport.Drift.Detected {
		// Step 4: Create DriftReport CR
		log.Info("Drift detected, creating DriftReport", "events", len(driftReport.Events))
		if err := r.createDriftReport(ctx, &clusterSpec, driftReport); err != nil {
			log.Error(err, "Failed to create DriftReport")
		}

		// Step 5: Remediate drift using existing pkg/enforcer
		log.Info("Remediating drift")
		if err := r.remediateDrift(ctx, &clusterSpec, driftReport); err != nil {
			log.Error(err, "Failed to remediate drift")
			// Continue even if remediation fails
		}
	}

	// Step 6: Update ClusterSpecification status
	if err := r.updateStatus(ctx, &clusterSpec, scanResult, driftReport); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Step 7: Clean up old reports
	if err := r.cleanupOldReports(ctx, &clusterSpec); err != nil {
		log.Error(err, "Failed to cleanup old reports")
		// Don't fail reconciliation if cleanup fails
	}

	log.Info("Reconciliation complete", "phase", clusterSpec.Status.Phase, "score", clusterSpec.Status.ComplianceScore)

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

	// TODO: Optionally remove enforced policies
	// For now, we leave policies in place for safety
	// In production, this could be configurable

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
func (r *ClusterSpecReconciler) runComplianceScan(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification) (*scanner.ScanResult, error) {
	// Convert ClusterSpecification to spec.ClusterSpecification
	specToScan := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    clusterSpec.Name,
			Version: clusterSpec.ResourceVersion,
		},
		Spec: clusterSpec.Spec.SpecFields,
	}

	// Run scan using existing scanner
	result, err := r.Scanner.Scan(ctx, specToScan)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return result, nil
}

// detectDrift detects drift using the existing drift detector
func (r *ClusterSpecReconciler) detectDrift(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification) (*drift.DriftReport, error) {
	// Convert ClusterSpecification to spec.ClusterSpecification
	specToCheck := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    clusterSpec.Name,
			Version: clusterSpec.ResourceVersion,
		},
		Spec: clusterSpec.Spec.SpecFields,
	}

	// Detect drift using existing detector
	opts := drift.DetectOptions{
		EnabledTypes: []drift.DriftType{
			drift.DriftTypePolicy,
			drift.DriftTypeCompliance,
		},
	}

	driftReport, err := r.DriftDetector.Detect(ctx, specToCheck, opts)
	if err != nil {
		return nil, fmt.Errorf("drift detection failed: %w", err)
	}

	return driftReport, nil
}

// remediateDrift remediates detected drift

// remediateDrift remediates detected drift
func (r *ClusterSpecReconciler) remediateDrift(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, driftReport *drift.DriftReport) error {
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

	_, err := drift.RemediateAll(ctx, r.KubeClient, r.DynamicClient, specToRemediate, remediateOpts)
	if err != nil {
		return fmt.Errorf("drift remediation failed: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterSpecReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kspecv1alpha1.ClusterSpecification{}).
		Owns(&kspecv1alpha1.ComplianceReport{}).
		Owns(&kspecv1alpha1.DriftReport{}).
		Complete(r)
}

// NewClusterSpecReconciler creates a new ClusterSpecReconciler
func NewClusterSpecReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	kubeClient kubernetes.Interface,
	dynamicClient dynamic.Interface,
) *ClusterSpecReconciler {
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

	return &ClusterSpecReconciler{
		Client:        client,
		Scheme:        scheme,
		KubeClient:    kubeClient,
		DynamicClient: dynamicClient,
		Scanner:       scanner.NewScanner(kubeClient, checkList),
		DriftDetector: drift.NewDetector(kubeClient, dynamicClient),
		Enforcer:      enforcer.NewEnforcer(kubeClient, dynamicClient),
	}
}
