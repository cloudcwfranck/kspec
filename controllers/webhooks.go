package controllers

import (
	"context"
	"fmt"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

const (
	// ValidatingWebhookConfigName is the name of the validating webhook configuration
	ValidatingWebhookConfigName = "kspec-validating-webhook"

	// WebhookServiceName is the name of the webhook service
	WebhookServiceName = "kspec-webhook-service"

	// WebhookPath is the webhook endpoint path
	WebhookPath = "/validate"
)

// manageValidatingWebhook creates or updates the ValidatingWebhookConfiguration
func (r *ClusterSpecReconciler) manageValidatingWebhook(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
) error {
	log := log.FromContext(ctx)

	// Check if webhooks are enabled and certificate is ready
	if clusterSpec.Spec.Webhooks == nil || !clusterSpec.Spec.Webhooks.Enabled {
		log.V(1).Info("Webhooks disabled, skipping webhook configuration")
		return nil
	}

	if clusterSpec.Status.Webhooks == nil || !clusterSpec.Status.Webhooks.CertificateReady {
		log.Info("Certificate not ready, skipping webhook configuration")
		return nil
	}

	// Get webhook configuration
	failurePolicy := admissionv1.Ignore // Default to fail-open
	if clusterSpec.Spec.Webhooks.FailurePolicy == "Fail" {
		failurePolicy = admissionv1.Fail
	}

	timeoutSeconds := int32(10) // Default timeout
	if clusterSpec.Spec.Webhooks.TimeoutSeconds > 0 {
		timeoutSeconds = clusterSpec.Spec.Webhooks.TimeoutSeconds
	}

	sideEffects := admissionv1.SideEffectClassNone
	port := int32(9443)
	path := WebhookPath

	// Create webhook configuration
	webhook := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ValidatingWebhookConfigName,
			Labels: map[string]string{
				"kspec.io/component": "webhook",
			},
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name: "pod-validation.kspec.io",
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      WebhookServiceName,
						Namespace: ReportNamespace,
						Path:      &path,
						Port:      &port,
					},
					CABundle: nil, // Will be injected by cert-manager
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: []admissionv1.OperationType{
							admissionv1.Create,
							admissionv1.Update,
						},
						Rule: admissionv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				FailurePolicy:           &failurePolicy,
				SideEffects:             &sideEffects,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          &timeoutSeconds,
			},
		},
	}

	// Add cert-manager annotation for CA injection
	if webhook.Annotations == nil {
		webhook.Annotations = make(map[string]string)
	}
	webhook.Annotations["cert-manager.io/inject-ca-from"] = fmt.Sprintf("%s/%s", ReportNamespace, WebhookCertificateName)

	// Check if webhook config already exists
	existing := &admissionv1.ValidatingWebhookConfiguration{}
	err := r.Get(ctx, types.NamespacedName{Name: ValidatingWebhookConfigName}, existing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get webhook configuration: %w", err)
		}

		// Create new webhook configuration
		if err := r.Create(ctx, webhook); err != nil {
			return fmt.Errorf("failed to create webhook configuration: %w", err)
		}
		log.Info("Created ValidatingWebhookConfiguration")
	} else {
		// Update existing webhook configuration
		existing.Webhooks = webhook.Webhooks
		existing.Annotations = webhook.Annotations
		existing.Labels = webhook.Labels

		if err := r.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update webhook configuration: %w", err)
		}
		log.Info("Updated ValidatingWebhookConfiguration")
	}

	return nil
}

// cleanupValidatingWebhook removes the ValidatingWebhookConfiguration
func (r *ClusterSpecReconciler) cleanupValidatingWebhook(ctx context.Context) error {
	log := log.FromContext(ctx)

	webhook := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ValidatingWebhookConfigName,
		},
	}

	err := r.Delete(ctx, webhook)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete ValidatingWebhookConfiguration")
		return err
	}

	log.Info("Cleaned up ValidatingWebhookConfiguration")
	return nil
}
