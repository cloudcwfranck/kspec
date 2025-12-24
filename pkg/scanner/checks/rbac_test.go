package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRBACCheck_Pass(t *testing.T) {
	// Compliant ClusterRole
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "compliant-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	client := fake.NewSimpleClientset(role)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				MinimumRules: []spec.RBACRule{
					{
						APIGroup: "",
						Resource: "serviceaccounts",
						Verbs:    []string{"get", "list"},
					},
				},
				ForbiddenRules: []spec.RBACRule{
					{
						APIGroup: "*",
						Resource: "*",
						Verbs:    []string{"*"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
	assert.Contains(t, result.Message, "complies with requirements")
}

func TestRBACCheck_FailForbiddenWildcard(t *testing.T) {
	// ClusterRole with wildcard permissions
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	client := fake.NewSimpleClientset(role)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				ForbiddenRules: []spec.RBACRule{
					{
						APIGroup: "*",
						Resource: "*",
						Verbs:    []string{"*"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Equal(t, scanner.SeverityHigh, result.Severity)
	assert.Contains(t, result.Evidence, "violations")
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "forbidden rule")
}

func TestRBACCheck_FailMissingMinimumRule(t *testing.T) {
	// ClusterRole without required permissions
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "limited-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
		},
	}

	client := fake.NewSimpleClientset(role)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				MinimumRules: []spec.RBACRule{
					{
						APIGroup: "",
						Resource: "serviceaccounts",
						Verbs:    []string{"get", "list"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "Required RBAC rule not found")
}

func TestRBACCheck_SystemRolesIgnored(t *testing.T) {
	// System ClusterRole with wildcard permissions (should be ignored)
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:admin",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	client := fake.NewSimpleClientset(role)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				ForbiddenRules: []spec.RBACRule{
					{
						APIGroup: "*",
						Resource: "*",
						Verbs:    []string{"*"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestRBACCheck_NamespaceRoles(t *testing.T) {
	// Namespaced Role with forbidden permissions
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dangerous-role",
			Namespace: "default",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	client := fake.NewSimpleClientset(role)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				ForbiddenRules: []spec.RBACRule{
					{
						APIGroup: "*",
						Resource: "*",
						Verbs:    []string{"*"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "default/dangerous-role")
}

func TestRBACCheck_WildcardCoversRequired(t *testing.T) {
	// ClusterRole with wildcard that covers required permissions
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "wildcard-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"*"},
			},
		},
	}

	client := fake.NewSimpleClientset(role)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				MinimumRules: []spec.RBACRule{
					{
						APIGroup: "",
						Resource: "serviceaccounts",
						Verbs:    []string{"get", "list"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestRBACCheck_Skip(t *testing.T) {
	client := fake.NewSimpleClientset()
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			// No RBAC spec
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusSkip, result.Status)
}

func TestRBACCheck_Name(t *testing.T) {
	check := &RBACCheck{}
	assert.Equal(t, "rbac.validation", check.Name())
}

func TestRBACCheck_MultipleViolations(t *testing.T) {
	// Multiple roles with different violations
	role1 := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	role2 := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "super-admin",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	client := fake.NewSimpleClientset(role1, role2)
	check := &RBACCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			RBAC: &spec.RBACSpec{
				ForbiddenRules: []spec.RBACRule{
					{
						APIGroup: "*",
						Resource: "*",
						Verbs:    []string{"*"},
					},
				},
				MinimumRules: []spec.RBACRule{
					{
						APIGroup: "",
						Resource: "configmaps",
						Verbs:    []string{"get"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	// Should have violations from both roles (wildcard permissions cover minimum rule)
	assert.Equal(t, 2, len(violations))
}
