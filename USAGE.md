# kspec Usage Contract and Validation

This document defines the **testing contract** for kspec - the expected behaviors, validation criteria, and usage patterns that must be maintained across all versions.

## Contract Overview

kspec is an enterprise-grade Kubernetes compliance enforcer that **must** provide:

1. **Read-only scanning** - Safe to run in production without modifications
2. **Deterministic enforcement** - Same spec generates identical policies
3. **Type-safe operations** - Compile-time validation of all Kubernetes objects
4. **Comprehensive error handling** - Clear error messages with remediation guidance
5. **Idempotent operations** - Running commands multiple times produces consistent results

## Usage Patterns

### 1. Specification Validation

**Purpose**: Validate cluster specification syntax before use.

**Contract**:
- MUST detect invalid YAML syntax
- MUST validate required fields (apiVersion, kind, metadata, spec)
- MUST validate semantic correctness (version ranges, regex patterns)
- MUST NOT require cluster access
- MUST exit with code 0 on success, non-zero on failure

**Usage**:
```bash
kspec validate --spec cluster-spec.yaml
```

**Expected Behavior**:
```
# Success case
Validation successful

# Failure case
Validation failed: spec.kubernetes.minVersion is required
```

**Testing Contract**:
- [ ] Valid spec files return exit code 0
- [ ] Invalid YAML returns exit code 1
- [ ] Missing required fields return exit code 1
- [ ] Invalid version ranges return exit code 1
- [ ] Helpful error messages include field name and issue

### 2. Cluster Scanning

**Purpose**: Read-only compliance check against cluster state.

**Contract**:
- MUST be read-only (no cluster modifications)
- MUST support all Kubernetes versions >= 1.21
- MUST handle RBAC permission errors gracefully
- MUST provide multiple output formats (text, JSON, OSCAL, SARIF, Markdown)
- MUST exit with code 0 if all checks pass, non-zero if any fail
- MUST complete scan within 60 seconds for clusters < 100 nodes

**Usage**:
```bash
# Text output
kspec scan --spec cluster-spec.yaml

# JSON output (machine-readable)
kspec scan --spec cluster-spec.yaml --output json

# OSCAL compliance report
kspec scan --spec cluster-spec.yaml --output oscal > report.json

# SARIF security report
kspec scan --spec cluster-spec.yaml --output sarif > results.sarif

# Markdown documentation
kspec scan --spec cluster-spec.yaml --output markdown > COMPLIANCE.md
```

**Expected Behavior**:
```
┌─────────────────────────────────────────┐
│ kspec dev — Compliance Report           │
├─────────────────────────────────────────┤
│ Cluster: production                     │
│ Spec: my-cluster-spec v1.0.0            │
│ Scanned: 2025-12-24T10:30:00Z           │
└─────────────────────────────────────────┘

COMPLIANCE: 5/7 checks passed (71%)

[PASS] PASSED CHECKS (5)
─────────────────────
1. Cluster version 1.28.0 is within spec range 1.26.0 - 1.30.0
2. Pod Security Standards enforced in all namespaces
...

[FAIL] FAILURES (2)
───────────────────
1. [HIGH] Missing required network policy default-deny-ingress
2. [MEDIUM] Deployment nginx-deployment: missing resource limits
```

**Testing Contract**:
- [ ] Scan completes without errors on valid kubeconfig
- [ ] Exit code 0 when all checks pass
- [ ] Exit code 1 when checks fail
- [ ] All output formats are well-formed (valid JSON, valid OSCAL, etc.)
- [ ] Scan is truly read-only (verified via audit logs)
- [ ] Permission errors display helpful remediation
- [ ] Results are deterministic (same cluster state = same results)

### 3. Policy Enforcement

**Purpose**: Generate and deploy Kyverno policies from specifications.

**Contract**:
- MUST generate valid Kyverno ClusterPolicy resources
- MUST use typed Kyverno objects (NOT generic unstructured)
- MUST support dry-run mode (preview without deployment)
- MUST support policy export to YAML file
- MUST be idempotent (re-running creates or updates policies)
- MUST verify Kyverno installation before deployment
- MUST provide clear installation instructions if Kyverno missing
- MUST NOT deploy policies unless explicitly requested (no dry-run)

**Usage**:
```bash
# Dry-run: preview policies
kspec enforce --spec cluster-spec.yaml --dry-run

# Export policies to file
kspec enforce --spec cluster-spec.yaml --dry-run --output policies.yaml

# Deploy policies (requires Kyverno)
kspec enforce --spec cluster-spec.yaml

# Skip Kyverno installation check (CI/CD)
kspec enforce --spec cluster-spec.yaml --skip-install
```

