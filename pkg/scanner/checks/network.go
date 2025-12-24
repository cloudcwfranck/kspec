// Package checks contains all compliance check implementations.
package checks

import (
	"context"
	"fmt"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetworkPolicyCheck validates network policy requirements.
type NetworkPolicyCheck struct{}

// Name returns the check identifier.
func (c *NetworkPolicyCheck) Name() string {
	return "network.policies"
}

// Run executes the network policy check.
func (c *NetworkPolicyCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Skip check if network policies are not specified
	if clusterSpec.Spec.Network == nil {
		return &scanner.CheckResult{
			Name:    c.Name(),
			Status:  scanner.StatusSkip,
			Message: "Network policies not specified in cluster spec",
		}, nil
	}

	network := clusterSpec.Spec.Network
	var violations []string
	evidence := make(map[string]interface{})

	// Check default-deny requirement
	if network.DefaultDeny {
		namespacesWithoutDefaultDeny, err := c.checkDefaultDeny(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("failed to check default-deny policies: %w", err)
		}

		if len(namespacesWithoutDefaultDeny) > 0 {
			violations = append(violations, fmt.Sprintf(
				"%d namespaces missing default-deny NetworkPolicy: %v",
				len(namespacesWithoutDefaultDeny),
				namespacesWithoutDefaultDeny,
			))
			evidence["namespaces_without_default_deny"] = namespacesWithoutDefaultDeny
		}
		evidence["default_deny_required"] = true
		evidence["default_deny_violations"] = len(namespacesWithoutDefaultDeny)
	}

	// Check required policies
	if len(network.RequiredPolicies) > 0 {
		missingPolicies, err := c.checkRequiredPolicies(ctx, client, network.RequiredPolicies)
		if err != nil {
			return nil, fmt.Errorf("failed to check required policies: %w", err)
		}

		if len(missingPolicies) > 0 {
			for _, policyName := range missingPolicies {
				violations = append(violations, fmt.Sprintf(
					"required NetworkPolicy %s not found in cluster",
					policyName,
				))
			}
			evidence["missing_required_policies"] = missingPolicies
		}
		evidence["required_policies_count"] = len(network.RequiredPolicies)
	}

	// Return result
	if len(violations) > 0 {
		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: scanner.SeverityHigh,
			Message: fmt.Sprintf(
				"Found %d network policy violations",
				len(violations),
			),
			Evidence:    evidence,
			Remediation: c.buildRemediation(violations),
		}, nil
	}

	passMessage := "All network policy requirements met"
	if network.DefaultDeny {
		passMessage += " (default-deny policies present)"
	}

	return &scanner.CheckResult{
		Name:     c.Name(),
		Status:   scanner.StatusPass,
		Message:  passMessage,
		Evidence: evidence,
	}, nil
}

// checkDefaultDeny checks for default-deny network policies in all user namespaces.
func (c *NetworkPolicyCheck) checkDefaultDeny(ctx context.Context, client kubernetes.Interface) ([]string, error) {
	// Get all namespaces
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var namespacesWithoutDefaultDeny []string

	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if isSystemNamespace(ns.Name) {
			continue
		}

		// Get network policies in this namespace
		policies, err := client.NetworkingV1().NetworkPolicies(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list network policies in namespace %s: %w", ns.Name, err)
		}

		// Check if there's a default-deny policy
		hasDefaultDeny := false
		for _, policy := range policies.Items {
			// A default-deny policy typically has an empty podSelector
			// and no ingress/egress rules, or explicit deny rules
			if len(policy.Spec.PodSelector.MatchLabels) == 0 &&
				len(policy.Spec.PodSelector.MatchExpressions) == 0 {
				// Check if it denies ingress or egress
				if len(policy.Spec.Ingress) == 0 || len(policy.Spec.Egress) == 0 {
					hasDefaultDeny = true
					break
				}
			}
		}

		if !hasDefaultDeny {
			namespacesWithoutDefaultDeny = append(namespacesWithoutDefaultDeny, ns.Name)
		}
	}

	return namespacesWithoutDefaultDeny, nil
}

// checkRequiredPolicies checks if all required policies exist.
func (c *NetworkPolicyCheck) checkRequiredPolicies(ctx context.Context, client kubernetes.Interface, requiredPolicies []spec.RequiredPolicy) ([]string, error) {
	// Get all network policies across all namespaces
	allPolicies := make(map[string]bool)

	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		policies, err := client.NetworkingV1().NetworkPolicies(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue // Skip namespaces we can't read
		}

		for _, policy := range policies.Items {
			allPolicies[policy.Name] = true
		}
	}

	// Check which required policies are missing
	var missingPolicies []string
	for _, required := range requiredPolicies {
		if !allPolicies[required.Name] {
			missingPolicies = append(missingPolicies, required.Name)
		}
	}

	return missingPolicies, nil
}

// buildRemediation generates remediation guidance.
func (c *NetworkPolicyCheck) buildRemediation(violations []string) string {
	remediation := "Network policy violations found:\n\n"

	for _, violation := range violations {
		remediation += fmt.Sprintf("- %s\n", violation)
	}

	remediation += "\nTo create a default-deny NetworkPolicy:\n\n"
	remediation += `kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: <namespace>
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
EOF
`

	return remediation
}
