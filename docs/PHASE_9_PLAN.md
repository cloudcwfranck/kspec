# Phase 9: Production Hardening & Operational Excellence

**Version**: v0.4.0 (targeting)
**Timeline**: 4-5 weeks
**Status**: 🚧 In Progress
**Start Date**: December 30, 2025

---

## Executive Summary

Phase 9 focuses on **hardening the foundation** built in Phases 1-8, improving operational excellence, and ensuring kspec is truly production-ready for enterprise deployments.

**Goals**:
- ✅ Achieve 80%+ test coverage
- ✅ Resolve all high-priority technical debt
- ✅ Add operational tooling (alerts, monitoring)
- ✅ Security hardening
- ✅ Complete documentation

**Why Phase 9 Before New Features?**
- Solidify foundation before adding complexity
- Improve confidence through comprehensive testing
- Add operational tooling for production success
- Resolve technical debt while codebase is fresh
- Security hardening is critical for enterprises

---

## Success Metrics

| **Metric** | **Baseline (v0.3.0)** | **Target (v0.4.0)** | **Priority** |
|------------|----------------------|-------------------|--------------|
| Test Coverage | ~50-60% | **80%+** | 🔴 Critical |
| High-Priority TODOs | 4 | **0** | 🔴 Critical |
| Alert Integrations | 0 | **3** (Slack, PD, Webhook) | 🔴 Critical |
| Security Issues | 4 identified | **0** | 🔴 Critical |
| Documentation Coverage | ~60% | **95%+** | 🟡 High |
| Skipped Tests | 8 | **0** | 🟡 High |
| Performance Benchmarks | 0 | **3** scenarios | 🟢 Medium |
| Package Documentation | ~40% | **100%** | 🟢 Medium |

---

## Phase 9 Tracks (Parallel Execution)

### **Track 1: Testing & Quality** (Weeks 1-2)
**Owner**: Engineering
**Priority**: 🔴 Critical

### **Track 2: Alert Integrations** (Weeks 2-3)
**Owner**: Engineering + SRE
**Priority**: 🔴 Critical

### **Track 3: Technical Debt Resolution** (Week 3)
**Owner**: Engineering
**Priority**: 🟡 High

### **Track 4: Security Hardening** (Week 4)
**Owner**: Security + Engineering
**Priority**: 🔴 Critical

### **Track 5: Observability Enhancement** (Week 4)
**Owner**: SRE + Engineering
**Priority**: 🟡 High

### **Track 6: Documentation** (Week 5)
**Owner**: Technical Writing + Engineering
**Priority**: 🟢 Medium

---

## Track 1: Testing & Quality (Weeks 1-2)

**Goal**: Achieve 80%+ test coverage and fix all skipped tests

### **1.1 Fix Skipped Tests** (Week 1, Days 1-2)

**Current State**: 8 tests skipped due to fake client setup issues

**Tasks**:
- [ ] Set up proper fake client infrastructure
  ```go
  // test/fixtures/fake_client.go
  package fixtures

  func NewFakeClientWithCRDs() (client.Client, error) {
      // Create fake client with CRD registration
      scheme := runtime.NewScheme()
      _ = kspecv1alpha1.AddToScheme(scheme)
      _ = kyvernov1.AddToScheme(scheme)

      return fake.NewClientBuilder().
          WithScheme(scheme).
          WithStatusSubresource(&kspecv1alpha1.ClusterSpecification{}).
          Build(), nil
  }
  ```

- [ ] Fix drift detector tests (5 skipped)
  - `pkg/drift/detector_test.go:TestDetectPolicyDrift`
  - `pkg/drift/detector_test.go:TestDetectComplianceDrift`
  - `pkg/drift/detector_test.go:TestDetectConfigDrift`
  - `pkg/drift/detector_test.go:TestCompareResources`
  - `pkg/drift/detector_test.go:TestDriftSeverityCalculation`

