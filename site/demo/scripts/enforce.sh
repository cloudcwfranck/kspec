#!/usr/bin/env bash
#
# Demo: Enforce
# Pain: "I need guardrails, not a manual checklist"
# Fix: Automated policy generation & enforcement
#
# This script demonstrates real kspec enforce functionality.
# Requires Kyverno to be installed in the cluster.

set -euo pipefail

SPEC_FILE="${SPEC_FILE:-specs/examples/strict.yaml}"

if [ ! -f "$SPEC_FILE" ]; then
  echo "Error: Spec file $SPEC_FILE not found"
  exit 1
fi

echo "Generating policies (dry-run)..."
echo ""

# Preview policies without deploying
../../../kspec enforce --spec "$SPEC_FILE" --dry-run

echo ""
echo "Deploying policies to cluster..."
echo ""

# Deploy policies
../../../kspec enforce --spec "$SPEC_FILE"

echo ""
echo "Verifying deployed policies..."
echo ""

# List created policies
kubectl get clusterpolicies
