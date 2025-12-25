package enforcer

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudcwfranck/kspec/pkg/enforcer/kyverno"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Enforcer orchestrates policy enforcement.
type Enforcer struct {
	client           kubernetes.Interface
	dynamicClient    dynamic.Interface
	kyvernoGen       *kyverno.Generator
	kyvernoInstaller *kyverno.Installer
	kyvernoValidator *kyverno.Validator
}

// NewEnforcer creates a new policy enforcer.
func NewEnforcer(client kubernetes.Interface, dynamicClient dynamic.Interface) *Enforcer {
	return &Enforcer{
		client:           client,
		dynamicClient:    dynamicClient,
		kyvernoGen:       kyverno.NewGenerator(),
		kyvernoInstaller: kyverno.NewInstaller(),
		kyvernoValidator: kyverno.NewValidator(),
	}
}

// EnforceOptions contains options for policy enforcement.
type EnforceOptions struct {
	DryRun      bool
	SkipInstall bool
}

// EnforceResult contains the results of policy enforcement.
type EnforceResult struct {
	KyvernoInstalled  bool
	KyvernoVersion    string
	PoliciesGenerated int
	PoliciesApplied   int
	Policies          []runtime.Object
	Errors            []string
}

// Enforce generates and optionally deploys policies from a cluster specification.
func (e *Enforcer) Enforce(ctx context.Context, clusterSpec *spec.ClusterSpecification, opts EnforceOptions) (*EnforceResult, error) {
	result := &EnforceResult{
		Policies: []runtime.Object{},
		Errors:   []string{},
	}

	// Check if Kyverno is installed
	installed, err := e.kyvernoInstaller.IsInstalled(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to check Kyverno installation: %w", err)
	}

	result.KyvernoInstalled = installed
	fmt.Printf("DEBUG: Kyverno installed check: %v\n", installed)

	if installed {
		version, err := e.kyvernoInstaller.GetVersion(ctx, e.client)
		if err == nil {
			result.KyvernoVersion = version
			fmt.Printf("DEBUG: Kyverno version: %s\n", version)
		} else {
			fmt.Printf("DEBUG: Failed to get Kyverno version: %v\n", err)
		}
	} else {
		fmt.Printf("DEBUG: Kyverno not detected as installed\n")
	}

	// Generate policies
	policies, err := e.kyvernoGen.GeneratePolicies(clusterSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate policies: %w", err)
	}

	result.Policies = policies
	result.PoliciesGenerated = len(policies)

	// Validate generated policies before deployment
	if err := e.validatePolicies(policies); err != nil {
		return nil, fmt.Errorf("policy validation failed: %w", err)
	}

	// If dry-run, stop here
	if opts.DryRun {
		fmt.Printf("DEBUG: Dry-run mode, skipping policy deployment\n")
		return result, nil
	}

	fmt.Printf("DEBUG: Not dry-run, proceeding with deployment. Installed=%v, SkipInstall=%v\n", installed, opts.SkipInstall)

	// Check if Kyverno is installed before applying
	if !installed && !opts.SkipInstall {
		fmt.Printf("DEBUG: Kyverno not installed and skip-install not set, returning error\n")
		return result, fmt.Errorf("Kyverno is not installed. Install it first or use --skip-install flag.\n\n%s",
			e.kyvernoInstaller.GetInstallInstructions())
	}

	// Apply policies (if not dry-run and Kyverno is installed)
	if installed {
		fmt.Printf("DEBUG: Calling applyPolicies with %d policies\n", len(policies))
		applied, applyErrors := e.applyPolicies(ctx, policies)
		result.PoliciesApplied = applied
		result.Errors = applyErrors

		// CRITICAL: If policies failed to apply, return error
		if len(applyErrors) > 0 {
			return nil, fmt.Errorf("failed to apply %d policies: %v", len(applyErrors), applyErrors)
		}
		fmt.Printf("DEBUG: Successfully applied all policies\n")
	} else {
		fmt.Printf("DEBUG: Kyverno not installed, skipping policy application\n")
	}

	return result, nil
}

