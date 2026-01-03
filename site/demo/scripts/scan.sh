#!/usr/bin/env bash
#
# Demo: Scan
# Pain: "I don't know what's wrong"
# Fix: Automated security & compliance scanning
#
# This script demonstrates real kspec scan functionality.
# Expected to run on kind cluster with kspec installed.

set -euo pipefail

# Ensure we're in the right context
echo "Current cluster context:"
kubectl config current-context

# Apply a test spec if it doesn't exist
SPEC_FILE="${SPEC_FILE:-specs/examples/strict.yaml}"

if [ ! -f "$SPEC_FILE" ]; then
  echo "Error: Spec file $SPEC_FILE not found"
  echo "Run this script from the kspec repository root"
  exit 1
fi

echo ""
echo "Running compliance scan..."
echo ""

# Run scan with text output (default)
./kspec scan --spec "$SPEC_FILE"

echo ""
echo "Generating SARIF report for GitHub Code Scanning..."
echo ""

# Generate SARIF report
./kspec scan --spec "$SPEC_FILE" --output sarif > compliance-report.sarif

echo "✓ SARIF report written to compliance-report.sarif"
echo ""
echo "Upload to GitHub Code Scanning:"
echo "  gh api repos/owner/repo/code-scanning/sarifs -F sarif=@compliance-report.sarif"
