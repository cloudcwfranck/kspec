package policy

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PolicyTemplate represents a reusable policy template with parameters
type PolicyTemplate struct {
	Name        string
	Description string
	Category    string // security, compliance, cost-optimization, etc.
	Parameters  []TemplateParameter
	BasePolicy  PolicyDefinition
}

// TemplateParameter defines a configurable parameter in a template
type TemplateParameter struct {
	Name          string
	Description   string
	Type          string // string, int, bool, array
	Default       interface{}
	Required      bool
	AllowedValues []interface{}
}

// PolicyDefinition contains the actual policy rules
type PolicyDefinition struct {
	RequiredFields  []FieldRequirement
	ForbiddenFields []FieldRequirement
	Validations     []ValidationRule
}

// FieldRequirement defines a required or forbidden field
type FieldRequirement struct {
	Key   string
	Value string
}

// ValidationRule defines custom validation logic
type ValidationRule struct {
	Name       string
	Expression string // CEL expression or similar
	Message    string
}

// PolicyInheritance manages policy composition and inheritance
type PolicyInheritance struct {
	BasePolicies []string // Names of parent policies
	Overrides    map[string]interface{}
	Additions    PolicyDefinition
}

// NamespaceScope defines namespace-level policy scoping
type NamespaceScope struct {
	IncludeNamespaces []string
	ExcludeNamespaces []string
	NamespaceSelector *metav1.LabelSelector
}

// TimeBasedActivation defines time-based policy activation
type TimeBasedActivation struct {
	Enabled       bool
	ActivePeriods []TimePeriod
	Timezone      string // e.g., "America/New_York"
}

// TimePeriod defines a time range for policy activation
type TimePeriod struct {
	StartTime  string   // e.g., "09:00"
	EndTime    string   // e.g., "17:00"
	DaysOfWeek []string // e.g., ["Monday", "Tuesday", "Wednesday"]
	StartDate  *metav1.Time
	EndDate    *metav1.Time
}

// PolicyExemption defines resources exempt from policy enforcement
type PolicyExemption struct {
	Name       string
	Reason     string
	ExpiresAt  *metav1.Time
	Namespaces []string
	Resources  []ResourceSelector
	Approver   string
	CreatedAt  metav1.Time
}

// ResourceSelector identifies resources for exemption
type ResourceSelector struct {
	Kind      string
	Name      string
	Namespace string
	Labels    map[string]string
}

// AdvancedPolicyManager manages advanced policy features
type AdvancedPolicyManager struct {
	Client    client.Client
	Templates map[string]*PolicyTemplate
}

// NewAdvancedPolicyManager creates a new policy manager
func NewAdvancedPolicyManager(client client.Client) *AdvancedPolicyManager {
	return &AdvancedPolicyManager{
		Client:    client,
		Templates: initializeDefaultTemplates(),
	}
}

// ApplyTemplate applies a policy template with given parameters
func (m *AdvancedPolicyManager) ApplyTemplate(
	ctx context.Context,
	templateName string,
	parameters map[string]interface{},
) (*PolicyDefinition, error) {
	log := log.FromContext(ctx).WithValues("template", templateName)

	template, exists := m.Templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	log.Info("Applying policy template")

	// Validate parameters
	if err := m.validateParameters(template, parameters); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Merge parameters with defaults
	mergedParams := m.mergeParameters(template, parameters)

	// Apply parameters to base policy
	policy := m.applyParametersToPolicy(&template.BasePolicy, mergedParams)

	log.Info("Policy template applied successfully")
	return policy, nil
}

// InheritPolicies combines multiple policies through inheritance
func (m *AdvancedPolicyManager) InheritPolicies(
	ctx context.Context,
	basePolicyNames []string,
	overrides map[string]interface{},
	additions *PolicyDefinition,
) (*PolicyDefinition, error) {
	log := log.FromContext(ctx)

	log.Info("Inheriting policies", "basePolicies", basePolicyNames)

	result := &PolicyDefinition{
		RequiredFields:  make([]FieldRequirement, 0),
		ForbiddenFields: make([]FieldRequirement, 0),
		Validations:     make([]ValidationRule, 0),
	}

	// Merge base policies
	for _, policyName := range basePolicyNames {
		basePolicy, err := m.getBasePolicy(ctx, policyName)
		if err != nil {
			return nil, fmt.Errorf("failed to get base policy %s: %w", policyName, err)
		}

		result = m.mergePolicies(result, basePolicy)
	}

	// Apply overrides
	if len(overrides) > 0 {
		result = m.applyOverrides(result, overrides)
	}

	// Add additional rules
	if additions != nil {
		result = m.mergePolicies(result, additions)
	}

	log.Info("Policy inheritance completed")
	return result, nil
}

