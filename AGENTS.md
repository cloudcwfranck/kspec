# AGENTS.md — kspec Project Instructions

**Product**: kspec (Kubernetes Specification Enforcer)  
**Version**: 1.0.0  
**Status**: Foundation build from scratch  
**Last Updated**: 2025-01-15

-----

## 0) Executive Summary

kspec is a Kubernetes cluster compliance enforcer that validates clusters against versioned specifications, enforces security policies, generates audit evidence, and detects configuration drift. Unlike cluster installers, kspec works with existing clusters and focuses on proving compliance rather than provisioning infrastructure.

**Core value proposition**: “Ship a cluster spec, not a platform.”

-----

## 1) Primary Outcomes (Must Be Demonstrably True)

kspec MUST provide:

### 1.1 Cluster Scanning

- **Requirement**: Read-only analysis of any Kubernetes cluster (v1.26+)
- **Proof**: `kspec scan --spec <file>` returns compliance report with pass/fail/warn status
- **Scope**: Must work against:
  - Managed clusters: EKS, AKS, GKE (no cloud API dependencies)
  - On-prem: vanilla Kubernetes, OpenShift, Rancher
  - Local: kind, k3d, minikube
- **Output**: JSON report + human-readable summary

### 1.2 Specification-Driven Validation

- **Requirement**: YAML-based cluster specification format
- **Proof**: Spec file validates against JSON schema
- **Capabilities**:
  - Kubernetes version requirements
  - Pod Security Standards enforcement levels
  - Network policy requirements
  - Workload security context rules
  - RBAC baseline
  - Admission control requirements
  - Observability baseline
- **Example**: [See Section 4.1 for schema]

### 1.3 Policy Enforcement

- **Requirement**: Deploy admission policies based on spec
- **Proof**: `kspec enforce --spec <file>` installs Kyverno policies that block non-compliant workloads
- **Scope**:
  - Generate Kyverno ClusterPolicy resources from spec
  - Support dry-run mode
  - Validate policies before deployment
  - Report existing violations

### 1.4 Compliance Evidence Generation

- **Requirement**: Machine-readable compliance reports
- **Proof**: `kspec report --format oscal` generates NIST OSCAL-compliant JSON
- **Formats**:
  - JSON (default, machine-readable)
  - OSCAL (NIST compliance framework)
  - SARIF (security scanning standard)
  - Markdown (human-readable)
- **Use case**: Auditors can import output directly

### 1.5 Drift Detection

- **Requirement**: Identify changes from approved baseline
- **Proof**: `kspec diff --baseline <file>` shows config changes since last scan
- **Capabilities**:
  - Compare current state vs. previous scan
  - Detect policy deletions
  - Identify new non-compliant workloads
  - Track spec version changes

### 1.6 Remediation Guidance

- **Requirement**: Actionable fix recommendations
- **Proof**: For each violation, provide kubectl commands or YAML patches
- **Modes**:
  - `--dry-run`: Show what would be fixed
  - `--apply`: Execute fixes automatically (with confirmation)
- **Safety**: Never delete workloads, only update policies/configs

-----

## 2) Non-Goals (Explicitly NOT in v1.0)

### Infrastructure Provisioning

- ❌ kspec does NOT create clusters
- ❌ kspec does NOT install Kubernetes components (except policies)
- ❌ kspec does NOT manage node pools or cloud resources

### GitOps Integration

- ❌ No Flux/ArgoCD dependencies
- ❌ No git repository management
- ❌ Users bring their own GitOps (kspec outputs can be committed)

### Runtime Monitoring

- ❌ No in-cluster continuous monitoring controller (v2.0 feature)
- ❌ No alerting system
- ❌ No dashboard UI

### Multi-Cluster Management

- ❌ No centralized control plane
- ❌ No cluster fleet view (run kspec per cluster)
- ❌ No cross-cluster policy sync

### Cloud Provider Features

- ❌ No cloud-specific integrations (IAM, networking, etc.)
- ❌ No cost analysis
- ❌ No cloud billing integration

-----

## 3) Design Principles (Non-Negotiable)

### 3.1 Read-First, Enforce-Second

- All scanning is read-only and safe
- Enforcement requires explicit `--enforce` flag
- Dry-run mode is default for destructive operations

### 3.2 Specification as Truth

- Cluster state is validated against versioned specs
- Specs are immutable (version changes tracked)
- No implicit defaults (everything explicit in spec)

### 3.3 Evidence-Based Compliance

- Every check produces verifiable evidence
- Reports include timestamps, cluster identity, spec version
- Evidence format is standardized (OSCAL, SARIF)

### 3.4 Policy-Engine Agnostic

- v1.0 targets Kyverno (most mature)
- Architecture supports OPA Gatekeeper in v1.1
- Policy generation is pluggable

### 3.5 Minimal Runtime Footprint

- CLI-first design (no required in-cluster components)
- Enforcement deploys only standard k8s resources (Policies, NetworkPolicies)
- No custom CRDs in v1.0

### 3.6 Offline-Capable

- Scanning works without internet access
- Spec validation is local
- Only enforcement may need image pulls

-----

## 4) Architecture

### 4.1 Specification Schema

**File**: `cluster-spec.yaml`

