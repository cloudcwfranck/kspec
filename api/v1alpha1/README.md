# kspec API v1alpha1

This package contains the v1alpha1 API definitions for kspec Kubernetes operator.

## Custom Resource Definitions (CRDs)

### ClusterSpecification

The `ClusterSpecification` CRD defines the desired security and compliance state for a Kubernetes cluster.

**Scope:** Cluster-scoped

**Short Names:** `clusterspec`, `cspec`

**Example:**
```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: production-cluster
spec:
  kubernetes:
    minVersion: "1.27.0"
  podSecurity:
    enforce: restricted
  # ... more configuration
status:
  phase: Active
  complianceScore: 95
  lastScanTime: "2025-01-26T10:30:00Z"
```

**Status Fields:**
- `phase`: Current phase (Pending, Active, Failed)
- `complianceScore`: Overall compliance score 0-100
- `lastScanTime`: When the last scan was performed
- `conditions`: Standard Kubernetes conditions
- `summary`: Summary of compliance results

### ComplianceReport

The `ComplianceReport` CRD stores the results of a compliance scan.

**Scope:** Namespaced

**Short Name:** `cr`

**Example:**
```yaml
apiVersion: kspec.io/v1alpha1
kind: ComplianceReport
metadata:
  name: production-cluster-20250126
  namespace: kspec-system
spec:
  clusterSpecRef:
    name: production-cluster
  scanTime: "2025-01-26T10:30:00Z"
  summary:
    total: 48
    passed: 45
    failed: 3
  results:
    - name: pod-security-standards
      category: podSecurity
      status: Pass
      severity: Critical
```

Reports are immutable after creation and serve as audit evidence.

### DriftReport

The `DriftReport` CRD records detected drift from the desired state and remediation actions.

**Scope:** Namespaced

**Short Name:** `dr`

**Example:**
```yaml
apiVersion: kspec.io/v1alpha1
kind: DriftReport
metadata:
  name: production-cluster-drift-20250126
  namespace: kspec-system
spec:
  clusterSpecRef:
    name: production-cluster
  detectionTime: "2025-01-26T10:35:00Z"
  driftDetected: true
  severity: high
  events:
    - type: Policy
      severity: high
      resource:
        kind: ClusterPolicy
        name: require-run-as-non-root
      driftType: deleted
      remediation:
        action: create
        status: success
```

## Installation

Install the CRDs:

```bash
kubectl apply -k config/crd
```

Verify installation:

```bash
kubectl get crds | grep kspec.io
```

Expected output:
```
clusterspecifications.kspec.io
compliancereports.kspec.io
driftreports.kspec.io
```

## Usage

### Create a ClusterSpecification

```bash
kubectl apply -f config/samples/kspec_v1alpha1_clusterspecification.yaml
```

### View ClusterSpecification Status

```bash
kubectl get clusterspec production-cluster -o yaml
```

### List Compliance Reports

```bash
kubectl get compliancereports -n kspec-system
```

### View Drift Events

```bash
kubectl get driftreports -n kspec-system
```

## API Reference

Full API documentation is available in the CRD manifests:

- [ClusterSpecification](../../config/crd/kspec.io_clusterspecifications.yaml)
- [ComplianceReport](../../config/crd/kspec.io_compliancereports.yaml)
- [DriftReport](../../config/crd/kspec.io_driftreports.yaml)

## Development

### Generating Code

After modifying types in this package, regenerate code:

```bash
# Generate deepcopy methods
controller-gen object:headerFile=hack/boilerplate.go.txt paths="./api/v1alpha1/..."

# Generate CRD manifests
controller-gen crd:crdVersions=v1 paths="./api/v1alpha1/..." output:crd:artifacts:config=config/crd
```

### Kubebuilder Markers

The types use kubebuilder markers for code generation:

- `+kubebuilder:object:root=true` - Marks this as a root type (CRD)
- `+kubebuilder:subresource:status` - Adds status subresource
- `+kubebuilder:resource:scope=Cluster` - Sets resource scope
- `+kubebuilder:validation:*` - Adds OpenAPI validation rules
- `+kubebuilder:printcolumn:*` - Adds kubectl column output

See [Kubebuilder Book](https://book.kubebuilder.io/) for more details.
