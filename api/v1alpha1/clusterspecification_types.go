package v1alpha1

import (
	"github.com/cloudcwfranck/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSpecificationSpec defines the desired state of ClusterSpecification
// This reuses the existing SpecFields from pkg/spec/schema.go
type ClusterSpecificationSpec struct {
	// ClusterRef is an optional reference to a ClusterTarget defining a remote cluster
	// If not specified, the operator will scan the local cluster (backwards compatible)
	// +optional
	ClusterRef *ClusterReference `json:"clusterRef,omitempty"`

	// Enforcement defines enforcement behavior for this specification
	// +optional
	Enforcement *EnforcementSpec `json:"enforcement,omitempty"`

	// Webhooks configures admission webhook behavior
	// +optional
	Webhooks *WebhooksSpec `json:"webhooks,omitempty"`

	// PolicyTemplate references a policy template to use
	// +optional
	PolicyTemplate *PolicyTemplateRef `json:"policyTemplate,omitempty"`

	// PolicyInheritance defines policy composition through inheritance
	// +optional
	PolicyInheritance *PolicyInheritanceSpec `json:"policyInheritance,omitempty"`

	// NamespaceScope restricts policy to specific namespaces
	// +optional
	NamespaceScope *NamespaceScopeSpec `json:"namespaceScope,omitempty"`

	// TimeBasedActivation enables time-based policy activation
	// +optional
	TimeBasedActivation *TimeBasedActivationSpec `json:"timeBasedActivation,omitempty"`

	// PolicyExemptions defines resources exempt from this policy
	// +optional
	PolicyExemptions []PolicyExemptionSpec `json:"policyExemptions,omitempty"`

	spec.SpecFields `json:",inline"`
}

// PolicyTemplateRef references a policy template
type PolicyTemplateRef struct {
	// Name of the policy template
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Parameters for the template
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// PolicyInheritanceSpec defines policy inheritance
type PolicyInheritanceSpec struct {
	// BasePolicies are parent policies to inherit from
	// +optional
	BasePolicies []string `json:"basePolicies,omitempty"`

	// MergeStrategy defines how to merge inherited policies
	// +optional
	// +kubebuilder:validation:Enum=merge;override;append
	// +kubebuilder:default=merge
	MergeStrategy string `json:"mergeStrategy,omitempty"`
}

// NamespaceScopeSpec defines namespace-level scoping
type NamespaceScopeSpec struct {
	// IncludeNamespaces lists namespaces to include
	// +optional
	IncludeNamespaces []string `json:"includeNamespaces,omitempty"`

	// ExcludeNamespaces lists namespaces to exclude
	// +optional
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`

	// NamespaceSelector selects namespaces by labels
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// TimeBasedActivationSpec defines time-based activation
type TimeBasedActivationSpec struct {
	// Enabled controls time-based activation
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Schedule defines when the policy is active (cron format)
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Timezone for schedule evaluation
	// +optional
	// +kubebuilder:default=UTC
	Timezone string `json:"timezone,omitempty"`

	// ActivePeriods defines specific time ranges
	// +optional
	ActivePeriods []TimePeriodSpec `json:"activePeriods,omitempty"`
}

// TimePeriodSpec defines a time range
type TimePeriodSpec struct {
	// StartTime in HH:MM format
	// +optional
	StartTime string `json:"startTime,omitempty"`

	// EndTime in HH:MM format
	// +optional
	EndTime string `json:"endTime,omitempty"`

	// DaysOfWeek when this period is active
	// +optional
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`

	// StartDate for the period
	// +optional
	StartDate *metav1.Time `json:"startDate,omitempty"`

	// EndDate for the period
	// +optional
	EndDate *metav1.Time `json:"endDate,omitempty"`
}

// PolicyExemptionSpec defines a policy exemption
type PolicyExemptionSpec struct {
	// Name of the exemption
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Reason for the exemption
	// +optional
	Reason string `json:"reason,omitempty"`

	// ExpiresAt defines when the exemption expires
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Namespaces covered by this exemption
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// Resources covered by this exemption
	// +optional
	Resources []ResourceSelectorSpec `json:"resources,omitempty"`

	// Approver who approved this exemption
	// +optional
	Approver string `json:"approver,omitempty"`
}

// ResourceSelectorSpec selects specific resources
type ResourceSelectorSpec struct {
	// Kind of resource
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name of resource
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace of resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// LabelSelector for resources
	// +optional
	LabelSelector map[string]string `json:"labelSelector,omitempty"`
}

// ClusterReference references a ClusterTarget resource
type ClusterReference struct {
	// Name is the name of the ClusterTarget resource
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the ClusterTarget resource
	// If not specified, uses the same namespace as the ClusterSpecification
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// EnforcementSpec defines enforcement behavior
type EnforcementSpec struct {
	// Enabled controls whether enforcement is active
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Mode defines the enforcement mode: monitor, audit, enforce
	// monitor: no enforcement, only monitoring
	// audit: log violations but don't block
	// enforce: actively block violations
	// +optional
	// +kubebuilder:validation:Enum=monitor;audit;enforce
	// +kubebuilder:default=monitor
	Mode string `json:"mode,omitempty"`

	// AutoRemediate enables automatic remediation of violations
	// +optional
	// +kubebuilder:default=false
	AutoRemediate bool `json:"autoRemediate,omitempty"`
}

// WebhooksSpec defines webhook admission control configuration
type WebhooksSpec struct {
	// Enabled controls whether admission webhooks are active
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// FailurePolicy defines behavior when webhook fails: Ignore or Fail
	// Ignore: continue on webhook failure (fail-open, safe default)
	// Fail: reject request on webhook failure (fail-closed)
	// +optional
	// +kubebuilder:validation:Enum=Ignore;Fail
	// +kubebuilder:default=Ignore
	FailurePolicy string `json:"failurePolicy,omitempty"`

	// TimeoutSeconds is the webhook timeout in seconds
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=10
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// Certificate configures TLS certificate for webhooks
	// +optional
	Certificate *CertificateSpec `json:"certificate,omitempty"`
}

// CertificateSpec defines certificate configuration
type CertificateSpec struct {
	// Issuer is the name of the cert-manager Issuer/ClusterIssuer
	// +optional
	Issuer string `json:"issuer,omitempty"`

	// IssuerKind is the kind of issuer (Issuer or ClusterIssuer)
	// +optional
	// +kubebuilder:validation:Enum=Issuer;ClusterIssuer
	// +kubebuilder:default=ClusterIssuer
	IssuerKind string `json:"issuerKind,omitempty"`
}

// ClusterSpecificationStatus defines the observed state of ClusterSpecification
type ClusterSpecificationStatus struct {
	// Phase represents the current phase of the cluster specification
	// +kubebuilder:validation:Enum=Pending;Active;Failed
	// +kubebuilder:default=Pending
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastScanTime is the timestamp of the last compliance scan
	// +optional
	LastScanTime *metav1.Time `json:"lastScanTime,omitempty"`

	// ComplianceScore is the overall compliance score (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	ComplianceScore int `json:"complianceScore,omitempty"`

	// Conditions represent the latest available observations of the ClusterSpecification's state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Summary contains a summary of compliance check results
	// +optional
	Summary *ComplianceSummary `json:"summary,omitempty"`

	// Enforcement tracks enforcement state
	// +optional
	Enforcement *EnforcementStatus `json:"enforcement,omitempty"`

	// Webhooks tracks webhook state
	// +optional
	Webhooks *WebhooksStatus `json:"webhooks,omitempty"`
}

// ComplianceSummary provides a summary of compliance check results
type ComplianceSummary struct {
	// TotalChecks is the total number of compliance checks performed
	TotalChecks int `json:"totalChecks"`

	// PassedChecks is the number of checks that passed
	PassedChecks int `json:"passedChecks"`

	// FailedChecks is the number of checks that failed
	FailedChecks int `json:"failedChecks"`

	// PoliciesEnforced is the number of policies currently enforced
	// +optional
	PoliciesEnforced int `json:"policiesEnforced,omitempty"`

	// DriftEvents is the number of drift events detected
	// +optional
	DriftEvents int `json:"driftEvents,omitempty"`
}

// EnforcementStatus tracks enforcement state
type EnforcementStatus struct {
	// Active indicates if enforcement is currently active
	Active bool `json:"active"`

	// Mode is the current enforcement mode
	Mode string `json:"mode,omitempty"`

	// PoliciesGenerated is the number of Kyverno policies generated
	PoliciesGenerated int `json:"policiesGenerated,omitempty"`

	// LastEnforcementTime is when enforcement was last updated
	// +optional
	LastEnforcementTime *metav1.Time `json:"lastEnforcementTime,omitempty"`
}

// WebhooksStatus tracks webhook state
type WebhooksStatus struct {
	// Active indicates if webhooks are currently active
	Active bool `json:"active"`

	// CertificateReady indicates if TLS certificate is ready
	CertificateReady bool `json:"certificateReady"`

	// ErrorRate is the webhook error rate (0.0-1.0)
	// +optional
	ErrorRate float64 `json:"errorRate,omitempty"`

	// CircuitBreakerTripped indicates if circuit breaker is active
	// +optional
	CircuitBreakerTripped bool `json:"circuitBreakerTripped,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=clusterspec;cspec
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Score",type=integer,JSONPath=`.status.complianceScore`
// +kubebuilder:printcolumn:name="Last Scan",type=date,JSONPath=`.status.lastScanTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterSpecification is the Schema for the clusterspecifications API
type ClusterSpecification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpecificationSpec   `json:"spec,omitempty"`
	Status ClusterSpecificationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterSpecificationList contains a list of ClusterSpecification
type ClusterSpecificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterSpecification `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterSpecification{}, &ClusterSpecificationList{})
}
