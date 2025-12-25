# E2E Testing Contract

This document defines the testing contract for kspec's end-to-end tests. These tests verify the complete workflow from installation through policy deployment and enforcement.

## Test Environment

- **Cluster**: kind (Kubernetes in Docker) v1.29.0
- **Kyverno**: v1.11.4 with server-side apply
- **Test Namespace**: `kspec-test`
- **Timeout**: 10 minutes total

## Test Workflow

### 1. Cluster Setup

```bash
# Create kind cluster
kind create cluster --name kspec-test --config=kind-config.yaml

# Create test namespace with Pod Security Standards labels
kubectl create namespace kspec-test
kubectl label namespace kspec-test \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

### 2. Kyverno Installation

**Critical Requirements:**
- Use server-side apply to handle large CRD annotations (>262KB limit)
- Wait for ALL 4 deployments to be ready
- Wait for webhook configurations to be populated (not just created)
- Add grace period for webhook service to stabilize

```bash
# Install Kyverno with server-side apply
kubectl apply --server-side=true -f \
  https://github.com/kyverno/kyverno/releases/download/v1.11.4/install.yaml

# Wait for all deployments
kubectl wait --for=condition=available --timeout=300s \
  deployment/kyverno-admission-controller -n kyverno
kubectl wait --for=condition=available --timeout=300s \
  deployment/kyverno-background-controller -n kyverno
kubectl wait --for=condition=available --timeout=300s \
  deployment/kyverno-cleanup-controller -n kyverno
kubectl wait --for=condition=available --timeout=300s \
  deployment/kyverno-reports-controller -n kyverno

# Wait for webhook configurations to be populated
# This is critical - webhook configs can exist with WEBHOOKS=0
for i in {1..60}; do
  VALIDATING_COUNT=$(kubectl get validatingwebhookconfiguration \
    kyverno-resource-validating-webhook-cfg -o jsonpath='{.webhooks}' | \
    grep -o "name" | wc -l || echo "0")
  MUTATING_COUNT=$(kubectl get mutatingwebhookconfiguration \
    kyverno-resource-mutating-webhook-cfg -o jsonpath='{.webhooks}' | \
    grep -o "name" | wc -l || echo "0")

  if [ "$VALIDATING_COUNT" -gt "0" ] && [ "$MUTATING_COUNT" -gt "0" ]; then
    break
  fi
  sleep 2
done

# Wait for all pods to be ready
kubectl wait --for=condition=ready pod --all -n kyverno --timeout=120s

# Grace period for webhook service to stabilize
sleep 20
```

**Why this is important:**
- Webhook configurations are created immediately but take time to populate with actual webhook entries
- The webhook service needs time to bind to endpoints and start accepting connections
- Without proper waiting, policy creation fails with "connection refused" errors

### 3. Test Cases

#### 3.1 Policy Enforcement (Dry-Run)

**Test**: Generate policies without deploying them

```bash
kspec enforce --spec specs/examples/comprehensive.yaml --dry-run
```

**Expected Behavior:**
- Generates 7 policies
- Does not deploy to cluster
- Displays policy list and next steps

#### 3.2 Policy Enforcement (Actual Deployment)

**Test**: Deploy policies to cluster

```bash
kspec enforce --spec specs/examples/comprehensive.yaml
```

**Expected Behavior:**
- Detects Kyverno as installed
- Generates 7 policies
- Deploys all policies successfully
- Reports 7 policies applied

**Verification:**
```bash
# All policies should exist
kubectl get clusterpolicies

# Specific policies should be queryable
kubectl get clusterpolicy require-run-as-non-root
kubectl get clusterpolicy disallow-privilege-escalation
kubectl get clusterpolicy require-resource-limits
kubectl get clusterpolicy disallow-privileged-containers
kubectl get clusterpolicy disallow-host-namespaces
kubectl get clusterpolicy require-image-digests
kubectl get clusterpolicy block-image-registries
```

#### 3.3 Policy Enforcement (Idempotency)

**Test**: Re-run enforcement to verify idempotent updates

```bash
kspec enforce --spec specs/examples/comprehensive.yaml
```

**Expected Behavior:**
- Updates existing policies (not creates)
- Successfully fetches resourceVersion from existing policies
- Applies updates without errors
- Policy count remains at 7 (no duplicates)

**Implementation Detail:**
When a policy already exists, kspec must:
1. Detect "already exists" error
2. Fetch the existing policy to get its resourceVersion
3. Set the resourceVersion on the new policy object
4. Perform update operation

This prevents the error: `metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`

#### 3.4 Policy Enforcement (Blocking Behavior)

**Test**: Verify policies actually block non-compliant workloads

```bash
# Wait for policies to propagate to webhook
sleep 10

# Try to create a privileged pod (should be blocked)
cat <<EOF | kubectl apply -f - 2>&1 | tee policy-test-output.txt
apiVersion: v1
kind: Pod
metadata:
  name: privileged-pod-test
  namespace: default
