# Changelog

All notable changes to kspec will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-15

### 🎉 Initial Release

This is the first public release of kspec, a Kubernetes Cluster Compliance Enforcer that validates clusters against versioned specifications, enforces security policies, and monitors for configuration drift.

### Added

#### Phase 1: Security Scanning
- **Comprehensive security checks** - 14 built-in security checks covering:
  - Kubernetes version validation
  - Pod Security Standards (PSS) enforcement
  - Network policy requirements
  - Workload security controls (privileged containers, hostNetwork, etc.)
  - RBAC best practices
  - Admission controller validation
  - Observability requirements
- **Multiple output formats** - Text, JSON, SARIF, OSCAL, Markdown
- **Read-only operation** - Safe to run in production environments
- **Detailed compliance reporting** with pass/fail status for each check

#### Phase 2: Cluster Specification
- **ClusterSpecification CRD** (`kspec.dev/v1`)
- **YAML-based specification format** for defining cluster requirements
- **Version-controlled compliance** - Specifications are versioned and auditable
- **Comprehensive validation** - Validates specs before enforcement
- **Example specifications** included in `specs/examples/`

#### Phase 3: Policy Generation
- **Automatic Kyverno policy generation** from ClusterSpecification
- **13 built-in policy generators** covering:
  - Pod Security Standards
  - Image digest requirements
  - Resource quotas and limits
  - Privileged workload prevention
  - Network policy enforcement
- **Dry-run mode** - Preview policies before applying
- **Policy export** - Save generated policies as YAML files

#### Phase 4: Kyverno Integration
- **Automated Kyverno installation** (optional, with user confirmation)
- **Policy deployment** to clusters via dynamic client
- **Webhook verification** - Ensures policies are active
- **Installation status checks** - Detects existing Kyverno installations
- **Install instructions** - Provides manual installation guidance when needed

#### Phase 5: Policy Enforcement
- **Full enforcement workflow** - `kspec enforce` command
- **Policy lifecycle management** - Create, update, and verify policies
- **Skip installation flag** - `--skip-install` for pre-installed Kyverno
- **Enforcement verification** - Tests that policies actively block violations
- **Enforcement reporting** - Shows applied policies and enforcement status

#### Phase 6: Drift Detection & Auto-Remediation
- **Drift detection** - Identifies deviations from specification:
  - **Policy drift** - Missing, modified, or extra Kyverno policies
  - **Compliance drift** - New violations of security checks
  - **Configuration drift** - (future) Resource-level changes
- **Automatic remediation** - Restores missing/modified policies
- **Dry-run remediation** - Preview changes before applying
- **Drift history** - Track drift events over time
- **Continuous monitoring** - Watch mode for ongoing drift detection
- **Storage backends** - In-memory and file-based drift history
- **CronJob deployment** - Automated drift monitoring in clusters
- **Drift severity levels** - Critical, High, Medium, Low classifications
- **Conservative safety** - No destructive actions without explicit flags

### CLI Commands

- `kspec version` - Display version information
- `kspec validate` - Validate cluster specification files
- `kspec scan` - Scan cluster for compliance issues
- `kspec export` - Export generated policies to YAML
- `kspec enforce` - Enforce security policies on cluster
- `kspec drift detect` - Detect configuration drift
- `kspec drift remediate` - Auto-remediate detected drift
- `kspec drift history` - View drift detection history

### Documentation

- Complete README with quickstart guide
- Per-phase documentation in `docs/`:
  - `SCANNING.md` - Security scanning guide
  - `ENFORCEMENT.md` - Policy enforcement guide
  - `DRIFT_DETECTION.md` - Drift detection and remediation guide
  - `PHASE_*_PLAN.md` - Detailed implementation plans
- Example specifications in `specs/examples/`
- Deployment manifests for drift monitoring in `deploy/drift/`

### CI/CD

- **GitHub Actions workflows**:
  - Lint and test on every push
  - E2E policy enforcement tests
  - E2E drift detection tests
  - Automated releases with GoReleaser
- **Test coverage**: 77.6% for core packages
- **Multi-platform testing**: Linux, multiple Kubernetes versions

### Distribution

- **Pre-built binaries** for:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- **Automated releases** via GoReleaser
- **Checksums** provided for all binaries
- **GitHub Releases** with comprehensive release notes

### Security

- **Least-privilege RBAC** - Minimal required permissions
- **Read-only scanning** - No cluster modifications during scans
- **Conservative remediation** - Safe defaults, no auto-delete
- **Credential protection** - Never logs sensitive information
- **Pod Security Standards** - Enforces baseline security by default

### Performance

- **Fast scanning** - Completes full cluster scan in seconds
- **Concurrent checks** - Parallel execution of security checks
- **Efficient policy generation** - Minimal API calls
- **Graceful error handling** - Continues on non-critical failures

## [0.2.0] - 2025-12-27

### 🎉 Kubernetes Operator Release

This release introduces the **kspec Kubernetes Operator** for continuous cluster compliance monitoring. The operator automates compliance scanning, drift detection, and report generation using Kubernetes-native custom resources.

### Added

#### Phase 7: Kubernetes Operator
- **Kubernetes Operator** for continuous compliance monitoring
  - Automatic reconciliation every 5 minutes
  - Multi-cluster scanning support via ClusterTarget CRD
  - Immutable audit trail with ComplianceReport and DriftReport CRs
  - Label-based resource cleanup with proper finalizers
