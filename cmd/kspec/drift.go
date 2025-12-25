package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cloudcwfranck/kspec/pkg/drift"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func driftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Detect and remediate configuration drift",
		Long: `Detect when cluster state deviates from specification and optionally auto-remediate.

Drift detection compares the current cluster state against your specification to identify:
- Missing policies (policies that should exist but don't)
- Modified policies (policies that have been changed)
- Compliance violations (new failures in compliance checks)

Automatic remediation can restore drift to the expected state.`,
		Example: `  # Detect drift once
  kspec drift detect --spec cluster-spec.yaml

  # Continuous monitoring
  kspec drift detect --spec cluster-spec.yaml --watch

  # Remediate detected drift
  kspec drift remediate --spec cluster-spec.yaml

  # View drift history
  kspec drift history --spec cluster-spec.yaml`,
	}

	cmd.AddCommand(driftDetectCommand())
	cmd.AddCommand(driftRemediateCommand())
	cmd.AddCommand(driftHistoryCommand())

	return cmd
}

func driftDetectCommand() *cobra.Command {
	var (
		specFile       string
		kubeconfigPath string
		watch          bool
		watchInterval  time.Duration
		outputFormat   string
		outputFile     string
	)

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect configuration drift",
		Long: `Detect drift between desired state (specification) and actual state (cluster).

This command compares:
1. Expected policies (from spec) vs deployed policies (in cluster)
2. Expected compliance (from spec) vs actual compliance (from checks)

Outputs a drift report showing what has changed.`,
		Example: `  # Detect drift once
  kspec drift detect --spec cluster-spec.yaml

  # Watch for drift continuously (check every 5 minutes)
  kspec drift detect --spec cluster-spec.yaml --watch --watch-interval=5m

  # Output drift report to file
  kspec drift detect --spec cluster-spec.yaml --output drift-report.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load spec
			clusterSpec, err := spec.LoadFromFile(specFile)
			if err != nil {
				return fmt.Errorf("failed to load spec: %w", err)
			}

			// Create Kubernetes clients
			client, dynamicClient, err := createClients(kubeconfigPath)
			if err != nil {
				return fmt.Errorf("failed to create clients: %w", err)
			}

			// Watch mode - continuous monitoring
			if watch {
				return runContinuousMonitoring(ctx, client, dynamicClient, clusterSpec, watchInterval)
			}

			// One-time drift detection
			detector := drift.NewDetector(client, dynamicClient)
			report, err := detector.Detect(ctx, clusterSpec, drift.DetectOptions{
				OutputFormat: outputFormat,
				OutputFile:   outputFile,
			})
			if err != nil {
				return fmt.Errorf("drift detection failed: %w", err)
			}

			// Print report
			printDriftReport(report, outputFormat, outputFile)

			// Exit with code 1 if drift detected
			if report.Drift.Detected {
				return fmt.Errorf("drift detected")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Path to cluster spec file (required)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().BoolVar(&watch, "watch", false, "Continuous monitoring mode")
	cmd.Flags().DurationVar(&watchInterval, "watch-interval", 5*time.Minute, "Polling interval for watch mode")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text|json")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "Write report to file")
	cmd.MarkFlagRequired("spec")

	return cmd
}

func driftRemediateCommand() *cobra.Command {
	var (
		specFile       string
		kubeconfigPath string
		dryRun         bool
		force          bool
		types          []string
	)

	cmd := &cobra.Command{
		Use:   "remediate",
		Short: "Remediate detected drift",
		Long: `Automatically fix detected drift to restore cluster to specification.

Remediation actions:
- Missing policies: Create them
- Modified policies: Update them to match spec
- Extra policies: Report (delete with --force)
- Compliance drift: Report (manual remediation required)`,
		Example: `  # Dry-run (show what would be fixed)
  kspec drift remediate --spec cluster-spec.yaml --dry-run

  # Remediate all policy drift
  kspec drift remediate --spec cluster-spec.yaml

  # Remediate specific types only
  kspec drift remediate --spec cluster-spec.yaml --types=policy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load spec
			clusterSpec, err := spec.LoadFromFile(specFile)
			if err != nil {
				return fmt.Errorf("failed to load spec: %w", err)
			}

			// Create Kubernetes clients
			client, dynamicClient, err := createClients(kubeconfigPath)
			if err != nil {
				return fmt.Errorf("failed to create clients: %w", err)
			}

			// Convert type strings to DriftType
			var driftTypes []drift.DriftType
			for _, t := range types {
				driftTypes = append(driftTypes, drift.DriftType(t))
			}

			// Detect and remediate
			report, err := drift.RemediateAll(ctx, client, dynamicClient, clusterSpec, drift.RemediateOptions{
				DryRun: dryRun,
				Types:  driftTypes,
				Force:  force,
			})
			if err != nil {
				return fmt.Errorf("remediation failed: %w", err)
			}

			// Print remediation report
			printRemediationReport(report, dryRun)

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Path to cluster spec file (required)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be fixed without applying changes")
	cmd.Flags().BoolVar(&force, "force", false, "Delete extra policies (use with caution)")
	cmd.Flags().StringSliceVar(&types, "types", []string{"policy"}, "Drift types to remediate: policy,compliance")
	cmd.MarkFlagRequired("spec")

	return cmd
}

