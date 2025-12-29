package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// WebhookRequestsTotal tracks total webhook requests by result
	WebhookRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_webhook_requests_total",
			Help: "Total number of webhook validation requests",
		},
		[]string{"result"}, // result: success, error, circuit_breaker_tripped
	)

	// WebhookRequestDuration tracks webhook request latency
	WebhookRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kspec_webhook_request_duration_seconds",
			Help:    "Webhook request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	// WebhookValidationResults tracks validation outcomes
	WebhookValidationResults = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_webhook_validation_results_total",
			Help: "Total number of webhook validation results",
		},
		[]string{"result", "mode"}, // result: allowed, denied, mode: audit, enforce
	)

	// CircuitBreakerTripped indicates if circuit breaker is currently tripped
	CircuitBreakerTripped = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kspec_circuit_breaker_tripped",
			Help: "Circuit breaker status (1=tripped, 0=normal)",
		},
	)

	// CircuitBreakerErrorRate tracks current error rate
	CircuitBreakerErrorRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kspec_circuit_breaker_error_rate",
			Help: "Current error rate in circuit breaker (0.0-1.0)",
		},
	)

	// CircuitBreakerTotalRequests tracks total requests through circuit breaker
	CircuitBreakerTotalRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kspec_circuit_breaker_total_requests",
			Help: "Total requests tracked by circuit breaker",
		},
	)

	// PolicyEnforcementActions tracks enforcement actions
	PolicyEnforcementActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kspec_policy_enforcement_actions_total",
			Help: "Total number of policy enforcement actions",
		},
		[]string{"policy", "action"}, // action: allowed, denied, warned
	)

	// ActiveClusterSpecs tracks number of active ClusterSpecs
	ActiveClusterSpecs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kspec_active_cluster_specs",
			Help: "Number of active ClusterSpecs by mode",
		},
		[]string{"mode"}, // mode: monitor, audit, enforce
	)
)

func init() {
	// Register metrics with controller-runtime metrics registry
	metrics.Registry.MustRegister(
		WebhookRequestsTotal,
		WebhookRequestDuration,
		WebhookValidationResults,
		CircuitBreakerTripped,
		CircuitBreakerErrorRate,
		CircuitBreakerTotalRequests,
		PolicyEnforcementActions,
		ActiveClusterSpecs,
	)
}
