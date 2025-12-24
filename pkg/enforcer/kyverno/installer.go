package kyverno

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Installer handles Kyverno installation checks.
type Installer struct{}

// NewInstaller creates a new Kyverno installer.
func NewInstaller() *Installer {
	return &Installer{}
}

// IsInstalled checks if Kyverno is installed in the cluster.
func (i *Installer) IsInstalled(ctx context.Context, client kubernetes.Interface) (bool, error) {
	// Check for Kyverno deployment in kyverno namespace
	deployment, err := client.AppsV1().Deployments("kyverno").Get(ctx, "kyverno", metav1.GetOptions{})
	if err != nil {
		// Namespace or deployment doesn't exist
		return false, nil
	}

	// Check if deployment is running
	if deployment != nil && deployment.Status.ReadyReplicas > 0 {
		return true, nil
	}

	return false, nil
}

// GetInstallInstructions returns installation instructions for Kyverno.
func (i *Installer) GetInstallInstructions() string {
	return `Kyverno is not installed. To install Kyverno, run:

# Add Kyverno Helm repository
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update

# Install Kyverno
helm install kyverno kyverno/kyverno \\
  --namespace kyverno \\
  --create-namespace \\
  --wait

# Verify installation
kubectl get deployments -n kyverno
kubectl get pods -n kyverno

For more information, visit: https://kyverno.io/docs/installation/`
}

// GetVersion attempts to get the installed Kyverno version.
func (i *Installer) GetVersion(ctx context.Context, client kubernetes.Interface) (string, error) {
	deployment, err := client.AppsV1().Deployments("kyverno").Get(ctx, "kyverno", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Kyverno deployment: %w", err)
	}

	// Extract version from image tag
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		image := deployment.Spec.Template.Spec.Containers[0].Image
		return image, nil
	}

	return "unknown", nil
}
