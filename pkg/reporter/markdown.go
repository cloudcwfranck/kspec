// Package reporter provides output formatting for scan results.
package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
)

// MarkdownReporter outputs scan results in Markdown format.
type MarkdownReporter struct {
	writer io.Writer
}

// NewMarkdownReporter creates a new Markdown reporter.
func NewMarkdownReporter(w io.Writer) *MarkdownReporter {
	return &MarkdownReporter{writer: w}
}

// Report writes the scan results in Markdown format to the configured writer.
func (r *MarkdownReporter) Report(result *scanner.ScanResult) error {
	var sb strings.Builder

	// Title and metadata
	r.writeHeader(&sb, result)

	// Executive summary
	r.writeSummary(&sb, result)

	// Detailed results
	r.writeDetailedResults(&sb, result)

	// Remediation section
	r.writeRemediationSection(&sb, result)

	// Write to output
	if _, err := r.writer.Write([]byte(sb.String())); err != nil {
		return fmt.Errorf("failed to write markdown report: %w", err)
	}

	return nil
}

// writeHeader writes the report header.
func (r *MarkdownReporter) writeHeader(sb *strings.Builder, result *scanner.ScanResult) {
	sb.WriteString(fmt.Sprintf("# Kubernetes Compliance Report\n\n"))
	sb.WriteString(fmt.Sprintf("**Specification**: %s v%s\n\n",
		result.Metadata.Spec.Name, result.Metadata.Spec.Version))
	sb.WriteString(fmt.Sprintf("**Cluster**: %s (%s)\n\n",
		result.Metadata.Cluster.Name, result.Metadata.Cluster.Version))
	sb.WriteString(fmt.Sprintf("**Scan Date**: %s\n\n", result.Metadata.ScanTime))
	sb.WriteString(fmt.Sprintf("**Tool**: kspec %s\n\n", result.Metadata.KspecVersion))
	sb.WriteString("---\n\n")
}

// writeSummary writes the executive summary.
func (r *MarkdownReporter) writeSummary(sb *strings.Builder, result *scanner.ScanResult) {
	sb.WriteString("## Executive Summary\n\n")

	passRate := 0
	if result.Summary.TotalChecks > 0 {
		passRate = (result.Summary.Passed * 100) / result.Summary.TotalChecks
	}

	// Status badge
	status := "[PASS]"
	if result.Summary.Failed > 0 {
		status = "[FAIL]"
	} else if result.Summary.Warnings > 0 {
		status = "[WARN]"
	}

	sb.WriteString(fmt.Sprintf("**Overall Status**: %s\n\n", status))
	sb.WriteString(fmt.Sprintf("**Compliance Rate**: %d%% (%d/%d checks passed)\n\n",
		passRate, result.Summary.Passed, result.Summary.TotalChecks))

	// Summary table
	sb.WriteString("| Metric | Count |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Checks | %d |\n", result.Summary.TotalChecks))
	sb.WriteString(fmt.Sprintf("| Passed | %d |\n", result.Summary.Passed))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", result.Summary.Failed))
	sb.WriteString(fmt.Sprintf("| Warnings | %d |\n", result.Summary.Warnings))
	sb.WriteString(fmt.Sprintf("| Skipped | %d |\n\n", result.Summary.Skipped))
}

// writeDetailedResults writes detailed results by category.
func (r *MarkdownReporter) writeDetailedResults(sb *strings.Builder, result *scanner.ScanResult) {
	sb.WriteString("## Detailed Results\n\n")

	// Failed checks
	failures := r.filterByStatus(result.Results, scanner.StatusFail)
	if len(failures) > 0 {
		sb.WriteString("### [FAIL] Failed Checks\n\n")
		for _, check := range failures {
			r.writeCheckDetail(sb, check)
		}
	}

	// Warnings
	warnings := r.filterByStatus(result.Results, scanner.StatusWarn)
	if len(warnings) > 0 {
		sb.WriteString("### [WARN] Warnings\n\n")
		for _, check := range warnings {
			r.writeCheckDetail(sb, check)
		}
	}

	// Passed checks
	passed := r.filterByStatus(result.Results, scanner.StatusPass)
	if len(passed) > 0 {
		sb.WriteString("### [PASS] Passed Checks\n\n")
		for _, check := range passed {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", check.Name, check.Message))
		}
		sb.WriteString("\n")
	}

	// Skipped checks
	skipped := r.filterByStatus(result.Results, scanner.StatusSkip)
	if len(skipped) > 0 {
		sb.WriteString("### [SKIP] Skipped Checks\n\n")
		for _, check := range skipped {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", check.Name, check.Message))
		}
		sb.WriteString("\n")
	}
}

