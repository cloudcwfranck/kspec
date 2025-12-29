/*
Copyright 2025 kspec contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ComplianceChecksTotal tracks total compliance checks per cluster
	ComplianceChecksTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_compliance_checks_total",
			Help: "Total number of compliance checks performed per cluster",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec"},
	)

	// ComplianceChecksPassed tracks passed compliance checks per cluster
	ComplianceChecksPassed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_compliance_checks_passed",
			Help: "Number of compliance checks passed per cluster",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec"},
	)

	// ComplianceChecksFailed tracks failed compliance checks per cluster
	ComplianceChecksFailed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_compliance_checks_failed",
			Help: "Number of compliance checks failed per cluster",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec"},
	)

	// ComplianceScore tracks the compliance score (0-100) per cluster
	ComplianceScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_compliance_score",
			Help: "Compliance score percentage (0-100) per cluster",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec"},
	)

	// DriftDetected tracks whether drift is currently detected
	DriftDetected = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_drift_detected",
			Help: "Whether drift is currently detected (1=yes, 0=no)",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec"},
	)

	// DriftEventsTotal tracks total drift events per cluster
	DriftEventsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_drift_events_total",
			Help: "Total number of drift events detected per cluster",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec"},
	)

	// DriftEventsByType tracks drift events by type
	DriftEventsByType = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_drift_events_by_type",
			Help: "Number of drift events by type per cluster",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec", "drift_kind"},
	)

	// RemediationActions tracks total remediation actions
	RemediationActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_remediation_actions_total",
			Help: "Total number of remediation actions performed",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec", "action"},
	)

	// RemediationErrors tracks remediation errors
	RemediationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_remediation_errors_total",
			Help: "Total number of remediation errors",
		},
		[]string{"cluster_name", "cluster_uid", "cluster_spec", "error_type"},
	)

	// ClusterTargetHealthy tracks whether a cluster target is reachable
	ClusterTargetHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_cluster_target_healthy",
			Help: "Whether cluster target is healthy (1=yes, 0=no)",
		},
		[]string{"cluster_name", "namespace"},
	)

	// ClusterTargetInfo provides cluster target metadata
	ClusterTargetInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_cluster_target_info",
			Help: "Cluster target information (always 1)",
		},
		[]string{"cluster_name", "namespace", "platform", "version", "api_server"},
	)

	// ClusterTargetNodeCount tracks node count per cluster
	ClusterTargetNodeCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_cluster_target_nodes",
			Help: "Number of nodes in the cluster",
		},
		[]string{"cluster_name", "namespace"},
	)

	// ScanDuration tracks scan duration in seconds
	ScanDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kspec_scan_duration_seconds",
			Help:    "Duration of compliance scans in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"cluster_name", "cluster_spec"},
	)

	// ReconcileTotal tracks total reconciliation attempts
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_reconcile_total",
			Help: "Total number of reconciliation attempts",
		},
		[]string{"controller", "cluster_spec"},
	)

	// ReconcileErrors tracks reconciliation errors
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_reconcile_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"controller", "cluster_spec", "error_type"},
	)

	// ReconcileDuration tracks reconciliation duration
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kspec_reconcile_duration_seconds",
			Help:    "Duration of reconciliation loops in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller", "cluster_spec"},
	)

	// FleetSummaryTotal tracks fleet-wide totals
	FleetSummaryTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_fleet_summary_total",
			Help: "Fleet-wide summary metrics",
		},
		[]string{"metric_type"}, // clusters, checks, failures, etc.
	)

	// ReportsGenerated tracks total reports generated
	ReportsGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_reports_generated_total",
			Help: "Total number of reports generated",
		},
		[]string{"report_type", "cluster_name"}, // compliance, drift
	)

	// KyvernoPolicyCreated tracks Kyverno policy creation (Phase 5)
	KyvernoPolicyCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_kyverno_policy_created_total",
			Help: "Total number of Kyverno policies created",
		},
		[]string{"cluster_spec", "policy_type"},
	)

	// CertificateProvisioningDuration tracks certificate provisioning time (Phase 5)
	CertificateProvisioningDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kspec_certificate_provisioning_duration_seconds",
			Help:    "Time to provision webhook certificates",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
	)

	// CertificateRenewalTotal tracks certificate renewals (Phase 5)
	CertificateRenewalTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kspec_certificate_renewal_total",
			Help: "Total number of certificate renewals",
		},
	)
)

func init() {
	// Register all metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		ComplianceChecksTotal,
		ComplianceChecksPassed,
		ComplianceChecksFailed,
		ComplianceScore,
		DriftDetected,
		DriftEventsTotal,
		DriftEventsByType,
		RemediationActions,
		RemediationErrors,
		ClusterTargetHealthy,
		ClusterTargetInfo,
		ClusterTargetNodeCount,
		ScanDuration,
		ReconcileTotal,
		ReconcileErrors,
		ReconcileDuration,
		FleetSummaryTotal,
		ReportsGenerated,
		KyvernoPolicyCreated,
		CertificateProvisioningDuration,
		CertificateRenewalTotal,
	)
}

// RecordComplianceMetrics records compliance metrics for a cluster
func RecordComplianceMetrics(clusterName, clusterUID, clusterSpec string, total, passed, failed int) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"cluster_uid":  clusterUID,
		"cluster_spec": clusterSpec,
	}

	ComplianceChecksTotal.With(labels).Set(float64(total))
	ComplianceChecksPassed.With(labels).Set(float64(passed))
	ComplianceChecksFailed.With(labels).Set(float64(failed))

	// Calculate and record compliance score
	score := 0.0
	if total > 0 {
		score = float64(passed) / float64(total) * 100
	}
	ComplianceScore.With(labels).Set(score)
}

// RecordDriftMetrics records drift metrics for a cluster
func RecordDriftMetrics(clusterName, clusterUID, clusterSpec string, detected bool, eventCount int, eventsByType map[string]int) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"cluster_uid":  clusterUID,
		"cluster_spec": clusterSpec,
	}

	if detected {
		DriftDetected.With(labels).Set(1)
	} else {
		DriftDetected.With(labels).Set(0)
	}

	DriftEventsTotal.With(labels).Set(float64(eventCount))

	// Record events by type
	for driftKind, count := range eventsByType {
		typeLabels := prometheus.Labels{
			"cluster_name": clusterName,
			"cluster_uid":  clusterUID,
			"cluster_spec": clusterSpec,
			"drift_kind":   driftKind,
		}
		DriftEventsByType.With(typeLabels).Set(float64(count))
	}
}

// RecordRemediationAction records a remediation action
func RecordRemediationAction(clusterName, clusterUID, clusterSpec, action string) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"cluster_uid":  clusterUID,
		"cluster_spec": clusterSpec,
		"action":       action,
	}
	RemediationActions.With(labels).Inc()
}

// RecordRemediationError records a remediation error
func RecordRemediationError(clusterName, clusterUID, clusterSpec, errorType string) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"cluster_uid":  clusterUID,
		"cluster_spec": clusterSpec,
		"error_type":   errorType,
	}
	RemediationErrors.With(labels).Inc()
}

// RecordClusterTargetHealth records cluster target health status
func RecordClusterTargetHealth(clusterName, namespace string, healthy bool) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"namespace":    namespace,
	}

	if healthy {
		ClusterTargetHealthy.With(labels).Set(1)
	} else {
		ClusterTargetHealthy.With(labels).Set(0)
	}
}

// RecordClusterTargetInfo records cluster target metadata
func RecordClusterTargetInfo(clusterName, namespace, platform, version, apiServer string, nodeCount int32) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"namespace":    namespace,
		"platform":     platform,
		"version":      version,
		"api_server":   apiServer,
	}

	ClusterTargetInfo.With(labels).Set(1)

	nodeLabels := prometheus.Labels{
		"cluster_name": clusterName,
		"namespace":    namespace,
	}
	ClusterTargetNodeCount.With(nodeLabels).Set(float64(nodeCount))
}

// RecordScanDuration records the duration of a scan
func RecordScanDuration(clusterName, clusterSpec string, durationSeconds float64) {
	labels := prometheus.Labels{
		"cluster_name": clusterName,
		"cluster_spec": clusterSpec,
	}
	ScanDuration.With(labels).Observe(durationSeconds)
}

// RecordReconcile records a reconciliation attempt
func RecordReconcile(controller, clusterSpec string) {
	labels := prometheus.Labels{
		"controller":   controller,
		"cluster_spec": clusterSpec,
	}
	ReconcileTotal.With(labels).Inc()
}

// RecordReconcileError records a reconciliation error
func RecordReconcileError(controller, clusterSpec, errorType string) {
	labels := prometheus.Labels{
		"controller":   controller,
		"cluster_spec": clusterSpec,
		"error_type":   errorType,
	}
	ReconcileErrors.With(labels).Inc()
}

// RecordReconcileDuration records reconciliation duration
func RecordReconcileDuration(controller, clusterSpec string, durationSeconds float64) {
	labels := prometheus.Labels{
		"controller":   controller,
		"cluster_spec": clusterSpec,
	}
	ReconcileDuration.With(labels).Observe(durationSeconds)
}

// RecordReportGenerated records a report generation
func RecordReportGenerated(reportType, clusterName string) {
	labels := prometheus.Labels{
		"report_type":  reportType,
		"cluster_name": clusterName,
	}
	ReportsGenerated.With(labels).Inc()
}

// UpdateFleetMetrics updates fleet-wide summary metrics
func UpdateFleetMetrics(totalClusters, healthyClusters, totalChecks, passedChecks, failedChecks, clustersWithDrift int) {
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "clusters_total"}).Set(float64(totalClusters))
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "clusters_healthy"}).Set(float64(healthyClusters))
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "checks_total"}).Set(float64(totalChecks))
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "checks_passed"}).Set(float64(passedChecks))
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "checks_failed"}).Set(float64(failedChecks))
	FleetSummaryTotal.With(prometheus.Labels{"metric_type": "clusters_with_drift"}).Set(float64(clustersWithDrift))
}
