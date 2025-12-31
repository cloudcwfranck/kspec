#!/usr/bin/env bash
# E2E smoke test for kspec v0.3.1
# Tests: kind cluster, cert-manager, Kyverno (Helm), kspec operator, monitor→enforce mode, webhook validation

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CLUSTER_NAME="kspec-e2e-v0.3.1"
KYVERNO_NAMESPACE="kyverno"
KSPEC_NAMESPACE="kspec-system"
TIMEOUT=300s

# Cleanup function
cleanup() {
    local exit_code=$?
    echo ""
    echo -e "${YELLOW}=== Cleanup ===${NC}"

    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        echo "Deleting kind cluster: ${CLUSTER_NAME}"
        kind delete cluster --name "${CLUSTER_NAME}" || true
    fi

    if [ ${exit_code} -eq 0 ]; then
        echo -e "${GREEN}✓ E2E test completed successfully${NC}"
    else
        echo -e "${RED}✗ E2E test failed with exit code ${exit_code}${NC}"
    fi

    exit ${exit_code}
}

trap cleanup EXIT

log_step() {
    echo ""
    echo -e "${GREEN}=== $1 ===${NC}"
}

log_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

log_info() {
    echo -e "${YELLOW}INFO: $1${NC}"
}

wait_for_pods() {
    local namespace=$1
    local label=$2
    local timeout=${3:-300s}

    log_info "Waiting for pods in namespace ${namespace} with label ${label} (timeout: ${timeout})"
    if ! kubectl wait --for=condition=ready pod -l "${label}" -n "${namespace}" --timeout="${timeout}"; then
        log_error "Pods not ready in time"
        kubectl get pods -n "${namespace}" -l "${label}"
        kubectl describe pods -n "${namespace}" -l "${label}"
        return 1
    fi
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found. Please install it first."
        return 1
    fi
}

# Preflight checks
log_step "Preflight Checks"
check_command kind
check_command kubectl
check_command helm
check_command docker

# Clean up any existing cluster
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    log_info "Deleting existing cluster: ${CLUSTER_NAME}"
    kind delete cluster --name "${CLUSTER_NAME}"
fi

# Create kind cluster
log_step "Creating kind cluster: ${CLUSTER_NAME}"
cat <<EOF | kind create cluster --name "${CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 30080
    protocol: TCP
EOF

# Verify cluster is ready
log_info "Verifying cluster is ready"
kubectl cluster-info --context "kind-${CLUSTER_NAME}"
kubectl get nodes

# Install cert-manager
log_step "Installing cert-manager"
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

log_info "Waiting for cert-manager to be ready"
wait_for_pods "cert-manager" "app.kubernetes.io/instance=cert-manager" "300s"

# Verify cert-manager webhooks are ready
log_info "Verifying cert-manager webhook"
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=120s

# Install Kyverno via Helm
log_step "Installing Kyverno via Helm"
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update

helm install kyverno kyverno/kyverno \
  --namespace "${KYVERNO_NAMESPACE}" \
  --create-namespace \
  --set admissionController.replicas=1 \
  --set backgroundController.replicas=1 \
  --set cleanupController.replicas=1 \
  --set reportsController.replicas=1 \
  --wait \
  --timeout=5m

log_info "Verifying Kyverno is ready"
wait_for_pods "${KYVERNO_NAMESPACE}" "app.kubernetes.io/instance=kyverno" "300s"

# Verify Kyverno webhook
log_info "Verifying Kyverno webhooks"
kubectl get validatingwebhookconfigurations | grep kyverno
kubectl get mutatingwebhookconfigurations | grep kyverno

# Build and load kspec operator image (optional - can use existing image)
log_step "Preparing kspec operator deployment"
log_info "Using image from manifests (ensure it's built and loaded into kind)"

# For local testing, we'll use the manifests as-is
# In CI, you would do: make docker-build && kind load docker-image kspec-operator:latest --name ${CLUSTER_NAME}

# Install kspec CRDs
log_step "Installing kspec CRDs"
kubectl apply -f config/crd/bases/

log_info "Verifying CRDs are installed"
kubectl get crd | grep kspec.io || {
    log_error "kspec CRDs not found"
    exit 1
}

# Install kspec operator
log_step "Installing kspec operator"
# Create namespace
kubectl create namespace "${KSPEC_NAMESPACE}" || true

# Deploy operator using kustomize
kubectl apply -k config/default

log_info "Waiting for kspec operator to be ready"
# Note: Leader election is disabled in E2E tests (as per release notes)
# We need to patch the deployment to disable it
kubectl patch deployment kspec-operator-controller-manager \
  -n "${KSPEC_NAMESPACE}" \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--leader-elect=false"}]' || true

wait_for_pods "${KSPEC_NAMESPACE}" "control-plane=controller-manager" "300s"

# Verify operator logs
log_info "Checking operator logs (last 20 lines)"
kubectl logs -n "${KSPEC_NAMESPACE}" -l control-plane=controller-manager --tail=20

# Create ClusterTarget for testing
log_step "Creating test ClusterTarget (in-cluster)"
cat <<EOF | kubectl apply -f -
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
metadata:
  name: local-cluster
  namespace: ${KSPEC_NAMESPACE}
spec:
  inCluster: true
  platform: kind
  version: "1.27.0"
EOF

# Wait for ClusterTarget to be ready
log_info "Waiting for ClusterTarget to be ready"
sleep 5
kubectl get clustertarget -n "${KSPEC_NAMESPACE}" local-cluster -o yaml

# Test 1: Monitor mode (enforcement disabled)
log_step "Test 1: ClusterSpecification in MONITOR mode"
cat <<EOF | kubectl apply -f -
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: test-spec-monitor
  namespace: ${KSPEC_NAMESPACE}
