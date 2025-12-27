package checks

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WorkloadSecurityCheck validates workload security requirements.
type WorkloadSecurityCheck struct{}

// Name returns the check name.
func (c *WorkloadSecurityCheck) Name() string {
	return "workload.security"
}

// Run executes the workload security check.
func (c *WorkloadSecurityCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Skip if not specified
	if clusterSpec.Spec.Workloads == nil {
		return &scanner.CheckResult{
			Name:    c.Name(),
			Status:  scanner.StatusSkip,
			Message: "Workload security requirements not specified in cluster spec",
		}, nil
	}

	// Get all pods
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	violations := []string{}
	evidence := make(map[string]interface{})
	violatingPods := []string{}

	// Check each pod
	for _, pod := range pods.Items {
		// Skip system namespaces
		if isSystemNamespace(pod.Namespace) {
			continue
		}

		podViolations := c.checkPod(&pod, clusterSpec.Spec.Workloads)
		if len(podViolations) > 0 {
			violations = append(violations, podViolations...)
			violatingPods = append(violatingPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
	}

	if len(violations) > 0 {
		evidence["violations"] = violations
		evidence["violating_pods"] = violatingPods
		evidence["violation_count"] = len(violations)

		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: scanner.SeverityHigh,
			Message:  fmt.Sprintf("Found %d workload security violations across %d pods", len(violations), len(violatingPods)),
			Evidence: evidence,
			Remediation: `Review and fix workload security violations:
1. Ensure containers run as non-root (securityContext.runAsNonRoot: true)
2. Disable privilege escalation (securityContext.allowPrivilegeEscalation: false)
3. Add resource limits and requests to all containers
4. Avoid privileged containers, hostNetwork, and hostPID
5. Use approved container registries
6. Use image digests instead of tags`,
		}, nil
	}

	totalPods := len(pods.Items) - countSystemPods(pods.Items)
	return &scanner.CheckResult{
		Name:    c.Name(),
		Status:  scanner.StatusPass,
		Message: fmt.Sprintf("All %d workloads comply with security requirements", totalPods),
		Evidence: map[string]interface{}{
			"total_pods": totalPods,
		},
	}, nil
}

// checkPod validates a single pod against workload requirements.
func (c *WorkloadSecurityCheck) checkPod(pod *corev1.Pod, spec *spec.WorkloadsSpec) []string {
	violations := []string{}
	podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

	// Check containers
	if spec.Containers != nil {
		for i, container := range pod.Spec.Containers {
			containerKey := fmt.Sprintf("%s[%d]:%s", podKey, i, container.Name)

			// Check required fields
			for _, req := range spec.Containers.Required {
				violation := c.checkRequiredField(pod, &container, req, containerKey)
				if violation != "" {
					violations = append(violations, violation)
				}
			}

			// Check forbidden fields
			for _, forbidden := range spec.Containers.Forbidden {
				violation := c.checkForbiddenField(pod, &container, forbidden, containerKey)
				if violation != "" {
					violations = append(violations, violation)
				}
			}
		}

		// Check init containers
		for i, container := range pod.Spec.InitContainers {
			containerKey := fmt.Sprintf("%s[init-%d]:%s", podKey, i, container.Name)

			for _, req := range spec.Containers.Required {
				violation := c.checkRequiredField(pod, &container, req, containerKey)
				if violation != "" {
					violations = append(violations, violation)
				}
			}

			for _, forbidden := range spec.Containers.Forbidden {
				violation := c.checkForbiddenField(pod, &container, forbidden, containerKey)
				if violation != "" {
					violations = append(violations, violation)
				}
			}
		}
	}

	// Check images
	if spec.Images != nil {
		for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			violation := c.checkImage(&container, spec.Images, podKey)
			if violation != "" {
				violations = append(violations, violation)
			}
		}
	}

	return violations
}

