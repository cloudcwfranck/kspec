package drift

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// DriftType represents the type of drift detected.
type DriftType string

const (
	// DriftTypePolicy indicates policy drift (Kyverno policies).
	DriftTypePolicy DriftType = "policy"

	// DriftTypeCompliance indicates compliance drift (failed checks).
	DriftTypeCompliance DriftType = "compliance"

	// DriftTypeConfiguration indicates configuration drift (cluster config).
	DriftTypeConfiguration DriftType = "configuration"
)

// DriftStatus represents the drift status after remediation.
type DriftStatus string

const (
	// DriftStatusDetected indicates drift was detected but not remediated.
	DriftStatusDetected DriftStatus = "detected"

	// DriftStatusRemediated indicates drift was successfully remediated.
	DriftStatusRemediated DriftStatus = "remediated"

	// DriftStatusFailed indicates remediation failed.
	DriftStatusFailed DriftStatus = "failed"

	// DriftStatusManualRequired indicates manual intervention required.
	DriftStatusManualRequired DriftStatus = "manual-required"
)

// DriftSeverity represents the severity of drift.
type DriftSeverity string

const (
	// SeverityCritical indicates critical drift requiring immediate attention.
	SeverityCritical DriftSeverity = "critical"

	// SeverityHigh indicates high-priority drift.
	SeverityHigh DriftSeverity = "high"

	// SeverityMedium indicates medium-priority drift.
	SeverityMedium DriftSeverity = "medium"

	// SeverityLow indicates low-priority drift.
	SeverityLow DriftSeverity = "low"
)

// DriftEvent represents a single drift detection event.
type DriftEvent struct {
	// Timestamp when drift was detected
	Timestamp time.Time `json:"timestamp"`

	// Type of drift
	Type DriftType `json:"type"`

	// Severity of drift
	Severity DriftSeverity `json:"severity"`

	// Resource identifies the resource that drifted
	Resource DriftResource `json:"resource"`

	// DriftKind describes what kind of drift occurred
	DriftKind string `json:"drift_kind"` // e.g., "deleted", "modified", "new-violation"

	// Expected state (what should exist)
	Expected interface{} `json:"expected,omitempty"`

	// Actual state (what actually exists)
	Actual interface{} `json:"actual,omitempty"`

	// Diff describes the difference (for modified resources)
	Diff *DriftDiff `json:"diff,omitempty"`

	// Message provides human-readable description
	Message string `json:"message"`

	// Remediation information
	Remediation *RemediationResult `json:"remediation,omitempty"`
}

// DriftResource identifies a specific resource.
type DriftResource struct {
	// Kind of resource (e.g., "ClusterPolicy", "Pod")
	Kind string `json:"kind"`

	// Name of resource
	Name string `json:"name"`

	// Namespace (empty for cluster-scoped resources)
	Namespace string `json:"namespace,omitempty"`

	// Full resource path (e.g., "ClusterPolicy/require-run-as-non-root")
	Path string `json:"path"`
}

// DriftDiff represents the difference between expected and actual state.
type DriftDiff struct {
	// Added fields/values
	Added map[string]interface{} `json:"added,omitempty"`

	// Removed fields/values
	Removed map[string]interface{} `json:"removed,omitempty"`

	// Modified fields/values (old -> new)
	Modified map[string]DriftModification `json:"modified,omitempty"`
}

