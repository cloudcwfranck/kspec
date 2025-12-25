package spec

import (
	"testing"
)

func TestValidate_ValidSpec(t *testing.T) {
	clusterSpec := &ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: Metadata{
			Name:    "test-cluster",
			Version: "1.0.0",
		},
		Spec: SpecFields{
			Kubernetes: KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	err := Validate(clusterSpec)
	if err != nil {
		t.Errorf("Validate failed for valid spec: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	clusterSpec := &ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: Metadata{
			Name:    "",
			Version: "1.0.0",
		},
		Spec: SpecFields{
			Kubernetes: KubernetesSpec{
				MinVersion: "1.26.0",
			},
		},
	}

	err := Validate(clusterSpec)
	if err == nil {
		t.Error("Expected validation error for missing name, got nil")
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	clusterSpec := &ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: Metadata{
			Name:    "test-cluster",
			Version: "",
		},
		Spec: SpecFields{
			Kubernetes: KubernetesSpec{
				MinVersion: "1.26.0",
			},
		},
	}

	err := Validate(clusterSpec)
	if err == nil {
		t.Error("Expected validation error for missing version, got nil")
	}
}

func TestValidate_MissingMinVersion(t *testing.T) {
	clusterSpec := &ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: Metadata{
			Name:    "test-cluster",
			Version: "1.0.0",
		},
		Spec: SpecFields{
			Kubernetes: KubernetesSpec{
				MinVersion: "",
			},
		},
	}

	err := Validate(clusterSpec)
	if err == nil {
		t.Error("Expected validation error for missing minVersion, got nil")
	}
}

func TestValidate_InvalidVersionRange(t *testing.T) {
	clusterSpec := &ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: Metadata{
			Name:    "test-cluster",
			Version: "1.0.0",
		},
		Spec: SpecFields{
			Kubernetes: KubernetesSpec{
				MinVersion: "1.30.0",
				MaxVersion: "1.26.0", // Max < Min
			},
		},
	}

	err := Validate(clusterSpec)
	if err == nil {
		t.Error("Expected validation error for invalid version range, got nil")
	}
}

func TestValidate_NilSpec(t *testing.T) {
	err := Validate(nil)
	if err == nil {
		t.Error("Expected validation error for nil spec, got nil")
	}
}

func TestValidate_WithOptionalFields(t *testing.T) {
	clusterSpec := &ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: Metadata{
			Name:        "test-cluster",
			Version:     "1.0.0",
			Description: "Test cluster specification",
			Labels: map[string]string{
				"environment": "test",
			},
		},
		Spec: SpecFields{
			Kubernetes: KubernetesSpec{
				MinVersion:       "1.26.0",
				MaxVersion:       "1.30.0",
				ExcludedVersions: []string{"1.27.0"},
			},
			PodSecurity: &PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "restricted",
				Warn:    "restricted",
			},
		},
	}

	err := Validate(clusterSpec)
	if err != nil {
		t.Errorf("Validate failed for spec with optional fields: %v", err)
	}
}
