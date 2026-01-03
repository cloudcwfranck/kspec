#!/usr/bin/env bash
#
# Demo: Reports
# Pain: "Auditors want proof"
# Fix: Export evidence in standard formats (OSCAL, SARIF, Markdown)
#
# This script demonstrates real kspec reporting capabilities.
# Reports are generated via scan --output <format>

set -euo pipefail

SPEC_FILE="${SPEC_FILE:-specs/examples/strict.yaml}"

if [ ! -f "$SPEC_FILE" ]; then
  echo "Error: Spec file $SPEC_FILE not found"
  exit 1
fi

echo "Generating OSCAL assessment report..."
echo ""

# Generate OSCAL report
../../../kspec scan --spec "$SPEC_FILE" --output oscal > oscal-assessment.json

echo "✓ OSCAL assessment report written to oscal-assessment.json"
echo ""
echo "Report Summary:"
wc -l oscal-assessment.json | awk '{print "  Lines:", $1}'
du -h oscal-assessment.json | awk '{print "  Size:", $1}'

echo ""
echo "Generating Markdown compliance report..."
echo ""

# Generate Markdown report
../../../kspec scan --spec "$SPEC_FILE" --output markdown > COMPLIANCE.md

echo "✓ Markdown compliance report written to COMPLIANCE.md"
echo ""
echo "Report includes:"
echo "  • Executive summary with compliance score"
echo "  • Critical findings with remediation steps"
echo "  • Failed checks by severity"
echo "  • Passed checks summary"
echo "  • Cluster metadata and scan timestamp"

echo ""
echo "Preview of COMPLIANCE.md:"
echo ""
head -20 COMPLIANCE.md
