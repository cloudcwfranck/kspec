package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPodSecurityStandardsCheck_Pass(t *testing.T) {
	// Setup
	check := &PodSecurityStandardsCheck{}

	// Create namespaces with correct PSS labels
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "baseline",
				"pod-security.kubernetes.io/audit":   "restricted",
				"pod-security.kubernetes.io/warn":    "restricted",
			},
		},
	}

	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-2",
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "baseline",
				"pod-security.kubernetes.io/audit":   "restricted",
				"pod-security.kubernetes.io/warn":    "restricted",
			},
		},
	}

	client := fake.NewSimpleClientset(ns1, ns2)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "restricted",
				Warn:    "restricted",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "podsecurity.standards", result.Name)
	assert.Equal(t, scanner.StatusPass, result.Status)
	assert.Contains(t, result.Message, "correct Pod Security Standards")
}

func TestPodSecurityStandardsCheck_FailMissingLabels(t *testing.T) {
	// Setup
	check := &PodSecurityStandardsCheck{}

	// Create namespace without PSS labels
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "app-1",
			Labels: map[string]string{},
		},
	}

	client := fake.NewSimpleClientset(ns1)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "restricted",
				Warn:    "restricted",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityHigh, result.Severity)
	assert.Contains(t, result.Message, "violations")
	assert.NotEmpty(t, result.Remediation)
}

func TestPodSecurityStandardsCheck_FailWrongLevel(t *testing.T) {
	// Setup
	check := &PodSecurityStandardsCheck{}

	// Create namespace with wrong enforce level
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "privileged", // should be baseline
				"pod-security.kubernetes.io/audit":   "restricted",
				"pod-security.kubernetes.io/warn":    "restricted",
			},
		},
	}

	client := fake.NewSimpleClientset(ns1)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "restricted",
				Warn:    "restricted",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Contains(t, result.Message, "violations")
}

func TestPodSecurityStandardsCheck_WithExemption(t *testing.T) {
	// Setup
	check := &PodSecurityStandardsCheck{}

	// Create namespaces
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "privileged",
				"pod-security.kubernetes.io/audit":   "privileged",
				"pod-security.kubernetes.io/warn":    "privileged",
			},
		},
	}

	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "baseline",
				"pod-security.kubernetes.io/audit":   "restricted",
				"pod-security.kubernetes.io/warn":    "restricted",
			},
		},
	}

	client := fake.NewSimpleClientset(ns1, ns2)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "restricted",
				Warn:    "restricted",
				Exemptions: []spec.PodSecurityExemption{
					{
						Namespace: "kube-system",
						Level:     "privileged",
						Reason:    "System components require host access",
					},
				},
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
	assert.Contains(t, result.Evidence, "exempted")
	assert.Equal(t, 1, result.Evidence["exempted"])
}

func TestPodSecurityStandardsCheck_Skip(t *testing.T) {
	// Setup
	check := &PodSecurityStandardsCheck{}
	client := fake.NewSimpleClientset()

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			// PodSecurity not specified
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusSkip, result.Status)
	assert.Contains(t, result.Message, "not specified")
}

func TestPodSecurityStandardsCheck_Name(t *testing.T) {
	check := &PodSecurityStandardsCheck{}
	assert.Equal(t, "podsecurity.standards", check.Name())
}

func TestPodSecurityStandardsCheck_SystemNamespacesIgnored(t *testing.T) {
	// Setup
	check := &PodSecurityStandardsCheck{}

	// Create system namespaces without PSS labels and one app namespace with labels
	sysNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kube-system",
			Labels: map[string]string{}, // No PSS labels
		},
	}

	appNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
			Labels: map[string]string{
				"pod-security.kubernetes.io/enforce": "baseline",
				"pod-security.kubernetes.io/audit":   "restricted",
				"pod-security.kubernetes.io/warn":    "restricted",
			},
		},
	}

	client := fake.NewSimpleClientset(sysNs, appNs)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "restricted",
				Warn:    "restricted",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
	// kube-system should be ignored, only app-1 checked
	assert.Equal(t, 1, result.Evidence["checked"])
}
