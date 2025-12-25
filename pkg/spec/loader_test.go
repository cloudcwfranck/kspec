package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile_ValidSpec(t *testing.T) {
	// Create a temporary valid spec file
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")

	validSpec := `apiVersion: kspec.dev/v1
kind: ClusterSpecification
metadata:
  name: test-cluster
  version: "1.0.0"
spec:
  kubernetes:
    minVersion: "1.26.0"
    maxVersion: "1.30.0"
`

	err := os.WriteFile(specFile, []byte(validSpec), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp spec file: %v", err)
	}

	// Load spec
	clusterSpec, err := LoadFromFile(specFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify metadata
	if clusterSpec.Metadata.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", clusterSpec.Metadata.Name)
	}

	if clusterSpec.Metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", clusterSpec.Metadata.Version)
	}

	// Verify Kubernetes spec
	if clusterSpec.Spec.Kubernetes.MinVersion != "1.26.0" {
		t.Errorf("Expected minVersion '1.26.0', got '%s'", clusterSpec.Spec.Kubernetes.MinVersion)
	}

	if clusterSpec.Spec.Kubernetes.MaxVersion != "1.30.0" {
		t.Errorf("Expected maxVersion '1.30.0', got '%s'", clusterSpec.Spec.Kubernetes.MaxVersion)
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `this is not: [valid yaml`

	err := os.WriteFile(specFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = LoadFromFile(specFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
