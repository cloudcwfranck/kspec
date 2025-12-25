# Drift Detection & Automatic Remediation

This guide explains how to use kspec's drift detection and automatic remediation features to maintain continuous compliance in your Kubernetes clusters.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Drift Types](#drift-types)
- [Commands](#commands)
- [Deployment Options](#deployment-options)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

**Drift** occurs when your cluster's actual state deviates from the desired state defined in your specification. This can happen when:

- Policies are manually deleted or modified
- Configuration changes are made outside of GitOps workflows
- Compliance violations occur due to new workload deployments
- Cluster upgrades introduce unexpected changes

kspec provides three levels of drift management:

1. **Detection** - Identify when drift occurs
2. **Reporting** - Detailed reports of what has changed
3. **Remediation** - Automatic or manual fixing of drift

## Quick Start

### 1. One-Time Drift Detection

Detect drift immediately:

```bash
kspec drift detect --spec cluster-spec.yaml
```

**Example Output:**
```
┌─────────────────────────────────────────┐
│ kspec vdev — Drift Detection          │
└─────────────────────────────────────────┘

[DRIFT] Detected 2 drift events
Severity: high

Policy Drift: 2
Compliance Drift: 0

Drift Events:
─────────────
[high] ClusterPolicy/require-run-as-non-root: ClusterPolicy 'require-run-as-non-root' is missing from cluster
[medium] ClusterPolicy/disallow-host-namespaces: ClusterPolicy 'disallow-host-namespaces' has been modified
```

### 2. Continuous Monitoring

Monitor for drift continuously:

```bash
# Check every 5 minutes (default)
kspec drift detect --spec cluster-spec.yaml --watch

# Custom interval
kspec drift detect --spec cluster-spec.yaml --watch --watch-interval=10m
```

### 3. Automatic Remediation

Fix detected drift automatically:

```bash
# Dry-run first (recommended)
kspec drift remediate --spec cluster-spec.yaml --dry-run

# Apply fixes
kspec drift remediate --spec cluster-spec.yaml
```

### 4. View Drift History

See historical drift events:

```bash
# All history
kspec drift history --spec cluster-spec.yaml

# Last 24 hours
kspec drift history --spec cluster-spec.yaml --since=24h

# JSON output
kspec drift history --spec cluster-spec.yaml --output json
```

## Drift Types

kspec detects three types of drift:

### 1. Policy Drift

**What it detects:**
- Missing policies (policies in spec but not deployed)
- Modified policies (deployed policies differ from spec)
- Extra policies (kspec-generated policies not in spec)

**Example:**
```bash
$ kspec drift detect --spec cluster-spec.yaml --output json
{
  "events": [
    {
      "type": "policy",
      "severity": "high",
      "driftKind": "missing",
      "resource": {
        "kind": "ClusterPolicy",
        "name": "require-run-as-non-root"
      },
      "message": "ClusterPolicy 'require-run-as-non-root' is missing from cluster"
    }
  ]
}
```

**Remediation:**
- **Missing**: Automatically created
- **Modified**: Automatically updated to match spec
- **Extra**: Reported only (use `--force` to delete)

### 2. Compliance Drift

**What it detects:**
- New compliance check failures
- Degraded compliance from previous scans

**Example:**
```json
{
  "type": "compliance",
  "severity": "high",
  "driftKind": "new-violation",
  "resource": {
    "kind": "Check",
    "name": "kubernetes-version"
  },
  "message": "Cluster version 1.31.0 exceeds maximum allowed version 1.30.0"
}
```

**Remediation:**
- **Manual required** - Compliance violations cannot be auto-fixed
- kspec provides detailed remediation guidance

### 3. Configuration Drift

**What it detects:**
- Kubernetes version changes
- Cluster configuration changes

**Remediation:**
- **Manual required** - Cluster config changes require administrator intervention

## Commands

### `kspec drift detect`

Detect drift between current state and specification.

```bash
kspec drift detect --spec <file> [flags]
```

**Flags:**
- `--spec` (required) - Path to cluster specification
- `--output` - Output format: `text` (default) or `json`
- `--output-file` - Write report to file
- `--watch` - Continuous monitoring mode
- `--watch-interval` - Polling interval for watch mode (default: 5m)
- `--kubeconfig` - Path to kubeconfig file

**Examples:**

```bash
# Basic detection
kspec drift detect --spec cluster-spec.yaml

# JSON output to file
kspec drift detect --spec cluster-spec.yaml --output json --output-file drift-report.json

# Continuous monitoring (every 10 minutes)
kspec drift detect --spec cluster-spec.yaml --watch --watch-interval=10m

# With custom kubeconfig
kspec drift detect --spec cluster-spec.yaml --kubeconfig ~/.kube/prod-config
```

### `kspec drift remediate`

Remediate detected drift automatically.

```bash
kspec drift remediate --spec <file> [flags]
```

**Flags:**
- `--spec` (required) - Path to cluster specification
- `--dry-run` - Show what would be fixed without applying
- `--force` - Delete extra policies (default: report only)
- `--types` - Drift types to remediate: `policy`, `compliance`
- `--kubeconfig` - Path to kubeconfig file

**Examples:**

```bash
# Dry-run (preview changes)
kspec drift remediate --spec cluster-spec.yaml --dry-run

# Apply remediation
kspec drift remediate --spec cluster-spec.yaml

# Remediate only policy drift
kspec drift remediate --spec cluster-spec.yaml --types=policy

# Force delete extra policies
kspec drift remediate --spec cluster-spec.yaml --force
```

**Remediation Output:**
```
┌─────────────────────────────────────────┐
│ kspec vdev — Drift Remediation        │
└─────────────────────────────────────────┘

Remediation Summary:
───────────────────
Total events: 2
Remediated: 2
Failed: 0
Manual required: 0

Remediated:
  [OK] ClusterPolicy/require-run-as-non-root: Created missing policy
  [OK] ClusterPolicy/disallow-host-namespaces: Updated policy to match spec

[OK] Remediation complete
```

### `kspec drift history`

View historical drift events.

```bash
kspec drift history --spec <file> [flags]
```

**Flags:**
- `--spec` (required) - Path to cluster specification
- `--since` - Show events since duration (e.g., `24h`, `7d`)
- `--output` - Output format: `text` (default) or `json`

**Examples:**

```bash
# All history
kspec drift history --spec cluster-spec.yaml

# Last 24 hours
kspec drift history --spec cluster-spec.yaml --since=24h

# Last week in JSON
kspec drift history --spec cluster-spec.yaml --since=168h --output json
```

## Deployment Options

### Option 1: Manual Execution

Run drift detection manually or via scripts:

```bash
#!/bin/bash
# check-drift.sh
kspec drift detect --spec /path/to/spec.yaml --output json > /tmp/drift-report.json

if [ $(jq -r '.drift.summary.total' /tmp/drift-report.json) -gt 0 ]; then
  echo "Drift detected!"
  # Send alert, trigger remediation, etc.
fi
```

### Option 2: CronJob (Recommended)

Deploy as a Kubernetes CronJob for automated monitoring.

See [deploy/drift/README.md](../deploy/drift/README.md) for full deployment guide.

**Quick deployment:**

```bash
# Deploy drift monitoring
kubectl apply -k deploy/drift/

# Verify deployment
kubectl get cronjobs -n kspec-system
```

**Configuration:**

```yaml
# deploy/drift/cronjob.yaml
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
```

### Option 3: CI/CD Integration

Integrate drift detection into your CI/CD pipeline:

```yaml
# .github/workflows/drift-check.yaml
name: Drift Detection

on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  detect-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up kspec
        run: |
          curl -L https://github.com/cloudcwfranck/kspec/releases/latest/download/kspec-linux-amd64 -o kspec
          chmod +x kspec

      - name: Detect drift
        run: |
          ./kspec drift detect --spec cluster-spec.yaml --output json > drift-report.json

      - name: Upload report
        uses: actions/upload-artifact@v4
        with:
          name: drift-report
          path: drift-report.json
```

## Troubleshooting

### Drift Not Detected

**Problem:** Expected drift but none was reported.

**Solutions:**

1. **Verify policies have kspec annotation:**
   ```bash
   kubectl get clusterpolicy <name> -o yaml | grep "kspec.dev/generated"
   ```

   Only policies with `kspec.dev/generated: "true"` are tracked.

2. **Check spec matches deployed policies:**
   ```bash
   # Compare expected vs actual
   kspec enforce --spec cluster-spec.yaml --dry-run --output policies.yaml
   kubectl get clusterpolicies -o yaml > actual-policies.yaml
   diff policies.yaml actual-policies.yaml
   ```

3. **Enable verbose logging:**
   ```bash
   kspec drift detect --spec cluster-spec.yaml -v
   ```

### False Positives

**Problem:** Drift detected for expected changes.

**Solutions:**

1. **Update your spec** to match intended state
2. **Use exemptions** in your spec for acceptable variances
3. **Filter drift types**:
   ```bash
   kspec drift detect --spec cluster-spec.yaml --types=policy
   ```

### Remediation Fails

**Problem:** `kspec drift remediate` fails to fix drift.

**Solutions:**

1. **Check RBAC permissions:**
   ```bash
   kubectl auth can-i create clusterpolicies
   kubectl auth can-i update clusterpolicies
   ```

2. **Verify Kyverno is healthy:**
   ```bash
   kubectl get pods -n kyverno
   kubectl get validatingwebhookconfigurations | grep kyverno
   ```

3. **Check for resource conflicts:**
   ```bash
   # See if another controller is managing the resource
   kubectl get clusterpolicy <name> -o yaml | grep "ownerReferences"
   ```

4. **Run in dry-run first:**
   ```bash
   kspec drift remediate --spec cluster-spec.yaml --dry-run
   ```

### Performance Issues

**Problem:** Drift detection is slow on large clusters.

**Solutions:**

1. **Increase resource limits** (if running in-cluster):
   ```yaml
   resources:
     limits:
       cpu: 1000m
       memory: 512Mi
   ```

2. **Reduce scan frequency**:
   ```yaml
   schedule: "0 * * * *"  # Every hour instead of every 5 minutes
   ```

3. **Filter drift types:**
   ```bash
   # Only check policy drift (faster)
   kspec drift detect --spec cluster-spec.yaml --types=policy
   ```

## Best Practices

### 1. Start with Detection Only

Don't enable auto-remediation immediately:

```bash
# Week 1: Monitor and understand drift patterns
kspec drift detect --spec cluster-spec.yaml --watch

# Week 2: Test remediation in dry-run
kspec drift remediate --spec cluster-spec.yaml --dry-run

# Week 3: Enable auto-remediation (if confident)
kspec drift remediate --spec cluster-spec.yaml
```

### 2. Use Conservative Remediation Settings

Default settings are safe:

- ✅ **Create** missing policies
- ✅ **Update** modified policies
- ❌ **Don't delete** extra policies (unless `--force`)
- ❌ **Don't modify** workloads/configs

This prevents accidental data loss.

### 3. Monitor Drift Trends

Track drift over time to identify patterns:

```bash
# Daily drift summary
kspec drift history --spec cluster-spec.yaml --since=24h --output json \
  | jq -r '.stats'
```

Look for:
- Recurring drift (indicates process issues)
- Drift spikes (indicates incidents)
- Decreasing drift (indicates improvement)

### 4. Integrate with Alerts

Send notifications on drift detection:

```bash
#!/bin/bash
# drift-alert.sh

DRIFT_COUNT=$(kspec drift detect --spec spec.yaml --output json | jq -r '.drift.summary.total')

if [ "$DRIFT_COUNT" -gt 0 ]; then
  curl -X POST $SLACK_WEBHOOK \
    -H 'Content-Type: application/json' \
    -d "{\"text\": \"⚠️ Drift detected: $DRIFT_COUNT events\"}"
fi
```

### 5. Document Intentional Drift

If drift is expected (e.g., during migration):

1. Add comments to your spec explaining why
2. Create exemptions if possible
3. Schedule remediation after migration completes
4. Update runbooks with drift handling procedures

### 6. Regular Spec Reviews

Review and update your spec regularly:

```bash
# Monthly: Review drift history
kspec drift history --spec cluster-spec.yaml --since=720h --output json

# Identify patterns
# Update spec to match reality (or vice versa)
# Document changes in git
```

### 7. Test Before Production

Always test drift detection and remediation in non-prod first:

```bash
# Dev cluster
kspec drift detect --spec dev-spec.yaml --kubeconfig ~/.kube/dev

# Staging cluster
kspec drift detect --spec staging-spec.yaml --kubeconfig ~/.kube/staging

# Production cluster (after testing)
kspec drift detect --spec prod-spec.yaml --kubeconfig ~/.kube/prod
```

### 8. Version Control Everything

Keep all drift-related artifacts in git:

```
repo/
├── specs/
│   ├── prod-cluster-spec.yaml
│   ├── staging-cluster-spec.yaml
│   └── dev-cluster-spec.yaml
├── deploy/
│   └── drift/
│       ├── cronjob.yaml
│       ├── rbac.yaml
│       └── configmap.yaml
└── docs/
    └── drift-runbook.md
```

## Severity Levels

Drift events are classified by severity:

| Severity | Description | Example |
|----------|-------------|---------|
| **Critical** | Severe security risk | Security policy deleted |
| **High** | Major compliance violation | Required policy missing |
| **Medium** | Moderate issue | Policy modified but still functional |
| **Low** | Minor variance | Extra policy exists but causes no harm |

**Severity-based filtering:**

```bash
# Only show critical and high severity
kspec drift detect --spec spec.yaml --output json \
  | jq '.events[] | select(.severity == "critical" or .severity == "high")'
```

## Drift Event Structure

Understanding the drift report format:

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "spec": {
    "name": "production-cluster",
    "version": "1.0.0"
  },
  "drift": {
    "detected": true,
    "severity": "high",
    "types": ["policy"],
    "summary": {
      "total": 1,
      "policies": 1,
      "compliance": 0
    }
  },
  "events": [
    {
      "timestamp": "2025-01-15T10:30:00Z",
      "type": "policy",
      "severity": "high",
      "resource": {
        "kind": "ClusterPolicy",
        "name": "require-run-as-non-root"
      },
      "driftKind": "missing",
      "message": "ClusterPolicy 'require-run-as-non-root' is missing from cluster",
      "expected": {
        "apiVersion": "kyverno.io/v1",
        "kind": "ClusterPolicy",
        "metadata": {
          "name": "require-run-as-non-root"
        }
      },
      "actual": null,
      "remediation": {
        "action": "create",
        "status": "remediated",
        "timestamp": "2025-01-15T10:30:05Z"
      }
    }
  ]
}
```

## Next Steps

1. **Test drift detection**: Run on a development cluster first
2. **Establish baseline**: Deploy policies and verify no drift
3. **Test remediation**: Manually create drift and verify auto-fix
4. **Deploy monitoring**: Set up CronJob for continuous detection
5. **Integrate alerts**: Connect to your monitoring system
6. **Document runbooks**: Create procedures for handling drift

## Related Documentation

- [CronJob Deployment Guide](../deploy/drift/README.md)
- [kspec CLI Reference](../README.md)
- [Policy Enforcement Guide](../README.md#4-enforce-policies-prevent-future-violations)
- [Troubleshooting Guide](../README.md#troubleshooting)

## Getting Help

- **Issues**: https://github.com/cloudcwfranck/kspec/issues
- **Discussions**: https://github.com/cloudcwfranck/kspec/discussions
- **Examples**: See `specs/examples/` for sample specifications
