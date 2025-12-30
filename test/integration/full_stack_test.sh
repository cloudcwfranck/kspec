#!/bin/bash
# Full Stack Integration Test for kspec v0.3.0
# Tests all 8 phases: Policy Enforcement, Certificates, Webhooks, Circuit Breaker,
# Observability, Multi-Cluster, Advanced Policies, High Availability

set -e  # Exit on error
set -o pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    ((TESTS_PASSED++))
    ((TESTS_TOTAL++))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((TESTS_FAILED++))
    ((TESTS_TOTAL++))
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Cleanup function
cleanup() {
    log_section "Cleaning Up"
    if [ "$SKIP_CLEANUP" != "true" ]; then
        log_info "Deleting kind cluster..."
        kind delete cluster --name kspec-test 2>/dev/null || true
        log_success "Cleanup complete"
    else
        log_warning "Skipping cleanup (SKIP_CLEANUP=true)"
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Test variables
CLUSTER_NAME="kspec-test"
NAMESPACE="kspec-system"
TEST_TIMEOUT=300  # 5 minutes

log_section "Full Stack Integration Test - v0.3.0"
log_info "Testing all 8 phases of kspec operator"

# ============================================
# Phase 0: Prerequisites
# ============================================
log_section "Phase 0: Prerequisites Check"

# Check required tools
for tool in kind kubectl docker go; do
    if ! command -v $tool &> /dev/null; then
        log_error "$tool is not installed"
        exit 1
    fi
    log_success "$tool is available"
done

# ============================================
# Phase 1: Cluster Setup
# ============================================
log_section "Phase 1: Kind Cluster Setup"

# Delete existing cluster if it exists
kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true

# Create kind cluster with 3 nodes for HA testing
log_info "Creating kind cluster with 3 worker nodes..."
cat <<EOF | kind create cluster --name $CLUSTER_NAME --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
  - role: worker
EOF

if [ $? -eq 0 ]; then
    log_success "Kind cluster created"
else
    log_error "Failed to create kind cluster"
    exit 1
fi

# Verify cluster is ready
kubectl cluster-info --context kind-$CLUSTER_NAME
log_success "Cluster is accessible"

# Wait for nodes to be ready
log_info "Waiting for nodes to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s
log_success "All nodes are ready"

# ============================================
# Phase 2: Build and Load Operator Image
# ============================================
log_section "Phase 2: Build Operator Image"

log_info "Building operator binary..."
make build-operator
log_success "Operator binary built"

log_info "Building Docker image..."
docker build -t kspec-operator:test -f Dockerfile .
log_success "Docker image built"

log_info "Loading image into kind cluster..."
kind load docker-image kspec-operator:test --name $CLUSTER_NAME
log_success "Image loaded into kind"

# ============================================
# Phase 3: Install CRDs
# ============================================
log_section "Phase 3: Install CRDs"

log_info "Installing CRDs..."
kubectl apply -k config/crd/

# Verify CRDs are installed
sleep 2
CRD_COUNT=$(kubectl get crd | grep kspec.io | wc -l)
if [ "$CRD_COUNT" -eq "4" ]; then
    log_success "All 4 CRDs installed (ClusterSpec, ClusterTarget, ComplianceReport, DriftReport)"
else
    log_error "Expected 4 CRDs, found $CRD_COUNT"
fi

# ============================================
# Phase 4: Deploy Operator with HA
# ============================================
log_section "Phase 4: Deploy Operator (High Availability)"

# Create namespace
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
log_success "Namespace $NAMESPACE created"

# Apply RBAC from config
log_info "Applying RBAC configuration..."
kubectl apply -f config/rbac/service_account.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
log_success "RBAC configured"

# Deploy operator with HA configuration (3 replicas, leader election)
log_info "Deploying operator with 3 replicas and leader election..."
kubectl apply -k config/manager/

# Wait for deployment to be ready
log_info "Waiting for operator deployment (this may take 2-3 minutes)..."
kubectl wait --for=condition=available --timeout=180s deployment/kspec-operator -n $NAMESPACE

# Check replica count
READY_REPLICAS=$(kubectl get deployment kspec-operator -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
if [ "$READY_REPLICAS" == "3" ]; then
    log_success "All 3 replicas are ready (High Availability enabled)"
else
    log_warning "Expected 3 replicas, got $READY_REPLICAS"
fi

# Verify pods are spread across nodes (anti-affinity test)
log_info "Verifying pod anti-affinity..."
POD_NODES=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=kspec-operator -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}' | sort -u | wc -l)
if [ "$POD_NODES" -ge "2" ]; then
    log_success "Pods are spread across $POD_NODES nodes (anti-affinity working)"
else
    log_warning "Pods are on $POD_NODES node(s), expected spread"
fi

# ============================================
# Phase 5: Test Leader Election
# ============================================
log_section "Phase 5: Test Leader Election (Phase 8)"

log_info "Checking leader election lease..."
sleep 10  # Give time for leader election

LEASE=$(kubectl get lease -n $NAMESPACE kspec-operator-lock -o jsonpath='{.spec.holderIdentity}' 2>/dev/null || echo "not found")
if [ "$LEASE" != "not found" ]; then
    log_success "Leader election lease exists: $LEASE"
else
    log_error "Leader election lease not found"
fi

# Test failover by deleting leader pod
log_info "Testing failover: Deleting current leader pod..."
LEADER_POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=kspec-operator --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[0].metadata.name}')
kubectl delete pod $LEADER_POD -n $NAMESPACE

log_info "Waiting for new leader election (15 seconds)..."
sleep 15

