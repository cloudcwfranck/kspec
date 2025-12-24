// Package main is the entry point for the kspec CLI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kspec/kspec/pkg/reporter"
	"github.com/kspec/kspec/pkg/scanner"
	"github.com/kspec/kspec/pkg/scanner/checks"
	"github.com/kspec/kspec/pkg/spec"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// Version is the kspec version (can be overridden by build flags)
	version = "1.0.0"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kspec",
		Short: "Kubernetes cluster compliance enforcer",
		Long: `kspec validates Kubernetes clusters against versioned specifications,
enforces security policies, and generates compliance evidence for audits.`,
	}

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newScanCmd())

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kspec v%s\n", version)
		},
	}
}

func newValidateCmd() *cobra.Command {
	var specFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate spec file syntax",
		Long:  `Validate checks that a cluster specification file is syntactically correct.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load spec
			clusterSpec, err := spec.LoadFromFile(specFile)
			if err != nil {
				return fmt.Errorf("failed to load spec: %w", err)
			}

			// Validate spec
			if err := spec.Validate(clusterSpec); err != nil {
				return fmt.Errorf("spec validation failed: %w", err)
			}

			fmt.Printf("✓ Spec file is valid\n")
			fmt.Printf("  Name: %s\n", clusterSpec.Metadata.Name)
			fmt.Printf("  Version: %s\n", clusterSpec.Metadata.Version)
			return nil
		},
	}

	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Path to cluster spec file (required)")
	cmd.MarkFlagRequired("spec")

	return cmd
}

func newScanCmd() *cobra.Command {
	var (
		specFile       string
		kubeconfigPath string
		outputFormat   string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan cluster against specification",
		Long: `Scan validates a Kubernetes cluster against a kspec specification file.
This operation is read-only and safe to run in production.`,
		Example: `  # Scan with JSON output
  kspec scan --spec cluster-spec.yaml --output json

  # Scan with custom kubeconfig
  kspec scan --spec cluster-spec.yaml --kubeconfig ~/.kube/prod-config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load spec
			clusterSpec, err := spec.LoadFromFile(specFile)
			if err != nil {
				return fmt.Errorf("failed to load spec: %w", err)
			}

			// Validate spec
			if err := spec.Validate(clusterSpec); err != nil {
				return fmt.Errorf("spec validation failed: %w", err)
			}

			// Create Kubernetes client
			client, err := createKubernetesClient(kubeconfigPath)
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes client: %w", err)
			}

			// Create scanner with checks
			checks := []scanner.Check{
				&checks.KubernetesVersionCheck{},
			}
			s := scanner.NewScanner(client, checks)

			// Run scan
			fmt.Fprintf(os.Stderr, "Scanning cluster...\n")
			result, err := s.Scan(ctx, clusterSpec)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Output results
			switch outputFormat {
			case "json":
				reporter := reporter.NewJSONReporter(os.Stdout)
				if err := reporter.Report(result); err != nil {
					return fmt.Errorf("failed to output results: %w", err)
				}
			case "text":
				printTextReport(result)
			default:
				return fmt.Errorf("unsupported output format: %s", outputFormat)
			}

			// Exit with code 1 if there are failures
			if result.Summary.Failed > 0 {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Path to cluster spec file (required)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text|json")
	cmd.MarkFlagRequired("spec")

	return cmd
}

// createKubernetesClient creates a Kubernetes client from kubeconfig.
func createKubernetesClient(kubeconfigPath string) (kubernetes.Interface, error) {
	// Use default kubeconfig path if not specified
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		}
	}

	// Build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

// printTextReport prints a human-readable text report.
func printTextReport(result *scanner.ScanResult) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────┐\n")
	fmt.Printf("│ kspec v%s — Compliance Report        │\n", version)
	fmt.Printf("├─────────────────────────────────────────┤\n")
	fmt.Printf("│ Cluster: %-31s │\n", result.Metadata.Cluster.Name)
	fmt.Printf("│ Spec: %-34s │\n", result.Metadata.Spec.Name+" v"+result.Metadata.Spec.Version)
	fmt.Printf("│ Scanned: %-30s │\n", result.Metadata.ScanTime)
	fmt.Printf("└─────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	// Summary
	passRate := 0
	if result.Summary.TotalChecks > 0 {
		passRate = (result.Summary.Passed * 100) / result.Summary.TotalChecks
	}
	fmt.Printf("COMPLIANCE: %d/%d checks passed (%d%%)\n", result.Summary.Passed, result.Summary.TotalChecks, passRate)
	fmt.Printf("\n")

	// Critical failures
	criticalFailures := filterResults(result.Results, scanner.StatusFail, scanner.SeverityCritical)
	if len(criticalFailures) > 0 {
		fmt.Printf("❌ CRITICAL FAILURES (%d)\n", len(criticalFailures))
		fmt.Printf("─────────────────────────\n")
		for _, r := range criticalFailures {
			fmt.Printf("[%s] %s\n", r.Name, r.Message)
			if r.Remediation != "" {
				fmt.Printf("  Fix: %s\n", r.Remediation)
			}
			fmt.Printf("\n")
		}
	}

	// Other failures
	otherFailures := filterResults(result.Results, scanner.StatusFail, "")
	otherFailures = excludeBySeverity(otherFailures, scanner.SeverityCritical)
	if len(otherFailures) > 0 {
		fmt.Printf("⚠️  FAILURES (%d)\n", len(otherFailures))
		fmt.Printf("─────────────────────────\n")
		for _, r := range otherFailures {
			fmt.Printf("[%s] %s\n", r.Name, r.Message)
			if r.Remediation != "" {
				fmt.Printf("  Fix: %s\n", r.Remediation)
			}
			fmt.Printf("\n")
		}
	}

	// Warnings
	warnings := filterResults(result.Results, scanner.StatusWarn, "")
	if len(warnings) > 0 {
		fmt.Printf("⚠️  WARNINGS (%d)\n", len(warnings))
		fmt.Printf("─────────────────\n")
		for _, r := range warnings {
			fmt.Printf("[%s] %s\n", r.Name, r.Message)
			fmt.Printf("\n")
		}
	}

	// Passed checks
	passed := filterResults(result.Results, scanner.StatusPass, "")
	if len(passed) > 0 {
		fmt.Printf("✅ PASSED CHECKS (%d)\n", len(passed))
		fmt.Printf("─────────────────────\n")
		for _, r := range passed {
			fmt.Printf("✓ %s\n", r.Message)
		}
		fmt.Printf("\n")
	}
}

// filterResults filters results by status and optionally by severity.
func filterResults(results []scanner.CheckResult, status scanner.Status, severity scanner.Severity) []scanner.CheckResult {
	var filtered []scanner.CheckResult
	for _, r := range results {
		if r.Status == status {
			if severity == "" || r.Severity == severity {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

// excludeBySeverity excludes results with the given severity.
func excludeBySeverity(results []scanner.CheckResult, severity scanner.Severity) []scanner.CheckResult {
	var filtered []scanner.CheckResult
	for _, r := range results {
		if r.Severity != severity {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
