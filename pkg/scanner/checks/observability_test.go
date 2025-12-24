package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestObservabilityCheck_PassWithMetricsServer(t *testing.T) {
	// Metrics server deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-server",
			Namespace: "kube-system",
		},
	}

	client := fake.NewSimpleClientset(deployment)
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  true,
					Providers: []string{"metrics-server"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestObservabilityCheck_PassWithPrometheus(t *testing.T) {
	// Prometheus pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-server",
			Namespace: "monitoring",
			Labels: map[string]string{
				"app.kubernetes.io/name": "prometheus",
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  true,
					Providers: []string{"prometheus"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestObservabilityCheck_FailNoMetricsProvider(t *testing.T) {
	// No metrics providers installed
	client := fake.NewSimpleClientset()
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  true,
					Providers: []string{"metrics-server"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityMedium, result.Severity)
	assert.Contains(t, result.Evidence, "violations")
}

func TestObservabilityCheck_PassWithAnyProvider(t *testing.T) {
	// Multiple providers specified, one found
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-0",
			Namespace: "monitoring",
			Labels: map[string]string{
				"app": "prometheus",
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  true,
					Providers: []string{"prometheus", "metrics-server"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
	// Should show one found, one missing
	assert.Contains(t, result.Evidence, "found_providers")
	foundProviders := result.Evidence["found_providers"].([]string)
	assert.Contains(t, foundProviders, "prometheus")
}

func TestObservabilityCheck_AuditLogWithConfigMap(t *testing.T) {
	// Audit policy ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "audit-policy",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"audit-policy.yaml": "apiVersion: audit.k8s.io/v1",
		},
	}

	client := fake.NewSimpleClientset(cm)
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Logging: &spec.LoggingSpec{
					AuditLog: &spec.AuditLogSpec{
						Required:         true,
						MinRetentionDays: 90,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	// Will have a warning about retention validation
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Contains(t, result.Evidence, "audit_policy_found")
	assert.True(t, result.Evidence["audit_policy_found"].(bool))
}

func TestObservabilityCheck_FailNoAuditLog(t *testing.T) {
	// No audit logging configured
	client := fake.NewSimpleClientset()
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Logging: &spec.LoggingSpec{
					AuditLog: &spec.AuditLogSpec{
						Required:         true,
						MinRetentionDays: 90,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.True(t, len(violations) > 0)
}

func TestObservabilityCheck_Skip(t *testing.T) {
	client := fake.NewSimpleClientset()
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			// No observability spec
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusSkip, result.Status)
}

func TestObservabilityCheck_Name(t *testing.T) {
	check := &ObservabilityCheck{}
	assert.Equal(t, "observability.validation", check.Name())
}

func TestObservabilityCheck_MetricsNotRequired(t *testing.T) {
	// Metrics specified but not required
	client := fake.NewSimpleClientset()
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  false,
					Providers: []string{"metrics-server"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestObservabilityCheck_GenericProvider(t *testing.T) {
	// Custom metrics provider pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "datadog-agent-xyz",
			Namespace: "monitoring",
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  true,
					Providers: []string{"datadog"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestObservabilityCheck_MetricsServerByPod(t *testing.T) {
	// Metrics server pod (not deployment)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-server-abc123",
			Namespace: "kube-system",
			Labels: map[string]string{
				"k8s-app": "metrics-server",
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	check := &ObservabilityCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required:  true,
					Providers: []string{"metrics-server"},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}
