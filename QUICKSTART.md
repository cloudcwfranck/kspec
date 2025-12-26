# kspec Quick Start Guide

Get kspec up and running with monitoring in **1-2 commands**.

## Installation

### Option 1: One-Command Install (Recommended)

```bash
# Install kspec operator with monitoring
./hack/quick-start.sh

# Or with Grafana included
ENABLE_GRAFANA=true ./hack/quick-start.sh
```

That's it! The operator is now running with Prometheus metrics enabled.

### Option 2: Manual Install

```bash
# Install CRDs
kubectl apply -f config/crd/

# Install operator
kubectl apply -f config/manager/

# Install samples
kubectl apply -f config/samples/
```

## View Dashboard

### Built-in CLI Dashboard (Zero Setup)

```bash
# View dashboard once
kspec dashboard

# Live updating dashboard (refreshes every 10s)
kspec dashboard --watch

# Filter by specific ClusterSpec
kspec dashboard --cluster-spec prod-baseline --watch
```

**Example Output:**
```
┌────────────────────────────────────────────────────────────────────────────┐
│ kspec Compliance Dashboard                                                 │
│ Updated: 2025-12-26 10:30:00                                              │
└────────────────────────────────────────────────────────────────────────────┘

📋 ClusterSpec: prod-baseline
─────────────────────────────────────────────────────────────────────────────
  ✅ Compliance: 94.5% (425/450 checks passed)
  🏢 Clusters:   5 total | 4 healthy | 1 unhealthy
  ✨ Drift:      No drift detected

CLUSTER            COMPLIANCE  CHECKS   DRIFT       PLATFORM  NODES  LAST SCAN  STATUS
───────            ──────────  ──────   ─────       ────────  ─────  ─────────  ──────
prod-us-east       ✓ 96.5%     82/85    ✓ None      EKS       12     5m ago     ✓
prod-eu-west       ⚠ 92.1%     94/102   ⚡ 3 events  GKE       8      3m ago     ✓
staging-dev        ✓ 100%      45/45    ✓ None      AKS       4      2m ago     ✓
test-local         ⚠ 88.3%     76/86    ⚡ 1 events  Vanilla   3      10m ago    ✓
legacy-cluster     N/A         0/0      N/A         Unknown   -      Never      ✗ Unreachable

[Press Ctrl+C to exit | Refreshing every 10s]
```

## Common Workflows

### 1. Add Clusters to Monitor

```bash
# Discover clusters from your kubeconfig
kspec cluster discover

# Add a specific cluster
kspec cluster add prod-cluster | kubectl apply -f -

# Or generate manifest to review first
kspec cluster add prod-cluster > cluster.yaml
kubectl apply -f cluster.yaml
```

### 2. Create ClusterSpec

```bash
# Use interactive wizard
kspec init

# Or apply sample
kubectl apply -f config/samples/kspec_v1alpha1_clusterspecification.yaml
```

### 3. View Compliance

```bash
# Live dashboard
kspec dashboard --watch

# Get raw reports
kubectl get compliancereports -A
kubectl get driftreports -A

# Get cluster health
kubectl get clustertargets -A
```

### 4. Access Metrics

```bash
# Port forward metrics endpoint
kubectl port-forward -n kspec-system svc/kspec-operator-metrics 8080:8080

# View metrics
curl http://localhost:8080/metrics | grep kspec_

# Key metrics:
# - kspec_compliance_score
# - kspec_drift_detected
# - kspec_cluster_target_healthy
```

### 5. Grafana Dashboard (Optional)

If you installed with `ENABLE_GRAFANA=true`:

```bash
# Port forward Grafana
kubectl port-forward -n kspec-system svc/grafana 3000:80

# Open http://localhost:3000
# Login: admin/admin123
```

Then import pre-configured dashboards or create custom ones using the kspec metrics.

## Advanced: Prometheus Queries

Once Prometheus is scraping metrics, you can query:

```promql
# Average fleet compliance score
avg(kspec_compliance_score)

# Clusters below 90% compliance
kspec_compliance_score < 90

# Count unhealthy clusters
count(kspec_cluster_target_healthy == 0)

# Drift by cluster
sum by (cluster_name) (kspec_drift_detected == 1)

# Scan duration 95th percentile
histogram_quantile(0.95, rate(kspec_scan_duration_seconds_bucket[5m]))
```

## Troubleshooting

### Dashboard shows "No ClusterSpecifications found"

Create a ClusterSpec:
```bash
kubectl apply -f config/samples/kspec_v1alpha1_clusterspecification.yaml
```

### Metrics not showing in Prometheus

Check if ServiceMonitor was created:
```bash
kubectl get servicemonitor -n kspec-system kspec-operator
```

If not, create it manually:
```bash
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kspec-operator
  namespace: kspec-system
spec:
  selector:
    matchLabels:
      app: kspec-operator
  endpoints:
  - port: metrics
    interval: 30s
EOF
```

### Operator not starting

Check logs:
```bash
kubectl logs -n kspec-system deployment/kspec-controller-manager
```

## Next Steps

- **Multi-Cluster Setup**: [docs/multi-cluster.md](docs/multi-cluster.md)
- **Custom Checks**: [docs/custom-checks.md](docs/custom-checks.md)
- **CI/CD Integration**: [docs/cicd.md](docs/cicd.md)
- **Alerting**: [docs/alerting.md](docs/alerting.md)

## Uninstall

```bash
# Delete all kspec resources
kubectl delete -f config/samples/
kubectl delete -f config/manager/
kubectl delete -f config/crd/

# Delete namespace
kubectl delete namespace kspec-system
```
