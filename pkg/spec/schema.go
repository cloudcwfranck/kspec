// Package spec defines the cluster specification schema for kspec.
package spec

// ClusterSpecification represents the complete cluster specification.
type ClusterSpecification struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind" json:"kind"`
	Metadata   Metadata   `yaml:"metadata" json:"metadata"`
	Spec       SpecFields `yaml:"spec" json:"spec"`
}

// Metadata contains specification metadata.
type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// SpecFields contains all specification requirements.
type SpecFields struct {
	Kubernetes    KubernetesSpec     `yaml:"kubernetes" json:"kubernetes"`
	PodSecurity   *PodSecuritySpec   `yaml:"podSecurity,omitempty" json:"podSecurity,omitempty"`
	Network       *NetworkSpec       `yaml:"network,omitempty" json:"network,omitempty"`
	Workloads     *WorkloadsSpec     `yaml:"workloads,omitempty" json:"workloads,omitempty"`
	RBAC          *RBACSpec          `yaml:"rbac,omitempty" json:"rbac,omitempty"`
	Admission     *AdmissionSpec     `yaml:"admission,omitempty" json:"admission,omitempty"`
	Observability *ObservabilitySpec `yaml:"observability,omitempty" json:"observability,omitempty"`
	Compliance    *ComplianceSpec    `yaml:"compliance,omitempty" json:"compliance,omitempty"`
}

// KubernetesSpec defines Kubernetes version requirements.
type KubernetesSpec struct {
	MinVersion       string   `yaml:"minVersion" json:"minVersion"`
	MaxVersion       string   `yaml:"maxVersion" json:"maxVersion"`
	ExcludedVersions []string `yaml:"excludedVersions,omitempty" json:"excludedVersions,omitempty"`
}

// PodSecuritySpec defines Pod Security Standards requirements.
type PodSecuritySpec struct {
	Enforce    string                 `yaml:"enforce" json:"enforce"` // baseline, restricted, privileged
	Audit      string                 `yaml:"audit" json:"audit"`
	Warn       string                 `yaml:"warn" json:"warn"`
	Exemptions []PodSecurityExemption `yaml:"exemptions,omitempty" json:"exemptions,omitempty"`
}

// PodSecurityExemption defines exemptions from Pod Security Standards.
type PodSecurityExemption struct {
	Namespace string `yaml:"namespace" json:"namespace"`
	Level     string `yaml:"level" json:"level"`
	Reason    string `yaml:"reason" json:"reason"`
}

// NetworkSpec defines network policy requirements.
type NetworkSpec struct {
	DefaultDeny         bool             `yaml:"defaultDeny" json:"defaultDeny"`
	RequiredPolicies    []RequiredPolicy `yaml:"requiredPolicies,omitempty" json:"requiredPolicies,omitempty"`
	AllowedServiceTypes []string         `yaml:"allowedServiceTypes,omitempty" json:"allowedServiceTypes,omitempty"`
	DisallowedPorts     []int            `yaml:"disallowedPorts,omitempty" json:"disallowedPorts,omitempty"`
}

// RequiredPolicy defines a required network policy.
type RequiredPolicy struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
}

// WorkloadsSpec defines workload security requirements.
type WorkloadsSpec struct {
	Containers *ContainerSpec `yaml:"containers,omitempty" json:"containers,omitempty"`
	Images     *ImageSpec     `yaml:"images,omitempty" json:"images,omitempty"`
}

// ContainerSpec defines container security requirements.
type ContainerSpec struct {
	Required  []FieldRequirement `yaml:"required,omitempty" json:"required,omitempty"`
	Forbidden []FieldRequirement `yaml:"forbidden,omitempty" json:"forbidden,omitempty"`
}

// FieldRequirement defines a required or forbidden field.
type FieldRequirement struct {
	Key    string `yaml:"key" json:"key"`
	Value  string `yaml:"value,omitempty" json:"value,omitempty"`
	Exists *bool  `yaml:"exists,omitempty" json:"exists,omitempty"`
}

