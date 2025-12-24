package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ObservabilityCheck validates observability requirements.
type ObservabilityCheck struct{}

// Name returns the check name.
func (c *ObservabilityCheck) Name() string {
	return "observability.validation"
}

// Run executes the observability check.
func (c *ObservabilityCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Skip if not specified
	if clusterSpec.Spec.Observability == nil {
		return &scanner.CheckResult{
			Name:    c.Name(),
			Status:  scanner.StatusSkip,
			Message: "Observability requirements not specified in cluster spec",
		}, nil
	}

	violations := []string{}
	evidence := make(map[string]interface{})

	// Check metrics requirements
	if clusterSpec.Spec.Observability.Metrics != nil && clusterSpec.Spec.Observability.Metrics.Required {
		metricsViolations, metricsEvidence := c.checkMetrics(ctx, client, clusterSpec.Spec.Observability.Metrics)
		violations = append(violations, metricsViolations...)
		for k, v := range metricsEvidence {
			evidence[k] = v
		}
	}

	// Check logging requirements
	if clusterSpec.Spec.Observability.Logging != nil && clusterSpec.Spec.Observability.Logging.AuditLog != nil {
		if clusterSpec.Spec.Observability.Logging.AuditLog.Required {
			loggingViolations, loggingEvidence := c.checkAuditLog(ctx, client, clusterSpec.Spec.Observability.Logging.AuditLog)
			violations = append(violations, loggingViolations...)
			for k, v := range loggingEvidence {
				evidence[k] = v
			}
		}
	}

	if len(violations) > 0 {
		evidence["violations"] = violations
		evidence["violation_count"] = len(violations)

		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: scanner.SeverityMedium,
			Message:  fmt.Sprintf("Found %d observability violations", len(violations)),
			Evidence: evidence,
			Remediation: `Review and fix observability violations:
1. Install metrics server:
   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
2. Install Prometheus:
   helm install prometheus prometheus-community/prometheus --namespace monitoring --create-namespace
3. Configure audit logging on the API server
4. Ensure audit log retention meets requirements

Verify metrics server:
kubectl top nodes`,
		}, nil
	}

	return &scanner.CheckResult{
		Name:    c.Name(),
		Status:  scanner.StatusPass,
		Message: "Observability requirements satisfied",
		Evidence: evidence,
	}, nil
}

// checkMetrics validates metrics provider requirements.
func (c *ObservabilityCheck) checkMetrics(ctx context.Context, client kubernetes.Interface, metricsSpec *spec.MetricsSpec) ([]string, map[string]interface{}) {
	violations := []string{}
	evidence := make(map[string]interface{})

	if len(metricsSpec.Providers) == 0 {
		violations = append(violations, "Metrics required but no providers specified")
		return violations, evidence
	}

	foundProviders := []string{}
	missingProviders := []string{}

	for _, provider := range metricsSpec.Providers {
		found := false

		switch strings.ToLower(provider) {
		case "prometheus":
			found = c.checkPrometheus(ctx, client)
		case "metrics-server":
			found = c.checkMetricsServer(ctx, client)
		default:
			// Unknown provider, try generic check
			found = c.checkGenericProvider(ctx, client, provider)
		}

		if found {
			foundProviders = append(foundProviders, provider)
		} else {
			missingProviders = append(missingProviders, provider)
		}
	}

	evidence["found_providers"] = foundProviders
	evidence["missing_providers"] = missingProviders

	// At least one provider must be available
	if len(foundProviders) == 0 {
		violations = append(violations, fmt.Sprintf("No metrics providers found. Checked: %v", metricsSpec.Providers))
	}

	return violations, evidence
}

// checkPrometheus checks if Prometheus is installed.
func (c *ObservabilityCheck) checkPrometheus(ctx context.Context, client kubernetes.Interface) bool {
	// Check for Prometheus pods or deployments
	namespaces := []string{"monitoring", "prometheus", "kube-system", "default"}

	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=prometheus",
		})
		if err == nil && len(pods.Items) > 0 {
			return true
		}

		// Also check for prometheus operator
		pods, err = client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: "app=prometheus",
		})
		if err == nil && len(pods.Items) > 0 {
			return true
		}
	}

	return false
}

// checkMetricsServer checks if metrics-server is installed.
func (c *ObservabilityCheck) checkMetricsServer(ctx context.Context, client kubernetes.Interface) bool {
	// Check for metrics-server deployment
	deployment, err := client.AppsV1().Deployments("kube-system").Get(ctx, "metrics-server", metav1.GetOptions{})
	if err == nil && deployment != nil {
		return true
	}

	// Also check pods
	pods, err := client.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=metrics-server",
	})
	if err == nil && len(pods.Items) > 0 {
		return true
	}

	return false
}

// checkGenericProvider checks for a generic metrics provider by name.
func (c *ObservabilityCheck) checkGenericProvider(ctx context.Context, client kubernetes.Interface, providerName string) bool {
	// Check for pods with provider name in common namespaces
	namespaces := []string{"monitoring", "kube-system", "observability", "default"}

	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			if strings.Contains(strings.ToLower(pod.Name), strings.ToLower(providerName)) {
				return true
			}
		}
	}

	return false
}

// checkAuditLog validates audit logging requirements.
func (c *ObservabilityCheck) checkAuditLog(ctx context.Context, client kubernetes.Interface, auditSpec *spec.AuditLogSpec) ([]string, map[string]interface{}) {
	violations := []string{}
	evidence := make(map[string]interface{})

	// Check for audit policy ConfigMap
	auditPolicyFound := false

	// Common locations for audit policy
	namespaces := []string{"kube-system", "default"}
	configMapNames := []string{"audit-policy", "kube-apiserver-audit-policy", "audit"}

	for _, ns := range namespaces {
		for _, name := range configMapNames {
			cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
			if err == nil && cm != nil {
				auditPolicyFound = true
				evidence["audit_policy_configmap"] = fmt.Sprintf("%s/%s", ns, name)
				break
			}
		}
		if auditPolicyFound {
			break
		}
	}

	// Note: AuditSinks API is alpha and not commonly available
	// We rely on ConfigMap detection for audit policy validation

	if !auditPolicyFound {
		violations = append(violations, "Audit logging required but no audit policy found")
		evidence["audit_policy_found"] = false
	} else {
		evidence["audit_policy_found"] = true

		// Note: We cannot validate retention days from cluster state
		// as this is typically configured on the API server
		// We document this limitation
		evidence["retention_validation"] = "Cannot validate retention days from cluster state (server-side configuration)"

		if auditSpec.MinRetentionDays > 0 {
			evidence["required_retention_days"] = auditSpec.MinRetentionDays
			violations = append(violations, fmt.Sprintf("Audit log retention requirement (%d days) cannot be validated from cluster state - manual verification required", auditSpec.MinRetentionDays))
		}
	}

	return violations, evidence
}
