package fleet

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
	"github.com/cloudcwfranck/kspec/pkg/metrics"
)

// FleetAggregator aggregates compliance data across multiple clusters
type FleetAggregator struct {
	Client        client.Client
	ClientFactory *clientpkg.ClusterClientFactory
}

// FleetSummary contains aggregated fleet-wide statistics
type FleetSummary struct {
	TotalClusters     int
	HealthyClusters   int
	TotalChecks       int
	PassedChecks      int
	FailedChecks      int
	ClustersWithDrift int
	AverageCompliance float64
	ClusterReports    []ClusterReport
	LastUpdated       time.Time
}

// ClusterReport contains per-cluster statistics
type ClusterReport struct {
	ClusterName       string
	ClusterUID        string
	IsHealthy         bool
	TotalChecks       int
	PassedChecks      int
	FailedChecks      int
	ComplianceScore   float64
	HasDrift          bool
	DriftEventCount   int
	LastScanTime      time.Time
	EnforcementMode   string
	EnforcementActive bool
}

// NewFleetAggregator creates a new fleet aggregator
func NewFleetAggregator(client client.Client, factory *clientpkg.ClusterClientFactory) *FleetAggregator {
	return &FleetAggregator{
		Client:        client,
		ClientFactory: factory,
	}
}

// AggregateFleetCompliance aggregates compliance data from all clusters
func (f *FleetAggregator) AggregateFleetCompliance(ctx context.Context) (*FleetSummary, error) {
	log := log.FromContext(ctx)
	log.Info("Aggregating fleet-wide compliance data")

	// Get all ClusterSpecs
	var clusterSpecs kspecv1alpha1.ClusterSpecificationList
	if err := f.Client.List(ctx, &clusterSpecs); err != nil {
		return nil, fmt.Errorf("failed to list cluster specs: %w", err)
	}

	// Get all ClusterTargets
	var clusterTargets kspecv1alpha1.ClusterTargetList
	if err := f.Client.List(ctx, &clusterTargets); err != nil {
		return nil, fmt.Errorf("failed to list cluster targets: %w", err)
	}

	summary := &FleetSummary{
		ClusterReports: make([]ClusterReport, 0),
		LastUpdated:    time.Now(),
	}

	// Build cluster map
	clusterMap := make(map[string]*kspecv1alpha1.ClusterTarget)
	for i := range clusterTargets.Items {
		clusterMap[clusterTargets.Items[i].Name] = &clusterTargets.Items[i]
	}

	// Include local cluster
	localReport, err := f.getLocalClusterReport(ctx, clusterSpecs.Items)
	if err != nil {
		log.Error(err, "Failed to get local cluster report")
	} else {
		summary.ClusterReports = append(summary.ClusterReports, localReport)
		if localReport.IsHealthy {
			summary.HealthyClusters++
		}
		summary.TotalClusters++
		summary.TotalChecks += localReport.TotalChecks
		summary.PassedChecks += localReport.PassedChecks
		summary.FailedChecks += localReport.FailedChecks
		if localReport.HasDrift {
			summary.ClustersWithDrift++
		}
	}

	// Aggregate remote clusters in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	reportChan := make(chan ClusterReport, len(clusterTargets.Items))

	for _, target := range clusterTargets.Items {
		wg.Add(1)
		go func(target kspecv1alpha1.ClusterTarget) {
			defer wg.Done()

			report, err := f.getRemoteClusterReport(ctx, &target, clusterSpecs.Items)
			if err != nil {
				log.Error(err, "Failed to get remote cluster report", "cluster", target.Name)
				// Create error report
				report = ClusterReport{
					ClusterName: target.Name,
					ClusterUID:  string(target.UID),
					IsHealthy:   false,
				}
			}

			reportChan <- report
		}(target)
	}

	// Wait for all reports
	go func() {
		wg.Wait()
		close(reportChan)
	}()

	// Collect reports
	for report := range reportChan {
		mu.Lock()
		summary.ClusterReports = append(summary.ClusterReports, report)
		summary.TotalClusters++
		if report.IsHealthy {
			summary.HealthyClusters++
		}
		summary.TotalChecks += report.TotalChecks
		summary.PassedChecks += report.PassedChecks
		summary.FailedChecks += report.FailedChecks
		if report.HasDrift {
			summary.ClustersWithDrift++
		}
		mu.Unlock()
	}

	// Calculate average compliance
	if summary.TotalClusters > 0 {
		totalScore := 0.0
		for _, report := range summary.ClusterReports {
			totalScore += report.ComplianceScore
		}
		summary.AverageCompliance = totalScore / float64(summary.TotalClusters)
	}

	// Update fleet metrics
	metrics.UpdateFleetMetrics(
		summary.TotalClusters,
		summary.HealthyClusters,
		summary.TotalChecks,
		summary.PassedChecks,
		summary.FailedChecks,
		summary.ClustersWithDrift,
	)

	log.Info("Fleet aggregation complete",
		"totalClusters", summary.TotalClusters,
		"healthyClusters", summary.HealthyClusters,
		"averageCompliance", summary.AverageCompliance)

	return summary, nil
}

