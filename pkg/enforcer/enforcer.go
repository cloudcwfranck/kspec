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

	if installed {
		version, err := e.kyvernoInstaller.GetVersion(ctx, e.client)
		if err == nil {
			result.KyvernoVersion = version
		}
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
		return result, nil
	}

	// Check if Kyverno is installed before applying
	if !installed && !opts.SkipInstall {
		return result, fmt.Errorf("Kyverno is not installed. Install it first or use --skip-install flag.\n\n%s",
			e.kyvernoInstaller.GetInstallInstructions())
	}

	// Apply policies (if not dry-run and Kyverno is installed)
	if installed {
		applied, applyErrors := e.applyPolicies(ctx, policies)
		result.PoliciesApplied = applied
		result.Errors = applyErrors

		// CRITICAL: If policies failed to apply, return error
		if len(applyErrors) > 0 {
			return nil, fmt.Errorf("failed to apply %d policies: %v", len(applyErrors), applyErrors)
		}
	}

	return result, nil
}

// applyPolicies applies Kyverno policies to the cluster.
func (e *Enforcer) applyPolicies(ctx context.Context, policies []runtime.Object) (int, []string) {
	applied := 0
	errors := []string{}

	// Define Kyverno ClusterPolicy GVR
	gvr := schema.GroupVersionResource{
		Group:    "kyverno.io",
		Version:  "v1",
		Resource: "clusterpolicies",
	}

	for i, policyObj := range policies {
		// Convert our typed ClusterPolicy to unstructured for dynamic client
		unstructuredPolicy, err := runtime.DefaultUnstructuredConverter.ToUnstructured(policyObj)
		if err != nil {
			errors = append(errors, fmt.Sprintf("policy[%d]: failed to convert to unstructured: %v", i, err))
			continue
		}

		u := &unstructured.Unstructured{Object: unstructuredPolicy}

		// CRITICAL: Ensure APIVersion and Kind are set (required by dynamic client)
		u.SetAPIVersion("kyverno.io/v1")
		u.SetKind("ClusterPolicy")

		policyName := u.GetName()
		if policyName == "" {
			errors = append(errors, fmt.Sprintf("policy[%d]: missing name after conversion", i))
			continue
		}

		// Debug: Log what we're trying to create
		fmt.Printf("DEBUG: Attempting to create ClusterPolicy '%s' (APIVersion=%s, Kind=%s)\n",
			policyName, u.GetAPIVersion(), u.GetKind())

		// Try to create the policy, or update if it already exists
		_, createErr := e.dynamicClient.Resource(gvr).Create(ctx, u, metav1.CreateOptions{})
		if createErr != nil {
			// If policy exists, update it
			if strings.Contains(createErr.Error(), "already exists") {
				fmt.Printf("DEBUG: Policy '%s' already exists, updating...\n", policyName)
				_, updateErr := e.dynamicClient.Resource(gvr).Update(ctx, u, metav1.UpdateOptions{})
				if updateErr != nil {
					errors = append(errors, fmt.Sprintf("policy '%s': failed to update: %v", policyName, updateErr))
					continue
				}
				fmt.Printf("DEBUG: Successfully updated policy '%s'\n", policyName)
			} else {
				errors = append(errors, fmt.Sprintf("policy '%s': failed to create: %v", policyName, createErr))
				fmt.Printf("DEBUG: Failed to create policy '%s': %v\n", policyName, createErr)
				continue
			}
		} else {
			fmt.Printf("DEBUG: Successfully created policy '%s'\n", policyName)
		}

		applied++
	}

	fmt.Printf("DEBUG: Applied %d/%d policies, %d errors\n", applied, len(policies), len(errors))

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