// IsActiveInTimeWindow checks if a policy is active based on time-based activation
func (m *AdvancedPolicyManager) IsActiveInTimeWindow(
	activation *TimeBasedActivation,
	currentTime time.Time,
) bool {
	if activation == nil || !activation.Enabled {
		return true // Always active if time-based activation is disabled
	}

	// Load timezone
	location, err := time.LoadLocation(activation.Timezone)
	if err != nil {
		// Default to UTC on error
		location = time.UTC
	}

	currentTime = currentTime.In(location)
	currentDay := currentTime.Weekday().String()
	currentTimeStr := currentTime.Format("15:04")

	// Check if current time falls within any active period
	for _, period := range activation.ActivePeriods {
		// Check day of week
		if len(period.DaysOfWeek) > 0 && !contains(period.DaysOfWeek, currentDay) {
			continue
		}

		// Check date range
		if period.StartDate != nil && currentTime.Before(period.StartDate.Time) {
			continue
		}
		if period.EndDate != nil && currentTime.After(period.EndDate.Time) {
			continue
		}

		// Check time range
		if currentTimeStr >= period.StartTime && currentTimeStr <= period.EndTime {
			return true
		}
	}

	return false
}

// IsExempt checks if a resource is exempt from policy enforcement
func (m *AdvancedPolicyManager) IsExempt(
	ctx context.Context,
	exemptions []PolicyExemption,
	resourceKind, resourceName, resourceNamespace string,
	resourceLabels map[string]string,
) (bool, string) {
	currentTime := time.Now()

	for _, exemption := range exemptions {
		// Check if exemption has expired
		if exemption.ExpiresAt != nil && currentTime.After(exemption.ExpiresAt.Time) {
			continue
		}

		// Check namespace
		if len(exemption.Namespaces) > 0 && !contains(exemption.Namespaces, resourceNamespace) {
			continue
		}

		// Check resource selectors
		for _, selector := range exemption.Resources {
			if m.matchesSelector(selector, resourceKind, resourceName, resourceNamespace, resourceLabels) {
				return true, exemption.Reason
			}
		}
	}

	return false, ""
}

// ApplyNamespaceScope filters ClusterSpecs based on namespace scoping
func (m *AdvancedPolicyManager) ApplyNamespaceScope(
	scope *NamespaceScope,
	targetNamespace string,
) bool {
	if scope == nil {
		return true // No scoping, applies to all namespaces
	}

	// Check exclusions first
	if len(scope.ExcludeNamespaces) > 0 && contains(scope.ExcludeNamespaces, targetNamespace) {
		return false
	}

	// Check inclusions
	if len(scope.IncludeNamespaces) > 0 {
		return contains(scope.IncludeNamespaces, targetNamespace)
	}

	// TODO: Implement label selector matching
	// if scope.NamespaceSelector != nil {
	//     return matchesLabelSelector(targetNamespace, scope.NamespaceSelector)
	// }

	return true
}

// Helper functions

func (m *AdvancedPolicyManager) validateParameters(
	template *PolicyTemplate,
	parameters map[string]interface{},
) error {
	for _, param := range template.Parameters {
		value, provided := parameters[param.Name]

		// Check required parameters
		if param.Required && !provided {
			return fmt.Errorf("required parameter %s not provided", param.Name)
		}

		// Validate allowed values
		if provided && len(param.AllowedValues) > 0 {
			if !containsValue(param.AllowedValues, value) {
				return fmt.Errorf("parameter %s has invalid value: %v", param.Name, value)
			}
		}
	}

	return nil
}

func (m *AdvancedPolicyManager) mergeParameters(
	template *PolicyTemplate,
	parameters map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})

	// Add defaults
	for _, param := range template.Parameters {
		if param.Default != nil {
			result[param.Name] = param.Default
		}
	}

	// Override with provided parameters
	for k, v := range parameters {
		result[k] = v
	}

	return result
}