// writeCheckDetail writes detailed information for a check.
func (r *MarkdownReporter) writeCheckDetail(sb *strings.Builder, check scanner.CheckResult) {
	sb.WriteString(fmt.Sprintf("#### %s\n\n", check.Name))

	// Severity badge
	if check.Severity != "" {
		severityLabel := r.getSeverityLabel(check.Severity)
		sb.WriteString(fmt.Sprintf("**Severity**: %s %s\n\n",
			severityLabel, strings.ToUpper(string(check.Severity))))
	}

	// Message
	sb.WriteString(fmt.Sprintf("**Finding**: %s\n\n", check.Message))

	// Evidence
	if len(check.Evidence) > 0 {
		sb.WriteString("**Evidence**:\n\n")
		for key, value := range check.Evidence {
			sb.WriteString(fmt.Sprintf("- `%s`: %v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// Remediation
	if check.Remediation != "" {
		sb.WriteString("**Remediation**:\n\n")
		sb.WriteString("```\n")
		sb.WriteString(check.Remediation)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("---\n\n")
}

// writeRemediationSection writes the remediation summary.
func (r *MarkdownReporter) writeRemediationSection(sb *strings.Builder, result *scanner.ScanResult) {
	failures := r.filterByStatus(result.Results, scanner.StatusFail)
	if len(failures) == 0 {
		return
	}

	sb.WriteString("## Remediation Summary\n\n")
	sb.WriteString("The following actions are required to achieve compliance:\n\n")

	// Group by severity
	critical := r.filterBySeverity(failures, scanner.SeverityCritical)
	high := r.filterBySeverity(failures, scanner.SeverityHigh)
	medium := r.filterBySeverity(failures, scanner.SeverityMedium)
	low := r.filterBySeverity(failures, scanner.SeverityLow)

	if len(critical) > 0 {
		sb.WriteString("### [CRITICAL] Critical Priority\n\n")
		for i, check := range critical {
			sb.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, check.Name, check.Message))
		}
		sb.WriteString("\n")
	}

	if len(high) > 0 {
		sb.WriteString("### [HIGH] High Priority\n\n")
		for i, check := range high {
			sb.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, check.Name, check.Message))
		}
		sb.WriteString("\n")
	}

	if len(medium) > 0 {
		sb.WriteString("### [MEDIUM] Medium Priority\n\n")
		for i, check := range medium {
			sb.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, check.Name, check.Message))
		}
		sb.WriteString("\n")
	}

	if len(low) > 0 {
		sb.WriteString("### [LOW] Low Priority\n\n")
		for i, check := range low {
			sb.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, check.Name, check.Message))
		}
		sb.WriteString("\n")
	}
}

// filterByStatus filters check results by status.
func (r *MarkdownReporter) filterByStatus(results []scanner.CheckResult, status scanner.Status) []scanner.CheckResult {
	filtered := make([]scanner.CheckResult, 0)
	for _, result := range results {
		if result.Status == status {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// filterBySeverity filters check results by severity.
func (r *MarkdownReporter) filterBySeverity(results []scanner.CheckResult, severity scanner.Severity) []scanner.CheckResult {
	filtered := make([]scanner.CheckResult, 0)
	for _, result := range results {
		if result.Severity == severity {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// getSeverityLabel returns a label for a severity level.
func (r *MarkdownReporter) getSeverityLabel(severity scanner.Severity) string {
	switch severity {
	case scanner.SeverityCritical:
		return "[CRITICAL]"
	case scanner.SeverityHigh:
		return "[HIGH]"
	case scanner.SeverityMedium:
		return "[MEDIUM]"
	case scanner.SeverityLow:
		return "[LOW]"
	default:
		return "[INFO]"
	}
}
