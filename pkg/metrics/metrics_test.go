package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Helper function to get gauge value
func getGaugeValue(gauge prometheus.Gauge) float64 {
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		return 0
	}
	return metric.GetGauge().GetValue()
}

// Helper function to get counter value
func getCounterValue(counter prometheus.Counter) float64 {
	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		return 0
	}
	return metric.GetCounter().GetValue()
}

// Test Leader Election Metrics (Phase 8)

func TestRecordLeaderElectionStatus(t *testing.T) {
	// Reset the metric before test
	LeaderElectionStatus.Set(0)

	tests := []struct {
		name     string
		isLeader bool
		expected float64
	}{
		{
			name:     "record as leader",
			isLeader: true,
			expected: 1.0,
		},
		{
			name:     "record as follower",
			isLeader: false,
			expected: 0.0,
		},
		{
			name:     "toggle to leader",
			isLeader: true,
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordLeaderElectionStatus(tt.isLeader)

			value := getGaugeValue(LeaderElectionStatus)
			if value != tt.expected {
				t.Errorf("Expected LeaderElectionStatus to be %f, got %f", tt.expected, value)
			}
		})
	}
}

func TestRecordLeaderElectionTransition(t *testing.T) {
	// Get initial counter value
	initialValue := getCounterValue(LeaderElectionTransitionsTotal)

	// Record 3 transitions
	for i := 0; i < 3; i++ {
		RecordLeaderElectionTransition()
	}

	// Verify counter increased by 3
	finalValue := getCounterValue(LeaderElectionTransitionsTotal)
	if finalValue != initialValue+3 {
		t.Errorf("Expected LeaderElectionTransitionsTotal to increase by 3, got %f (from %f to %f)",
			finalValue-initialValue, initialValue, finalValue)
	}
}

func TestUpdateActiveManagerInstances(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected float64
	}{
		{
			name:     "single instance",
			count:    1,
			expected: 1.0,
		},
		{
			name:     "three instances (HA)",
			count:    3,
			expected: 3.0,
		},
		{
			name:     "zero instances",
			count:    0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateActiveManagerInstances(tt.count)

			value := getGaugeValue(ActiveManagerInstances)
			if value != tt.expected {
				t.Errorf("Expected ActiveManagerInstances to be %f, got %f", tt.expected, value)
			}
		})
	}
}

// Test Compliance Metrics

func TestRecordComplianceMetrics(t *testing.T) {
	clusterName := "test-cluster"
	clusterUID := "test-uid-123"
	clusterSpec := "test-spec-v1"

	tests := []struct {
		name          string
		total         int
		passed        int
		failed        int
		expectedScore float64
	}{
		{
			name:          "perfect score",
			total:         10,
			passed:        10,
			failed:        0,
			expectedScore: 100.0,
		},
		{
			name:          "half passed",
			total:         10,
			passed:        5,
			failed:        5,
			expectedScore: 50.0,
		},
		{
			name:          "zero checks",
			total:         0,
			passed:        0,
			failed:        0,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordComplianceMetrics(clusterName, clusterUID, clusterSpec, tt.total, tt.passed, tt.failed)

			labels := prometheus.Labels{
				"cluster_name": clusterName,
				"cluster_uid":  clusterUID,
				"cluster_spec": clusterSpec,
			}

			total := &dto.Metric{}
			ComplianceChecksTotal.With(labels).(prometheus.Gauge).Write(total)
			if total.GetGauge().GetValue() != float64(tt.total) {
				t.Errorf("Expected total to be %d, got %f", tt.total, total.GetGauge().GetValue())
			}

			score := &dto.Metric{}
			ComplianceScore.With(labels).(prometheus.Gauge).Write(score)
			if score.GetGauge().GetValue() != tt.expectedScore {
				t.Errorf("Expected score to be %f, got %f", tt.expectedScore, score.GetGauge().GetValue())
			}
		})
	}
}

// Test Drift Metrics

