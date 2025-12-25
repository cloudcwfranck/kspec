package kyverno

import (
	"fmt"
	"strings"
)

// Validator validates Kyverno ClusterPolicy syntax.
type Validator struct{}

// NewValidator creates a new policy validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate performs comprehensive validation on a ClusterPolicy.
func (v *Validator) Validate(policy *ClusterPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}

	// Validate TypeMeta
	if err := v.validateTypeMeta(policy); err != nil {
		return fmt.Errorf("invalid TypeMeta: %w", err)
	}

	// Validate ObjectMeta
	if err := v.validateObjectMeta(policy); err != nil {
		return fmt.Errorf("invalid ObjectMeta: %w", err)
	}

	// Validate Spec
	if err := v.validateSpec(&policy.Spec); err != nil {
		return fmt.Errorf("invalid Spec: %w", err)
	}

	return nil
}

// validateTypeMeta validates the TypeMeta fields.
func (v *Validator) validateTypeMeta(policy *ClusterPolicy) error {
	if policy.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}

	if policy.APIVersion != "kyverno.io/v1" {
		return fmt.Errorf("apiVersion must be 'kyverno.io/v1', got '%s'", policy.APIVersion)
	}

	if policy.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	if policy.Kind != "ClusterPolicy" {
		return fmt.Errorf("kind must be 'ClusterPolicy', got '%s'", policy.Kind)
	}

	return nil
}

// validateObjectMeta validates the ObjectMeta fields.
func (v *Validator) validateObjectMeta(policy *ClusterPolicy) error {
	if policy.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// Kubernetes resource name validation (RFC 1123 subdomain)
	if len(policy.Name) > 253 {
		return fmt.Errorf("metadata.name is too long (max 253 characters)")
	}

	if !isValidDNSSubdomain(policy.Name) {
		return fmt.Errorf("metadata.name must be a valid DNS subdomain (lowercase, alphanumeric, '-', '.')")
	}

	return nil
}

// validateSpec validates the ClusterPolicySpec fields.
func (v *Validator) validateSpec(spec *ClusterPolicySpec) error {
	// ValidationFailureAction is optional, but if set must be valid
	if spec.ValidationFailureAction != "" {
		if spec.ValidationFailureAction != Enforce && spec.ValidationFailureAction != Audit {
			return fmt.Errorf("validationFailureAction must be 'Enforce' or 'Audit', got '%s'", spec.ValidationFailureAction)
		}
	}

	// Rules are required
	if len(spec.Rules) == 0 {
		return fmt.Errorf("spec.rules is required and must contain at least one rule")
	}

	// Validate each rule
	for i, rule := range spec.Rules {
		if err := v.validateRule(&rule, i); err != nil {
			return fmt.Errorf("rule[%d]: %w", i, err)
		}
	}

	return nil
}

// validateRule validates a single Rule.
func (v *Validator) validateRule(rule *Rule, index int) error {
	// Name is required
	if rule.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Name must be valid
	if !isValidRuleName(rule.Name) {
		return fmt.Errorf("name '%s' must be lowercase alphanumeric with hyphens", rule.Name)
	}

	// Match is required
	if len(rule.Match.Any) == 0 && len(rule.Match.All) == 0 {
		return fmt.Errorf("match.any or match.all is required")
	}

	// Cannot have both Any and All
	if len(rule.Match.Any) > 0 && len(rule.Match.All) > 0 {
		return fmt.Errorf("match cannot have both 'any' and 'all'")
	}

	// Validate match resources
	if err := v.validateMatchResources(&rule.Match); err != nil {
		return fmt.Errorf("invalid match: %w", err)
	}

	// Either validation or mutation is required
	if rule.Validation == nil && rule.Mutation == nil {
		return fmt.Errorf("either validate or mutate is required")
	}

	// Cannot have both validation and mutation
	if rule.Validation != nil && rule.Mutation != nil {
		return fmt.Errorf("cannot have both validate and mutate in the same rule")
	}

	// Validate validation block
	if rule.Validation != nil {
		if err := v.validateValidation(rule.Validation); err != nil {
			return fmt.Errorf("invalid validation: %w", err)
		}
	}

	return nil
}

// validateMatchResources validates MatchResources.
func (v *Validator) validateMatchResources(match *MatchResources) error {
	filters := match.Any
	if len(match.All) > 0 {
		filters = match.All
	}

	for i, filter := range filters {
		if filter.Resources == nil {
			return fmt.Errorf("filter[%d]: resources is required", i)
		}

		if len(filter.Resources.Kinds) == 0 {
			return fmt.Errorf("filter[%d]: resources.kinds is required", i)
		}

		// Validate kinds
		for _, kind := range filter.Resources.Kinds {
			if kind == "" {
				return fmt.Errorf("filter[%d]: empty kind in resources.kinds", i)
			}
		}
	}

	return nil
}

// validateValidation validates a Validation block.
func (v *Validator) validateValidation(validation *Validation) error {
	// Must have either pattern, anyPattern, or deny
	hasPattern := validation.Pattern != nil
	hasAnyPattern := len(validation.AnyPattern) > 0
	hasDeny := validation.Deny != nil

	if !hasPattern && !hasAnyPattern && !hasDeny {
		return fmt.Errorf("must have pattern, anyPattern, or deny")
	}

	// Cannot have multiple pattern types
	count := 0
	if hasPattern {
		count++
	}
	if hasAnyPattern {
		count++
	}
	if hasDeny {
		count++
	}

	if count > 1 {
		return fmt.Errorf("can only have one of: pattern, anyPattern, or deny")
	}

	// Message is optional but recommended
	if validation.Message == "" {
		// Not an error, but could be a warning in the future
	}

	return nil
}

// isValidDNSSubdomain checks if a string is a valid DNS subdomain.
// Per RFC 1123: lowercase alphanumeric characters, '-', or '.'
func isValidDNSSubdomain(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.') {
			return false
		}
		// Cannot start or end with '-' or '.'
		if (i == 0 || i == len(name)-1) && (c == '-' || c == '.') {
			return false
		}
	}

	return true
}

// isValidRuleName checks if a rule name is valid.
// Rule names should be lowercase alphanumeric with hyphens.
func isValidRuleName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	for i, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
		// Cannot start or end with '-'
		if (i == 0 || i == len(name)-1) && c == '-' {
			return false
		}
	}

	return true
}

// ValidateBatch validates multiple policies and returns all errors.
func (v *Validator) ValidateBatch(policies []*ClusterPolicy) []error {
	var errors []error

	for i, policy := range policies {
		if err := v.Validate(policy); err != nil {
			errors = append(errors, fmt.Errorf("policy[%d] (%s): %w", i, policy.Name, err))
		}
	}

	return errors
}

// FormatValidationErrors formats multiple validation errors into a single error message.
func FormatValidationErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	if len(errors) == 1 {
		return errors[0]
	}

	var msgs []string
	for _, err := range errors {
		msgs = append(msgs, err.Error())
	}

	return fmt.Errorf("policy validation failed with %d errors:\n  - %s",
		len(errors), strings.Join(msgs, "\n  - "))
}
