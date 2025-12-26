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

package aggregation

import (
	"context"
	"fmt"
	"sort"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

// FleetSummary represents aggregated compliance across all clusters
type FleetSummary struct {
	TotalClusters     int
	HealthyClusters   int
	UnhealthyClusters int

	TotalChecks  int
	PassedChecks int
	FailedChecks int

	ClustersWithDrift int
	TotalDriftEvents  int

	LastUpdated time.Time
}

// ClusterCompliance represents compliance status for a single cluster
type ClusterCompliance struct {
	ClusterName string
	ClusterUID  string
	IsLocal     bool

	LastScanTime time.Time
	TotalChecks  int
	PassedChecks int
	FailedChecks int

	HasDrift        bool
	DriftEventCount int

	ComplianceScore float64 // Percentage of passed checks
}

// ReportAggregator aggregates compliance and drift reports across clusters
type ReportAggregator struct {
	client.Client
}

// NewReportAggregator creates a new ReportAggregator
func NewReportAggregator(k8sClient client.Client) *ReportAggregator {
	return &ReportAggregator{
		Client: k8sClient,
	}
}

// GetFleetSummary returns an aggregated view of compliance across all clusters
func (a *ReportAggregator) GetFleetSummary(ctx context.Context, clusterSpecName string) (*FleetSummary, error) {
	// Get all compliance reports for this ClusterSpec across all clusters
	var reports kspecv1alpha1.ComplianceReportList
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpecName,
		},
	}

	if err := a.List(ctx, &reports, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list compliance reports: %w", err)
	}

	// Group reports by cluster (get latest report per cluster)
	latestReports := a.getLatestReportPerCluster(reports.Items)

	summary := &FleetSummary{
		TotalClusters: len(latestReports),
		LastUpdated:   time.Now(),
	}

	for _, report := range latestReports {
		// Aggregate check counts
		summary.TotalChecks += report.Spec.Summary.Total
		summary.PassedChecks += report.Spec.Summary.Passed
		summary.FailedChecks += report.Spec.Summary.Failed

		// Classify cluster health
		if report.Spec.Summary.Failed == 0 {
			summary.HealthyClusters++
		} else {
			summary.UnhealthyClusters++
		}
	}

	// Get drift reports
	var driftReports kspecv1alpha1.DriftReportList
	if err := a.List(ctx, &driftReports, listOpts...); err != nil {
		return summary, nil // Non-fatal: continue without drift data
	}

	latestDrifts := a.getLatestDriftPerCluster(driftReports.Items)
	for _, drift := range latestDrifts {
		if drift.Spec.DriftDetected {
			summary.ClustersWithDrift++
			summary.TotalDriftEvents += len(drift.Spec.Events)
		}
	}

	return summary, nil
}

// GetClusterCompliance returns detailed compliance status for each cluster
func (a *ReportAggregator) GetClusterCompliance(ctx context.Context, clusterSpecName string) ([]ClusterCompliance, error) {
	// Get all compliance reports
	var reports kspecv1alpha1.ComplianceReportList
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpecName,
		},
	}

	if err := a.List(ctx, &reports, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list compliance reports: %w", err)
	}

	// Group by cluster
	latestReports := a.getLatestReportPerCluster(reports.Items)

	// Get drift reports
	var driftReports kspecv1alpha1.DriftReportList
	latestDrifts := make(map[string]*kspecv1alpha1.DriftReport)
	if err := a.List(ctx, &driftReports, listOpts...); err == nil {
		latestDrifts = a.getLatestDriftPerCluster(driftReports.Items)
	}

	// Build cluster compliance list
	result := make([]ClusterCompliance, 0, len(latestReports))
	for clusterName, report := range latestReports {
		compliance := ClusterCompliance{
			ClusterName:  report.Spec.ClusterName,
			ClusterUID:   report.Spec.ClusterUID,
			IsLocal:      report.Spec.ClusterName == "local",
			LastScanTime: report.Spec.ScanTime.Time,
			TotalChecks:  report.Spec.Summary.Total,
			PassedChecks: report.Spec.Summary.Passed,
			FailedChecks: report.Spec.Summary.Failed,
		}

		// Calculate compliance score
		if compliance.TotalChecks > 0 {
			compliance.ComplianceScore = float64(compliance.PassedChecks) / float64(compliance.TotalChecks) * 100
		}

		// Add drift information
		if drift, ok := latestDrifts[clusterName]; ok {
			compliance.HasDrift = drift.Spec.DriftDetected
			compliance.DriftEventCount = len(drift.Spec.Events)
		}

		result = append(result, compliance)
	}

	// Sort by cluster name for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].ClusterName < result[j].ClusterName
	})

	return result, nil
}

