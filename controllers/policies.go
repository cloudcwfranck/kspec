package controllers

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/enforcer/kyverno"
	"github.com/cloudcwfranck/kspec/pkg/spec"
)

// managePolicyEnforcement handles policy generation and application
func (r *ClusterSpecReconciler) managePolicyEnforcement(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	dynamicClient dynamic.Interface,
) error {
	log := log.FromContext(ctx)

	// Check if enforcement is enabled
	if clusterSpec.Spec.Enforcement == nil || !clusterSpec.Spec.Enforcement.Enabled {
		log.V(1).Info("Enforcement disabled, skipping policy generation")
		return nil
	}

	// Determine validation failure action based on mode
	mode := clusterSpec.Spec.Enforcement.Mode
	if mode == "" {
		mode = "monitor"
	}

	log.Info("Managing policy enforcement", "mode", mode)

	// Generate policies from ClusterSpec
	generator := kyverno.NewGenerator()
	specForGeneration := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    clusterSpec.Name,
			Version: clusterSpec.ResourceVersion,
		},
		Spec: clusterSpec.Spec.SpecFields,
	}

	policies, err := generator.GeneratePolicies(specForGeneration)
	if err != nil {
		return fmt.Errorf("failed to generate policies: %w", err)
	}

	log.Info("Generated policies", "count", len(policies))

	// Apply policies to cluster
	policiesApplied := 0
	for _, policyObj := range policies {
		policy, ok := policyObj.(*kyverno.ClusterPolicy)
		if !ok {
			log.Error(fmt.Errorf("unexpected policy type"), "Skipping policy")
			continue
		}

		// Set validation failure action based on enforcement mode
		switch mode {
		case "monitor":
			// In monitor mode, don't apply policies at all
			log.V(1).Info("Monitor mode: not applying policy", "policy", policy.Name)
			continue
		case "audit":
			policy.Spec.ValidationFailureAction = kyverno.Audit
		case "enforce":
			policy.Spec.ValidationFailureAction = kyverno.Enforce
		default:
			log.Info("Unknown enforcement mode, defaulting to audit", "mode", mode)
			policy.Spec.ValidationFailureAction = kyverno.Audit
		}

		// Add ownership labels for tracking
		if policy.Labels == nil {
			policy.Labels = make(map[string]string)
		}
		policy.Labels["kspec.io/cluster-spec"] = clusterSpec.Name
		policy.Labels["kspec.io/generated"] = "true"
		policy.Labels["kspec.io/enforcement-mode"] = mode

		// Convert to unstructured for dynamic client
		unstructuredPolicy, err := runtime.DefaultUnstructuredConverter.ToUnstructured(policy)
		if err != nil {
			return fmt.Errorf("failed to convert policy to unstructured: %w", err)
		}

		u := &unstructured.Unstructured{Object: unstructuredPolicy}
		u.SetGroupVersionKind(policy.GroupVersionKind())

		// Apply policy using dynamic client
		policyResource := dynamicClient.Resource(kyverno.ClusterPolicyGVR())
		_, err = policyResource.Create(ctx, u, metav1.CreateOptions{})
		if err != nil {
			// If already exists, update it
			if client.IgnoreAlreadyExists(err) == nil {
				_, err = policyResource.Update(ctx, u, metav1.UpdateOptions{})
				if err != nil {
					log.Error(err, "Failed to update policy", "policy", policy.Name)
					continue
				}
			} else {
				log.Error(err, "Failed to create policy", "policy", policy.Name)
				continue
			}
		}

		policiesApplied++
		log.V(1).Info("Applied policy", "policy", policy.Name, "mode", mode)
	}

	log.Info("Policy enforcement complete", "applied", policiesApplied, "total", len(policies))
	return nil
}

// cleanupPolicies removes policies generated for this ClusterSpec
func (r *ClusterSpecReconciler) cleanupPolicies(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	dynamicClient dynamic.Interface,
) error {
	log := log.FromContext(ctx)

	// List all ClusterPolicies with our label
	policyResource := dynamicClient.Resource(kyverno.ClusterPolicyGVR())
	policyList, err := policyResource.List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kspec.io/cluster-spec=%s", clusterSpec.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to list policies: %w", err)
	}

	// Delete each policy
	for _, policy := range policyList.Items {
		err := policyResource.Delete(ctx, policy.GetName(), metav1.DeleteOptions{})
		if err != nil {
			log.Error(err, "Failed to delete policy", "policy", policy.GetName())
			continue
		}
		log.V(1).Info("Deleted policy", "policy", policy.GetName())
	}

	log.Info("Cleaned up policies", "count", len(policyList.Items))
	return nil
}

// updateEnforcementStatus updates the enforcement status fields
func (r *ClusterSpecReconciler) updateEnforcementStatus(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	policiesGenerated int,
) {
	if clusterSpec.Status.Enforcement == nil {
		clusterSpec.Status.Enforcement = &kspecv1alpha1.EnforcementStatus{}
	}

	// Update enforcement status
	if clusterSpec.Spec.Enforcement != nil && clusterSpec.Spec.Enforcement.Enabled {
		clusterSpec.Status.Enforcement.Active = true
		clusterSpec.Status.Enforcement.Mode = clusterSpec.Spec.Enforcement.Mode
		clusterSpec.Status.Enforcement.PoliciesGenerated = policiesGenerated
		now := metav1.Now()
		clusterSpec.Status.Enforcement.LastEnforcementTime = &now
	} else {
		clusterSpec.Status.Enforcement.Active = false
		clusterSpec.Status.Enforcement.Mode = ""
		clusterSpec.Status.Enforcement.PoliciesGenerated = 0
	}
}
