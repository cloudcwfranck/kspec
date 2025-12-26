# Phase 7: Kubernetes Operator

## Overview

Phase 7 transforms kspec from a CLI tool with periodic CronJob monitoring into a production-grade Kubernetes Operator with real-time compliance enforcement and drift detection.

**Key Goals:**
- **Real-time enforcement** via admission webhooks (vs. periodic CronJob)
- **Declarative configuration** via Kubernetes CRDs
- **Automatic remediation** via controller reconciliation loops
- **Zero infrastructure cost** (runs in-cluster)
- **Cloud-agnostic** (works on any Kubernetes cluster)
- **Enterprise-ready** (foundation for Phase 8 Control Plane)

## Why Phase 7?

### Current State (Phase 6)
```
User runs: kspec init
    ↓
Creates: cluster-spec.yaml
    ↓
Deploys: CronJob (runs every 5-15 minutes)
    ↓
Limitations:
  - Reactive (not real-time)
  - Non-compliant pods can run until next scan
  - Manual spec file management
  - No GitOps integration
```

### After Phase 7
```
User runs: kubectl apply -f clusterspec.yaml
    ↓
Operator:
  - Validates workloads in real-time (admission webhook)
  - Continuously reconciles state (controller)
  - Auto-remediates drift immediately
  - GitOps-native (Argo/Flux compatible)
  - Blocks non-compliant pods before creation
```

## Use Cases

### 1. Real-Time Admission Control
**Scenario**: Developer tries to deploy privileged pod in production namespace
**Current**: Pod runs until next CronJob scan (5-15 min delay)
**Phase 7**: Pod blocked immediately by validating webhook

### 2. GitOps Integration
**Scenario**: Security team wants to manage compliance via Git
**Current**: Manual `kspec enforce` commands
**Phase 7**: ClusterSpec in Git → ArgoCD syncs → Operator enforces

### 3. Instant Drift Remediation
**Scenario**: Someone deletes a Kyverno policy
**Current**: Detected at next CronJob run (5-15 min delay)
**Phase 7**: Controller detects and restores within seconds

### 4. Multi-Cluster Management (Foundation)
**Scenario**: Manage 50 clusters with same security posture
**Current**: Run kspec CLI 50 times
**Phase 7**: Foundation for Phase 8 control plane (single dashboard)

## Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│  Kubernetes Cluster                                         │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ kspec-operator (Deployment)                            │ │
│  │                                                        │ │
│  │  ┌──────────────────────────────────────────────┐    │ │
│  │  │ Controller Manager (leader-elected)          │    │ │
│  │  │                                              │    │ │
│  │  │  Reconcilers:                                │    │ │
│  │  │  ├─ ClusterSpecReconciler                    │    │ │
│  │  │  ├─ ComplianceReportReconciler               │    │ │
│  │  │  └─ DriftReportReconciler                    │    │ │
│  │  │                                              │    │ │
│  │  │  Webhooks:                                   │    │ │
│  │  │  ├─ Validating (reject non-compliant)        │    │ │
│  │  │  └─ Mutating (inject security defaults)      │    │ │
│  │  └──────────────────────────────────────────────┘    │ │
│  └────────────────────────────────────────────────────────┘ │
│           │            │            │                        │
│           ↓            ↓            ↓                        │
│   ┌─────────────┐ ┌─────────┐ ┌──────────┐                 │
│   │ pkg/scanner │ │pkg/     │ │pkg/drift │  (REUSE 80%!)   │
│   │  (checks)   │ │enforcer │ │(detector)│                 │
│   └─────────────┘ └─────────┘ └──────────┘                 │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Custom Resources (CRDs)                                │ │
│  │                                                        │ │
│  │ ┌──────────────────────┐ ┌─────────────────────────┐  │ │
│  │ │ClusterSpecification  │ │ComplianceReport        │  │ │
│  │ │(user creates)        │ │(operator creates)      │  │ │
│  │ └──────────────────────┘ └─────────────────────────┘  │ │
│  │                                                        │ │
│  │ ┌──────────────────────┐                              │ │
│  │ │DriftReport           │                              │ │
│  │ │(operator creates)    │                              │ │
│  │ └──────────────────────┘                              │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Workloads                                              │ │
│  │ Pods, Deployments, etc. → Validated by webhooks       │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Code Reuse Analysis

