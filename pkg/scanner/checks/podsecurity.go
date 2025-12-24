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

const (
	// Pod Security Standards label keys
	psEnforceLabel = "pod-security.kubernetes.io/enforce"
	psAuditLabel   = "pod-security.kubernetes.io/audit"
	psWarnLabel    = "pod-security.kubernetes.io/warn"
)

// PodSecurityStandardsCheck validates Pod Security Standards configuration.
type PodSecurityStandardsCheck struct{}

// Name returns the check identifier.
func (c *PodSecurityStandardsCheck) Name() string {
	return "podsecurity.standards"
}

// Run executes the Pod Security Standards check.
func (c *PodSecurityStandardsCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Skip check if Pod Security Standards are not specified
	if clusterSpec.Spec.PodSecurity == nil {
		return &scanner.CheckResult{
			Name:   c.Name(),
			Status: scanner.StatusSkip,
			Message: "Pod Security Standards not specified in cluster spec",
		}, nil
	}

	pss := clusterSpec.Spec.PodSecurity

	// Get all namespaces
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	// Build exemption map for quick lookup
	exemptions := make(map[string]spec.PodSecurityExemption)
	for _, exemption := range pss.Exemptions {
		exemptions[exemption.Namespace] = exemption
	}

	var (
		violations    []string
		checkedCount  int
		exemptedCount int
	)

	// Check each namespace
	for _, ns := range namespaces.Items {
		// Skip system namespaces by default
		if isSystemNamespace(ns.Name) && !hasExemption(ns.Name, exemptions) {
			continue
		}

		// Check if namespace is exempted
		if exemption, exists := exemptions[ns.Name]; exists {
			exemptedCount++
			// Verify exemption level is set correctly
			if ns.Labels[psEnforceLabel] != exemption.Level {
				violations = append(violations, fmt.Sprintf(
					"namespace %s: exemption level %s not configured (current: %s)",
					ns.Name, exemption.Level, ns.Labels[psEnforceLabel],
				))
			}
			continue
		}

		checkedCount++

		// Check enforce level
		if enforce := ns.Labels[psEnforceLabel]; enforce != pss.Enforce {
			violations = append(violations, fmt.Sprintf(
				"namespace %s: enforce level should be %s (current: %s)",
				ns.Name, pss.Enforce, enforce,
			))
		}

		// Check audit level
		if audit := ns.Labels[psAuditLabel]; audit != pss.Audit {
			violations = append(violations, fmt.Sprintf(
				"namespace %s: audit level should be %s (current: %s)",
				ns.Name, pss.Audit, audit,
			))
		}

		// Check warn level
		if warn := ns.Labels[psWarnLabel]; warn != pss.Warn {
			violations = append(violations, fmt.Sprintf(
				"namespace %s: warn level should be %s (current: %s)",
				ns.Name, pss.Warn, warn,
			))
		}
	}

	// Build evidence
	evidence := map[string]interface{}{
		"total_namespaces": len(namespaces.Items),
		"checked":          checkedCount,
		"exempted":         exemptedCount,
		"violations":       len(violations),
		"required_enforce": pss.Enforce,
		"required_audit":   pss.Audit,
		"required_warn":    pss.Warn,
	}

	// Return result
	if len(violations) > 0 {
		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: scanner.SeverityHigh,
			Message: fmt.Sprintf(
				"Found %d Pod Security Standards violations across %d namespaces",
				len(violations), checkedCount,
			),
			Evidence:    evidence,
			Remediation: c.buildRemediation(pss, violations),
		}, nil
	}

	return &scanner.CheckResult{
		Name:   c.Name(),
		Status: scanner.StatusPass,
		Message: fmt.Sprintf(
			"All %d namespaces have correct Pod Security Standards configured (enforce=%s, audit=%s, warn=%s)",
			checkedCount, pss.Enforce, pss.Audit, pss.Warn,
		),
		Evidence: evidence,
	}, nil
}

// buildRemediation generates remediation guidance.
func (c *PodSecurityStandardsCheck) buildRemediation(pss *spec.PodSecuritySpec, violations []string) string {
	remediation := fmt.Sprintf(
		"Apply Pod Security Standards labels to namespaces:\n\nkubectl label namespace <namespace> \\\n  %s=%s \\\n  %s=%s \\\n  %s=%s\n\nViolations:\n",
		psEnforceLabel, pss.Enforce,
		psAuditLabel, pss.Audit,
		psWarnLabel, pss.Warn,
	)

	// Add first few violations as examples
	maxViolations := 5
	for i, violation := range violations {
		if i >= maxViolations {
			remediation += fmt.Sprintf("  ... and %d more\n", len(violations)-maxViolations)
			break
		}
		remediation += fmt.Sprintf("  - %s\n", violation)
	}

	return remediation
}

// isSystemNamespace checks if a namespace is a system namespace.
func isSystemNamespace(name string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
	}
	for _, sysNs := range systemNamespaces {
		if name == sysNs {
			return true
		}
	}
	return false
}

// hasExemption checks if a namespace has an exemption.
func hasExemption(namespace string, exemptions map[string]spec.PodSecurityExemption) bool {
	_, exists := exemptions[namespace]
	return exists
}
