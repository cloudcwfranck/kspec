package policy

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createTestClient() client.Client {
	scheme := runtime.NewScheme()
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

// Test Policy Template Application

func TestApplyTemplate(t *testing.T) {
	ctx := context.Background()
	client := createTestClient()
	manager := NewAdvancedPolicyManager(client)

	// Register a test template
	manager.Templates["test-template"] = &PolicyTemplate{
		Name:        "test-template",
		Description: "Test template with parameters",
		Category:    "security",
		Parameters: []TemplateParameter{
			{
				Name:        "severity",
				Description: "Severity level",
				Type:        "string",
				Default:     "medium",
				Required:    false,
				AllowedValues: []interface{}{
					"low", "medium", "high",
				},
			},
			{
				Name:        "max_replicas",
				Description: "Maximum replicas",
				Type:        "int",
				Default:     10,
				Required:    true,
			},
		},
		BasePolicy: PolicyDefinition{
			RequiredFields: []FieldRequirement{
				{
					Key:   "spec.replicas",
					Value: "<=max_replicas",
				},
			},
		},
	}

	tests := []struct {
		name         string
		templateName string
		params       map[string]interface{}
		expectError  bool
	}{
		{
			name:         "valid parameters",
			templateName: "test-template",
			params: map[string]interface{}{
				"severity":     "high",
				"max_replicas": 5,
			},
			expectError: false,
		},
		{
			name:         "missing required parameter",
			templateName: "test-template",
			params: map[string]interface{}{
				"severity": "high",
				// max_replicas is missing
			},
			expectError: true,
		},
		{
			name:         "invalid parameter value",
			templateName: "test-template",
			params: map[string]interface{}{
				"severity":     "invalid", // not in allowed values
				"max_replicas": 5,
			},
			expectError: true,
		},
		{
			name:         "template not found",
			templateName: "non-existent-template",
			params:       map[string]interface{}{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := manager.ApplyTemplate(ctx, tt.templateName, tt.params)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && policy == nil {
				t.Error("Expected policy but got nil")
			}
		})
	}
}

// Test Policy Inheritance

func TestInheritPolicies(t *testing.T) {
	ctx := context.Background()
	client := createTestClient()
	manager := NewAdvancedPolicyManager(client)

	// This test verifies the function signature is correct
	// In a real test, we'd register base policies first
	basePolicies := []string{"base-security-policy"}
	overrides := map[string]interface{}{
		"severity": "critical",
	}
	additions := &PolicyDefinition{
		RequiredFields: []FieldRequirement{
			{
				Key:   "spec.custom",
				Value: "value",
			},
		},
	}

	// Test inheritance - this will fail because we don't have base policies loaded,
	// but we're testing that the function doesn't panic
	_, err := manager.InheritPolicies(ctx, basePolicies, overrides, additions)

	// We expect an error because base policies don't exist in our test
	if err == nil {
		t.Log("Inheritance processed (expected to fail without base policies)")
	} else {
		t.Logf("Expected error occurred: %v", err)
	}
}

// Test Time-Based Activation

func TestIsActiveInTimeWindow(t *testing.T) {
	client := createTestClient()
	manager := NewAdvancedPolicyManager(client)

	// Use fixed time for testing
	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC) // Monday, 2:30 PM UTC

	tests := []struct {
		name       string
		activation *TimeBasedActivation
		testTime   time.Time
		expected   bool
	}{
		{
			name: "disabled activation (always active)",
			activation: &TimeBasedActivation{
				Enabled: false,
			},
			testTime: testTime,
			expected: true,
		},
		{
			name: "active time window - within range",
			activation: &TimeBasedActivation{
				Enabled: true,
				ActivePeriods: []TimePeriod{
					{
						StartTime:  "09:00",
						EndTime:    "17:00",
						DaysOfWeek: []string{"Monday"},
					},
				},
				Timezone: "UTC",
			},
			testTime: testTime, // Monday 14:30
			expected: true,
		},
		{
			name: "inactive time window - outside time range",
			activation: &TimeBasedActivation{
				Enabled: true,
				ActivePeriods: []TimePeriod{
					{
						StartTime:  "01:00",
						EndTime:    "02:00",
						DaysOfWeek: []string{"Monday"},
					},
				},
				Timezone: "UTC",
			},
			testTime: testTime, // Monday 14:30 (outside 1-2 AM)
			expected: false,
		},
		{
			name: "wrong day of week",
			activation: &TimeBasedActivation{
				Enabled: true,
				ActivePeriods: []TimePeriod{
					{
						StartTime:  "09:00",
						EndTime:    "17:00",
						DaysOfWeek: []string{"Tuesday", "Wednesday"},
					},
				},
				Timezone: "UTC",
			},
			testTime: testTime, // Monday
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsActiveInTimeWindow(tt.activation, tt.testTime)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v (time: %s)", tt.expected, result, tt.testTime.Format("Monday 15:04"))
			}
		})
	}
}

