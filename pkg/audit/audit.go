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

package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EventType represents the type of audit event
type EventType string

const (
	// EventTypeComplianceScan represents a compliance scan event
	EventTypeComplianceScan EventType = "compliance_scan"

	// EventTypeDriftDetection represents a drift detection event
	EventTypeDriftDetection EventType = "drift_detection"

	// EventTypeRemediation represents a remediation action
	EventTypeRemediation EventType = "remediation"

	// EventTypeEnforcement represents an enforcement action
	EventTypeEnforcement EventType = "enforcement"

	// EventTypeClusterAccess represents cluster access attempt
	EventTypeClusterAccess EventType = "cluster_access"

	// EventTypeCredentialAccess represents credential access
	EventTypeCredentialAccess EventType = "credential_access"

	// EventTypeReportGeneration represents report generation
	EventTypeReportGeneration EventType = "report_generation"

	// EventTypeHealthCheck represents health check event
	EventTypeHealthCheck EventType = "health_check"
)

// Severity represents the severity of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType EventType              `json:"event_type"`
	Severity  Severity               `json:"severity"`
	Actor     string                 `json:"actor"` // controller, user, etc.
	Action    string                 `json:"action"`
	Resource  ResourceInfo           `json:"resource"`
	Result    string                 `json:"result"` // success, failure, partial
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ResourceInfo identifies the resource involved in the audit event
type ResourceInfo struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace,omitempty"`
	ClusterName  string `json:"cluster_name,omitempty"`
	ClusterUID   string `json:"cluster_uid,omitempty"`
	ClusterSpec  string `json:"cluster_spec,omitempty"`
	APIServerURL string `json:"api_server_url,omitempty"`
}

// Logger provides structured audit logging
type Logger struct {
	logger logr.Logger
}

// NewLogger creates a new audit logger
func NewLogger(ctx context.Context) *Logger {
	return &Logger{
		logger: log.FromContext(ctx).WithName("audit"),
	}
}

// LogEvent logs an audit event
func (l *Logger) LogEvent(event AuditEvent) {
	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert event to JSON for structured logging
	eventJSON, err := json.Marshal(event)
	if err != nil {
		l.logger.Error(err, "Failed to marshal audit event")
		return
	}

	// Log with appropriate level based on severity
	keysAndValues := []interface{}{
		"audit_event", string(eventJSON),
		"event_type", event.EventType,
		"severity", event.Severity,
		"actor", event.Actor,
		"action", event.Action,
		"resource_kind", event.Resource.Kind,
		"resource_name", event.Resource.Name,
		"cluster_name", event.Resource.ClusterName,
		"result", event.Result,
	}

	switch event.Severity {
	case SeverityError, SeverityCritical:
		if event.Error != "" {
			l.logger.Error(nil, event.Message, keysAndValues...)
		} else {
			l.logger.Info(event.Message, keysAndValues...)
		}
	default:
		l.logger.Info(event.Message, keysAndValues...)
	}
}

// LogComplianceScan logs a compliance scan event
func (l *Logger) LogComplianceScan(clusterName, clusterUID, clusterSpec string, totalChecks, passed, failed int, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Compliance scan completed"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Compliance scan failed"
		errorMsg = err.Error()
	} else if failed > 0 {
		severity = SeverityWarning
		message = "Compliance scan completed with failures"
	}

	event := AuditEvent{
		EventType: EventTypeComplianceScan,
		Severity:  severity,
		Actor:     "clusterspec-controller",
		Action:    "scan",
		Resource: ResourceInfo{
			Kind:        "ClusterSpecification",
			ClusterName: clusterName,
			ClusterUID:  clusterUID,
			ClusterSpec: clusterSpec,
		},
		Result:  result,
		Message: message,
		Metadata: map[string]interface{}{
			"total_checks":  totalChecks,
			"passed_checks": passed,
			"failed_checks": failed,
		},
		Error: errorMsg,
	}

	l.LogEvent(event)
}

// LogDriftDetection logs a drift detection event
func (l *Logger) LogDriftDetection(clusterName, clusterUID, clusterSpec string, driftDetected bool, eventCount int, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Drift detection completed"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Drift detection failed"
		errorMsg = err.Error()
	} else if driftDetected {
		severity = SeverityWarning
		message = "Drift detected"
	}

	event := AuditEvent{
		EventType: EventTypeDriftDetection,
		Severity:  severity,
		Actor:     "clusterspec-controller",
		Action:    "detect_drift",
		Resource: ResourceInfo{
			Kind:        "ClusterSpecification",
			ClusterName: clusterName,
			ClusterUID:  clusterUID,
			ClusterSpec: clusterSpec,
		},
		Result:  result,
		Message: message,
		Metadata: map[string]interface{}{
			"drift_detected": driftDetected,
			"event_count":    eventCount,
		},
		Error: errorMsg,
	}

	l.LogEvent(event)
}