func (m *AdvancedPolicyManager) applyParametersToPolicy(
	basePolicy *PolicyDefinition,
	parameters map[string]interface{},
) *PolicyDefinition {
	// Create a copy of the base policy
	policy := &PolicyDefinition{
		RequiredFields:  make([]FieldRequirement, len(basePolicy.RequiredFields)),
		ForbiddenFields: make([]FieldRequirement, len(basePolicy.ForbiddenFields)),
		Validations:     make([]ValidationRule, len(basePolicy.Validations)),
	}

	copy(policy.RequiredFields, basePolicy.RequiredFields)
	copy(policy.ForbiddenFields, basePolicy.ForbiddenFields)
	copy(policy.Validations, basePolicy.Validations)

	// TODO: Apply parameter substitution to policy fields
	// This would replace template variables like {{paramName}} with actual values

	return policy
}

func (m *AdvancedPolicyManager) getBasePolicy(
	ctx context.Context,
	policyName string,
) (*PolicyDefinition, error) {
	// In a real implementation, this would fetch the policy from a storage system
	// For now, return an empty policy
	return &PolicyDefinition{
		RequiredFields:  make([]FieldRequirement, 0),
		ForbiddenFields: make([]FieldRequirement, 0),
		Validations:     make([]ValidationRule, 0),
	}, nil
}

func (m *AdvancedPolicyManager) mergePolicies(
	policy1, policy2 *PolicyDefinition,
) *PolicyDefinition {
	result := &PolicyDefinition{
		RequiredFields:  append(policy1.RequiredFields, policy2.RequiredFields...),
		ForbiddenFields: append(policy1.ForbiddenFields, policy2.ForbiddenFields...),
		Validations:     append(policy1.Validations, policy2.Validations...),
	}

	return result
}

func (m *AdvancedPolicyManager) applyOverrides(
	policy *PolicyDefinition,
	overrides map[string]interface{},
) *PolicyDefinition {
	// TODO: Implement override logic
	// This would allow specific fields to be overridden
	return policy
}

func (m *AdvancedPolicyManager) matchesSelector(
	selector ResourceSelector,
	kind, name, namespace string,
	labels map[string]string,
) bool {
	if selector.Kind != "" && selector.Kind != kind {
		return false
	}
	if selector.Name != "" && selector.Name != name {
		return false
	}
	if selector.Namespace != "" && selector.Namespace != namespace {
		return false
	}

	// Check label matching
	if len(selector.Labels) > 0 {
		for k, v := range selector.Labels {
			if labels[k] != v {
				return false
			}
		}
	}

	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsValue(slice []interface{}, value interface{}) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// initializeDefaultTemplates creates built-in policy templates
func initializeDefaultTemplates() map[string]*PolicyTemplate {
	templates := make(map[string]*PolicyTemplate)

	// Security baseline template
	templates["security-baseline"] = &PolicyTemplate{
		Name:        "security-baseline",
		Description: "Basic security requirements for all workloads",
		Category:    "security",
		Parameters: []TemplateParameter{
			{
				Name:        "allowPrivilegeEscalation",
				Description: "Allow privilege escalation",
				Type:        "bool",
				Default:     false,
				Required:    false,
			},
			{
				Name:        "runAsNonRoot",
				Description: "Require running as non-root user",
				Type:        "bool",
				Default:     true,
				Required:    false,
			},
		},
		BasePolicy: PolicyDefinition{
			RequiredFields: []FieldRequirement{
				{Key: "securityContext.runAsNonRoot", Value: "true"},
			},
			ForbiddenFields: []FieldRequirement{
				{Key: "securityContext.privileged", Value: "true"},
				{Key: "hostNetwork", Value: "true"},
			},
		},
	}

	// Compliance template
	templates["compliance-strict"] = &PolicyTemplate{
		Name:        "compliance-strict",
		Description: "Strict compliance requirements",
		Category:    "compliance",
		Parameters: []TemplateParameter{
			{
				Name:        "requireDigests",
				Description: "Require image digests",
				Type:        "bool",
				Default:     true,
				Required:    false,
			},
		},
		BasePolicy: PolicyDefinition{
			RequiredFields: []FieldRequirement{
				{Key: "resources.limits.memory", Value: "true"},
				{Key: "resources.limits.cpu", Value: "true"},
			},
		},
	}

	return templates
}
