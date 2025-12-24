package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNetworkPolicyCheck_Pass(t *testing.T) {
	// Setup
	check := &NetworkPolicyCheck{}

	// Create namespace with default-deny policy
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
		},
	}

	// Create default-deny network policy
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny",
			Namespace: "app-1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // Empty selector = all pods
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{}, // Empty = deny all
			Egress:  []networkingv1.NetworkPolicyEgressRule{},  // Empty = deny all
		},
	}

	client := fake.NewSimpleClientset(ns, policy)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Network: &spec.NetworkSpec{
				DefaultDeny: true,
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "network.policies", result.Name)
	assert.Equal(t, scanner.StatusPass, result.Status)
	assert.Contains(t, result.Message, "network policy requirements met")
}

func TestNetworkPolicyCheck_FailMissingDefaultDeny(t *testing.T) {
	// Setup
	check := &NetworkPolicyCheck{}

	// Create namespace without network policy
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
		},
	}

	client := fake.NewSimpleClientset(ns)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Network: &spec.NetworkSpec{
				DefaultDeny: true,
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
	assert.Contains(t, result.Evidence, "namespaces_without_default_deny")
}

func TestNetworkPolicyCheck_FailMissingRequiredPolicy(t *testing.T) {
	// Setup
	check := &NetworkPolicyCheck{}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
		},
	}

	client := fake.NewSimpleClientset(ns)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Network: &spec.NetworkSpec{
				RequiredPolicies: []spec.RequiredPolicy{
					{
						Name:        "allow-dns",
						Description: "Allow DNS resolution",
					},
				},
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Contains(t, result.Message, "violations")
	assert.Contains(t, result.Evidence, "missing_required_policies")
}

func TestNetworkPolicyCheck_PassWithRequiredPolicy(t *testing.T) {
	// Setup
	check := &NetworkPolicyCheck{}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
		},
	}

	// Create required policy
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-dns",
			Namespace: "app-1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	client := fake.NewSimpleClientset(ns, policy)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Network: &spec.NetworkSpec{
				RequiredPolicies: []spec.RequiredPolicy{
					{
						Name:        "allow-dns",
						Description: "Allow DNS resolution",
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
}

func TestNetworkPolicyCheck_Skip(t *testing.T) {
	// Setup
	check := &NetworkPolicyCheck{}
	client := fake.NewSimpleClientset()

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			// Network not specified
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusSkip, result.Status)
	assert.Contains(t, result.Message, "not specified")
}

func TestNetworkPolicyCheck_Name(t *testing.T) {
	check := &NetworkPolicyCheck{}
	assert.Equal(t, "network.policies", check.Name())
}

func TestNetworkPolicyCheck_SystemNamespacesIgnored(t *testing.T) {
	// Setup
	check := &NetworkPolicyCheck{}

	// Create system namespace without policy and app namespace with policy
	sysNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
		},
	}

	appNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-1",
		},
	}

	// Create default-deny policy only for app namespace
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny",
			Namespace: "app-1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
		},
	}

	client := fake.NewSimpleClientset(sysNs, appNs, policy)

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Network: &spec.NetworkSpec{
				DefaultDeny: true,
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
	// kube-system should be ignored, only app-1 checked
}