```yaml
apiVersion: kspec.dev/v1
kind: ClusterSpecification
metadata:
  name: <spec-name>
  version: "<semver>"  # e.g., "1.0.0"
  description: "<human-readable purpose>"
  labels:
    compliance-framework: "NIST-800-53"  # optional
    environment: "production"  # optional

spec:
  # Kubernetes version requirements
  kubernetes:
    minVersion: "1.28.0"  # inclusive
    maxVersion: "1.30.0"  # inclusive
    excludedVersions:  # optional, for known CVEs
      - "1.29.0"

  # Pod Security Standards
  podSecurity:
    enforce: "baseline"  # baseline | restricted | privileged
    audit: "restricted"
    warn: "restricted"
    exemptions:  # optional
      - namespace: "kube-system"
        level: "privileged"
        reason: "System components require host access"

  # Network policies
  network:
    defaultDeny: true  # require default-deny in all non-system namespaces
    requiredPolicies:  # policies that MUST exist
      - name: "allow-dns"
        description: "Allow DNS resolution"
      - name: "deny-metadata-server"
        description: "Block cloud metadata endpoints"
    allowedServiceTypes:  # ClusterIP, LoadBalancer, NodePort
      - ClusterIP
      - LoadBalancer
    disallowedPorts:  # optional
      - 22   # SSH
      - 3389 # RDP

  # Workload security requirements
  workloads:
    containers:
      required:  # all containers MUST have these
        - key: "securityContext.runAsNonRoot"
          value: true
        - key: "securityContext.allowPrivilegeEscalation"
          value: false
        - key: "resources.limits.memory"
          exists: true
        - key: "resources.requests.cpu"
          exists: true
      forbidden:  # containers MUST NOT have these
        - key: "securityContext.privileged"
          value: true
        - key: "hostNetwork"
          value: true
        - key: "hostPID"
          value: true
    
    # Image security
    images:
      allowedRegistries:  # optional whitelist
        - "ghcr.io"
        - "*.azurecr.io"
      blockedRegistries:  # explicit deny
        - "docker.io"  # example: block Docker Hub
      requireDigests: true  # must use sha256 digests, not tags
      requireSignatures: false  # v1.1 feature

  # RBAC requirements
  rbac:
    minimumRules:
      - apiGroup: ""
        resource: "serviceaccounts"
        verbs: ["get", "list"]
    forbiddenRules:  # rules that MUST NOT exist
      - apiGroup: "*"
        resource: "*"
        verbs: ["*"]  # no cluster-admin-like roles

  # Admission control
  admission:
    required:
      - type: "ValidatingWebhookConfiguration"
        namePattern: "kyverno-.*"
        minCount: 1
    policies:  # for Kyverno
      minCount: 5  # cluster must have at least 5 policies
      requiredPolicies:  # specific policies that must exist
        - name: "disallow-privileged-containers"
        - name: "require-resource-limits"

  # Observability requirements
  observability:
    metrics:
      required: true
      providers:  # at least one must exist
        - "prometheus"
        - "metrics-server"
    logging:
      auditLog:
        required: true
        minRetentionDays: 90
      
  # Compliance mappings (metadata for reports)
  compliance:
    frameworks:
      - name: "NIST-800-53"
        revision: "Rev 5"
        controls:
          - id: "AC-2"
            title: "Account Management"
            mappings:
              - check: "rbac.minimumRules"
              - check: "podSecurity.enforce"
          - id: "SC-7"
            title: "Boundary Protection"
            mappings:
              - check: "network.defaultDeny"
      - name: "CIS-Kubernetes-v1.8"
        controls:
          - id: "5.2.1"
            title: "Minimize admission of privileged containers"
            mappings:
              - check: "workloads.containers.forbidden.privileged"
```

### 4.2 CLI Architecture

**Language**: Go  
**Framework**: cobra (CLI), client-go (Kubernetes), viper (config)

**Commands**:

```
kspec
├── version              # Print version info
├── validate             # Validate spec file syntax
├── scan                 # Scan cluster against spec
│   ├── --spec <file>
│   ├── --output <json|text|sarif|oscal>
│   ├── --severity <critical|high|medium|low>
│   └── --namespace <ns>  # optional filter
├── enforce              # Deploy policies from spec
│   ├── --spec <file>
│   ├── --dry-run
│   ├── --confirm
│   └── --skip-install  # don't install Kyverno
├── report               # Generate compliance report
│   ├── --spec <file>
│   ├── --format <oscal|sarif|json|markdown>
│   ├── --framework <nist|cis|pci>
│   └── --output <file>
├── diff                 # Compare current vs baseline
│   ├── --spec <file>
│   ├── --baseline <scan-result.json>
│   └── --output <text|json>
├── remediate            # Fix violations
│   ├── --spec <file>
│   ├── --scan-result <file>  # from previous scan
│   ├── --dry-run
│   └── --auto-approve
└── docs                 # Generate spec documentation
    ├── --spec <file>
    └── --output <markdown|html>
```

### 4.3 Internal Package Structure

