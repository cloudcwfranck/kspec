// Package spec defines the cluster specification schema for kspec.
package spec

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// Validate checks if a cluster specification is valid.
func Validate(spec *ClusterSpecification) error {
	if spec == nil {
		return fmt.Errorf("spec cannot be nil")
	}

	// Validate APIVersion
	if spec.APIVersion != "kspec.dev/v1" {
		return fmt.Errorf("unsupported apiVersion: %s (expected kspec.dev/v1)", spec.APIVersion)
	}

	// Validate Kind
	if spec.Kind != "ClusterSpecification" {
		return fmt.Errorf("unsupported kind: %s (expected ClusterSpecification)", spec.Kind)
	}

	// Validate metadata
	if spec.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if spec.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}

	// Validate metadata version is valid semver
	if _, err := semver.NewVersion(spec.Metadata.Version); err != nil {
		return fmt.Errorf("metadata.version must be valid semver: %w", err)
	}

	// Validate Kubernetes version requirements
	if err := validateKubernetesSpec(&spec.Spec.Kubernetes); err != nil {
		return fmt.Errorf("invalid kubernetes spec: %w", err)
	}

	// Validate Pod Security Standards if specified
	if spec.Spec.PodSecurity != nil {
		if err := validatePodSecuritySpec(spec.Spec.PodSecurity); err != nil {
			return fmt.Errorf("invalid podSecurity spec: %w", err)
		}
	}

	return nil
}

// validateKubernetesSpec validates the Kubernetes version specification.
func validateKubernetesSpec(k *KubernetesSpec) error {
	if k.MinVersion == "" {
		return fmt.Errorf("minVersion is required")
	}

	if k.MaxVersion == "" {
		return fmt.Errorf("maxVersion is required")
	}

	minVer, err := semver.NewVersion(k.MinVersion)
	if err != nil {
		return fmt.Errorf("minVersion must be valid semver: %w", err)
	}

	maxVer, err := semver.NewVersion(k.MaxVersion)
	if err != nil {
		return fmt.Errorf("maxVersion must be valid semver: %w", err)
	}

	if minVer.GreaterThan(maxVer) {
		return fmt.Errorf("minVersion (%s) cannot be greater than maxVersion (%s)", k.MinVersion, k.MaxVersion)
	}

	// Validate excluded versions
	for _, ver := range k.ExcludedVersions {
		if _, err := semver.NewVersion(ver); err != nil {
			return fmt.Errorf("excludedVersion %s must be valid semver: %w", ver, err)
		}
	}

	return nil
}

// validatePodSecuritySpec validates the Pod Security Standards specification.
func validatePodSecuritySpec(pss *PodSecuritySpec) error {
	validLevels := map[string]bool{
		"privileged": true,
		"baseline":   true,
		"restricted": true,
	}

	if !validLevels[pss.Enforce] {
		return fmt.Errorf("enforce must be one of: privileged, baseline, restricted (got: %s)", pss.Enforce)
	}

	if !validLevels[pss.Audit] {
		return fmt.Errorf("audit must be one of: privileged, baseline, restricted (got: %s)", pss.Audit)
	}

	if !validLevels[pss.Warn] {
		return fmt.Errorf("warn must be one of: privileged, baseline, restricted (got: %s)", pss.Warn)
	}

	return nil
}
