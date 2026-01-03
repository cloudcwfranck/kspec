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

# List ValidatingWebhookConfiguration names matching Kyverno pattern
list_vwc_names() {
    kubectl get validatingwebhookconfigurations -o json 2>/dev/null \
        | jq -r '.items[].metadata.name' 2>/dev/null \
        | grep -E '^kyverno-.*validating-webhook-cfg$' || true
}

# List MutatingWebhookConfiguration names matching Kyverno pattern
list_mwc_names() {
    kubectl get mutatingwebhookconfigurations -o json 2>/dev/null \
        | jq -r '.items[].metadata.name' 2>/dev/null \
        | grep -E '^kyverno-.*mutating-webhook-cfg$' || true
}

# Wait for a secret to exist
# Usage: wait_for_secret <namespace> <secret_name> <timeout_seconds>
wait_for_secret() {
    local namespace="$1"
    local secret_name="$2"
    local timeout="$3"
    local interval=5
    local elapsed=0

    info "Waiting for secret $namespace/$secret_name (timeout: ${timeout}s)..."

    while [ "$elapsed" -lt "$timeout" ]; do
        if kubectl get secret -n "$namespace" "$secret_name" >/dev/null 2>&1; then
            pass "Secret $namespace/$secret_name exists"
            return 0
        fi

        sleep "$interval"
        elapsed=$((elapsed + interval))
    done

    fail "Secret $namespace/$secret_name not found after ${timeout}s timeout"
    return 1
}

