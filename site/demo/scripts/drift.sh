#!/usr/bin/env bash
#
# Demo: Drift Detection
# Pain: "We were compliant last week... now we aren't"
# Fix: Continuous drift detection & remediation
#
# This script demonstrates real kspec drift detection.

set -euo pipefail

SPEC_FILE="${SPEC_FILE:-specs/examples/strict.yaml}"

if [ ! -f "$SPEC_FILE" ]; then
  echo "Error: Spec file $SPEC_FILE not found"
  exit 1
fi

echo "Detecting configuration drift..."
echo ""

# Detect drift
../../../kspec drift detect --spec "$SPEC_FILE" || true

echo ""
echo "Previewing remediation (dry-run)..."
echo ""

# Preview remediation
../../../kspec drift remediate --spec "$SPEC_FILE" --dry-run || true

echo ""
echo "Applying remediation..."
echo ""

# Apply remediation
../../../kspec drift remediate --spec "$SPEC_FILE" || true
