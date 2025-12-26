# Phase 7 Operator - Quick Start Guide

This document provides a condensed getting-started guide for building the kspec operator.

## TL;DR

**What we're building:** Kubernetes Operator for real-time security compliance
**Duration:** 12 weeks
**Code reuse:** 80% (scanner, enforcer, drift detector)
**New code:** ~3,000 lines
**Cost:** $0 infrastructure

## Week-by-Week Roadmap

### Weeks 1-2: Foundation
```bash
# Initialize project
kubebuilder init --domain kspec.io --repo github.com/cloudcwfranck/kspec

# Create CRDs
kubebuilder create api --group kspec --version v1alpha1 --kind ClusterSpecification
kubebuilder create api --group kspec --version v1alpha1 --kind ComplianceReport
kubebuilder create api --group kspec --version v1alpha1 --kind DriftReport

# Generate manifests
make manifests
```

**Deliverable:** CRDs install successfully

### Weeks 3-5: Controller Logic
```go
// controllers/clusterspecification_controller.go
func (r *ClusterSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) {
    // 1. Get ClusterSpec
    var clusterSpec kspecv1alpha1.ClusterSpecification
    r.Get(ctx, req.NamespacedName, &clusterSpec)

    // 2. Scan cluster (REUSE pkg/scanner)
    scanner := scanner.NewScanner(r.Client, r.Checks)
    result := scanner.Scan(ctx, &clusterSpec.Spec)

    // 3. Detect drift (REUSE pkg/drift)
    detector := drift.NewDetector(r.Client, r.DynamicClient)
    driftReport := detector.Detect(ctx, &clusterSpec.Spec)

    // 4. Remediate (REUSE pkg/enforcer)
    if driftReport.Drift.Detected {
        enforcer := enforcer.NewEnforcer(r.Client, r.DynamicClient)
        enforcer.Remediate(ctx, driftReport)
    }

    // 5. Update status
    clusterSpec.Status.ComplianceScore = result.Summary.PassRate
    r.Status().Update(ctx, &clusterSpec)

    return ctrl.Result{RequeueAfter: 5*time.Minute}, nil
}
```

**Deliverable:** ClusterSpec triggers scans and enforcement

### Weeks 6-7: Admission Webhooks
```go
// pkg/webhooks/pod_webhook.go
func (v *PodValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    pod := obj.(*corev1.Pod)

    // Get ClusterSpec
    clusterSpec := v.getClusterSpec(ctx)

    // Validate (REUSE scanner checks)
    violations := v.validatePod(pod, clusterSpec)

    if len(violations) > 0 {
        return admission.Denied(violations[0].Message)
    }
    return nil
}
```

**Deliverable:** Non-compliant pods blocked

### Weeks 8-9: Helm Chart
```bash
# Create chart
helm create charts/kspec-operator

# Install
helm install kspec-operator ./charts/kspec-operator

# Test
kubectl apply -f examples/production-cluster.yaml
```

**Deliverable:** One-command installation

### Week 10: Testing
```bash
# Unit tests
go test ./controllers/... -v

# Integration tests (envtest)
make test-integration

# E2E tests (kind)
make test-e2e
```

**Deliverable:** >80% test coverage

### Week 11: Documentation
- Installation guide
- Migration from CronJob guide
- API reference
- Examples

**Deliverable:** Complete user documentation

### Week 12: Release
```bash
# Build release
make release VERSION=v0.2.0

# Publish
docker push ghcr.io/cloudcwfranck/kspec-operator:v0.2.0
helm package charts/kspec-operator
```

**Deliverable:** v0.2.0 released

## Component Reuse Map

| Existing Code | Operator Usage | Lines Reused |
|---------------|----------------|--------------|
| pkg/scanner/ | Webhook validation | ~2,000 |
| pkg/enforcer/ | Reconciler enforcement | ~1,500 |
| pkg/drift/ | Reconciler drift detection | ~1,000 |
| pkg/spec/ | CRD schema | ~800 |
| deploy/drift/ | Operator RBAC | ~200 |
| **Total** | | **~5,500 lines** |

**New code needed:** ~3,000 lines
- Controllers: ~800
- Webhooks: ~600
- CRD types: ~400
- Helm chart: ~400
- Testing: ~400
- Status management: ~400

