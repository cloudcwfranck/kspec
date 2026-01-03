package alerts

import (
	"context"
	"time"
)

// AlertLevel represents the severity of an alert
type AlertLevel string

const (
	// AlertLevelInfo is for informational alerts
	AlertLevelInfo AlertLevel = "info"
	// AlertLevelWarning is for warning alerts
	AlertLevelWarning AlertLevel = "warning"
	// AlertLevelCritical is for critical alerts
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents a notification to be sent
type Alert struct {
	// Level is the severity level of the alert
	Level AlertLevel

	// Title is a short summary of the alert
	Title string

	// Description provides detailed information about the alert
	Description string

	// Source identifies where the alert originated (e.g., "ClusterSpec/prod-cluster")
	Source string

	// Timestamp is when the alert was created
	Timestamp time.Time

	// Labels are key-value pairs for routing and filtering
	Labels map[string]string

	// Metadata contains additional structured data
	Metadata map[string]interface{}

	// EventType identifies the type of event (for filtering)
	// Examples: DriftDetected, ComplianceFailure, PolicyViolation, CircuitBreakerTripped, RemediationPerformed
	EventType string
}

// Notifier is the interface that all alert notifiers must implement
type Notifier interface {
	// Send sends an alert
	Send(ctx context.Context, alert Alert) error

	// Name returns the name of this notifier
	Name() string

	// Enabled returns whether this notifier is currently enabled
	Enabled() bool

	// ShouldSend determines if this alert should be sent based on filters
	ShouldSend(alert Alert) bool
}

// NotifierStats tracks statistics for a notifier
type NotifierStats struct {
	Name         string
	Sent         int64
	Failed       int64
	LastSent     time.Time
	LastError    error
	LastErrorMsg string
}
