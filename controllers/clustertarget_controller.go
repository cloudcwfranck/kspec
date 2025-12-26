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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
)

const (
	// HealthCheckInterval is how often to health check ClusterTargets
	HealthCheckInterval = 2 * time.Minute

	// ConditionTypeReady indicates the ClusterTarget is ready
	ConditionTypeReady = "Ready"

	// ConditionTypeCredentialsValid indicates credentials are valid
	ConditionTypeCredentialsValid = "CredentialsValid"
)

// ClusterTargetReconciler reconciles a ClusterTarget object
type ClusterTargetReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	LocalConfig   *rest.Config
	ClientFactory *clientpkg.ClusterClientFactory
}

// +kubebuilder:rbac:groups=kspec.io,resources=clustertargets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kspec.io,resources=clustertargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kspec.io,resources=clustertargets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile performs the reconciliation loop for ClusterTarget
func (r *ClusterTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("clustertarget", req.NamespacedName)

	// Fetch the ClusterTarget instance
	var clusterTarget kspecv1alpha1.ClusterTarget
	if err := r.Get(ctx, req.NamespacedName, &clusterTarget); err != nil {
		log.Info("ClusterTarget resource not found, ignoring since object must be deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Perform health check
	if err := r.healthCheck(ctx, &clusterTarget); err != nil {
		log.Error(err, "Health check failed")
		// Update status to indicate failure, but continue reconciliation
	}

	// Update status
	if err := r.Status().Update(ctx, &clusterTarget); err != nil {
		log.Error(err, "Failed to update ClusterTarget status")
		return ctrl.Result{}, err
	}

	// Requeue for periodic health checks
	return ctrl.Result{RequeueAfter: HealthCheckInterval}, nil
}

// healthCheck performs a health check on the cluster
func (r *ClusterTargetReconciler) healthCheck(ctx context.Context, clusterTarget *kspecv1alpha1.ClusterTarget) error {
	log := log.FromContext(ctx)

	now := metav1.Now()
	clusterTarget.Status.LastChecked = &now

	// Try to create clients for this cluster
	kubeClient, _, clusterInfo, err := r.ClientFactory.CreateClientsForClusterSpec(
		ctx,
		&kspecv1alpha1.ClusterSpecification{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "health-check",
				Namespace: clusterTarget.Namespace,
			},
			Spec: kspecv1alpha1.ClusterSpecificationSpec{
				ClusterRef: &kspecv1alpha1.ClusterReference{
					Name:      clusterTarget.Name,
					Namespace: clusterTarget.Namespace,
				},
			},
		},
	)

	if err != nil {
		// Cluster unreachable or credentials invalid
		clusterTarget.Status.Reachable = false
		r.setCondition(clusterTarget, ConditionTypeReady, metav1.ConditionFalse, "Unreachable", err.Error())

		// Check if it's a credential issue
		if isCredentialError(err) {
			r.setCondition(clusterTarget, ConditionTypeCredentialsValid, metav1.ConditionFalse, "InvalidCredentials", err.Error())
		}

		return fmt.Errorf("cluster unreachable: %w", err)
	}

	// Successfully connected - update status with cluster info
	clusterTarget.Status.Reachable = true
	clusterTarget.Status.UID = clusterInfo.UID
	clusterTarget.Status.Version = clusterInfo.Version

	// Detect platform if not already set
	if clusterTarget.Status.Platform == "" || clusterTarget.Status.Platform == "unknown" {
		platform := clientpkg.DetectPlatform(ctx, kubeClient)
		clusterTarget.Status.Platform = platform
	}

	// Count nodes
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		clusterTarget.Status.NodeCount = int32(len(nodes.Items))
	}

	// Set success conditions
	r.setCondition(clusterTarget, ConditionTypeReady, metav1.ConditionTrue, "ClusterReachable", "Successfully connected to cluster")
	r.setCondition(clusterTarget, ConditionTypeCredentialsValid, metav1.ConditionTrue, "CredentialsValid", "Credentials are valid")

	// Update observed generation
	clusterTarget.Status.ObservedGeneration = clusterTarget.Generation

	log.Info("Health check successful",
		"cluster", clusterTarget.Name,
		"version", clusterTarget.Status.Version,
		"platform", clusterTarget.Status.Platform,
		"nodes", clusterTarget.Status.NodeCount)

	return nil
}

// setCondition sets a condition on the ClusterTarget status
func (r *ClusterTargetReconciler) setCondition(
	clusterTarget *kspecv1alpha1.ClusterTarget,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) {
	now := metav1.Now()

	// Find existing condition
	for i, cond := range clusterTarget.Status.Conditions {
		if cond.Type == conditionType {
			// Update existing condition
			if cond.Status != status || cond.Reason != reason {
				clusterTarget.Status.Conditions[i].Status = status
				clusterTarget.Status.Conditions[i].Reason = reason
				clusterTarget.Status.Conditions[i].Message = message
				clusterTarget.Status.Conditions[i].LastTransitionTime = now
				clusterTarget.Status.Conditions[i].ObservedGeneration = clusterTarget.Generation
			}
			return
		}
	}

	// Add new condition
	clusterTarget.Status.Conditions = append(clusterTarget.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: clusterTarget.Generation,
	})
}

// isCredentialError checks if an error is related to invalid credentials
func isCredentialError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	// Check for common credential error patterns
	credErrorPatterns := []string{
		"failed to get secret",
		"secret does not contain key",
		"secret is empty",
		"failed to get kubeconfig",
		"failed to get token",
		"unauthorized",
		"forbidden",
		"authentication",
	}

	for _, pattern := range credErrorPatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager
func (r *ClusterTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kspecv1alpha1.ClusterTarget{}).
		Complete(r)
}

// NewClusterTargetReconciler creates a new ClusterTargetReconciler
func NewClusterTargetReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	localConfig *rest.Config,
	clientFactory *clientpkg.ClusterClientFactory,
) *ClusterTargetReconciler {
	return &ClusterTargetReconciler{
		Client:        client,
		Scheme:        scheme,
		LocalConfig:   localConfig,
		ClientFactory: clientFactory,
	}
}