```
/
├── cmd/
│   └── kspec/
│       └── main.go
├── pkg/
│   ├── spec/
│   │   ├── schema.go          # ClusterSpecification struct
│   │   ├── validator.go       # Validate spec syntax
│   │   ├── loader.go          # Load from YAML/JSON
│   │   └── versioning.go      # Semver handling
│   ├── scanner/
│   │   ├── client.go          # Kubernetes client wrapper
│   │   ├── checks/
│   │   │   ├── kubernetes.go  # Version checks
│   │   │   ├── podsecurity.go # PSS validation
│   │   │   ├── network.go     # NetworkPolicy checks
│   │   │   ├── workloads.go   # Container security
│   │   │   ├── rbac.go        # RBAC validation
│   │   │   ├── admission.go   # Admission controller checks
│   │   │   └── observability.go # Metrics/logging
│   │   └── report.go          # ScanResult aggregation
│   ├── enforcer/
│   │   ├── kyverno/
│   │   │   ├── generator.go   # Generate Kyverno policies
│   │   │   ├── installer.go   # Install Kyverno helm chart
│   │   │   └── validator.go   # Validate policy syntax
│   │   └── enforcer.go        # Enforcement orchestrator
│   ├── reporter/
│   │   ├── json.go            # JSON output
│   │   ├── oscal.go           # OSCAL format
│   │   ├── sarif.go           # SARIF format
│   │   └── markdown.go        # Human-readable
│   ├── remediation/
│   │   ├── planner.go         # Generate remediation plan
│   │   ├── executor.go        # Apply fixes
│   │   └── dryrun.go          # Simulate fixes
│   └── diff/
│       └── comparator.go      # Baseline vs current
├── internal/
│   ├── config/
│   │   └── config.go          # CLI configuration
│   └── kubernetes/
│       └── client.go          # K8s client helpers
├── specs/
│   ├── examples/
│   │   ├── fedramp-moderate.yaml
│   │   ├── cis-benchmark.yaml
│   │   ├── nist-800-53.yaml
│   │   └── minimal.yaml
│   └── schema/
│       └── cluster-spec-v1.schema.json  # JSON Schema
├── test/
│   ├── e2e/
│   │   ├── scan_test.go
│   │   ├── enforce_test.go
│   │   └── remediate_test.go
│   ├── fixtures/
│   │   ├── compliant-cluster/
│   │   └── non-compliant-cluster/
│   └── integration/
│       └── kind/
│           └── test-cluster.yaml
├── docs/
│   ├── architecture.md
│   ├── spec-reference.md
│   ├── compliance-frameworks.md
│   └── contributing.md
├── .github/
│   └── workflows/
│       ├── ci.yaml            # Lint, test, build
│       ├── e2e.yaml           # kind-based integration tests
│       └── release.yaml       # Goreleaser
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
├── README.md
└── AGENTS.md  # This file
```

### 4.4 Scanning Engine Design

**Flow**:

1. Load cluster spec from YAML
1. Validate spec against JSON schema
1. Connect to Kubernetes cluster (via kubeconfig)
1. Run check modules in parallel:

- Kubernetes version check
- Pod Security Standards check
- Network policy check
- Workload security check
- RBAC check
- Admission controller check
- Observability check

1. Aggregate results into ScanResult struct
1. Apply severity levels (critical, high, medium, low)
1. Output in requested format

**Check Module Interface**:

```go
type Check interface {
    Name() string
    Run(ctx context.Context, client kubernetes.Interface, spec *Specification) (*CheckResult, error)
}

type CheckResult struct {
    Name      string
    Status    Status  // Pass, Fail, Warn, Skip
    Severity  Severity
    Message   string
    Evidence  map[string]interface{}  // JSON-serializable
    Remediation string  // kubectl commands or YAML
}

type Status string
const (
    StatusPass Status = "pass"
    StatusFail Status = "fail"
    StatusWarn Status = "warn"
    StatusSkip Status = "skip"
)

type Severity string
const (
    SeverityCritical Severity = "critical"
    SeverityHigh     Severity = "high"
    SeverityMedium   Severity = "medium"
    SeverityLow      Severity = "low"
)
```

**Example Check Implementation**:

```go
// pkg/scanner/checks/kubernetes.go
type KubernetesVersionCheck struct{}

func (c *KubernetesVersionCheck) Run(ctx context.Context, client kubernetes.Interface, spec *Specification) (*CheckResult, error) {
    version, err := client.Discovery().ServerVersion()
    if err != nil {
        return nil, err
    }
    
    current, _ := semver.Parse(version.GitVersion)
    min, _ := semver.Parse(spec.Spec.Kubernetes.MinVersion)
    max, _ := semver.Parse(spec.Spec.Kubernetes.MaxVersion)
    
    if current.LT(min) || current.GT(max) {
        return &CheckResult{
            Name:     "kubernetes.version",
            Status:   StatusFail,
            Severity: SeverityCritical,
            Message:  fmt.Sprintf("Cluster running %s, spec requires %s - %s", current, min, max),
            Evidence: map[string]interface{}{
                "current": current.String(),
                "required_min": min.String(),
                "required_max": max.String(),
            },
            Remediation: "Upgrade cluster to supported Kubernetes version",
        }, nil
    }
    
    return &CheckResult{
        Name:   "kubernetes.version",
        Status: StatusPass,
        Message: fmt.Sprintf("Cluster version %s is within spec range", current),
    }, nil
}
```

### 4.5 Policy Enforcement Design

**Flow**:

1. Load cluster spec
1. Check if Kyverno is installed

- If not: optionally install via Helm (requires `--install-kyverno` flag)

1. Generate Kyverno ClusterPolicy resources from spec
1. Validate generated policies (syntax check)
1. In dry-run mode: show policies without applying
1. In enforce mode: apply policies to cluster
1. Scan for existing violations
1. Report violations and remediation steps

**Policy Generation Example**:

```go
// pkg/enforcer/kyverno/generator.go
func GeneratePolicies(spec *Specification) ([]*kyvernov1.ClusterPolicy, error) {
    policies := []*kyvernov1.ClusterPolicy{}
    
    // Generate policy from workloads.containers.required
    if spec.Spec.Workloads != nil {
        for _, req := range spec.Spec.Workloads.Containers.Required {
            if req.Key == "securityContext.runAsNonRoot" && req.Value == true {
                policy := &kyvernov1.ClusterPolicy{
                    ObjectMeta: metav1.ObjectMeta{
                        Name: "require-run-as-non-root",
                    },
                    Spec: kyvernov1.Spec{
                        ValidationFailureAction: "enforce",
                        Rules: []kyvernov1.Rule{
                            {
                                Name: "check-runAsNonRoot",
                                Match: kyvernov1.MatchResources{
                                    Resources: kyvernov1.ResourceDescription{
                                        Kinds: []string{"Pod"},
                                    },
                                },
                                Validate: kyvernov1.Validation{
                                    Message: "Containers must run as non-root",
                                    Pattern: &apiextv1.JSON{
                                        Raw: []byte(`{
                                            "spec": {
                                                "securityContext": {
                                                    "runAsNonRoot": true
                                                }
                                            }
                                        }`),
                                    },
                                },
                            },
                        },
                    },
                }
                policies = append(policies, policy)
            }
        }
    }
    
    return policies, nil
}
```

### 4.6 Evidence Generation Design

**OSCAL Output Structure**:

```json
{
  "system-security-plan": {
    "uuid": "<uuid>",
    "metadata": {
      "title": "kspec Compliance Report",
      "published": "2025-01-15T10:30:00Z",
      "version": "1.0.0",
      "oscal-version": "1.0.4"
    },
    "system-characteristics": {
      "system-name": "aks-prod-eastus",
      "system-id": "<cluster-uid>",
      "description": "Kubernetes cluster scanned by kspec"
    },
    "implemented-requirements": [
      {
        "uuid": "<uuid>",
        "control-id": "AC-2",
        "description": "Account Management",
        "statements": [
          {
            "statement-id": "AC-2.a",
            "uuid": "<uuid>",
            "description": "Service account management and RBAC enforcement",
            "responsible-roles": [
              {
                "role-id": "platform-team"
              }
            ]
          }
        ],
        "props": [
          {
            "name": "implementation-status",
            "value": "implemented"
          }
        ],
        "links": [
          {
            "href": "#evidence-ac2",
            "rel": "evidence"
          }
        ]
      }
    ],
    "back-matter": {
      "resources": [
        {
          "uuid": "evidence-ac2",
          "title": "RBAC Scan Results",
          "props": [
            {
              "name": "scan-timestamp",
              "value": "2025-01-15T10:30:00Z"
            },
            {
              "name": "check-status",
              "value": "pass"
            }
          ],
          "description": "All service accounts have restricted RBAC roles",
          "rlinks": [
            {
              "href": "file://rbac-scan-20250115.json"
            }
          ]
        }
      ]
    }
  }
}
```

**SARIF Output** (for security scanners):

```json
{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "kspec",
          "version": "1.0.0",
          "informationUri": "https://kspec.dev",
          "rules": [
            {
              "id": "K8S-001",
              "shortDescription": {
                "text": "Kubernetes version out of range"
              },
              "fullDescription": {
                "text": "Cluster Kubernetes version must be within specified range"
              },
              "defaultConfiguration": {
                "level": "error"
              }
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "K8S-001",
          "level": "error",
          "message": {
            "text": "Cluster running 1.27.8, spec requires 1.28.0 - 1.30.0"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "cluster://kubernetes-version"
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
```

-----

## 5) User Experience Contract

### 5.1 Installation

```bash
# macOS
brew install kspec/tap/kspec

# Linux (amd64)
curl -sL https://github.com/cloudcwfranck/kspec/releases/latest/download/kspec-linux-amd64 -o kspec
chmod +x kspec
sudo mv kspec /usr/local/bin/

# Windows
scoop install kspec

# From source
go install github.com/cloudcwfranck/kspec/cmd/kspec@latest
```

### 5.2 Quickstart Workflow

```bash
# 1. Get example spec
kspec docs --template fedramp > cluster-spec.yaml

# 2. Validate spec syntax
kspec validate --spec cluster-spec.yaml

# 3. Scan cluster (read-only)
kspec scan --spec cluster-spec.yaml

# Output:
# ┌─────────────────────────────────────────┐
# │ kspec v1.0.0 — Compliance Report        │
# ├─────────────────────────────────────────┤
# │ Cluster: aks-prod-eastus                │
# │ Spec: fedramp-moderate v1.0.0           │
# │ Scanned: 2025-01-15T10:30:00Z           │
# └─────────────────────────────────────────┘
# 
# COMPLIANCE: 87/100 checks passed
# 
# ❌ CRITICAL (3)
#   - kubernetes.version: Running 1.27.8, required >= 1.28
#   - network.defaultDeny: 12 namespaces missing policies
#   - admission.required: Kyverno not installed
# 
# ⚠️  WARNINGS (10)
#   - workloads.resources: 45 pods missing limits
# 
# ✅ PASSED (87)
# 
# 📊 Full report: scan-result-20250115.json

# 4. Generate compliance report
kspec report --spec cluster-spec.yaml --format oscal --output fedramp-evidence.json

# 5. Enforce policies (dry-run first)
kspec enforce --spec cluster-spec.yaml --dry-run

# 6. Apply policies
kspec enforce --spec cluster-spec.yaml --confirm

# 7. Remediate violations
kspec remediate --spec cluster-spec.yaml --scan-result scan-result-20250115.json --dry-run
```

