# Phase 6: Drift Detection & Remediation Automation

## Overview

Phase 6 adds continuous compliance monitoring and automatic remediation to kspec, enabling:
- **Drift Detection** - Detect when cluster state deviates from specification
- **Automatic Remediation** - Auto-fix drift when detected
- **Compliance Monitoring** - Continuous scanning with alerting
- **Audit Trail** - Track all drift events and remediations

## Use Cases

### 1. Policy Drift Detection
**Scenario**: Someone manually deletes or modifies a Kyverno policy
**Expected**: kspec detects the drift and restores the policy

### 2. Compliance Drift
**Scenario**: A pod is deployed that violates the specification (e.g., privileged pod in a namespace that should block it)
**Expected**: kspec detects the violation and reports/remediates

### 3. Configuration Drift
**Scenario**: Kubernetes version upgrade violates min/max version constraints
**Expected**: kspec alerts on the drift

### 4. Continuous Monitoring
**Scenario**: Run kspec as a CronJob to continuously monitor compliance
**Expected**: Regular drift detection reports, automatic remediation

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────┐
│                    kspec drift                          │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Detector   │→ │  Evaluator   │→ │  Remediator  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│         │                 │                   │        │
│         ↓                 ↓                   ↓        │
│  ┌──────────────────────────────────────────────────┐ │
│  │              State Store / Audit Log             │ │
│  └──────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
         │                                    │
         ↓                                    ↓
  ┌─────────────┐                      ┌─────────────┐
  │  Kubernetes │                      │   Alerts    │
  │   Cluster   │                      │ (Slack/etc) │
  └─────────────┘                      └─────────────┘
```

### New Commands

#### `kspec drift detect`
Detect drift between current state and specification

```bash
# Detect drift once
kspec drift detect --spec cluster-spec.yaml

# Continuous monitoring (watch mode)
kspec drift detect --spec cluster-spec.yaml --watch

# Export drift report
kspec drift detect --spec cluster-spec.yaml --output drift-report.json
```

#### `kspec drift remediate`
Remediate detected drift

```bash
# Dry-run (show what would be fixed)
kspec drift remediate --spec cluster-spec.yaml --dry-run

# Automatic remediation
kspec drift remediate --spec cluster-spec.yaml

# Remediate specific drift types only
kspec drift remediate --spec cluster-spec.yaml --types=policies,compliance
```

#### `kspec drift history`
Show drift detection history

```bash
# Show recent drift events
kspec drift history --spec cluster-spec.yaml

# Show drift for specific resource
kspec drift history --resource=clusterpolicy/require-run-as-non-root
```

### Drift Types

| Type | Detection | Remediation |
|------|-----------|-------------|
| **Policy Drift** | Compare deployed policies vs. spec | Re-apply missing/modified policies |
| **Compliance Drift** | Re-run compliance checks | Report (manual remediation) |
| **Configuration Drift** | Compare cluster config vs. spec | Report (manual intervention) |
| **Resource Drift** | Detect non-compliant resources | Delete/modify resources (configurable) |

## Implementation Plan

### Milestone 1: Basic Drift Detection (Week 1)

**Goal**: Detect when Kyverno policies drift from specification

**Deliverables**:
- [x] `pkg/drift/detector.go` - Core drift detection logic
- [x] `pkg/drift/types.go` - Drift event types
- [x] `cmd/kspec/drift.go` - `kspec drift` command
- [x] Policy drift detection (compare deployed vs. expected)
- [x] Unit tests for drift detection

**Acceptance Criteria**:
- Can detect when a policy is deleted
- Can detect when a policy is modified
- Can detect missing policies
- Reports drift in structured format

### Milestone 2: Automatic Remediation (Week 2)

**Goal**: Automatically fix policy drift

**Deliverables**:
- [x] `pkg/drift/remediator.go` - Remediation engine
- [x] Auto-restore deleted policies
- [x] Auto-update modified policies
- [x] Dry-run mode for remediation
- [x] Remediation report generation

**Acceptance Criteria**:
- Deleted policies are restored
- Modified policies are updated to match spec
- Dry-run shows what would be fixed
- Generates remediation report

### Milestone 3: Compliance Drift Detection (Week 3)

**Goal**: Detect when compliance checks start failing

**Deliverables**:
- [x] Compliance drift detection (re-run scanner)
- [x] Compare scan results over time
- [x] Detect new violations
- [x] Trend analysis (improving/degrading)

**Acceptance Criteria**:
- Detects new compliance violations
- Shows compliance trend (pass % over time)
- Identifies which checks are failing

### Milestone 4: Continuous Monitoring (Week 4)

**Goal**: Enable continuous drift detection

**Deliverables**:
- [x] Watch mode (`--watch` flag)
- [x] Configurable polling interval
- [x] State persistence (SQLite/JSON)
- [x] Drift history storage
- [x] Alerting integration (webhook)

**Acceptance Criteria**:
- Can run continuously in watch mode
- Stores drift history
- Sends alerts on drift detection
- Performance: <100MB memory, <5% CPU

### Milestone 5: CronJob Deployment (Week 5)

**Goal**: Deploy kspec as a CronJob for automated monitoring

**Deliverables**:
- [x] Kubernetes manifests (CronJob, RBAC)
- [x] ConfigMap for specification
- [x] ServiceAccount with required permissions
- [x] Alerting configuration
- [x] Deployment documentation

**Acceptance Criteria**:
- Can deploy as CronJob
- Runs every N minutes
- Sends alerts to configured webhook
- Has proper RBAC permissions

## Configuration

### Drift Detection Config

```yaml
# drift-config.yaml
apiVersion: kspec.dev/v1
kind: DriftConfig
metadata:
  name: production-drift-detection

