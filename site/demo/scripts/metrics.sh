#!/usr/bin/env bash
#
# Demo: Metrics
# Pain: "I need to monitor this continuously"
# Fix: Prometheus metrics exposed by kspec-operator
#
# This script demonstrates real kspec metrics collection.
# Requires kspec-operator to be deployed.

set -euo pipefail

# Check if kspec-operator is running
if ! kubectl get deployment -n kspec-system kspec-operator &>/dev/null; then
  echo "Warning: kspec-operator not found in kspec-system namespace"
  echo "Metrics demo requires kspec-operator to be deployed"
  exit 1
fi

echo "Setting up port-forward to kspec-operator metrics endpoint..."
echo ""

# Port-forward in background
kubectl -n kspec-system port-forward deploy/kspec-operator 8080:8080 &
PF_PID=$!

# Wait for port-forward to be ready
sleep 3

# Ensure cleanup on exit
trap "kill $PF_PID 2>/dev/null || true" EXIT

echo "Fetching compliance metrics..."
echo ""

# Query compliance metrics
curl -s http://localhost:8080/metrics | grep kspec_compliance || echo "No compliance metrics found"

echo ""
echo ""
echo "Fetching drift metrics..."
echo ""

# Query drift metrics
curl -s http://localhost:8080/metrics | grep kspec_drift || echo "No drift metrics found"

echo ""
echo ""
echo "All kspec metrics available at: http://localhost:8080/metrics"
