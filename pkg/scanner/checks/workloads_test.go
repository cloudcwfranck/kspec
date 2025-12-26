package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWorkloadSecurityCheck_Pass(t *testing.T) {
	// Create compliant pod
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compliant-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/myapp@sha256:abc123",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot:             &runAsNonRoot,
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	existsTrue := true
	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "securityContext.runAsNonRoot", Value: "true"},
						{Key: "securityContext.allowPrivilegeEscalation", Value: "false"},
						{Key: "resources.limits.memory", Exists: &existsTrue},
						{Key: "resources.requests.cpu", Exists: &existsTrue},
					},
					Forbidden: []spec.FieldRequirement{
						{Key: "securityContext.privileged", Value: "true"},
						{Key: "hostNetwork", Value: "true"},
					},
				},
				Images: &spec.ImageSpec{
					AllowedRegistries: []string{"ghcr.io"},
					RequireDigests:    true,
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
	assert.Contains(t, result.Message, "comply with security requirements")
}

func TestWorkloadSecurityCheck_FailMissingSecurityContext(t *testing.T) {
	// Pod missing runAsNonRoot
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "insecure-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/myapp:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "securityContext.runAsNonRoot", Value: "true"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityHigh, result.Severity)
	assert.Contains(t, result.Evidence, "violations")
}

func TestWorkloadSecurityCheck_FailPrivilegedContainer(t *testing.T) {
	// Privileged container
	privileged := true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "privileged-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/myapp:latest",
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Forbidden: []spec.FieldRequirement{
						{Key: "securityContext.privileged", Value: "true"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Contains(t, result.Evidence, "violations")
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "privileged containers are forbidden")
}

func TestWorkloadSecurityCheck_FailMissingResources(t *testing.T) {
	// Pod missing resource limits
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-limits-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/myapp:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	existsTrue := true
	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "resources.limits.memory", Exists: &existsTrue},
						{Key: "resources.requests.cpu", Exists: &existsTrue},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.True(t, len(violations) >= 2)
}

func TestWorkloadSecurityCheck_FailBlockedRegistry(t *testing.T) {
	// Pod using blocked registry
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docker-hub-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Images: &spec.ImageSpec{
					BlockedRegistries: []string{"docker.io"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "blocked registry")
}

func TestWorkloadSecurityCheck_FailNotAllowedRegistry(t *testing.T) {
	// Pod not using allowed registry
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "quay-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "quay.io/myapp:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Images: &spec.ImageSpec{
					AllowedRegistries: []string{"ghcr.io", "*.azurecr.io"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "not from allowed registry")
}

func TestWorkloadSecurityCheck_FailMissingDigest(t *testing.T) {
	// Pod using tag instead of digest
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tag-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/myapp:v1.0.0",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Images: &spec.ImageSpec{
					AllowedRegistries: []string{"ghcr.io"},
					RequireDigests:    true,
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "must use digest")
}

func TestWorkloadSecurityCheck_FailHostNetwork(t *testing.T) {
	// Pod using hostNetwork
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host-network-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/myapp:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Forbidden: []spec.FieldRequirement{
						{Key: "hostNetwork", Value: "true"},
						{Key: "hostPID", Value: "true"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "hostNetwork is forbidden")
}

func TestWorkloadSecurityCheck_Skip(t *testing.T) {
	client := fake.NewSimpleClientset()
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			// No workloads spec
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusSkip, result.Status)
}

func TestWorkloadSecurityCheck_Name(t *testing.T) {
	check := &WorkloadSecurityCheck{}
	assert.Equal(t, "workload.security", check.Name())
}

func TestWorkloadSecurityCheck_SystemNamespacesIgnored(t *testing.T) {
	// Pod in kube-system (should be ignored)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-pod",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "system:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "securityContext.runAsNonRoot", Value: "true"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestWorkloadSecurityCheck_InitContainers(t *testing.T) {
	// Pod with non-compliant init container
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "init-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "init",
					Image: "ghcr.io/init:latest",
					// Missing security context
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "ghcr.io/app:latest",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &WorkloadSecurityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "securityContext.runAsNonRoot", Value: "true"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	// Should have violations for both init and regular container
	assert.True(t, len(violations) >= 2)
}
