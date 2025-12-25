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

## [Unreleased]

### Planned

- Phase 7: Release preparation and public launch
- Phase 8: Advanced features (webhooks, multi-cluster, trends)
- Homebrew formula for easy installation
- Docker images for container-based usage
- Documentation website (Vercel-hosted)
- Alert integrations (Slack, PagerDuty, webhooks)
- DriftConfig CRD for advanced configuration
- Multi-cluster drift monitoring
- Trend analysis and reporting
- SQLite storage backend for drift history

---

[0.1.0]: https://github.com/cloudcwfranck/kspec/releases/tag/v0.1.0