### 5.3 Output Format Examples

**JSON Output** (`--output json`):

```json
{
  "metadata": {
    "kspec_version": "1.0.0",
    "scan_time": "2025-01-15T10:30:00Z",
    "cluster": {
      "name": "aks-prod-eastus",
      "version": "1.27.8",
      "uid": "abc-123-def"
    },
    "spec": {
      "name": "fedramp-moderate",
      "version": "1.0.0"
    }
  },
  "summary": {
    "total_checks": 100,
    "passed": 87,
    "failed": 3,
    "warnings": 10,
    "skipped": 0
  },
  "results": [
    {
      "check": "kubernetes.version",
      "status": "fail",
      "severity": "critical",
      "message": "Cluster running 1.27.8, spec requires 1.28.0 - 1.30.0",
      "evidence": {
        "current": "1.27.8",
        "required_min": "1.28.0",
        "required_max": "1.30.0"
      },
      "remediation": "Upgrade cluster to Kubernetes 1.28 or later"
    }
  ]
}
```

**Text Output** (default):

```
kspec v1.0.0 — Compliance Report
─────────────────────────────────
Cluster: aks-prod-eastus
Spec: fedramp-moderate v1.0.0
Scanned: 2025-01-15 10:30:00 UTC

COMPLIANCE: 87/100 checks passed (87%)

❌ CRITICAL FAILURES (3)
─────────────────────────
[kubernetes.version] Cluster version out of range
  Current: 1.27.8
  Required: 1.28.0 - 1.30.0
  Fix: Upgrade cluster to Kubernetes 1.28+

[network.defaultDeny] Missing default-deny NetworkPolicies
  Affected namespaces: app-1, app-2, ... (12 total)
  Fix: Apply default-deny policies
    kubectl apply -f https://kspec.dev/policies/default-deny.yaml

[admission.required] Kyverno not installed
  Required: kyverno ValidatingWebhookConfiguration
  Fix: Install Kyverno
    kspec enforce --spec cluster-spec.yaml --install-kyverno

⚠️  WARNINGS (10)
─────────────────
[workloads.resources] Pods missing resource limits
  Count: 45 pods across 8 namespaces
  Fix: Add resources.limits to pod specs

✅ PASSED CHECKS (87)
─────────────────────
✓ Pod Security Standards enforced (baseline)
✓ RBAC roles follow least-privilege
✓ Observability metrics available
... (84 more)

────────────────────────────────────────────
Next steps:
1. Review failures above
2. Run: kspec enforce --spec cluster-spec.yaml --dry-run
3. Generate evidence: kspec report --format oscal

Full JSON report: scan-result-20250115.json
```

### 5.4 Configuration File

**File**: `~/.kspec/config.yaml` (optional)

```yaml
# Default kubeconfig (override with --kubeconfig)
kubeconfig: ~/.kube/config

# Default output format
output: text

# Scanner settings
scanner:
  parallel: true
  timeout: 5m
  
# Enforcer settings
enforcer:
  kyverno:
    chart: kyverno/kyverno
    version: 3.1.4
    namespace: kyverno
  dry_run_default: true

# Reporter settings
reporter:
  include_passed: false  # only show failures/warnings in output
  evidence_dir: ./evidence
```

-----

## 6) Quality Gates (Must Be Enforced)

### 6.1 Code Quality

- `go test ./... -race -cover` must pass with >80% coverage
- `golangci-lint run` must pass (config in `.golangci.yml`)
- `go vet ./...` must pass
- No `gofmt` diffs
- `go mod tidy` leaves no changes

### 6.2 Testing Requirements

**Unit Tests**:

- All check modules must have unit tests
- Policy generator must have unit tests
- Report formatters must have golden file tests

**Integration Tests**:

- kind-based tests in `test/integration/`
- Must cover:
  - Compliant cluster scan (all checks pass)
  - Non-compliant cluster scan (failures detected)
  - Policy enforcement
  - Remediation dry-run

**E2E Tests**:

- GitHub Actions workflow runs nightly
- Tests on kind v1.28, v1.29, v1.30
- Tests on EKS (optional, requires AWS creds)

### 6.3 Security Requirements

**Supply Chain**:

- All releases signed with Sigstore
- SBOM generated with Syft
- Vulnerability scan with Grype (advisory only in v1.0)
- Dependabot enabled

**Code Security**:

- gosec static analysis in CI
- No hardcoded credentials
- All Kubernetes clients use short-lived tokens

### 6.4 Documentation Requirements

**README.md must include**:

- What is kspec (1 paragraph)
- Why not just use X? (vs Polaris, Kyverno, etc.)
- Quickstart (5 minutes to first scan)
- Installation instructions
- Basic usage examples
- Link to full docs

**docs/ must include**:

- Architecture diagram
- Spec reference (all fields documented)
- Compliance framework mappings (NIST, CIS)
- Check catalog (all checks explained)
- Policy generation guide
- Contributing guide

-----

## 7) Implementation Roadmap

### Phase 1: Core Scanning (Weeks 1-3)

**Goals**:

