# kspec Drift Monitor Deployment

This directory contains Kubernetes manifests for deploying automated drift detection and remediation as a CronJob.

## Overview

The drift monitor runs kspec periodically to:
1. Detect configuration drift between your cluster and specification
2. Optionally auto-remediate detected drift
3. Report drift events for monitoring and alerting

## Prerequisites

1. Kubernetes cluster (v1.26+)
2. Kyverno installed (for policy enforcement)
3. kspec container image available
4. Cluster admin access (for RBAC setup)

## Quick Start

### 1. Update the Cluster Specification

Edit `configmap.yaml` and replace the example spec with your actual cluster requirements:

```bash
# Edit the ConfigMap
vi configmap.yaml

# Or create from your existing spec file
kubectl create configmap kspec-cluster-spec \
  --from-file=cluster-spec.yaml=path/to/your/spec.yaml \
  --namespace=kspec-system \
  --dry-run=client -o yaml > configmap.yaml
```

### 2. Deploy Using kubectl

```bash
# Create namespace
kubectl apply -f namespace.yaml

# Deploy RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding)
kubectl apply -f rbac.yaml

# Deploy cluster spec
kubectl apply -f configmap.yaml

# Deploy CronJobs
kubectl apply -f cronjob.yaml
```

### 3. Deploy Using Kustomize

```bash
# Deploy everything
kubectl apply -k .

# Or build and inspect first
kubectl kustomize . | kubectl apply -f -
```

## Configuration

### Drift Detection Schedule

Edit `cronjob.yaml` to adjust the detection frequency:

```yaml
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes (default)
```

Common schedules:
- `*/5 * * * *` - Every 5 minutes
- `*/15 * * * *` - Every 15 minutes
- `0 * * * *` - Every hour
- `0 */6 * * *` - Every 6 hours
- `0 0 * * *` - Once per day at midnight

### Auto-Remediation

The remediation CronJob is **suspended by default** for safety.

To enable auto-remediation:

1. **Test in dry-run mode first**:
   ```bash
   # Verify what would be fixed
   kubectl logs -l app.kubernetes.io/component=drift-remediate -n kspec-system
   ```

2. **Remove dry-run flag** from `cronjob.yaml`:
   ```yaml
   command:
     - /usr/local/bin/kspec
     - drift
     - remediate
     - --spec=/etc/kspec/cluster-spec.yaml
     # Remove the --dry-run line
   ```

3. **Enable the CronJob**:
   ```yaml
   spec:
     suspend: false  # Change from true to false
   ```

### Container Image

Update the image in `kustomization.yaml`:

```yaml
images:
  - name: kspec
    newName: your-registry/kspec
    newTag: v1.0.0
```

Or directly in `cronjob.yaml`:

```yaml
containers:
  - name: drift-detect
    image: your-registry/kspec:v1.0.0
```

### Resource Limits

Adjust CPU/memory limits based on your cluster size:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m      # Increase for large clusters
    memory: 256Mi  # Increase for large clusters
```

## Monitoring

### View Recent Jobs

```bash
# List drift detection jobs
kubectl get jobs -n kspec-system -l app.kubernetes.io/component=drift-monitor

# List remediation jobs
kubectl get jobs -n kspec-system -l app.kubernetes.io/component=drift-remediate
```

### View Logs

```bash
# Drift detection logs
kubectl logs -n kspec-system -l app.kubernetes.io/component=drift-monitor --tail=100

# Remediation logs
kubectl logs -n kspec-system -l app.kubernetes.io/component=drift-remediate --tail=100

# Follow logs in real-time
kubectl logs -n kspec-system -l app.kubernetes.io/component=drift-monitor -f
```

### Check CronJob Status

```bash
# View CronJob configuration
kubectl get cronjobs -n kspec-system

# Describe for details
kubectl describe cronjob kspec-drift-monitor -n kspec-system
```

### Manual Trigger

Trigger a job manually without waiting for schedule:

```bash
# Drift detection
kubectl create job --from=cronjob/kspec-drift-monitor manual-drift-check -n kspec-system

