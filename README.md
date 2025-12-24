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

```bash
# From source (requires Go 1.21+)
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

```bash
# Generate policies (dry-run, see what would be created)
kspec enforce --spec cluster-spec.yaml --dry-run

# Save generated policies to file
kspec enforce --spec cluster-spec.yaml --dry-run --output policies.yaml

# Deploy policies to cluster (requires Kyverno installed)
kspec enforce --spec cluster-spec.yaml
```

**Enforcement Output:**
```
┌─────────────────────────────────────────┐
│ kspec v1.0.0 — Policy Enforcement       │
└─────────────────────────────────────────┘

✅ Kyverno Status: Installed
   Version: ghcr.io/kyverno/kyverno:v1.11.0

📋 Policies Generated: 7

Generated Policies:
───────────────────
1. require-run-as-non-root
2. disallow-privilege-escalation
3. disallow-privileged-containers
4. disallow-host-namespaces
5. require-resource-limits
6. require-image-digests
7. block-image-registries

Next Steps:
───────────
1. Review the generated policies above
2. Run: kspec enforce --spec <file> (without --dry-run) to deploy
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

✅ **Phase 5: Policy Enforcement** (NEW)
- **`kspec enforce` command** - Generate and deploy Kyverno policies from specs
- **Kyverno Policy Generator** - Automatic policy creation from workload/image requirements
- **Dry-run mode** - Preview policies before deployment
- **Installation checking** - Verify Kyverno is installed
- **Policy export** - Save generated policies to YAML files
- **7 policy types**: runAsNonRoot, privilege escalation, privileged containers, host namespaces, resource limits, image digests, registry blocks

📅 **Roadmap (Future Phases)**
- Phase 6: Drift detection & remediation automation
- Additional policy types: Network policies, RBAC policies
- Multi-cluster policy deployment
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

**Status**: Phase 5 complete - Policy enforcement with Kyverno implemented. Detection + Prevention complete! See [AGENTS.md](./AGENTS.md) for full implementation roadmap.
