// Package checks contains all compliance check implementations.
package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"k8s.io/client-go/kubernetes"
)

// KubernetesVersionCheck validates cluster version against spec requirements.
type KubernetesVersionCheck struct{}

// Name returns the check identifier.
func (c *KubernetesVersionCheck) Name() string {
	return "kubernetes.version"
}

// Run executes the version check.
func (c *KubernetesVersionCheck) Run(ctx context.Context, client kubernetes.Interface, clusterSpec *spec.ClusterSpecification) (*scanner.CheckResult, error) {
	// Get cluster version
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster version: %w", err)
	}

	// Parse versions - remove 'v' prefix if present
	gitVersion := strings.TrimPrefix(version.GitVersion, "v")
	current, err := semver.NewVersion(gitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cluster version %s: %w", version.GitVersion, err)
	}

	min, err := semver.NewVersion(clusterSpec.Spec.Kubernetes.MinVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse minVersion %s: %w", clusterSpec.Spec.Kubernetes.MinVersion, err)
	}

	max, err := semver.NewVersion(clusterSpec.Spec.Kubernetes.MaxVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse maxVersion %s: %w", clusterSpec.Spec.Kubernetes.MaxVersion, err)
	}

	// Check if version is in excluded list
	for _, excludedVer := range clusterSpec.Spec.Kubernetes.ExcludedVersions {
		excluded, err := semver.NewVersion(excludedVer)
		if err != nil {
			continue // Skip invalid excluded versions
		}
		if current.Equal(excluded) {
			return &scanner.CheckResult{
				Name:     c.Name(),
				Status:   scanner.StatusFail,
				Severity: scanner.SeverityCritical,
				Message:  fmt.Sprintf("Cluster version %s is explicitly excluded", current.String()),
				Evidence: map[string]interface{}{
					"current":          current.String(),
					"excluded_version": excludedVer,
				},
				Remediation: fmt.Sprintf("Upgrade cluster to a version between %s and %s (excluding %v)", min.String(), max.String(), clusterSpec.Spec.Kubernetes.ExcludedVersions),
			}, nil
		}
	}

	// Check if version is within range
	if current.LessThan(min) || current.GreaterThan(max) {
		return &scanner.CheckResult{
			Name:     c.Name(),
			Status:   scanner.StatusFail,
			Severity: scanner.SeverityCritical,
			Message:  fmt.Sprintf("Cluster version %s is outside allowed range %s - %s", current.String(), min.String(), max.String()),
			Evidence: map[string]interface{}{
				"current":      current.String(),
				"required_min": min.String(),
				"required_max": max.String(),
			},
			Remediation: fmt.Sprintf("Upgrade cluster to Kubernetes version between %s and %s", min.String(), max.String()),
		}, nil
	}

	// Check passed
	return &scanner.CheckResult{
		Name:    c.Name(),
		Status:  scanner.StatusPass,
		Message: fmt.Sprintf("Cluster version %s is within spec range %s - %s", current.String(), min.String(), max.String()),
		Evidence: map[string]interface{}{
			"current":      current.String(),
			"required_min": min.String(),
			"required_max": max.String(),
		},
	}, nil
}
