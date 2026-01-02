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

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/alerts"
)

const (
	// ConditionTypeConfigured indicates the AlertConfig is configured
	ConditionTypeConfigured = "Configured"
)

// AlertConfigReconciler reconciles an AlertConfig object
type AlertConfigReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	AlertManager *alerts.Manager
}

// +kubebuilder:rbac:groups=kspec.io,resources=alertconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kspec.io,resources=alertconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kspec.io,resources=alertconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile performs the reconciliation loop for AlertConfig
func (r *AlertConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("alertconfig", req.NamespacedName)

	// Fetch the AlertConfig instance
	var alertConfig kspecv1alpha1.AlertConfig
	if err := r.Get(ctx, req.NamespacedName, &alertConfig); err != nil {
		log.Info("AlertConfig resource not found, clearing notifiers")
		// Clear all notifiers when AlertConfig is deleted
		r.AlertManager.Clear()
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if alerting is globally disabled
	if alertConfig.Spec.Enabled != nil && !*alertConfig.Spec.Enabled {
		log.Info("Alerting is globally disabled, clearing notifiers")
		r.AlertManager.Clear()
		r.setCondition(&alertConfig, ConditionTypeConfigured, metav1.ConditionFalse, "Disabled", "Alerting is globally disabled")
		if err := r.Status().Update(ctx, &alertConfig); err != nil {
			log.Error(err, "Failed to update AlertConfig status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Clear existing notifiers and reconfigure
	r.AlertManager.Clear()

	var errors []string

	// Configure Slack notifier if present
	if alertConfig.Spec.Slack != nil && alertConfig.Spec.Slack.Enabled {
		if err := r.configureSlackNotifier(ctx, &alertConfig); err != nil {
			log.Error(err, "Failed to configure Slack notifier")
			errors = append(errors, fmt.Sprintf("slack: %v", err))
		} else {
			log.Info("Slack notifier configured successfully")
		}
	}

	// Configure webhook notifiers
	for i, webhookConfig := range alertConfig.Spec.Webhooks {
		if err := r.configureWebhookNotifier(ctx, &alertConfig, &webhookConfig); err != nil {
			log.Error(err, "Failed to configure webhook notifier", "webhook", webhookConfig.Name)
			errors = append(errors, fmt.Sprintf("webhook[%d] %s: %v", i, webhookConfig.Name, err))
		} else {
			log.Info("Webhook notifier configured successfully", "webhook", webhookConfig.Name)
		}
	}

	// Update status
	if len(errors) > 0 {
		r.setCondition(&alertConfig, ConditionTypeConfigured, metav1.ConditionFalse, "ConfigurationErrors", fmt.Sprintf("Errors: %v", errors))
	} else {
		r.setCondition(&alertConfig, ConditionTypeConfigured, metav1.ConditionTrue, "Configured", "All notifiers configured successfully")
	}

	// Update notifier status from stats
	r.updateNotifierStatus(&alertConfig)

	if err := r.Status().Update(ctx, &alertConfig); err != nil {
		log.Error(err, "Failed to update AlertConfig status")
		return ctrl.Result{}, err
	}

	log.Info("AlertConfig reconciled successfully",
		"slack_enabled", alertConfig.Spec.Slack != nil && alertConfig.Spec.Slack.Enabled,
		"webhooks_count", len(alertConfig.Spec.Webhooks),
		"notifiers_count", len(r.AlertManager.ListNotifiers()))

	return ctrl.Result{}, nil
}

// configureSlackNotifier configures the Slack notifier from AlertConfig
func (r *AlertConfigReconciler) configureSlackNotifier(ctx context.Context, alertConfig *kspecv1alpha1.AlertConfig) error {
	slackConfig := alertConfig.Spec.Slack

	// Get webhook URL from secret or direct config
	webhookURL := slackConfig.WebhookURL
	if slackConfig.WebhookURLSecretRef != nil {
		var err error
		webhookURL, err = r.getSecretValue(ctx, alertConfig.Namespace, slackConfig.WebhookURLSecretRef)
		if err != nil {
			return fmt.Errorf("failed to get webhook URL from secret: %w", err)
		}
	}

	if webhookURL == "" {
		return fmt.Errorf("webhook URL is required but not provided")
	}

	// Set defaults
	username := slackConfig.Username
	if username == "" {
		username = "kspec-bot"
	}

	iconEmoji := slackConfig.IconEmoji
	if iconEmoji == "" {
		iconEmoji = ":shield:"
	}

	// Create Slack notifier
	notifier := alerts.NewSlackNotifier(webhookURL, slackConfig.Channel, username, iconEmoji)
	notifier.EventFilter = slackConfig.Events

	return r.AlertManager.AddNotifier(notifier)
}

// configureWebhookNotifier configures a generic webhook notifier from AlertConfig
func (r *AlertConfigReconciler) configureWebhookNotifier(ctx context.Context, alertConfig *kspecv1alpha1.AlertConfig, webhookConfig *kspecv1alpha1.WebhookConfig) error {
	// Get URL from secret or direct config
	url := webhookConfig.URL
	if webhookConfig.URLSecretRef != nil {
		var err error
		url, err = r.getSecretValue(ctx, alertConfig.Namespace, webhookConfig.URLSecretRef)
		if err != nil {
			return fmt.Errorf("failed to get URL from secret: %w", err)
		}
	}

	if url == "" {
		return fmt.Errorf("webhook URL is required but not provided")
	}

	// Get headers from secret if provided
	headers := webhookConfig.Headers
	if webhookConfig.HeadersSecretRef != nil {
		secretHeaders, err := r.getSecretData(ctx, alertConfig.Namespace, webhookConfig.HeadersSecretRef)
		if err != nil {
			return fmt.Errorf("failed to get headers from secret: %w", err)
		}
		// Merge secret headers with configured headers
		if headers == nil {
			headers = make(map[string]string)
		}
		for key, value := range secretHeaders {
			headers[key] = value
		}
	}

	// Set defaults
	method := webhookConfig.Method
	if method == "" {
		method = "POST"
	}

	retryAttempts := webhookConfig.RetryAttempts
	if retryAttempts == 0 {
		retryAttempts = 3
	}

	timeoutSeconds := webhookConfig.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 10
	}

	// Create webhook notifier
	notifier := alerts.NewWebhookNotifier(webhookConfig.Name, url, method, headers, webhookConfig.Template)
	notifier.EventFilter = webhookConfig.Events
	notifier.RetryAttempts = retryAttempts
	notifier.Timeout = time.Duration(timeoutSeconds) * time.Second

	return r.AlertManager.AddNotifier(notifier)
}

// getSecretValue retrieves a single value from a secret
func (r *AlertConfigReconciler) getSecretValue(ctx context.Context, namespace string, secretRef *kspecv1alpha1.SecretReference) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}, &secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretRef.Name, err)
	}

	key := secretRef.Key
	if key == "" {
		key = "url" // Default key
	}

	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("secret %s does not contain key %s", secretRef.Name, key)
	}

	return string(value), nil
}

