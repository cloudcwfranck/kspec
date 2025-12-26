# kspec Operator Quickstart Guide

Get started with the kspec Kubernetes Operator in 10 minutes - continuous compliance monitoring, automatic drift remediation, and multi-cluster management.

## 🎯 What You'll Learn

- Install the kspec operator in your cluster
- Create your first ClusterSpecification
- Monitor compliance automatically
- Set up multi-cluster scanning
- View real-time compliance dashboards

---

## Prerequisites

✅ **Kubernetes 1.24+** cluster with kubectl access  
✅ **cluster-admin** permissions (for CRD installation)  
⚠️ **Kyverno 1.10+** (optional, for policy enforcement)

---

## Quick Install (3 minutes)

### Step 1: Install the Operator

**Option A: Kustomize (Recommended)**
```bash
# Install CRDs and operator
kubectl apply -k github.com/cloudcwfranck/kspec/config/default

# Verify installation
kubectl get pods -n kspec-system
```

Expected output:
```
NAME                              READY   STATUS    RESTARTS   AGE
kspec-operator-59f7d8c5b4-x7k2p   1/1     Running   0          30s
```

**Option B: GitOps (ArgoCD)**
```yaml
# apps/kspec-operator.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kspec-operator
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/cloudcwfranck/kspec
    targetRevision: main
    path: config/default
  destination:
    server: https://kubernetes.default.svc
    namespace: kspec-system
  syncPolicy:
    automated: {prune: true, selfHeal: true}
    syncOptions: [CreateNamespace=true]
```

### Step 2: Create a ClusterSpecification

```bash
kubectl apply -f - <<EOF
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: my-cluster
  namespace: kspec-system
spec:
  kubernetes:
    minVersion: "1.27.0"
    maxVersion: "1.30.0"

  podSecurity:
    enforce: restricted
    exemptions:
      namespaces: [kube-system, kspec-system]

  network:
    requireNetworkPolicies: true
    defaultDenyIngress: true

  workloads:
    containers:
      required:
        - key: securityContext.runAsNonRoot
          value: "true"
        - key: securityContext.allowPrivilegeEscalation
          value: "false"
EOF
```

### Step 3: Watch the Magic Happen

```bash
# Watch ClusterSpecification status
kubectl get clusterspec -n kspec-system -w
```

Within 30 seconds, you'll see:
```
NAME         PHASE   SCORE   LAST SCAN           AGE
my-cluster   Active  95      2025-01-15T10:30Z   1m
```

---

## Understanding Your Cluster Status

### View Compliance Score

```bash
kubectl get clusterspec my-cluster -n kspec-system \
  -o jsonpath='{.status.complianceScore}'
# Output: 95
```

### View Detailed Status

```bash
kubectl describe clusterspec my-cluster -n kspec-system
```

Key status fields:
- **Phase**: `Pending` → `Active` → `Failed` (if issues)
- **ComplianceScore**: 0-100 percentage
- **LastScanTime**: Most recent scan
- **Summary**: Total/Passed/Failed check counts
- **Conditions**: Ready, PolicyEnforced, DriftDetected

### View Compliance Reports

```bash
# List all reports
kubectl get compliancereport -n kspec-system

# View latest report details
kubectl get compliancereport -n kspec-system \
  -l kspec.io/cluster-spec=my-cluster \
  --sort-by=.metadata.creationTimestamp \
  -o yaml | tail -1
```

---

## Multi-Cluster Setup (5 minutes)

Monitor compliance across multiple clusters from a single operator.

### Step 1: Discover Your Clusters

```bash
# List clusters from kubeconfig
kspec cluster discover

# Output:
# NAME             CONTEXT           API SERVER                        AUTH TYPE
# prod-eks         prod-eks-admin    https://prod.eks.amazonaws.com    certificate
# staging-gke      staging-gke       https://staging.gke.google.com    token
```

### Step 2: Add Remote Cluster

```bash
# Generate ClusterTarget manifests
kspec cluster add prod-eks > clustertarget-prod.yaml

# Apply to management cluster
kubectl apply -f clustertarget-prod.yaml
```

This creates:
1. **Secret** with kubeconfig credentials
2. **ClusterTarget** CR defining the remote cluster

### Step 3: Create ClusterSpec for Remote Cluster

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: prod-cluster-spec
  namespace: kspec-system
