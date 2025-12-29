# KSpec Observability Setup

This directory contains configuration files for comprehensive observability of the KSpec policy enforcement platform.

## Prerequisites

- Prometheus Operator (for ServiceMonitor and PrometheusRule)
- Grafana (for dashboards)
- cert-manager (for webhook TLS certificates)

## Quick Start

### 1. Install Prometheus Operator

```bash
# Using Helm
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace
```

### 2. Apply Monitoring Configuration

```bash
# Apply ServiceMonitor for metrics scraping
kubectl apply -f config/prometheus/servicemonitor.yaml

# Apply alerting rules
kubectl apply -f config/prometheus/kspec-alerts.yaml
```

### 3. Import Grafana Dashboard

1. Access Grafana UI (default: `kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80`)
2. Navigate to Dashboards → Import
3. Upload `config/grafana/kspec-dashboard.json`

## Metrics Reference

### Webhook Metrics

- `kspec_webhook_requests_total` - Total webhook requests by result
- `kspec_webhook_request_duration_seconds` - Webhook request latency histogram
- `kspec_webhook_validation_results_total` - Validation results (allowed/denied) by mode
- `kspec_circuit_breaker_tripped` - Circuit breaker status (0=normal, 1=tripped)
- `kspec_circuit_breaker_error_rate` - Current error rate (0.0-1.0)
- `kspec_circuit_breaker_total_requests` - Total requests tracked by circuit breaker
- `kspec_policy_enforcement_actions_total` - Policy enforcement actions by type

### Controller Metrics

- `kspec_reconcile_total` - Total reconciliations by controller
- `kspec_reconcile_errors_total` - Reconciliation errors by type
- `kspec_reconcile_duration_seconds` - Reconciliation duration histogram
- `kspec_scan_duration_seconds` - Compliance scan duration histogram

### Compliance Metrics

- `kspec_compliance_checks_total` - Total compliance checks per cluster
- `kspec_compliance_checks_passed` - Passed checks per cluster
- `kspec_compliance_checks_failed` - Failed checks per cluster
- `kspec_compliance_score` - Compliance score percentage (0-100)

### Drift Detection Metrics

- `kspec_drift_detected` - Whether drift is detected (1=yes, 0=no)
- `kspec_drift_events_total` - Total drift events per cluster
- `kspec_drift_events_by_type` - Drift events by type

### Cluster Health Metrics

- `kspec_cluster_target_healthy` - Cluster target health (1=healthy, 0=unhealthy)
- `kspec_cluster_target_info` - Cluster metadata (platform, version, etc.)
- `kspec_cluster_target_nodes` - Node count per cluster

### Certificate Metrics

- `kspec_certificate_provisioning_duration_seconds` - Certificate provisioning duration
- `kspec_certificate_renewal_total` - Total certificate renewals

## Alert Rules

### Critical Alerts

- **KSpecCircuitBreakerTripped** - Circuit breaker has tripped (>50% error rate)
- **KSpecCriticalComplianceScore** - Compliance score below 50%
- **KSpecClusterTargetUnhealthy** - Cluster target is unreachable

### Warning Alerts

- **KSpecHighWebhookErrorRate** - Error rate >25%
- **KSpecWebhookLatencyHigh** - p95 latency >1s
- **KSpecLowComplianceScore** - Compliance score <70%
- **KSpecDriftDetected** - Configuration drift detected
- **KSpecReconciliationFailures** - High reconciliation error rate
- **KSpecReconciliationSlow** - Slow reconciliation loops (p95 >60s)

### Info Alerts

- **KSpecPolicyEnforcementDisabled** - No ClusterSpecs in enforce mode

## Dashboard Panels

The Grafana dashboard includes:

1. **Webhook Request Rate** - Real-time webhook traffic
2. **Circuit Breaker Status** - Current circuit breaker state
3. **Error Rate** - Current error rate gauge
4. **Webhook Request Duration** - p95/p50 latency trends
5. **Validation Results** - Pie chart of allowed vs denied
6. **Compliance Score by Cluster** - Trend lines per cluster
7. **Policy Enforcement Actions** - Actions by policy and type
8. **Reconciliation Duration** - Controller performance
9. **Reconciliation Errors** - Error trends by type
10. **Drift Detection** - Current drift status
11. **Total Drift Events** - Drift event counts
12. **Cluster Targets Health** - Health status of all targets
13. **Certificate Provisioning Duration** - Cert-manager performance
14. **Active ClusterSpecs by Mode** - Distribution of enforcement modes

## Troubleshooting

### Metrics Not Appearing

1. Check ServiceMonitor is in the correct namespace:
   ```bash
   kubectl get servicemonitor -n kspec-system
   ```

2. Verify Prometheus is scraping targets:
   ```bash
   # Access Prometheus UI
   kubectl port-forward -n monitoring svc/prometheus-operated 9090:9090
   # Navigate to Status → Targets
   ```

3. Check kspec controller logs:
   ```bash
   kubectl logs -n kspec-system -l app=kspec -f
   ```

### Alerts Not Firing

1. Verify PrometheusRule is loaded:
   ```bash
   kubectl get prometheusrule -n kspec-system
   ```

2. Check Prometheus rules:
   ```bash
   # In Prometheus UI, navigate to Status → Rules
   ```

3. Verify alert configuration in Alertmanager:
   ```bash
   kubectl port-forward -n monitoring svc/prometheus-alertmanager 9093:9093
   ```

## Performance Tuning

### High Cardinality Labels

If you have many clusters, consider:

1. Increasing Prometheus retention:
   ```yaml
   prometheus:
     prometheusSpec:
       retention: 30d
       storageSpec:
         volumeClaimTemplate:
           spec:
             resources:
               requests:
                 storage: 50Gi
   ```

2. Enabling remote write for long-term storage:
   ```yaml
   prometheus:
     prometheusSpec:
       remoteWrite:
         - url: "http://remote-storage:9090/api/v1/write"
   ```

### Scrape Interval Optimization

For large deployments, increase scrape intervals:

```yaml
# In ServiceMonitor
endpoints:
  - port: metrics
    interval: 60s  # Instead of 30s
```

## Integration Examples

### Slack Notifications

```yaml
# In Alertmanager config
receivers:
  - name: 'slack-kspec'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#kspec-alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### PagerDuty Integration

```yaml
receivers:
  - name: 'pagerduty-kspec'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'
        description: '{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary }}'
```

## Advanced Queries

### Compliance Trend Analysis

```promql
# Average compliance score across all clusters (1 hour)
avg_over_time(kspec_compliance_score[1h])

# Clusters with declining compliance (>10% drop in 24h)
(kspec_compliance_score - kspec_compliance_score offset 24h) < -10
```

### Webhook Performance

```promql
# Success rate (last 5 minutes)
rate(kspec_webhook_requests_total{result="success"}[5m])
/
rate(kspec_webhook_requests_total[5m])

# Request rate by result
sum by (result) (rate(kspec_webhook_requests_total[5m]))
```

### Fleet-Wide Statistics

```promql
# Total clusters being monitored
count(kspec_cluster_target_healthy)

# Percentage of healthy clusters
sum(kspec_cluster_target_healthy) / count(kspec_cluster_target_healthy) * 100

# Total policy denials across fleet (last hour)
sum(increase(kspec_webhook_validation_results_total{result="denied"}[1h]))
```

## References

- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)
- [Prometheus Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
