package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComplianceReportSpec defines the desired state of ComplianceReport
type ComplianceReportSpec struct {
	// ClusterSpecRef references the ClusterSpecification this report is for
	// +kubebuilder:validation:Required
	ClusterSpecRef ObjectReference `json:"clusterSpecRef"`

	// ScanTime is when this compliance scan was performed
	// +kubebuilder:validation:Required
	ScanTime metav1.Time `json:"scanTime"`

	// Summary provides an overview of the compliance results
	// +kubebuilder:validation:Required
	Summary ReportSummary `json:"summary"`

	// Results contains the detailed compliance check results
	// +optional
	Results []CheckResult `json:"results,omitempty"`
}

// ObjectReference contains enough information to locate a referenced object
type ObjectReference struct {
	// Name of the referenced object
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Version of the specification
	// +optional
	Version string `json:"version,omitempty"`
}

// ReportSummary provides summary statistics for a compliance report
type ReportSummary struct {
	// Total number of checks performed
	// +kubebuilder:validation:Minimum=0
	Total int `json:"total"`

	// Number of checks that passed
	// +kubebuilder:validation:Minimum=0
	Passed int `json:"passed"`

	// Number of checks that failed
	// +kubebuilder:validation:Minimum=0
	Failed int `json:"failed"`

	// Overall pass rate percentage (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	PassRate int `json:"passRate"`
}

// CheckResult represents the result of a single compliance check
type CheckResult struct {
	// Name of the check
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Category of the check (e.g., kubernetes, podSecurity, network)
	// +kubebuilder:validation:Required
	Category string `json:"category"`

	// Status of the check
	// +kubebuilder:validation:Enum=Pass;Fail;Error
	// +kubebuilder:validation:Required
	Status string `json:"status"`

	// Severity of the check
	// +kubebuilder:validation:Enum=Low;Medium;High;Critical
	// +kubebuilder:validation:Required
	Severity string `json:"severity"`

	// Message describing the check result
	// +optional
	Message string `json:"message,omitempty"`

	// Details provides additional context about the check result
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Details map[string]interface{} `json:"details,omitempty"`
}

// ComplianceReportStatus defines the observed state of ComplianceReport
type ComplianceReportStatus struct {
	// Phase represents the current phase of the report
	// +kubebuilder:validation:Enum=Pending;Completed;Failed
	// +kubebuilder:default=Pending
	Phase string `json:"phase,omitempty"`

	// ReportURL is the URL where the full report can be accessed (for Phase 8 control plane)
	// +optional
	ReportURL string `json:"reportURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=cr
// +kubebuilder:printcolumn:name="Cluster Spec",type=string,JSONPath=`.spec.clusterSpecRef.name`
// +kubebuilder:printcolumn:name="Pass Rate",type=integer,JSONPath=`.spec.summary.passRate`
// +kubebuilder:printcolumn:name="Scan Time",type=date,JSONPath=`.spec.scanTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ComplianceReport is the Schema for the compliancereports API
type ComplianceReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComplianceReportSpec   `json:"spec,omitempty"`
	Status ComplianceReportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComplianceReportList contains a list of ComplianceReport
type ComplianceReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComplianceReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ComplianceReport{}, &ComplianceReportList{})
}
