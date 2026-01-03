# Phase 9A: Production Hardening PR Series

This document outlines the implementation plan for Phase 9A production hardening deliverables.

## Overview

Phase 9A focuses on making kspec production-ready by:
1. Fixing critical Kyverno TLS reliability issues
2. Implementing alert integrations
3. Raising test coverage to 80%+
4. Closing high-impact TODOs
5. Security hardening
6. Creating comprehensive demo/runbook

## Deliverable 1: Kyverno Install / TLS Reliability Fix ✅

**Status**: IN PROGRESS

**Problem**: Kyverno admission controller fails due to missing TLS secret "kyverno-svc.kyverno.svc.kyverno-tls-pair"

**Root Cause**: E2E workflow installs Kyverno without cert-manager, which is required for TLS certificate generation.

**Solution**:

### Changes Made:

1. **Created comprehensive troubleshooting documentation** ✅
   - File: `docs/TROUBLESHOOTING_KYVERNO.md`
   - Covers common TLS issues, validation steps, diagnostic commands
   - Includes automated validation script reference

2. **Fixed e2e.yaml workflow** ✅
   - Added cert-manager installation step before Kyverno
   - Added TLS secret validation step
   - Validates webhook service endpoints
   - Added certificate validation using openssl

3. **Created validation test script** ✅
   - File: `test/integration/validate-kyverno.sh`
   - Validates:
     - Kyverno namespace exists
     - All 4 deployments are ready
     - All pods are running
     - TLS secrets exist and are valid
     - Webhook configurations are populated
     - Webhook service has endpoints
     - CRDs are installed
     - Can create test policies
     - cert-manager is running (recommended)

### Files Modified:
- `.github/workflows/e2e.yaml` - Added cert-manager installation and TLS validation
- `docs/TROUBLESHOOTING_KYVERNO.md` - New troubleshooting guide
- `test/integration/validate-kyverno.sh` - New validation script

### Testing:
- E2E workflow now validates TLS setup explicitly
- Validation script can be run standalone: `./test/integration/validate-kyverno.sh`
- CI will fail if TLS secrets are not properly created

---

## Deliverable 2: Alerting (Track 2)

**Status**: PENDING

**Goal**: Implement alert integrations for Slack, generic webhooks, and PagerDuty

### Implementation Plan:

1. **Create AlertConfig CRD** (`api/v1alpha1/alertconfig_types.go`)
   ```go
   type AlertConfigSpec struct {
       Slack     *SlackConfig     `json:"slack,omitempty"`
       Webhooks  []WebhookConfig  `json:"webhooks,omitempty"`
       Routes    []AlertRoute     `json:"routes,omitempty"`
   }
   ```

2. **Implement alert infrastructure** (`pkg/alerts/`)
   - `types.go` - Alert types and Notifier interface
   - `manager.go` - Alert manager
   - `slack.go` - Slack notifier
   - `webhook.go` - Generic webhook notifier

3. **Integrate into controllers**
   - `controllers/clusterspecification_controller.go` - Alert on drift, compliance failures
   - `pkg/webhooks/circuitbreaker.go` - Alert on circuit breaker trips

4. **Alert triggers**:
   - Compliance score below threshold
   - Drift detected
   - Remediation performed
   - Circuit breaker trips

5. **Tests**:
   - Unit tests for each notifier
   - Golden test files for rendered payloads
   - Integration test for alert routing

### Files to Create:
- `api/v1alpha1/alertconfig_types.go`
- `pkg/alerts/types.go`
- `pkg/alerts/manager.go`
- `pkg/alerts/slack.go`
- `pkg/alerts/webhook.go`
- `pkg/alerts/manager_test.go`
- `pkg/alerts/slack_test.go`
- `pkg/alerts/webhook_test.go`
- `pkg/alerts/testdata/slack_payload.golden.json`
- `controllers/alertconfig_controller.go`

---

## Deliverable 3: Test Coverage (Track 1)

**Status**: PENDING

**Current Coverage**: 19.7%
**Target Coverage**: 80%+

### Implementation Plan:

1. **Fix drift tests**
   - Create `test/fixtures/fake_client.go` with proper scheme registration
   - Unskip tests in `pkg/drift/detector_test.go`
   - Unskip tests in `pkg/drift/remediator_test.go`

