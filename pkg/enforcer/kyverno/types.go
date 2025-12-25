package kyverno

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterPolicy defines a Kyverno cluster policy.
// This is a vendored subset of github.com/kyverno/kyverno/api/kyverno/v1
// to avoid heavyweight dependencies while maintaining API compatibility.
type ClusterPolicy struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              ClusterPolicySpec `json:"spec" yaml:"spec"`
}

// ClusterPolicySpec defines the policy specification.
type ClusterPolicySpec struct {
	// ValidationFailureAction controls the action on validation failure
	ValidationFailureAction ValidationFailureAction `json:"validationFailureAction,omitempty"`

	// Background controls whether the policy applies to existing resources
	Background *bool `json:"background,omitempty"`

	// Rules is a list of policy rules
	Rules []Rule `json:"rules"`
}

// ValidationFailureAction defines the action on validation failure.
type ValidationFailureAction string

const (
	// Enforce blocks the operation
	Enforce ValidationFailureAction = "Enforce"
	// Audit allows the operation but logs a policy violation
	Audit ValidationFailureAction = "Audit"
)

// Rule defines a single policy rule.
type Rule struct {
	// Name is the rule name
	Name string `json:"name"`

	// Match defines when this rule should be evaluated
	Match MatchResources `json:"match,omitempty"`

	// Exclude defines when this rule should not be evaluated
	Exclude MatchResources `json:"exclude,omitempty"`

	// Validation defines the validation rule
	Validation *Validation `json:"validate,omitempty"`

	// Mutation defines the mutation rule
	Mutation *Mutation `json:"mutate,omitempty"`
}

// MatchResources defines resource filters for a rule.
type MatchResources struct {
	// Any allows matching any of the specified resources
	Any []ResourceFilter `json:"any,omitempty"`

	// All requires matching all of the specified resources
	All []ResourceFilter `json:"all,omitempty"`
}

// ResourceFilter defines resource selection criteria.
type ResourceFilter struct {
	// Resources defines the resource types to match
	Resources *ResourceDescription `json:"resources,omitempty"`

	// Subjects defines the subjects to match (for RBAC policies)
	Subjects []Subject `json:"subjects,omitempty"`

	// Roles defines the roles to match
	Roles []string `json:"roles,omitempty"`

	// ClusterRoles defines the cluster roles to match
	ClusterRoles []string `json:"clusterRoles,omitempty"`
}

// ResourceDescription defines Kubernetes resources to match.
type ResourceDescription struct {
	// Kinds is a list of resource kinds
	Kinds []string `json:"kinds,omitempty"`

	// Names is a list of resource names
	Names []string `json:"names,omitempty"`

	// Namespaces is a list of namespaces
	Namespaces []string `json:"namespaces,omitempty"`

	// Selector is a label selector
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// Subject defines a subject for RBAC policies.
type Subject struct {
	// Kind of subject (User, Group, ServiceAccount)
	Kind string `json:"kind,omitempty"`

	// Name of the subject
	Name string `json:"name,omitempty"`

	// Namespace of the subject (for ServiceAccount)
	Namespace string `json:"namespace,omitempty"`
}

// Validation defines validation rules.
type Validation struct {
	// Message is the message to display on validation failure
	Message string `json:"message,omitempty"`

	// Pattern defines the validation pattern
	Pattern interface{} `json:"pattern,omitempty"`

	// AnyPattern defines multiple validation patterns (OR logic)
	AnyPattern []interface{} `json:"anyPattern,omitempty"`

	// Deny specifies a deny condition
	Deny *Deny `json:"deny,omitempty"`
}

// Deny defines a deny condition.
type Deny struct {
	// Conditions defines the conditions for denial
	Conditions interface{} `json:"conditions,omitempty"`
}

// Mutation defines mutation rules.
type Mutation struct {
	// PatchStrategicMerge defines a strategic merge patch
	PatchStrategicMerge interface{} `json:"patchStrategicMerge,omitempty"`

	// PatchesJSON6902 defines JSON patches (RFC 6902)
	PatchesJSON6902 string `json:"patchesJson6902,omitempty"`
}

// NewClusterPolicy creates a new ClusterPolicy with standard defaults.
func NewClusterPolicy(name string) *ClusterPolicy {
	trueVal := true
	return &ClusterPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kyverno.io/v1",
			Kind:       "ClusterPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"kspec.dev/generated": "true",
			},
		},
		Spec: ClusterPolicySpec{
			ValidationFailureAction: Enforce,
			Background:              &trueVal,
			Rules:                   []Rule{},
		},
	}
}

// DeepCopyObject implements runtime.Object interface for ClusterPolicy.
// This is required for ClusterPolicy to be used as a Kubernetes API object.
func (c *ClusterPolicy) DeepCopyObject() runtime.Object {
	if c == nil {
		return nil
	}
	out := new(ClusterPolicy)
	c.DeepCopyInto(out)
	return out
}

// DeepCopyInto performs a deep copy of ClusterPolicy into out.
func (c *ClusterPolicy) DeepCopyInto(out *ClusterPolicy) {
	*out = *c
	out.TypeMeta = c.TypeMeta
	c.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	c.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopyInto performs a deep copy of ClusterPolicySpec into out.
func (s *ClusterPolicySpec) DeepCopyInto(out *ClusterPolicySpec) {
	*out = *s
	if s.Background != nil {
		in, out := &s.Background, &out.Background
		*out = new(bool)
		**out = **in
	}
	if s.Rules != nil {
		in, out := &s.Rules, &out.Rules
		*out = make([]Rule, len(*in))
		copy(*out, *in)
	}
}
