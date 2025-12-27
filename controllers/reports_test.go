/*
Copyright 2025 kspec contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
)

// TestNormalizeStatus ensures all scanner status values are correctly mapped to CRD enums
func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Lowercase values from scanner
		{"pass lowercase", "pass", "Pass"},
		{"fail lowercase", "fail", "Fail"},
		{"skip lowercase", "skip", "Pass"}, // Skip maps to Pass since CRD doesn't support it
		{"error lowercase", "error", "Error"},

		// Capitalized values (already correct)
		{"Pass capitalized", "Pass", "Pass"},
		{"Fail capitalized", "Fail", "Fail"},
		{"Error capitalized", "Error", "Error"},

		// Alternative forms
		{"passed alternative", "passed", "Pass"},
		{"failed alternative", "failed", "Fail"},
		{"skipped alternative", "skipped", "Pass"},

		// Unknown/invalid values should default to Error
		{"empty string", "", "Error"},
		{"unknown value", "unknown", "Error"},
		{"random value", "foobar", "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeStatus(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeStatus(%q) = %q, expected %q", tt.input, result, tt.expected)
			}

			// Verify result is a valid CRD enum value
			if result != "Pass" && result != "Fail" && result != "Error" {
				t.Errorf("normalizeStatus(%q) returned invalid CRD value: %q", tt.input, result)
			}
		})
	}
}

// TestNormalizeSeverity ensures all scanner severity values are correctly mapped to CRD enums
func TestNormalizeSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Lowercase values from scanner
		{"low lowercase", "low", "Low"},
		{"medium lowercase", "medium", "Medium"},
		{"high lowercase", "high", "High"},
		{"critical lowercase", "critical", "Critical"},

		// Capitalized values (already correct)
		{"Low capitalized", "Low", "Low"},
		{"Medium capitalized", "Medium", "Medium"},
		{"High capitalized", "High", "High"},
		{"Critical capitalized", "Critical", "Critical"},

		// Alternative forms
		{"moderate alternative", "moderate", "Medium"},
		{"Moderate capitalized", "Moderate", "Medium"},

		// Empty/unknown values should default to Low
		{"empty string", "", "Low"},
		{"unknown value", "unknown", "Low"},
		{"random value", "foobar", "Low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSeverity(%q) = %q, expected %q", tt.input, result, tt.expected)
			}

			// Verify result is a valid CRD enum value
			if result != "Low" && result != "Medium" && result != "High" && result != "Critical" {
				t.Errorf("normalizeSeverity(%q) returned invalid CRD value: %q", tt.input, result)
			}
		})
	}
}

// TestNormalizeType ensures all drift type values are correctly mapped to CRD enums
func TestNormalizeType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Lowercase values from drift package
		{"policy lowercase", "policy", "Policy"},
		{"compliance lowercase", "compliance", "Compliance"},
		{"configuration lowercase", "configuration", "Configuration"},

		// Capitalized values (already correct)
		{"Policy capitalized", "Policy", "Policy"},
		{"Compliance capitalized", "Compliance", "Compliance"},
		{"Configuration capitalized", "Configuration", "Configuration"},

		// Unknown values should default to Policy
		{"empty string", "", "Policy"},
		{"unknown value", "unknown", "Policy"},
		{"random value", "foobar", "Policy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeType(%q) = %q, expected %q", tt.input, result, tt.expected)
			}

			// Verify result is a valid CRD enum value
			if result != "Policy" && result != "Compliance" && result != "Configuration" {
				t.Errorf("normalizeType(%q) returned invalid CRD value: %q", tt.input, result)
			}
		})
	}
}

// TestNormalizeDriftKind ensures all drift kind values are correctly mapped to CRD enums
func TestNormalizeDriftKind(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Valid CRD values
		{"deleted", "deleted", "deleted"},
		{"modified", "modified", "modified"},
		{"violation", "violation", "violation"},
		{"extra", "extra", "extra"},

		// Values that need mapping
		{"missing maps to deleted", "missing", "deleted"},

		// Unknown values should default to modified
		{"empty string", "", "modified"},
		{"unknown value", "unknown", "modified"},
		{"random value", "foobar", "modified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDriftKind(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeDriftKind(%q) = %q, expected %q", tt.input, result, tt.expected)
			}

			// Verify result is a valid CRD enum value
			validValues := map[string]bool{
				"deleted":   true,
				"modified":  true,
				"violation": true,
				"extra":     true,
			}
			if !validValues[result] {
				t.Errorf("normalizeDriftKind(%q) returned invalid CRD value: %q", tt.input, result)
			}
		})
	}
}

// TestInferCategory ensures category inference works correctly
func TestInferCategory(t *testing.T) {
	tests := []struct {
		name      string
		checkName string
		expected  string
	}{
		{"standard category", "kubernetes.version", "kubernetes"},
		{"podSecurity category", "podSecurity.pss", "podSecurity"},
		{"network category", "network.policies", "network"},
		{"no dot returns whole name", "custom", "custom"},
		{"empty string", "", "unknown"},
		{"multiple dots", "foo.bar.baz", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferCategory(tt.checkName)
			if result != tt.expected {
				t.Errorf("inferCategory(%q) = %q, expected %q", tt.checkName, result, tt.expected)
			}
		})
	}
}

// TestCalculatePassRate ensures pass rate calculation is correct
func TestCalculatePassRate(t *testing.T) {
	tests := []struct {
		name     string
		summary  scanner.ScanSummary
		expected int
	}{
		{"100% pass rate", scanner.ScanSummary{TotalChecks: 10, Passed: 10, Failed: 0}, 100},
		{"50% pass rate", scanner.ScanSummary{TotalChecks: 10, Passed: 5, Failed: 5}, 50},
		{"0% pass rate", scanner.ScanSummary{TotalChecks: 10, Passed: 0, Failed: 10}, 0},
		{"zero checks", scanner.ScanSummary{TotalChecks: 0, Passed: 0, Failed: 0}, 0},
		{"rounding down", scanner.ScanSummary{TotalChecks: 3, Passed: 1, Failed: 2}, 33},
		{"rounding up", scanner.ScanSummary{TotalChecks: 3, Passed: 2, Failed: 1}, 66},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePassRate(tt.summary)
			if result != tt.expected {
				t.Errorf("calculatePassRate(%+v) = %d, expected %d", tt.summary, result, tt.expected)
			}

			// Verify pass rate is in valid range
			if result < 0 || result > 100 {
				t.Errorf("calculatePassRate(%+v) returned out-of-range value: %d", tt.summary, result)
			}
		})
	}
}
