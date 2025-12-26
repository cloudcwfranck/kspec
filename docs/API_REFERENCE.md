# kspec API Reference

Complete reference for all kspec Custom Resource Definitions (CRDs).

## Table of Contents

- [ClusterSpecification](#clusterspecification)
- [ClusterTarget](#clustertarget)
- [ComplianceReport](#compliancereport)
- [DriftReport](#driftreport)
- [Common Types](#common-types)

---

## ClusterSpecification

Defines compliance requirements for a Kubernetes cluster.

### API Version

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
```

### Scope

**Namespaced** - ClusterSpecification resources must be created in a namespace (typically `kspec-system`).

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `clusterRef` | [ClusterReference](#clusterreference) | No | Reference to a ClusterTarget for scanning remote clusters. If nil, scans the local cluster. |
| `kubernetes` | [KubernetesSpec](#kubernetesspec) | No | Kubernetes version constraints |
| `podSecurity` | [PodSecuritySpec](#podsecurityspec) | No | Pod Security Standards requirements |
| `network` | [NetworkSpec](#networkspec) | No | Network policy requirements |
| `workloads` | [WorkloadsSpec](#workloadsspec) | No | Workload security requirements |
| `rbac` | [RBACSpec](#rbacspec) | No | RBAC requirements |
| `admission` | [AdmissionSpec](#admissionspec) | No | Admission controller requirements |
| `observability` | [ObservabilitySpec](#observabilityspec) | No | Observability requirements |
| `compliance` | [ComplianceSpec](#compliancespec) | No | Compliance framework mappings |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase: `Pending`, `Active`, or `Failed` |
| `observedGeneration` | int64 | Latest generation observed by controller |
| `lastScanTime` | metav1.Time | Timestamp of last compliance scan |
| `complianceScore` | int | Compliance score 0-100 |
| `summary` | [ComplianceSummary](#compliancesummary) | Aggregate compliance statistics |
| `conditions` | []metav1.Condition | Standard Kubernetes conditions |

### Status Conditions

| Type | Status | Reason | Description |
|------|--------|--------|-------------|
| `Ready` | True | `CompliancePassing` | All or most checks passing |
| `Ready` | False | `ComplianceFailed` | Critical checks failing |
| `PolicyEnforced` | True | `KyvernoPoliciesDeployed` | Policies successfully deployed |
| `PolicyEnforced` | False | `KyvernoNotInstalled` | Kyverno not available |
| `DriftDetected` | True | `ConfigurationDrift` | Drift found and reported |
| `DriftDetected` | False | `NoDrift` | No drift detected |

### Example

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: production-cluster
  namespace: kspec-system
spec:
  # Scan local cluster (omit clusterRef)

  kubernetes:
    minVersion: "1.27.0"
    maxVersion: "1.30.0"
    excludedVersions:
      - "1.28.3"  # Known CVE

  podSecurity:
    enforce: restricted
    audit: restricted
    warn: restricted
    exemptions:
      namespaces:
        - kube-system
        - kspec-system

  network:
    requireNetworkPolicies: true
    defaultDenyIngress: true
    defaultDenyEgress: false
    requiredPolicies:
      - allow-dns
      - deny-metadata-server
    allowedServiceTypes:
      - ClusterIP
      - LoadBalancer
    disallowedPorts:
      - 22    # SSH
      - 3389  # RDP

  workloads:
    containers:
      required:
        - key: securityContext.runAsNonRoot
          value: "true"
        - key: securityContext.allowPrivilegeEscalation
          value: "false"
        - key: securityContext.capabilities.drop
          value: "[\"ALL\"]"
      forbidden:
        - key: securityContext.privileged
          value: "true"
        - key: hostNetwork
          value: "true"
    resources:
      requireLimits: true
      requireRequests: true
    images:
      requireDigests: true
      allowedRegistries:
        - "*.gcr.io"
        - "*.ecr.amazonaws.com"
        - "ghcr.io/myorg/*"
      blockedRegistries:
        - "docker.io/library/*"  # Unverified images

  rbac:
    minimumRules:
      - resources: ["pods"]
        verbs: ["get", "list"]
    forbiddenRules:
      - resources: ["*"]
        verbs: ["*"]  # No wildcards
      - resources: ["secrets"]
        verbs: ["*"]  # Limited secret access
    disallowDefaultServiceAccountTokens: true
    disallowClusterAdminBindings: true

  admission:
    requireWebhooks: true
    minimumValidatingWebhooks: 1
    minimumMutatingWebhooks: 0
    requiredPolicies:
      - require-pod-security-standards
      - disallow-privileged-containers

  observability:
    requireMetricsProvider: true
    allowedMetricsProviders:
      - prometheus
      - metrics-server
    requireAuditLogging: true
    auditLogConfig:
      level: Metadata

  compliance:
    frameworks:
      - nist-800-53
      - cis-kubernetes-1.8
      - pci-dss-4.0

status:
  phase: Active
  observedGeneration: 1
  lastScanTime: "2025-01-15T10:30:00Z"
  complianceScore: 92

  summary:
    totalChecks: 14
    passedChecks: 13
    failedChecks: 1
    score: 92

  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2025-01-15T10:00:00Z"
    reason: CompliancePassing
    message: "13/14 checks passed (92% compliance)"

  - type: PolicyEnforced
    status: "True"
    lastTransitionTime: "2025-01-15T10:00:00Z"
    reason: KyvernoPoliciesDeployed
    message: "7 Kyverno policies enforced"

  - type: DriftDetected
    status: "False"
    lastTransitionTime: "2025-01-15T10:30:00Z"
    reason: NoDrift
    message: "No configuration drift detected"
```

---

## ClusterTarget

Defines a remote Kubernetes cluster for multi-cluster scanning.

### API Version

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
```

### Scope

**Namespaced**

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiServerURL` | string | Yes | Kubernetes API server URL (must be HTTPS) |
| `authMode` | string | Yes | Authentication mode: `kubeconfig`, `serviceAccount`, or `token` |
| `kubeconfigSecretRef` | [SecretReference](#secretreference) | Conditional | Required if authMode=kubeconfig |
| `serviceAccountSecretRef` | [SecretReference](#secretreference) | Conditional | Required if authMode=serviceAccount |
| `tokenSecretRef` | [SecretReference](#secretreference) | Conditional | Required if authMode=token |
| `caData` | string | No | PEM-encoded CA certificate (base64) |
| `insecureSkipTLSVerify` | bool | No | Skip TLS verification (testing only, default: false) |
| `proxyURL` | string | No | HTTP proxy URL |
| `allowEnforcement` | bool | No | Allow policy enforcement on this cluster (default: false) |
| `scanInterval` | string | No | Custom scan interval (e.g., "10m", default: 5m) |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `reachable` | bool | Cluster health status |
| `lastChecked` | metav1.Time | Last health check timestamp |
| `version` | string | Kubernetes version (e.g., "v1.28.0") |
| `uid` | string | Cluster unique identifier |
| `platform` | string | Detected platform: `eks`, `gke`, `aks`, `openshift`, `vanilla`, `unknown` |
| `nodeCount` | int | Number of nodes in cluster |
| `conditions` | []metav1.Condition | Health conditions |

### Status Conditions

| Type | Status | Reason | Description |
|------|--------|--------|-------------|
| `Ready` | True | `ClusterReachable` | Cluster is accessible |
| `Ready` | False | `ClusterUnreachable` | Cannot connect to cluster |
| `CredentialsValid` | True | `AuthenticationSuccessful` | Credentials are valid |
| `CredentialsValid` | False | `AuthenticationFailed` | Invalid credentials |

### Example: Kubeconfig Auth

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: prod-cluster-kubeconfig
  namespace: kspec-system
type: Opaque
stringData:
  kubeconfig: |
    apiVersion: v1
    kind: Config
    clusters:
    - name: prod-cluster
      cluster:
        server: https://prod.example.com:6443
        certificate-authority-data: LS0tLS...
    users:
    - name: prod-admin
      user:
        client-certificate-data: LS0tLS...
        client-key-data: LS0tLS...
    contexts:
    - name: prod
      context:
        cluster: prod-cluster
        user: prod-admin
    current-context: prod
---
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
metadata:
  name: prod-cluster
  namespace: kspec-system
spec:
  apiServerURL: https://prod.example.com:6443
  authMode: kubeconfig
  kubeconfigSecretRef:
    name: prod-cluster-kubeconfig
    key: kubeconfig
  allowEnforcement: true
  scanInterval: 5m

status:
  reachable: true
  lastChecked: "2025-01-15T10:35:00Z"
  version: "v1.28.0"
  uid: "abc-123-xyz"
  platform: eks
  nodeCount: 5

  conditions:
  - type: Ready
    status: "True"
    reason: ClusterReachable
    message: "Cluster is healthy"

  - type: CredentialsValid
    status: "True"
    reason: AuthenticationSuccessful
    message: "Successfully authenticated"
```

### Example: Service Account Auth

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: prod-cluster-sa-token
  namespace: kspec-system
type: Opaque
stringData:
  token: eyJhbGciOiJSUzI1NiIsImtpZCI6Ii...
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIDfTCCAmWgAwIBAgIUX...
    -----END CERTIFICATE-----
---
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
metadata:
  name: remote-cluster
  namespace: kspec-system
spec:
  apiServerURL: https://remote.example.com:6443
  authMode: serviceAccount
  serviceAccountSecretRef:
    name: prod-cluster-sa-token
    key: token
  caData: LS0tLS1CRUdJTi...  # Base64-encoded CA cert
  allowEnforcement: false  # Read-only
```

---

## ComplianceReport

Immutable record of a compliance scan.

### API Version

```yaml
apiVersion: kspec.io/v1alpha1
kind: ComplianceReport
```

### Scope

**Namespaced**

### Spec Fields

| Field | Type | Description |
|-------|------|-------------|
| `clusterSpecRef` | [NamespacedName](#namespacedname) | Parent ClusterSpecification |
| `clusterName` | string | "local" or ClusterTarget name |
| `clusterUID` | string | Cluster unique identifier |
| `scanTime` | metav1.Time | When scan was performed |
| `summary` | [ReportSummary](#reportsummary) | Aggregate results |
| `results` | [][CheckResult](#checkresult) | Detailed check results |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Pending`, `Completed`, or `Failed` |
| `reportURL` | string | Optional external report URL |

### Example

```yaml
apiVersion: kspec.io/v1alpha1
kind: ComplianceReport
metadata:
  name: production-cluster-20250115-103000
  namespace: kspec-system
  labels:
    kspec.io/cluster-spec: production-cluster
    kspec.io/cluster: local
  ownerReferences:
  - apiVersion: kspec.io/v1alpha1
    kind: ClusterSpecification
    name: production-cluster
    uid: abc-123

spec:
  clusterSpecRef:
    name: production-cluster
    namespace: kspec-system

  clusterName: local
  clusterUID: "cluster-abc-123"
  scanTime: "2025-01-15T10:30:00Z"

  summary:
    total: 14
    passed: 13
    failed: 1
    passRate: 92

  results:
  - name: kubernetes-version
    category: kubernetes
    status: Pass
    severity: High
    message: "Kubernetes version v1.28.0 is within allowed range (1.27.0-1.30.0)"
    details: {}

  - name: pod-security-standards
    category: podSecurity
    status: Pass
    severity: Critical
    message: "Pod Security Standards enforced: restricted level"
    details:
      enforcedNamespaces: 12
      exemptedNamespaces: 2

  - name: network-policies
    category: network
    status: Fail
    severity: High
    message: "3 namespaces missing required NetworkPolicies"
    details:
      missingNamespaces:
        - dev
        - staging
        - test
    remediation: |
      Apply NetworkPolicies to namespaces:
        kubectl apply -f networkpolicy-dev.yaml
        kubectl apply -f networkpolicy-staging.yaml
        kubectl apply -f networkpolicy-test.yaml

status:
  phase: Completed
  reportURL: ""  # Optional
```

---

## DriftReport

Records detected configuration drift and remediation actions.

### API Version

```yaml
apiVersion: kspec.io/v1alpha1
kind: DriftReport
```

### Scope

**Namespaced**

### Spec Fields

| Field | Type | Description |
|-------|------|-------------|
| `clusterSpecRef` | [NamespacedName](#namespacedname) | Parent ClusterSpecification |
| `clusterName` | string | Cluster name |
| `clusterUID` | string | Cluster UID |
| `detectionTime` | metav1.Time | When drift detected |
| `driftDetected` | bool | Whether drift found |
| `severity` | string | `low`, `medium`, `high`, `critical` |
| `events` | [][DriftEvent](#driftevent) | Drift events |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Pending`, `InProgress`, `Completed`, `Failed` |
| `totalEvents` | int | Total drift events |
| `remediatedEvents` | int | Successfully remediated |
| `pendingEvents` | int | Pending remediation |

### Example

```yaml
apiVersion: kspec.io/v1alpha1
kind: DriftReport
metadata:
  name: production-cluster-drift-20250115-103500
  namespace: kspec-system
  labels:
    kspec.io/cluster-spec: production-cluster
    kspec.io/severity: high

spec:
  clusterSpecRef:
    name: production-cluster
    namespace: kspec-system

  clusterName: local
  clusterUID: "cluster-abc-123"
  detectionTime: "2025-01-15T10:35:00Z"

  driftDetected: true
  severity: high

  events:
  - type: Policy
    severity: high
    resource:
      kind: ClusterPolicy
      name: require-run-as-non-root
      namespace: ""
    driftType: deleted
    message: "ClusterPolicy 'require-run-as-non-root' was deleted from cluster"

    expected:
      apiVersion: kyverno.io/v1
      kind: ClusterPolicy
      metadata:
        name: require-run-as-non-root
      spec:
        validationFailureAction: Enforce
        # ... policy spec

    actual: null

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
      namespace: ""
    driftType: modified
    message: "ClusterPolicy 'disallow-privileged' was modified"

    expected:
      validationFailureAction: Enforce

    actual:
      validationFailureAction: Audit

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
      action: report
      status: manual-required
      error: "Manual intervention required to delete pod"

status:
  phase: Completed
  totalEvents: 3
  remediatedEvents: 2
  pendingEvents: 1
```

---

## Common Types

### ClusterReference

Reference to a ClusterTarget.

```yaml
clusterRef:
  name: prod-cluster
  namespace: kspec-system
```

### KubernetesSpec

Kubernetes version constraints.

```yaml
kubernetes:
  minVersion: "1.27.0"
  maxVersion: "1.30.0"
  excludedVersions:
    - "1.28.3"
```

### PodSecuritySpec

Pod Security Standards configuration.

```yaml
podSecurity:
  enforce: restricted      # privileged | baseline | restricted
  audit: restricted
  warn: restricted
  exemptions:
    namespaces:
      - kube-system
      - kube-public
```

### NetworkSpec

Network policy requirements.

```yaml
network:
  requireNetworkPolicies: true
  defaultDenyIngress: true
  defaultDenyEgress: false
  requiredPolicies:
    - allow-dns
    - deny-metadata-server
  allowedServiceTypes:
    - ClusterIP
    - LoadBalancer
  disallowedPorts:
    - 22
    - 3389
```

### WorkloadsSpec

Workload security requirements.

```yaml
workloads:
  containers:
    required:
      - key: securityContext.runAsNonRoot
        value: "true"
    forbidden:
      - key: securityContext.privileged
        value: "true"
  resources:
    requireLimits: true
    requireRequests: true
  images:
    requireDigests: true
    allowedRegistries:
      - "*.gcr.io"
    blockedRegistries:
      - "docker.io/library/*"
```

### RBACSpec

RBAC requirements.

```yaml
rbac:
  minimumRules:
    - resources: ["pods"]
      verbs: ["get", "list"]
  forbiddenRules:
    - resources: ["*"]
      verbs: ["*"]
  disallowDefaultServiceAccountTokens: true
  disallowClusterAdminBindings: true
```

### AdmissionSpec

Admission controller requirements.

```yaml
admission:
  requireWebhooks: true
  minimumValidatingWebhooks: 1
  minimumMutatingWebhooks: 0
  requiredPolicies:
    - require-pod-security-standards
```

### ObservabilitySpec

Observability requirements.

```yaml
observability:
  requireMetricsProvider: true
  allowedMetricsProviders:
    - prometheus
    - metrics-server
  requireAuditLogging: true
  auditLogConfig:
    level: Metadata  # None | Metadata | Request | RequestResponse
```

### ComplianceSpec

Compliance framework mappings.

```yaml
compliance:
  frameworks:
    - nist-800-53
    - cis-kubernetes-1.8
    - pci-dss-4.0
    - soc2
```

### SecretReference

Reference to a Secret.

```yaml
kubeconfigSecretRef:
  name: cluster-kubeconfig
  key: kubeconfig
```

### NamespacedName

Namespaced resource reference.

```yaml
clusterSpecRef:
  name: production-cluster
  namespace: kspec-system
```

### ComplianceSummary

Aggregate compliance statistics.

```yaml
summary:
  totalChecks: 14
  passedChecks: 13
  failedChecks: 1
  score: 92
```

### ReportSummary

Report summary.

```yaml
summary:
  total: 14
  passed: 13
  failed: 1
  passRate: 92
```

### CheckResult

Individual check result.

```yaml
- name: kubernetes-version
  category: kubernetes
  status: Pass  # Pass | Fail | Error
  severity: High  # Low | Medium | High | Critical
  message: "Kubernetes version v1.28.0 is valid"
  details:
    version: "v1.28.0"
  remediation: ""
```

### DriftEvent

Individual drift event.

```yaml
type: Policy  # Policy | Compliance | Configuration
severity: high  # low | medium | high | critical
resource:
  kind: ClusterPolicy
  name: require-run-as-non-root
  namespace: ""
driftType: deleted  # deleted | modified | violation
message: "Policy was deleted"
expected: {...}
actual: null
remediation:
  action: create  # create | update | delete | report
  status: success  # success | failed | pending | manual-required
  appliedAt: "2025-01-15T10:35:00Z"
  error: ""
```

---

## Validation Rules

### ClusterSpecification

- `phase`: Must be one of: `Pending`, `Active`, `Failed`
- `complianceScore`: Range 0-100

### ClusterTarget

- `apiServerURL`: Must be HTTPS URL (pattern: `^https://`)
- `authMode`: Must be one of: `kubeconfig`, `serviceAccount`, `token`
- `scanInterval`: Must be valid duration (e.g., "5m", "10m")

### ComplianceReport

- `status`: Must be one of: `Pass`, `Fail`, `Error`
- `severity`: Must be one of: `Low`, `Medium`, `High`, `Critical`
- `passRate`: Range 0-100

### DriftReport

- `severity`: Must be one of: `low`, `medium`, `high`, `critical`
- `type`: Must be one of: `Policy`, `Compliance`, `Configuration`
- `driftType`: Must be one of: `deleted`, `modified`, `violation`
- `action`: Must be one of: `create`, `update`, `delete`, `report`
- `status`: Must be one of: `success`, `failed`, `pending`, `manual-required`

---

## Printer Columns

Columns shown in `kubectl get` output:

### ClusterSpecification

```
NAME                PHASE   SCORE   LAST SCAN           AGE
production-cluster  Active  92      2025-01-15T10:30Z   5m
```

### ClusterTarget

```
NAME           REACHABLE   VERSION   PLATFORM   NODES   AGE
prod-cluster   true        v1.28.0   eks        5       10m
```

### ComplianceReport

```
NAME                                     CLUSTER   SCORE   SCANNED             AGE
production-cluster-20250115-103000       local     92      2025-01-15T10:30Z   5m
```

### DriftReport

```
NAME                                           CLUSTER   SEVERITY   EVENTS   AGE
production-cluster-drift-20250115-103500       local     high       3        2m
```

---

## RBAC Permissions

Recommended RBAC for users managing kspec resources:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kspec-user
rules:
# ClusterSpecification management
- apiGroups: ["kspec.io"]
  resources: ["clusterspecifications"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# ClusterTarget management
- apiGroups: ["kspec.io"]
  resources: ["clustertargets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Read-only access to reports
- apiGroups: ["kspec.io"]
  resources: ["compliancereports", "driftreports"]
  verbs: ["get", "list", "watch"]
```

---

## Best Practices

### Naming Conventions

- **ClusterSpecification**: `{environment}-cluster` (e.g., `production-cluster`)
- **ClusterTarget**: `{cluster-name}` (e.g., `prod-eks-us-east-1`)
- **Reports**: Auto-generated with timestamp

### Resource Organization

- Store all kspec resources in `kspec-system` namespace
- Use labels for filtering:
  - `environment: production`
  - `team: platform`
  - `compliance-framework: nist-800-53`

### Security

- Store kubeconfigs/tokens in Secrets
- Set `allowEnforcement: false` for read-only clusters
- Use least-privilege service accounts
- Enable `insecureSkipTLSVerify: false` in production

### Performance

- Adjust `scanInterval` based on cluster size (larger = longer intervals)
- Limit `MaxReportsToKeep` to prevent storage bloat
- Use label selectors to filter reports

---

## Additional Resources

- [Operator Quickstart](./OPERATOR_QUICKSTART.md) - Getting started guide
- [Multi-Cluster Setup](./MULTI_CLUSTER.md) - Multi-cluster patterns
- [GitOps Integration](../GITOPS.md) - CI/CD integration

---

**Questions?** File an issue at https://github.com/cloudcwfranck/kspec/issues
