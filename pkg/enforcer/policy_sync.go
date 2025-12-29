package enforcer

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/metrics"
)

// PolicySynchronizer synchronizes policies across multiple clusters
type PolicySynchronizer struct {
	LocalDynamicClient dynamic.Interface
}

// NewPolicySynchronizer creates a new policy synchronizer
func NewPolicySynchronizer(localClient dynamic.Interface) *PolicySynchronizer {
	return &PolicySynchronizer{
		LocalDynamicClient: localClient,
	}
}

// SyncPolicyToCluster synchronizes a ClusterSpec's policies to a target cluster
func (p *PolicySynchronizer) SyncPolicyToCluster(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	targetClient dynamic.Interface,
	clusterName string,
) error {
	log := log.FromContext(ctx).WithValues("cluster", clusterName, "clusterSpec", clusterSpec.Name)

	// Skip if enforcement not enabled
	if clusterSpec.Spec.Enforcement == nil || !clusterSpec.Spec.Enforcement.Enabled {
		log.Info("Enforcement not enabled, skipping policy sync")
		return nil
	}

	// Skip if using webhooks instead of Kyverno
	if clusterSpec.Spec.Webhooks != nil && clusterSpec.Spec.Webhooks.Enabled {
		log.Info("Webhooks enabled, Kyverno policy sync not needed")
		return nil
	}

	log.Info("Syncing Kyverno policies to cluster")

	// Get existing policies from local cluster that match this ClusterSpec
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	policyName := fmt.Sprintf("kspec-%s", clusterSpec.Name)
	policy, err := p.LocalDynamicClient.Resource(gvr).Get(ctx, policyName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("No local policy found to sync", "policy", policyName)
			return nil
		}
		return fmt.Errorf("failed to get local policy: %w", err)
	}

	// Create or update policy in target cluster
	if err := p.applyPolicyToCluster(ctx, policy, targetClient, clusterName); err != nil {
		return fmt.Errorf("failed to apply policy: %w", err)
	}

	// Record metric
	metrics.KyvernoPolicyCreated.WithLabelValues(clusterSpec.Name, "cluster-policy").Inc()

	log.Info("Successfully synced policy to cluster")
	return nil
}

// applyPolicyToCluster applies a Kyverno policy to a target cluster
func (p *PolicySynchronizer) applyPolicyToCluster(
	ctx context.Context,
	policy *unstructured.Unstructured,
	targetClient dynamic.Interface,
	clusterName string,
) error {
	// Add cluster annotation
	annotations := policy.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["kspec.io/target-cluster"] = clusterName
	annotations["kspec.io/managed-by"] = "kspec-controller"
	policy.SetAnnotations(annotations)

	// Define Kyverno ClusterPolicy GVR
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	// Try to get existing policy
	policyName := policy.GetName()
	_, err := targetClient.Resource(gvr).Get(ctx, policyName, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		// Create new policy
		_, err = targetClient.Resource(gvr).Create(ctx, policy, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create policy: %w", err)
		}
	} else if err == nil {
		// Update existing policy
		_, err = targetClient.Resource(gvr).Update(ctx, policy, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update policy: %w", err)
		}
	} else {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}

	return nil
}

// RemovePolicyFromCluster removes a policy from a target cluster
func (p *PolicySynchronizer) RemovePolicyFromCluster(
	ctx context.Context,
	policyName string,
	targetClient dynamic.Interface,
	clusterName string,
) error {
	log := log.FromContext(ctx).WithValues("cluster", clusterName, "policy", policyName)

	log.Info("Removing policy from cluster")

	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	err := targetClient.Resource(gvr).Delete(ctx, policyName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	log.Info("Successfully removed policy from cluster")
	return nil
}

// SyncPoliciesAcrossFleet synchronizes policies across all clusters in the fleet
func (p *PolicySynchronizer) SyncPoliciesAcrossFleet(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	clusterClients map[string]dynamic.Interface,
) error {
	log := log.FromContext(ctx).WithValues("clusterSpec", clusterSpec.Name)

	log.Info("Syncing policies across fleet", "clusterCount", len(clusterClients))

	var lastErr error
	successCount := 0

	for clusterName, client := range clusterClients {
		if err := p.SyncPolicyToCluster(ctx, clusterSpec, client, clusterName); err != nil {
			log.Error(err, "Failed to sync policy to cluster", "cluster", clusterName)
			lastErr = err
		} else {
			successCount++
		}
	}

	log.Info("Fleet policy sync complete",
		"total", len(clusterClients),
		"success", successCount,
		"failed", len(clusterClients)-successCount)

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("all policy syncs failed, last error: %w", lastErr)
	}

	return nil
}

// ValidatePolicyConsistency validates that policies are consistent across clusters
func (p *PolicySynchronizer) ValidatePolicyConsistency(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	clusterClients map[string]dynamic.Interface,
) (bool, []string, error) {
	log := log.FromContext(ctx).WithValues("clusterSpec", clusterSpec.Name)

	log.Info("Validating policy consistency across fleet")

	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	policyName := fmt.Sprintf("kspec-%s", clusterSpec.Name)
	inconsistencies := make([]string, 0)

	// Get policy from first cluster as baseline
	var baselinePolicy *unstructured.Unstructured
	var baselineCluster string

	for clusterName, client := range clusterClients {
		policy, err := client.Resource(gvr).Get(ctx, policyName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				inconsistencies = append(inconsistencies,
					fmt.Sprintf("Policy %s not found in cluster %s", policyName, clusterName))
				continue
			}
			return false, inconsistencies, fmt.Errorf("failed to get policy from %s: %w", clusterName, err)
		}

		if baselinePolicy == nil {
			baselinePolicy = policy
			baselineCluster = clusterName
			continue
		}

		// Compare policy specs
		baselineSpec, _, _ := unstructured.NestedMap(baselinePolicy.Object, "spec")
		currentSpec, _, _ := unstructured.NestedMap(policy.Object, "spec")

		// Simple comparison (in production, would use deep comparison)
		if fmt.Sprintf("%v", baselineSpec) != fmt.Sprintf("%v", currentSpec) {
			inconsistencies = append(inconsistencies,
				fmt.Sprintf("Policy %s in cluster %s differs from baseline in %s",
					policyName, clusterName, baselineCluster))
		}
	}

	isConsistent := len(inconsistencies) == 0
	log.Info("Policy consistency validation complete",
		"consistent", isConsistent,
		"inconsistencies", len(inconsistencies))

	return isConsistent, inconsistencies, nil
}