**Expected Behavior**:
```
┌─────────────────────────────────────────┐
│ kspec dev — Policy Enforcement          │
└─────────────────────────────────────────┘

[OK] Kyverno Status: Installed
   Version: ghcr.io/kyverno/kyverno:v1.11.0

Policies Generated: 7

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

**Testing Contract**:
- [ ] Dry-run generates policies without deployment
- [ ] Generated policies are valid Kyverno ClusterPolicy YAML
- [ ] All policies have kspec.dev/generated annotation
- [ ] Enforcement creates policies in cluster (verified via kubectl)
- [ ] Re-running enforcement updates existing policies (idempotent)
- [ ] Policy generation is deterministic (same spec = same policies)
- [ ] Kyverno check fails gracefully with installation instructions
- [ ] Skip-install flag bypasses Kyverno check

## Validation Criteria

### Functional Requirements

#### FR-1: Specification Schema
- [x] Support ClusterSpecification v1 schema
- [x] Validate kubernetes version requirements
- [x] Validate Pod Security Standards levels
- [x] Validate network policy requirements
- [x] Validate workload security requirements
- [x] Validate RBAC requirements
- [x] Validate admission controller requirements
- [x] Validate observability requirements
- [x] Validate image security requirements

#### FR-2: Compliance Checks
- [x] Kubernetes version check (semantic version comparison)
- [x] Pod Security Standards check (namespace labels)
- [x] Network policy check (default-deny, required policies)
- [x] Workload security check (security contexts, resource limits, images)
- [x] RBAC check (minimum rules, forbidden wildcards)
- [x] Admission controller check (webhooks, Kyverno policies)
- [x] Observability check (metrics providers, audit logging)

#### FR-3: Policy Generation
- [x] Generate runAsNonRoot policy
- [x] Generate disallow privilege escalation policy
- [x] Generate disallow privileged containers policy
- [x] Generate disallow host namespaces policy
- [x] Generate require resource limits policy
- [x] Generate require image digests policy
- [x] Generate block image registries policy

#### FR-4: Output Formats
- [x] Text format (human-readable)
- [x] JSON format (machine-readable)
- [x] OSCAL format (NIST compliance)
- [x] SARIF format (security tools)
- [x] Markdown format (documentation)

### Non-Functional Requirements

#### NFR-1: Performance
- Scan completion: < 60 seconds for clusters with < 100 nodes
- Scan completion: < 120 seconds for clusters with < 500 nodes
- Policy generation: < 5 seconds per specification
- Memory usage: < 500MB during normal operation

#### NFR-2: Reliability
- Exit codes: 0 on success, 1 on validation failure, 2 on system error
- Error messages: Include field name, issue, and remediation
- Idempotency: All operations can be run multiple times safely
- Crash recovery: No partial state on unexpected termination

#### NFR-3: Security
- Least privilege: Read-only access for scanning
- No secret logging: Never log credentials or sensitive data
- Safe defaults: Enforcement requires explicit confirmation
- Audit trail: All enforcement actions logged

#### NFR-4: Compatibility
- Kubernetes versions: 1.21 - 1.30
- Kyverno versions: 1.10 - 1.12
- Go version: 1.21+
- Client-go: v0.29.x

## Error Handling

### Error Categories

1. **Validation Errors** (exit code 1)
   - Invalid YAML syntax
   - Missing required fields
   - Invalid semantic values

2. **Permission Errors** (exit code 2)
   - Missing kubeconfig
   - Insufficient RBAC permissions
   - Cluster unreachable

3. **System Errors** (exit code 2)
   - Out of memory
   - Disk full
   - Network timeout

### Error Message Format

All error messages MUST follow this format:
```
Error: <brief description>

Details:
  <detailed explanation>

Remediation:
  <actionable steps to fix>
```

Example:
```
Error: Kyverno is not installed

Details:
  Policy enforcement requires Kyverno admission controller to be installed
  in the cluster. Kyverno was not detected in namespace 'kyverno'.

Remediation:
  Install Kyverno using Helm:

  helm repo add kyverno https://kyverno.github.io/kyverno/
  helm repo update
  helm install kyverno kyverno/kyverno \
    --namespace kyverno \
    --create-namespace \
    --wait

  For more information: https://kyverno.io/docs/installation/
```

## Testing Requirements

### Unit Testing
- Minimum coverage: 80% overall
- Critical paths: 100% coverage (policy generation, enforcement)
- Test naming: `TestFunctionName_Scenario`
- Mock external dependencies (Kubernetes API)

### Integration Testing
- Test against real kind clusters
- Test all output formats
- Test error conditions
- Test permission failures
- Test version compatibility

### End-to-End Testing
- Full workflow: validate → scan → enforce
- Multi-node cluster scenarios
- Different Kubernetes versions (1.21, 1.26, 1.29)
- Different Kyverno versions (1.10, 1.11, 1.12)

## Breaking Changes

The following changes are considered **BREAKING** and require major version increment:

1. Specification schema changes (apiVersion bump)
2. CLI command/flag removal or rename
3. Output format changes (JSON structure, OSCAL schema)
4. Exit code meaning changes
5. Kubernetes version support removal
6. Generated policy structure changes

## Deprecation Policy

1. **Announcement**: Feature marked deprecated in release notes
2. **Grace Period**: Minimum 2 minor versions
3. **Removal**: Only in major version increment
4. **Documentation**: Update all docs with migration path

## Compliance Mappings

### NIST 800-53 Controls
- AC-6: Least Privilege (RBAC checks)
- AU-2: Audit Events (observability checks)
- CM-2: Baseline Configuration (specification validation)
- CM-7: Least Functionality (workload security)
- SC-7: Boundary Protection (network policies)
- SI-7: Software Integrity (image digest verification)

### CIS Kubernetes Benchmarks
- 5.2.x: Pod Security Policies (Pod Security Standards)
- 5.3.x: Network Policies (default-deny)
- 5.7.x: General Policies (resource limits, security contexts)

## Version Compatibility Matrix

| kspec Version | Kubernetes Versions | Kyverno Versions | Go Version |
|---------------|---------------------|------------------|------------|
| v1.0.x        | 1.21 - 1.30         | 1.10 - 1.12      | 1.21+      |
| v2.0.x        | 1.26 - 1.32         | 1.11 - 1.13      | 1.22+      |

## Support and Maintenance

- **Active Support**: Latest minor version
- **Security Fixes**: Latest 2 minor versions
- **Bug Fixes**: Latest minor version
- **Feature Development**: Next minor/major version
