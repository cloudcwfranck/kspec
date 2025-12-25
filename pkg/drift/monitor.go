package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudcwfranck/kspec/pkg/spec"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Monitor continuously monitors for drift.
type Monitor struct {
	client        kubernetes.Interface
	dynamicClient dynamic.Interface
	detector      *Detector
	remediator    *Remediator
	storage       Storage
	config        *MonitorConfig
}

// NewMonitor creates a new drift monitor.
func NewMonitor(client kubernetes.Interface, dynamicClient dynamic.Interface, config *MonitorConfig) (*Monitor, error) {
	// Create storage
	storage, err := NewStorage(config.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Monitor{
		client:        client,
		dynamicClient: dynamicClient,
		detector:      NewDetector(client, dynamicClient),
		remediator:    NewRemediator(client, dynamicClient),
		storage:       storage,
		config:        config,
	}, nil
}

// Start starts continuous monitoring.
func (m *Monitor) Start(ctx context.Context, clusterSpec *spec.ClusterSpecification) error {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Run initial check immediately
	if err := m.checkOnce(ctx, clusterSpec); err != nil {
		fmt.Printf("[WARN] Initial drift check failed: %v\n", err)
	}

	// Then check periodically
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.checkOnce(ctx, clusterSpec); err != nil {
				fmt.Printf("[ERROR] Drift check failed: %v\n", err)
			}
		}
	}
}

// checkOnce performs a single drift check.
func (m *Monitor) checkOnce(ctx context.Context, clusterSpec *spec.ClusterSpecification) error {
	// Detect drift
	report, err := m.detector.Detect(ctx, clusterSpec, DetectOptions{
		EnabledTypes: m.config.EnabledTypes,
	})
	if err != nil {
		return fmt.Errorf("drift detection failed: %w", err)
	}

	// Store all events
	for _, event := range report.Events {
		if err := m.storage.Store(event); err != nil {
			fmt.Printf("[WARN] Failed to store drift event: %v\n", err)
		}
	}

	// Auto-remediate if configured and drift detected
	if m.config.AutoRemediate && report.Drift.Detected {
		remediateOpts := RemediateOptions{
			DryRun: false,
			Types:  m.config.RemediateTypes,
		}

		if err := m.remediator.Remediate(ctx, clusterSpec, report, remediateOpts); err != nil {
			fmt.Printf("[ERROR] Auto-remediation failed: %v\n", err)
		} else {
			fmt.Printf("[OK] Auto-remediated %d drift events\n", len(report.Events))
		}
	}

	// Print summary
	if report.Drift.Detected {
		fmt.Printf("[DRIFT] Detected %d events (severity: %s)\n",
			report.Drift.Counts.Total, report.Drift.Severity)
	}

	return nil
}

// GetHistory returns drift history.
func (m *Monitor) GetHistory(since time.Time) (*DriftHistory, error) {
	return m.storage.GetHistory(since)
}