spec:
  # What to detect
  detection:
    policies: true           # Detect policy drift
    compliance: true         # Detect compliance drift
    configuration: false     # Skip config drift (manual only)

  # How often to check
  interval: 5m

  # Remediation settings
  remediation:
    auto: true              # Auto-remediate on drift
    dryRun: false           # Actually apply fixes
    types:
      - policies            # Auto-fix policy drift
      # - compliance       # Don't auto-fix compliance (report only)

  # Alerting
  alerts:
    enabled: true
    webhook:
      url: https://hooks.slack.com/services/XXX
      severity: warning     # Alert on warning+ severity
    email:
      enabled: false

  # Storage
  storage:
    type: sqlite
    path: /var/lib/kspec/drift.db
    retention: 30d          # Keep 30 days of history
```

### Usage

```bash
# Use drift config
kspec drift detect --spec cluster-spec.yaml --config drift-config.yaml

# Override auto-remediation
kspec drift detect --spec cluster-spec.yaml --config drift-config.yaml --no-auto-remediate
```

## Drift Report Format

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
    "types": ["policy", "compliance"],
    "summary": {
      "total": 3,
      "policies": 2,
      "compliance": 1,
      "configuration": 0
    }
  },
  "events": [
    {
      "type": "policy",
      "severity": "high",
      "resource": "ClusterPolicy/require-run-as-non-root",
      "drift": "deleted",
      "expected": {
        "name": "require-run-as-non-root",
        "validationFailureAction": "Enforce"
      },
      "actual": null,
      "remediation": {
        "action": "create",
        "status": "applied",
        "timestamp": "2025-01-15T10:30:05Z"
      }
    },
    {
      "type": "policy",
      "severity": "medium",
      "resource": "ClusterPolicy/disallow-privileged-containers",
      "drift": "modified",
      "expected": {
        "validationFailureAction": "Enforce"
      },
      "actual": {
        "validationFailureAction": "Audit"
      },
      "remediation": {
        "action": "update",
        "status": "applied",
        "timestamp": "2025-01-15T10:30:06Z"
      }
    },
    {
      "type": "compliance",
      "severity": "high",
      "check": "workload-security",
      "drift": "new-violation",
      "resource": "Pod/default/privileged-pod",
      "message": "Privileged pod detected in production namespace",
      "remediation": {
        "action": "report",
        "status": "manual-required"
      }
    }
  ]
}
```

## Testing Strategy

### Unit Tests
- Drift detection logic
- Remediation engine
- State persistence
- Alert formatting

### Integration Tests
- Policy drift detection end-to-end
- Compliance drift detection
- Remediation workflow
- Watch mode behavior

### E2E Tests
```yaml
# .github/workflows/e2e-drift.yaml
- name: Test drift detection
  steps:
    - Deploy policies
    - Manually delete a policy
    - Run drift detection
    - Verify drift was detected
    - Run remediation
    - Verify policy was restored
```

## Security Considerations

1. **RBAC Permissions**
   - Drift detection: Read-only permissions
   - Remediation: Write permissions for policies
   - Principle of least privilege

2. **Audit Trail**
   - All drift events logged
   - All remediation actions logged
   - Tamper-proof audit log

3. **Alert Security**
   - Don't include sensitive data in alerts
   - Use secure webhook URLs
   - Rate limiting on alerts

## Performance Requirements

- **Memory**: <100MB for continuous monitoring
- **CPU**: <5% average, <20% peak
- **Disk**: <10MB for 30 days of drift history
- **Network**: Minimal (only on drift detection)
- **Detection Latency**: <30 seconds from drift to detection

## Success Metrics

- **Detection Rate**: 100% of policy drift detected within 1 minute
- **False Positive Rate**: <1%
- **Remediation Success**: >99% of auto-remediation succeeds
- **Uptime**: >99.9% for continuous monitoring
- **Alert Latency**: <1 minute from detection to alert

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Auto-remediation causes outage | High | Dry-run by default, manual approval mode |
| Drift detection performance issues | Medium | Incremental scanning, caching |
| Alert fatigue | Low | Configurable thresholds, aggregation |
| Audit log growth | Low | Automatic retention cleanup |

## Rollout Plan

### Phase 1: Alpha (Internal Testing)
- Basic drift detection only
- Manual remediation
- Local testing

### Phase 2: Beta (Limited Production)
- Auto-remediation with approval
- Single cluster deployment
- 1 week monitoring

### Phase 3: GA (General Availability)
- Full auto-remediation
- Multi-cluster support
- Documentation complete

## Documentation Deliverables

- [x] Architecture documentation
- [x] User guide for drift detection
- [x] User guide for remediation
- [x] CronJob deployment guide
- [x] Troubleshooting guide
- [x] API reference

## Timeline

| Week | Milestone | Status |
|------|-----------|--------|
| 1 | Basic Drift Detection | Planning |
| 2 | Automatic Remediation | Planning |
| 3 | Compliance Drift | Planning |
| 4 | Continuous Monitoring | Planning |
| 5 | CronJob Deployment | Planning |

**Estimated Completion**: 5 weeks

## Open Questions

1. Should we support custom drift detection plugins?
2. What alerting channels should be supported? (Slack, PagerDuty, email, webhook)
3. Should drift history be stored in-cluster (ConfigMap) or external (database)?
4. Should we support drift detection for non-Kyverno resources?
5. What should the default auto-remediation policy be? (conservative vs aggressive)

## Next Steps

1. Review and approve this plan
2. Create feature branch `phase-6-drift-detection`
3. Implement Milestone 1: Basic Drift Detection
4. Write unit tests
5. Update documentation
6. Create PR for review

---

**Ready to proceed?** This plan provides a comprehensive roadmap for Phase 6. Let me know if you'd like to adjust any aspects before we begin implementation.
