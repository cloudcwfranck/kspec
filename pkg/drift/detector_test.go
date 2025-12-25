package drift

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/spec"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDetectPolicyDrift_MissingPolicy(t *testing.T) {
	ctx := context.Background()

	// Create fake clients with no policies deployed
	client := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	// Create detector
	detector := NewDetector(client, dynamicClient)

	// Create spec that expects a policy
	clusterSpec := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    "test-spec",
			Version: "1.0.0",
		},
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{
							Key:   "securityContext.runAsNonRoot",
							Value: true,
						},
					},
				},
			},
		},
	}

	// Detect drift
	events, err := detector.DetectPolicyDrift(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("DetectPolicyDrift failed: %v", err)
	}

	// Should detect missing policy
	if len(events) == 0 {
		t.Fatal("Expected drift events for missing policy, got none")
	}

	// Verify event details
	foundMissing := false
	for _, event := range events {
		if event.DriftKind == "missing" && event.Type == DriftTypePolicy {
			foundMissing = true
			if event.Severity != SeverityHigh {
				t.Errorf("Expected severity %s for missing policy, got %s", SeverityHigh, event.Severity)
			}
		}
	}

	if !foundMissing {
		t.Error("Expected at least one 'missing' policy drift event")
	}
}

func TestDetectPolicyDrift_ModifiedPolicy(t *testing.T) {
	ctx := context.Background()

	// Create a policy with kspec annotation
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "require-run-as-non-root",
				"annotations": map[string]interface{}{
					"kspec.dev/generated": "true",
				},
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"name": "check-run-as-non-root",
						"match": map[string]interface{}{
							"resources": map[string]interface{}{
								"kinds": []interface{}{"Pod"},
							},
						},
						"validate": map[string]interface{}{
							"message": "MODIFIED MESSAGE", // This is different from expected
						},
					},
				},
			},
		},
	}
	policy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kyverno.io",
		Version: "v1",
		Kind:    "ClusterPolicy",
	})

	// Create fake clients with the modified policy
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, policy)

	client := fake.NewSimpleClientset()

	detector := NewDetector(client, dynamicClient)

	// Create spec
	clusterSpec := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    "test-spec",
			Version: "1.0.0",
		},
		Spec: spec.SpecFields{
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{
							Key:   "securityContext.runAsNonRoot",
							Value: true,
						},
					},
				},
			},
		},
	}

	// Detect drift
	events, err := detector.DetectPolicyDrift(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("DetectPolicyDrift failed: %v", err)
	}

	// Should detect modified policy
	foundModified := false
	for _, event := range events {
		if event.DriftKind == "modified" && event.Resource.Name == "require-run-as-non-root" {
			foundModified = true
			if event.Severity != SeverityMedium {
				t.Errorf("Expected severity %s for modified policy, got %s", SeverityMedium, event.Severity)
			}
			if event.Diff == nil {
				t.Error("Expected diff for modified policy")
			}
		}
	}

	if !foundModified {
		t.Error("Expected 'modified' policy drift event")
	}

	_ = gvr // Unused for now but may be needed
}

func TestDetectPolicyDrift_ExtraPolicy(t *testing.T) {
	ctx := context.Background()

	// Create an extra policy with kspec annotation
	extraPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kyverno.io/v1",
			"kind":       "ClusterPolicy",
			"metadata": map[string]interface{}{
				"name": "extra-policy",
				"annotations": map[string]interface{}{
					"kspec.dev/generated": "true",
				},
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"name": "some-rule",
					},
				},
			},
		},
	}
	extraPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kyverno.io",
		Version: "v1",
		Kind:    "ClusterPolicy",
	})

	// Create fake clients with the extra policy
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, extraPolicy)

	client := fake.NewSimpleClientset()

	detector := NewDetector(client, dynamicClient)

	// Create spec with no policies required
	clusterSpec := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    "test-spec",
			Version: "1.0.0",
		},
		Spec: spec.SpecFields{
			// No workload requirements = no policies expected
		},
	}

	// Detect drift
	events, err := detector.DetectPolicyDrift(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("DetectPolicyDrift failed: %v", err)
	}

	// Should detect extra policy
	foundExtra := false
	for _, event := range events {
		if event.DriftKind == "extra" && event.Resource.Name == "extra-policy" {
			foundExtra = true
			if event.Severity != SeverityLow {
				t.Errorf("Expected severity %s for extra policy, got %s", SeverityLow, event.Severity)
			}
		}
	}

	if !foundExtra {
		t.Error("Expected 'extra' policy drift event")
	}
}

