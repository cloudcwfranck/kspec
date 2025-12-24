package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesVersionCheck_Pass(t *testing.T) {
	// Setup
	check := &KubernetesVersionCheck{}
	client := fake.NewSimpleClientset()

	// Set cluster version to 1.28.0
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "expected FakeDiscovery")
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.28.0",
	}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "kubernetes.version", result.Name)
	assert.Equal(t, scanner.StatusPass, result.Status)
	assert.Contains(t, result.Message, "1.28.0")
	assert.Contains(t, result.Message, "within spec range")
}

func TestKubernetesVersionCheck_FailTooLow(t *testing.T) {
	// Setup
	check := &KubernetesVersionCheck{}
	client := fake.NewSimpleClientset()

	// Set cluster version to 1.25.0 (below minimum)
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "expected FakeDiscovery")
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.25.0",
	}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "kubernetes.version", result.Name)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityCritical, result.Severity)
	assert.Contains(t, result.Message, "outside allowed range")
	assert.NotEmpty(t, result.Remediation)
	assert.Contains(t, result.Remediation, "Upgrade")
}

func TestKubernetesVersionCheck_FailTooHigh(t *testing.T) {
	// Setup
	check := &KubernetesVersionCheck{}
	client := fake.NewSimpleClientset()

	// Set cluster version to 1.31.0 (above maximum)
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "expected FakeDiscovery")
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.31.0",
	}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityCritical, result.Severity)
	assert.Contains(t, result.Message, "outside allowed range")
}

func TestKubernetesVersionCheck_FailExcludedVersion(t *testing.T) {
	// Setup
	check := &KubernetesVersionCheck{}
	client := fake.NewSimpleClientset()

	// Set cluster version to 1.29.0 (excluded)
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "expected FakeDiscovery")
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.29.0",
	}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion:       "1.26.0",
				MaxVersion:       "1.30.0",
				ExcludedVersions: []string{"1.29.0"},
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityCritical, result.Severity)
	assert.Contains(t, result.Message, "excluded")
	assert.NotEmpty(t, result.Remediation)
}

func TestKubernetesVersionCheck_PassMinVersion(t *testing.T) {
	// Setup
	check := &KubernetesVersionCheck{}
	client := fake.NewSimpleClientset()

	// Set cluster version to exactly minimum
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "expected FakeDiscovery")
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.26.0",
	}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestKubernetesVersionCheck_PassMaxVersion(t *testing.T) {
	// Setup
	check := &KubernetesVersionCheck{}
	client := fake.NewSimpleClientset()

	// Set cluster version to exactly maximum
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	require.True(t, ok, "expected FakeDiscovery")
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.30.0",
	}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	// Execute
	result, err := check.Run(context.Background(), client, clusterSpec)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestKubernetesVersionCheck_Name(t *testing.T) {
	check := &KubernetesVersionCheck{}
	assert.Equal(t, "kubernetes.version", check.Name())
}
