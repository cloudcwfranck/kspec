//go:build e2e
// +build e2e

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

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/controllers"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
)

// TestFramework provides test infrastructure for E2E tests
type TestFramework struct {
	Config        *rest.Config
	Client        client.Client
	Scheme        *runtime.Scheme
	TestEnv       *envtest.Environment
	Cancel        context.CancelFunc
	ClientFactory *clientpkg.ClusterClientFactory
	Manager       ctrl.Manager
}

// NewTestFramework creates a new test framework
func NewTestFramework(t *testing.T) *TestFramework {
	logf.SetLogger(zap.New(zap.WriteTo(testLogger{t}), zap.UseDevMode(true)))

	// Create scheme
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatalf("Failed to add k8s scheme: %v", err)
	}
	if err := kspecv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("Failed to add kspec scheme: %v", err)
	}

	// Setup test environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("Failed to start test environment: %v", err)
	}

	// Create client
	k8sClient, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		testEnv.Stop()
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: s,
		Metrics: metricsserver.Options{
			BindAddress: "0", // Disable metrics server for tests
		},
	})
	if err != nil {
		testEnv.Stop()
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create client factory
	clientFactory := clientpkg.NewClusterClientFactory(cfg, k8sClient)

	// Setup controllers
	if err := setupControllers(mgr, cfg, clientFactory); err != nil {
		testEnv.Stop()
		t.Fatalf("Failed to setup controllers: %v", err)
	}

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("Manager failed: %v", err)
		}
	}()

	// Wait for manager to be ready
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		cancel()
		testEnv.Stop()
		t.Fatal("Failed to sync cache")
	}

	return &TestFramework{
		Config:        cfg,
		Client:        k8sClient,
		Scheme:        s,
		TestEnv:       testEnv,
		Cancel:        cancel,
		ClientFactory: clientFactory,
		Manager:       mgr,
	}
}

// Cleanup tears down the test environment
func (f *TestFramework) Cleanup(t *testing.T) {
	if f.Cancel != nil {
		f.Cancel()
	}
	if f.TestEnv != nil {
		if err := f.TestEnv.Stop(); err != nil {
			t.Errorf("Failed to stop test environment: %v", err)
		}
	}
}

// CreateClusterSpec creates a ClusterSpecification for testing
func (f *TestFramework) CreateClusterSpec(ctx context.Context, name, namespace string) (*kspecv1alpha1.ClusterSpecification, error) {
	cs := &kspecv1alpha1.ClusterSpecification{}
	cs.Name = name
	// ClusterSpecification is cluster-scoped, so don't set namespace
	cs.Spec.Kubernetes.MinVersion = "1.28.0"
	cs.Spec.Kubernetes.MaxVersion = "1.30.0"

	if err := f.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("failed to create ClusterSpec: %w", err)
	}

	return cs, nil
}

// CreateClusterTarget creates a ClusterTarget for testing
func (f *TestFramework) CreateClusterTarget(ctx context.Context, name, namespace string) (*kspecv1alpha1.ClusterTarget, error) {
	ct := &kspecv1alpha1.ClusterTarget{}
	ct.Name = name
	ct.Namespace = namespace
	ct.Spec.APIServerURL = "https://test-cluster:6443"
	ct.Spec.AuthMode = "serviceAccount"
	ct.Spec.AllowEnforcement = false

	if err := f.Client.Create(ctx, ct); err != nil {
		return nil, fmt.Errorf("failed to create ClusterTarget: %w", err)
	}

	return ct, nil
}

// WaitForClusterSpecReady waits for ClusterSpec to be ready
func (f *TestFramework) WaitForClusterSpecReady(ctx context.Context, name, namespace string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ClusterSpec %s to be ready", name)
		case <-ticker.C:
			var cs kspecv1alpha1.ClusterSpecification
			// ClusterSpecification is cluster-scoped, so don't use namespace
			if err := f.Client.Get(ctx, client.ObjectKey{Name: name}, &cs); err != nil {
				continue
			}
			if cs.Status.Phase == "Ready" || cs.Status.Phase == "Synced" || cs.Status.Phase == "Active" {
				return nil
			}
		}
	}
}

// WaitForComplianceReport waits for a ComplianceReport to be created
func (f *TestFramework) WaitForComplianceReport(ctx context.Context, clusterSpecName string, timeout time.Duration) (*kspecv1alpha1.ComplianceReport, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for ComplianceReport for ClusterSpec %s", clusterSpecName)
		case <-ticker.C:
			var reports kspecv1alpha1.ComplianceReportList
			if err := f.Client.List(ctx, &reports, client.MatchingLabels{
				"kspec.io/cluster-spec": clusterSpecName,
			}); err != nil {
				continue
			}
			if len(reports.Items) > 0 {
				return &reports.Items[0], nil
			}
		}
	}
}

// setupControllers sets up all controllers for testing
func setupControllers(mgr ctrl.Manager, cfg *rest.Config, clientFactory *clientpkg.ClusterClientFactory) error {
	// Setup ClusterSpecification controller
	if err := (&controllers.ClusterSpecReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		LocalConfig:   cfg,
		ClientFactory: clientFactory,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to setup ClusterSpec controller: %w", err)
	}

	// Setup ClusterTarget controller
	if err := (&controllers.ClusterTargetReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		LocalConfig:   cfg,
		ClientFactory: clientFactory,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to setup ClusterTarget controller: %w", err)
	}

	return nil
}

// testLogger implements logr.LogSink for test output
type testLogger struct {
	t *testing.T
}

func (l testLogger) Write(p []byte) (n int, err error) {
	l.t.Log(string(p))
	return len(p), nil
}