// DriftModification represents a modified field.
type DriftModification struct {
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// RemediationResult represents the result of remediation.
type RemediationResult struct {
	// Action taken (e.g., "create", "update", "delete", "report")
	Action string `json:"action"`

	// Status of remediation
	Status DriftStatus `json:"status"`

	// Timestamp when remediation was performed
	Timestamp time.Time `json:"timestamp"`

	// Error message if remediation failed
	Error string `json:"error,omitempty"`

	// Details about what was done
	Details string `json:"details,omitempty"`
}

// DriftReport represents a complete drift detection report.
type DriftReport struct {
	// Timestamp when report was generated
	Timestamp time.Time `json:"timestamp"`

	// Spec information
	Spec SpecInfo `json:"spec"`

	// Drift summary
	Drift DriftSummary `json:"drift"`

	// Individual drift events
	Events []DriftEvent `json:"events"`
}

// SpecInfo contains information about the specification.
type SpecInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DriftSummary provides a summary of all drift.
type DriftSummary struct {
	// Whether any drift was detected
	Detected bool `json:"detected"`

	// Highest severity level detected
	Severity DriftSeverity `json:"severity"`

	// Types of drift detected
	Types []DriftType `json:"types"`

	// Count by type
	Counts DriftCounts `json:"counts"`
}

// DriftCounts provides counts of drift by type.
type DriftCounts struct {
	Total         int `json:"total"`
	Policies      int `json:"policies"`
	Compliance    int `json:"compliance"`
	Configuration int `json:"configuration"`
}

// DetectOptions contains options for drift detection.
type DetectOptions struct {
	// Watch enables continuous monitoring
	Watch bool

	// WatchInterval is the polling interval for watch mode
	WatchInterval time.Duration

	// EnabledTypes specifies which drift types to detect
	EnabledTypes []DriftType

	// OutputFormat for drift report (json, text)
	OutputFormat string

	// OutputFile to write drift report
	OutputFile string
}

// RemediateOptions contains options for drift remediation.
type RemediateOptions struct {
	// DryRun shows what would be fixed without applying changes
	DryRun bool

	// Types specifies which drift types to remediate
	Types []DriftType

	// AutoRemediate enables automatic remediation
	AutoRemediate bool

	// Force enables remediation even for risky operations
	Force bool
}

// PolicyDrift represents drift in Kyverno policies.
type PolicyDrift struct {
	// Expected policy (from spec)
	Expected runtime.Object

	// Actual policy (from cluster)
	Actual runtime.Object

	// Kind of drift
	Kind string // "missing", "modified", "extra"
}

// ComplianceDrift represents drift in compliance status.
type ComplianceDrift struct {
	// Check name
	CheckName string

	// Previous status
	PreviousStatus string

	// Current status
	CurrentStatus string

	// Violations detected
	Violations []string
}

// DriftHistory represents historical drift events.
type DriftHistory struct {
	// Events stored in chronological order
	Events []DriftEvent `json:"events"`

	// Summary statistics
	Stats DriftStats `json:"stats"`
}

// DriftStats provides statistics about drift over time.
type DriftStats struct {
	// Total drift events
	TotalEvents int `json:"total_events"`

	// Events by type
	EventsByType map[DriftType]int `json:"events_by_type"`

	// Events by severity
	EventsBySeverity map[DriftSeverity]int `json:"events_by_severity"`

	// Remediation success rate
	RemediationSuccessRate float64 `json:"remediation_success_rate"`

	// First event timestamp
	FirstEvent time.Time `json:"first_event,omitempty"`

	// Last event timestamp
	LastEvent time.Time `json:"last_event,omitempty"`
}

// MonitorConfig configures continuous drift monitoring.
type MonitorConfig struct {
	// Interval between drift checks
	Interval time.Duration

	// Types of drift to monitor
	EnabledTypes []DriftType

	// Auto-remediation settings
	AutoRemediate  bool
	RemediateTypes []DriftType

	// Alerting configuration
	Alerts *AlertConfig

	// Storage configuration
	Storage *StorageConfig
}

// AlertConfig configures drift alerting.
type AlertConfig struct {
	// Enable alerting
	Enabled bool

	// Webhook URL for alerts
	WebhookURL string

	// Minimum severity to alert on
	MinSeverity DriftSeverity

	// Rate limiting (max alerts per hour)
	RateLimit int
}

// StorageConfig configures drift history storage.
type StorageConfig struct {
	// Storage type ("memory", "file", "sqlite")
	Type string

	// File path for file/sqlite storage
	Path string

	// Retention period (how long to keep history)
	Retention time.Duration
}
