# Troubleshooting Kyverno Installation and TLS Issues

## Overview

Kyverno is a policy engine for Kubernetes that kspec uses for policy enforcement. Kyverno operates as an admission webhook, which requires TLS certificates to function properly.

## Common Issues

### 1. Missing TLS Secret Error

**Symptom:**
```
Error: admission webhook "..." failed: failed calling webhook "...": Post "https://kyverno-svc.kyverno.svc:443/...": x509: certificate signed by unknown authority
```

Or in Kyverno admission controller logs:
```
Failed to read certificate: secret "kyverno-svc.kyverno.svc.kyverno-tls-pair" not found
```

**Root Cause:**
Kyverno's admission webhooks require TLS certificates. These can be provided by:
1. **cert-manager** (recommended) - Automatically generates and rotates certificates
2. **Self-signed certificates** - Kyverno generates its own certificates

**Solution:**

#### Option A: Install cert-manager (Recommended)

```bash
# Install cert-manager BEFORE Kyverno
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=120s
kubectl wait --for=condition=Available deployment/cert-manager-cainjector -n cert-manager --timeout=120s

# Now install Kyverno (it will use cert-manager automatically)
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update

helm install kyverno kyverno/kyverno \
  --namespace kyverno \
  --create-namespace \
  --wait \
  --timeout=5m
```

#### Option B: Use Kyverno's Built-in Certificate Generation

If you cannot use cert-manager, ensure Kyverno's webhook controller is enabled:

```bash
helm install kyverno kyverno/kyverno \
  --namespace kyverno \
  --create-namespace \
  --set webhooksCleanup.enabled=true \
  --wait \
  --timeout=5m
```

### 2. Kyverno Admission Controller Not Ready

**Symptom:**
```
kubectl get pods -n kyverno
NAME                                        READY   STATUS    RESTARTS   AGE
kyverno-admission-controller-xxx            0/1     Running   0          2m
```

**Diagnostic Steps:**

```bash
# 1. Check pod logs
kubectl logs -n kyverno deployment/kyverno-admission-controller --tail=50

# 2. Check pod events
kubectl describe pod -n kyverno -l app.kubernetes.io/component=admission-controller

# 3. Check TLS secrets exist
kubectl get secrets -n kyverno | grep tls

# Expected secrets:
# - kyverno-svc.kyverno.svc.kyverno-tls-ca (CA certificate)
# - kyverno-svc.kyverno.svc.kyverno-tls-pair (TLS certificate and key)

# 4. Check webhook configurations
kubectl get validatingwebhookconfigurations | grep kyverno
kubectl get mutatingwebhookconfigurations | grep kyverno

# 5. Verify webhook service
kubectl get svc -n kyverno kyverno-svc
kubectl get endpoints -n kyverno kyverno-svc
```

**Common Issues:**

- **Missing TLS secrets**: Install cert-manager or wait for Kyverno to generate self-signed certs
- **Webhook not registered**: Wait for Kyverno to fully start (may take 60-120 seconds)
- **Service has no endpoints**: Check if admission-controller pods are running

**Solution:**

```bash
# Delete and recreate the Kyverno installation
helm uninstall kyverno -n kyverno
kubectl delete namespace kyverno

# Ensure cert-manager is installed
kubectl get pods -n cert-manager

# Reinstall Kyverno
helm install kyverno kyverno/kyverno \
  --namespace kyverno \
  --create-namespace \
  --wait \
  --timeout=5m

# Wait for all components to be ready
kubectl wait --for=condition=ready pod --all -n kyverno --timeout=300s
```

### 3. Webhook Timeouts or Connection Refused

**Symptom:**
```
Error: Internal error occurred: failed calling webhook "validate.kyverno.svc-fail": Post "https://kyverno-svc.kyverno.svc:443/validate": context deadline exceeded
```

**Diagnostic Steps:**

```bash
# Check if webhook service is responding
kubectl run -n kyverno test-curl --image=curlimages/curl:latest --rm -it --restart=Never \
  -- curl -k https://kyverno-svc:443/health

# Expected output: {"status":"ok"} or similar
```

**Common Causes:**

1. **Network policies blocking webhook traffic**: Ensure kube-system can reach kyverno namespace
2. **Resource constraints**: Kyverno pods may be throttled or OOMKilled
3. **Slow startup**: Kyverno may still be initializing

**Solution:**

```bash
# Increase resource limits if needed
helm upgrade kyverno kyverno/kyverno \
  --namespace kyverno \
  --reuse-values \
  --set admissionController.resources.limits.memory=512Mi \
  --set admissionController.resources.limits.cpu=1000m

# Check for network policies that might block webhook traffic
kubectl get networkpolicies -A
```

### 4. Policies Not Being Enforced

**Symptom:**
Non-compliant resources are created successfully even though Kyverno policies exist.

**Diagnostic Steps:**

