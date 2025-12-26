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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/aggregation"
)

var (
	dashboardKubeconfig  string
	dashboardWatch       bool
	dashboardInterval    int
	dashboardClusterSpec string
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Display real-time compliance dashboard",
	Long: `Display a live dashboard showing fleet-wide compliance status, drift detection,
and cluster health across all managed clusters.`,
	Example: `  # Show dashboard once
  kspec dashboard

  # Live updating dashboard (refresh every 30s)
  kspec dashboard --watch --interval 30

  # Dashboard for specific ClusterSpec
  kspec dashboard --cluster-spec prod-baseline`,
	RunE: runDashboard,
}

func init() {
	dashboardCmd.Flags().StringVar(&dashboardKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	dashboardCmd.Flags().BoolVarP(&dashboardWatch, "watch", "w", false, "Watch mode - continuously update dashboard")
	dashboardCmd.Flags().IntVar(&dashboardInterval, "interval", 10, "Refresh interval in seconds (when using --watch)")
	dashboardCmd.Flags().StringVar(&dashboardClusterSpec, "cluster-spec", "", "Filter by specific ClusterSpec name")
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	k8sClient, err := createDashboardClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create aggregator
	aggregator := aggregation.NewReportAggregator(k8sClient)

	if dashboardWatch {
		// Watch mode - clear screen and refresh
		return runWatchMode(ctx, aggregator)
	}

	// Single display
	return displayDashboard(ctx, aggregator)
}

func runWatchMode(ctx context.Context, aggregator *aggregation.ReportAggregator) error {
	ticker := time.NewTicker(time.Duration(dashboardInterval) * time.Second)
	defer ticker.Stop()

	for {
		// Clear screen
		fmt.Print("\033[H\033[2J")

		if err := displayDashboard(ctx, aggregator); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		fmt.Printf("\n[Press Ctrl+C to exit | Refreshing every %ds]\n", dashboardInterval)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return nil
		}
	}
}

func displayDashboard(ctx context.Context, aggregator *aggregation.ReportAggregator) error {
	// Get ClusterSpecs to monitor
	var clusterSpecs kspecv1alpha1.ClusterSpecificationList
	listOpts := []client.ListOption{}
	if dashboardClusterSpec != "" {
		listOpts = append(listOpts, client.MatchingFields{"metadata.name": dashboardClusterSpec})
	}

	if err := aggregator.List(ctx, &clusterSpecs, listOpts...); err != nil {
		return fmt.Errorf("failed to list ClusterSpecs: %w", err)
	}

	if len(clusterSpecs.Items) == 0 {
		fmt.Println("No ClusterSpecifications found. Deploy some ClusterSpecs to see compliance data.")
		return nil
	}

	// Print header
	printHeader()

	// For each ClusterSpec, show compliance data
	for _, cs := range clusterSpecs.Items {
		if err := displayClusterSpecDashboard(ctx, aggregator, &cs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to get data for %s: %v\n", cs.Name, err)
		}
	}

	return nil
}

func displayClusterSpecDashboard(ctx context.Context, aggregator *aggregation.ReportAggregator, cs *kspecv1alpha1.ClusterSpecification) error {
	// Get fleet summary for this ClusterSpec
	summary, err := aggregator.GetFleetSummary(ctx, cs.Name)
	if err != nil {
		return err
	}

	// Get per-cluster compliance
	clusterCompliance, err := aggregator.GetClusterCompliance(ctx, cs.Name)
	if err != nil {
		return err
	}

	// Get cluster targets health
	targets, err := aggregator.GetClusterTargets(ctx, cs.Namespace)
	if err != nil {
		// Non-fatal
		targets = []kspecv1alpha1.ClusterTarget{}
	}

	// Print summary section
	printSummary(cs.Name, summary)

	// Print cluster details
	printClusterTable(clusterCompliance, targets)

	// Print recent failures if any
	if summary.FailedChecks > 0 {
		printRecentFailures(ctx, aggregator, cs.Name)
	}

	return nil
}