spec:
  # Reference remote cluster
  clusterRef:
    name: prod-eks
    namespace: kspec-system

  # Same compliance requirements
  kubernetes:
    minVersion: "1.27.0"
  podSecurity:
    enforce: restricted
  # ... etc
```

### Step 4: Monitor Fleet Health

```bash
# Check all cluster targets
kubectl get clustertarget -n kspec-system

# Output:
# NAME          REACHABLE   VERSION   PLATFORM   NODES   AGE
# prod-eks      true        v1.28.0   eks        10      5m
# staging-gke   true        v1.29.0   gke        3       5m
```

---

## Real-Time Compliance Dashboard

### Terminal Dashboard

```bash
# Launch interactive dashboard
kspec dashboard --watch

# Output:
# ┌─────────────────────────────────────┐
# │ kspec Fleet Dashboard               │
# ├─────────────────────────────────────┤
# │ Fleet Summary                       │
# │   Total Clusters: 3                 │
# │   Average Compliance: 94%           │
# │   Clusters with Drift: 1            │
# ├─────────────────────────────────────┤
# │ Cluster Details                     │
# │ NAME         SCORE  DRIFT  LAST SCAN│
# │ my-cluster   95%    No     1m ago   │
# │ prod-eks     92%    Yes    2m ago   │
# │ staging-gke  98%    No     1m ago   │
# └─────────────────────────────────────┘
```

### Web Dashboard

```bash
# Port-forward to web dashboard
kubectl port-forward -n kspec-system svc/kspec-dashboard 8080:8080

# Open http://localhost:8080
```

Features:
- Fleet-wide compliance metrics
- Per-cluster compliance scores
- Failed checks by cluster
- Drift event history
- Auto-refresh every 30s

---

## Automatic Drift Detection & Remediation

The operator automatically detects and fixes drift every 5 minutes.

### View Drift Reports

```bash
# List drift events
kubectl get driftreport -n kspec-system

# View latest drift report
kubectl get driftreport -n kspec-system \
  -l kspec.io/cluster-spec=my-cluster \
  --sort-by=.metadata.creationTimestamp \
  -o yaml | tail -1
```

### What Gets Auto-Remediated?

| Drift Type | Detection | Remediation |
|------------|-----------|-------------|
| **Missing Policy** | Policy in spec but not in cluster | ✅ Auto-created |
| **Modified Policy** | Policy changed from spec | ✅ Auto-updated |
| **Extra Policy** | Policy in cluster but not in spec | ⚠️ Reported only |
| **Compliance Violation** | New check failure | ⚠️ Reported, manual fix |

### Enable/Disable Auto-Remediation

```yaml
# For remote clusters, control enforcement
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
metadata:
  name: prod-eks
spec:
  # ...
  allowEnforcement: true   # Enable auto-remediation
  # allowEnforcement: false  # Read-only mode
```

---

## Monitoring & Observability

### Prometheus Metrics

The operator exposes metrics at `:8080/metrics`:

```promql
# Compliance score per cluster
kspec_compliance_score{cluster="my-cluster"}

# Drift detection
kspec_drift_detected{cluster="my-cluster"}

# Scan performance
kspec_scan_duration_seconds{cluster="my-cluster"}
```

### Health Checks

```bash
# Liveness probe
curl http://kspec-operator:8081/healthz

# Readiness probe
curl http://kspec-operator:8081/readyz
```

### Structured Logs

```bash
# View operator logs
kubectl logs -n kspec-system deployment/kspec-operator -f

# Filter for specific cluster
kubectl logs -n kspec-system deployment/kspec-operator -f \
  | grep "clusterspec=my-cluster"

# View audit events only
kubectl logs -n kspec-system deployment/kspec-operator -f \
  | jq 'select(.audit==true)'
```

---

## Common Operations

### Update Compliance Requirements

```bash
# Edit ClusterSpecification
kubectl edit clusterspec my-cluster -n kspec-system

# The operator automatically:
# 1. Detects the change (observedGeneration increments)
# 2. Re-runs compliance scan
# 3. Updates policies if needed
# 4. Creates new ComplianceReport
```

### Pause/Resume Monitoring

```bash
# Pause: Delete ClusterSpecification (keeps history)
kubectl delete clusterspec my-cluster -n kspec-system

