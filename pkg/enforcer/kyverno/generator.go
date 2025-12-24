package kyverno

import (
	"fmt"

	"github.com/cloudcwfranck/kspec/pkg/spec"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
func (g *Generator) createRunAsNonRootPolicy() *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "require-run-as-non-root",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Require runAsNonRoot",
				"policies.kyverno.io/category":    "Pod Security Standards (Restricted)",
				"policies.kyverno.io/severity":    "medium",
				"policies.kyverno.io/description": "Containers must run as non-root users",
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "check-runAsNonRoot",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": "Containers must run as non-root (securityContext.runAsNonRoot must be true)",
						"pattern": map[string]interface{}{
							"spec": map[string]interface{}{
								"securityContext": map[string]interface{}{
									"runAsNonRoot": true,
								},
							},
						},
					},
				},
			},
		},
	})
	return policy
}

// createDisallowPrivilegeEscalationPolicy creates a policy disallowing privilege escalation.
func (g *Generator) createDisallowPrivilegeEscalationPolicy() *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "disallow-privilege-escalation",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Disallow Privilege Escalation",
				"policies.kyverno.io/category":    "Pod Security Standards (Restricted)",
				"policies.kyverno.io/severity":    "medium",
				"policies.kyverno.io/description": "Privilege escalation must be disabled",
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "check-allowPrivilegeEscalation",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": "Privilege escalation is disallowed (securityContext.allowPrivilegeEscalation must be false)",
						"pattern": map[string]interface{}{
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
			},
		},
	})
	return policy
}

// createDisallowPrivilegedPolicy creates a policy disallowing privileged containers.
func (g *Generator) createDisallowPrivilegedPolicy() *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "disallow-privileged-containers",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Disallow Privileged Containers",
				"policies.kyverno.io/category":    "Pod Security Standards (Baseline)",
				"policies.kyverno.io/severity":    "high",
				"policies.kyverno.io/description": "Privileged containers are not allowed",
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "check-privileged",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": "Privileged containers are not allowed",
						"pattern": map[string]interface{}{
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
			},
		},
	})
	return policy
}

// createDisallowHostNamespacesPolicy creates a policy disallowing host namespaces.
func (g *Generator) createDisallowHostNamespacesPolicy() *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "disallow-host-namespaces",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Disallow Host Namespaces",
				"policies.kyverno.io/category":    "Pod Security Standards (Baseline)",
				"policies.kyverno.io/severity":    "high",
				"policies.kyverno.io/description": "Host namespaces (hostNetwork, hostPID, hostIPC) are not allowed",
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "check-host-namespaces",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": "Host namespaces are not allowed",
						"pattern": map[string]interface{}{
							"spec": map[string]interface{}{
								"=(hostNetwork)": false,
								"=(hostPID)":     false,
								"=(hostIPC)":     false,
							},
						},
					},
				},
			},
		},
	})
	return policy
}

// createRequireResourceLimitsPolicy creates a policy requiring resource limits.
func (g *Generator) createRequireResourceLimitsPolicy() *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "require-resource-limits",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Require Resource Limits",
				"policies.kyverno.io/category":    "Best Practices",
				"policies.kyverno.io/severity":    "medium",
				"policies.kyverno.io/description": "All containers must have memory and CPU limits defined",
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "check-resource-limits",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": "All containers must have memory and CPU limits",
						"pattern": map[string]interface{}{
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
			},
		},
	})
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
func (g *Generator) createRequireDigestsPolicy() *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "require-image-digests",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Require Image Digests",
				"policies.kyverno.io/category":    "Supply Chain Security",
				"policies.kyverno.io/severity":    "medium",
				"policies.kyverno.io/description": "Images must use digests (not tags) for immutability",
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "check-image-digest",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": "Images must use digests (e.g., image@sha256:...) not tags",
						"pattern": map[string]interface{}{
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
			},
		},
	})
	return policy
}

// createBlockedRegistriesPolicy creates a policy blocking specific registries.
func (g *Generator) createBlockedRegistriesPolicy(blockedRegistries []string) *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "kyverno.io/v1",
		"kind":       "ClusterPolicy",
		"metadata": map[string]interface{}{
			"name": "block-image-registries",
			"annotations": map[string]interface{}{
				"policies.kyverno.io/title":       "Block Specific Image Registries",
				"policies.kyverno.io/category":    "Supply Chain Security",
				"policies.kyverno.io/severity":    "high",
				"policies.kyverno.io/description": fmt.Sprintf("Block images from: %v", blockedRegistries),
				"kspec.dev/generated":             "true",
			},
		},
		"spec": map[string]interface{}{
			"validationFailureAction": "Enforce",
			"background":              true,
			"rules": []interface{}{
				map[string]interface{}{
					"name": "block-registries",
					"match": map[string]interface{}{
						"any": []interface{}{
							map[string]interface{}{
								"resources": map[string]interface{}{
									"kinds": []interface{}{"Pod"},
								},
							},
						},
					},
					"validate": map[string]interface{}{
						"message": fmt.Sprintf("Images from blocked registries are not allowed: %v", blockedRegistries),
						"pattern": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "!docker.io/*",
									},
								},
							},
						},
					},
				},
			},
		},
	})
	return policy
}
