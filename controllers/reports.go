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
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
	"github.com/cloudcwfranck/kspec/pkg/drift"
	"github.com/cloudcwfranck/kspec/pkg/scanner"
)

// createComplianceReport creates a ComplianceReport CR from scan results
func (r *ClusterSpecReconciler) createComplianceReport(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	scanResult *scanner.ScanResult,
	clusterInfo *clientpkg.ClusterInfo,
) error {
	log := log.FromContext(ctx)

	// Generate report name with timestamp and cluster name
	timestamp := time.Now().UTC().Format("20060102-150405")
	reportName := fmt.Sprintf("%s-%s-%s", clusterInfo.Name, clusterSpec.Name, timestamp)

	// Convert scanner.CheckResult to kspecv1alpha1.CheckResult
	results := make([]kspecv1alpha1.CheckResult, len(scanResult.Results))
	for i, result := range scanResult.Results {
		results[i] = kspecv1alpha1.CheckResult{
			Name:     result.Name,
			Category: inferCategory(result.Name),
			Status:   normalizeStatus(string(result.Status)),
			Severity: normalizeSeverity(string(result.Severity)),
			Message:  result.Message,
			Details:  nil, // TODO: Convert evidence to runtime.RawExtension
		}
	}

	// Create ComplianceReport
	report := &kspecv1alpha1.ComplianceReport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      reportName,
			Namespace: ReportNamespace,
			Labels: map[string]string{
				"kspec.io/cluster-spec": clusterSpec.Name,
				"kspec.io/cluster-name": clusterInfo.Name,
				"kspec.io/report-type":  "compliance",
			},
		},
		Spec: kspecv1alpha1.ComplianceReportSpec{
			ClusterSpecRef: kspecv1alpha1.ObjectReference{
				Name:    clusterSpec.Name,
				Version: clusterSpec.ResourceVersion,
			},
			ClusterName: clusterInfo.Name,
			ClusterUID:  clusterInfo.UID,
			ScanTime:    metav1.Time{Time: time.Now().UTC()},
			Summary: kspecv1alpha1.ReportSummary{
				Total:    scanResult.Summary.TotalChecks,
				Passed:   scanResult.Summary.Passed,
				Failed:   scanResult.Summary.Failed,
				PassRate: calculatePassRate(scanResult.Summary),
			},
			Results: results,
		},
		Status: kspecv1alpha1.ComplianceReportStatus{
			Phase: "Completed",
		},
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetOwnerReference(clusterSpec, report, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create the report
	if err := r.Create(ctx, report); err != nil {
		return fmt.Errorf("failed to create ComplianceReport: %w", err)
	}

	log.Info("ComplianceReport created", "name", reportName, "passRate", report.Spec.Summary.PassRate)
	return nil
}