- CLI skeleton with cobra
- Spec schema definition + JSON Schema
- Kubernetes client wrapper
- Basic check modules:
  - Kubernetes version
  - Pod Security Standards
  - Network policies (existence check)
- JSON output format

**Deliverables**:

- `kspec scan` works on kind cluster
- Example specs in `specs/examples/`
- Unit tests for check modules

### Phase 2: Policy Enforcement (Weeks 4-6)

**Goals**:

- Kyverno policy generator
- Kyverno installer (Helm-based)
- Dry-run mode for enforcement
- Violation scanner (detect existing non-compliant pods)

**Deliverables**:

- `kspec enforce` works on kind cluster
- Generated policies are valid Kyverno YAML
- Integration tests for enforcement

### Phase 3: Reporting & Evidence (Weeks 7-9)

**Goals**:

- OSCAL report generator
- SARIF report generator
- Markdown report generator
- Compliance framework mappings (NIST 800-53, CIS)

**Deliverables**:

- `kspec report --format oscal` produces valid OSCAL JSON
- Documentation on framework mappings
- Example reports in docs

### Phase 4: Advanced Checks (Weeks 10-12)

**Goals**:

- Workload security checks (full implementation)
- RBAC checks
- Admission controller checks
- Observability checks
- Image registry checks

**Deliverables**:

- Comprehensive check coverage
- All checks have unit tests
- Check catalog documentation

### Phase 5: Remediation & Polish (Weeks 13-16)

**Goals**:

- Remediation planner
- Dry-run remediation mode
- Apply mode for safe fixes
- Diff functionality (baseline comparison)
- CLI polish (colors, progress bars, etc.)

**Deliverables**:

- `kspec remediate` works safely
- `kspec diff` compares scan results
- Professional CLI UX

### Phase 6: Release Prep (Weeks 17-18)

**Goals**:

- Goreleaser configuration
- Homebrew tap
- GitHub release automation
- Website/docs site (hugo or mkdocs)
- Marketing materials (blog post, demo video)

**Deliverables**:

- v1.0.0 release
- Public documentation site
- Announcement blog post

-----

## 8) Technical Constraints & Decisions

### 8.1 Language & Frameworks

**Language**: Go 1.21+

**Why Go**:

- Native Kubernetes ecosystem (client-go)
- Excellent CLI tooling (cobra)
- Strong static typing for spec validation
- Cross-platform compilation

**Key Dependencies**:

```go
// Core
k8s.io/client-go        // Kubernetes API client
k8s.io/apimachinery     // API machinery
k8s.io/api              // API types

// CLI
github.com/spf13/cobra  // CLI framework
github.com/spf13/viper  // Configuration

// Validation
github.com/xeipuuv/gojsonschema  // JSON Schema validation

// YAML/JSON
gopkg.in/yaml.v3
encoding/json

// Versioning
github.com/Masterminds/semver/v3

// Testing
github.com/stretchr/testify

// Reporting
github.com/olekukonko/tablewriter  // Tables
github.com/fatih/color             // Colors
```

### 8.2 Kubernetes Client Strategy

**Requirements**:

- Support any Kubernetes cluster (1.26+)
- No cloud provider dependencies
- Work with kubeconfig files
- Support in-cluster config (for future in-cluster mode)

**Implementation**:

```go
// internal/kubernetes/client.go
func NewClient(kubeconfigPath string) (kubernetes.Interface, error) {
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
    if err != nil {
        return nil, err
    }
    
    return kubernetes.NewForConfig(config)
}
```

**API Access**:

- Use Discovery API for version checks
- Use dynamic client for CRD inspection
- Use typed clients for core resources
- Implement caching for repeated queries

### 8.3 Policy Engine Support

**v1.0**: Kyverno only

**Why Kyverno**:

- YAML-based policies (easier generation)
- Native Kubernetes resources (no separate deployment)
- Built-in reporting
- Mutation support (future use)
- CNCF Incubating (good governance)

**v1.1+**: Add OPA Gatekeeper support

**Abstraction**:

```go
// pkg/enforcer/enforcer.go
type PolicyEngine interface {
    Name() string
    IsInstalled(ctx context.Context, client kubernetes.Interface) (bool, error)
    Install(ctx context.Context, client kubernetes.Interface) error
    GeneratePolicies(spec *Specification) ([]runtime.Object, error)
    ApplyPolicies(ctx context.Context, client kubernetes.Interface, policies []runtime.Object) error
}
```

### 8.4 Spec Versioning Strategy

**Format**: `apiVersion: kspec.dev/v1`

**Versioning**:

- Spec API version follows Kubernetes conventions
- v1, v1beta1, v2alpha1, etc.
- Breaking changes require new API version
- Support multiple versions via conversion

**Compatibility**:

- Specs include `metadata.version` (semver)
- kspec CLI validates against JSON Schema for that version
- Future versions can deprecate fields

### 8.5 Evidence Storage

**v1.0**: Local files only

- JSON reports written to `./evidence/`
- OSCAL files written to `./evidence/oscal/`
- User responsible for uploading to compliance systems

**v1.1+**: Optional remote storage

- S3-compatible object storage
- Azure Blob Storage
- Google Cloud Storage

### 8.6 Error Handling

**Principles**:

- Never fail silently
- Provide actionable error messages
- Include remediation steps when possible
- Log errors with context

**Example**:

```go
// Bad
return nil, err

// Good
return nil, fmt.Errorf("failed to scan namespace %s: %w", namespace, err)
```

**CLI Exit Codes**:

- 0: Success
- 1: Scan found violations
- 2: Spec validation failed
- 3: Cluster connection failed
- 4: Permission denied
- 5: Internal error

-----

## 9) Example Workflows

### 9.1 FedRAMP Compliance Workflow

**Scenario**: Government contractor needs FedRAMP Moderate evidence

```bash
# 1. Get FedRAMP spec template
kspec docs --template fedramp-moderate > cluster-spec.yaml

# 2. Customize for your environment
vim cluster-spec.yaml  # add specific requirements

# 3. Scan dev cluster
kspec scan --spec cluster-spec.yaml --output json > dev-scan.json

# 4. Fix violations in dev
kspec remediate --spec cluster-spec.yaml --scan-result dev-scan.json --apply

# 5. Enforce policies in dev
kspec enforce --spec cluster-spec.yaml --confirm

# 6. Verify compliance
kspec scan --spec cluster-spec.yaml
# Should show 100% compliance

# 7. Scan production (read-only first)
kspec scan --spec cluster-spec.yaml --kubeconfig ~/.kube/prod-config

# 8. Generate OSCAL evidence for auditors
kspec report \
  --spec cluster-spec.yaml \
  --format oscal \
  --framework nist-800-53 \
  --output fedramp-evidence.json

# 9. Schedule monthly compliance checks
# Add to CI/CD:
kspec scan --spec cluster-spec.yaml --output sarif > results.sarif
# Upload results.sarif to compliance dashboard
```

### 9.2 Multi-Environment Promotion

**Scenario**: Promote validated spec from dev → stage → prod

```bash
# Dev environment
kspec scan --spec dev-spec.yaml
kspec enforce --spec dev-spec.yaml

# Stage environment (stricter spec)
kspec scan --spec stage-spec.yaml  # inherits from dev-spec
kspec enforce --spec stage-spec.yaml

# Production (strictest spec)
kspec scan --spec prod-spec.yaml  # inherits from stage-spec
kspec diff --baseline stage-scan.json  # compare to stage

# Only enforce if diff is acceptable
kspec enforce --spec prod-spec.yaml --confirm
```

### 9.3 Continuous Compliance Monitoring

**Scenario**: Detect drift in production cluster

```bash
# Baseline scan (week 1)
kspec scan --spec cluster-spec.yaml --output json > baseline-w1.json

# Weekly comparison (week 2)
kspec scan --spec cluster-spec.yaml --output json > scan-w2.json
kspec diff --baseline baseline-w1.json --current scan-w2.json

# Output shows:
# NEW VIOLATIONS (3):
#   - namespace/app-3: missing NetworkPolicy
#   - deployment/api: container running as root
# 
# RESOLVED VIOLATIONS (1):
#   - namespace/app-1: NetworkPolicy added

# Remediate new violations
kspec remediate --scan-result scan-w2.json --auto-approve
```

-----

## 10) Non-Functional Requirements

### 10.1 Performance

**Scanning**:

- Full cluster scan (100 namespaces, 1000 pods) < 2 minutes
- Check modules run in parallel where possible
- Kubernetes API calls are cached
- Progress bar shows scan status

**Enforcement**:

- Policy generation < 5 seconds for typical spec
- Policy application < 30 seconds (limited by Kubernetes API)

**Memory**:

- CLI memory usage < 100MB for typical scan
- No memory leaks in long-running scans

### 10.2 Reliability

**Idempotency**:

- `kspec scan` can be run multiple times safely
- `kspec enforce` is idempotent (same result if run twice)
- `kspec remediate` only applies necessary changes

**Partial Failures**:

- If one check fails, continue with others
- Report all results, even if some checks error
- Provide summary of check failures

**Network Resilience**:

- Retry Kubernetes API calls (3 attempts)
- Configurable timeouts
- Graceful handling of API throttling

### 10.3 Security

**Credentials**:

- Never log credentials or tokens
- Support RBAC least-privilege (read-only for scan)
- Enforce mode requires write permissions (documented)

**Isolation**:

- No exec into pods
- No shell commands executed on nodes
- All operations via Kubernetes API

**Audit Trail**:

- All scans logged with timestamp, user, cluster
- Evidence files include provenance metadata
- Reports are tamper-evident (checksums)

### 10.4 Usability

**Error Messages**:

```
# Bad
Error: failed to get pods

# Good
Error: failed to list pods in namespace 'production'
  Reason: Forbidden (403)
  
  You don't have permission to list pods in this namespace.
  
  Required RBAC:
    apiGroup: ""
    resource: pods
    verb: list
    namespace: production
  
  Ask your cluster admin to grant this permission, or run:
    kubectl create rolebinding kspec-scan \
      --clusterrole=view \
      --user=<your-user> \
      --namespace=production
```

**Progress Indicators**:

```
Scanning cluster aks-prod-eastus...
[##########..................] 50% (50/100 checks)
  ✓ Kubernetes version check
  ✓ Pod Security Standards
  ⧗ Network policies (scanning 30 namespaces...)
```

**Verbosity Levels**:

- Default: Summary only
- `-v`: Include check details
- `-vv`: Include debug logs
- `--quiet`: Minimal output (exit code only)

-----

## 11) Future Enhancements (Post-v1.0)

### v1.1 — OPA Gatekeeper Support