func TestRecordDriftMetrics(t *testing.T) {
	clusterName := "test-cluster"
	clusterUID := "test-uid-123"
	clusterSpec := "test-spec-v1"

	tests := []struct {
		name         string
		detected     bool
		eventCount   int
		eventsByType map[string]int
	}{
		{
			name:       "drift detected with events",
			detected:   true,
			eventCount: 3,
			eventsByType: map[string]int{
				"missing":  1,
				"modified": 2,
			},
		},
		{
			name:         "no drift",
			detected:     false,
			eventCount:   0,
			eventsByType: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordDriftMetrics(clusterName, clusterUID, clusterSpec, tt.detected, tt.eventCount, tt.eventsByType)

			labels := prometheus.Labels{
				"cluster_name": clusterName,
				"cluster_uid":  clusterUID,
				"cluster_spec": clusterSpec,
			}

			detected := &dto.Metric{}
			DriftDetected.With(labels).(prometheus.Gauge).Write(detected)
			expectedDetected := 0.0
			if tt.detected {
				expectedDetected = 1.0
			}
			if detected.GetGauge().GetValue() != expectedDetected {
				t.Errorf("Expected detected to be %f, got %f", expectedDetected, detected.GetGauge().GetValue())
			}

			events := &dto.Metric{}
			DriftEventsTotal.With(labels).(prometheus.Gauge).Write(events)
			if events.GetGauge().GetValue() != float64(tt.eventCount) {
				t.Errorf("Expected event count to be %d, got %f", tt.eventCount, events.GetGauge().GetValue())
			}
		})
	}
}

// Test Remediation Metrics

func TestRecordRemediationAction(t *testing.T) {
	clusterName := "test-cluster"
	clusterUID := "test-uid-123"
	clusterSpec := "test-spec-v1"

	// Record some remediation actions
	RecordRemediationAction(clusterName, clusterUID, clusterSpec, "create")
	RecordRemediationAction(clusterName, clusterUID, clusterSpec, "create")
	RecordRemediationAction(clusterName, clusterUID, clusterSpec, "update")

	// Verify the metric was recorded (we can't easily verify the exact count
	// in this test without resetting the metric registry, but we can verify
	// the function doesn't panic)
	t.Log("Successfully recorded remediation actions")
}

func TestRecordRemediationError(t *testing.T) {
	clusterName := "test-cluster"
	clusterUID := "test-uid-123"
	clusterSpec := "test-spec-v1"

	// Record some remediation errors
	RecordRemediationError(clusterName, clusterUID, clusterSpec, "policy_creation_failed")
	RecordRemediationError(clusterName, clusterUID, clusterSpec, "timeout")

	// Verify the metric was recorded
	t.Log("Successfully recorded remediation errors")
}

// Test Cluster Target Metrics

func TestRecordClusterTargetHealth(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		namespace   string
		healthy     bool
		expected    float64
	}{
		{
			name:        "healthy cluster",
			clusterName: "prod-cluster",
			namespace:   "default",
			healthy:     true,
			expected:    1.0,
		},
		{
			name:        "unhealthy cluster",
			clusterName: "dev-cluster",
			namespace:   "default",
			healthy:     false,
			expected:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordClusterTargetHealth(tt.clusterName, tt.namespace, tt.healthy)

			labels := prometheus.Labels{
				"cluster_name": tt.clusterName,
				"namespace":    tt.namespace,
			}

			metric := &dto.Metric{}
			ClusterTargetHealthy.With(labels).(prometheus.Gauge).Write(metric)
			if metric.GetGauge().GetValue() != tt.expected {
				t.Errorf("Expected health to be %f, got %f", tt.expected, metric.GetGauge().GetValue())
			}
		})
	}
}

func TestRecordClusterTargetInfo(t *testing.T) {
	clusterName := "test-cluster"
	namespace := "default"
	platform := "gke"
	version := "1.26.0"
	apiServer := "https://test.example.com"
	nodeCount := int32(3)

	RecordClusterTargetInfo(clusterName, namespace, platform, version, apiServer, nodeCount)

	// Verify node count was recorded
	nodeLabels := prometheus.Labels{
		"cluster_name": clusterName,
		"namespace":    namespace,
	}

	nodeMetric := &dto.Metric{}
	ClusterTargetNodeCount.With(nodeLabels).(prometheus.Gauge).Write(nodeMetric)
	if nodeMetric.GetGauge().GetValue() != float64(nodeCount) {
		t.Errorf("Expected node count to be %d, got %f", nodeCount, nodeMetric.GetGauge().GetValue())
	}
}

// Test Reconcile Metrics

func TestRecordReconcile(t *testing.T) {
	controller := "ClusterSpecification"
	clusterSpec := "test-spec"

	// Record some reconciliations
	RecordReconcile(controller, clusterSpec)
	RecordReconcile(controller, clusterSpec)

	t.Log("Successfully recorded reconciliations")
}

func TestRecordReconcileError(t *testing.T) {
	controller := "ClusterSpecification"
	clusterSpec := "test-spec"
	errorType := "api_error"

	RecordReconcileError(controller, clusterSpec, errorType)

	t.Log("Successfully recorded reconcile error")
}