## Critical Files to Create

### 1. api/v1alpha1/clusterspecification_types.go
```go
package v1alpha1

// ClusterSpecification is the Schema for the clusterspecifications API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type ClusterSpecification struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   spec.SpecFields          `json:"spec,omitempty"`   // REUSE existing
    Status ClusterSpecificationStatus `json:"status,omitempty"`
}

type ClusterSpecificationStatus struct {
    Phase           string      `json:"phase,omitempty"`
    ComplianceScore int         `json:"complianceScore,omitempty"`
    LastScanTime    metav1.Time `json:"lastScanTime,omitempty"`
    Conditions      []metav1.Condition `json:"conditions,omitempty"`
}
```

### 2. controllers/clusterspecification_controller.go
```go
package controllers

type ClusterSpecReconciler struct {
    client.Client
    Scheme        *runtime.Scheme
    Scanner       *scanner.Scanner
    DriftDetector *drift.Detector
    Enforcer      *enforcer.Enforcer
}

func (r *ClusterSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Implementation in main plan
}
```

### 3. pkg/webhooks/pod_webhook.go
```go
package webhooks

type PodValidator struct {
    Client  client.Client
    Scanner *scanner.Scanner
}

func (v *PodValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    // Implementation in main plan
}
```

### 4. charts/kspec-operator/values.yaml
```yaml
image:
  repository: ghcr.io/cloudcwfranck/kspec-operator
  tag: v0.2.0

replicaCount: 2

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
```

## Development Workflow

### Day 1: Setup
```bash
# Clone repo
git clone https://github.com/cloudcwfranck/kspec
cd kspec

# Create feature branch
git checkout -b claude/phase-7-operator-sicfE

# Install kubebuilder
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder
sudo mv kubebuilder /usr/local/bin/

# Initialize project
kubebuilder init --domain kspec.io --repo github.com/cloudcwfranck/kspec
```

### Daily Development
```bash
# Make changes
vim controllers/clusterspecification_controller.go

# Generate manifests
make manifests

# Run tests
make test

# Test locally
make install  # Install CRDs
make run      # Run controller locally

# Create sample CR
kubectl apply -f config/samples/kspec_v1alpha1_clusterspecification.yaml
```

### Testing
```bash
# Unit tests
go test ./controllers/... -v -cover

# Integration tests with envtest
make test-integration

# E2E tests with kind
kind create cluster
make deploy
kubectl apply -f config/samples/
```

## Migration from CronJob (for users)

### Before (Phase 6 - CronJob)
```bash
# Install CLI
curl -sSL https://install.kspec.io | bash

# Run wizard
kspec init

# Deploy CronJob
kubectl apply -f deploy/drift/
```

### After (Phase 7 - Operator)
```bash
# Install operator
helm repo add kspec https://charts.kspec.io
helm install kspec-operator kspec/kspec-operator

# Migrate spec
kspec migrate --input cluster-spec.yaml --output clusterspec-cr.yaml

# Apply ClusterSpec
kubectl apply -f clusterspec-cr.yaml

# Remove CronJob
kubectl delete -f deploy/drift/
```

## Success Criteria

✅ **Week 2:** CRDs install successfully
✅ **Week 5:** ClusterSpec reconciliation works
✅ **Week 7:** Webhooks block non-compliant pods
✅ **Week 9:** Helm install succeeds
✅ **Week 10:** Tests pass with >80% coverage
✅ **Week 11:** Documentation complete
✅ **Week 12:** v0.2.0 released

## Common Pitfalls to Avoid

1. **Don't rewrite scanner/enforcer/drift code**
   - Wrap existing packages, don't duplicate

2. **Don't skip webhook testing**
   - Webhooks can cause production outages
   - Test extensively with real clusters

3. **Don't ignore status conditions**
   - Critical for debugging and monitoring
   - Users need to know why things failed

4. **Don't forget finalizers**
   - Required for cleanup on deletion
   - Prevents orphaned resources

5. **Don't hardcode values**
   - Make reconciliation interval configurable
   - Allow namespace exemptions for webhooks

## Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Operator SDK](https://sdk.operatorframework.io/)
- [Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)

## Questions?

See full plan: [PHASE_7_PLAN.md](./PHASE_7_PLAN.md)
