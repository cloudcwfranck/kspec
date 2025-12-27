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
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

func TestClusterSpecCreation(t *testing.T) {
	fw := NewTestFramework(t)
	defer fw.Cleanup(t)

	ctx := context.Background()

	t.Run("should create ClusterSpecification successfully", func(t *testing.T) {
		cs, err := fw.CreateClusterSpec(ctx, "test-clusterspec", "default")
		if err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// Verify it was created
		var retrieved kspecv1alpha1.ClusterSpecification
		if err := fw.Client.Get(ctx, client.ObjectKey{
			Name: cs.Name,
			// ClusterSpecification is cluster-scoped, no namespace
		}, &retrieved); err != nil {
			t.Fatalf("Failed to retrieve ClusterSpec: %v", err)
		}

		if retrieved.Spec.Kubernetes.MinVersion != "1.28.0" {
			t.Errorf("Expected Kubernetes.MinVersion 1.28.0, got %s", retrieved.Spec.Kubernetes.MinVersion)
		}
	})

	t.Run("should generate compliance report", func(t *testing.T) {
		cs, err := fw.CreateClusterSpec(ctx, "test-compliance", "default")
		if err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// Wait for compliance report to be generated
		report, err := fw.WaitForComplianceReport(ctx, cs.Name, 60*time.Second)
		if err != nil {
			t.Fatalf("Failed to get ComplianceReport: %v", err)
		}

		if report.Spec.Summary.Total == 0 {
			t.Error("Expected compliance report to have checks, got 0")
		}

		t.Logf("Compliance report generated: %d total checks, %d passed, %d failed",
			report.Spec.Summary.Total,
			report.Spec.Summary.Passed,
			report.Spec.Summary.Failed)
	})

	t.Run("should update status on reconciliation", func(t *testing.T) {
		cs, err := fw.CreateClusterSpec(ctx, "test-status", "default")
		if err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// Wait for status to be updated
		err = fw.WaitForClusterSpecReady(ctx, cs.Name, "", 30*time.Second)
		if err != nil {
			t.Fatalf("ClusterSpec did not become ready: %v", err)
		}

		// Verify status fields
		var updated kspecv1alpha1.ClusterSpecification
		if err := fw.Client.Get(ctx, client.ObjectKey{
			Name: cs.Name,
		}, &updated); err != nil {
			t.Fatalf("Failed to retrieve updated ClusterSpec: %v", err)
		}

		if updated.Status.LastScanTime == nil {
			t.Error("Expected LastScanTime to be set")
		}
	})
}

func TestClusterSpecWithClusterRef(t *testing.T) {
	fw := NewTestFramework(t)
	defer fw.Cleanup(t)

	ctx := context.Background()

	t.Run("should handle ClusterRef to non-existent cluster", func(t *testing.T) {
		cs := &kspecv1alpha1.ClusterSpecification{}
		cs.Name = "test-remote-spec"
		// ClusterSpecification is cluster-scoped, no namespace
		cs.Spec.Kubernetes.MinVersion = "1.28.0"
		cs.Spec.ClusterRef = &kspecv1alpha1.ClusterReference{
			Name:      "non-existent-cluster",
			Namespace: "default",
		}

		if err := fw.Client.Create(ctx, cs); err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// Wait a bit for reconciliation
		time.Sleep(5 * time.Second)

		// Verify status reflects the error
		var updated kspecv1alpha1.ClusterSpecification
		if err := fw.Client.Get(ctx, client.ObjectKey{
			Name: cs.Name,
		}, &updated); err != nil {
			t.Fatalf("Failed to retrieve ClusterSpec: %v", err)
		}

		if updated.Status.Phase == "Ready" {
			t.Error("Expected ClusterSpec to not be Ready when ClusterTarget doesn't exist")
		}
	})

	t.Run("should work with valid ClusterRef", func(t *testing.T) {
		// Create ClusterTarget first
		ct, err := fw.CreateClusterTarget(ctx, "test-target", "default")
		if err != nil {
			t.Fatalf("Failed to create ClusterTarget: %v", err)
		}

		// Create ClusterSpec referencing it
		cs := &kspecv1alpha1.ClusterSpecification{}
		cs.Name = "test-remote-valid"
		// ClusterSpecification is cluster-scoped, no namespace
		cs.Spec.Kubernetes.MinVersion = "1.28.0"
		cs.Spec.ClusterRef = &kspecv1alpha1.ClusterReference{
			Name:      ct.Name,
			Namespace: ct.Namespace,
		}

		if err := fw.Client.Create(ctx, cs); err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// ClusterSpec should be created successfully
		// (Note: It may not become fully Ready since the target cluster is fake,
		// but it should at least be created and attempt reconciliation)
		var updated kspecv1alpha1.ClusterSpecification
		if err := fw.Client.Get(ctx, client.ObjectKey{
			Name: cs.Name,
		}, &updated); err != nil {
			t.Fatalf("Failed to retrieve ClusterSpec: %v", err)
		}

		if updated.Spec.ClusterRef.Name != ct.Name {
			t.Errorf("Expected ClusterRef.Name %s, got %s", ct.Name, updated.Spec.ClusterRef.Name)
		}
	})
}

func TestClusterSpecDeletion(t *testing.T) {
	fw := NewTestFramework(t)
	defer fw.Cleanup(t)

	ctx := context.Background()

	t.Run("should delete ClusterSpecification and cleanup reports", func(t *testing.T) {
		cs, err := fw.CreateClusterSpec(ctx, "test-deletion", "default")
		if err != nil {
			t.Fatalf("Failed to create ClusterSpec: %v", err)
		}

		// Wait for at least one report
		_, err = fw.WaitForComplianceReport(ctx, cs.Name, 60*time.Second)
		if err != nil {
			t.Logf("Warning: No compliance report generated before deletion: %v", err)
		}

		// Delete the ClusterSpec
		if err := fw.Client.Delete(ctx, cs); err != nil {
			t.Fatalf("Failed to delete ClusterSpec: %v", err)
		}

		// Wait for deletion to complete
		time.Sleep(5 * time.Second)

		// Verify it's gone
		var deleted kspecv1alpha1.ClusterSpecification
		err = fw.Client.Get(ctx, client.ObjectKey{
			Name: cs.Name,
		}, &deleted)

		if err == nil {
			t.Error("Expected ClusterSpec to be deleted, but it still exists")
		}

		// Verify reports are also cleaned up (owner references)
		var reports kspecv1alpha1.ComplianceReportList
		if err := fw.Client.List(ctx, &reports, client.MatchingLabels{
			"kspec.io/cluster-spec": cs.Name,
		}); err == nil {
			if len(reports.Items) > 0 {
				t.Errorf("Expected reports to be cleaned up, found %d reports", len(reports.Items))
			}
		}
	})
}