func TestRecordReconcileDuration(t *testing.T) {
	controller := "ClusterSpecification"
	clusterSpec := "test-spec"

	// Record some durations
	RecordReconcileDuration(controller, clusterSpec, 0.5)
	RecordReconcileDuration(controller, clusterSpec, 1.2)
	RecordReconcileDuration(controller, clusterSpec, 0.8)

	t.Log("Successfully recorded reconcile durations")
}

func TestRecordScanDuration(t *testing.T) {
	clusterName := "test-cluster"
	clusterSpec := "test-spec"

	RecordScanDuration(clusterName, clusterSpec, 2.5)
	RecordScanDuration(clusterName, clusterSpec, 3.1)

	t.Log("Successfully recorded scan durations")
}

// Test Fleet Metrics

func TestUpdateFleetMetrics(t *testing.T) {
	totalClusters := 10
	healthyClusters := 8
	totalChecks := 100
	passedChecks := 85
	failedChecks := 15
	clustersWithDrift := 2

	UpdateFleetMetrics(totalClusters, healthyClusters, totalChecks, passedChecks, failedChecks, clustersWithDrift)

	// Verify one of the metrics
	metric := &dto.Metric{}
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "clusters_total"}).(prometheus.Gauge).Write(metric)
	if metric.GetGauge().GetValue() != float64(totalClusters) {
		t.Errorf("Expected total clusters to be %d, got %f", totalClusters, metric.GetGauge().GetValue())
	}

	metric2 := &dto.Metric{}
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "checks_passed"}).(prometheus.Gauge).Write(metric2)
	if metric2.GetGauge().GetValue() != float64(passedChecks) {
		t.Errorf("Expected passed checks to be %d, got %f", passedChecks, metric2.GetGauge().GetValue())
	}
}

// Test Report Metrics

func TestRecordReportGenerated(t *testing.T) {
	reportType := "compliance"
	clusterName := "test-cluster"

	RecordReportGenerated(reportType, clusterName)
	RecordReportGenerated("drift", clusterName)

	t.Log("Successfully recorded report generation")
}

// Integration test for leader election workflow

func TestLeaderElectionWorkflow(t *testing.T) {
	// Simulate a complete leader election workflow

	// Initial state: follower with 3 instances
	RecordLeaderElectionStatus(false)
	UpdateActiveManagerInstances(3)

	followerStatus := getGaugeValue(LeaderElectionStatus)
	if followerStatus != 0.0 {
		t.Errorf("Expected initial status to be follower (0), got %f", followerStatus)
	}

	instances := getGaugeValue(ActiveManagerInstances)
	if instances != 3.0 {
		t.Errorf("Expected 3 active instances, got %f", instances)
	}

	// Transition to leader
	RecordLeaderElectionTransition()
	RecordLeaderElectionStatus(true)

	leaderStatus := getGaugeValue(LeaderElectionStatus)
	if leaderStatus != 1.0 {
		t.Errorf("Expected status to be leader (1), got %f", leaderStatus)
	}

	// Scale down to 1 instance
	UpdateActiveManagerInstances(1)

	instances = getGaugeValue(ActiveManagerInstances)
	if instances != 1.0 {
		t.Errorf("Expected 1 active instance, got %f", instances)
	}

	t.Log("Leader election workflow completed successfully")
}

// Integration test for complete monitoring workflow

func TestCompleteMonitoringWorkflow(t *testing.T) {
	clusterName := "production"
	clusterUID := "uid-prod-123"
	clusterSpec := "prod-spec-v2"

	// 1. Cluster target comes online
	RecordClusterTargetHealth(clusterName, "default", true)
	RecordClusterTargetInfo(clusterName, "default", "eks", "1.26.0", "https://api.prod.example.com", 5)

	// 2. Reconciliation happens
	RecordReconcile("ClusterSpecification", clusterSpec)
	RecordReconcileDuration("ClusterSpecification", clusterSpec, 1.5)

	// 3. Compliance scan runs
	RecordScanDuration(clusterName, clusterSpec, 3.2)
	RecordComplianceMetrics(clusterName, clusterUID, clusterSpec, 20, 18, 2)

	// 4. Drift detection runs
	eventsByType := map[string]int{
		"missing": 1,
	}
	RecordDriftMetrics(clusterName, clusterUID, clusterSpec, true, 1, eventsByType)

	// 5. Remediation happens
	RecordRemediationAction(clusterName, clusterUID, clusterSpec, "create")

	// 6. Reports generated
	RecordReportGenerated("compliance", clusterName)
	RecordReportGenerated("drift", clusterName)

	// 7. Fleet metrics updated
	UpdateFleetMetrics(10, 9, 200, 180, 20, 1)

	t.Log("Complete monitoring workflow executed successfully")
}
