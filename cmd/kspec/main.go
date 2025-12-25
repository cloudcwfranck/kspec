// Package main is the entry point for the kspec CLI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudcwfranck/kspec/pkg/enforcer"
	"github.com/cloudcwfranck/kspec/pkg/reporter"
	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/scanner/checks"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var (
	// Version information (injected by goreleaser at build time)
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "manual"
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
	rootCmd.AddCommand(newEnforceCmd())
	rootCmd.AddCommand(driftCommand())

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if verbose {
				fmt.Printf("kspec version: %s\n", version)
				fmt.Printf("commit: %s\n", commit)
				fmt.Printf("built: %s\n", date)
				fmt.Printf("built by: %s\n", builtBy)
			} else {
				fmt.Printf("kspec %s\n", version)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed version information")

	return cmd
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

  # Scan with OSCAL compliance report
  kspec scan --spec cluster-spec.yaml --output oscal > report.json

  # Scan with SARIF security report
  kspec scan --spec cluster-spec.yaml --output sarif > results.sarif

  # Scan with Markdown documentation
  kspec scan --spec cluster-spec.yaml --output markdown > COMPLIANCE.md

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
			checkList := []scanner.Check{
				&checks.KubernetesVersionCheck{},
				&checks.PodSecurityStandardsCheck{},
				&checks.NetworkPolicyCheck{},
				&checks.WorkloadSecurityCheck{},
				&checks.RBACCheck{},
				&checks.AdmissionCheck{},
				&checks.ObservabilityCheck{},
			}
			s := scanner.NewScanner(client, checkList)

			// Run scan
			fmt.Fprintf(os.Stderr, "Scanning cluster...\n")
			result, err := s.Scan(ctx, clusterSpec)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Output results
			switch outputFormat {
			case "json":
				r := reporter.NewJSONReporter(os.Stdout)
				if err := r.Report(result); err != nil {
					return fmt.Errorf("failed to output results: %w", err)
				}
			case "oscal":
				r := reporter.NewOSCALReporter(os.Stdout)
				if err := r.Report(result); err != nil {
					return fmt.Errorf("failed to output results: %w", err)
				}
			case "sarif":
				r := reporter.NewSARIFReporter(os.Stdout)
				if err := r.Report(result); err != nil {
					return fmt.Errorf("failed to output results: %w", err)
				}
			case "markdown":
				r := reporter.NewMarkdownReporter(os.Stdout)
				if err := r.Report(result); err != nil {
					return fmt.Errorf("failed to output results: %w", err)
				}
			case "text":
				printTextReport(result)
			default:
				return fmt.Errorf("unsupported output format: %s (supported: text, json, oscal, sarif, markdown)", outputFormat)
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
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text|json|oscal|sarif|markdown")
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
		fmt.Printf("[CRITICAL] FAILURES (%d)\n", len(criticalFailures))
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
		fmt.Printf("[FAIL] FAILURES (%d)\n", len(otherFailures))
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
		fmt.Printf("[WARN] WARNINGS (%d)\n", len(warnings))
		fmt.Printf("─────────────────\n")
		for _, r := range warnings {
			fmt.Printf("[%s] %s\n", r.Name, r.Message)
			fmt.Printf("\n")
		}
	}

	// Passed checks
	passed := filterResults(result.Results, scanner.StatusPass, "")
	if len(passed) > 0 {
		fmt.Printf("[PASS] PASSED CHECKS (%d)\n", len(passed))
		fmt.Printf("─────────────────────\n")
		for _, r := range passed {
			fmt.Printf("  %s\n", r.Message)
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

func newEnforceCmd() *cobra.Command {
	var (
		specFile       string
		kubeconfigPath string
		dryRun         bool
		skipInstall    bool
		outputFile     string
	)

	cmd := &cobra.Command{
		Use:   "enforce",
		Short: "Generate and deploy Kyverno policies from specification",
		Long: `Enforce generates Kyverno ClusterPolicy resources from a kspec specification
and optionally deploys them to the cluster. This enables proactive policy enforcement
to prevent non-compliant workloads from being deployed.`,
		Example: `  # Generate policies (dry-run, see what would be created)
  kspec enforce --spec cluster-spec.yaml --dry-run

  # Deploy policies to cluster (requires Kyverno installed)
  kspec enforce --spec cluster-spec.yaml

  # Save generated policies to file
  kspec enforce --spec cluster-spec.yaml --dry-run --output policies.yaml

  # Skip Kyverno installation check
  kspec enforce --spec cluster-spec.yaml --skip-install`,
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

			// Create dynamic client for applying policies
			// Use default kubeconfig path if not specified
			kubeconfigToUse := kubeconfigPath
			if kubeconfigToUse == "" {
				kubeconfigToUse = os.Getenv("KUBECONFIG")
				if kubeconfigToUse == "" {
					kubeconfigToUse = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
				}
			}

			config, err := clientcmd.BuildConfigFromFlags("", kubeconfigToUse)
			if err != nil {
				return fmt.Errorf("failed to build config: %w", err)
			}
			dynamicClient, err := dynamic.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("failed to create dynamic client: %w", err)
			}

			// Create enforcer
			enf := enforcer.NewEnforcer(client, dynamicClient)

			// Enforce policies
			fmt.Fprintf(os.Stderr, "Generating policies from spec...\n")
			result, err := enf.Enforce(ctx, clusterSpec, enforcer.EnforceOptions{
				DryRun:      dryRun,
				SkipInstall: skipInstall,
			})
			if err != nil {
				return fmt.Errorf("enforcement failed: %w", err)
			}

			// Print results
			printEnforceResult(result, dryRun, outputFile)

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFile, "spec", "s", "", "Path to cluster spec file (required)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Generate policies without deploying them")
	cmd.Flags().BoolVar(&skipInstall, "skip-install", false, "Skip Kyverno installation check")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Save generated policies to file (YAML)")
	cmd.MarkFlagRequired("spec")

	return cmd
}

// printEnforceResult prints the enforcement result.
func printEnforceResult(result *enforcer.EnforceResult, dryRun bool, outputFile string) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────┐\n")
	fmt.Printf("│ kspec v%s — Policy Enforcement       │\n", version)
	fmt.Printf("└─────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	// Kyverno status
	if result.KyvernoInstalled {
		fmt.Printf("[OK] Kyverno Status: Installed\n")
		if result.KyvernoVersion != "" {
			fmt.Printf("     Version: %s\n", result.KyvernoVersion)
		}
	} else {
		fmt.Printf("[ERROR] Kyverno Status: Not Installed\n")
	}
	fmt.Printf("\n")

	// Policies generated
	fmt.Printf("Policies Generated: %d\n", result.PoliciesGenerated)

	if dryRun {
		fmt.Printf("Mode: Dry-run (policies not deployed)\n")
	} else {
		fmt.Printf("Policies Applied: %d\n", result.PoliciesApplied)
	}
	fmt.Printf("\n")

	// List generated policies
	if result.PoliciesGenerated > 0 {
		fmt.Printf("Generated Policies:\n")
		fmt.Printf("───────────────────\n")
		for i, policy := range result.Policies {
			// Extract policy name from unstructured object
			policyName := fmt.Sprintf("policy-%d", i+1)
			if unstruct, ok := policy.(interface{ GetName() string }); ok {
				policyName = unstruct.GetName()
			}
			fmt.Printf("  %d. %s\n", i+1, policyName)
		}
		fmt.Printf("\n")
	}

	// Save to file if requested
	if outputFile != "" && result.PoliciesGenerated > 0 {
		if err := savePolicies(result.Policies, outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to save policies to file: %v\n", err)
		} else {
			fmt.Printf("[OK] Policies saved to: %s\n\n", outputFile)
		}
	}

	// Errors
	if len(result.Errors) > 0 {
		fmt.Printf("[ERROR] Policy Application Errors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
		fmt.Printf("\n")
	}

	// Next steps
	if dryRun {
		fmt.Printf("Next Steps:\n")
		fmt.Printf("───────────\n")
		if !result.KyvernoInstalled {
			fmt.Printf("1. Install Kyverno in your cluster\n")
			fmt.Printf("2. Run: kspec enforce --spec <file> (without --dry-run)\n")
		} else {
			fmt.Printf("1. Review the generated policies above\n")
			fmt.Printf("2. Run: kspec enforce --spec <file> (without --dry-run) to deploy\n")
		}
		fmt.Printf("\n")
	} else if result.PoliciesApplied > 0 {
		fmt.Printf("[OK] Policies successfully deployed\n")
		fmt.Printf("\n")
		fmt.Printf("Verify policies:\n")
		fmt.Printf("  kubectl get clusterpolicies\n")
		fmt.Printf("  kubectl describe clusterpolicy <policy-name>\n")
		fmt.Printf("\n")
	}
}

// savePolicies saves generated policies to a YAML file.
func savePolicies(policies []runtime.Object, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for i, policy := range policies {
		// Add document separator before each policy (except the first)
		if i > 0 {
			fmt.Fprintln(file, "---")
		}

		// Use Kubernetes YAML marshaler which properly handles TypeMeta
		yamlBytes, err := yaml.Marshal(policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy %d: %w", i, err)
		}

		// Write the YAML to file
		if _, err := file.Write(yamlBytes); err != nil {
			return fmt.Errorf("failed to write policy %d: %w", i, err)
		}
	}

	return nil
}
