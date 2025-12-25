# kspec - Kubernetes Cluster Compliance Enforcer

**Ship a cluster spec, not a platform.**

kspec validates Kubernetes clusters against versioned specifications, enforces security policies, and generates compliance evidence for audits (FedRAMP, NIST 800-53, CIS benchmarks).

## What is kspec?

kspec is a CLI tool that scans your Kubernetes clusters to ensure they comply with security and compliance requirements defined in a YAML specification file. Unlike cluster installers, kspec works with existing clusters and focuses on proving compliance rather than provisioning infrastructure.

**Key Features:**
- 📋 **Specification-driven validation** - Define requirements in YAML, validate clusters against them
- 🔍 **Read-only scanning** - Safe to run in production with no modifications
- 🛡️ **Policy enforcement** - Deploy Kyverno policies from specifications
- 📊 **Compliance evidence** - Generate OSCAL, SARIF, and JSON reports for auditors
- 🌐 **Cloud-agnostic** - Works on EKS, AKS, GKE, on-prem, kind, k3d equally well
- 🔒 **Security-first** - Never logs credentials, uses least-privilege RBAC

## Why not just use X?

| Tool | Purpose | kspec Difference |
|------|---------|------------------|
| **Polaris** | Cluster best practices | kspec enforces custom specs + generates audit evidence |
| **Kyverno** | Policy engine | kspec generates policies from specs + validates compliance |
| **Terraform** | Infrastructure provisioning | kspec validates existing clusters, doesn't provision |
| **OpenSCAP** | OS compliance | kspec is Kubernetes-native with k8s-specific checks |

kspec is the bridge between compliance frameworks (NIST, CIS) and your Kubernetes clusters.

## Quickstart (5 minutes)

### Installation

#### Binary Releases (Recommended)