// getSecretData retrieves all key-value pairs from a secret
func (r *AlertConfigReconciler) getSecretData(ctx context.Context, namespace string, secretRef *kspecv1alpha1.SecretReference) (map[string]string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secretRef.Name, err)
	}

	result := make(map[string]string)
	for key, value := range secret.Data {
		result[key] = string(value)
	}

	return result, nil
}

// updateNotifierStatus updates the notifier status from manager stats
func (r *AlertConfigReconciler) updateNotifierStatus(alertConfig *kspecv1alpha1.AlertConfig) {
	stats := r.AlertManager.GetStats()

	if alertConfig.Status.NotifierStatus == nil {
		alertConfig.Status.NotifierStatus = make(map[string]kspecv1alpha1.NotifierStatus)
	}

	for name, stat := range stats {
		status := kspecv1alpha1.NotifierStatus{
			AlertsSent:   stat.Sent,
			AlertsFailed: stat.Failed,
		}

		if !stat.LastSent.IsZero() {
			lastSent := metav1.NewTime(stat.LastSent)
			status.LastAlertTime = &lastSent
		}

		if stat.LastError != nil {
			status.LastError = stat.LastError.Error()
		}

		alertConfig.Status.NotifierStatus[name] = status
	}

	// Update global stats
	var totalSent, totalFailed int64
	var lastAlertTime *metav1.Time

	for _, stat := range stats {
		totalSent += stat.Sent
		totalFailed += stat.Failed

		if !stat.LastSent.IsZero() {
			t := metav1.NewTime(stat.LastSent)
			if lastAlertTime == nil || stat.LastSent.After(lastAlertTime.Time) {
				lastAlertTime = &t
			}
		}
	}

	alertConfig.Status.AlertsSent = totalSent
	alertConfig.Status.AlertsFailed = totalFailed
	alertConfig.Status.LastAlertTime = lastAlertTime
}

// setCondition sets a condition on the AlertConfig status
func (r *AlertConfigReconciler) setCondition(
	alertConfig *kspecv1alpha1.AlertConfig,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) {
	now := metav1.Now()

	// Find existing condition
	for i, cond := range alertConfig.Status.Conditions {
		if cond.Type == conditionType {
			// Update existing condition
			if cond.Status != status || cond.Reason != reason {
				alertConfig.Status.Conditions[i].Status = status
				alertConfig.Status.Conditions[i].Reason = reason
				alertConfig.Status.Conditions[i].Message = message
				alertConfig.Status.Conditions[i].LastTransitionTime = now
				alertConfig.Status.Conditions[i].ObservedGeneration = alertConfig.Generation
			}
			return
		}
	}

	// Add new condition
	alertConfig.Status.Conditions = append(alertConfig.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: alertConfig.Generation,
	})
}

// SetupWithManager sets up the controller with the Manager
func (r *AlertConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kspecv1alpha1.AlertConfig{}).
		Complete(r)
}

// NewAlertConfigReconciler creates a new AlertConfigReconciler
func NewAlertConfigReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	alertManager *alerts.Manager,
) *AlertConfigReconciler {
	return &AlertConfigReconciler{
		Client:       client,
		Scheme:       scheme,
		AlertManager: alertManager,
	}
}