func TestDetectPolicyDrift_NoDrift(t *testing.T) {
	ctx := context.Background()

	// Create fake clients with no policies
	client := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	detector := NewDetector(client, dynamicClient)

	// Create spec with no policy requirements
	clusterSpec := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    "test-spec",
			Version: "1.0.0",
		},
		Spec: spec.SpecFields{},
	}

	// Detect drift
	events, err := detector.DetectPolicyDrift(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("DetectPolicyDrift failed: %v", err)
	}

	// Should have no drift
	if len(events) != 0 {
		t.Errorf("Expected no drift events, got %d", len(events))
	}
}

func TestDetect_IntegrationTest(t *testing.T) {
	ctx := context.Background()

	// Create fake clients
	client := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	detector := NewDetector(client, dynamicClient)

	// Create spec
	clusterSpec := &spec.ClusterSpecification{
		Metadata: spec.Metadata{
			Name:    "test-spec",
			Version: "1.0.0",
		},
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
			},
		},
	}

	// Detect all drift types
	report, err := detector.Detect(ctx, clusterSpec, DetectOptions{
		EnabledTypes: []DriftType{DriftTypePolicy, DriftTypeCompliance},
	})
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Verify report structure
	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if report.Spec.Name != "test-spec" {
		t.Errorf("Expected spec name 'test-spec', got '%s'", report.Spec.Name)
	}

	if report.Spec.Version != "1.0.0" {
		t.Errorf("Expected spec version '1.0.0', got '%s'", report.Spec.Version)
	}

	// Events should be populated (may be empty or have some entries)
	if report.Events == nil {
		t.Error("Expected non-nil events slice")
	}
}

func TestIsKspecGenerated(t *testing.T) {
	client := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	detector := NewDetector(client, dynamicClient)

	tests := []struct {
		name     string
		policy   runtime.Object
		expected bool
	}{
		{
			name: "policy with kspec annotation",
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"kspec.dev/generated": "true",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "policy without kspec annotation",
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"other.annotation": "value",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "policy with no annotations",
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isKspecGenerated(tt.policy)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUpdateSummary(t *testing.T) {
	client := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	detector := NewDetector(client, dynamicClient)

	report := &DriftReport{
		Events: []DriftEvent{
			{
				Type:     DriftTypePolicy,
				Severity: SeverityHigh,
			},
			{
				Type:     DriftTypeCompliance,
				Severity: SeverityMedium,
			},
			{
				Type:     DriftTypePolicy,
				Severity: SeverityCritical,
			},
		},
	}

	detector.updateSummary(report)

	// Verify summary
	if !report.Drift.Detected {
		t.Error("Expected drift detected to be true")
	}

	if report.Drift.Counts.Total != 3 {
		t.Errorf("Expected total count 3, got %d", report.Drift.Counts.Total)
	}

	if report.Drift.Counts.Policies != 2 {
		t.Errorf("Expected policy count 2, got %d", report.Drift.Counts.Policies)
	}

	if report.Drift.Counts.Compliance != 1 {
		t.Errorf("Expected compliance count 1, got %d", report.Drift.Counts.Compliance)
	}

	if report.Drift.Severity != SeverityCritical {
		t.Errorf("Expected highest severity %s, got %s", SeverityCritical, report.Drift.Severity)
	}

	if len(report.Drift.Types) != 2 {
		t.Errorf("Expected 2 unique drift types, got %d", len(report.Drift.Types))
	}
}