- [ ] Fix remediator tests (3 skipped)
  - `pkg/drift/remediator_test.go:TestRemediateDryRun`
  - `pkg/drift/remediator_test.go:TestRemediateWithApply`
  - `pkg/drift/remediator_test.go:TestRemediateMultipleDrifts`

**Acceptance Criteria**:
- ✅ All 8 skipped tests now pass
- ✅ Tests run in CI without skips
- ✅ Test coverage for drift package: >75%

---

### **1.2 Add Leader Election Tests** (Week 1, Days 3-4)

**Current State**: No tests for Phase 8 leader election

**Tasks**:
- [ ] Create leader election test suite
  ```go
  // test/integration/leader_election_test.go
  package integration

  func TestLeaderElection(t *testing.T) {
      // Test scenarios:
      // 1. Leader acquires lease
      // 2. Follower waits for lease
      // 3. Leader releases lease on shutdown
      // 4. Follower becomes leader on leader failure
  }
  ```

- [ ] Test failover scenarios
  ```go
  func TestLeaderFailover(t *testing.T) {
      // 1. Kill leader pod
      // 2. Verify new leader elected within 15s
      // 3. Verify reconciliation continues
  }
  ```

- [ ] Test lease renewal
  ```go
  func TestLeaseRenewal(t *testing.T) {
      // 1. Verify leader renews lease every 10s
      // 2. Verify lease duration is 15s
      // 3. Verify retry period is 2s
  }
  ```

**Acceptance Criteria**:
- ✅ 5+ leader election test cases
- ✅ Failover tested and verified <15s
- ✅ Tests run in envtest environment

---

### **1.3 Add Advanced Policy Tests** (Week 1, Day 5)

**Current State**: No tests for Phase 7 advanced policies

**Tasks**:
- [ ] Test policy templates
  ```go
  func TestPolicyTemplates(t *testing.T) {
      // Test security-baseline template
      // Test compliance-strict template
      // Test parameter substitution
  }
  ```

- [ ] Test policy inheritance
  ```go
  func TestPolicyInheritance(t *testing.T) {
      // Test merge strategy
      // Test override strategy
      // Test append strategy
  }
  ```

- [ ] Test namespace scoping
  ```go
  func TestNamespaceScoping(t *testing.T) {
      // Test include/exclude lists
      // Test label selectors
  }
  ```

- [ ] Test time-based activation
  ```go
  func TestTimeBasedActivation(t *testing.T) {
      // Test timezone handling
      // Test time windows
      // Test schedule parsing
  }
  ```

- [ ] Test policy exemptions
  ```go
  func TestPolicyExemptions(t *testing.T) {
      // Test exemption matching
      // Test expiration
      // Test approval tracking
  }
  ```

**Acceptance Criteria**:
- ✅ 15+ test cases for advanced policies
- ✅ Coverage for policy package >80%
- ✅ All edge cases covered

---

### **1.4 Add Webhook Validation Tests** (Week 2, Days 1-2)

**Current State**: Minimal webhook testing

**Tasks**:
- [ ] Test webhook validation logic
  ```go
  func TestWebhookValidation(t *testing.T) {
      // Test pod validation
      // Test deployment validation
      // Test enforcement modes (monitor, audit, enforce)
  }
  ```

- [ ] Test circuit breaker
  ```go
  func TestCircuitBreaker(t *testing.T) {
      // Test error threshold (50%)
      // Test auto-trip
      // Test recovery after cooldown
      // Test manual reset
  }
  ```

- [ ] Test webhook integration with advanced policies
  ```go
  func TestWebhookAdvancedPolicies(t *testing.T) {
      // Test namespace scoping
      // Test time-based activation
      // Test exemptions
  }
  ```

**Acceptance Criteria**:
- ✅ 10+ webhook test cases
- ✅ Circuit breaker fully tested
- ✅ Integration tests with policies

---

### **1.5 Integration Test Suite** (Week 2, Days 3-4)