Download pre-built binaries from the [latest release](https://github.com/cloudcwfranck/kspec/releases/latest):

**Linux (x86_64)**
```bash
curl -L https://github.com/cloudcwfranck/kspec/releases/latest/download/kspec_0.1.0_Linux_amd64.tar.gz | tar xz
sudo mv kspec /usr/local/bin/
kspec version
```

**macOS (Apple Silicon)**
```bash
curl -L https://github.com/cloudcwfranck/kspec/releases/latest/download/kspec_0.1.0_Darwin_arm64.tar.gz | tar xz
sudo mv kspec /usr/local/bin/
kspec version
```

**macOS (Intel)**
```bash
curl -L https://github.com/cloudcwfranck/kspec/releases/latest/download/kspec_0.1.0_Darwin_amd64.tar.gz | tar xz
sudo mv kspec /usr/local/bin/
kspec version
```

**Windows**
Download the `.zip` file from the [releases page](https://github.com/cloudcwfranck/kspec/releases/latest) and extract `kspec.exe` to a directory in your PATH.

#### From Source

```bash
# Requires Go 1.21+
git clone https://github.com/cloudcwfranck/kspec
cd kspec
go build -o kspec ./cmd/kspec
sudo mv kspec /usr/local/bin/
```

### Basic Usage

1. **Create a cluster specification**

```yaml
# cluster-spec.yaml
apiVersion: kspec.dev/v1
kind: ClusterSpecification
metadata:
  name: my-cluster-spec
  version: "1.0.0"
  description: "Production cluster requirements"

spec:
  kubernetes:
    minVersion: "1.26.0"
    maxVersion: "1.30.0"
```

2. **Validate the spec**

```bash
kspec validate --spec cluster-spec.yaml
```

3. **Scan your cluster**

```bash
# Scan with text output (human-readable)
kspec scan --spec cluster-spec.yaml

# Scan with JSON output (machine-readable)
kspec scan --spec cluster-spec.yaml --output json

# Generate OSCAL compliance report (NIST framework)
kspec scan --spec cluster-spec.yaml --output oscal > report.json

# Generate SARIF security report (for security tools)
kspec scan --spec cluster-spec.yaml --output sarif > results.sarif

# Generate Markdown documentation
kspec scan --spec cluster-spec.yaml --output markdown > COMPLIANCE.md
```

**Output:**
```
┌─────────────────────────────────────────┐
│ kspec v1.0.0 — Compliance Report        │
├─────────────────────────────────────────┤
│ Cluster: unknown                        │
│ Spec: my-cluster-spec v1.0.0            │
│ Scanned: 2025-01-15T10:30:00Z           │
└─────────────────────────────────────────┘

COMPLIANCE: 1/1 checks passed (100%)

✅ PASSED CHECKS (1)
─────────────────────
✓ Cluster version 1.28.0 is within spec range 1.26.0 - 1.30.0
```

4. **Enforce policies (prevent future violations)**

kspec can automatically generate and deploy Kyverno ClusterPolicy resources from your specification. This enables **proactive enforcement** - blocking non-compliant workloads before they're created.

**Prerequisites:**
- [Kyverno](https://kyverno.io) v1.11+ installed in your cluster
- Required cluster permissions to create ClusterPolicy resources

**Install Kyverno:**
```bash
kubectl create namespace kyverno
kubectl apply -f https://github.com/kyverno/kyverno/releases/download/v1.11.4/install.yaml
```

**Policy Enforcement Workflow:**

```bash
# 1. Preview policies (dry-run, see what would be created)
kspec enforce --spec cluster-spec.yaml --dry-run

# 2. Save generated policies to file for review
kspec enforce --spec cluster-spec.yaml --dry-run --output policies.yaml

# 3. Deploy policies to cluster (requires Kyverno installed)
kspec enforce --spec cluster-spec.yaml

# 4. Re-run to update policies (idempotent)
kspec enforce --spec cluster-spec.yaml
```

**What gets enforced:**

Based on your `spec.workloads` and `spec.workloads.images` configuration, kspec generates:

| Policy | Generated When | Blocks |
|--------|----------------|--------|
| `require-run-as-non-root` | `securityContext.runAsNonRoot: true` required | Pods without runAsNonRoot=true |
| `disallow-privilege-escalation` | `allowPrivilegeEscalation: false` required | Containers with allowPrivilegeEscalation=true |
| `disallow-privileged-containers` | `privileged: true` forbidden | Privileged containers |
| `disallow-host-namespaces` | `hostNetwork/hostPID/hostIPC: true` forbidden | Pods using host namespaces |
| `require-resource-limits` | Resource limits required | Containers without CPU/memory limits |
| `require-image-digests` | `requireDigests: true` | Images using tags instead of digests |
| `block-image-registries` | `blockedRegistries` specified | Images from blocked registries |

**Enforcement Output:**
```
┌─────────────────────────────────────────┐
│ kspec vdev — Policy Enforcement       │
└─────────────────────────────────────────┘

[OK] Kyverno Status: Installed
     Version: ghcr.io/kyverno/kyverno:v1.11.4

Policies Generated: 7
Policies Applied: 7

Generated Policies:
───────────────────
  1. require-run-as-non-root
  2. disallow-privilege-escalation
  3. require-resource-limits
  4. disallow-privileged-containers
  5. disallow-host-namespaces
  6. require-image-digests
  7. block-image-registries

[OK] Policies successfully deployed
```

**Testing enforcement:**
```bash
# Try to create a privileged pod (should be blocked)
kubectl run test --image=nginx --privileged=true

# Output:
# Error from server: admission webhook "validate.kyverno.svc-fail" denied the request:
# resource Pod/default/test was blocked due to the following policies
# disallow-privileged-containers:
#   check-privileged: 'validation error: Privileged containers are not allowed'
```

5. **Drift Detection (monitor for configuration changes)**

kspec can detect when your cluster state drifts from your specification and automatically remediate it. This provides **continuous compliance** - ensuring your cluster stays in the desired state over time.

**Drift Detection Workflow:**

```bash
# 1. Detect drift once
kspec drift detect --spec cluster-spec.yaml

# 2. Watch for drift continuously (check every 5 minutes)
kspec drift detect --spec cluster-spec.yaml --watch --watch-interval=5m

# 3. Output drift report to file
kspec drift detect --spec cluster-spec.yaml --output json > drift-report.json

# 4. Remediate detected drift (dry-run first to preview)
kspec drift remediate --spec cluster-spec.yaml --dry-run

# 5. Apply remediation
kspec drift remediate --spec cluster-spec.yaml

# 6. View drift history
kspec drift history --spec cluster-spec.yaml --since=24h
```

**What gets detected:**

kspec detects three types of drift:

| Drift Type | Description | Remediation |
|------------|-------------|-------------|
| **Missing Policies** | Policies in spec but not in cluster | Automatically created |
| **Modified Policies** | Policies changed from spec | Automatically updated |
| **Extra Policies** | Policies in cluster but not in spec | Reported (delete with --force) |
| **Compliance Violations** | New failures in compliance checks | Manual remediation required |

**Drift Detection Output:**

```
┌─────────────────────────────────────────┐
│ kspec vdev — Drift Detection          │
└─────────────────────────────────────────┘

[DRIFT] Detected 3 drift events
Severity: high

Policy Drift: 2
Compliance Drift: 1

Drift Events:
─────────────
[high] ClusterPolicy/require-run-as-non-root: ClusterPolicy 'require-run-as-non-root' is missing from cluster
[medium] ClusterPolicy/disallow-host-namespaces: ClusterPolicy 'disallow-host-namespaces' has been modified
[high] Check/kubernetes-version: Check failed
```

**Automatic Remediation Output:**

```
┌─────────────────────────────────────────┐
│ kspec vdev — Drift Remediation        │
└─────────────────────────────────────────┘

Remediation Summary:
───────────────────
Total events: 3
Remediated: 2
Failed: 0
Manual required: 1

Remediated:
  [OK] ClusterPolicy/require-run-as-non-root: Created missing policy
  [OK] ClusterPolicy/disallow-host-namespaces: Updated policy to match spec

[OK] Remediation complete
```

## Example Specifications

See `specs/examples/` for ready-to-use templates:
- `minimal.yaml` - Basic cluster validation (Kubernetes version only)
- `moderate.yaml` - CIS Kubernetes baseline security (baseline PSS, network policies)
- `strict.yaml` - NIST 800-53 high-compliance (restricted PSS, full compliance mappings)
- `comprehensive.yaml` - Complete security baseline demonstrating all Phase 4 checks

## What's Implemented (Phases 1-4 Complete)

✅ **Phase 1: Foundation**
- Specification schema with YAML support
- Kubernetes version validation check
- JSON and text output formats
- CLI with `version`, `validate`, and `scan` commands
- Unit tests with >80% coverage

✅ **Phase 2: Core Checks**
- Pod Security Standards check (enforce/audit/warn levels)
- Network policy checks (default-deny, required policies)
- Namespace exemption support
- 21 comprehensive unit tests

✅ **Phase 3: Advanced Reporting**
- OSCAL reporter (NIST compliance framework)
- SARIF reporter (security scanning standard)
- Markdown reporter (human-readable documentation)
- Multi-format output support (JSON, text, OSCAL, SARIF, Markdown)

✅ **Phase 4: Advanced Checks**
- **Workload Security Check** - Container security contexts, resource limits/requests, privileged containers, host namespaces
- **RBAC Check** - Minimum required rules, forbidden wildcard permissions, least-privilege validation
- **Admission Controller Check** - ValidatingWebhookConfigurations, MutatingWebhookConfigurations, Kyverno policy validation
- **Observability Check** - Metrics providers (Prometheus, metrics-server), audit logging requirements
- **Image Security** - Allowed/blocked registries, digest requirements, wildcard registry support
- 60+ comprehensive unit tests with 84.6% coverage
- Comprehensive example spec demonstrating all checks

✅ **Phase 5: Policy Enforcement**
- **`kspec enforce` command** - Generate and deploy Kyverno ClusterPolicy resources from specifications
- **Automatic policy generation** - 7 policy types based on workload/image security requirements
- **Dry-run mode** - Preview policies before deployment with `--dry-run`
- **Policy export** - Save generated policies to YAML files with `--output`
- **Kyverno detection** - Auto-detect Kyverno v1.11+ installation
- **Idempotent deployment** - Update existing policies using resourceVersion
- **Policy validation** - RFC 1123 DNS subdomain validation for policy names
- **Comprehensive e2e tests** - Full test coverage including webhook readiness, policy creation, idempotency, and blocking behavior
- **Supported policies**: runAsNonRoot, privilege escalation, privileged containers, host namespaces, resource limits, image digests, registry blocks

✅ **Phase 6: Drift Detection**
- **`kspec drift detect` command** - Detect configuration drift between specification and cluster state
- **`kspec drift remediate` command** - Automatic remediation of detected drift
- **`kspec drift history` command** - View historical drift events and statistics
- **Policy drift detection** - Missing, modified, and extra Kyverno policies
- **Compliance drift detection** - New failures in compliance checks
- **Automatic remediation** - Create missing policies, update modified policies
- **Continuous monitoring** - Watch mode with configurable polling intervals
- **Drift storage** - In-memory and file-based storage for drift history
- **Conservative remediation** - Extra policies reported only (delete with --force)
- **Severity levels** - Critical, High, Medium, Low drift classification

📅 **Roadmap (Future Phases)**
- Additional policy types: Network policies, RBAC policies
- Multi-cluster policy deployment
- CronJob deployment for automated drift monitoring
- Policy testing framework
- Additional checks: Image vulnerability scanning, secret management validation

## Development

### Build from source

```bash
# Clone repository
git clone https://github.com/cloudcwfranck/kspec
cd kspec

# Install dependencies
go mod tidy

# Build
go build -o kspec ./cmd/kspec

# Run tests
go test ./... -v

# Run with coverage
go test ./... -cover
```

### Project Structure

```
kspec/
├── cmd/kspec/          # CLI entry point
├── pkg/
│   ├── spec/           # Specification schema and validation
│   ├── scanner/        # Compliance checks
│   │   └── checks/     # Individual check implementations
│   └── reporter/       # Output formatters (JSON, text, etc.)
├── specs/examples/     # Example specification files
├── test/               # Integration and E2E tests
└── docs/               # Documentation
```

## Contributing

We welcome contributions! Please see `docs/contributing.md` for guidelines.

## Architecture

kspec follows these design principles:

1. **Read-First, Enforce-Second** - All scanning is read-only and safe
2. **Specification as Truth** - Cluster state validated against versioned specs
3. **Evidence-Based Compliance** - Every check produces verifiable evidence
4. **Minimal Runtime Footprint** - CLI-first design, no required in-cluster components

For detailed architecture, see [AGENTS.md](./AGENTS.md).

## License

Apache 2.0 - see [LICENSE](./LICENSE) for details.

## Support

- **Issues**: https://github.com/cloudcwfranck/kspec/issues
- **Discussions**: https://github.com/cloudcwfranck/kspec/discussions
- **Documentation**: https://kspec.dev (coming soon)

---

**Status**: Phase 6 complete - Drift detection and automatic remediation implemented. Detection + Prevention + Drift Monitoring complete! See [AGENTS.md](./AGENTS.md) for full implementation roadmap.
