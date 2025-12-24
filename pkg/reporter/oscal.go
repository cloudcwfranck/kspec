// Package reporter provides output formatting for scan results.
package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/google/uuid"
)

// OSCALReporter outputs scan results in OSCAL (Open Security Controls Assessment Language) format.
type OSCALReporter struct {
	writer io.Writer
}

// NewOSCALReporter creates a new OSCAL reporter.
func NewOSCALReporter(w io.Writer) *OSCALReporter {
	return &OSCALReporter{writer: w}
}

// Report writes the scan results in OSCAL format to the configured writer.
func (r *OSCALReporter) Report(result *scanner.ScanResult) error {
	oscal := r.buildOSCAL(result)

	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(oscal); err != nil {
		return fmt.Errorf("failed to encode scan result as OSCAL: %w", err)
	}
	return nil
}

// buildOSCAL constructs the OSCAL document structure.
func (r *OSCALReporter) buildOSCAL(result *scanner.ScanResult) map[string]interface{} {
	// Build OSCAL Assessment Results structure
	return map[string]interface{}{
		"assessment-results": map[string]interface{}{
			"uuid":     uuid.New().String(),
			"metadata": r.buildMetadata(result),
			"results": []map[string]interface{}{
				r.buildResult(result),
			},
		},
	}
}

// buildMetadata constructs the OSCAL metadata section.
func (r *OSCALReporter) buildMetadata(result *scanner.ScanResult) map[string]interface{} {
	return map[string]interface{}{
		"title":         fmt.Sprintf("kspec Compliance Assessment - %s", result.Metadata.Spec.Name),
		"published":     result.Metadata.ScanTime,
		"last-modified": result.Metadata.ScanTime,
		"version":       result.Metadata.Spec.Version,
		"oscal-version": "1.0.4",
		"properties": []map[string]interface{}{
			{
				"name":  "kspec-version",
				"value": result.Metadata.KspecVersion,
			},
			{
				"name":  "cluster-name",
				"value": result.Metadata.Cluster.Name,
			},
			{
				"name":  "cluster-version",
				"value": result.Metadata.Cluster.Version,
			},
		},
	}
}

// buildResult constructs the OSCAL results section.
func (r *OSCALReporter) buildResult(scanResult *scanner.ScanResult) map[string]interface{} {
	return map[string]interface{}{
		"uuid":        uuid.New().String(),
		"title":       fmt.Sprintf("kspec Scan Results - %s", scanResult.Metadata.Spec.Name),
		"description": fmt.Sprintf("Compliance scan of cluster %s", scanResult.Metadata.Cluster.Name),
		"start":       scanResult.Metadata.ScanTime,
		"end":         scanResult.Metadata.ScanTime,
		"props": []map[string]interface{}{
			{
				"name":  "total-checks",
				"value": fmt.Sprintf("%d", scanResult.Summary.TotalChecks),
			},
			{
				"name":  "passed",
				"value": fmt.Sprintf("%d", scanResult.Summary.Passed),
			},
			{
				"name":  "failed",
				"value": fmt.Sprintf("%d", scanResult.Summary.Failed),
			},
			{
				"name":  "warnings",
				"value": fmt.Sprintf("%d", scanResult.Summary.Warnings),
			},
		},
		"observations": r.buildObservations(scanResult.Results),
		"findings":     r.buildFindings(scanResult.Results),
	}
}

// buildObservations constructs the observations from check results.
func (r *OSCALReporter) buildObservations(results []scanner.CheckResult) []map[string]interface{} {
	observations := make([]map[string]interface{}, 0, len(results))

	for _, result := range results {
		obs := map[string]interface{}{
			"uuid":        uuid.New().String(),
			"title":       result.Name,
			"description": result.Message,
			"collected":   time.Now().UTC().Format(time.RFC3339),
			"methods":     []string{"TEST-AUTOMATED"},
			"props": []map[string]interface{}{
				{
					"name":  "status",
					"value": string(result.Status),
				},
			},
		}

		// Add severity if present
		if result.Severity != "" {
			obs["props"] = append(obs["props"].([]map[string]interface{}), map[string]interface{}{
				"name":  "severity",
				"value": string(result.Severity),
			})
		}

		// Add evidence if present
		if len(result.Evidence) > 0 {
			evidenceJSON, _ := json.Marshal(result.Evidence)
			obs["props"] = append(obs["props"].([]map[string]interface{}), map[string]interface{}{
				"name":  "evidence",
				"value": string(evidenceJSON),
			})
		}

		observations = append(observations, obs)
	}

	return observations
}

// buildFindings constructs findings from failed checks.
func (r *OSCALReporter) buildFindings(results []scanner.CheckResult) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	for _, result := range results {
		if result.Status == scanner.StatusFail {
			finding := map[string]interface{}{
				"uuid":        uuid.New().String(),
				"title":       fmt.Sprintf("Failed Check: %s", result.Name),
				"description": result.Message,
				"props": []map[string]interface{}{
					{
						"name":  "severity",
						"value": string(result.Severity),
					},
					{
						"name":  "check-id",
						"value": result.Name,
					},
				},
			}

			// Add remediation if present
			if result.Remediation != "" {
				finding["description"] = fmt.Sprintf("%s\n\nRemediation:\n%s", result.Message, result.Remediation)
			}

			findings = append(findings, finding)
		}
	}

	return findings
}