# Wait for webhook configurations to be created and configured
# Usage: wait_for_webhooks <kind> <min_count> <timeout_seconds>
wait_for_webhooks() {
    local kind="$1"
    local min_count="$2"
    local timeout="$3"
    local interval=5
    local elapsed=0

    info "Waiting for $kind webhooks (min: $min_count, timeout: ${timeout}s)..."

    while [ "$elapsed" -lt "$timeout" ]; do
        local names
        if [ "$kind" = "validatingwebhookconfigurations" ]; then
            names=$(list_vwc_names)
        else
            names=$(list_mwc_names)
        fi

        # Count matching webhook configs
        local config_count
        config_count=$(echo "$names" | grep -c . || echo "0")
        config_count=$(sanitize_int "$config_count")

        if [ "$config_count" -gt 0 ]; then
            # Found configs, now count webhooks within them
            local total=0
            while IFS= read -r name; do
                [ -z "$name" ] && continue
                local count
                count=$(kubectl get "$kind" "$name" -o json 2>/dev/null \
                    | jq '.webhooks | length' 2>/dev/null || echo "0")
                count=$(sanitize_int "$count")
                total=$((total + count))
            done <<< "$names"

            info "Found $config_count config(s) with $total webhook(s): $(echo "$names" | tr '\n' ' ')"

            if [ "$total" -ge "$min_count" ]; then
                return 0
            fi
        else
            info "No $kind found matching pattern (elapsed: ${elapsed}s)"
        fi

        sleep "$interval"
        elapsed=$((elapsed + interval))
    done

    # Timeout reached - print diagnostics
    warn "Timeout waiting for $kind after ${timeout}s"
    echo ""
    echo "=== Diagnostic Output ==="
    echo "All webhook configurations:"
    kubectl get "$kind" -o wide 2>&1 || true
    echo ""
    echo "Kyverno webhook configurations (by name pattern):"
    kubectl get "$kind" -o name 2>&1 | grep kyverno || echo "  (none found)"
    echo ""

    if [ "$kind" = "validatingwebhookconfigurations" ]; then
        local vwc_names
        vwc_names=$(list_vwc_names)
        if [ -n "$vwc_names" ]; then
            echo "Details of Kyverno validating webhook configs:"
            while IFS= read -r name; do
                [ -z "$name" ] && continue
                echo "--- $name ---"
                kubectl describe validatingwebhookconfiguration "$name" 2>&1 || true
            done <<< "$vwc_names"
        fi
    else
        local mwc_names
        mwc_names=$(list_mwc_names)
        if [ -n "$mwc_names" ]; then
            echo "Details of Kyverno mutating webhook configs:"
            while IFS= read -r name; do
                [ -z "$name" ] && continue
                echo "--- $name ---"
                kubectl describe mutatingwebhookconfiguration "$name" 2>&1 || true
            done <<< "$mwc_names"
        fi
    fi

    echo ""
    echo "Kyverno admission controller logs (last 100 lines):"
    kubectl logs -n kyverno -l app.kubernetes.io/component=admission-controller --tail=100 2>&1 || true
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

# 4. Wait for TLS CA secret (required before webhook configs are created)
echo ""
info "Waiting for TLS secrets (required for webhook configuration)..."

# Wait for CA secret first - Kyverno needs this before creating webhook configs
if ! wait_for_secret "kyverno" "kyverno-svc.kyverno.svc.kyverno-tls-ca" 180; then
    warn "TLS CA secret not found - webhook configuration may be delayed"
fi

# Check TLS pair secret
TLS_SECRETS=("kyverno-svc.kyverno.svc.kyverno-tls-pair")
for secret in "${TLS_SECRETS[@]}"; do
    if kubectl get secret -n kyverno "$secret" >/dev/null 2>&1; then
        pass "TLS secret '$secret' exists"

        # Validate certificate
        if kubectl get secret -n kyverno "$secret" -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -text >/dev/null 2>&1; then
            pass "TLS certificate in '$secret' is valid"

            # Show certificate details
            echo "  Certificate details:"
            kubectl get secret -n kyverno "$secret" -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -subject -issuer -dates 2>/dev/null | sed 's/^/    /'
        else
            fail "TLS certificate in '$secret' is invalid"
        fi
    else
        fail "TLS secret '$secret' not found"
        warn "If using cert-manager, ensure it is installed and running"
        warn "If using self-signed certs, Kyverno may still be generating them"
    fi
done

# 5. Check webhook configurations (with wait/retry)
# Increased timeout to 300s for CI environments (kind can be slow)
echo ""
info "Checking webhook configurations..."

# Wait for ValidatingWebhookConfigurations
if wait_for_webhooks "validatingwebhookconfigurations" 1 300; then
    VWC_NAMES=$(list_vwc_names)
    VWC_COUNT=$(echo "$VWC_NAMES" | grep -c . || echo "0")
    VWC_COUNT=$(sanitize_int "$VWC_COUNT")

    # Count total webhooks across all configs
    VWC_TOTAL=0
    while IFS= read -r name; do
        [ -z "$name" ] && continue
        local count
        count=$(kubectl get validatingwebhookconfigurations "$name" -o json 2>/dev/null \
            | jq '.webhooks | length' 2>/dev/null || echo "0")
        count=$(sanitize_int "$count")
        VWC_TOTAL=$((VWC_TOTAL + count))
    done <<< "$VWC_NAMES"

    if [ "$VWC_TOTAL" -gt 0 ]; then
        pass "Found $VWC_COUNT validating webhook config(s) with $VWC_TOTAL total webhook(s)"
        echo "  Configs: $(echo "$VWC_NAMES" | tr '\n' ' ')"
    else
        fail "No validating webhooks configured"
    fi
else
    fail "Validating webhook configuration not ready after timeout"
fi

# Wait for MutatingWebhookConfigurations
if wait_for_webhooks "mutatingwebhookconfigurations" 1 300; then
    MWC_NAMES=$(list_mwc_names)
    MWC_COUNT=$(echo "$MWC_NAMES" | grep -c . || echo "0")
    MWC_COUNT=$(sanitize_int "$MWC_COUNT")

    # Count total webhooks across all configs
    MWC_TOTAL=0
    while IFS= read -r name; do
        [ -z "$name" ] && continue
        local count
        count=$(kubectl get mutatingwebhookconfigurations "$name" -o json 2>/dev/null \
            | jq '.webhooks | length' 2>/dev/null || echo "0")
        count=$(sanitize_int "$count")
        MWC_TOTAL=$((MWC_TOTAL + count))
    done <<< "$MWC_NAMES"

    if [ "$MWC_TOTAL" -gt 0 ]; then
        pass "Found $MWC_COUNT mutating webhook config(s) with $MWC_TOTAL total webhook(s)"
        echo "  Configs: $(echo "$MWC_NAMES" | tr '\n' ' ')"
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
