// Package scanner provides the cluster scanning functionality.
package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/kspec/kspec/pkg/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// Version is the kspec version
	Version = "1.0.0"
)

// Scanner orchestrates compliance checks against a cluster.
type Scanner struct {
	client kubernetes.Interface
	checks []Check
}

// NewScanner creates a new scanner with the given Kubernetes client.
func NewScanner(client kubernetes.Interface, checks []Check) *Scanner {
	return &Scanner{
		client: client,
		checks: checks,
	}
}

// Scan runs all checks against the cluster and returns aggregated results.
func (s *Scanner) Scan(ctx context.Context, clusterSpec *spec.ClusterSpecification) (*ScanResult, error) {
	if clusterSpec == nil {
		return nil, fmt.Errorf("cluster spec cannot be nil")
	}

	// Get cluster information
	clusterInfo, err := s.getClusterInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Run all checks
	var results []CheckResult
	for _, check := range s.checks {
		result, err := check.Run(ctx, s.client, clusterSpec)
		if err != nil {
			// If a check fails to run, record it as a failure
			results = append(results, CheckResult{
				Name:     check.Name(),
				Status:   StatusFail,
				Severity: SeverityHigh,
				Message:  fmt.Sprintf("Check failed to execute: %v", err),
			})
			continue
		}
		results = append(results, *result)
	}

	// Calculate summary
	summary := calculateSummary(results)

	// Build scan result
	scanResult := &ScanResult{
		Metadata: ScanMetadata{
			KspecVersion: Version,
			ScanTime:     time.Now().UTC().Format(time.RFC3339),
			Cluster:      *clusterInfo,
			Spec: SpecInfo{
				Name:    clusterSpec.Metadata.Name,
				Version: clusterSpec.Metadata.Version,
			},
		},
		Summary: summary,
		Results: results,
	}

	return scanResult, nil
}

// getClusterInfo retrieves information about the cluster.
func (s *Scanner) getClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	version, err := s.client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	// Get cluster UID (using kube-system namespace UID as proxy)
	ns, err := s.client.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	var clusterUID string
	if err == nil {
		clusterUID = string(ns.UID)
	}

	// Try to get cluster name from kubeconfig context (simplified)
	clusterName := "unknown"

	return &ClusterInfo{
		Name:    clusterName,
		Version: version.GitVersion,
		UID:     clusterUID,
	}, nil
}

// calculateSummary calculates summary statistics from check results.
func calculateSummary(results []CheckResult) ScanSummary {
	summary := ScanSummary{
		TotalChecks: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case StatusPass:
			summary.Passed++
		case StatusFail:
			summary.Failed++
		case StatusWarn:
			summary.Warnings++
		case StatusSkip:
			summary.Skipped++
		}
	}

	return summary
}