// LogRemediation logs a remediation action
func (l *Logger) LogRemediation(clusterName, clusterUID, clusterSpec, resourceKind, resourceName string, action string, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Remediation applied"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Remediation failed"
		errorMsg = err.Error()
	}

	event := AuditEvent{
		EventType: EventTypeRemediation,
		Severity:  severity,
		Actor:     "clusterspec-controller",
		Action:    action,
		Resource: ResourceInfo{
			Kind:        resourceKind,
			Name:        resourceName,
			ClusterName: clusterName,
			ClusterUID:  clusterUID,
			ClusterSpec: clusterSpec,
		},
		Result:  result,
		Message: message,
		Error:   errorMsg,
	}

	l.LogEvent(event)
}

// LogEnforcement logs an enforcement action
func (l *Logger) LogEnforcement(clusterName, clusterUID, clusterSpec, policyName string, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Policy enforcement applied"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Policy enforcement failed"
		errorMsg = err.Error()
	}

	event := AuditEvent{
		EventType: EventTypeEnforcement,
		Severity:  severity,
		Actor:     "clusterspec-controller",
		Action:    "apply_policy",
		Resource: ResourceInfo{
			Kind:        "Policy",
			Name:        policyName,
			ClusterName: clusterName,
			ClusterUID:  clusterUID,
			ClusterSpec: clusterSpec,
		},
		Result:  result,
		Message: message,
		Error:   errorMsg,
	}

	l.LogEvent(event)
}

// LogClusterAccess logs a cluster access attempt
func (l *Logger) LogClusterAccess(clusterName, apiServerURL, authMode string, success bool, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Cluster access successful"
	errorMsg := ""

	if !success || err != nil {
		result = "failure"
		severity = SeverityError
		message = "Cluster access failed"
		if err != nil {
			errorMsg = err.Error()
		}
	}

	event := AuditEvent{
		EventType: EventTypeClusterAccess,
		Severity:  severity,
		Actor:     "cluster-client-factory",
		Action:    "connect",
		Resource: ResourceInfo{
			Kind:         "ClusterTarget",
			ClusterName:  clusterName,
			APIServerURL: apiServerURL,
		},
		Result:  result,
		Message: message,
		Metadata: map[string]interface{}{
			"auth_mode": authMode,
		},
		Error: errorMsg,
	}

	l.LogEvent(event)
}

// LogCredentialAccess logs credential access (without exposing secrets)
func (l *Logger) LogCredentialAccess(secretName, secretNamespace, clusterName string, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Credential access successful"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Credential access failed"
		errorMsg = err.Error()
	}

	event := AuditEvent{
		EventType: EventTypeCredentialAccess,
		Severity:  severity,
		Actor:     "cluster-client-factory",
		Action:    "read_secret",
		Resource: ResourceInfo{
			Kind:        "Secret",
			Name:        secretName,
			Namespace:   secretNamespace,
			ClusterName: clusterName,
		},
		Result:  result,
		Message: message,
		Error:   errorMsg,
	}

	l.LogEvent(event)
}

// LogReportGeneration logs report generation
func (l *Logger) LogReportGeneration(reportType, reportName, clusterName string, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Report generated"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Report generation failed"
		errorMsg = err.Error()
	}

	event := AuditEvent{
		EventType: EventTypeReportGeneration,
		Severity:  severity,
		Actor:     "clusterspec-controller",
		Action:    "create_report",
		Resource: ResourceInfo{
			Kind:        reportType,
			Name:        reportName,
			ClusterName: clusterName,
		},
		Result:  result,
		Message: message,
		Error:   errorMsg,
	}

	l.LogEvent(event)
}

// LogHealthCheck logs a health check event
func (l *Logger) LogHealthCheck(clusterName, namespace string, healthy bool, err error) {
	result := "success"
	severity := SeverityInfo
	message := "Health check successful"
	errorMsg := ""

	if err != nil {
		result = "failure"
		severity = SeverityError
		message = "Health check failed"
		errorMsg = err.Error()
	} else if !healthy {
		severity = SeverityWarning
		message = "Cluster unhealthy"
	}

	event := AuditEvent{
		EventType: EventTypeHealthCheck,
		Severity:  severity,
		Actor:     "clustertarget-controller",
		Action:    "health_check",
		Resource: ResourceInfo{
			Kind:        "ClusterTarget",
			Name:        clusterName,
			Namespace:   namespace,
			ClusterName: clusterName,
		},
		Result:  result,
		Message: message,
		Metadata: map[string]interface{}{
			"healthy": healthy,
		},
		Error: errorMsg,
	}

	l.LogEvent(event)
}