| Component | Existing Code | Operator Usage | Reuse % |
|-----------|---------------|----------------|---------|
| **Scanner checks** | pkg/scanner/checks/*.go | Validation webhook | 100% |
| **Policy enforcement** | pkg/enforcer/*.go | ClusterSpec reconciler | 100% |
| **Drift detection** | pkg/drift/detector.go | Reconciliation loop | 100% |
| **Spec schema** | pkg/spec/schema.go | CRD definition | 95% |
| **RBAC manifests** | deploy/drift/rbac.yaml | Operator ServiceAccount | 80% |
| **Reporting** | pkg/reporter/*.go | ComplianceReport CR | 90% |
| **Total** | ~15,000 lines | | **~80%** |

**New code needed:** ~3,000 lines
- Controller reconcilers: ~800 lines
- Admission webhooks: ~600 lines
- CRD definitions: ~400 lines
- Helm chart: ~400 lines
- Status management: ~400 lines
- Testing: ~400 lines

## Custom Resource Definitions (CRDs)

### 1. ClusterSpecification

**Purpose:** User-defined security specification for the cluster

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: production-cluster
  namespace: kspec-system
spec:
  # Existing spec.ClusterSpecification schema (from pkg/spec/schema.go)
  kubernetes:
    minVersion: "1.27.0"
    maxVersion: "1.30.0"

  podSecurity:
    enforce: restricted
    audit: restricted
    warn: restricted

  network:
    requireNetworkPolicies: true
    defaultDenyIngress: true
    defaultDenyEgress: false

  workloads:
    containers:
      required:
        - key: securityContext.runAsNonRoot
          value: true
        - key: securityContext.allowPrivilegeEscalation
          value: false
    images:
      requireDigests: true
      allowedRegistries:
        - "*.gcr.io"
        - "*.ecr.amazonaws.com"

  rbac:
    disallowDefaultServiceAccountTokens: true
    disallowClusterAdminBindings: true

  admission:
    requireKyverno: true
    requirePolicyReports: true

  compliance:
    frameworks:
      - cis-k8s-1.8
      - pci-dss-4.0

status:
  # Operator populates this
  phase: Active  # Pending, Active, Failed
  observedGeneration: 1
  lastScanTime: "2025-01-15T10:30:00Z"
  complianceScore: 95

  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2025-01-15T10:00:00Z"
      reason: AllChecksPassed
      message: "Cluster is compliant"

    - type: PolicyEnforced
      status: "True"
      lastTransitionTime: "2025-01-15T10:00:00Z"
      reason: KyvernoPoliciesDeployed
      message: "45 policies enforced"

  summary:
    totalChecks: 48
    passedChecks: 45
    failedChecks: 3
    policiesEnforced: 45
    driftEvents: 0
```

**Controller Logic:**
```go
func (r *ClusterSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) {
    // 1. Get ClusterSpecification
    var clusterSpec kspecv1alpha1.ClusterSpecification
    r.Get(ctx, req.NamespacedName, &clusterSpec)

    // 2. Run compliance scan (REUSE pkg/scanner)
    scanner := scanner.NewScanner(r.Client, r.Checks)
    scanResult := scanner.Scan(ctx, &clusterSpec.Spec)

    // 3. Detect drift (REUSE pkg/drift)
    detector := drift.NewDetector(r.Client, r.DynamicClient)
    driftReport := detector.Detect(ctx, &clusterSpec.Spec)

    // 4. Enforce policies (REUSE pkg/enforcer)
    if driftReport.Drift.Detected {
        enforcer := enforcer.NewEnforcer(r.Client, r.DynamicClient)
        enforcer.Remediate(ctx, driftReport)
    }

    // 5. Update status
    clusterSpec.Status.ComplianceScore = scanResult.Summary.PassRate
    clusterSpec.Status.LastScanTime = time.Now()
    r.Status().Update(ctx, &clusterSpec)

    // 6. Create ComplianceReport CR
    r.createComplianceReport(ctx, &clusterSpec, scanResult)

    // Requeue after 5 minutes
    return ctrl.Result{RequeueAfter: 5*time.Minute}, nil
}
```

### 2. ComplianceReport

**Purpose:** Snapshot of compliance scan results (auditable, immutable)

```yaml
apiVersion: kspec.io/v1alpha1
kind: ComplianceReport
metadata:
  name: production-cluster-20250115-103000
  namespace: kspec-system
  labels:
    kspec.io/cluster-spec: production-cluster
    kspec.io/report-type: compliance
  ownerReferences:
    - apiVersion: kspec.io/v1alpha1
      kind: ClusterSpecification
      name: production-cluster
      uid: abc-123
spec:
  clusterSpecRef:
    name: production-cluster
    version: "1.2.0"
  scanTime: "2025-01-15T10:30:00Z"

  # Immutable - never updated after creation
  summary:
    total: 48
    passed: 45
    failed: 3
    passRate: 94

  results:
    - name: kubernetes-version
      category: kubernetes
      status: Pass
      severity: High
      message: "Kubernetes version 1.28.0 is within allowed range"

    - name: pod-security-standards
      category: podSecurity
      status: Pass
      severity: Critical
      message: "Pod Security Standards enforced: restricted"

    - name: network-policies
      category: network
      status: Fail
      severity: High
      message: "3 namespaces missing NetworkPolicies: dev, staging, test"
      details:
        missingNamespaces:
          - dev
          - staging
          - test

    - name: privileged-containers
      category: workloads
      status: Fail
      severity: Critical
      message: "2 privileged containers detected"
      details:
        violations:
          - namespace: kube-system
            pod: metrics-server-abc
            container: metrics-server

    - name: rbac-cluster-admin
      category: rbac
      status: Fail
      severity: High
      message: "1 ClusterRoleBinding grants cluster-admin"
      details:
        bindings:
          - name: admin-user-binding
            subjects:
              - kind: User
                name: admin@example.com

status:
  # Report is immutable, status only for metadata
  phase: Completed
  reportURL: "https://kspec.io/reports/abc-123"  # For Phase 8 control plane
```

**Controller Logic:**
```go
// No reconciliation - reports are immutable
// Just watch for cleanup based on retention policy
```

### 3. DriftReport

**Purpose:** Record of detected drift and remediation actions

```yaml
apiVersion: kspec.io/v1alpha1
kind: DriftReport
metadata:
  name: production-cluster-drift-20250115-103500
  namespace: kspec-system
  labels:
    kspec.io/cluster-spec: production-cluster
    kspec.io/severity: high
  ownerReferences:
    - apiVersion: kspec.io/v1alpha1
      kind: ClusterSpecification
      name: production-cluster
spec:
  clusterSpecRef:
    name: production-cluster
  detectionTime: "2025-01-15T10:35:00Z"

  driftDetected: true
  severity: high  # low, medium, high, critical

  events:
    - type: Policy  # Policy, Compliance, Configuration
      severity: high
      resource:
        kind: ClusterPolicy
        name: require-run-as-non-root
        namespace: ""
      driftType: deleted

      expected:
        apiVersion: kyverno.io/v1
        kind: ClusterPolicy
        metadata:
          name: require-run-as-non-root
        spec:
          validationFailureAction: Enforce
          rules:
            - name: check-runAsNonRoot
              # ... policy spec

      actual: null  # Policy was deleted

      remediation:
        action: create
        status: success
        appliedAt: "2025-01-15T10:35:05Z"
        error: ""

    - type: Policy
      severity: medium
      resource:
        kind: ClusterPolicy
        name: disallow-privileged
      driftType: modified

      expected:
        validationFailureAction: Enforce

      actual:
        validationFailureAction: Audit  # Someone changed it!

      remediation:
        action: update
        status: success
        appliedAt: "2025-01-15T10:35:06Z"

    - type: Compliance
      severity: high
      check: workload-security
      resource:
        kind: Pod
        name: privileged-pod
        namespace: default
      driftType: violation

      message: "Privileged pod detected in production namespace"

      remediation:
        action: report  # Manual intervention required
        status: pending

status:
  phase: Completed
  totalEvents: 3
  remediatedEvents: 2
  pendingEvents: 1
```

**Controller Logic:**
```go
// Similar to ComplianceReport - mostly immutable
// But can update remediation status as actions complete
```

## Implementation Plan

### Milestone 1: Project Setup & CRD Definitions (Week 1-2)

**Goal:** Set up operator project structure and define CRDs

**Tasks:**
- [ ] Initialize operator project with kubebuilder/operator-sdk
  ```bash
  kubebuilder init --domain kspec.io --repo github.com/cloudcwfranck/kspec
  kubebuilder create api --group kspec --version v1alpha1 --kind ClusterSpecification
  kubebuilder create api --group kspec --version v1alpha1 --kind ComplianceReport
  kubebuilder create api --group kspec --version v1alpha1 --kind DriftReport
  ```

- [ ] Define CRD types in `api/v1alpha1/`
  - Convert `pkg/spec/schema.go` → `clusterspecification_types.go`
  - Create `compliancereport_types.go`
  - Create `driftreport_types.go`
  - Add kubebuilder markers for validation, status subresource

- [ ] Generate CRD manifests
  ```bash
  make manifests
  ```

- [ ] Create controller stubs
  ```bash
  kubebuilder create controller --group kspec --version v1alpha1 --kind ClusterSpecification
  kubebuilder create controller --group kspec --version v1alpha1 --kind ComplianceReport
  ```

- [ ] Set up project structure
  ```
  kspec/
  ├── api/v1alpha1/              # CRD types (NEW)
  ├── cmd/
  │   ├── kspec/                 # Existing CLI
  │   └── manager/               # Operator manager (NEW)
  ├── controllers/               # Reconcilers (NEW)
  ├── pkg/
  │   ├── scanner/               # REUSE
  │   ├── enforcer/              # REUSE
  │   ├── drift/                 # REUSE
  │   ├── spec/                  # REUSE (migrate to api/v1alpha1)
  │   └── webhooks/              # NEW
  ├── config/                    # Kustomize manifests (NEW)
  └── charts/kspec-operator/     # Helm chart (NEW)
  ```

**Acceptance Criteria:**
- [ ] CRDs install successfully: `kubectl apply -f config/crd/`
- [ ] Can create ClusterSpecification CR
- [ ] Controller manager starts (no reconciliation logic yet)
- [ ] Unit tests for CRD types

**Estimated Effort:** 2 weeks

### Milestone 2: ClusterSpec Controller & Reconciliation (Week 3-5)

**Goal:** Implement core reconciliation logic using existing pkg/ code

**Tasks:**
- [ ] Implement `ClusterSpecReconciler.Reconcile()`
  - Watch ClusterSpecification CRs
  - Trigger on create/update/delete
  - Handle finalizers for cleanup

- [ ] Integrate pkg/scanner
  ```go
  scanner := scanner.NewScanner(r.Client, r.Checks)
  result := scanner.Scan(ctx, &clusterSpec.Spec)
  ```

- [ ] Integrate pkg/drift
  ```go
  detector := drift.NewDetector(r.Client, r.DynamicClient)
  driftReport := detector.Detect(ctx, &clusterSpec.Spec)
  ```

- [ ] Integrate pkg/enforcer
  ```go
  if driftReport.Drift.Detected {
      enforcer := enforcer.NewEnforcer(r.Client, r.DynamicClient)
      enforcer.Remediate(ctx, driftReport)
  }
  ```

- [ ] Implement status updates
  - Update `status.phase` (Pending → Active → Failed)
  - Update `status.complianceScore`
  - Update `status.conditions` (Ready, PolicyEnforced)
  - Update `status.lastScanTime`

- [ ] Create child ComplianceReport CRs
  - Set OwnerReferences for garbage collection
  - Implement retention policy (keep last 30 reports)

- [ ] Create child DriftReport CRs when drift detected

- [ ] Add finalizers for cleanup
  - Remove policies when ClusterSpec is deleted
  - Clean up child resources

**Acceptance Criteria:**
- [ ] Creating a ClusterSpecification triggers a scan
- [ ] Compliance score updates in status
- [ ] ComplianceReport CRs are created automatically
- [ ] DriftReport CRs are created when drift detected
- [ ] Policies are enforced via existing pkg/enforcer
- [ ] Reconciliation runs every 5 minutes (configurable)
- [ ] Integration tests with envtest

**Estimated Effort:** 3 weeks

### Milestone 3: Admission Webhooks (Week 6-7)

**Goal:** Real-time validation of workloads before admission

**Tasks:**
- [ ] Create validating webhook
  ```bash
  kubebuilder create webhook --group core --version v1 --kind Pod --programmatic-validation
  ```

- [ ] Implement Pod validation logic
  ```go
  func (v *PodValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
      pod := obj.(*corev1.Pod)

      // Get active ClusterSpecification
      clusterSpec := v.getClusterSpec(ctx)

      // Reuse scanner checks
      violations := v.validatePod(pod, clusterSpec)

      if len(violations) > 0 {
          return admission.Denied(violations[0].Message)
      }
      return nil
  }
  ```

- [ ] Extend webhook to other workload types
  - Deployment
  - StatefulSet
  - DaemonSet
  - Job
  - CronJob

- [ ] Create mutating webhook (optional)
  - Auto-inject security defaults
  - Add missing securityContext
  - Set runAsNonRoot: true if missing

- [ ] Configure webhook with cert-manager
  - Generate TLS certificates
  - Configure ValidatingWebhookConfiguration
  - Set up cert rotation

- [ ] Add webhook failure policy
  - `failurePolicy: Fail` (block on webhook failure)
  - `timeoutSeconds: 10`
  - Namespace exemptions (kube-system)

**Acceptance Criteria:**
- [ ] Non-compliant pods are rejected before creation
- [ ] Webhook latency <100ms (p95)
- [ ] Webhook has 99.9% availability
- [ ] Webhook fails closed (blocks on error)
- [ ] E2E tests with real cluster

**Estimated Effort:** 2 weeks

### Milestone 4: Helm Chart & Deployment (Week 8-9)

**Goal:** Package operator for easy installation

**Tasks:**
- [ ] Create Helm chart structure
  ```
  charts/kspec-operator/
  ├── Chart.yaml
  ├── values.yaml
  ├── templates/
  │   ├── deployment.yaml
  │   ├── service.yaml
  │   ├── serviceaccount.yaml
  │   ├── rbac.yaml
  │   ├── crds/  (from config/crd/)
  │   └── webhooks/
  │       ├── service.yaml
  │       ├── validatingwebhook.yaml
  │       └── certificate.yaml (cert-manager)
  ```

- [ ] Configure Helm values
  ```yaml
  # values.yaml
  image:
    repository: ghcr.io/cloudcwfranck/kspec-operator
    tag: v0.2.0

  replicaCount: 2  # HA with leader election

  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

  webhook:
    enabled: true
    certManager:
      enabled: true
    failurePolicy: Fail

  reconciliation:
    interval: 5m

  complianceReports:
    retention: 30  # Keep last 30 reports
  ```

- [ ] Add RBAC permissions
  - ClusterRole for operator
  - Read: Pods, Deployments, Namespaces, etc.
  - Write: ClusterPolicies (Kyverno)
  - Create/Update: ComplianceReports, DriftReports

- [ ] Configure cert-manager integration
  - Certificate resource for webhook TLS
  - CA injection into webhook configs

- [ ] Add deployment manifests
  - Deployment with leader election
  - Service for webhook
  - ServiceMonitor for Prometheus (optional)

- [ ] Create installation documentation
  ```bash
  # Install cert-manager (prerequisite)
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

  # Install kspec operator
  helm repo add kspec https://charts.kspec.io
  helm install kspec-operator kspec/kspec-operator

  # Create ClusterSpecification
  kubectl apply -f examples/production-cluster.yaml
  ```

**Acceptance Criteria:**
- [ ] Helm install succeeds on clean cluster
- [ ] Operator starts and becomes Ready
- [ ] Webhooks are registered and functional
- [ ] Upgrade from v0.1.0 → v0.2.0 works
- [ ] Uninstall cleans up all resources
- [ ] Documentation is clear and accurate

**Estimated Effort:** 2 weeks

### Milestone 5: Testing & Observability (Week 10)

**Goal:** Production-grade testing and monitoring

**Tasks:**
- [ ] Unit tests for controllers
  - Mock Kubernetes clients
  - Test reconciliation logic
  - Test status updates
  - Target: 80% coverage

- [ ] Integration tests with envtest
  - Test full reconciliation loop
  - Test webhook validation
  - Test child resource creation
  - Test finalizer cleanup

- [ ] E2E tests with kind cluster
  - Install operator via Helm
  - Create ClusterSpecification
  - Deploy non-compliant workload (should be blocked)
  - Deploy compliant workload (should succeed)
  - Trigger drift and verify remediation
  - Test upgrade scenario

- [ ] Add Prometheus metrics
  ```go
  var (
      reconciliationsTotal = prometheus.NewCounterVec(...)
      reconciliationDuration = prometheus.NewHistogramVec(...)
      complianceScore = prometheus.NewGaugeVec(...)
      driftEventsTotal = prometheus.NewCounterVec(...)
  )
  ```

- [ ] Add structured logging
  ```go
  log := ctrl.LoggerFrom(ctx).WithValues(
      "clusterSpec", req.NamespacedName,
      "generation", clusterSpec.Generation,
  )
  log.Info("Starting reconciliation")
  ```

- [ ] Create Grafana dashboards
  - Compliance score over time
  - Drift events
  - Webhook latency
  - Reconciliation duration

- [ ] Add health checks
  - `/healthz` - liveness probe
  - `/readyz` - readiness probe
  - Leader election status

**Acceptance Criteria:**
- [ ] All tests pass in CI/CD
- [ ] Code coverage >80%
- [ ] E2E tests run in GitHub Actions
- [ ] Metrics are exported to Prometheus
- [ ] Dashboards display correctly
- [ ] Health checks respond correctly

**Estimated Effort:** 1 week

### Milestone 6: Documentation & Examples (Week 11)

**Goal:** Complete documentation for users and contributors

**Tasks:**
- [ ] User documentation
  - Installation guide
  - Quick start tutorial
  - Configuration reference
  - Troubleshooting guide
  - Migration from CLI/CronJob

- [ ] API reference
  - CRD field descriptions
  - Status conditions reference
  - Webhook behavior documentation

- [ ] Example ClusterSpecifications
  - Production cluster (high security)
  - Development cluster (permissive)
  - Compliance cluster (CIS, PCI-DSS)
  - Multi-tenant cluster

- [ ] Architecture documentation
  - How operator works
  - Reconciliation flow diagrams
  - Webhook flow diagrams
  - Code structure

- [ ] Contributor guide
  - Development setup
  - How to add new checks
  - How to run tests
  - How to cut releases

**Acceptance Criteria:**
- [ ] New users can install and configure operator in <15 minutes
- [ ] All CRD fields are documented
- [ ] Examples cover common use cases
- [ ] Architecture is clearly explained
- [ ] Contributors can set up dev environment

**Estimated Effort:** 1 week

### Milestone 7: Release & Migration (Week 12)

**Goal:** Release v0.2.0 with operator

**Tasks:**
- [ ] Create release checklist
  - All tests passing
  - Documentation complete
  - CHANGELOG updated
  - Version bumped

- [ ] Build and publish artifacts
  - Container images (multi-arch: amd64, arm64)
  - Helm chart
  - CLI binaries (still supported)
  - GitHub release

- [ ] Migration guide
  - CronJob → Operator migration steps
  - CLI → CRD migration tool
  - Rollback procedure

- [ ] Create migration tool
  ```bash
  # Convert existing cluster-spec.yaml to ClusterSpecification CR
  kspec migrate --input cluster-spec.yaml --output clusterspec-cr.yaml

  # Uninstall CronJob
  kubectl delete -f deploy/drift/

  # Install operator
  helm install kspec-operator kspec/kspec-operator

  # Apply ClusterSpec
  kubectl apply -f clusterspec-cr.yaml
  ```

- [ ] Announcement & communication
  - Blog post
  - GitHub release notes
  - Community update
  - v0.1.0 → v0.2.0 upgrade guide

**Acceptance Criteria:**
- [ ] v0.2.0 released on GitHub
- [ ] Helm chart published
- [ ] Migration guide tested
- [ ] Community informed

**Estimated Effort:** 1 week

## Timeline Summary

| Week | Milestone | Deliverables |
|------|-----------|--------------|
| 1-2 | Project Setup | CRD definitions, project structure |
| 3-5 | Controller | ClusterSpec reconciliation, status updates |
| 6-7 | Webhooks | Validating/mutating admission webhooks |
| 8-9 | Helm Chart | Installation package, documentation |
| 10 | Testing | Unit, integration, E2E tests, metrics |
| 11 | Documentation | User guides, API reference, examples |
| 12 | Release | v0.2.0 release, migration guide |

**Total Duration:** 12 weeks (~3 months)

## Technical Decisions

### 1. Framework: Kubebuilder vs Operator SDK

**Decision:** Use **Kubebuilder**

**Reasoning:**
- More lightweight (Operator SDK is built on Kubebuilder)
- Better documentation
- Widely adopted (Kubernetes SIG)
- Easier to customize

### 2. Leader Election

**Decision:** Enable leader election (HA deployment)

**Reasoning:**
- Production-grade reliability
- Zero downtime during upgrades
- Standard practice for operators

### 3. Webhook Failure Policy

**Decision:** `failurePolicy: Fail` (fail-closed)

**Reasoning:**
- Security-first approach
- Better to block deployment than allow non-compliant workload
- Users can exempt namespaces if needed

### 4. CRD Versioning

**Decision:** Start with `v1alpha1`

**Reasoning:**
- Signals early stage
- Allows breaking changes before v1beta1/v1
- Standard Kubernetes practice

### 5. Reconciliation Interval

**Decision:** Default 5 minutes, configurable

**Reasoning:**
- Balance between real-time and resource usage
- Webhooks provide real-time validation
- Periodic reconciliation catches drift

### 6. CLI vs Operator

**Decision:** Keep both, operator is primary

**Reasoning:**
- CLI still useful for CI/CD, testing, debugging
- Operator is production deployment method
- Gradual migration path

## Testing Strategy

### Unit Tests
```bash
# Test individual reconcilers
go test ./controllers/... -v

# Test webhooks
go test ./pkg/webhooks/... -v

# Test with coverage
go test ./... -coverprofile=coverage.out
```

### Integration Tests (envtest)
```bash
# Use controller-runtime envtest
make test-integration

# Spins up local etcd + apiserver
# Tests full reconciliation without real cluster
```

### E2E Tests (kind)
```yaml
# .github/workflows/e2e-operator.yaml
- name: Create kind cluster
  run: kind create cluster

- name: Install operator
  run: helm install kspec-operator ./charts/kspec-operator

- name: Create ClusterSpec
  run: kubectl apply -f examples/production-cluster.yaml

- name: Wait for Ready
  run: kubectl wait --for=condition=Ready clusterspec/production-cluster

- name: Deploy compliant pod (should succeed)
  run: kubectl apply -f test/fixtures/compliant-pod.yaml

- name: Deploy non-compliant pod (should fail)
  run: |
    kubectl apply -f test/fixtures/privileged-pod.yaml || true
    # Verify it was rejected

- name: Verify compliance score
  run: |
    score=$(kubectl get clusterspec production-cluster -o jsonpath='{.status.complianceScore}')
    [ "$score" -gt "90" ] || exit 1
```

## Security Considerations

### 1. RBAC Permissions

**Principle:** Least privilege

```yaml
# Operator needs:
rules:
  # Read cluster state
  - apiGroups: [""]
    resources: [pods, namespaces, serviceaccounts, ...]
    verbs: [get, list, watch]

  # Write policies
  - apiGroups: [kyverno.io]
    resources: [clusterpolicies]
    verbs: [get, list, watch, create, update, delete]

  # Manage own CRDs
  - apiGroups: [kspec.io]
    resources: [clusterspecifications, compliancereports, driftreports]
    verbs: [get, list, watch, create, update, delete]

  # Update status
  - apiGroups: [kspec.io]
    resources: [clusterspecifications/status]
    verbs: [get, update, patch]
```

### 2. Webhook Security

- TLS required (cert-manager)
- Namespace exemptions (kube-system, kspec-system)
- Timeout protection (10s max)
- Fail-closed by default

### 3. Audit Trail

- All actions logged with structured logging
- ComplianceReports are immutable (audit evidence)
- DriftReports track all remediation actions

### 4. Secret Management

- Never log credentials
- Use RBAC to limit secret access
- Support external secret stores (future)

## Success Metrics

### Performance
- **Reconciliation latency**: <30s (p95)
- **Webhook latency**: <100ms (p95)
- **Memory usage**: <512MB per replica
- **CPU usage**: <0.5 core per replica

### Reliability
- **Operator uptime**: >99.9%
- **Webhook availability**: >99.9%
- **Drift detection rate**: 100% within 5 minutes
- **Auto-remediation success**: >99%

### Adoption
- **v0.2.0 downloads**: 1,000+ in first month
- **GitHub stars**: 500+ (from current ~100)
- **Community contributions**: 5+ external PRs
- **Migration rate**: 50% of CronJob users

## Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Webhook causes production outage | **Critical** | Low | Fail-closed, namespace exemptions, extensive testing |
| Performance issues at scale | High | Medium | Resource limits, caching, pagination |
| Breaking changes during alpha | Medium | High | Clear versioning, migration guides |
| Complex migration from CronJob | Medium | Medium | Automated migration tool, documentation |
| Kubebuilder learning curve | Low | High | Team training, documentation |

## Dependencies

### Required
- Kubernetes 1.24+ (for API server)
- cert-manager 1.13+ (for webhook certs)
- Kyverno 1.10+ (for policy enforcement)

### Optional
- Prometheus (for metrics)
- Grafana (for dashboards)
- ArgoCD/Flux (for GitOps)

## Open Questions

1. **Should we support in-place migration from CronJob?**
   - Auto-detect existing CronJob and offer upgrade
   - Or require manual migration?

2. **Should ClusterSpecification be cluster-scoped or namespaced?**
   - Cluster-scoped makes sense (cluster-wide spec)
   - But namespace-scoped allows multi-tenancy

3. **How long should we keep ComplianceReports?**
   - Default 30 reports? Or 30 days?
   - Make it configurable?

4. **Should webhooks be opt-in or opt-out?**
   - Opt-in: Safer, gradual rollout
   - Opt-out: More secure by default

5. **Do we need a separate DriftReport controller?**
   - Or just create DriftReports from ClusterSpec reconciler?

## Phase 8 Foundation

Phase 7 lays the groundwork for Phase 8 (Control Plane SaaS):

### What Phase 7 Enables:
- ✅ Operators running in customer clusters (data stays local)
- ✅ CRDs for structured data (easy to aggregate)
- ✅ Immutable reports (audit trail)
- ✅ Status fields (health monitoring)

### Phase 8 Will Add:
- Multi-cluster aggregation API
- Central dashboard (reads operator status via API)
- Advanced analytics
- SSO/SAML for enterprises
- Billing integration

**Architecture Preview:**
```
┌─────────────────────────────────────────┐
│  kspec Control Plane (Phase 8)          │
│  - Multi-cluster dashboard              │
│  - Aggregated compliance view           │
│  - Trend analysis                       │
└─────────────────────────────────────────┘
           ▲
           │ (operator reports metrics)
           │
  ┌────────┴─────────┬──────────────┐
  │                  │              │
Customer's       Customer's    Customer's
EKS Cluster      GKE Cluster   On-Prem
(Phase 7         (Phase 7      (Phase 7
 operator)        operator)     operator)
```

## Next Steps

1. **Review this plan** - Feedback and approval
2. **Set up development branch** - `claude/phase-7-operator-sicfE`
3. **Milestone 1 kickoff** - Initialize kubebuilder project
4. **Weekly check-ins** - Track progress, adjust timeline
5. **Community preview** - Alpha release at Week 8

---

**Ready to build the operator?** This comprehensive plan takes kspec from a CLI tool to a production-grade Kubernetes Operator, positioning it perfectly for commercial success.

**Estimated total effort:** 12 weeks
**Code reuse:** ~80%
**New code:** ~3,000 lines
**Infrastructure cost:** $0 (runs in-cluster)
