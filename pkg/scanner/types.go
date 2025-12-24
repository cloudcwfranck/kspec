// Package scanner provides the cluster scanning functionality.
package scanner

import (
	"context"

	"github.com/kspec/kspec/pkg/spec"
	"k8s.io/client-go/kubernetes"
)

// Check represents a compliance check that can be run against a cluster.
type Check interface {
	// Name returns the unique identifier for this check (e.g., "kubernetes.version")
	Name() string

	// Run executes the check against the cluster
	Run(ctx context.Context, client kubernetes.Interface, spec *spec.ClusterSpecification) (*CheckResult, error)
}

// CheckResult represents the result of running a compliance check.
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Severity    Severity               `json:"severity,omitempty"`
	Message     string                 `json:"message"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
}

// Status represents the status of a check.
type Status string

const (
	// StatusPass indicates the check passed
	StatusPass Status = "pass"
	// StatusFail indicates the check failed
	StatusFail Status = "fail"
	// StatusWarn indicates the check found a warning
	StatusWarn Status = "warn"
	// StatusSkip indicates the check was skipped
	StatusSkip Status = "skip"
)

// Severity represents the severity of a check failure.
type Severity string

const (
	// SeverityCritical indicates a critical failure
	SeverityCritical Severity = "critical"
	// SeverityHigh indicates a high-severity failure
	SeverityHigh Severity = "high"
	// SeverityMedium indicates a medium-severity failure
	SeverityMedium Severity = "medium"
	// SeverityLow indicates a low-severity failure
	SeverityLow Severity = "low"
)

// ScanResult represents the aggregated results of all checks.
type ScanResult struct {
	Metadata ScanMetadata  `json:"metadata"`
	Summary  ScanSummary   `json:"summary"`
	Results  []CheckResult `json:"results"`
}

// ScanMetadata contains metadata about the scan.
type ScanMetadata struct {
	KspecVersion string        `json:"kspec_version"`
	ScanTime     string        `json:"scan_time"`
	Cluster      ClusterInfo   `json:"cluster"`
	Spec         SpecInfo      `json:"spec"`
}

// ClusterInfo contains information about the scanned cluster.
type ClusterInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	UID     string `json:"uid"`
}

// SpecInfo contains information about the specification used.
type SpecInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ScanSummary contains summary statistics of the scan.
type ScanSummary struct {
	TotalChecks int `json:"total_checks"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Warnings    int `json:"warnings"`
	Skipped     int `json:"skipped"`
}
