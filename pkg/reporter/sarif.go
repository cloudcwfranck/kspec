// Package reporter provides output formatting for scan results.
package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
)

// SARIFReporter outputs scan results in SARIF (Static Analysis Results Interchange Format) format.
type SARIFReporter struct {
	writer io.Writer
}

// NewSARIFReporter creates a new SARIF reporter.
func NewSARIFReporter(w io.Writer) *SARIFReporter {
	return &SARIFReporter{writer: w}
}

// Report writes the scan results in SARIF format to the configured writer.
func (r *SARIFReporter) Report(result *scanner.ScanResult) error {
	sarif := r.buildSARIF(result)

	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(sarif); err != nil {
		return fmt.Errorf("failed to encode scan result as SARIF: %w", err)
	}
	return nil
}

// buildSARIF constructs the SARIF document structure.
func (r *SARIFReporter) buildSARIF(result *scanner.ScanResult) map[string]interface{} {
	return map[string]interface{}{
		"version": "2.1.0",
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs": []map[string]interface{}{
			r.buildRun(result),
		},
	}
}

// buildRun constructs a SARIF run.
func (r *SARIFReporter) buildRun(result *scanner.ScanResult) map[string]interface{} {
	return map[string]interface{}{
		"tool": map[string]interface{}{
			"driver": map[string]interface{}{
				"name":           "kspec",
				"version":        result.Metadata.KspecVersion,
				"informationUri": "https://github.com/cloudcwfranck/kspec",
				"rules":          r.buildRules(result.Results),
			},
		},
		"results": r.buildResults(result.Results),
		"properties": map[string]interface{}{
			"cluster-name":    result.Metadata.Cluster.Name,
			"cluster-version": result.Metadata.Cluster.Version,
			"spec-name":       result.Metadata.Spec.Name,
			"spec-version":    result.Metadata.Spec.Version,
			"scan-time":       result.Metadata.ScanTime,
		},
	}
}

// buildRules constructs SARIF rules from check results.
func (r *SARIFReporter) buildRules(results []scanner.CheckResult) []map[string]interface{} {
	rulesMap := make(map[string]map[string]interface{})

	for _, result := range results {
		if _, exists := rulesMap[result.Name]; !exists {
			rule := map[string]interface{}{
				"id": result.Name,
				"shortDescription": map[string]interface{}{
					"text": result.Name,
				},
				"fullDescription": map[string]interface{}{
					"text": r.getRuleDescription(result.Name),
				},
				"defaultConfiguration": map[string]interface{}{
					"level": r.mapSeverityToLevel(result.Severity),
				},
				"help": map[string]interface{}{
					"text": result.Message,
				},
			}

			// Add remediation if available
			if result.Remediation != "" {
				rule["help"].(map[string]interface{})["text"] = fmt.Sprintf("%s\n\nRemediation:\n%s",
					result.Message, result.Remediation)
			}

			rulesMap[result.Name] = rule
		}
	}

	// Convert map to slice
	rules := make([]map[string]interface{}, 0, len(rulesMap))
	for _, rule := range rulesMap {
		rules = append(rules, rule)
	}

	return rules
}

// buildResults constructs SARIF results from check results.
func (r *SARIFReporter) buildResults(results []scanner.CheckResult) []map[string]interface{} {
	sarifResults := make([]map[string]interface{}, 0)

	for _, result := range results {
		// Only report failures and warnings in SARIF
		if result.Status != scanner.StatusFail && result.Status != scanner.StatusWarn {
			continue
		}

		sarifResult := map[string]interface{}{
			"ruleId": result.Name,
			"level":  r.mapStatusToLevel(result.Status, result.Severity),
			"message": map[string]interface{}{
				"text": result.Message,
			},
			"locations": []map[string]interface{}{
				{
					"physicalLocation": map[string]interface{}{
						"artifactLocation": map[string]interface{}{
							"uri": fmt.Sprintf("cluster://%s", result.Name),
						},
					},
				},
			},
		}

		// Add evidence as properties
		if len(result.Evidence) > 0 {
			sarifResult["properties"] = result.Evidence
		}

		sarifResults = append(sarifResults, sarifResult)
	}

	return sarifResults
}

// getRuleDescription returns a description for a given check rule.
func (r *SARIFReporter) getRuleDescription(ruleName string) string {
	descriptions := map[string]string{
		"kubernetes.version":    "Validates Kubernetes cluster version is within specified range",
		"podsecurity.standards": "Validates Pod Security Standards labels on namespaces",
		"network.policies":      "Validates network policy requirements",
	}

	if desc, exists := descriptions[ruleName]; exists {
		return desc
	}
	return fmt.Sprintf("Validates %s compliance requirement", ruleName)
}

// mapSeverityToLevel maps kspec severity to SARIF level.
func (r *SARIFReporter) mapSeverityToLevel(severity scanner.Severity) string {
	switch severity {
	case scanner.SeverityCritical:
		return "error"
	case scanner.SeverityHigh:
		return "error"
	case scanner.SeverityMedium:
		return "warning"
	case scanner.SeverityLow:
		return "note"
	default:
		return "warning"
	}
}

// mapStatusToLevel maps kspec status and severity to SARIF level.
func (r *SARIFReporter) mapStatusToLevel(status scanner.Status, severity scanner.Severity) string {
	if status == scanner.StatusFail {
		return r.mapSeverityToLevel(severity)
	}
	if status == scanner.StatusWarn {
		return "warning"
	}
	return "note"
}