// Test Policy Exemptions

func TestIsExempt(t *testing.T) {
	ctx := context.Background()
	client := createTestClient()
	manager := NewAdvancedPolicyManager(client)

	futureTime := metav1.NewTime(time.Now().Add(24 * time.Hour))
	pastTime := metav1.NewTime(time.Now().Add(-24 * time.Hour))

	tests := []struct {
		name              string
		exemptions        []PolicyExemption
		resourceKind      string
		resourceName      string
		resourceNamespace string
		resourceLabels    map[string]string
		expectedExempt    bool
		expectedReason    string
	}{
		{
			name: "exact match by name and namespace",
			exemptions: []PolicyExemption{
				{
					Name:      "test-exemption",
					Reason:    "maintenance window",
					ExpiresAt: &futureTime,
					Resources: []ResourceSelector{
						{
							Kind:      "Pod",
							Name:      "test-pod",
							Namespace: "default",
						},
					},
				},
			},
			resourceKind:      "Pod",
			resourceName:      "test-pod",
			resourceNamespace: "default",
			resourceLabels:    nil,
			expectedExempt:    true,
			expectedReason:    "maintenance window",
		},
		{
			name: "expired exemption",
			exemptions: []PolicyExemption{
				{
					Name:      "expired-exemption",
					Reason:    "old exemption",
					ExpiresAt: &pastTime,
					Resources: []ResourceSelector{
						{
							Kind:      "Pod",
							Name:      "test-pod",
							Namespace: "default",
						},
					},
				},
			},
			resourceKind:      "Pod",
			resourceName:      "test-pod",
			resourceNamespace: "default",
			resourceLabels:    nil,
			expectedExempt:    false,
			expectedReason:    "",
		},
		{
			name: "label-based exemption",
			exemptions: []PolicyExemption{
				{
					Name:      "label-exemption",
					Reason:    "critical workload",
					ExpiresAt: &futureTime,
					Resources: []ResourceSelector{
						{
							Kind:      "Pod",
							Namespace: "default",
							Labels: map[string]string{
								"app": "critical",
							},
						},
					},
				},
			},
			resourceKind:      "Pod",
			resourceName:      "any-pod",
			resourceNamespace: "default",
			resourceLabels: map[string]string{
				"app": "critical",
			},
			expectedExempt: true,
			expectedReason: "critical workload",
		},
		{
			name: "non-matching resource",
			exemptions: []PolicyExemption{
				{
					Name:      "test-exemption",
					Reason:    "test",
					ExpiresAt: &futureTime,
					Resources: []ResourceSelector{
						{
							Kind:      "Pod",
							Name:      "other-pod",
							Namespace: "default",
						},
					},
				},
			},
			resourceKind:      "Pod",
			resourceName:      "test-pod",
			resourceNamespace: "default",
			resourceLabels:    nil,
			expectedExempt:    false,
			expectedReason:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exempt, reason := manager.IsExempt(ctx, tt.exemptions, tt.resourceKind, tt.resourceName, tt.resourceNamespace, tt.resourceLabels)
			if exempt != tt.expectedExempt {
				t.Errorf("Expected exempt=%v, got %v", tt.expectedExempt, exempt)
			}
			if reason != tt.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tt.expectedReason, reason)
			}
		})
	}
}

// Test Namespace Scoping

func TestApplyNamespaceScope(t *testing.T) {
	client := createTestClient()
	manager := NewAdvancedPolicyManager(client)

	tests := []struct {
		name      string
		scope     *NamespaceScope
		namespace string
		expected  bool
	}{
		{
			name: "included namespace",
			scope: &NamespaceScope{
				IncludeNamespaces: []string{"default", "kube-system"},
			},
			namespace: "default",
			expected:  true,
		},
		{
			name: "excluded namespace",
			scope: &NamespaceScope{
				ExcludeNamespaces: []string{"test", "dev"},
			},
			namespace: "test",
			expected:  false,
		},
		{
			name: "not in include list",
			scope: &NamespaceScope{
				IncludeNamespaces: []string{"production"},
			},
			namespace: "dev",
			expected:  false,
		},
		{
			name: "in include list but also excluded",
			scope: &NamespaceScope{
				IncludeNamespaces: []string{"default", "test"},
				ExcludeNamespaces: []string{"test"},
			},
			namespace: "test",
			expected:  false, // Exclusions take precedence
		},
		{
			name:      "no scope (applies to all)",
			scope:     nil,
			namespace: "any-namespace",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.ApplyNamespaceScope(tt.scope, tt.namespace)
			if result != tt.expected {
				t.Errorf("Expected %v for namespace %s, got %v", tt.expected, tt.namespace, result)
			}
		})
	}
}

