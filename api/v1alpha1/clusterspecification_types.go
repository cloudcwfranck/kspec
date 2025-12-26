package v1alpha1

import (
	"github.com/cloudcwfranck/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSpecificationSpec defines the desired state of ClusterSpecification
// This reuses the existing SpecFields from pkg/spec/schema.go
type ClusterSpecificationSpec struct {
	// Spec contains the cluster security and compliance requirements
	// +kubebuilder:validation:Required
	spec.SpecFields `json:",inline"`
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
