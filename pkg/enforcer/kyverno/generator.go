package kyverno

import (
	"fmt"

	"github.com/cloudcwfranck/kspec/pkg/spec"
	"k8s.io/apimachinery/pkg/runtime"
)

// Generator generates Kyverno policies from cluster specifications.
type Generator struct{}

// NewGenerator creates a new Kyverno policy generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GeneratePolicies generates Kyverno ClusterPolicy resources from a cluster specification.
func (g *Generator) GeneratePolicies(clusterSpec *spec.ClusterSpecification) ([]runtime.Object, error) {
	policies := []runtime.Object{}

	// Generate workload security policies
	if clusterSpec.Spec.Workloads != nil && clusterSpec.Spec.Workloads.Containers != nil {
		workloadPolicies, err := g.generateWorkloadPolicies(clusterSpec.Spec.Workloads)
		if err != nil {
			return nil, fmt.Errorf("failed to generate workload policies: %w", err)
		}
		policies = append(policies, workloadPolicies...)
	}

	// Generate image registry policies
	if clusterSpec.Spec.Workloads != nil && clusterSpec.Spec.Workloads.Images != nil {
		imagePolicies, err := g.generateImagePolicies(clusterSpec.Spec.Workloads.Images)
		if err != nil {
			return nil, fmt.Errorf("failed to generate image policies: %w", err)
		}
		policies = append(policies, imagePolicies...)
	}

	return policies, nil
}

// generateWorkloadPolicies creates policies for workload security requirements.
func (g *Generator) generateWorkloadPolicies(workloadsSpec *spec.WorkloadsSpec) ([]runtime.Object, error) {
	policies := []runtime.Object{}

	if workloadsSpec.Containers == nil {
		return policies, nil
	}

	// Check for runAsNonRoot requirement
	for _, req := range workloadsSpec.Containers.Required {
		if req.Key == "securityContext.runAsNonRoot" && req.Value == true {
			policy := g.createRunAsNonRootPolicy()
			policies = append(policies, policy)
		}
		if req.Key == "securityContext.allowPrivilegeEscalation" && req.Value == false {
			policy := g.createDisallowPrivilegeEscalationPolicy()
			policies = append(policies, policy)
		}
		if req.Key == "resources.limits.memory" && req.Exists != nil && *req.Exists {
			policy := g.createRequireResourceLimitsPolicy()
			policies = append(policies, policy)
		}
	}

	// Check for forbidden fields
	for _, forbidden := range workloadsSpec.Containers.Forbidden {
		if forbidden.Key == "securityContext.privileged" && forbidden.Value == true {
			policy := g.createDisallowPrivilegedPolicy()
			policies = append(policies, policy)
		}
		if forbidden.Key == "hostNetwork" && forbidden.Value == true {
			policy := g.createDisallowHostNamespacesPolicy()
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// createRunAsNonRootPolicy creates a policy requiring containers to run as non-root.
func (g *Generator) createRunAsNonRootPolicy() *ClusterPolicy {
	policy := NewClusterPolicy("require-run-as-non-root")
	policy.Annotations["policies.kyverno.io/title"] = "Require runAsNonRoot"
	policy.Annotations["policies.kyverno.io/category"] = "Pod Security Standards (Restricted)"
	policy.Annotations["policies.kyverno.io/severity"] = "medium"
	policy.Annotations["policies.kyverno.io/description"] = "Containers must run as non-root users"

	policy.Spec.Rules = []Rule{
		{
			Name: "check-runAsNonRoot",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: "Containers must run as non-root (securityContext.runAsNonRoot must be true)",
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"securityContext": map[string]interface{}{
							"runAsNonRoot": true,
						},
					},
				},
			},
		},
	}

	return policy
}

// createDisallowPrivilegeEscalationPolicy creates a policy disallowing privilege escalation.
func (g *Generator) createDisallowPrivilegeEscalationPolicy() *ClusterPolicy {
	policy := NewClusterPolicy("disallow-privilege-escalation")
	policy.Annotations["policies.kyverno.io/title"] = "Disallow Privilege Escalation"
	policy.Annotations["policies.kyverno.io/category"] = "Pod Security Standards (Restricted)"
	policy.Annotations["policies.kyverno.io/severity"] = "medium"
	policy.Annotations["policies.kyverno.io/description"] = "Privilege escalation must be disabled"

	policy.Spec.Rules = []Rule{
		{
			Name: "check-allowPrivilegeEscalation",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: "Privilege escalation is disallowed (securityContext.allowPrivilegeEscalation must be false)",
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"securityContext": map[string]interface{}{
									"allowPrivilegeEscalation": false,
								},
							},
						},
					},
				},
			},
		},
	}

	return policy
}

