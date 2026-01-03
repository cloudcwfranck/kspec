#!/usr/bin/env bash
# Validate Kyverno Installation and TLS Setup
# This script verifies that Kyverno is properly installed with TLS certificates

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    PASSED=$((PASSED + 1))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    FAILED=$((FAILED + 1))
}

warn() {
    echo -e "${YELLOW}⚠ WARN${NC}: $1"
}

info() {
    echo "ℹ INFO: $1"
}

# Sanitize a value to a safe integer (strip whitespace, default to 0)
sanitize_int() {
    local val="$1"
    val="$(echo "$val" | tr -d '[:space:]')"
    if [[ "$val" =~ ^[0-9]+$ ]]; then
        echo "$val"
    else
        echo "0"
    fi
}

# Check if jq is installed
check_jq() {
    if ! command -v jq &>/dev/null; then
        fail "jq is not installed"
        echo ""
        echo "jq is required for webhook validation. Please install it:"
        echo "  macOS:   brew install jq"
        echo "  Ubuntu:  apt-get install jq"
        echo "  Alpine:  apk add jq"
        exit 1
    fi
}

# Wait for webhooks to be configured
# Usage: wait_for_webhooks <kind> <min_count> <timeout_seconds>
wait_for_webhooks() {
    local kind="$1"
    local min_count="$2"
    local timeout="$3"
    local interval=5
    local elapsed=0

    info "Waiting for $kind webhooks (min: $min_count, timeout: ${timeout}s)..."

    while [ "$elapsed" -lt "$timeout" ]; do
        local total
        total=$(kubectl get "$kind" -l app.kubernetes.io/instance=kyverno -o json 2>/dev/null \
            | jq '[.items[].webhooks | length] | add // 0' 2>/dev/null || echo "0")
        total=$(sanitize_int "$total")

        if [ "$total" -ge "$min_count" ]; then
            return 0
        fi

        sleep "$interval"
        elapsed=$((elapsed + interval))
    done

    # Timeout reached - print diagnostics
    warn "Timeout waiting for $kind webhooks after ${timeout}s"
    echo ""
    echo "=== Diagnostic Output ==="
    echo "Webhook configurations:"
    kubectl get "$kind" -l app.kubernetes.io/instance=kyverno -o wide 2>&1 || true
    echo ""
    echo "Webhook configuration details:"
    kubectl describe "$kind" -l app.kubernetes.io/instance=kyverno 2>&1 || true
    echo ""
    echo "Kyverno admission controller logs (last 50 lines):"
    kubectl logs -n kyverno -l app.kubernetes.io/component=admission-controller --tail=50 2>&1 || true
    echo "=== End Diagnostic Output ==="
    echo ""

    return 1
}

echo "=== Kyverno Installation Validation ==="
echo ""

# 0. Check prerequisites
check_jq

# 1. Check namespace exists
info "Checking kyverno namespace..."
if kubectl get namespace kyverno >/dev/null 2>&1; then
    pass "kyverno namespace exists"
else
    fail "kyverno namespace does not exist"
    exit 1
fi

# 2. Check deployments
echo ""
info "Checking Kyverno deployments..."
DEPLOYMENTS=("kyverno-admission-controller" "kyverno-background-controller" "kyverno-cleanup-controller" "kyverno-reports-controller")