func driftHistoryCommand() *cobra.Command {
	var (
		specFile       string
		kubeconfigPath string
		since          string
		outputFormat   string
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show drift detection history",
		Long:  `Display historical drift events and statistics.`,
		Example: `  # Show all drift history
  kspec drift history --spec cluster-spec.yaml

  # Show drift from last 24 hours
  kspec drift history --spec cluster-spec.yaml --since=24h

  # Output as JSON
  kspec drift history --spec cluster-spec.yaml --output=json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse since duration (currently unused, will be used when storage is connected)
			if since != "" {
				_, err := time.ParseDuration(since)
				if err != nil {
					return fmt.Errorf("invalid duration '%s': %w", since, err)
				}
				// TODO: use parsed duration to filter history when storage is connected
			}

			// For now, just return empty history
			// In a full implementation, this would read from storage
			history := &drift.DriftHistory{
				Events: []drift.DriftEvent{},
				Stats: drift.DriftStats{
					TotalEvents: 0,
				},
			}

			printDriftHistory(history, outputFormat)
			return nil
		},
	}

	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Path to cluster spec file (required)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().StringVar(&since, "since", "", "Show history since duration (e.g., 24h, 7d)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text|json")
	cmd.MarkFlagRequired("spec")

	return cmd
}

// Helper functions

func createClients(kubeconfigPath string) (kubernetes.Interface, dynamic.Interface, error) {
	// Use default kubeconfig path if not specified
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	client, err := createKubernetesClient(kubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return client, dynamicClient, nil
}

func runContinuousMonitoring(ctx context.Context, client kubernetes.Interface, dynamicClient dynamic.Interface, clusterSpec *spec.ClusterSpecification, interval time.Duration) error {
	fmt.Printf("Starting continuous drift monitoring (interval: %s)\n", interval)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	monitor, err := drift.NewMonitor(client, dynamicClient, &drift.MonitorConfig{
		Interval:     interval,
		EnabledTypes: []drift.DriftType{drift.DriftTypePolicy, drift.DriftTypeCompliance},
		AutoRemediate: false,
	})
	if err != nil {
		return err
	}

	return monitor.Start(ctx, clusterSpec)
}

func printDriftReport(report *drift.DriftReport, format, outputFile string) {
	if format == "json" {
		data, _ := json.MarshalIndent(report, "", "  ")
		if outputFile != "" {
			os.WriteFile(outputFile, data, 0644)
			fmt.Printf("Drift report written to: %s\n", outputFile)
		} else {
			fmt.Println(string(data))
		}
		return
	}

	// Text format
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────┐\n")
	fmt.Printf("│ kspec v%s — Drift Detection        │\n", version)
	fmt.Printf("└─────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	if !report.Drift.Detected {
		fmt.Printf("[OK] No drift detected\n")
		fmt.Printf("\n")
		return
	}

	fmt.Printf("[DRIFT] Detected %d drift events\n", report.Drift.Counts.Total)
	fmt.Printf("Severity: %s\n", report.Drift.Severity)
	fmt.Printf("\n")

	if report.Drift.Counts.Policies > 0 {
		fmt.Printf("Policy Drift: %d\n", report.Drift.Counts.Policies)
	}
	if report.Drift.Counts.Compliance > 0 {
		fmt.Printf("Compliance Drift: %d\n", report.Drift.Counts.Compliance)
	}
	fmt.Printf("\n")

	fmt.Printf("Drift Events:\n")
	fmt.Printf("─────────────\n")
	for _, event := range report.Events {
		fmt.Printf("[%s] %s: %s\n", event.Severity, event.Resource.Path, event.Message)
	}
	fmt.Printf("\n")
}

func printRemediationReport(report *drift.DriftReport, dryRun bool) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────┐\n")
	fmt.Printf("│ kspec v%s — Drift Remediation      │\n", version)
	fmt.Printf("└─────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	if dryRun {
		fmt.Printf("Mode: Dry-run (no changes applied)\n\n")
	}

	remediatedCount := 0
	failedCount := 0
	manualCount := 0

	for _, event := range report.Events {
		if event.Remediation != nil {
			switch event.Remediation.Status {
			case drift.DriftStatusRemediated:
				remediatedCount++
			case drift.DriftStatusFailed:
				failedCount++
			case drift.DriftStatusManualRequired:
				manualCount++
			}
		}
	}

	fmt.Printf("Remediation Summary:\n")
	fmt.Printf("───────────────────\n")
	fmt.Printf("Total events: %d\n", len(report.Events))
	fmt.Printf("Remediated: %d\n", remediatedCount)
	fmt.Printf("Failed: %d\n", failedCount)
	fmt.Printf("Manual required: %d\n", manualCount)
	fmt.Printf("\n")

	if remediatedCount > 0 {
		fmt.Printf("Remediated:\n")
		for _, event := range report.Events {
			if event.Remediation != nil && event.Remediation.Status == drift.DriftStatusRemediated {
				fmt.Printf("  [OK] %s: %s\n", event.Resource.Path, event.Remediation.Details)
			}
		}
		fmt.Printf("\n")
	}

	if failedCount > 0 {
		fmt.Printf("[ERROR] Failed remediations:\n")
		for _, event := range report.Events {
			if event.Remediation != nil && event.Remediation.Status == drift.DriftStatusFailed {
				fmt.Printf("  [FAIL] %s: %s\n", event.Resource.Path, event.Remediation.Error)
			}
		}
		fmt.Printf("\n")
	}
}

func printDriftHistory(history *drift.DriftHistory, format string) {
	if format == "json" {
		data, _ := json.MarshalIndent(history, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────┐\n")
	fmt.Printf("│ kspec v%s — Drift History          │\n", version)
	fmt.Printf("└─────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	fmt.Printf("Total events: %d\n", history.Stats.TotalEvents)
	if history.Stats.TotalEvents > 0 {
		fmt.Printf("First event: %s\n", history.Stats.FirstEvent.Format(time.RFC3339))
		fmt.Printf("Last event: %s\n", history.Stats.LastEvent.Format(time.RFC3339))
		fmt.Printf("Remediation success rate: %.1f%%\n", history.Stats.RemediationSuccessRate*100)
	}
	fmt.Printf("\n")
}
