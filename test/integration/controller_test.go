/*
Copyright 2025 kspec contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

func TestClusterTargetReconciler(t *testing.T) {
	s := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	ctx := context.Background()

	t.Run("should create ClusterTarget resource", func(t *testing.T) {
		ct := &kspecv1alpha1.ClusterTarget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "default",
			},
			Spec: kspecv1alpha1.ClusterTargetSpec{
				APIServerURL: "https://test:6443",
				AuthMode:     "serviceAccount",
			},
		}

		if err := fakeClient.Create(ctx, ct); err != nil {
			t.Fatalf("Failed to create ClusterTarget: %v", err)
		}

		// Verify creation
		var retrieved kspecv1alpha1.ClusterTarget
		if err := fakeClient.Get(ctx, client.ObjectKey{
			Name:      ct.Name,
			Namespace: ct.Namespace,
		}, &retrieved); err != nil {
			t.Fatalf("Failed to retrieve ClusterTarget: %v", err)
		}

		if retrieved.Spec.APIServerURL != "https://test:6443" {
			t.Errorf("Expected APIServerURL https://test:6443, got %s", retrieved.Spec.APIServerURL)
		}

		// Note: Actual reconciliation with ClientFactory requires E2E tests
		// Integration tests only validate resource creation/lifecycle
	})
}

func TestClusterSpecReconciler(t *testing.T) {
	s := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	ctx := context.Background()

	t.Run("should create ClusterSpec resource", func(t *testing.T) {
		cs := &kspecv1alpha1.ClusterSpecification{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-spec",
				Namespace: "default",
			},
		}
		cs.Spec.Kubernetes.MinVersion = "1.28.0"

		if err := fakeClient.Create(ctx, cs); err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// Verify creation
		var retrieved kspecv1alpha1.ClusterSpecification
		if err := fakeClient.Get(ctx, client.ObjectKey{
			Name:      cs.Name,
			Namespace: cs.Namespace,
		}, &retrieved); err != nil {
			t.Fatalf("Failed to retrieve ClusterSpec: %v", err)
		}

		if retrieved.Spec.Kubernetes.MinVersion != "1.28.0" {
			t.Errorf("Expected Kubernetes.MinVersion 1.28.0, got %s", retrieved.Spec.Kubernetes.MinVersion)
		}

		// Note: Actual reconciliation with ClientFactory requires E2E tests
		// Integration tests only validate resource creation/lifecycle
	})

	t.Run("should validate spec structure", func(t *testing.T) {
		cs := &kspecv1alpha1.ClusterSpecification{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-validation",
				Namespace: "default",
			},
		}
		cs.Spec.Kubernetes.MinVersion = "1.26.0"
		cs.Spec.Kubernetes.MaxVersion = "1.30.0"

		if err := fakeClient.Create(ctx, cs); err != nil {
			t.Fatalf("Failed to create ClusterSpec with version range: %v", err)
		}

		var retrieved kspecv1alpha1.ClusterSpecification
		if err := fakeClient.Get(ctx, client.ObjectKey{
			Name:      cs.Name,
			Namespace: cs.Namespace,
		}, &retrieved); err != nil {
			t.Fatalf("Failed to retrieve ClusterSpec: %v", err)
		}

		if retrieved.Spec.Kubernetes.MaxVersion != "1.30.0" {
			t.Errorf("Expected Kubernetes.MaxVersion 1.30.0, got %s", retrieved.Spec.Kubernetes.MaxVersion)
		}
	})
}

func TestMetricsRecording(t *testing.T) {
	// Test that metrics are recorded correctly
	t.Run("should record compliance metrics", func(t *testing.T) {
		// Import metrics package
		// metrics.RecordComplianceMetrics(...)
		// This is more of an integration test with Prometheus
		t.Skip("Metrics integration test - requires Prometheus setup")
	})
}

func TestAuditLogging(t *testing.T) {
	t.Run("should create audit logger", func(t *testing.T) {
		ctx := context.Background()
		// auditLog := audit.NewLogger(ctx)
		// Test audit log creation
		_ = ctx
		t.Skip("Audit logging test - requires log aggregation setup")
	})
}

func TestReportAggregation(t *testing.T) {
	s := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	ctx := context.Background()

	t.Run("should aggregate empty report list", func(t *testing.T) {
		// Create aggregator
		// aggregator := aggregation.NewReportAggregator(fakeClient)

		// Test with no reports
		// summary, err := aggregator.GetFleetSummary(ctx, "test-spec")

		_ = ctx
		_ = fakeClient
		t.Skip("Aggregation test - add when aggregation package is fully testable")
	})
}

func TestClientFactory(t *testing.T) {
	t.Run("should create local clients", func(t *testing.T) {
		// Test client factory creation
		t.Skip("Client factory test - requires real Kubernetes config")
	})

	t.Run("should handle invalid cluster targets", func(t *testing.T) {
		// Test error handling
		t.Skip("Client factory error handling - requires real setup")
	})
}

func TestDiscovery(t *testing.T) {
	t.Run("should discover clusters from kubeconfig", func(t *testing.T) {
		// Test cluster discovery
		t.Skip("Discovery test - requires kubeconfig fixture")
	})

	t.Run("should sanitize cluster names", func(t *testing.T) {
		// Test name sanitization logic
		t.Skip("Name sanitization unit test - move to unit tests")
	})
}

func setupScheme(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatalf("Failed to add k8s scheme: %v", err)
	}
	if err := kspecv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("Failed to add kspec scheme: %v", err)
	}
	return s
}

// Helper to wait for condition
func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for condition")
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}