**Current State**: Basic integration tests exist

**Tasks**:
- [ ] Enhance full stack integration test
  ```bash
  # test/integration/full_stack_test.sh
  # Already created, enhance with:
  - Multi-cluster scenarios
  - Policy template testing
  - Alert webhook testing
  - Performance benchmarks
  ```

- [ ] Create multi-cluster test scenarios
  ```go
  func TestMultiClusterEnforcement(t *testing.T) {
      // 1. Create 3 kind clusters
      // 2. Deploy operator on cluster 1
      // 3. Add cluster 2 and 3 as targets
      // 4. Deploy ClusterSpec
      // 5. Verify policies synced to all clusters
      // 6. Verify fleet aggregation works
  }
  ```

- [ ] Test upgrade paths
  ```go
  func TestUpgrade_v030_to_v040(t *testing.T) {
      // 1. Deploy v0.3.0
      // 2. Create ClusterSpecs
      // 3. Upgrade to v0.4.0
      // 4. Verify CRDs migrated
      // 5. Verify no data loss
  }
  ```

**Acceptance Criteria**:
- ✅ Multi-cluster test suite passes
- ✅ Upgrade path tested
- ✅ Integration tests run in CI

---

### **1.6 Load & Performance Testing** (Week 2, Day 5)

**Current State**: No performance testing

**Tasks**:
- [ ] Create load test scenarios
  ```go
  // test/performance/load_test.go

  func BenchmarkReconciliation_10Clusters(b *testing.B) {
      // Test reconciliation with 10 clusters
  }

  func BenchmarkReconciliation_50Clusters(b *testing.B) {
      // Test reconciliation with 50 clusters
  }

  func BenchmarkReconciliation_100Clusters(b *testing.B) {
      // Test reconciliation with 100 clusters
  }
  ```

- [ ] Create webhook performance tests
  ```go
  func BenchmarkWebhookValidation_HighLoad(b *testing.B) {
      // Simulate 1000 requests/sec
  }
  ```

- [ ] Establish performance baselines
  ```markdown
  ## Performance Baselines (v0.4.0)

  - Reconciliation (10 clusters): <5s
  - Reconciliation (50 clusters): <30s
  - Reconciliation (100 clusters): <2min
  - Webhook latency (p95): <100ms
  - Webhook throughput: >500 req/sec
  ```

**Acceptance Criteria**:
- ✅ 3 load test scenarios
- ✅ Performance baselines established
- ✅ Results documented

---

### **Track 1 Deliverables**

**Code**:
- ✅ `test/fixtures/fake_client.go` - Fake client infrastructure
- ✅ `test/integration/leader_election_test.go` - Leader election tests
- ✅ `pkg/policy/advanced_test.go` - Advanced policy tests
- ✅ `pkg/webhooks/webhook_test.go` - Webhook tests
- ✅ `test/performance/load_test.go` - Performance benchmarks

**Documentation**:
- ✅ Test coverage report (>80%)
- ✅ Performance baseline report
- ✅ Testing best practices guide

**Metrics**:
- Test coverage: 50% → **80%+** ✅
- Skipped tests: 8 → **0** ✅
- Performance baselines: **Established** ✅

---

## Track 2: Alert Integrations (Weeks 2-3)

**Goal**: Add Slack, PagerDuty, and generic webhook alerting

### **2.1 Alert Infrastructure** (Week 2, Day 5 - Week 3, Day 1)

**Tasks**:
- [ ] Create alert types
  ```go
  // pkg/alerts/types.go
  package alerts

  type AlertLevel string

  const (
      AlertLevelInfo     AlertLevel = "info"
      AlertLevelWarning  AlertLevel = "warning"
      AlertLevelCritical AlertLevel = "critical"
  )

  type Alert struct {
      Level       AlertLevel
      Title       string
      Description string
      Source      string  // e.g., "ClusterSpec/prod-cluster"
      Timestamp   time.Time
      Labels      map[string]string
      Metadata    map[string]interface{}
  }

  type Notifier interface {
      Send(ctx context.Context, alert Alert) error
      Name() string
  }
  ```