// createDriftReport creates a DriftReport CR from drift detection results
func (r *ClusterSpecReconciler) createDriftReport(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	driftReport *drift.DriftReport,
	clusterInfo *clientpkg.ClusterInfo,
) error {
	log := log.FromContext(ctx)

	// Generate report name with timestamp and cluster name
	timestamp := time.Now().UTC().Format("20060102-150405")
	reportName := fmt.Sprintf("%s-%s-drift-%s", clusterInfo.Name, clusterSpec.Name, timestamp)

	// Convert drift.DriftEvent to kspecv1alpha1.DriftEvent
	events := make([]kspecv1alpha1.DriftEvent, len(driftReport.Events))
	for i, event := range driftReport.Events {
		var resourceRef *kspecv1alpha1.ResourceReference
		if event.Resource.Kind != "" {
			resourceRef = &kspecv1alpha1.ResourceReference{
				Kind:      event.Resource.Kind,
				Name:      event.Resource.Name,
				Namespace: event.Resource.Namespace,
			}
		}

		var remediation *kspecv1alpha1.RemediationAction
		if event.Remediation != nil {
			var appliedAt *metav1.Time
			if !event.Remediation.Timestamp.IsZero() {
				appliedAt = &metav1.Time{Time: event.Remediation.Timestamp}
			}

			remediation = &kspecv1alpha1.RemediationAction{
				Action:    string(event.Remediation.Action),
				Status:    string(event.Remediation.Status),
				AppliedAt: appliedAt,
				Error:     event.Remediation.Error,
			}
		}

		events[i] = kspecv1alpha1.DriftEvent{
			Type:        normalizeType(string(event.Type)),
			Severity:    string(event.Severity), // Severity is already lowercase in both drift package and CRD
			Resource:    resourceRef,
			DriftType:   normalizeDriftKind(event.DriftKind),
			Check:       "", // drift.DriftEvent has no Check field
			Message:     event.Message,
			Expected:    nil, // TODO: Convert to runtime.RawExtension
			Actual:      nil, // TODO: Convert to runtime.RawExtension
			Remediation: remediation,
		}
	}

	// Determine severity
	severity := "low"
	for _, event := range driftReport.Events {
		if event.Severity == drift.SeverityHigh || event.Severity == drift.SeverityCritical {
			severity = "high"
			break
		} else if event.Severity == drift.SeverityMedium {
			severity = "medium"
		}
	}

	// Create DriftReport
	report := &kspecv1alpha1.DriftReport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      reportName,
			Namespace: ReportNamespace,
			Labels: map[string]string{
				"kspec.io/cluster-spec": clusterSpec.Name,
				"kspec.io/cluster-name": clusterInfo.Name,
				"kspec.io/severity":     severity,
			},
		},
		Spec: kspecv1alpha1.DriftReportSpec{
			ClusterSpecRef: kspecv1alpha1.ObjectReference{
				Name:    clusterSpec.Name,
				Version: clusterSpec.ResourceVersion,
			},
			ClusterName:   clusterInfo.Name,
			ClusterUID:    clusterInfo.UID,
			DetectionTime: metav1.Time{Time: time.Now().UTC()},
			DriftDetected: driftReport.Drift.Detected,
			Severity:      severity,
			Events:        events,
		},
		Status: kspecv1alpha1.DriftReportStatus{
			Phase:            "Completed",
			TotalEvents:      len(events),
			RemediatedEvents: countRemediatedEvents(events),
			PendingEvents:    countPendingEvents(events),
		},
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetOwnerReference(clusterSpec, report, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create the report
	if err := r.Create(ctx, report); err != nil {
		return fmt.Errorf("failed to create DriftReport: %w", err)
	}

	log.Info("DriftReport created", "name", reportName, "events", len(events))
	return nil
}

// cleanupOldReports deletes old reports to maintain retention policy
func (r *ClusterSpecReconciler) cleanupOldReports(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, clusterInfo *clientpkg.ClusterInfo) error {
	log := log.FromContext(ctx)

	// Cleanup old ComplianceReports
	if err := r.cleanupOldComplianceReports(ctx, clusterSpec, clusterInfo); err != nil {
		log.Error(err, "Failed to cleanup old ComplianceReports")
	}

	// Cleanup old DriftReports
	if err := r.cleanupOldDriftReports(ctx, clusterSpec, clusterInfo); err != nil {
		log.Error(err, "Failed to cleanup old DriftReports")
	}

	return nil
}

// cleanupOldComplianceReports removes old ComplianceReports beyond retention limit
func (r *ClusterSpecReconciler) cleanupOldComplianceReports(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, clusterInfo *clientpkg.ClusterInfo) error {
	var reportList kspecv1alpha1.ComplianceReportList
	if err := r.List(ctx, &reportList,
		&client.ListOptions{
			Namespace: ReportNamespace,
		},
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpec.Name,
			"kspec.io/cluster-name": clusterInfo.Name,
		},
	); err != nil {
		return err
	}

	// Sort by creation timestamp (newest first)
	sort.Slice(reportList.Items, func(i, j int) bool {
		return reportList.Items[i].CreationTimestamp.After(reportList.Items[j].CreationTimestamp.Time)
	})

	// Delete reports beyond retention limit
	for i := MaxReportsToKeep; i < len(reportList.Items); i++ {
		if err := r.Delete(ctx, &reportList.Items[i]); err != nil {
			return err
		}
	}

	return nil
}

