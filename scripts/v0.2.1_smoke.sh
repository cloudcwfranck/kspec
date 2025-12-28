#!/usr/bin/env bash
#
# v0.2.1 Smoke Test
#
# Tests that a brand-new user can:
# 1. Install the operator from the repo using kustomize
# 2. Have the operator pod become Ready
# 3. Apply the sample ClusterSpecification
# 4. See at least one ComplianceReport created
# 5. Uninstall cleanly
#
# Usage: ./scripts/v0.2.1_smoke.sh
# Requires: kind, kubectl, kustomize (or kubectl 1.14+)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CLUSTER_NAME="kspec-v021-smoke"
NAMESPACE="kspec-system"

log() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ERROR:${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] WARN:${NC} $*"
}

cleanup() {
    log "Cleaning up kind cluster ${CLUSTER_NAME}..."
    kind delete cluster --name "${CLUSTER_NAME}" 2>/dev/null || true
}

fail() {
    error "$*"
    cleanup
    echo ""
    echo -e "${RED}❌ SMOKE TEST FAILED${NC}"
    exit 1
}

pass() {
    echo ""
    echo -e "${GREEN}✅ SMOKE TEST PASSED${NC}"
    cleanup
    exit 0
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

log "Starting v0.2.1 smoke test..."

# Check prerequisites
command -v kind >/dev/null 2>&1 || fail "kind not found. Install from https://kind.sigs.k8s.io/"
command -v kubectl >/dev/null 2>&1 || fail "kubectl not found"

# Step 1: Create kind cluster
log "Creating kind cluster: ${CLUSTER_NAME}"
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    warn "Cluster ${CLUSTER_NAME} already exists, deleting..."
    kind delete cluster --name "${CLUSTER_NAME}"
fi

kind create cluster --name "${CLUSTER_NAME}" --wait 60s || fail "Failed to create kind cluster"

# Verify cluster is ready
kubectl cluster-info --context "kind-${CLUSTER_NAME}" >/dev/null || fail "Cluster not accessible"
log "Cluster created and accessible"

# Step 2: Install operator using kustomize
log "Installing operator from local config/default..."
kubectl apply -k config/default || fail "Failed to apply kustomize manifests"

# Verify namespace was created
log "Verifying namespace ${NAMESPACE} was created..."
kubectl get namespace "${NAMESPACE}" >/dev/null || fail "Namespace ${NAMESPACE} not found"
log "Namespace ${NAMESPACE} exists"

# Verify CRDs were installed
log "Verifying CRDs were installed..."
CRD_COUNT=$(kubectl get crd | grep -c "kspec.io" || true)
if [ "${CRD_COUNT}" -ne 4 ]; then
    fail "Expected 4 CRDs, found ${CRD_COUNT}"
fi
log "All 4 CRDs installed"

# Step 3: Wait for operator deployment to be Ready
log "Waiting for operator deployment to become Ready (timeout: 120s)..."
kubectl wait --for=condition=available --timeout=120s \
    deployment/kspec-operator -n "${NAMESPACE}" || fail "Operator deployment did not become Ready"

log "Operator deployment is Ready"

# Verify operator pod is running
POD_NAME=$(kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=kspec-operator -o jsonpath='{.items[0].metadata.name}')
if [ -z "${POD_NAME}" ]; then
    fail "No operator pod found"
fi
log "Operator pod: ${POD_NAME}"

# Check operator logs for webhook status
log "Checking operator logs for webhook status..."
kubectl logs -n "${NAMESPACE}" "${POD_NAME}" --tail=50 | grep -i "webhook" || true
if kubectl logs -n "${NAMESPACE}" "${POD_NAME}" --tail=100 | grep -q "Webhooks disabled"; then
    log "✓ Webhooks disabled as expected"
else
    warn "Could not confirm webhook status from logs"
fi

# Check for TLS errors (should NOT exist)
if kubectl logs -n "${NAMESPACE}" "${POD_NAME}" --tail=100 | grep -qi "tls.crt.*no such file"; then
    fail "Found TLS certificate error in operator logs (webhooks should be disabled)"
fi

# Step 4: Apply sample ClusterSpecification
log "Applying sample ClusterSpecification..."
kubectl apply -f config/samples/kspec_v1alpha1_clusterspecification.yaml || \
    fail "Failed to apply sample ClusterSpecification"

# Verify ClusterSpec was created
CLUSTERSPEC_NAME="clusterspecification-sample"
kubectl get clusterspec "${CLUSTERSPEC_NAME}" >/dev/null || \
    fail "ClusterSpecification ${CLUSTERSPEC_NAME} not found"
log "ClusterSpecification ${CLUSTERSPEC_NAME} created"

# Step 5: Wait for ComplianceReport to be created
log "Waiting for ComplianceReport to be created (timeout: 30s)..."
for i in {1..30}; do
    REPORT_COUNT=$(kubectl get compliancereport -n "${NAMESPACE}" 2>/dev/null | grep -c "${CLUSTERSPEC_NAME}" || true)
    if [ "${REPORT_COUNT}" -gt 0 ]; then
        log "✓ ComplianceReport created (count: ${REPORT_COUNT})"
        break
    fi
    if [ "$i" -eq 30 ]; then
        error "No ComplianceReport found after 30 seconds"
        kubectl get compliancereport -n "${NAMESPACE}" -o wide
        kubectl describe clusterspec "${CLUSTERSPEC_NAME}"
        fail "ComplianceReport not created within timeout"
    fi
    sleep 1
done

# Verify ClusterSpec status
log "Checking ClusterSpec status..."
PHASE=$(kubectl get clusterspec "${CLUSTERSPEC_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
if [ -z "${PHASE}" ]; then
    warn "ClusterSpec has no status.phase (may still be reconciling)"
else
    log "ClusterSpec phase: ${PHASE}"
fi

# Step 6: Test finalizer cleanup
log "Testing finalizer cleanup by deleting ClusterSpec..."
kubectl delete clusterspec "${CLUSTERSPEC_NAME}" || fail "Failed to delete ClusterSpec"

# Wait for deletion to complete
log "Waiting for ClusterSpec deletion to complete..."
for i in {1..20}; do
    if ! kubectl get clusterspec "${CLUSTERSPEC_NAME}" 2>/dev/null >/dev/null; then
        log "✓ ClusterSpec deleted"
        break
    fi
    if [ "$i" -eq 20 ]; then
        fail "ClusterSpec not deleted after 20 seconds (finalizer issue?)"
    fi
    sleep 1
done

# Verify ComplianceReports were cleaned up
sleep 2
REMAINING_REPORTS=$(kubectl get compliancereport -n "${NAMESPACE}" 2>/dev/null | grep -c "${CLUSTERSPEC_NAME}" || true)
if [ "${REMAINING_REPORTS}" -gt 0 ]; then
    warn "Found ${REMAINING_REPORTS} ComplianceReports remaining (finalizer may not have cleaned up)"
    kubectl get compliancereport -n "${NAMESPACE}" -o wide
else
    log "✓ All ComplianceReports cleaned up"
fi

# Step 7: Uninstall operator
log "Uninstalling operator..."
kubectl delete -k config/default || fail "Failed to uninstall operator"

# Verify namespace is being deleted
log "Verifying namespace cleanup..."
for i in {1..20}; do
    if ! kubectl get namespace "${NAMESPACE}" 2>/dev/null >/dev/null; then
        log "✓ Namespace ${NAMESPACE} deleted"
        break
    fi
    PHASE=$(kubectl get namespace "${NAMESPACE}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "${PHASE}" = "Terminating" ]; then
        log "Namespace ${NAMESPACE} is Terminating..."
    fi
    if [ "$i" -eq 20 ]; then
        warn "Namespace ${NAMESPACE} still exists after 20 seconds (may be stuck in Terminating)"
        break
    fi
    sleep 1
done

# All tests passed!
pass
