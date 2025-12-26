package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudcwfranck/kspec/pkg/enforcer"
	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/scanner/checks"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func initCommand() *cobra.Command {
	var (
		kubeconfigPath string
		outputFile     string
		autoYes        bool
		template       string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard for kspec",
		Long: `Interactive setup wizard that helps you get started with kspec.

This command will:
1. Auto-detect your Kubernetes cluster
2. Ask about your security requirements
3. Generate a tailored cluster specification
4. Optionally enforce security policies
5. Optionally set up drift monitoring

Perfect for getting started quickly with sensible defaults.`,
		Example: `  # Run interactive setup wizard
  kspec init

  # Use a pre-configured template
  kspec init --template=production

  # Auto-accept defaults (non-interactive)
  kspec init --auto-yes

  # Save spec to custom location
  kspec init --output my-cluster-spec.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitWizard(kubeconfigPath, outputFile, autoYes, template)
		},
	}

	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "cluster-spec.yaml", "Output file for generated spec")
	cmd.Flags().BoolVarP(&autoYes, "auto-yes", "y", false, "Auto-accept defaults (non-interactive)")
	cmd.Flags().StringVar(&template, "template", "", "Use a pre-configured template (production, development, compliance)")

	return cmd
}

func runInitWizard(kubeconfigPath, outputFile string, autoYes bool, template string) error {
	ctx := context.Background()

	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│ 🎯 kspec Setup Wizard                   │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()

	// Step 1: Detect cluster
	fmt.Println("📡 Step 1: Detecting Kubernetes cluster...")
	client, dynamicClient, clusterVersion, err := detectCluster(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to detect cluster: %w", err)
	}
	fmt.Printf("   ✓ Connected to cluster (Kubernetes %s)\n\n", clusterVersion)

	// Step 2: Choose template or interactive configuration
	var clusterSpec *spec.ClusterSpecification
	if template != "" {
		fmt.Printf("📋 Step 2: Using template: %s\n", template)
		clusterSpec = generateTemplateSpec(template, clusterVersion)
	} else if autoYes {
		fmt.Println("📋 Step 2: Using production defaults...")
		clusterSpec = generateProductionSpec(clusterVersion)
	} else {
		fmt.Println("📋 Step 2: Configure your cluster specification")
		clusterSpec = interactiveSpecBuilder(clusterVersion)
	}

	// Step 3: Generate and save spec
	fmt.Printf("\n💾 Step 3: Saving specification to %s...\n", outputFile)
	if err := saveSpec(clusterSpec, outputFile); err != nil {
		return fmt.Errorf("failed to save spec: %w", err)
	}
	fmt.Printf("   ✓ Specification saved\n\n")

	// Step 4: Scan current cluster
	fmt.Println("🔍 Step 4: Scanning cluster for compliance...")
	scanResults, err := scanCluster(ctx, client, clusterSpec)
	if err != nil {
		fmt.Printf("   ⚠ Scan failed: %v\n", err)
	} else {
		printScanSummary(scanResults)
	}

	// Step 5: Offer to enforce policies
	if !autoYes {
		fmt.Println("\n🛡️  Step 5: Policy Enforcement")
		if askYesNo("Would you like to enforce security policies now?", true) {
			if err := enforcePolicies(ctx, client, dynamicClient, clusterSpec); err != nil {
				fmt.Printf("   ⚠ Policy enforcement failed: %v\n", err)
			}
		}
	}

	// Step 6: Offer to set up drift monitoring
	if !autoYes {
		fmt.Println("\n🔄 Step 6: Drift Monitoring")
		if askYesNo("Would you like to set up automatic drift monitoring?", false) {
			if err := setupDriftMonitoring(clusterSpec, outputFile); err != nil {
				fmt.Printf("   ⚠ Drift monitoring setup failed: %v\n", err)
			}
		}
	}

	// Success summary
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│ ✅ Setup Complete!                      │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("🎉 Your cluster is now configured with kspec!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review your spec: cat %s\n", outputFile)
	fmt.Println("  • Check policies: kubectl get clusterpolicies")
	fmt.Println("  • Scan again: kspec scan --spec cluster-spec.yaml")
	fmt.Println("  • Monitor drift: kspec drift detect --spec cluster-spec.yaml")
	fmt.Println()

	return nil
}

func detectCluster(kubeconfigPath string) (kubernetes.Interface, dynamic.Interface, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, nil, "", err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, "", err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, "", err
	}

	// Get server version
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return client, dynamicClient, "unknown", nil
	}

	return client, dynamicClient, version.GitVersion, nil
}

func interactiveSpecBuilder(clusterVersion string) *spec.ClusterSpecification {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("   What's your primary use case?")
	fmt.Println("   1) Production (Hardened security)")
	fmt.Println("   2) Development (Permissive)")
	fmt.Println("   3) Compliance (Strict policies)")
	fmt.Print("   Choice [1-3] (default: 1): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	fmt.Println()
	fmt.Println("   Security baseline?")
	fmt.Println("   1) Restricted (High security - recommended)")
	fmt.Println("   2) Baseline (Moderate security)")
	fmt.Print("   Choice [1-2] (default: 1): ")

	secChoice, _ := reader.ReadString('\n')
	secChoice = strings.TrimSpace(secChoice)
	if secChoice == "" {
		secChoice = "1"
	}

	// Generate spec based on choices
	switch choice {
	case "2":
		return generateDevelopmentSpec(clusterVersion)
	case "3":
		return generateComplianceSpec(clusterVersion)
	default:
		if secChoice == "2" {
			return generateBaselineSpec(clusterVersion)
		}
		return generateProductionSpec(clusterVersion)
	}
}

func generateProductionSpec(clusterVersion string) *spec.ClusterSpecification {
	return &spec.ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: spec.Metadata{
			Name:        "production-cluster",
			Version:     "1.0.0",
			Description: "Production cluster with hardened security (generated by kspec init)",
		},
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "restricted",
				Audit:   "restricted",
				Warn:    "restricted",
			},
			Network: &spec.NetworkSpec{
				DefaultDeny: true,
			},
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "securityContext.runAsNonRoot", Value: true},
						{Key: "securityContext.allowPrivilegeEscalation", Value: false},
						{Key: "securityContext.capabilities.drop", Value: []interface{}{"ALL"}},
					},
					Forbidden: []spec.FieldRequirement{
						{Key: "securityContext.privileged", Value: true},
						{Key: "hostNetwork", Value: true},
						{Key: "hostPID", Value: true},
						{Key: "hostIPC", Value: true},
					},
				},
				Images: &spec.ImageSpec{
					RequireDigests: true,
				},
			},
			Observability: &spec.ObservabilitySpec{
				Metrics: &spec.MetricsSpec{
					Required: true,
				},
			},
		},
	}
}

func generateDevelopmentSpec(clusterVersion string) *spec.ClusterSpecification {
	return &spec.ClusterSpecification{
		APIVersion: "kspec.dev/v1",
		Kind:       "ClusterSpecification",
		Metadata: spec.Metadata{
			Name:        "development-cluster",
			Version:     "1.0.0",
			Description: "Development cluster with permissive settings (generated by kspec init)",
		},
		Spec: spec.SpecFields{
			Kubernetes: spec.KubernetesSpec{
				MinVersion: "1.26.0",
				MaxVersion: "1.30.0",
			},
			PodSecurity: &spec.PodSecuritySpec{
				Enforce: "baseline",
				Audit:   "baseline",
				Warn:    "restricted",
			},
			Workloads: &spec.WorkloadsSpec{
				Containers: &spec.ContainerSpec{
					Required: []spec.FieldRequirement{
						{Key: "securityContext.runAsNonRoot", Value: true},
					},
					Forbidden: []spec.FieldRequirement{
						{Key: "securityContext.privileged", Value: true},
					},
				},
			},
		},
	}
}

func generateBaselineSpec(clusterVersion string) *spec.ClusterSpecification {
	clusterSpec := generateProductionSpec(clusterVersion)
	clusterSpec.Metadata.Name = "baseline-cluster"
	clusterSpec.Metadata.Description = "Baseline security cluster (generated by kspec init)"
	clusterSpec.Spec.PodSecurity.Enforce = "baseline"
	clusterSpec.Spec.PodSecurity.Audit = "restricted"
	clusterSpec.Spec.Workloads.Images.RequireDigests = false
	return clusterSpec
}

func generateComplianceSpec(clusterVersion string) *spec.ClusterSpecification {
	clusterSpec := generateProductionSpec(clusterVersion)
	clusterSpec.Metadata.Name = "compliance-cluster"
	clusterSpec.Metadata.Description = "Compliance-focused cluster (generated by kspec init)"
	clusterSpec.Spec.RBAC = &spec.RBACSpec{}
	clusterSpec.Spec.Admission = &spec.AdmissionSpec{}
	return clusterSpec
}

func generateTemplateSpec(template, clusterVersion string) *spec.ClusterSpecification {
	switch strings.ToLower(template) {
	case "development", "dev":
		return generateDevelopmentSpec(clusterVersion)
	case "compliance":
		return generateComplianceSpec(clusterVersion)
	case "baseline":
		return generateBaselineSpec(clusterVersion)
	default:
		return generateProductionSpec(clusterVersion)
	}
}

func saveSpec(clusterSpec *spec.ClusterSpecification, filename string) error {
	data, err := spec.MarshalYAML(clusterSpec)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func scanCluster(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.ScanResult, error) {
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
	return s.Scan(ctx, clusterSpec)
}

func printScanSummary(report *scanner.ScanResult) {
	passed := 0
	for _, result := range report.Results {
		if result.Status == scanner.StatusPass {
			passed++
		}
	}
	total := len(report.Results)

	fmt.Printf("   ✓ Scan complete: %d/%d checks passed\n", passed, total)
	if passed < total {
		fmt.Printf("   ⚠ %d issues found - review your cluster configuration\n", total-passed)
	}
}

func enforcePolicies(ctx context.Context, client kubernetes.Interface, dynamicClient dynamic.Interface, clusterSpec *spec.ClusterSpecification) error {
	fmt.Println("   Enforcing security policies...")

	enf := enforcer.NewEnforcer(client, dynamicClient)
	result, err := enf.Enforce(ctx, clusterSpec, enforcer.EnforceOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("   ✓ Enforced %d security policies\n", result.PoliciesApplied)
	return nil
}

func setupDriftMonitoring(clusterSpec *spec.ClusterSpecification, specFile string) error {
	fmt.Println("   Setting up drift monitoring...")
	fmt.Println()
	fmt.Println("   To deploy drift monitoring as a CronJob:")
	fmt.Println("   1. Review the deployment manifests: ls deploy/drift/")
	fmt.Println("   2. Deploy: kubectl apply -k deploy/drift/")
	fmt.Println()
	fmt.Println("   For manual monitoring:")
	fmt.Printf("   kspec drift detect --spec %s --watch\n", specFile)
	fmt.Println()
	return nil
}

func askYesNo(question string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	defaultStr := "Y/n"
	if !defaultYes {
		defaultStr = "y/N"
	}

	fmt.Printf("   %s [%s]: ", question, defaultStr)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}