func printHeader() {
	fmt.Println("┌────────────────────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ %-74s │\n", "kspec Compliance Dashboard")
	fmt.Printf("│ %-74s │\n", fmt.Sprintf("Updated: %s", time.Now().Format("2006-01-02 15:04:05")))
	fmt.Println("└────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func printSummary(clusterSpecName string, summary *aggregation.FleetSummary) {
	fmt.Printf("📋 ClusterSpec: %s\n", clusterSpecName)
	fmt.Println("─────────────────────────────────────────────────────────────────────────────")

	// Calculate compliance percentage
	compliancePercent := 0.0
	if summary.TotalChecks > 0 {
		compliancePercent = float64(summary.PassedChecks) / float64(summary.TotalChecks) * 100
	}

	// Choose emoji based on compliance
	statusEmoji := "✅"
	if compliancePercent < 80 {
		statusEmoji = "❌"
	} else if compliancePercent < 95 {
		statusEmoji = "⚠️"
	}

	fmt.Printf("  %s Compliance: %.1f%% (%d/%d checks passed)\n",
		statusEmoji, compliancePercent, summary.PassedChecks, summary.TotalChecks)

	fmt.Printf("  🏢 Clusters:   %d total | %d healthy | %d unhealthy\n",
		summary.TotalClusters, summary.HealthyClusters, summary.UnhealthyClusters)

	if summary.ClustersWithDrift > 0 {
		fmt.Printf("  ⚡ Drift:      %d clusters with drift (%d events total)\n",
			summary.ClustersWithDrift, summary.TotalDriftEvents)
	} else {
		fmt.Printf("  ✨ Drift:      No drift detected\n")
	}

	fmt.Println()
}

func printClusterTable(compliance []aggregation.ClusterCompliance, targets []kspecv1alpha1.ClusterTarget) {
	if len(compliance) == 0 {
		fmt.Println("  No compliance data available yet.")
		fmt.Println()
		return
	}

	// Create target lookup map
	targetMap := make(map[string]*kspecv1alpha1.ClusterTarget)
	for i := range targets {
		targetMap[targets[i].Name] = &targets[i]
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CLUSTER\tCOMPLIANCE\tCHECKS\tDRIFT\tPLATFORM\tNODES\tLAST SCAN\tSTATUS")
	fmt.Fprintln(w, "───────\t──────────\t──────\t─────\t────────\t─────\t─────────\t──────")

	for _, c := range compliance {
		// Format compliance score
		complianceStr := fmt.Sprintf("%.1f%%", c.ComplianceScore)
		if c.ComplianceScore >= 95 {
			complianceStr = "✓ " + complianceStr
		} else if c.ComplianceScore >= 80 {
			complianceStr = "⚠ " + complianceStr
		} else {
			complianceStr = "✗ " + complianceStr
		}

		// Format checks
		checksStr := fmt.Sprintf("%d/%d", c.PassedChecks, c.TotalChecks)

		// Format drift
		driftStr := "-"
		if c.HasDrift {
			driftStr = fmt.Sprintf("⚡ %d events", c.DriftEventCount)
		} else {
			driftStr = "✓ None"
		}

		// Get platform and nodes from target
		platform := "Unknown"
		nodes := "-"
		status := "✓"
		if target, ok := targetMap[c.ClusterName]; ok {
			if target.Status.Platform != "" {
				platform = target.Status.Platform
			}
			if target.Status.NodeCount > 0 {
				nodes = fmt.Sprintf("%d", target.Status.NodeCount)
			}
			if !target.Status.Reachable {
				status = "✗ Unreachable"
			}
		} else if c.IsLocal {
			platform = "Local"
			status = "✓"
		}

		// Format last scan time
		lastScan := "Never"
		if !c.LastScanTime.IsZero() {
			duration := time.Since(c.LastScanTime)
			if duration < time.Minute {
				lastScan = "Just now"
			} else if duration < time.Hour {
				lastScan = fmt.Sprintf("%dm ago", int(duration.Minutes()))
			} else if duration < 24*time.Hour {
				lastScan = fmt.Sprintf("%dh ago", int(duration.Hours()))
			} else {
				lastScan = fmt.Sprintf("%dd ago", int(duration.Hours()/24))
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			c.ClusterName,
			complianceStr,
			checksStr,
			driftStr,
			platform,
			nodes,
			lastScan,
			status,
		)
	}

	w.Flush()
	fmt.Println()
}

func printRecentFailures(ctx context.Context, aggregator *aggregation.ReportAggregator, clusterSpecName string) {
	failedChecks, err := aggregator.GetFailedChecksByCluster(ctx, clusterSpecName)
	if err != nil || len(failedChecks) == 0 {
		return
	}

	fmt.Println("❌ Recent Compliance Failures")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────")

	count := 0
	for clusterName, checks := range failedChecks {
		for _, check := range checks {
			if count >= 5 { // Limit to 5 most recent failures
				break
			}
			fmt.Printf("  [%s] %s: %s\n", clusterName, check.Name, check.Message)
			count++
		}
		if count >= 5 {
			break
		}
	}

	if count == 5 {
		fmt.Printf("  ... and %d more failures (use kubectl to view all reports)\n",
			getTotalFailures(failedChecks)-5)
	}

	fmt.Println()
}

func getTotalFailures(failedChecks map[string][]kspecv1alpha1.CheckResult) int {
	total := 0
	for _, checks := range failedChecks {
		total += len(checks)
	}
	return total
}

func createDashboardClient() (client.Client, error) {
	// Get kubeconfig path
	kubeconfigPath := dashboardKubeconfig
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		}
	}

	// Build config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	// Create scheme
	scheme, err := createScheme()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheme: %w", err)
	}

	// Create client
	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return k8sClient, nil
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := kspecv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

// Helper to truncate long strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Helper to create a progress bar
func progressBar(percent float64, width int) string {
	filled := int(percent / 100.0 * float64(width))
	empty := width - filled

	bar := "["
	bar += strings.Repeat("█", filled)
	bar += strings.Repeat("░", empty)
	bar += "]"

	return bar
}