spec:
  containers:
  - name: test
    image: nginx:latest
    securityContext:
      privileged: true
      allowPrivilegeEscalation: true
EOF
```

**Expected Behavior:**
- Pod creation is DENIED by admission webhook
- Error message lists all violated policies:
  - `disallow-privileged-containers` (privileged: true not allowed)
  - `disallow-privilege-escalation` (allowPrivilegeEscalation must be false)
  - `require-run-as-non-root` (runAsNonRoot must be true)
  - `require-resource-limits` (CPU/memory limits required)
  - `require-image-digests` (must use digest, not tag)

**Validation:**
```bash
# Check that error message contains policy violations
if ! grep -q "was blocked due to the following policies" policy-test-output.txt; then
  echo "ERROR: Policies did not block non-compliant pod"
  exit 1
fi
```

#### 3.5 Policy Export

**Test**: Export generated policies to YAML file

```bash
kspec enforce --spec specs/examples/comprehensive.yaml --dry-run --output policies.yaml
```

**Expected Behavior:**
- Creates `policies.yaml` file
- File contains valid Kubernetes YAML with:
  - `apiVersion: kyverno.io/v1`
  - `kind: ClusterPolicy`
  - Document separators (`---`) between policies
- All 7 policies are in the file

**Validation:**
```bash
# Check file exists
[ -f policies.yaml ] || exit 1

# Check valid YAML with apiVersion
grep -q "apiVersion: kyverno.io/v1" policies.yaml || exit 1

# Check it's valid YAML (can be parsed)
kubectl apply --dry-run=client -f policies.yaml
```

### 4. Test Matrix

| Test Case | Purpose | Expected Result |
|-----------|---------|-----------------|
| Kyverno Installation | Verify Kyverno v1.11.4 installs successfully | 4 deployments ready, webhooks populated |
| Dry-run Enforcement | Verify policy generation without deployment | 7 policies generated, none deployed |
| Policy Deployment | Verify policies are created in cluster | 7 ClusterPolicy resources exist |
| Policy Verification | Verify policies can be queried | All 7 policies queryable by name |
| Idempotent Update | Verify re-running updates existing policies | 7 policies updated, no errors |
| Blocking Behavior | Verify policies prevent non-compliant workloads | Privileged pod is denied |
| Policy Export | Verify policies export to valid YAML | Valid Kubernetes YAML file created |

## Common Issues and Solutions

### Issue: Webhook Connection Refused

**Symptom:**
```
failed calling webhook "mutate-policy.kyverno.svc": Post "https://kyverno-svc.kyverno.svc:443/policymutate?timeout=10s": dial tcp: connect: connection refused
```

**Root Cause:**
- Webhook configurations exist but entries aren't populated yet
- Webhook service hasn't started accepting connections

**Solution:**
- Wait for webhook entry count > 0 (not just existence)
- Add 20-second grace period after pods are ready
- Verify all pods are ready, not just deployments

### Issue: Policy Update Fails

**Symptom:**
```
metadata.resourceVersion: Invalid value: 0x0: must be specified for an update
```

**Root Cause:**
- Trying to update without fetching the existing policy's resourceVersion

**Solution:**
1. Detect "already exists" error
2. GET existing policy
3. Extract resourceVersion
4. Set on new policy object
5. Perform UPDATE

### Issue: Kyverno Not Detected

**Symptom:**
```
Kyverno is not installed
```

**Root Cause:**
- Checking for deployment named "kyverno"
- Kyverno v1.11+ uses "kyverno-admission-controller"

**Solution:**
- Update detection logic to check for "kyverno-admission-controller" deployment
- Verify in kyverno namespace

### Issue: Invalid Policy YAML Export

**Symptom:**
```
[FAIL] Invalid policy YAML
```

**Root Cause:**
- Using gopkg.in/yaml.v3 which doesn't handle TypeMeta correctly
- Missing apiVersion/kind in exported YAML

**Solution:**
- Use sigs.k8s.io/yaml (Kubernetes YAML marshaler)
- Add yaml struct tags: `yaml:",inline"` for TypeMeta

## Performance Benchmarks

- **Cluster creation**: ~30 seconds
- **Kyverno installation**: ~45 seconds
- **Policy generation**: <1 second
- **Policy deployment**: ~5 seconds (7 policies)
- **Total test time**: ~1.5 minutes

## CI/CD Integration

```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Install kind
        run: |
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind
      - name: Run E2E tests
        run: make test-e2e
```

## Future Enhancements

- [ ] Test policy priority and ordering
- [ ] Test policy exceptions
- [ ] Test multi-cluster deployment
- [ ] Test policy conflict detection
- [ ] Performance testing with 100+ policies
- [ ] Chaos testing (webhook failures, network issues)