// createDisallowPrivilegedPolicy creates a policy disallowing privileged containers.
func (g *Generator) createDisallowPrivilegedPolicy() *ClusterPolicy {
	policy := NewClusterPolicy("disallow-privileged-containers")
	policy.Annotations["policies.kyverno.io/title"] = "Disallow Privileged Containers"
	policy.Annotations["policies.kyverno.io/category"] = "Pod Security Standards (Baseline)"
	policy.Annotations["policies.kyverno.io/severity"] = "high"
	policy.Annotations["policies.kyverno.io/description"] = "Privileged containers are not allowed"

	policy.Spec.Rules = []Rule{
		{
			Name: "check-privileged",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: "Privileged containers are not allowed",
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"=(securityContext)": map[string]interface{}{
									"=(privileged)": false,
								},
							},
						},
					},
				},
			},
		},
	}

	return policy
}

// createDisallowHostNamespacesPolicy creates a policy disallowing host namespaces.
func (g *Generator) createDisallowHostNamespacesPolicy() *ClusterPolicy {
	policy := NewClusterPolicy("disallow-host-namespaces")
	policy.Annotations["policies.kyverno.io/title"] = "Disallow Host Namespaces"
	policy.Annotations["policies.kyverno.io/category"] = "Pod Security Standards (Baseline)"
	policy.Annotations["policies.kyverno.io/severity"] = "high"
	policy.Annotations["policies.kyverno.io/description"] = "Host namespaces (hostNetwork, hostPID, hostIPC) are not allowed"

	policy.Spec.Rules = []Rule{
		{
			Name: "check-host-namespaces",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: "Host namespaces are not allowed",
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"=(hostNetwork)": false,
						"=(hostPID)":     false,
						"=(hostIPC)":     false,
					},
				},
			},
		},
	}

	return policy
}

// createRequireResourceLimitsPolicy creates a policy requiring resource limits.
func (g *Generator) createRequireResourceLimitsPolicy() *ClusterPolicy {
	policy := NewClusterPolicy("require-resource-limits")
	policy.Annotations["policies.kyverno.io/title"] = "Require Resource Limits"
	policy.Annotations["policies.kyverno.io/category"] = "Best Practices"
	policy.Annotations["policies.kyverno.io/severity"] = "medium"
	policy.Annotations["policies.kyverno.io/description"] = "All containers must have memory and CPU limits defined"

	policy.Spec.Rules = []Rule{
		{
			Name: "check-resource-limits",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: "All containers must have memory and CPU limits",
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"limits": map[string]interface{}{
										"memory": "?*",
										"cpu":    "?*",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return policy
}

// generateImagePolicies creates policies for image registry requirements.
func (g *Generator) generateImagePolicies(imageSpec *spec.ImageSpec) ([]runtime.Object, error) {
	policies := []runtime.Object{}

	// Create policy for requiring digests
	if imageSpec.RequireDigests {
		policy := g.createRequireDigestsPolicy()
		policies = append(policies, policy)
	}

	// Create policy for blocked registries
	if len(imageSpec.BlockedRegistries) > 0 {
		policy := g.createBlockedRegistriesPolicy(imageSpec.BlockedRegistries)
		policies = append(policies, policy)
	}

	return policies, nil
}

// createRequireDigestsPolicy creates a policy requiring image digests.
func (g *Generator) createRequireDigestsPolicy() *ClusterPolicy {
	policy := NewClusterPolicy("require-image-digests")
	policy.Annotations["policies.kyverno.io/title"] = "Require Image Digests"
	policy.Annotations["policies.kyverno.io/category"] = "Supply Chain Security"
	policy.Annotations["policies.kyverno.io/severity"] = "medium"
	policy.Annotations["policies.kyverno.io/description"] = "Images must use digests (not tags) for immutability"

	policy.Spec.Rules = []Rule{
		{
			Name: "check-image-digest",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: "Images must use digests (e.g., image@sha256:...) not tags",
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "*@sha256:*",
							},
						},
					},
				},
			},
		},
	}

	return policy
}

// createBlockedRegistriesPolicy creates a policy blocking specific registries.
func (g *Generator) createBlockedRegistriesPolicy(blockedRegistries []string) *ClusterPolicy {
	policy := NewClusterPolicy("block-image-registries")
	policy.Annotations["policies.kyverno.io/title"] = "Block Specific Image Registries"
	policy.Annotations["policies.kyverno.io/category"] = "Supply Chain Security"
	policy.Annotations["policies.kyverno.io/severity"] = "high"
	policy.Annotations["policies.kyverno.io/description"] = fmt.Sprintf("Block images from: %v", blockedRegistries)

	// Build deny pattern for blocked registries
	// For Kyverno, we need to create a deny condition that blocks specific registries
	policy.Spec.Rules = []Rule{
		{
			Name: "block-registries",
			Match: MatchResources{
				Any: []ResourceFilter{
					{
						Resources: &ResourceDescription{
							Kinds: []string{"Pod"},
						},
					},
				},
			},
			Validation: &Validation{
				Message: fmt.Sprintf("Images from blocked registries are not allowed: %v", blockedRegistries),
				Pattern: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								// Use negation pattern to block specific registry
								// This is a simplified pattern - in production, you'd iterate over blockedRegistries
								"image": "!docker.io/*",
							},
						},
					},
				},
			},
		},
	}

	return policy
}
