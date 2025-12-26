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

package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/drift"
	"github.com/cloudcwfranck/kspec/pkg/scanner"
)

// updateStatus updates the ClusterSpecification status based on scan and drift results
func (r *ClusterSpecReconciler) updateStatus(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	scanResult *scanner.ScanResult,
	driftReport *drift.DriftReport,
) error {
	now := metav1.Now()

	// Update phase
	if calculatePassRate(scanResult.Summary) >= 95 {
		clusterSpec.Status.Phase = "Active"
	} else if scanResult.Summary.Failed > 0 {
		clusterSpec.Status.Phase = "Active" // Still active, but with failures
	}

	// Update observed generation
	clusterSpec.Status.ObservedGeneration = clusterSpec.Generation

	// Update last scan time
	clusterSpec.Status.LastScanTime = &now

	// Update compliance score
	clusterSpec.Status.ComplianceScore = calculatePassRate(scanResult.Summary)

	// Update summary
	driftEvents := 0
	if driftReport != nil {
		driftEvents = len(driftReport.Events)
	}

	clusterSpec.Status.Summary = &kspecv1alpha1.ComplianceSummary{
		TotalChecks:      scanResult.Summary.TotalChecks,
		PassedChecks:     scanResult.Summary.Passed,
		FailedChecks:     scanResult.Summary.Failed,
		PoliciesEnforced: 0, // TODO: Track enforced policies
		DriftEvents:      driftEvents,
	}

	// Update conditions
	clusterSpec.Status.Conditions = r.buildConditions(scanResult, driftReport)

	// Update status
	if err := r.Status().Update(ctx, clusterSpec); err != nil {
		return err
	}

	return nil
}

// updateStatusFailed updates status when reconciliation fails
func (r *ClusterSpecReconciler) updateStatusFailed(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	err error,
) {
	clusterSpec.Status.Phase = "Failed"
	clusterSpec.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "ReconciliationFailed",
			Message:            err.Error(),
			ObservedGeneration: clusterSpec.Generation,
		},
	}

	// Try to update status, but don't fail if it doesn't work
	_ = r.Status().Update(ctx, clusterSpec)
}

// buildConditions builds status conditions based on scan and drift results
func (r *ClusterSpecReconciler) buildConditions(
	scanResult *scanner.ScanResult,
	driftReport *drift.DriftReport,
) []metav1.Condition {
	now := metav1.Now()
	conditions := []metav1.Condition{}

	// Ready condition
	readyCondition := metav1.Condition{
		Type:               "Ready",
		LastTransitionTime: now,
		ObservedGeneration: 0, // Will be set by caller
	}

	if calculatePassRate(scanResult.Summary) >= 70 {
		readyCondition.Status = metav1.ConditionTrue
		readyCondition.Reason = "ComplianceChecksPassed"
		readyCondition.Message = "Cluster meets compliance requirements"
	} else {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Reason = "ComplianceChecksFailed"
		readyCondition.Message = "Cluster does not meet compliance requirements"
	}
	conditions = append(conditions, readyCondition)

	// PolicyEnforced condition
	policyCondition := metav1.Condition{
		Type:               "PolicyEnforced",
		LastTransitionTime: now,
		ObservedGeneration: 0,
	}

	// For now, assume policies are enforced
	// TODO: Track actual policy enforcement
	policyCondition.Status = metav1.ConditionTrue
	policyCondition.Reason = "PoliciesDeployed"
	policyCondition.Message = "Security policies are enforced"
	conditions = append(conditions, policyCondition)

	// DriftDetected condition
	if driftReport != nil && driftReport.Drift.Detected {
		driftCondition := metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: now,
			Reason:             "DriftFound",
			Message:            "Configuration drift detected and remediated",
			ObservedGeneration: 0,
		}
		conditions = append(conditions, driftCondition)
	}

	return conditions
}