```bash
# 1. Verify policies exist and are ready
kubectl get clusterpolicies
kubectl get clusterpolicy <policy-name> -o yaml

# 2. Check policy status
kubectl get clusterpolicy <policy-name> -o jsonpath='{.status.ready}'
# Expected: true

# 3. Check validationFailureAction
kubectl get clusterpolicy <policy-name> -o jsonpath='{.spec.validationFailureAction}'
# Should be "enforce" for blocking, "audit" for logging only

# 4. Test with a simple policy
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-labels
spec:
  validationFailureAction: enforce
  background: false
  rules:
  - name: check-for-labels
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "Label 'app' is required"
      pattern:
        metadata:
          labels:
            app: "?*"
EOF

# Try creating a pod without the label (should fail)
kubectl run test-pod --image=nginx
```

**Common Issues:**

- **Policy in audit mode**: Change `validationFailureAction` to `enforce`
- **Webhook not registered**: Wait for Kyverno to register webhooks
- **Namespace excluded**: Check if namespace has `pod-security.kubernetes.io/enforce` labels that override Kyverno

## Validation Checklist

Use this checklist to verify Kyverno is properly installed:

```bash
#!/bin/bash
# Kyverno Installation Validation

echo "=== Kyverno Installation Validation ==="

# 1. Deployments
echo "1. Checking Kyverno deployments..."
kubectl get deployments -n kyverno
kubectl wait --for=condition=available --timeout=60s deployment/kyverno-admission-controller -n kyverno || echo "FAIL: Admission controller not ready"
kubectl wait --for=condition=available --timeout=60s deployment/kyverno-background-controller -n kyverno || echo "FAIL: Background controller not ready"
kubectl wait --for=condition=available --timeout=60s deployment/kyverno-cleanup-controller -n kyverno || echo "FAIL: Cleanup controller not ready"
kubectl wait --for=condition=available --timeout=60s deployment/kyverno-reports-controller -n kyverno || echo "FAIL: Reports controller not ready"

# 2. Pods
echo "2. Checking Kyverno pods..."
kubectl get pods -n kyverno
ALL_READY=$(kubectl get pods -n kyverno -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o "True" | wc -l)
TOTAL_PODS=$(kubectl get pods -n kyverno --no-headers | wc -l)
echo "Ready pods: $ALL_READY / $TOTAL_PODS"

# 3. TLS Secrets
echo "3. Checking TLS secrets..."
kubectl get secrets -n kyverno | grep tls || echo "WARN: No TLS secrets found"
kubectl get secret -n kyverno kyverno-svc.kyverno.svc.kyverno-tls-pair -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -subject 2>/dev/null && echo "✓ TLS certificate valid" || echo "FAIL: TLS certificate invalid or missing"

# 4. Webhook Configurations
echo "4. Checking webhook configurations..."
kubectl get validatingwebhookconfigurations | grep kyverno
kubectl get mutatingwebhookconfigurations | grep kyverno

VALIDATING_COUNT=$(kubectl get validatingwebhookconfiguration kyverno-resource-validating-webhook-cfg -o jsonpath='{.webhooks}' 2>/dev/null | grep -o "name" | wc -l || echo "0")
MUTATING_COUNT=$(kubectl get mutatingwebhookconfiguration kyverno-resource-mutating-webhook-cfg -o jsonpath='{.webhooks}' 2>/dev/null | grep -o "name" | wc -l || echo "0")
echo "Validating webhooks: $VALIDATING_COUNT, Mutating webhooks: $MUTATING_COUNT"

# 5. Webhook Service
echo "5. Checking webhook service..."
kubectl get svc -n kyverno kyverno-svc
ENDPOINTS=$(kubectl get endpoints -n kyverno kyverno-svc -o jsonpath='{.subsets[*].addresses[*].ip}' | wc -w)
echo "Service endpoints: $ENDPOINTS"
[ "$ENDPOINTS" -gt "0" ] && echo "✓ Service has endpoints" || echo "FAIL: Service has no endpoints"

# 6. CRDs
echo "6. Checking Kyverno CRDs..."
kubectl get crd clusterpolicies.kyverno.io >/dev/null 2>&1 && echo "✓ ClusterPolicy CRD exists" || echo "FAIL: ClusterPolicy CRD missing"

# 7. Test Policy
echo "7. Testing sample policy..."
cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: test-validation-policy
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
      message: "Test policy"
      pattern:
        metadata:
          labels:
            test: "?*"
EOF

sleep 2
kubectl get clusterpolicy test-validation-policy >/dev/null 2>&1 && echo "✓ Can create policies" || echo "FAIL: Cannot create policies"
kubectl delete clusterpolicy test-validation-policy >/dev/null 2>&1

echo ""
echo "=== Validation Complete ==="
```

## Getting Help

If issues persist:

1. **Check Kyverno logs:**
   ```bash
   kubectl logs -n kyverno deployment/kyverno-admission-controller
   ```

2. **Check Kyverno version:**
   ```bash
   kubectl get deployment -n kyverno kyverno-admission-controller -o jsonpath='{.spec.template.spec.containers[0].image}'
   ```

3. **Consult Kyverno documentation:**
   - Installation: https://kyverno.io/docs/installation/
   - Troubleshooting: https://kyverno.io/docs/troubleshooting/

4. **Open an issue:**
   - kspec issues: https://github.com/cloudcwfranck/kspec/issues
   - Kyverno issues: https://github.com/kyverno/kyverno/issues