// checkRequiredField validates a required field exists and has the correct value.
func (c *WorkloadSecurityCheck) checkRequiredField(pod *corev1.Pod, container *corev1.Container, req spec.FieldRequirement, containerKey string) string {
	switch req.Key {
	case "securityContext.runAsNonRoot":
		if container.SecurityContext == nil || container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot {
			if pod.Spec.SecurityContext == nil || pod.Spec.SecurityContext.RunAsNonRoot == nil || !*pod.Spec.SecurityContext.RunAsNonRoot {
				return fmt.Sprintf("%s: missing securityContext.runAsNonRoot=true", containerKey)
			}
		}
	case "securityContext.allowPrivilegeEscalation":
		if req.Value == "false" {
			if container.SecurityContext == nil || container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
				return fmt.Sprintf("%s: allowPrivilegeEscalation must be false", containerKey)
			}
		}
	case "resources.limits.memory":
		if req.Exists != nil && *req.Exists {
			if container.Resources.Limits == nil || container.Resources.Limits.Memory().IsZero() {
				return fmt.Sprintf("%s: missing resources.limits.memory", containerKey)
			}
		}
	case "resources.requests.cpu":
		if req.Exists != nil && *req.Exists {
			if container.Resources.Requests == nil || container.Resources.Requests.Cpu().IsZero() {
				return fmt.Sprintf("%s: missing resources.requests.cpu", containerKey)
			}
		}
	case "resources.limits.cpu":
		if req.Exists != nil && *req.Exists {
			if container.Resources.Limits == nil || container.Resources.Limits.Cpu().IsZero() {
				return fmt.Sprintf("%s: missing resources.limits.cpu", containerKey)
			}
		}
	case "resources.requests.memory":
		if req.Exists != nil && *req.Exists {
			if container.Resources.Requests == nil || container.Resources.Requests.Memory().IsZero() {
				return fmt.Sprintf("%s: missing resources.requests.memory", containerKey)
			}
		}
	}
	return ""
}

// checkForbiddenField validates a forbidden field does not exist or has wrong value.
func (c *WorkloadSecurityCheck) checkForbiddenField(pod *corev1.Pod, container *corev1.Container, forbidden spec.FieldRequirement, containerKey string) string {
	switch forbidden.Key {
	case "securityContext.privileged":
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			return fmt.Sprintf("%s: privileged containers are forbidden", containerKey)
		}
	case "hostNetwork":
		if pod.Spec.HostNetwork {
			return fmt.Sprintf("%s: hostNetwork is forbidden", containerKey)
		}
	case "hostPID":
		if pod.Spec.HostPID {
			return fmt.Sprintf("%s: hostPID is forbidden", containerKey)
		}
	case "hostIPC":
		if pod.Spec.HostIPC {
			return fmt.Sprintf("%s: hostIPC is forbidden", containerKey)
		}
	}
	return ""
}

// checkImage validates image registry and digest requirements.
func (c *WorkloadSecurityCheck) checkImage(container *corev1.Container, imageSpec *spec.ImageSpec, podKey string) string {
	image := container.Image

	// Check blocked registries first
	for _, blocked := range imageSpec.BlockedRegistries {
		if matchesRegistry(image, blocked) {
			return fmt.Sprintf("%s: image uses blocked registry: %s", podKey, image)
		}
	}

	// Check allowed registries (if specified)
	if len(imageSpec.AllowedRegistries) > 0 {
		allowed := false
		for _, allowedRegistry := range imageSpec.AllowedRegistries {
			if matchesRegistry(image, allowedRegistry) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Sprintf("%s: image not from allowed registry: %s", podKey, image)
		}
	}

	// Check digest requirement
	if imageSpec.RequireDigests {
		if !strings.Contains(image, "@sha256:") {
			return fmt.Sprintf("%s: image must use digest, not tag: %s", podKey, image)
		}
	}

	return ""
}

// matchesRegistry checks if an image matches a registry pattern (supports wildcards).
func matchesRegistry(image, registryPattern string) bool {
	// Convert registry pattern to regex
	// Example: "*.azurecr.io" -> "^.*\.azurecr\.io/"
	pattern := strings.ReplaceAll(registryPattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	pattern = "^" + pattern + "/"

	matched, _ := regexp.MatchString(pattern, image)
	if matched {
		return true
	}

	// Also check without trailing slash for direct matches
	if strings.HasPrefix(image, registryPattern+"/") {
		return true
	}

	// Check if image has no registry prefix and pattern is for default registry
	if !strings.Contains(image, "/") || !strings.Contains(strings.Split(image, "/")[0], ".") {
		// Image uses default registry (docker.io)
		if registryPattern == "docker.io" || registryPattern == "*.docker.io" {
			return true
		}
	}

	return false
}

// countSystemPods counts pods in system namespaces.
func countSystemPods(pods []corev1.Pod) int {
	count := 0
	for _, pod := range pods {
		if isSystemNamespace(pod.Namespace) {
			count++
		}
	}
	return count
}
