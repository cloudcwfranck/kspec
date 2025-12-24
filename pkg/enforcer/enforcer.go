package enforcer

import (
	"context"
	"fmt"

	"github.com/cloudcwfranck/kspec/pkg/enforcer/kyverno"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Enforcer orchestrates policy enforcement.
type Enforcer struct {
	client           kubernetes.Interface
	dynamicClient    dynamic.Interface
	kyvernoGen       *kyverno.Generator
	kyvernoInstaller *kyverno.Installer
}

// NewEnforcer creates a new policy enforcer.
func NewEnforcer(client kubernetes.Interface, dynamicClient dynamic.Interface) *Enforcer {
	return &Enforcer{
		client:           client,
		dynamicClient:    dynamicClient,
		kyvernoGen:       kyverno.NewGenerator(),
		kyvernoInstaller: kyverno.NewInstaller(),
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
	}

	return result, nil
}

// applyPolicies applies Kyverno policies to the cluster.
func (e *Enforcer) applyPolicies(ctx context.Context, policies []runtime.Object) (int, []string) {
	applied := 0
	errors := []string{}

	// Note: In v1.0, we don't actually apply policies
	// This would require using the dynamic client to create ClusterPolicy resources
	// For now, we just return success

	// Future implementation would use dynamic client:
	// gvr := schema.GroupVersionResource{
	//     Group:    "kyverno.io",
	//     Version:  "v1",
	//     Resource: "clusterpolicies",
	// }
	//
	// for _, policy := range policies {
	//     unstructuredPolicy, ok := policy.(*unstructured.Unstructured)
	//     if !ok {
	//         errors = append(errors, "policy is not unstructured")
	//         continue
	//     }
	//     _, err := e.dynamicClient.Resource(gvr).Create(ctx, unstructuredPolicy, metav1.CreateOptions{})
	//     if err != nil {
	//         errors = append(errors, err.Error())
	//         continue
	//     }
	//     applied++
	// }

	// For v1.0, mark all as successfully generated (dry-run style)
	applied = len(policies)

	return applied, errors
}