// GetFailedChecksByCluster returns all failed checks grouped by cluster
func (a *ReportAggregator) GetFailedChecksByCluster(ctx context.Context, clusterSpecName string) (map[string][]kspecv1alpha1.CheckResult, error) {
	var reports kspecv1alpha1.ComplianceReportList
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpecName,
		},
	}

	if err := a.List(ctx, &reports, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list compliance reports: %w", err)
	}

	latestReports := a.getLatestReportPerCluster(reports.Items)

	result := make(map[string][]kspecv1alpha1.CheckResult)
	for clusterName, report := range latestReports {
		failedChecks := make([]kspecv1alpha1.CheckResult, 0)
		for _, check := range report.Spec.Results {
			if check.Status == "FAIL" {
				failedChecks = append(failedChecks, check)
			}
		}
		if len(failedChecks) > 0 {
			result[clusterName] = failedChecks
		}
	}

	return result, nil
}

// GetDriftEventsByCluster returns all drift events grouped by cluster
func (a *ReportAggregator) GetDriftEventsByCluster(ctx context.Context, clusterSpecName string) (map[string][]kspecv1alpha1.DriftEvent, error) {
	var driftReports kspecv1alpha1.DriftReportList
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpecName,
		},
	}

	if err := a.List(ctx, &driftReports, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list drift reports: %w", err)
	}

	latestDrifts := a.getLatestDriftPerCluster(driftReports.Items)

	result := make(map[string][]kspecv1alpha1.DriftEvent)
	for clusterName, drift := range latestDrifts {
		if drift.Spec.DriftDetected && len(drift.Spec.Events) > 0 {
			result[clusterName] = drift.Spec.Events
		}
	}

	return result, nil
}

// getLatestReportPerCluster returns the most recent compliance report for each cluster
func (a *ReportAggregator) getLatestReportPerCluster(reports []kspecv1alpha1.ComplianceReport) map[string]*kspecv1alpha1.ComplianceReport {
	result := make(map[string]*kspecv1alpha1.ComplianceReport)

	for i := range reports {
		report := &reports[i]
		clusterName := report.Spec.ClusterName

		existing, ok := result[clusterName]
		if !ok || report.Spec.ScanTime.After(existing.Spec.ScanTime.Time) {
			result[clusterName] = report
		}
	}

	return result
}

// getLatestDriftPerCluster returns the most recent drift report for each cluster
func (a *ReportAggregator) getLatestDriftPerCluster(reports []kspecv1alpha1.DriftReport) map[string]*kspecv1alpha1.DriftReport {
	result := make(map[string]*kspecv1alpha1.DriftReport)

	for i := range reports {
		report := &reports[i]
		clusterName := report.Spec.ClusterName

		existing, ok := result[clusterName]
		if !ok || report.Spec.DetectionTime.After(existing.Spec.DetectionTime.Time) {
			result[clusterName] = report
		}
	}

	return result
}

// GetClusterTargets returns all ClusterTarget resources
func (a *ReportAggregator) GetClusterTargets(ctx context.Context, namespace string) ([]kspecv1alpha1.ClusterTarget, error) {
	var targets kspecv1alpha1.ClusterTargetList
	listOpts := []client.ListOption{}
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	if err := a.List(ctx, &targets, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list cluster targets: %w", err)
	}

	return targets.Items, nil
}

// GetUnhealthyClusters returns ClusterTargets that are unreachable
func (a *ReportAggregator) GetUnhealthyClusters(ctx context.Context, namespace string) ([]kspecv1alpha1.ClusterTarget, error) {
	targets, err := a.GetClusterTargets(ctx, namespace)
	if err != nil {
		return nil, err
	}

	unhealthy := make([]kspecv1alpha1.ClusterTarget, 0)
	for _, target := range targets {
		if !target.Status.Reachable {
			unhealthy = append(unhealthy, target)
		}
	}

	return unhealthy, nil
}