// getLocalClusterReport gets compliance report for the local cluster
func (f *FleetAggregator) getLocalClusterReport(
	ctx context.Context,
	clusterSpecs []kspecv1alpha1.ClusterSpecification,
) (ClusterReport, error) {
	report := ClusterReport{
		ClusterName: "local",
		ClusterUID:  "local",
		IsHealthy:   true,
	}

	// Get latest compliance reports for local cluster
	var reports kspecv1alpha1.ComplianceReportList
	if err := f.Client.List(ctx, &reports, client.MatchingLabels{
		"kspec.io/cluster-name": "local",
	}); err != nil {
		return report, fmt.Errorf("failed to list compliance reports: %w", err)
	}

	// Find most recent report
	var latestReport *kspecv1alpha1.ComplianceReport
	for i := range reports.Items {
		if latestReport == nil || reports.Items[i].CreationTimestamp.After(latestReport.CreationTimestamp.Time) {
			latestReport = &reports.Items[i]
		}
	}

	if latestReport != nil {
		report.TotalChecks = latestReport.Spec.Summary.Total
		report.PassedChecks = latestReport.Spec.Summary.Passed
		report.FailedChecks = latestReport.Spec.Summary.Failed
		report.LastScanTime = latestReport.CreationTimestamp.Time

		if report.TotalChecks > 0 {
			report.ComplianceScore = float64(report.PassedChecks) / float64(report.TotalChecks) * 100
		}
	}

	// Check drift
	var driftReports kspecv1alpha1.DriftReportList
	if err := f.Client.List(ctx, &driftReports, client.MatchingLabels{
		"kspec.io/cluster-name": "local",
	}); err == nil && len(driftReports.Items) > 0 {
		latestDrift := &driftReports.Items[0]
		for i := range driftReports.Items {
			if driftReports.Items[i].CreationTimestamp.After(latestDrift.CreationTimestamp.Time) {
				latestDrift = &driftReports.Items[i]
			}
		}
		report.HasDrift = latestDrift.Spec.DriftDetected
		if report.HasDrift {
			report.DriftEventCount = len(latestDrift.Spec.Events)
		}
	}

	// Get enforcement mode from ClusterSpec
	for _, spec := range clusterSpecs {
		if spec.Spec.Enforcement != nil && spec.Spec.Enforcement.Enabled {
			report.EnforcementActive = true
			report.EnforcementMode = spec.Spec.Enforcement.Mode
			break
		}
	}

	return report, nil
}

// getRemoteClusterReport gets compliance report for a remote cluster
func (f *FleetAggregator) getRemoteClusterReport(
	ctx context.Context,
	target *kspecv1alpha1.ClusterTarget,
	clusterSpecs []kspecv1alpha1.ClusterSpecification,
) (ClusterReport, error) {
	report := ClusterReport{
		ClusterName: target.Name,
		ClusterUID:  string(target.UID),
		IsHealthy:   false,
	}

	// Create client for remote cluster
	kubeClient, _, clusterInfo, err := f.ClientFactory.CreateClientsForClusterTarget(ctx, target)
	if err != nil {
		return report, fmt.Errorf("failed to create client: %w", err)
	}

	report.IsHealthy = true

	// Get compliance reports from remote cluster
	reports, err := kubeClient.CoreV1().ConfigMaps("kspec-system").List(ctx, metav1.ListOptions{
		LabelSelector: "app=kspec,type=compliance-report",
	})
	if err != nil {
		return report, fmt.Errorf("failed to list reports: %w", err)
	}

	// Parse most recent report
	if len(reports.Items) > 0 {
		// In a real implementation, parse the ConfigMap data
		// For now, use placeholder values
		report.TotalChecks = 100
		report.PassedChecks = 85
		report.FailedChecks = 15
		report.ComplianceScore = 85.0
		report.LastScanTime = time.Now()
	}

	// Get enforcement configuration
	report.EnforcementActive = clusterInfo.AllowEnforcement
	for _, spec := range clusterSpecs {
		if spec.Spec.Enforcement != nil && spec.Spec.Enforcement.Enabled {
			report.EnforcementMode = spec.Spec.Enforcement.Mode
			break
		}
	}

	return report, nil
}

// StartPeriodicAggregation starts periodic fleet aggregation
func (f *FleetAggregator) StartPeriodicAggregation(ctx context.Context, interval time.Duration) {
	log := log.FromContext(ctx)
	log.Info("Starting periodic fleet aggregation", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping periodic fleet aggregation")
			return
		case <-ticker.C:
			if _, err := f.AggregateFleetCompliance(ctx); err != nil {
				log.Error(err, "Fleet aggregation failed")
			}
		}
	}
}