// Test Helper Functions

func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "grape",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestContainsValueFunction(t *testing.T) {
	tests := []struct {
		name     string
		slice    []interface{}
		value    interface{}
		expected bool
	}{
		{
			name:     "string value exists",
			slice:    []interface{}{"low", "medium", "high"},
			value:    "medium",
			expected: true,
		},
		{
			name:     "int value exists",
			slice:    []interface{}{1, 2, 3, 4, 5},
			value:    3,
			expected: true,
		},
		{
			name:     "value does not exist",
			slice:    []interface{}{"low", "medium", "high"},
			value:    "critical",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []interface{}{},
			value:    "anything",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsValue(tt.slice, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test Default Templates

func TestInitializeDefaultTemplates(t *testing.T) {
	templates := initializeDefaultTemplates()

	if len(templates) == 0 {
		t.Error("Expected at least one default template")
	}

	// Verify security-baseline template exists
	securityBaseline, exists := templates["security-baseline"]
	if !exists {
		t.Error("Expected 'security-baseline' default template to exist")
	}

	if securityBaseline != nil {
		if securityBaseline.Category != "security" {
			t.Errorf("Expected security-baseline template to have category 'security', got '%s'", securityBaseline.Category)
		}
		if len(securityBaseline.Parameters) == 0 {
			t.Error("Expected security-baseline template to have parameters")
		}
	}

	// Verify compliance-strict template exists
	complianceStrict, exists := templates["compliance-strict"]
	if !exists {
		t.Error("Expected 'compliance-strict' default template to exist")
	}

	if complianceStrict != nil {
		if complianceStrict.Category != "compliance" {
			t.Errorf("Expected compliance-strict template to have category 'compliance', got '%s'", complianceStrict.Category)
		}
	}
}

// Integration Test

func TestAdvancedPolicyWorkflow(t *testing.T) {
	ctx := context.Background()
	client := createTestClient()
	manager := NewAdvancedPolicyManager(client)

	// 1. Register and apply a template
	manager.Templates["production-security"] = &PolicyTemplate{
		Name:        "production-security",
		Description: "Security policy for production workloads",
		Category:    "security",
		Parameters: []TemplateParameter{
			{
				Name:     "env",
				Type:     "string",
				Required: true,
				AllowedValues: []interface{}{
					"dev", "staging", "production",
				},
			},
		},
		BasePolicy: PolicyDefinition{
			RequiredFields: []FieldRequirement{
				{
					Key:   "spec.securityContext.runAsNonRoot",
					Value: "true",
				},
			},
		},
	}

	// 2. Apply template with parameters
	params := map[string]interface{}{
		"env": "production",
	}

	policy, err := manager.ApplyTemplate(ctx, "production-security", params)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}
	if policy == nil {
		t.Fatal("Expected policy but got nil")
	}

	// 3. Test time-based activation
	activation := &TimeBasedActivation{
		Enabled: true,
		ActivePeriods: []TimePeriod{
			{
				StartTime:  "09:00",
				EndTime:    "17:00",
				DaysOfWeek: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			},
		},
		Timezone: "UTC",
	}

	testTime := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC) // Monday 2 PM
	isActive := manager.IsActiveInTimeWindow(activation, testTime)
	if !isActive {
		t.Error("Expected policy to be active during business hours")
	}

	// 4. Test namespace scoping
	scope := &NamespaceScope{
		IncludeNamespaces: []string{"production", "staging"},
		ExcludeNamespaces: []string{"test"},
	}

	shouldApply := manager.ApplyNamespaceScope(scope, "production")
	if !shouldApply {
		t.Error("Expected policy to apply to 'production' namespace")
	}

	shouldNotApply := manager.ApplyNamespaceScope(scope, "test")
	if shouldNotApply {
		t.Error("Expected policy NOT to apply to 'test' namespace")
	}

	// 5. Test exemptions
	futureTime := metav1.NewTime(time.Now().Add(24 * time.Hour))
	exemptions := []PolicyExemption{
		{
			Name:      "maintenance-exemption",
			Reason:    "Scheduled maintenance window",
			ExpiresAt: &futureTime,
			Resources: []ResourceSelector{
				{
					Kind:      "Pod",
					Name:      "maintenance-pod",
					Namespace: "production",
				},
			},
			Approver: "ops-team",
		},
	}

	isExempt, reason := manager.IsExempt(ctx, exemptions, "Pod", "maintenance-pod", "production", nil)
	if !isExempt {
		t.Error("Expected resource to be exempt")
	}
	if reason != "Scheduled maintenance window" {
		t.Errorf("Expected reason 'Scheduled maintenance window', got '%s'", reason)
	}

	t.Log("Advanced policy workflow completed successfully")
}