// GetClustersByPlatform groups clusters by their platform type
func (a *ReportAggregator) GetClustersByPlatform(ctx context.Context, namespace string) (map[string][]kspecv1alpha1.ClusterTarget, error) {
	targets, err := a.GetClusterTargets(ctx, namespace)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]kspecv1alpha1.ClusterTarget)
	for _, target := range targets {
		platform := target.Status.Platform
		if platform == "" {
			platform = "unknown"
		}
		result[platform] = append(result[platform], target)
	}

	return result, nil
}

// ComplianceHistory represents historical compliance data for a cluster
type ComplianceHistory struct {
	ClusterName string
	DataPoints  []ComplianceDataPoint
}

// ComplianceDataPoint represents a single point in time compliance measurement
type ComplianceDataPoint struct {
	Timestamp       time.Time
	TotalChecks     int
	PassedChecks    int
	FailedChecks    int
	ComplianceScore float64
}

// GetComplianceHistory returns historical compliance data for a cluster
func (a *ReportAggregator) GetComplianceHistory(ctx context.Context, clusterSpecName, clusterName string, limit int) (*ComplianceHistory, error) {
	var reports kspecv1alpha1.ComplianceReportList
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpecName,
			"kspec.io/cluster-name": clusterName,
		},
	}

	if err := a.List(ctx, &reports, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list compliance reports: %w", err)
	}

	// Sort by scan time (newest first)
	sort.Slice(reports.Items, func(i, j int) bool {
		return reports.Items[i].Spec.ScanTime.After(reports.Items[j].Spec.ScanTime.Time)
	})

	// Limit results
	if limit > 0 && len(reports.Items) > limit {
		reports.Items = reports.Items[:limit]
	}

	// Build data points
	dataPoints := make([]ComplianceDataPoint, len(reports.Items))
	for i, report := range reports.Items {
		score := 0.0
		if report.Spec.Summary.Total > 0 {
			score = float64(report.Spec.Summary.Passed) / float64(report.Spec.Summary.Total) * 100
		}

		dataPoints[i] = ComplianceDataPoint{
			Timestamp:       report.Spec.ScanTime.Time,
			TotalChecks:     report.Spec.Summary.Total,
			PassedChecks:    report.Spec.Summary.Passed,
			FailedChecks:    report.Spec.Summary.Failed,
			ComplianceScore: score,
		}
	}

	// Reverse to chronological order (oldest first)
	for i, j := 0, len(dataPoints)-1; i < j; i, j = i+1, j-1 {
		dataPoints[i], dataPoints[j] = dataPoints[j], dataPoints[i]
	}

	return &ComplianceHistory{
		ClusterName: clusterName,
		DataPoints:  dataPoints,
	}, nil
}

// GetRecentActivity returns recent compliance scans across all clusters
func (a *ReportAggregator) GetRecentActivity(ctx context.Context, clusterSpecName string, limit int) ([]ActivityEvent, error) {
	var reports kspecv1alpha1.ComplianceReportList
	listOpts := []client.ListOption{
		client.MatchingLabels{
			"kspec.io/cluster-spec": clusterSpecName,
		},
	}

	if err := a.List(ctx, &reports, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list compliance reports: %w", err)
	}

	// Sort by scan time (newest first)
	sort.Slice(reports.Items, func(i, j int) bool {
		return reports.Items[i].Spec.ScanTime.After(reports.Items[j].Spec.ScanTime.Time)
	})

	// Limit results
	if limit > 0 && len(reports.Items) > limit {
		reports.Items = reports.Items[:limit]
	}

	// Build activity events
	events := make([]ActivityEvent, len(reports.Items))
	for i, report := range reports.Items {
		severity := "info"
		if report.Spec.Summary.Failed > 0 {
			severity = "warning"
		}

		events[i] = ActivityEvent{
			Timestamp:   report.Spec.ScanTime.Time,
			ClusterName: report.Spec.ClusterName,
			EventType:   "compliance_scan",
			Severity:    severity,
			Message:     fmt.Sprintf("Scan completed: %d passed, %d failed", report.Spec.Summary.Passed, report.Spec.Summary.Failed),
		}
	}

	return events, nil
}

// ActivityEvent represents an activity event in the fleet
type ActivityEvent struct {
	Timestamp   time.Time
	ClusterName string
	EventType   string // compliance_scan, drift_detected, remediation, etc.
	Severity    string // info, warning, error
	Message     string
}