for deployment in "${DEPLOYMENTS[@]}"; do
    if kubectl get deployment -n kyverno "$deployment" >/dev/null 2>&1; then
        READY=$(kubectl get deployment -n kyverno "$deployment" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        DESIRED=$(kubectl get deployment -n kyverno "$deployment" -o jsonpath='{.status.replicas}' 2>/dev/null || echo "0")
        READY=$(sanitize_int "$READY")
        DESIRED=$(sanitize_int "$DESIRED")

        if [ "$READY" -ge 1 ] && [ "$READY" -eq "$DESIRED" ]; then
            pass "$deployment is ready ($READY/$DESIRED)"
        else
            fail "$deployment is not ready ($READY/$DESIRED)"
        fi
    else
        fail "$deployment not found"
    fi
done

# 3. Check pods (focus on deployment pods, ignore CronJob pods which may be pending)
echo ""
info "Checking Kyverno pods..."
kubectl get pods -n kyverno

# Count only pods from the main deployments (by name prefix)
# Exclude CronJob pods which may be in ImagePullBackOff
DEPLOYMENT_PODS=$(kubectl get pods -n kyverno --no-headers 2>/dev/null | grep -E "kyverno-(admission|background|cleanup|reports)-controller-" | wc -l || echo "0")
DEPLOYMENT_PODS=$(sanitize_int "$DEPLOYMENT_PODS")

if [ "$DEPLOYMENT_PODS" -eq 0 ]; then
    fail "No deployment pods found"
else
    DEPLOYMENT_READY=$(kubectl get pods -n kyverno --no-headers 2>/dev/null | grep -E "kyverno-(admission|background|cleanup|reports)-controller-" | grep "Running" | wc -l || echo "0")
    DEPLOYMENT_READY=$(sanitize_int "$DEPLOYMENT_READY")
    DEPLOYMENT_TOTAL=$DEPLOYMENT_PODS

    if [ "$DEPLOYMENT_READY" -eq "$DEPLOYMENT_TOTAL" ]; then
        pass "All deployment pods are ready ($DEPLOYMENT_READY/$DEPLOYMENT_TOTAL)"
    else
        fail "Not all deployment pods are ready ($DEPLOYMENT_READY/$DEPLOYMENT_TOTAL)"
    fi
fi

# Check if there are any CronJob cleanup pods (info only, not a failure)
CRONJOB_PODS=$(kubectl get pods -n kyverno --no-headers 2>/dev/null | grep -E "kyverno-cleanup-.*-[0-9]+-" | wc -l || echo "0")
CRONJOB_PODS=$(sanitize_int "$CRONJOB_PODS")

if [ "$CRONJOB_PODS" -gt 0 ]; then
    CRONJOB_READY=$(kubectl get pods -n kyverno --no-headers 2>/dev/null | grep -E "kyverno-cleanup-.*-[0-9]+-" | grep "Running" | wc -l || echo "0")
    CRONJOB_READY=$(sanitize_int "$CRONJOB_READY")

    if [ "$CRONJOB_READY" -eq "$CRONJOB_PODS" ]; then
        pass "CronJob cleanup pods are ready ($CRONJOB_READY/$CRONJOB_PODS)"
    else
        warn "Some CronJob cleanup pods are not ready ($CRONJOB_READY/$CRONJOB_PODS) - this is normal for periodic jobs"
    fi
fi

# 4. Check TLS secrets
echo ""
info "Checking TLS secrets..."

TLS_SECRETS=("kyverno-svc.kyverno.svc.kyverno-tls-pair" "kyverno-svc.kyverno.svc.kyverno-tls-ca")
for secret in "${TLS_SECRETS[@]}"; do
    if kubectl get secret -n kyverno "$secret" >/dev/null 2>&1; then
        pass "TLS secret '$secret' exists"

        # Validate certificate if it's the TLS pair
        if [[ "$secret" == *"tls-pair"* ]]; then
            if kubectl get secret -n kyverno "$secret" -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -text >/dev/null 2>&1; then
                pass "TLS certificate in '$secret' is valid"

                # Show certificate details
                echo "  Certificate details:"
                kubectl get secret -n kyverno "$secret" -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -subject -issuer -dates 2>/dev/null | sed 's/^/    /'
            else
                fail "TLS certificate in '$secret' is invalid"
            fi
        fi
    else
        fail "TLS secret '$secret' not found"
        warn "If using cert-manager, ensure it is installed and running"
        warn "If using self-signed certs, Kyverno may still be generating them"
    fi
done

# 5. Check webhook configurations (with wait/retry)
echo ""
info "Checking webhook configurations..."

# Wait for ValidatingWebhookConfigurations
if wait_for_webhooks "validatingwebhookconfigurations" 1 120; then
    VWC_TOTAL=$(kubectl get validatingwebhookconfigurations -l app.kubernetes.io/instance=kyverno -o json 2>/dev/null \
        | jq '[.items[].webhooks | length] | add // 0' 2>/dev/null || echo "0")
    VWC_TOTAL=$(sanitize_int "$VWC_TOTAL")

    if [ "$VWC_TOTAL" -gt 0 ]; then
        pass "Found $VWC_TOTAL validating webhook(s)"
    else
        fail "No validating webhooks configured"
    fi
else
    fail "Validating webhook configuration not ready after timeout"
fi

# Wait for MutatingWebhookConfigurations
if wait_for_webhooks "mutatingwebhookconfigurations" 1 120; then
    MWC_TOTAL=$(kubectl get mutatingwebhookconfigurations -l app.kubernetes.io/instance=kyverno -o json 2>/dev/null \
        | jq '[.items[].webhooks | length] | add // 0' 2>/dev/null || echo "0")
    MWC_TOTAL=$(sanitize_int "$MWC_TOTAL")

    if [ "$MWC_TOTAL" -gt 0 ]; then
        pass "Found $MWC_TOTAL mutating webhook(s)"
    else
        fail "No mutating webhooks configured"
    fi
else
    fail "Mutating webhook configuration not ready after timeout"
fi

# 6. Check webhook service
echo ""
info "Checking webhook service..."

if kubectl get svc -n kyverno kyverno-svc >/dev/null 2>&1; then
    pass "Webhook service 'kyverno-svc' exists"

    # Check service endpoints
    ENDPOINTS=$(kubectl get endpoints -n kyverno kyverno-svc -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w || echo "0")
    ENDPOINTS=$(sanitize_int "$ENDPOINTS")

    if [ "$ENDPOINTS" -gt 0 ]; then
        pass "Service has $ENDPOINTS endpoint(s)"
        kubectl get endpoints -n kyverno kyverno-svc
    else
        fail "Service has no endpoints"
        warn "Check if admission-controller pods are running"
    fi
else
    fail "Webhook service 'kyverno-svc' not found"
fi

# 7. Check CRDs
echo ""
info "Checking Kyverno CRDs..."

CRDS=("clusterpolicies.kyverno.io" "policies.kyverno.io" "clusteradmissionreports.kyverno.io")
for crd in "${CRDS[@]}"; do
    if kubectl get crd "$crd" >/dev/null 2>&1; then
        pass "CRD '$crd' exists"
    else
        fail "CRD '$crd' not found"
    fi
done

# 8. Test creating a simple policy
echo ""
info "Testing policy creation..."

TEST_POLICY="test-validation-$(date +%s)"
cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: ${TEST_POLICY}
spec:
  validationFailureAction: audit
  background: false
  rules:
  - name: test-rule
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "Test policy for validation"
      pattern:
        metadata:
          labels:
            test: "?*"
EOF

sleep 2

if kubectl get clusterpolicy "$TEST_POLICY" >/dev/null 2>&1; then
    pass "Can create ClusterPolicy resources"

    # Check policy status
    READY=$(kubectl get clusterpolicy "$TEST_POLICY" -o jsonpath='{.status.ready}' 2>/dev/null || echo "unknown")
    if [ "$READY" == "true" ]; then
        pass "Test policy is ready"
    elif [ "$READY" == "unknown" ]; then
        warn "Test policy status not available yet (may need more time)"
    else
        warn "Test policy is not ready (status: $READY)"
    fi

    # Cleanup
    kubectl delete clusterpolicy "$TEST_POLICY" >/dev/null 2>&1 || true
else
    fail "Cannot create ClusterPolicy resources"
fi

# 9. Check for cert-manager (recommended)
echo ""
info "Checking for cert-manager..."

if kubectl get namespace cert-manager >/dev/null 2>&1; then
    pass "cert-manager namespace exists"

    CERT_MANAGER_READY=$(kubectl get pods -n cert-manager -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -o "True" | wc -l || echo "0")
    CERT_MANAGER_PODS=$(kubectl get pods -n cert-manager --no-headers 2>/dev/null | wc -l || echo "0")
    CERT_MANAGER_READY=$(sanitize_int "$CERT_MANAGER_READY")
    CERT_MANAGER_PODS=$(sanitize_int "$CERT_MANAGER_PODS")

    if [ "$CERT_MANAGER_READY" -eq "$CERT_MANAGER_PODS" ] && [ "$CERT_MANAGER_PODS" -gt 0 ]; then
        pass "cert-manager is running ($CERT_MANAGER_READY/$CERT_MANAGER_PODS pods ready)"
    else
        warn "cert-manager pods may not be ready ($CERT_MANAGER_READY/$CERT_MANAGER_PODS)"
    fi
else
    warn "cert-manager not found (Kyverno can use self-signed certs)"
    info "For production, install cert-manager: kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml"
fi

# Summary
echo ""
echo "=== Validation Summary ==="
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ "$FAILED" -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
    echo ""
    echo "For troubleshooting, see: docs/TROUBLESHOOTING_KYVERNO.md"
    exit 1
else
    echo -e "${GREEN}All checks passed!${NC}"
    echo ""
    echo "Kyverno is properly installed and ready to enforce policies."
    exit 0
fi
