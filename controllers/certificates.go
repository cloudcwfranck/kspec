package controllers

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/enforcer/certmanager"
)

const (
	// WebhookCertificateName is the name of the webhook certificate
	WebhookCertificateName = "kspec-webhook-cert"

	// WebhookSecretName is the name of the secret holding the webhook certificate
	WebhookSecretName = "kspec-webhook-tls"

	// DefaultCertDuration is the default certificate validity duration (90 days)
	DefaultCertDuration = 90 * 24 * time.Hour

	// DefaultCertRenewBefore is the default renewal window (30 days before expiry)
	DefaultCertRenewBefore = 30 * 24 * time.Hour
)

// manageCertificate handles certificate lifecycle for webhooks
func (r *ClusterSpecReconciler) manageCertificate(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	dynamicClient dynamic.Interface,
) (bool, error) {
	log := log.FromContext(ctx)

	// Check if webhooks are enabled
	if clusterSpec.Spec.Webhooks == nil || !clusterSpec.Spec.Webhooks.Enabled {
		log.V(1).Info("Webhooks disabled, skipping certificate management")
		return false, nil
	}

	// Get certificate configuration
	certConfig := clusterSpec.Spec.Webhooks.Certificate
	if certConfig == nil {
		// Use default cert-manager issuer if not specified
		certConfig = &kspecv1alpha1.CertificateSpec{
			Issuer:     "selfsigned-issuer",
			IssuerKind: "ClusterIssuer",
		}
		log.Info("No certificate configuration specified, using default selfsigned-issuer")
	}

	// Create webhook DNS names
	webhookServiceName := "kspec-webhook-service"
	webhookNamespace := ReportNamespace // kspec-system
	dnsNames := []string{
		webhookServiceName,
		fmt.Sprintf("%s.%s", webhookServiceName, webhookNamespace),
		fmt.Sprintf("%s.%s.svc", webhookServiceName, webhookNamespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", webhookServiceName, webhookNamespace),
	}

	// Create Certificate resource
	issuerRef := certmanager.IssuerRef{
		Name: certConfig.Issuer,
		Kind: certConfig.IssuerKind,
		Group: "cert-manager.io",
	}

	cert := certmanager.NewCertificate(
		webhookNamespace,
		WebhookCertificateName,
		WebhookSecretName,
		dnsNames,
		issuerRef,
	)

	// Set duration and renewal
	duration := metav1.Duration{Duration: DefaultCertDuration}
	renewBefore := metav1.Duration{Duration: DefaultCertRenewBefore}
	cert.Spec.Duration = &duration
	cert.Spec.RenewBefore = &renewBefore

	// Add ownership labels
	if cert.Labels == nil {
		cert.Labels = make(map[string]string)
	}
	cert.Labels["kspec.io/cluster-spec"] = clusterSpec.Name
	cert.Labels["kspec.io/component"] = "webhook"

	// Convert to unstructured for dynamic client
	unstructuredCert, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cert)
	if err != nil {
		return false, fmt.Errorf("failed to convert certificate to unstructured: %w", err)
	}

	u := &unstructured.Unstructured{Object: unstructuredCert}
	u.SetGroupVersionKind(cert.GroupVersionKind())

	// Apply certificate using dynamic client
	certResource := dynamicClient.Resource(certmanager.CertificateGVR()).Namespace(webhookNamespace)
	_, err = certResource.Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		// If already exists, update it
		if client.IgnoreAlreadyExists(err) == nil {
			_, err = certResource.Update(ctx, u, metav1.UpdateOptions{})
			if err != nil {
				log.Error(err, "Failed to update certificate")
				return false, err
			}
			log.V(1).Info("Updated webhook certificate")
		} else {
			log.Error(err, "Failed to create certificate")
			return false, err
		}
	} else {
		log.Info("Created webhook certificate")
	}

	// Check certificate status
	certReady, err := r.checkCertificateStatus(ctx, dynamicClient, webhookNamespace, WebhookCertificateName)
	if err != nil {
		log.Error(err, "Failed to check certificate status")
		return false, err
	}

	if certReady {
		log.Info("Webhook certificate is ready")
	} else {
		log.Info("Webhook certificate is not ready yet")
	}

	return certReady, nil
}

// checkCertificateStatus checks if a certificate is ready
func (r *ClusterSpecReconciler) checkCertificateStatus(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	namespace string,
	name string,
) (bool, error) {
	// Get the certificate
	certResource := dynamicClient.Resource(certmanager.CertificateGVR()).Namespace(namespace)
	u, err := certResource.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get certificate: %w", err)
	}

	// Convert to Certificate type
	cert := &certmanager.Certificate{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cert)
	if err != nil {
		return false, fmt.Errorf("failed to convert unstructured to certificate: %w", err)
	}

	// Check if certificate is ready
	return certmanager.IsCertificateReady(cert), nil
}

// cleanupCertificate removes the webhook certificate
func (r *ClusterSpecReconciler) cleanupCertificate(
	ctx context.Context,
	dynamicClient dynamic.Interface,
) error {
	log := log.FromContext(ctx)

	// Delete the certificate
	certResource := dynamicClient.Resource(certmanager.CertificateGVR()).Namespace(ReportNamespace)
	err := certResource.Delete(ctx, WebhookCertificateName, metav1.DeleteOptions{})
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete certificate")
		return err
	}

	log.Info("Cleaned up webhook certificate")
	return nil
}

// updateWebhookStatus updates the webhook status fields
func (r *ClusterSpecReconciler) updateWebhookStatus(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	certificateReady bool,
) {
	if clusterSpec.Status.Webhooks == nil {
		clusterSpec.Status.Webhooks = &kspecv1alpha1.WebhooksStatus{}
	}

	// Update webhook status
	if clusterSpec.Spec.Webhooks != nil && clusterSpec.Spec.Webhooks.Enabled {
		clusterSpec.Status.Webhooks.Active = certificateReady // Only active if cert is ready
		clusterSpec.Status.Webhooks.CertificateReady = certificateReady
		clusterSpec.Status.Webhooks.ErrorRate = 0.0 // Will be updated by circuit breaker in Phase 4
		clusterSpec.Status.Webhooks.CircuitBreakerTripped = false
	} else {
		clusterSpec.Status.Webhooks.Active = false
		clusterSpec.Status.Webhooks.CertificateReady = false
		clusterSpec.Status.Webhooks.ErrorRate = 0.0
		clusterSpec.Status.Webhooks.CircuitBreakerTripped = false
	}
}