// ImageSpec defines image security requirements.
type ImageSpec struct {
	AllowedRegistries []string `yaml:"allowedRegistries,omitempty" json:"allowedRegistries,omitempty"`
	BlockedRegistries []string `yaml:"blockedRegistries,omitempty" json:"blockedRegistries,omitempty"`
	RequireDigests    bool     `yaml:"requireDigests" json:"requireDigests"`
	RequireSignatures bool     `yaml:"requireSignatures" json:"requireSignatures"`
}

// RBACSpec defines RBAC requirements.
type RBACSpec struct {
	MinimumRules   []RBACRule `yaml:"minimumRules,omitempty" json:"minimumRules,omitempty"`
	ForbiddenRules []RBACRule `yaml:"forbiddenRules,omitempty" json:"forbiddenRules,omitempty"`
}

// RBACRule defines an RBAC rule.
type RBACRule struct {
	APIGroup string   `yaml:"apiGroup" json:"apiGroup"`
	Resource string   `yaml:"resource" json:"resource"`
	Verbs    []string `yaml:"verbs" json:"verbs"`
}

// AdmissionSpec defines admission controller requirements.
type AdmissionSpec struct {
	Required []AdmissionRequirement `yaml:"required,omitempty" json:"required,omitempty"`
	Policies *PolicySpec            `yaml:"policies,omitempty" json:"policies,omitempty"`
}

// AdmissionRequirement defines a required admission controller.
type AdmissionRequirement struct {
	Type        string `yaml:"type" json:"type"`
	NamePattern string `yaml:"namePattern" json:"namePattern"`
	MinCount    int    `yaml:"minCount" json:"minCount"`
}

// PolicySpec defines policy requirements.
type PolicySpec struct {
	MinCount         int              `yaml:"minCount" json:"minCount"`
	RequiredPolicies []RequiredPolicy `yaml:"requiredPolicies,omitempty" json:"requiredPolicies,omitempty"`
}

// ObservabilitySpec defines observability requirements.
type ObservabilitySpec struct {
	Metrics *MetricsSpec `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Logging *LoggingSpec `yaml:"logging,omitempty" json:"logging,omitempty"`
}

// MetricsSpec defines metrics requirements.
type MetricsSpec struct {
	Required  bool     `yaml:"required" json:"required"`
	Providers []string `yaml:"providers,omitempty" json:"providers,omitempty"`
}

// LoggingSpec defines logging requirements.
type LoggingSpec struct {
	AuditLog *AuditLogSpec `yaml:"auditLog,omitempty" json:"auditLog,omitempty"`
}

// AuditLogSpec defines audit log requirements.
type AuditLogSpec struct {
	Required         bool `yaml:"required" json:"required"`
	MinRetentionDays int  `yaml:"minRetentionDays" json:"minRetentionDays"`
}

// ComplianceSpec defines compliance framework mappings.
type ComplianceSpec struct {
	Frameworks []ComplianceFramework `yaml:"frameworks,omitempty" json:"frameworks,omitempty"`
}

// ComplianceFramework defines a compliance framework.
type ComplianceFramework struct {
	Name     string              `yaml:"name" json:"name"`
	Revision string              `yaml:"revision,omitempty" json:"revision,omitempty"`
	Controls []ComplianceControl `yaml:"controls,omitempty" json:"controls,omitempty"`
}

// ComplianceControl defines a compliance control.
type ComplianceControl struct {
	ID       string           `yaml:"id" json:"id"`
	Title    string           `yaml:"title" json:"title"`
	Mappings []ControlMapping `yaml:"mappings,omitempty" json:"mappings,omitempty"`
}

// ControlMapping maps a compliance control to a check.
type ControlMapping struct {
	Check string `yaml:"check" json:"check"`
}
