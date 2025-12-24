// Package reporter provides output formatting for scan results.
package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
)

// JSONReporter outputs scan results as JSON.
type JSONReporter struct {
	writer io.Writer
}

// NewJSONReporter creates a new JSON reporter.
func NewJSONReporter(w io.Writer) *JSONReporter {
	return &JSONReporter{writer: w}
}

// Report writes the scan results as JSON to the configured writer.
func (r *JSONReporter) Report(result *scanner.ScanResult) error {
	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode scan result as JSON: %w", err)
	}
	return nil
}