# Drift remediation
kubectl create job --from=cronjob/kspec-drift-remediate manual-remediation -n kspec-system
```

## Alerting Integration

### Webhook Alerts (Future Enhancement)

To send alerts on drift detection, you can:

1. Wrap the drift command in a script
2. Parse JSON output
3. Send to webhook (Slack, PagerDuty, etc.)

Example wrapper script:

```bash
#!/bin/sh
DRIFT_REPORT=$(kspec drift detect --spec /etc/kspec/cluster-spec.yaml --output json)
DRIFT_COUNT=$(echo "$DRIFT_REPORT" | jq -r '.drift.summary.total')

if [ "$DRIFT_COUNT" -gt 0 ]; then
  # Send alert
  curl -X POST https://hooks.slack.com/services/XXX \
    -H 'Content-Type: application/json' \
    -d "{\"text\": \"Drift detected: $DRIFT_COUNT events\"}"
fi

echo "$DRIFT_REPORT"
```

## Security Considerations

### RBAC Permissions

The drift monitor requires:
- **Read** access to most cluster resources (for detection)
- **Write** access to Kyverno policies (for remediation)

Review and adjust `rbac.yaml` based on your security requirements.

### Principle of Least Privilege

For drift detection only (no remediation):
- Remove `create`, `update`, `patch` verbs from Kyverno policy rules
- Suspend the remediation CronJob

### Secrets Management

If your spec requires secrets (e.g., image pull secrets):
- Store sensitive data in Kubernetes Secrets
- Mount as environment variables or files
- Never commit secrets to git

## Troubleshooting

### CronJob Not Running

```bash
# Check if CronJob is suspended
kubectl get cronjob kspec-drift-monitor -n kspec-system -o yaml | grep suspend

# Check CronJob events
kubectl describe cronjob kspec-drift-monitor -n kspec-system

# Check for RBAC issues
kubectl auth can-i list clusterpolicies --as=system:serviceaccount:kspec-system:kspec-drift-monitor
```

### Job Failures

```bash
# Get failed job name
kubectl get jobs -n kspec-system | grep kspec-drift

# Check job logs
kubectl logs job/kspec-drift-monitor-xxxxx -n kspec-system

# Check job events
kubectl describe job kspec-drift-monitor-xxxxx -n kspec-system
```

### Image Pull Errors

```bash
# Check image pull secrets
kubectl get pods -n kspec-system

# Add image pull secret if needed
kubectl create secret docker-registry regcred \
  --docker-server=your-registry \
  --docker-username=user \
  --docker-password=pass \
  -n kspec-system

# Reference in cronjob.yaml:
# spec.jobTemplate.spec.template.spec.imagePullSecrets:
#   - name: regcred
```

### Drift Not Detected

```bash
# Verify spec is correct
kubectl exec -it deployment/kspec-drift-monitor -n kspec-system -- cat /etc/kspec/cluster-spec.yaml

# Run manually for debugging
kubectl run -it --rm debug --image=kspec:latest \
  --restart=Never \
  --serviceaccount=kspec-drift-monitor \
  -n kspec-system \
  -- kspec drift detect --spec /etc/kspec/cluster-spec.yaml
```

## Cleanup

Remove all drift monitoring resources:

```bash
# Using kubectl
kubectl delete -f cronjob.yaml
kubectl delete -f configmap.yaml
kubectl delete -f rbac.yaml
kubectl delete -f namespace.yaml

# Using kustomize
kubectl delete -k .

# Clean up job history
kubectl delete jobs -n kspec-system -l app.kubernetes.io/name=kspec
```

## Next Steps

1. **Customize the spec**: Tailor `configmap.yaml` to your requirements
2. **Test drift detection**: Run manually before automating
3. **Adjust schedule**: Set appropriate frequency for your environment
4. **Enable alerts**: Integrate with your monitoring system
5. **Review RBAC**: Ensure minimum necessary permissions
6. **Monitor performance**: Track job duration and resource usage

## Related Documentation

- [Drift Detection Guide](../../docs/DRIFT_DETECTION.md)
- [kspec CLI Reference](../../README.md)
- [Kubernetes CronJob Documentation](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/)