- Add OPA policy generator
- Support both Kyverno and OPA in same cluster
- Policy conversion tool (Kyverno → Rego)

### v1.2 — In-Cluster Monitoring

- Deploy kspec-monitor as Deployment
- Continuous compliance scanning
- Webhook for real-time drift alerts
- Integration with Prometheus metrics

### v1.3 — Multi-Cluster Management

- Central control plane for fleet management
- Cross-cluster policy sync
- Aggregated compliance dashboard
- Cluster comparison views

### v1.4 — Advanced Evidence

- PDF report generation
- Excel export for audit teams
- Integration with GRC platforms (ServiceNow, Archer)
- Blockchain-based evidence provenance

### v2.0 — Policy as Code Platform

- GitOps-native spec management
- Spec inheritance and overlays
- Policy testing framework
- Mutation policies (not just validation)

-----

## 12) Success Criteria

### v1.0 is successful if:

1. **Adoption**:

- 100+ GitHub stars in first month
- 3+ companies using in production
- 10+ community contributions

1. **Technical**:

- Scans 1000-pod cluster in < 2 minutes
- Zero critical bugs in first 3 months
- 90%+ unit test coverage

1. **Business**:

- At least 1 design partner willing to pay for support
- Featured in Kubernetes blog or podcast
- Positive feedback from compliance teams

1. **Community**:

- Active Slack/Discord community
- Regular contributor beyond core team
- Forks for custom compliance frameworks

-----

## 13) Risk Mitigation

### Risk: Kubernetes API changes break checks

**Mitigation**:

- Pin client-go versions
- Test against multiple k8s versions (1.26-1.30)
- Deprecation warnings in output

### Risk: Policy engine compatibility issues

**Mitigation**:

- Abstract policy generation behind interface
- Test generated policies in integration tests
- Provide escape hatch (custom policy mode)

### Risk: Spec complexity overwhelms users

**Mitigation**:

- Provide templates for common frameworks
- Docs with examples for every field
- Spec validation with helpful error messages
- `kspec docs --interactive` wizard (future)

### Risk: Performance degrades on large clusters

**Mitigation**:

- Implement caching for k8s API calls
- Parallel check execution
- Namespace filtering option
- Progress tracking for long scans

### Risk: Compliance frameworks change

**Mitigation**:

- Spec version includes framework version
- Document framework mappings separately
- Allow custom control mappings
- Regular updates to templates

-----

## 14) Communication Guidelines

### For GitHub Issues:

- Use issue templates (bug, feature, question)
- Require kspec version, k8s version, spec file
- Label by component (scanner, enforcer, reporter)

### For PRs:

- Require tests for new features
- Changelog entry required
- Sign-off required (DCO)
- Review from 2 maintainers

### For Releases:

- Semantic versioning (MAJOR.MINOR.PATCH)
- Changelog in GitHub releases
- Release notes blog post
- Announcement in Kubernetes Slack

-----

## 15) AI Agent Instructions

### When implementing kspec:

1. **Start with spec schema**:

- Define Go structs matching cluster-spec.yaml
- Generate JSON Schema for validation
- Create example specs

1. **Build scanner incrementally**:

- Start with Kubernetes version check (simplest)
- Add one check module at a time
- Test each check against kind cluster

1. **Use test-driven development**:

- Write check test first
- Implement check
- Verify against real cluster

1. **Generate policies correctly**:

- Use Kyverno documentation
- Validate generated YAML
- Test policies block violations

1. **Make errors helpful**:

- Include context in all errors
- Provide remediation steps
- Link to docs when relevant

1. **Document as you build**:

- Update spec reference for new fields
- Add check to catalog
- Update examples

1. **Prioritize user experience**:

- Add progress bars for long operations
- Use colors for pass/fail/warn
- Make JSON output machine-readable

1. **Security first**:

- Never log sensitive data
- Validate all user input
- Use least-privilege RBAC

-----

## 16) Success Metrics (Measurable)

### Technical Metrics:

- **Scan performance**: < 2 min for 1000-pod cluster
- **Check coverage**: 100+ distinct checks by v1.0
- **Test coverage**: > 80% unit test coverage
- **Policy accuracy**: Zero false positives on compliant clusters

### Adoption Metrics:

- **Downloads**: 1000+ CLI downloads in first 3 months
- **GitHub**: 100+ stars, 10+ contributors
- **Usage**: 50+ clusters scanned (telemetry opt-in)

### Business Metrics:

- **Design partners**: 3+ companies in beta
- **Revenue**: 1 paying customer by month 6
- **Community**: Active Slack with 50+ members

-----

## 17) Final Notes

### Philosophy:

kspec is **opinionated** about security but **flexible** about implementation. We believe:

- Compliance should be codified, not documented
- Evidence should be verifiable, not manual
- Policies should be enforced, not suggested
- Scanning should be continuous, not point-in-time

### What kspec is NOT:

- Not a Kubernetes installer (use kubeadm, EKS, AKS, etc.)
- Not a monitoring system (use Prometheus, Datadog, etc.)
- Not a service mesh (use Istio, Linkerd, etc.)
- Not a deployment tool (use Helm, Kustomize, etc.)

### What kspec IS:

- A specification language for cluster compliance
- A scanner that proves specs are met
- A policy enforcer that prevents violations
- An evidence generator for audits

**Build for clarity, not completeness. v1.0 is a foundation.**

-----

**END OF AGENTS.md**
