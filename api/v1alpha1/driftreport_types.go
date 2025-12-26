package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DriftReportSpec defines the desired state of DriftReport
type DriftReportSpec struct {
	// ClusterSpecRef references the ClusterSpecification this report is for
	// +kubebuilder:validation:Required
	ClusterSpecRef ObjectReference `json:"clusterSpecRef"`

	// ClusterName is the name of the cluster that was scanned
	// For local clusters this is "local", for remote clusters it's the ClusterTarget name
	// +kubebuilder:validation:Required
	ClusterName string `json:"clusterName"`

	// ClusterUID is the unique identifier of the cluster
	// This helps distinguish reports across different clusters
	// +optional
	ClusterUID string `json:"clusterUID,omitempty"`

	// DetectionTime is when drift was detected
	// +kubebuilder:validation:Required
	DetectionTime metav1.Time `json:"detectionTime"`

	// DriftDetected indicates whether drift was found
	// +kubebuilder:validation:Required
	DriftDetected bool `json:"driftDetected"`

	// Severity of the drift
	// +kubebuilder:validation:Enum=low;medium;high;critical
	// +optional
	Severity string `json:"severity,omitempty"`

	// Events contains the individual drift events detected
	// +optional
	Events []DriftEvent `json:"events,omitempty"`
}

// DriftEvent represents a single drift event
type DriftEvent struct {
	// Type of drift (Policy, Compliance, Configuration)
	// +kubebuilder:validation:Enum=Policy;Compliance;Configuration
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Severity of this drift event
	// +kubebuilder:validation:Enum=low;medium;high;critical
	// +kubebuilder:validation:Required
	Severity string `json:"severity"`

	// Resource identifies the resource that drifted
	// +optional
	Resource *ResourceReference `json:"resource,omitempty"`

	// DriftType describes how the resource drifted
	// +kubebuilder:validation:Enum=deleted;modified;violation
	// +optional
	DriftType string `json:"driftType,omitempty"`

	// Check name for compliance drift
	// +optional
	Check string `json:"check,omitempty"`

	// Message describes the drift
	// +optional
	Message string `json:"message,omitempty"`

	// Expected state
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Expected *runtime.RawExtension `json:"expected,omitempty"`

	// Actual state
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Actual *runtime.RawExtension `json:"actual,omitempty"`

	// Remediation describes the remediation action taken
	// +optional
	Remediation *RemediationAction `json:"remediation,omitempty"`
}

// ResourceReference identifies a Kubernetes resource
type ResourceReference struct {
	// Kind of the resource
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name of the resource
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the resource (empty for cluster-scoped)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// RemediationAction describes a remediation action
type RemediationAction struct {
	// Action taken (create, update, delete, report)
	// +kubebuilder:validation:Enum=create;update;delete;report
	// +kubebuilder:validation:Required
	Action string `json:"action"`

	// Status of the remediation
	// +kubebuilder:validation:Enum=success;failed;pending;manual-required
	// +kubebuilder:validation:Required
	Status string `json:"status"`

	// AppliedAt is when the remediation was applied
	// +optional
	AppliedAt *metav1.Time `json:"appliedAt,omitempty"`

	// Error message if remediation failed
	// +optional
	Error string `json:"error,omitempty"`
}

// DriftReportStatus defines the observed state of DriftReport
type DriftReportStatus struct {
	// Phase represents the current phase of drift remediation
	// +kubebuilder:validation:Enum=Pending;InProgress;Completed;Failed
	// +kubebuilder:default=Pending
	Phase string `json:"phase,omitempty"`

	// TotalEvents is the total number of drift events
	// +kubebuilder:validation:Minimum=0
	// +optional
	TotalEvents int `json:"totalEvents,omitempty"`

	// RemediatedEvents is the number of events successfully remediated
	// +kubebuilder:validation:Minimum=0
	// +optional
	RemediatedEvents int `json:"remediatedEvents,omitempty"`

	// PendingEvents is the number of events pending remediation
	// +kubebuilder:validation:Minimum=0
	// +optional
	PendingEvents int `json:"pendingEvents,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dr
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterName`
// +kubebuilder:printcolumn:name="Cluster Spec",type=string,JSONPath=`.spec.clusterSpecRef.name`
// +kubebuilder:printcolumn:name="Drift",type=boolean,JSONPath=`.spec.driftDetected`
// +kubebuilder:printcolumn:name="Severity",type=string,JSONPath=`.spec.severity`
// +kubebuilder:printcolumn:name="Events",type=integer,JSONPath=`.status.totalEvents`
// +kubebuilder:printcolumn:name="Detection Time",type=date,JSONPath=`.spec.detectionTime`

// DriftReport is the Schema for the driftreports API
type DriftReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DriftReportSpec   `json:"spec,omitempty"`
	Status DriftReportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DriftReportList contains a list of DriftReport
type DriftReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DriftReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DriftReport{}, &DriftReportList{})
}