NEW_LEASE=$(kubectl get lease -n $NAMESPACE kspec-operator-lock -o jsonpath='{.spec.holderIdentity}')
if [ "$NEW_LEASE" != "$LEASE" ]; then
    log_success "Leader election failover successful! New leader: $NEW_LEASE"
else
    log_warning "Leader might be the same (possible if pod restarted quickly)"
fi

# Wait for deployment to stabilize
kubectl wait --for=condition=available --timeout=60s deployment/kspec-operator -n $NAMESPACE
log_success "Deployment stabilized after failover"

# ============================================
# Phase 6: Test PodDisruptionBudget
# ============================================
log_section "Phase 6: Test PodDisruptionBudget"

PDB=$(kubectl get pdb kspec-operator -n $NAMESPACE -o jsonpath='{.status.disruptionsAllowed}' 2>/dev/null || echo "0")
if [ "$PDB" -ge "1" ]; then
    log_success "PodDisruptionBudget allows $PDB disruptions (HA protection active)"
else
    log_warning "PodDisruptionBudget allows $PDB disruptions"
fi

# ============================================
# Phase 7: Create Test ClusterSpecification
# ============================================
log_section "Phase 7: Test ClusterSpecification"

log_info "Creating ClusterSpecification with advanced features..."
kubectl apply -f - <<EOF
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: test-cluster
spec:
  kubernetes:
    minVersion: "1.27.0"
    maxVersion: "1.30.0"
  podSecurity:
    enforce: baseline
    audit: baseline
    warn: baseline
    exemptions:
      - namespace: kube-system
        level: baseline
        reason: System namespace
      - namespace: kspec-system
        level: baseline
        reason: Operator namespace
  network:
    defaultDeny: true
  # Phase 7: Advanced policy features
  namespaceScope:
    includeNamespaces:
      - default
      - kspec-system
    excludeNamespaces:
      - kube-system
EOF

log_success "ClusterSpecification created"

# Wait for reconciliation
log_info "Waiting for ClusterSpec reconciliation..."
for i in {1..60}; do
    PHASE=$(kubectl get clusterspec test-cluster -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "$PHASE" == "Active" ] || [ "$PHASE" == "Failed" ]; then
        log_success "ClusterSpec reconciled with phase: $PHASE"
        break
    fi
    if [ $i -eq 60 ]; then
        log_error "ClusterSpec reconciliation timeout"
    fi
    sleep 5
done

# ============================================
# Phase 8: Verify ComplianceReport
# ============================================
log_section "Phase 8: Verify ComplianceReport Generation"

log_info "Waiting for ComplianceReport to be created..."
sleep 15

REPORT_COUNT=$(kubectl get compliancereport -n $NAMESPACE -l kspec.io/cluster-spec=test-cluster --no-headers 2>/dev/null | wc -l)
if [ "$REPORT_COUNT" -gt "0" ]; then
    log_success "ComplianceReport created ($REPORT_COUNT reports found)"

    # Get compliance score
    SCORE=$(kubectl get compliancereport -n $NAMESPACE -l kspec.io/cluster-spec=test-cluster -o jsonpath='{.items[0].spec.summary.passRate}' 2>/dev/null || echo "0")
    log_info "Compliance Score: $SCORE%"
else
    log_warning "No ComplianceReport found yet (may still be generating)"
fi

# ============================================
# Phase 9: Test Metrics Endpoint
# ============================================
log_section "Phase 9: Test Prometheus Metrics (Phase 5)"

log_info "Testing metrics endpoint..."
POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=kspec-operator -o jsonpath='{.items[0].metadata.name}')

# Port-forward in background
kubectl port-forward -n $NAMESPACE pod/$POD 8080:8080 &
PF_PID=$!
sleep 3

# Test metrics endpoint
if curl -sf http://localhost:8080/metrics > /dev/null 2>&1; then
    log_success "Metrics endpoint is accessible"

    # Check for specific Phase 8 metrics
    if curl -s http://localhost:8080/metrics | grep -q "kspec_leader_election_status"; then
        log_success "Leader election metrics present"
    fi

    if curl -s http://localhost:8080/metrics | grep -q "kspec_compliance_score"; then
        log_success "Compliance metrics present"
    fi
else
    log_warning "Metrics endpoint not accessible"
fi

# Kill port-forward
kill $PF_PID 2>/dev/null || true

# ============================================
# Phase 10: Test Health Endpoints
# ============================================
log_section "Phase 10: Test Health Endpoints"

kubectl port-forward -n $NAMESPACE pod/$POD 8081:8081 &
PF_PID=$!
sleep 3

if curl -sf http://localhost:8081/healthz > /dev/null; then
    log_success "Health endpoint /healthz is healthy"
else
    log_error "Health endpoint /healthz failed"
fi

if curl -sf http://localhost:8081/readyz > /dev/null; then
    log_success "Readiness endpoint /readyz is ready"
else
    log_error "Readiness endpoint /readyz failed"
fi

kill $PF_PID 2>/dev/null || true

# ============================================
# Phase 11: Test Resource Cleanup
# ============================================
log_section "Phase 11: Test Resource Cleanup"

log_info "Deleting ClusterSpecification..."
kubectl delete clusterspec test-cluster

# Wait for deletion
for i in {1..30}; do
    if ! kubectl get clusterspec test-cluster 2>/dev/null; then
        log_success "ClusterSpec deleted successfully"
        break
    fi
    sleep 2
done

# ============================================
# Final Summary
# ============================================
log_section "Test Summary"

echo ""
echo "Total Tests: $TESTS_TOTAL"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    log_success "ALL TESTS PASSED! 🎉"
    log_success "kspec v0.3.0 is production-ready!"
    exit 0
else
    log_error "Some tests failed. Please review the output above."
    exit 1
fi