2. **Add leader election tests**
   - Create `test/integration/leader_election_test.go`
   - Test leader acquisition, failover, lease renewal

3. **Increase core package coverage**:
   - `pkg/scanner/` - Add edge case tests
   - `pkg/drift/` - Fix skipped tests
   - `pkg/enforcer/` - Add policy generation edge cases
   - `pkg/webhooks/` - Add circuit breaker edge cases

4. **Add kind-based integration suite to CI**
   - Update `.github/workflows/e2e.yaml` to run integration tests
   - Add `make test-integration` target

### Files to Modify/Create:
- `test/fixtures/fake_client.go` - NEW
- `pkg/drift/detector_test.go` - UNSKIP TESTS
- `pkg/drift/remediator_test.go` - UNSKIP TESTS
- `test/integration/leader_election_test.go` - NEW
- `pkg/scanner/checks/*_test.go` - ADD CASES
- `pkg/webhooks/circuitbreaker_test.go` - ADD CASES
- `.github/workflows/e2e.yaml` - Add integration tests

---

## Deliverable 4: Close High-Impact TODOs (Track 3)

**Status**: PENDING

**Current TODOs**: 11 in codebase

### Implementation Plan:

1. **Pin image version** (`pkg/enforcer/multicluster.go:184`)
   ```go
   // BEFORE:
   Image: "ghcr.io/cloudcwfranck/kspec:latest", // TODO: Use specific version

   // AFTER:
   Image: fmt.Sprintf("ghcr.io/cloudcwfranck/kspec:%s", r.OperatorVersion),
   ```

2. **Implement label selector matching** (`pkg/policy/advanced.go:484`)
   ```go
   func (m *AdvancedPolicyManager) matchesSelector(selector *metav1.LabelSelector, labels map[string]string) bool {
       labelSelector, err := metav1.LabelSelectorAsSelector(selector)
       if err != nil {
           return false
       }
       return labelSelector.Matches(labels.Set(labels))
   }
   ```

3. **Implement parameter substitution** (`pkg/policy/advanced.go:886`)
   ```go
   func (m *AdvancedPolicyManager) applyParametersToPolicy(policy *PolicyDefinition, params map[string]interface{}) *PolicyDefinition {
       // Implement template parameter substitution
   }
   ```

4. **Time-based activation** - DECISION NEEDED
   - Option A: Implement fully with cron parsing
   - Option B: Remove from API and docs (defer to Phase 10)
   - Recommendation: Remove for now (out of scope for Phase 9)

### Files to Modify:
- `pkg/enforcer/multicluster.go` - Pin image version
- `pkg/policy/advanced.go` - Implement label selector, parameter substitution
- `api/v1alpha1/clusterspecification_types.go` - Remove time-based activation (if deferring)
- `controllers/status.go` - Track policy enforcement count
- `pkg/client/factory.go` - Add proxy support (if needed)

---

## Deliverable 5: Security Hardening (Track 4)

**Status**: PENDING

### Implementation Plan:

1. **Webhook rate limiting** (`pkg/webhooks/ratelimit.go`)
   - Token bucket algorithm
   - Per-client/IP tracking
   - Configurable limits
   - Default: 100 req/s per client

2. **RBAC minimization audit**
   - Review `config/rbac/role.yaml`
   - Remove unnecessary permissions
   - Document required permissions in `docs/RBAC.md`
   - Separate read-only vs write permissions

3. **SBOM generation and cosign hooks**
   - Add `.goreleaser.yaml` SBOM generation
   - Add cosign signing step to release workflow
   - Document verification process in `docs/SECURITY.md`
   - Make it optional (not required for operation)

### Files to Modify/Create:
- `pkg/webhooks/ratelimit.go` - NEW
- `pkg/webhooks/ratelimit_test.go` - NEW
- `pkg/webhooks/server.go` - Integrate rate limiting
- `config/rbac/role.yaml` - Minimize permissions
- `config/rbac/role_readonly.yaml` - NEW (read-only role)
- `docs/RBAC.md` - NEW
- `docs/SECURITY.md` - NEW
- `.goreleaser.yaml` - Add SBOM/cosign
- `.github/workflows/release.yaml` - Add signing

---

## Deliverable 6: DEMO.md Runbook

**Status**: PENDING

### Implementation Plan:

Create `docs/DEMO.md` with complete end-to-end demo:

1. **Prerequisites**
   - kind, kubectl, helm installed
   - Docker running

2. **Steps**:
   - Create kind cluster
   - Install cert-manager
   - Install Kyverno
   - Validate Kyverno setup
   - Install kspec operator
   - Apply ClusterSpecification
   - Run scan
   - Enforce policies
   - Trigger drift detection
   - Deploy Prometheus/Grafana
   - Import dashboards
   - Show metrics

3. **Expected outputs at each step**
   - Screenshots/command outputs
   - Validation commands
   - Troubleshooting tips

4. **Cleanup**
   - Delete cluster
   - Clean up artifacts

### File to Create:
- `docs/DEMO.md`

---

## Commit Strategy

Each deliverable will be committed separately for clean history:

```
1. feat(kyverno): fix TLS certificate generation with cert-manager
2. feat(alerts): implement Slack and webhook notifiers
3. feat(alerts): add AlertConfig CRD and controller
4. test(drift): fix skipped tests with proper fake client
5. test(coverage): add comprehensive tests for core packages
6. feat(policy): implement label selector matching
7. feat(policy): implement parameter substitution
8. fix(multicluster): pin operator image version
9. feat(webhooks): add rate limiting
10. chore(rbac): minimize permissions to least-privilege
11. docs(demo): add comprehensive end-to-end demo guide
12. chore(security): add SBOM generation and cosign hooks
```

---

## Testing Strategy

**Before each commit**:
1. Run unit tests: `make test`
2. Check coverage: `make test-cover`
3. Run linters: `make lint`

**Before PR**:
1. Run E2E tests locally (if possible)
2. Test on kind cluster
3. Validate all acceptance criteria

**CI Requirements**:
- All unit tests pass
- E2E tests pass
- Coverage ≥ 80% for modified packages
- No new lint errors

---

## Acceptance Criteria

### Deliverable 1: Kyverno TLS ✅
- [x] cert-manager installed before Kyverno in e2e.yaml
- [x] TLS secrets validated in CI
- [x] Webhook service endpoints verified
- [x] TROUBLESHOOTING_KYVERNO.md created
- [x] Validation script created and executable
- [ ] E2E tests pass with TLS validation

### Deliverable 2: Alerting
- [ ] AlertConfig CRD created
- [ ] Slack notifier implemented and tested
- [ ] Webhook notifier implemented and tested
- [ ] Alerts integrate into controllers
- [ ] Golden test files for payloads
- [ ] Documentation for alert setup

### Deliverable 3: Test Coverage
- [ ] Drift tests unskipped and passing
- [ ] Coverage ≥ 80% for pkg/scanner
- [ ] Coverage ≥ 80% for pkg/drift
- [ ] Coverage ≥ 80% for pkg/enforcer
- [ ] Coverage ≥ 80% for pkg/webhooks
- [ ] Leader election tests added
- [ ] Integration tests run in CI

### Deliverable 4: TODOs
- [ ] Image version pinned (no :latest)
- [ ] Label selector matching implemented
- [ ] Parameter substitution implemented
- [ ] Time-based activation removed or implemented
- [ ] All TODO comments resolved

### Deliverable 5: Security
- [ ] Webhook rate limiting implemented
- [ ] RBAC minimized and documented
- [ ] SBOM generation added
- [ ] Cosign hooks added
- [ ] Security documentation created

### Deliverable 6: Demo
- [ ] DEMO.md created
- [ ] Tested end-to-end on fresh machine
- [ ] All commands work
- [ ] Expected outputs documented
- [ ] Screenshots included

---

## Timeline

**Estimated effort**: 5-7 days of focused development

- Day 1: Deliverable 1 (Kyverno TLS) ✅
- Day 2-3: Deliverable 2 (Alerting)
- Day 3-4: Deliverable 3 (Test Coverage)
- Day 4-5: Deliverable 4 (TODOs)
- Day 5-6: Deliverable 5 (Security)
- Day 6-7: Deliverable 6 (Demo + Final Testing)

---

## Notes

- Keep commits atomic and focused
- Write tests alongside implementation
- Update documentation as you go
- Don't break backwards compatibility
- Follow existing code patterns
- Add comments for complex logic

---

**Status**: Deliverable 1 in progress
**Last Updated**: 2025-12-31
