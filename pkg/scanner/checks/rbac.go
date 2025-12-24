package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RBACCheck validates RBAC requirements.
type RBACCheck struct{}

// Name returns the check name.
func (c *RBACCheck) Name() string {
	return "rbac.validation"
}

// Run executes the RBAC check.
func (c *RBACCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Skip if not specified
	if clusterSpec.Spec.RBAC == nil {
		return &scanner.CheckResult{
			Name:    c.Name(),
			Status:  scanner.StatusSkip,
			Message: "RBAC requirements not specified in cluster spec",
		}, nil
	}

	violations := []string{}
	evidence := make(map[string]interface{})

	// Get all ClusterRoles and Roles
	clusterRoles, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster roles: %w", err)
	}

	roles, err := client.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	// Check forbidden rules
	forbiddenViolations := c.checkForbiddenRules(clusterRoles.Items, roles.Items, clusterSpec.Spec.RBAC.ForbiddenRules)
	violations = append(violations, forbiddenViolations...)

	// Check minimum rules
	if len(clusterSpec.Spec.RBAC.MinimumRules) > 0 {
		minimumViolations := c.checkMinimumRules(clusterRoles.Items, roles.Items, clusterSpec.Spec.RBAC.MinimumRules)
		violations = append(violations, minimumViolations...)
	}

	if len(violations) > 0 {
		evidence["violations"] = violations
		evidence["violation_count"] = len(violations)
		evidence["cluster_roles_checked"] = len(clusterRoles.Items)
		evidence["roles_checked"] = len(roles.Items)

		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: scanner.SeverityHigh,
			Message:  fmt.Sprintf("Found %d RBAC violations", len(violations)),
			Evidence: evidence,
			Remediation: `Review and fix RBAC violations:
1. Remove overly permissive roles (e.g., cluster-admin-like wildcard permissions)
2. Ensure required minimum RBAC rules exist
3. Follow principle of least privilege
4. Audit role bindings regularly

Example: Remove wildcard permissions:
kubectl delete clusterrole <role-name>

Example: Create required RBAC role:
kubectl create role <name> --verb=get,list --resource=serviceaccounts`,
		}, nil
	}

	return &scanner.CheckResult{
		Name:    c.Name(),
		Status:  scanner.StatusPass,
		Message: fmt.Sprintf("RBAC configuration complies with requirements (%d cluster roles, %d roles checked)", len(clusterRoles.Items), len(roles.Items)),
		Evidence: map[string]interface{}{
			"cluster_roles_checked": len(clusterRoles.Items),
			"roles_checked":         len(roles.Items),
		},
	}, nil
}

// checkForbiddenRules checks for forbidden RBAC rules.
func (c *RBACCheck) checkForbiddenRules(clusterRoles []rbacv1.ClusterRole, roles []rbacv1.Role, forbiddenRules []spec.RBACRule) []string {
	violations := []string{}

	// Check ClusterRoles
	for _, role := range clusterRoles {
		// Skip system roles
		if strings.HasPrefix(role.Name, "system:") {
			continue
		}

		for _, rule := range role.Rules {
			for _, forbidden := range forbiddenRules {
				if c.ruleMatches(rule, forbidden) {
					violations = append(violations, fmt.Sprintf("ClusterRole '%s' has forbidden rule: apiGroup=%v, resource=%v, verbs=%v",
						role.Name, rule.APIGroups, rule.Resources, rule.Verbs))
				}
			}
		}
	}

	// Check Roles
	for _, role := range roles {
		for _, rule := range role.Rules {
			for _, forbidden := range forbiddenRules {
				if c.ruleMatches(rule, forbidden) {
					violations = append(violations, fmt.Sprintf("Role '%s/%s' has forbidden rule: apiGroup=%v, resource=%v, verbs=%v",
						role.Namespace, role.Name, rule.APIGroups, rule.Resources, rule.Verbs))
				}
			}
		}
	}

	return violations
}

// checkMinimumRules checks for required minimum RBAC rules.
func (c *RBACCheck) checkMinimumRules(clusterRoles []rbacv1.ClusterRole, roles []rbacv1.Role, minimumRules []spec.RBACRule) []string {
	violations := []string{}

	for _, required := range minimumRules {
		found := false

		// Check ClusterRoles
		for _, role := range clusterRoles {
			for _, rule := range role.Rules {
				if c.ruleCovers(rule, required) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		// Check Roles if not found in ClusterRoles
		if !found {
			for _, role := range roles {
				for _, rule := range role.Rules {
					if c.ruleCovers(rule, required) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		if !found {
			violations = append(violations, fmt.Sprintf("Required RBAC rule not found: apiGroup=%s, resource=%s, verbs=%v",
				required.APIGroup, required.Resource, required.Verbs))
		}
	}

	return violations
}

// ruleMatches checks if an RBAC rule matches a forbidden pattern.
func (c *RBACCheck) ruleMatches(rule rbacv1.PolicyRule, forbidden spec.RBACRule) bool {
	// Check if rule has wildcard permissions
	apiGroupMatch := false
	resourceMatch := false
	verbMatch := false

	// Check API groups
	for _, apiGroup := range rule.APIGroups {
		if forbidden.APIGroup == "*" && apiGroup == "*" {
			apiGroupMatch = true
			break
		}
		if apiGroup == forbidden.APIGroup {
			apiGroupMatch = true
			break
		}
	}

	// Check resources
	for _, resource := range rule.Resources {
		if forbidden.Resource == "*" && resource == "*" {
			resourceMatch = true
			break
		}
		if resource == forbidden.Resource {
			resourceMatch = true
			break
		}
	}

	// Check verbs
	for _, verb := range rule.Verbs {
		for _, forbiddenVerb := range forbidden.Verbs {
			if forbiddenVerb == "*" && verb == "*" {
				verbMatch = true
				break
			}
			if verb == forbiddenVerb {
				verbMatch = true
				break
			}
		}
		if verbMatch {
			break
		}
	}

	return apiGroupMatch && resourceMatch && verbMatch
}

// ruleCovers checks if an RBAC rule covers the required permissions.
func (c *RBACCheck) ruleCovers(rule rbacv1.PolicyRule, required spec.RBACRule) bool {
	// Check API groups
	apiGroupCovered := false
	for _, apiGroup := range rule.APIGroups {
		if apiGroup == "*" || apiGroup == required.APIGroup {
			apiGroupCovered = true
			break
		}
	}
	if !apiGroupCovered {
		return false
	}

	// Check resources
	resourceCovered := false
	for _, resource := range rule.Resources {
		if resource == "*" || resource == required.Resource {
			resourceCovered = true
			break
		}
	}
	if !resourceCovered {
		return false
	}

	// Check verbs - all required verbs must be present
	for _, requiredVerb := range required.Verbs {
		verbCovered := false
		for _, verb := range rule.Verbs {
			if verb == "*" || verb == requiredVerb {
				verbCovered = true
				break
			}
		}
		if !verbCovered {
			return false
		}
	}

	return true
}