- [ ] Create alert manager
  ```go
  // pkg/alerts/manager.go

  type Manager struct {
      notifiers []Notifier
      mu        sync.RWMutex
  }

  func (m *Manager) AddNotifier(n Notifier) {
      m.mu.Lock()
      defer m.mu.Unlock()
      m.notifiers = append(m.notifiers, n)
  }

  func (m *Manager) Send(ctx context.Context, alert Alert) error {
      m.mu.RLock()
      defer m.mu.RUnlock()

      var errs []error
      for _, notifier := range m.notifiers {
          if err := notifier.Send(ctx, alert); err != nil {
              errs = append(errs, fmt.Errorf("%s: %w", notifier.Name(), err))
          }
      }

      if len(errs) > 0 {
          return fmt.Errorf("alert failures: %v", errs)
      }
      return nil
  }
  ```

**Acceptance Criteria**:
- ✅ Alert types defined
- ✅ Notifier interface created
- ✅ Alert manager implemented
- ✅ Unit tests for alert manager

---

### **2.2 Slack Integration** (Week 3, Day 2)

**Tasks**:
- [ ] Implement Slack notifier
  ```go
  // pkg/alerts/slack.go

  type SlackNotifier struct {
      WebhookURL string
      Channel    string
      Username   string
      IconEmoji  string
  }

  func (s *SlackNotifier) Send(ctx context.Context, alert Alert) error {
      payload := map[string]interface{}{
          "channel":   s.Channel,
          "username":  s.Username,
          "icon_emoji": s.IconEmoji,
          "attachments": []map[string]interface{}{{
              "color":     s.alertColor(alert.Level),
              "title":     alert.Title,
              "text":      alert.Description,
              "footer":    fmt.Sprintf("Source: %s", alert.Source),
              "ts":        alert.Timestamp.Unix(),
              "fields":    s.buildFields(alert),
          }},
      }

      return s.sendWebhook(ctx, payload)
  }

  func (s *SlackNotifier) alertColor(level AlertLevel) string {
      switch level {
      case AlertLevelCritical:
          return "danger"    // Red
      case AlertLevelWarning:
          return "warning"   // Yellow
      default:
          return "good"      // Green
      }
  }
  ```

- [ ] Add Slack configuration to AlertConfig CRD
- [ ] Add tests for Slack notifier
- [ ] Document Slack setup

**Acceptance Criteria**:
- ✅ Slack notifier implemented
- ✅ Webhook tested with real Slack
- ✅ Configuration documented

---

### **2.3 PagerDuty Integration** (Week 3, Day 3)

**Tasks**:
- [ ] Implement PagerDuty notifier
  ```go
  // pkg/alerts/pagerduty.go

  type PagerDutyNotifier struct {
      IntegrationKey string
      Client         *pagerduty.Client
  }

  func (p *PagerDutyNotifier) Send(ctx context.Context, alert Alert) error {
      event := pagerduty.Event{
          RoutingKey:  p.IntegrationKey,
          Action:      p.eventAction(alert.Level),
          DedupKey:    p.dedupKey(alert),
          Payload: &pagerduty.Payload{
              Summary:   alert.Title,
              Severity:  p.severity(alert.Level),
              Source:    alert.Source,
              Timestamp: alert.Timestamp,
              CustomDetails: alert.Metadata,
          },
      }

      return p.Client.CreateEvent(ctx, event)
  }

  func (p *PagerDutyNotifier) eventAction(level AlertLevel) string {
      if level == AlertLevelCritical {
          return "trigger"  // Create incident
      }
      return "acknowledge"
  }

  func (p *PagerDutyNotifier) severity(level AlertLevel) string {
      switch level {
      case AlertLevelCritical:
          return "critical"
      case AlertLevelWarning:
          return "warning"
      default:
          return "info"
      }
  }
  ```