// applyPolicies applies Kyverno policies to the cluster.
func (e *Enforcer) applyPolicies(ctx context.Context, policies []runtime.Object) (int, []string) {
	applied := 0
	errors := []string{}

	fmt.Printf("\n=== POLICY DEPLOYMENT START ===\n")
	fmt.Printf("Total policies to deploy: %d\n", len(policies))

	// Define Kyverno ClusterPolicy GVR
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	for i, policyObj := range policies {
		fmt.Printf("\n[Policy %d/%d]\n", i+1, len(policies))

		// Log the policy type
		fmt.Printf("  Type: %T\n", policyObj)

		// Convert our typed ClusterPolicy to unstructured for dynamic client
		unstructuredPolicy, err := runtime.DefaultUnstructuredConverter.ToUnstructured(policyObj)
		if err != nil {
			errMsg := fmt.Sprintf("[ERROR] policy[%d]: failed to convert to unstructured: %v", i, err)
			fmt.Println(errMsg)
			errors = append(errors, errMsg)
			continue
		}

		u := &unstructured.Unstructured{Object: unstructuredPolicy}

		// Check what was converted
		fmt.Printf("  Converted to unstructured: apiVersion=%s, kind=%s, name=%s\n",
			u.GetAPIVersion(), u.GetKind(), u.GetName())

		// CRITICAL: Ensure APIVersion and Kind are set (required by dynamic client)
		u.SetAPIVersion("kyverno.io/v1")
		u.SetKind("ClusterPolicy")

		policyName := u.GetName()
		if policyName == "" {
			errMsg := fmt.Sprintf("[ERROR] policy[%d]: missing name after conversion", i)
			fmt.Println(errMsg)
			errors = append(errors, errMsg)
			continue
		}

		// Debug: Log what we're trying to create
		fmt.Printf("  Creating: name='%s', apiVersion='%s', kind='%s'\n",
			policyName, u.GetAPIVersion(), u.GetKind())

		// Try to create the policy, or update if it already exists
		_, createErr := e.dynamicClient.Resource(gvr).Create(ctx, u, metav1.CreateOptions{})
		if createErr != nil {
			// If policy exists, update it
			if strings.Contains(createErr.Error(), "already exists") {
				fmt.Printf("  Policy exists, fetching current version for update...\n")

				// Get the existing policy to retrieve its resourceVersion
				existing, getErr := e.dynamicClient.Resource(gvr).Get(ctx, policyName, metav1.GetOptions{})
				if getErr != nil {
					errMsg := fmt.Sprintf("[ERROR] '%s': failed to get existing policy: %v", policyName, getErr)
					fmt.Println(errMsg)
					errors = append(errors, errMsg)
					continue
				}

				// Set the resourceVersion from the existing policy (required for updates)
				u.SetResourceVersion(existing.GetResourceVersion())
				fmt.Printf("  Using resourceVersion: %s\n", existing.GetResourceVersion())

				_, updateErr := e.dynamicClient.Resource(gvr).Update(ctx, u, metav1.UpdateOptions{})
				if updateErr != nil {
					errMsg := fmt.Sprintf("[ERROR] '%s': update failed: %v", policyName, updateErr)
					fmt.Println(errMsg)
					errors = append(errors, errMsg)
					continue
				}
				fmt.Printf("  [SUCCESS] Updated '%s'\n", policyName)
			} else {
				errMsg := fmt.Sprintf("[ERROR] '%s': creation failed: %v", policyName, createErr)
				fmt.Println(errMsg)
				errors = append(errors, errMsg)
				continue
			}
		} else {
			fmt.Printf("  [SUCCESS] Created '%s'\n", policyName)
		}

		applied++
	}

	fmt.Printf("\n=== POLICY DEPLOYMENT SUMMARY ===\n")
	fmt.Printf("Successfully applied: %d/%d\n", applied, len(policies))
	fmt.Printf("Failed: %d\n", len(errors))

	if len(errors) > 0 {
		fmt.Printf("\n=== ERRORS ===\n")
		for _, errMsg := range errors {
			fmt.Println(errMsg)
		}
	}
	fmt.Printf("=================================\n\n")

	return applied, errors
}

// validatePolicies validates all generated policies before deployment.
func (e *Enforcer) validatePolicies(policies []runtime.Object) error {
	var clusterPolicies []*kyverno.ClusterPolicy

	// Convert runtime.Object to ClusterPolicy for validation
	for _, policyObj := range policies {
		policy, ok := policyObj.(*kyverno.ClusterPolicy)
		if !ok {
			return fmt.Errorf("policy is not a ClusterPolicy (got %T)", policyObj)
		}
		clusterPolicies = append(clusterPolicies, policy)
	}

	// Validate all policies
	validationErrors := e.kyvernoValidator.ValidateBatch(clusterPolicies)
	if len(validationErrors) > 0 {
		return kyverno.FormatValidationErrors(validationErrors)
	}

	return nil
}
