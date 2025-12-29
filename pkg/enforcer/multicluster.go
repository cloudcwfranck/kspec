package enforcer

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

// MultiClusterEnforcer manages policy enforcement across multiple clusters
type MultiClusterEnforcer struct {
	LocalKubeClient kubernetes.Interface
}

// NewMultiClusterEnforcer creates a new multi-cluster enforcer
func NewMultiClusterEnforcer(localClient kubernetes.Interface) *MultiClusterEnforcer {
	return &MultiClusterEnforcer{
		LocalKubeClient: localClient,
	}
}

// SyncEnforcementToCluster synchronizes enforcement configuration to a target cluster
func (m *MultiClusterEnforcer) SyncEnforcementToCluster(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	targetClient kubernetes.Interface,
	clusterName string,
) error {
	log := log.FromContext(ctx).WithValues("cluster", clusterName, "clusterSpec", clusterSpec.Name)

	// Skip if enforcement not enabled
	if clusterSpec.Spec.Enforcement == nil || !clusterSpec.Spec.Enforcement.Enabled {
		log.Info("Enforcement not enabled, skipping sync")
		return nil
	}

	// Step 1: Ensure kspec-system namespace exists in target cluster
	if err := m.ensureNamespace(ctx, targetClient, "kspec-system"); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Step 2: Deploy webhook server to target cluster (if webhooks enabled)
	if clusterSpec.Spec.Webhooks != nil && clusterSpec.Spec.Webhooks.Enabled {
		log.Info("Deploying webhook server to target cluster")
		if err := m.deployWebhookServer(ctx, targetClient, clusterSpec, clusterName); err != nil {
			return fmt.Errorf("failed to deploy webhook server: %w", err)
		}
	}

	// Step 3: Sync certificates to target cluster
	if clusterSpec.Spec.Webhooks != nil && clusterSpec.Spec.Webhooks.Enabled {
		log.Info("Syncing certificates to target cluster")
		if err := m.syncCertificates(ctx, targetClient, clusterSpec, clusterName); err != nil {
			return fmt.Errorf("failed to sync certificates: %w", err)
		}
	}

	log.Info("Successfully synced enforcement to cluster")
	return nil
}

// ensureNamespace ensures a namespace exists in the target cluster
func (m *MultiClusterEnforcer) ensureNamespace(ctx context.Context, client kubernetes.Interface, namespace string) error {
	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil // Namespace already exists
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "kspec",
				"app.kubernetes.io/managed-by": "kspec-controller",
			},
		},
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

// RemoveEnforcementFromCluster removes enforcement resources from a target cluster
func (m *MultiClusterEnforcer) RemoveEnforcementFromCluster(
	ctx context.Context,
	targetClient kubernetes.Interface,
	clusterName string,
	clusterSpecName string,
) error {
	log := log.FromContext(ctx).WithValues("cluster", clusterName, "clusterSpec", clusterSpecName)

	log.Info("Removing enforcement from cluster")

	// Remove webhook deployment
	err := targetClient.AppsV1().Deployments("kspec-system").Delete(
		ctx,
		fmt.Sprintf("kspec-webhook-%s", clusterSpecName),
		metav1.DeleteOptions{},
	)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete webhook deployment")
	}

	// Remove webhook service
	err = targetClient.CoreV1().Services("kspec-system").Delete(
		ctx,
		fmt.Sprintf("kspec-webhook-%s", clusterSpecName),
		metav1.DeleteOptions{},
	)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete webhook service")
	}

	log.Info("Successfully removed enforcement from cluster")
	return nil
}

// deployWebhookServer deploys the webhook server to a target cluster
func (m *MultiClusterEnforcer) deployWebhookServer(
	ctx context.Context,
	client kubernetes.Interface,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	clusterName string,
) error {
	deploymentName := fmt.Sprintf("kspec-webhook-%s", clusterSpec.Name)
	namespace := "kspec-system"

	// Define webhook deployment
	replicas := int32(2) // HA deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                          "kspec-webhook",
				"kspec.io/cluster-spec":        clusterSpec.Name,
				"app.kubernetes.io/name":       "kspec-webhook",
				"app.kubernetes.io/managed-by": "kspec-controller",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                   "kspec-webhook",
					"kspec.io/cluster-spec": clusterSpec.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                   "kspec-webhook",
						"kspec.io/cluster-spec": clusterSpec.Name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "kspec-webhook",
					Containers: []corev1.Container{
						{
							Name:  "webhook",
							Image: "ghcr.io/cloudcwfranck/kspec:latest", // TODO: Use specific version
							Args: []string{
								"webhook-server",
								"--port=9443",
								"--cluster-spec=" + clusterSpec.Name,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "webhook",
									ContainerPort: 9443,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cert",
									MountPath: "/tmp/k8s-webhook-server/serving-certs",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(9443),
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/readyz",
										Port:   intstr.FromInt(9443),
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("kspec-webhook-cert-%s", clusterSpec.Name),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create or update deployment
	_, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	} else if err == nil {
		_, err = client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to create/update deployment: %w", err)
	}

	// Create webhook service
	if err := m.createWebhookService(ctx, client, clusterSpec, namespace); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// createWebhookService creates a service for the webhook server
func (m *MultiClusterEnforcer) createWebhookService(
	ctx context.Context,
	client kubernetes.Interface,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	namespace string,
) error {
	serviceName := fmt.Sprintf("kspec-webhook-%s", clusterSpec.Name)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                   "kspec-webhook",
				"kspec.io/cluster-spec": clusterSpec.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":                   "kspec-webhook",
				"kspec.io/cluster-spec": clusterSpec.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "webhook",
					Port:       443,
					TargetPort: intstr.FromInt(9443),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	_, err := client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	} else if err == nil {
		_, err = client.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
	}

	return err
}

// syncCertificates syncs certificate configuration to target cluster
func (m *MultiClusterEnforcer) syncCertificates(
	ctx context.Context,
	client kubernetes.Interface,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	clusterName string,
) error {
	// In a real implementation, this would:
	// 1. Check if cert-manager is installed in target cluster
	// 2. Create Certificate CR in target cluster
	// 3. Wait for certificate to be ready
	// 4. Sync ValidatingWebhookConfiguration to target cluster
	//
	// For now, we'll log that this needs cert-manager in the target cluster
	log := log.FromContext(ctx)
	log.Info("Certificate sync requires cert-manager in target cluster",
		"cluster", clusterName,
		"clusterSpec", clusterSpec.Name)

	return nil
}