- [ ] Add PagerDuty configuration to AlertConfig CRD
- [ ] Add tests with PagerDuty mock
- [ ] Document PagerDuty setup

**Acceptance Criteria**:
- ✅ PagerDuty notifier implemented
- ✅ Tested with PagerDuty sandbox
- ✅ Incident creation verified

---

### **2.4 Generic Webhook Notifier** (Week 3, Day 4)

**Tasks**:
- [ ] Implement generic webhook
  ```go
  // pkg/alerts/webhook.go

  type WebhookNotifier struct {
      URL         string
      Method      string  // POST, PUT, etc.
      Headers     map[string]string
      Template    string  // Go template for payload
      RetryPolicy RetryPolicy
  }

  func (w *WebhookNotifier) Send(ctx context.Context, alert Alert) error {
      payload, err := w.renderPayload(alert)
      if err != nil {
          return fmt.Errorf("failed to render payload: %w", err)
      }

      req, err := http.NewRequestWithContext(ctx, w.Method, w.URL, bytes.NewReader(payload))
      if err != nil {
          return err
      }

      for key, value := range w.Headers {
          req.Header.Set(key, value)
      }

      return w.sendWithRetry(req)
  }

  func (w *WebhookNotifier) renderPayload(alert Alert) ([]byte, error) {
      tmpl, err := template.New("webhook").Parse(w.Template)
      if err != nil {
          return nil, err
      }

      var buf bytes.Buffer
      if err := tmpl.Execute(&buf, alert); err != nil {
          return nil, err
      }

      return buf.Bytes(), nil
  }
  ```

- [ ] Support payload templates
- [ ] Add retry logic
- [ ] Add tests

**Acceptance Criteria**:
- ✅ Generic webhook implemented
- ✅ Template rendering works
- ✅ Retry logic tested

---

### **2.5 AlertConfig CRD** (Week 3, Day 5)

**Tasks**:
- [ ] Create AlertConfig CRD
  ```go
  // api/v1alpha1/alertconfig_types.go

  type AlertConfigSpec struct {
      // Slack integration
      Slack *SlackConfig `json:"slack,omitempty"`

      // PagerDuty integration
      PagerDuty *PagerDutyConfig `json:"pagerduty,omitempty"`

      // Generic webhooks
      Webhooks []WebhookConfig `json:"webhooks,omitempty"`

      // Alert routing rules
      Routes []AlertRoute `json:"routes,omitempty"`
  }

  type SlackConfig struct {
      Enabled    bool   `json:"enabled"`
      WebhookURL string `json:"webhookURL"`
      Channel    string `json:"channel"`
      Events     []string `json:"events,omitempty"`  // Filter by event type
  }

  type PagerDutyConfig struct {
      Enabled        bool   `json:"enabled"`
      IntegrationKey string `json:"integrationKey"`
      Severity       string `json:"severity,omitempty"`  // minimum severity
  }

  type WebhookConfig struct {
      Name     string            `json:"name"`
      URL      string            `json:"url"`
      Method   string            `json:"method,omitempty"`
      Headers  map[string]string `json:"headers,omitempty"`
      Template string            `json:"template,omitempty"`
      Events   []string          `json:"events,omitempty"`
  }

  type AlertRoute struct {
      Match       map[string]string `json:"match"`       // Label matchers
      Notifiers   []string          `json:"notifiers"`   // Notifier names
      Continue    bool              `json:"continue"`    // Continue to next route
  }
  ```

- [ ] Create AlertConfig controller
- [ ] Integrate with alert manager
- [ ] Add webhook for AlertConfig

