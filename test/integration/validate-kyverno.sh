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
    ((PASSED++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAILED++))
}

warn() {
    echo -e "${YELLOW}⚠ WARN${NC}: $1"
}

info() {
    echo "ℹ INFO: $1"
}

echo "=== Kyverno Installation Validation ==="
echo ""

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
        if [ "$READY" -ge "1" ] && [ "$READY" -eq "$DESIRED" ]; then
            pass "$deployment is ready ($READY/$DESIRED)"
        else
            fail "$deployment is not ready ($READY/$DESIRED)"
        fi
    else
        fail "$deployment not found"
    fi
done

# 3. Check pods
echo ""
info "Checking Kyverno pods..."
kubectl get pods -n kyverno

ALL_READY=$(kubectl get pods -n kyverno -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o "True" | wc -l)
TOTAL_PODS=$(kubectl get pods -n kyverno --no-headers | wc -l)

if [ "$ALL_READY" -eq "$TOTAL_PODS" ] && [ "$TOTAL_PODS" -gt "0" ]; then
    pass "All Kyverno pods are ready ($ALL_READY/$TOTAL_PODS)"
else
    fail "Not all pods are ready ($ALL_READY/$TOTAL_PODS)"
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

# 5. Check webhook configurations
echo ""
info "Checking webhook configurations..."

if kubectl get validatingwebhookconfiguration kyverno-resource-validating-webhook-cfg >/dev/null 2>&1; then
    VALIDATING_COUNT=$(kubectl get validatingwebhookconfiguration kyverno-resource-validating-webhook-cfg -o jsonpath='{.webhooks}' 2>/dev/null | grep -o "name" | wc -l || echo "0")
    if [ "$VALIDATING_COUNT" -gt "0" ]; then
        pass "Validating webhook configuration exists with $VALIDATING_COUNT webhook(s)"
    else
        fail "Validating webhook configuration has no webhooks"
    fi
else
    fail "Validating webhook configuration not found"
fi

if kubectl get mutatingwebhookconfiguration kyverno-resource-mutating-webhook-cfg >/dev/null 2>&1; then
    MUTATING_COUNT=$(kubectl get mutatingwebhookconfiguration kyverno-resource-mutating-webhook-cfg -o jsonpath='{.webhooks}' 2>/dev/null | grep -o "name" | wc -l || echo "0")
    if [ "$MUTATING_COUNT" -gt "0" ]; then
        pass "Mutating webhook configuration exists with $MUTATING_COUNT webhook(s)"
    else
        fail "Mutating webhook configuration has no webhooks"
    fi
else
    fail "Mutating webhook configuration not found"
fi

# 6. Check webhook service
echo ""
info "Checking webhook service..."

if kubectl get svc -n kyverno kyverno-svc >/dev/null 2>&1; then
    pass "Webhook service 'kyverno-svc' exists"

    # Check service endpoints
    ENDPOINTS=$(kubectl get endpoints -n kyverno kyverno-svc -o jsonpath='{.subsets[*].addresses[*].ip}' | wc -w)
    if [ "$ENDPOINTS" -gt "0" ]; then
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

    CERT_MANAGER_READY=$(kubectl get pods -n cert-manager -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o "True" | wc -l)
    CERT_MANAGER_PODS=$(kubectl get pods -n cert-manager --no-headers | wc -l)

    if [ "$CERT_MANAGER_READY" -eq "$CERT_MANAGER_PODS" ] && [ "$CERT_MANAGER_PODS" -gt "0" ]; then
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
if [ "$FAILED" -gt "0" ]; then
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