# Resume: Re-apply specification
kubectl apply -f my-cluster-spec.yaml
```

### Force Immediate Scan

```bash
# Add annotation to trigger reconciliation
kubectl annotate clusterspec my-cluster -n kspec-system \
  kspec.io/reconcile="$(date +%s)" --overwrite
```

### Export Compliance Reports

```bash
# Get latest report as YAML
kubectl get compliancereport -n kspec-system \
  -l kspec.io/cluster-spec=my-cluster \
  --sort-by=.metadata.creationTimestamp \
  -o yaml | tail -1 > compliance-report.yaml

# Convert to JSON for processing
kubectl get compliancereport <name> -n kspec-system -o json \
  | jq '.spec.results[] | select(.status=="Fail")'
```

---

## Troubleshooting

### Operator Not Starting

```bash
# Check deployment status
kubectl get deployment -n kspec-system
kubectl describe deployment kspec-operator -n kspec-system

# Check CRDs installed
kubectl get crd | grep kspec.io

# Check RBAC
kubectl get serviceaccount kspec-operator -n kspec-system
```

### ClusterSpec Stuck in Pending

```bash
# Check status conditions
kubectl get clusterspec my-cluster -n kspec-system -o yaml \
  | yq '.status.conditions'

# Check operator logs
kubectl logs -n kspec-system deployment/kspec-operator \
  | grep "my-cluster"
```

### Remote Cluster Unreachable

```bash
# Check ClusterTarget status
kubectl get clustertarget prod-eks -n kspec-system -o yaml

# Test credentials manually
kubectl --kubeconfig=<extracted-kubeconfig> get nodes

# Check secret exists
kubectl get secret prod-eks-kubeconfig -n kspec-system
```

### Policies Not Enforced

```bash
# Check Kyverno installed
kubectl get deployment -n kyverno

# Check allowEnforcement flag
kubectl get clustertarget <name> -n kspec-system \
  -o jsonpath='{.spec.allowEnforcement}'

# View policy enforcement errors
kubectl logs -n kspec-system deployment/kspec-operator \
  | grep -i "policy\|kyverno"
```

---

## Advanced Configuration

### Custom Scan Intervals

```yaml
# For remote clusters
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
spec:
  scanInterval: 10m  # Scan every 10 minutes (default: 5m)
```

### Report Retention

By default, the operator keeps the last 30 reports per ClusterSpecification.

To change, modify:
```go
// controllers/clusterspecification_controller.go
const MaxReportsToKeep = 50
```

### High Availability

```yaml
# Run multiple replicas with leader election
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kspec-operator
spec:
  replicas: 3  # HA setup
  template:
    spec:
      containers:
      - name: manager
        args:
        - --leader-elect  # Enable leader election
```

---

## Migration from CLI/CronJob

If you're using the CLI with CronJob monitoring:

### Before (CronJob)
```bash
kspec init
kubectl apply -f deploy/drift/
```

### After (Operator)
```bash
# Install operator
kubectl apply -k github.com/cloudcwfranck/kspec/config/default

# Convert spec file to CR
cat cluster-spec.yaml # Your existing spec
# Copy spec: fields into ClusterSpecification CR

# Apply CR
kubectl apply -f clusterspec-cr.yaml

# Remove CronJob
kubectl delete -f deploy/drift/
```

---

## Next Steps

- **[API Reference](./API_REFERENCE.md)** - Complete CRD documentation
- **[Development Guide](./OPERATOR_DEVELOPMENT_GUIDE.md)** - Extend the operator
- **[GitOps Integration](../GITOPS.md)** - ArgoCD/Flux patterns
- **[Multi-Cluster Guide](./MULTI_CLUSTER.md)** - Advanced multi-cluster setup

---

## Get Help

- 📖 **Documentation**: https://kspec.dev (coming soon)
- 🐛 **Issues**: https://github.com/cloudcwfranck/kspec/issues
- 💬 **Discussions**: https://github.com/cloudcwfranck/kspec/discussions

---

**🎉 Congratulations! Your cluster compliance is now continuously monitored by the kspec operator.**

The operator will:
- ✅ Scan your cluster every 5 minutes
- ✅ Create immutable ComplianceReports for audit trail
- ✅ Detect and remediate configuration drift automatically
- ✅ Enforce security policies via Kyverno (if installed)
- ✅ Monitor multi-cluster fleet (if configured)

**Tip**: Run `kspec dashboard --watch` to see live compliance metrics across your fleet!