- **ClusterSpecification CRD** (`kspec.io/v1alpha1`)
  - Declarative cluster compliance definitions
  - Version-controlled security requirements
  - Status tracking with compliance scores and conditions
- **ClusterTarget CRD** for multi-cluster support
  - Remote cluster scanning via kubeconfig
  - Connection health monitoring
  - Per-cluster enforcement controls
- **ComplianceReport CRD** for audit trail
  - Immutable compliance scan results
  - Detailed check results with severity and status
  - Timestamp-based uniqueness (microsecond precision)
- **DriftReport CRD** for drift detection
  - Policy drift detection (missing, modified, extra)
  - Compliance drift tracking
  - Drift severity classification
- **Enterprise-grade deployment manifests**
  - Production security defaults (non-root, read-only FS, dropped capabilities)
  - Kustomize-based installation
  - RBAC with least-privilege permissions
  - Health and readiness probes
- **Container image distribution**
  - Published to GitHub Container Registry (ghcr.io)
  - Multi-arch support (amd64, arm64)
  - OCI-compliant image labels

### Fixed
- **CRD enum validation** - Normalized scanner outputs to match CRD requirements
  - Status: pass/fail/skip → Pass/Fail/Error
  - Severity: low/medium/high/critical → Low/Medium/High/Critical
  - Drift types: policy/compliance/configuration → Policy/Compliance/Configuration
- **Report name collisions** - Microsecond timestamp precision prevents duplicates
- **ClusterSpec phase transitions** - Always reaches Active after successful scan
- **Owner reference cleanup** - Label-based cleanup for cluster-scoped to namespaced resources
- **Go version compatibility** - Fixed go.mod to use Go 1.21
- **Build system** - Added kubebuilder-compatible Makefile targets
- **CRD installation** - Proper kustomize-based installation

### Changed
- **Webhooks disabled by default** - Webhooks require cert-manager and are marked experimental
  - Enable with `--enable-webhooks=true` flag
  - TLS provisioning deferred to v0.3.0
  - Operator safe to install on any cluster
- **Enterprise-grade test coverage** - 55+ normalization tests, 80%+ coverage
- **Security hardening** - Production-grade security defaults in all manifests

### Production-Ready Features
✅ **Continuous Compliance Monitoring** - ClusterSpecification reconciliation
✅ **Multi-Cluster Scanning** - ClusterTarget support for remote clusters
✅ **Immutable Audit Trail** - ComplianceReport and DriftReport CRs
✅ **Drift Detection** - Policy and compliance drift tracking
✅ **Enterprise Security** - Non-root, read-only FS, dropped capabilities
✅ **Health Monitoring** - Prometheus metrics, health probes

### Experimental Features
⚠️ **Admission Webhooks** - Disabled by default, requires cert-manager (v0.3.0)
⚠️ **Policy Enforcement** - Use CLI `kspec enforce` command for Kyverno integration

### Known Limitations
- **Webhook TLS** - Requires manual cert-manager setup (deferred to v0.3.0)
- **Policy Enforcement** - Operator performs monitoring only; use CLI for enforcement
- **Multi-cluster CLI commands** - `kspec cluster discover/add` not yet implemented
- **Web Dashboard** - Dashboard deployment exists but not documented

### Installation

**Quick Install:**
```bash
kubectl apply -k github.com/cloudcwfranck/kspec/config/default?ref=v0.2.0
```

**Verify Installation:**
```bash
kubectl get pods -n kspec-system
kubectl get crd | grep kspec.io
```

**Quick Start:**
```bash
# Apply sample ClusterSpecification
kubectl apply -f https://raw.githubusercontent.com/cloudcwfranck/kspec/v0.2.0/config/samples/kspec_v1alpha1_clusterspecification.yaml

# Watch status
kubectl get clusterspec -n kspec-system -w

# View compliance report
kubectl get compliancereport -n kspec-system
```

### Migration from v0.1.0
If using v0.1.0 CLI-based workflow:
1. Install operator: `kubectl apply -k github.com/cloudcwfranck/kspec/config/default?ref=v0.2.0`
2. Convert spec YAML to ClusterSpecification CR
3. Apply CR: `kubectl apply -f clusterspec.yaml`
4. Continue using CLI for policy enforcement: `kspec enforce --spec cluster-spec.yaml`

### Documentation
- [Operator Quickstart Guide](docs/OPERATOR_QUICKSTART.md)
- [API Reference](docs/API_REFERENCE.md)
- [Webhooks (Experimental)](docs/WEBHOOKS.md)
- [Development Guide](docs/OPERATOR_DEVELOPMENT_GUIDE.md)

---

## [Unreleased]

### Planned (v0.3.0)
- Webhook TLS with cert-manager integration
- Policy enforcement in operator (automated Kyverno policy generation)
- Multi-cluster CLI commands (`kspec cluster discover/add`)
- High availability with leader election
- Homebrew formula for easy installation
- Documentation website (Vercel-hosted)

### Planned (Future)
- Alert integrations (Slack, PagerDuty, webhooks)
- DriftConfig CRD for advanced configuration
- Trend analysis and reporting
- SQLite storage backend for drift history

---

[0.2.0]: https://github.com/cloudcwfranck/kspec/releases/tag/v0.2.0
[0.1.0]: https://github.com/cloudcwfranck/kspec/releases/tag/v0.1.0