// cleanupOldDriftReports removes old DriftReports beyond retention limit
func (r *ClusterSpecReconciler) cleanupOldDriftReports(ctx context.Context, clusterSpec *kspecv1alpha1.ClusterSpecification, clusterInfo *clientpkg.ClusterInfo) error {
	var reportList kspecv1alpha1.DriftReportList
	if err := r.List(ctx, &reportList,
		&client.ListOptions{
			Namespace: ReportNamespace,
		},
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpec.Name,
			"kspec.io/cluster-name": clusterInfo.Name,
		},
	); err != nil {
		return err
	}

	// Sort by creation timestamp (newest first)
	sort.Slice(reportList.Items, func(i, j int) bool {
		return reportList.Items[i].CreationTimestamp.After(reportList.Items[j].CreationTimestamp.Time)
	})

	// Delete reports beyond retention limit
	for i := MaxReportsToKeep; i < len(reportList.Items); i++ {
		if err := r.Delete(ctx, &reportList.Items[i]); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

func countRemediatedEvents(events []kspecv1alpha1.DriftEvent) int {
	count := 0
	for _, event := range events {
		if event.Remediation != nil && event.Remediation.Status == "success" {
			count++
		}
	}
	return count
}

func countPendingEvents(events []kspecv1alpha1.DriftEvent) int {
	count := 0
	for _, event := range events {
		if event.Remediation != nil && (event.Remediation.Status == "pending" || event.Remediation.Status == "manual-required") {
			count++
		}
	}
	return count
}

// inferCategory infers the check category from the check name
func inferCategory(checkName string) string {
	// Check names follow the pattern "category.subcategory" (e.g., "kubernetes.version")
	// Extract the first part as the category
	if len(checkName) == 0 {
		return "unknown"
	}

	// Find the first dot
	for i, c := range checkName {
		if c == '.' {
			return checkName[:i]
		}
	}

	// No dot found, return the whole name
	return checkName
}

// normalizeStatus converts scanner status values to CRD-compliant capitalized values
// CRD only allows: Pass, Fail, Error (no Skip)
func normalizeStatus(status string) string {
	switch status {
	case "pass", "passed":
		return "Pass"
	case "fail", "failed":
		return "Fail"
	case "skip", "skipped":
		// Skip is not a valid CRD status, treat as Pass since skipped checks don't fail
		return "Pass"
	case "error":
		return "Error"
	case "Pass", "Fail", "Error":
		// Already in correct format
		return status
	default:
		// Unknown status, default to Error
		return "Error"
	}
}

// normalizeSeverity converts scanner severity values to CRD-compliant values
func normalizeSeverity(severity string) string {
	switch severity {
	case "low", "Low":
		return "Low"
	case "medium", "Medium", "moderate", "Moderate":
		return "Medium"
	case "high", "High":
		return "High"
	case "critical", "Critical":
		return "Critical"
	default:
		// Default to Low for empty or unknown severity
		if severity == "" {
			return "Low"
		}
		return "Low"
	}
}

// normalizeType converts drift type values to CRD-compliant capitalized values
// DriftReport CRD requires: Policy, Compliance, Configuration (capitalized)
func normalizeType(driftType string) string {
	switch driftType {
	case "policy":
		return "Policy"
	case "compliance":
		return "Compliance"
	case "configuration":
		return "Configuration"
	case "Policy", "Compliance", "Configuration":
		// Already in correct format
		return driftType
	default:
		// Default to Policy for unknown types
		return "Policy"
	}
}

// normalizeDriftKind maps drift kinds to CRD-compliant values
// DriftReport CRD requires: deleted, modified, violation
func normalizeDriftKind(kind string) string {
	switch kind {
	case "missing":
		// Map "missing" to "deleted" as they represent the same concept
		return "deleted"
	case "deleted", "modified", "violation", "extra":
		// Already valid CRD values
		return kind
	default:
		// Default to modified for unknown kinds
		return "modified"
	}
}