**Example AlertConfig**:
```yaml
apiVersion: kspec.io/v1alpha1
kind: AlertConfig
metadata:
  name: production-alerts
  namespace: kspec-system
spec:
  slack:
    enabled: true
    webhookURL: https://hooks.slack.com/services/XXX
    channel: "#kspec-alerts"
    events:
      - PolicyViolation
      - DriftDetected
      - CircuitBreakerTripped

  pagerduty:
    enabled: true
    integrationKey: xxxxxxxxxxxxx
    severity: critical  # Only critical alerts

  webhooks:
    - name: custom-webhook
      url: https://example.com/kspec/alerts
      method: POST
      headers:
        Authorization: "Bearer ${WEBHOOK_TOKEN}"
      template: |
        {
          "alert": "{{ .Title }}",
          "level": "{{ .Level }}",
          "source": "{{ .Source }}",
          "timestamp": "{{ .Timestamp }}"
        }
      events:
        - PolicyViolation
        - ComplianceFailure

  routes:
    - match:
        severity: critical
      notifiers:
        - slack
        - pagerduty
      continue: false

    - match:
        severity: warning
      notifiers:
        - slack
      continue: true
```

**Acceptance Criteria**:
- ✅ AlertConfig CRD created
- ✅ Controller reconciles AlertConfig
- ✅ Alerts route correctly
- ✅ Example configs documented

---

### **2.6 Integration with Controllers** (Week 3, Weekend)

**Tasks**:
- [ ] Add alert manager to controllers
  ```go
  // controllers/clusterspec_controller.go

  type ClusterSpecReconciler struct {
      client.Client
      Scheme        *runtime.Scheme
      AlertManager  *alerts.Manager  // NEW
  }

  func (r *ClusterSpecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
      // ... existing code ...

      // Send alert on policy violation
      if len(violations) > 0 {
          r.AlertManager.Send(ctx, alerts.Alert{
              Level:       alerts.AlertLevelWarning,
              Title:       fmt.Sprintf("Policy violations in %s", clusterSpec.Name),
              Description: fmt.Sprintf("Found %d violations", len(violations)),
              Source:      fmt.Sprintf("ClusterSpec/%s", clusterSpec.Name),
              Timestamp:   time.Now(),
              Labels: map[string]string{
                  "cluster": clusterSpec.Name,
                  "type":    "PolicyViolation",
              },
          })
      }

      // Send alert on drift detection
      if driftDetected {
          r.AlertManager.Send(ctx, alerts.Alert{
              Level:       alerts.AlertLevelCritical,
              Title:       fmt.Sprintf("Configuration drift detected in %s", clusterSpec.Name),
              Description: "Cluster configuration has drifted from specification",
              Source:      fmt.Sprintf("ClusterSpec/%s", clusterSpec.Name),
              Timestamp:   time.Now(),
              Labels: map[string]string{
                  "cluster": clusterSpec.Name,
                  "type":    "DriftDetected",
              },
          })
      }
  }
  ```

- [ ] Add alerts to webhook server
  ```go
  // pkg/webhooks/server.go

  func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
      // ... existing code ...

      // Alert on circuit breaker trip
      if s.CircuitBreaker.IsTripped() {
          s.AlertManager.Send(r.Context(), alerts.Alert{
              Level:       alerts.AlertLevelCritical,
              Title:       "Circuit breaker tripped",
              Description: "Webhook circuit breaker has tripped due to high error rate",
              Source:      "Webhook",
              Timestamp:   time.Now(),
              Labels: map[string]string{
                  "type":     "CircuitBreakerTripped",
                  "endpoint": "/validate",
              },
          })
      }
  }
  ```

**Acceptance Criteria**:
- ✅ Alerts integrated into all controllers
- ✅ Alerts sent on critical events
- ✅ Alert routing tested

---

### **Track 2 Deliverables**

**Code**:
- ✅ `pkg/alerts/` - Complete alert package
- ✅ `pkg/alerts/slack.go` - Slack integration
- ✅ `pkg/alerts/pagerduty.go` - PagerDuty integration
- ✅ `pkg/alerts/webhook.go` - Generic webhooks
- ✅ `api/v1alpha1/alertconfig_types.go` - AlertConfig CRD
- ✅ `controllers/alertconfig_controller.go` - Controller