spec:
  targetClusterRef:
    name: local-cluster
  enforcementMode: monitor
  policies:
    - id: "pod-security-standards"
      title: "Pod Security Standards"
      description: "Enforce baseline pod security"
      severity: high
      checks:
        - id: "require-run-as-non-root"
          title: "Require runAsNonRoot"
          kyvernoPolicy: |
            apiVersion: kyverno.io/v1
            kind: ClusterPolicy
            metadata:
              name: require-run-as-non-root
            spec:
              validationFailureAction: audit
              background: true
              rules:
              - name: check-runAsNonRoot
                match:
                  any:
                  - resources:
                      kinds:
                      - Pod
                validate:
                  message: "Containers must run as non-root user"
                  pattern:
                    spec:
                      securityContext:
                        runAsNonRoot: true
EOF

log_info "Waiting for reconciliation (30s)"
sleep 30

log_info "Verifying Kyverno policy was created in AUDIT mode"
kubectl get clusterpolicy require-run-as-non-root -o yaml | grep -q "validationFailureAction: audit" || {
    log_error "Policy not in audit mode"
    kubectl get clusterpolicy require-run-as-non-root -o yaml
    exit 1
}

log_info "✓ Monitor mode: Policy created in audit mode"

# Test non-compliant pod in monitor mode (should be allowed)
log_info "Testing non-compliant pod in monitor mode (should succeed)"
cat <<EOF | kubectl apply -f - || {
    log_error "Pod creation failed in monitor mode (should have succeeded)"
    exit 1
}
apiVersion: v1
kind: Pod
metadata:
  name: test-pod-monitor
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    # Missing runAsNonRoot - policy violation
EOF

log_info "✓ Monitor mode: Non-compliant pod created successfully (audit mode)"

# Clean up test pod
kubectl delete pod test-pod-monitor -n default --wait=false || true

# Test 2: Enforce mode
log_step "Test 2: Switching to ENFORCE mode"
kubectl patch clusterspecification test-spec-monitor \
  -n "${KSPEC_NAMESPACE}" \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/enforcementMode", "value": "enforce"}]'

log_info "Waiting for reconciliation (30s)"
sleep 30

log_info "Verifying Kyverno policy is now in ENFORCE mode"
kubectl get clusterpolicy require-run-as-non-root -o yaml | grep -q "validationFailureAction: enforce" || {
    log_error "Policy not in enforce mode"
    kubectl get clusterpolicy require-run-as-non-root -o yaml
    exit 1
}

log_info "✓ Enforce mode: Policy updated to enforce mode"

# Test non-compliant pod in enforce mode (should be denied)
log_info "Testing non-compliant pod in enforce mode (should be denied)"
if kubectl apply -f - <<EOF 2>&1 | grep -q "denied\|blocked\|violation"; then
apiVersion: v1
kind: Pod
metadata:
  name: test-pod-enforce
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    # Missing runAsNonRoot - should be denied
EOF
    log_info "✓ Enforce mode: Non-compliant pod correctly denied by webhook"
else
    log_error "Non-compliant pod was not denied (webhook not working)"
    kubectl get clusterpolicy require-run-as-non-root -o yaml
    exit 1
fi

# Test compliant pod in enforce mode (should succeed)
log_info "Testing compliant pod in enforce mode (should succeed)"
cat <<EOF | kubectl apply -f - || {
    log_error "Compliant pod creation failed"
    exit 1
}
apiVersion: v1
kind: Pod
metadata:
  name: test-pod-compliant
  namespace: default
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
  containers:
  - name: nginx
    image: nginx:1.21
EOF

log_info "✓ Enforce mode: Compliant pod created successfully"

# Clean up test pod
kubectl delete pod test-pod-compliant -n default || true

# Verify ComplianceReport exists
log_step "Verifying ComplianceReport"
kubectl get compliancereport -n "${KSPEC_NAMESPACE}" || log_info "No ComplianceReports yet (may take time)"

# Verify DriftReport exists
log_step "Verifying DriftReport"
kubectl get driftreport -n "${KSPEC_NAMESPACE}" || log_info "No DriftReports yet (may take time)"

# Check operator metrics endpoint
log_step "Verifying metrics endpoint"
kubectl get pods -n "${KSPEC_NAMESPACE}" -l control-plane=controller-manager -o name | head -1 | xargs -I {} kubectl port-forward -n "${KSPEC_NAMESPACE}" {} 8080:8080 &
PORT_FORWARD_PID=$!
sleep 3

if curl -s http://localhost:8080/metrics | grep -q "kspec_"; then
    log_info "✓ Metrics endpoint accessible"
else
    log_info "⚠ Metrics endpoint not responding (may not be exposed on 8080)"
fi

kill ${PORT_FORWARD_PID} 2>/dev/null || true

# Final summary
log_step "E2E Test Summary"
echo -e "${GREEN}✓ Kind cluster created${NC}"
echo -e "${GREEN}✓ cert-manager installed and ready${NC}"
echo -e "${GREEN}✓ Kyverno installed via Helm${NC}"
echo -e "${GREEN}✓ kspec operator deployed${NC}"
echo -e "${GREEN}✓ Monitor mode: Policies in audit mode${NC}"
echo -e "${GREEN}✓ Monitor mode: Non-compliant pods allowed${NC}"
echo -e "${GREEN}✓ Enforce mode: Policies in enforce mode${NC}"
echo -e "${GREEN}✓ Enforce mode: Non-compliant pods denied${NC}"
echo -e "${GREEN}✓ Enforce mode: Compliant pods allowed${NC}"

log_step "All E2E tests passed!"
exit 0