**Documentation**:
- ✅ Alert configuration guide
- ✅ Slack setup instructions
- ✅ PagerDuty setup instructions
- ✅ Webhook template examples

**Metrics**:
- Alert integrations: 0 → **3** (Slack, PD, Webhook) ✅
- AlertConfig CRD: **Created** ✅
- Alert routing: **Implemented** ✅

---

## Track 3: Technical Debt Resolution (Week 3)

**Goal**: Resolve all 18 TODOs in codebase

### **3.1 High-Priority TODOs** (Week 3, Days 1-3)

**Priority 1: Image Version**
```go
// BEFORE (pkg/enforcer/multicluster.go:184)
Image: "ghcr.io/cloudcwfranck/kspec:latest", // TODO: Use specific version

// AFTER
Image: fmt.Sprintf("ghcr.io/cloudcwfranck/kspec:%s", r.OperatorVersion),
```

**Priority 2: Policy Enforcement Tracking**
```go
// BEFORE (controllers/status.go)
PoliciesEnforced: 0, // TODO: Track enforced policies

// AFTER
func (r *ClusterSpecReconciler) countEnforcedPolicies(ctx context.Context, spec *kspecv1alpha1.ClusterSpecification) int {
    // Query Kyverno for policies with owner reference to this ClusterSpec
    // Return count of active policies
}
```

**Priority 3: Label Selector Matching**
```go
// BEFORE (pkg/policy/advanced.go)
// TODO: Implement label selector matching

// AFTER
func (m *AdvancedPolicyManager) matchesSelector(selector *metav1.LabelSelector, labels map[string]string) bool {
    labelSelector, err := metav1.LabelSelectorAsSelector(selector)
    if err != nil {
        return false
    }
    return labelSelector.Matches(labels.Set(labels))
}
```

**Priority 4: Parameter Substitution**
```go
// BEFORE (pkg/policy/advanced.go)
// TODO: Apply parameter substitution to policy fields

// AFTER
func (m *AdvancedPolicyManager) applyParametersToPolicy(policy *PolicyDefinition, params map[string]interface{}) *PolicyDefinition {
    // Deep copy policy
    result := policy.DeepCopy()

    // Walk policy fields and substitute {{param}} with values
    return m.substituteParameters(result, params)
}
```

**All TODOs**:
- [ ] `pkg/client/factory.go` - Add proxy support
- [ ] `pkg/enforcer/multicluster.go` - Use specific version
- [ ] `controllers/status.go` - Track policy enforcement (2 TODOs)
- [ ] `pkg/policy/advanced.go` - Label selector matching
- [ ] `pkg/policy/advanced.go` - Parameter substitution
- [ ] `pkg/policy/advanced.go` - Override logic
- [ ] `cmd/kspec/drift.go` - Use parsed duration for filtering
- [ ] `controllers/reports.go` - Convert evidence to RawExtension (3 TODOs)

**Acceptance Criteria**:
- ✅ All 18 TODOs resolved
- ✅ Code reviewed and tested
- ✅ Documentation updated

---

### **Track 3 Deliverables**

**Code Changes**:
- ✅ 18 TODOs resolved
- ✅ Unit tests for new functionality
- ✅ Integration tests updated

**Documentation**:
- ✅ Technical debt resolution report
- ✅ Code quality improvements documented

**Metrics**:
- High-priority TODOs: 4 → **0** ✅
- Total TODOs: 18 → **0** ✅

---

## Track 4: Security Hardening (Week 4)

### **4.1 Image Security** (Days 1-2)
- [ ] Sign images with cosign
- [ ] Generate SBOM with Syft
- [ ] Scan with Trivy
- [ ] Enforce image policies

### **4.2 Secret Management** (Days 2-3)
- [ ] Implement secret rotation
- [ ] Add external secret store support
- [ ] Encrypt sensitive CRD fields

### **4.3 Rate Limiting** (Day 4)
- [ ] Add webhook rate limiting
- [ ] Add metrics endpoint rate limiting
- [ ] Add per-client rate limiting

### **4.4 RBAC Audit** (Day 5)
- [ ] Review and minimize permissions
- [ ] Add RBAC audit logging
- [ ] Document security model

---

## Track 5: Observability Enhancement (Week 4)

### **5.1 OpenTelemetry** (Days 1-2)
- [ ] Add distributed tracing
- [ ] Add span context propagation
- [ ] Integrate with Jaeger

### **5.2 Enhanced Dashboards** (Days 3-4)
- [ ] Executive dashboard
- [ ] Operations dashboard
- [ ] Security dashboard

### **5.3 Audit Logging** (Day 5)
- [ ] Policy enforcement audit log
- [ ] Exemption audit log
- [ ] Export to standard formats

---

## Track 6: Documentation (Week 5)

### **6.1 User Documentation**
- [ ] Production deployment guide
- [ ] Multi-cluster setup guide
- [ ] Troubleshooting guide
- [ ] Best practices guide

### **6.2 API Documentation**
- [ ] Complete API reference
- [ ] CRD field descriptions
- [ ] Metrics reference

### **6.3 Operational Runbooks**
- [ ] Incident response playbook
- [ ] Backup and recovery
- [ ] Upgrade procedures
- [ ] Scaling guide

---

## Phase 9 Timeline

```
Week 1: Testing & Quality (Part 1)
├── Day 1-2: Fix skipped tests
├── Day 3-4: Leader election tests
└── Day 5: Advanced policy tests

Week 2: Testing & Quality (Part 2) + Alert Prep
├── Day 1-2: Webhook tests
├── Day 3-4: Integration tests
├── Day 5: Load tests + Alert infrastructure

Week 3: Alerts + Technical Debt
├── Day 1: Alert infrastructure
├── Day 2: Slack integration
├── Day 3: PagerDuty integration
├── Day 4: Webhook notifier
└── Day 5: AlertConfig CRD + Technical debt

Week 4: Security + Observability
├── Day 1-2: Image security + OTel
├── Day 2-3: Secret management + Dashboards
├── Day 4: Rate limiting + Audit logging
└── Day 5: RBAC audit

Week 5: Documentation
├── Day 1-2: User documentation
├── Day 3-4: API documentation
└── Day 5: Operational runbooks
```

---

## Getting Started with Phase 9

### **For Contributors**

1. **Pick a Track**: Choose based on your expertise
2. **Check Progress**: See [PHASE_9_PROGRESS.md](./PHASE_9_PROGRESS.md)
3. **Create Branch**: `git checkout -b phase-9-track-N-description`
4. **Implement**: Follow the plan above
5. **Test**: Ensure tests pass and coverage increases
6. **Document**: Update relevant documentation
7. **PR**: Submit PR with "Phase 9: Track N" prefix

### **For Users**

Track progress and provide feedback:
- 📊 [Progress Dashboard](#) (coming soon)
- 🐛 [Report Issues](https://github.com/cloudcwfranck/kspec/issues)
- 💬 [Discussions](https://github.com/cloudcwfranck/kspec/discussions)

---

## Success Criteria

Phase 9 is complete when:
- ✅ Test coverage ≥80%
- ✅ All TODOs resolved
- ✅ Alert integrations working
- ✅ Security hardening complete
- ✅ Documentation comprehensive
- ✅ Performance benchmarks established
- ✅ All deliverables reviewed and merged

---

## Next Phase Preview

After Phase 9, we'll proceed with:

**Phase 10: Multi-Cluster CLI & Fleet Management**
- Enhanced CLI commands
- Fleet-wide operations
- DriftConfig CRD
- Cluster grouping

---

**Phase 9 Status**: 🚧 **In Progress**
**Last Updated**: December 30, 2025
**Contact**: @cloudcwfranck
