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

package client

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

// ClusterClientFactory creates Kubernetes clients for local and remote clusters
type ClusterClientFactory struct {
	localConfig *rest.Config
	k8sClient   client.Client
}

// NewClusterClientFactory creates a new ClusterClientFactory
func NewClusterClientFactory(localConfig *rest.Config, k8sClient client.Client) *ClusterClientFactory {
	return &ClusterClientFactory{
		localConfig: localConfig,
		k8sClient:   k8sClient,
	}
}

// CreateClientsForClusterSpec creates Kubernetes clients based on ClusterSpecification
// If clusterRef is nil, returns clients for the local cluster
// If clusterRef is set, creates clients for the remote cluster defined by ClusterTarget
func (f *ClusterClientFactory) CreateClientsForClusterSpec(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
) (kubernetes.Interface, dynamic.Interface, *ClusterInfo, error) {
	// If no clusterRef, use local cluster (backwards compatible)
	if clusterSpec.Spec.ClusterRef == nil {
		return f.createLocalClients(ctx)
	}

	// Fetch ClusterTarget
	clusterTarget, err := f.getClusterTarget(ctx, clusterSpec.Spec.ClusterRef, clusterSpec.Namespace)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get ClusterTarget: %w", err)
	}

	// Create remote clients
	return f.createRemoteClients(ctx, clusterTarget)
}

// createLocalClients creates clients for the local cluster
func (f *ClusterClientFactory) createLocalClients(ctx context.Context) (kubernetes.Interface, dynamic.Interface, *ClusterInfo, error) {
	kubeClient, err := kubernetes.NewForConfig(f.localConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create local kube client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(f.localConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create local dynamic client: %w", err)
	}

	// Get cluster UID from kube-system namespace
	clusterUID, err := f.getClusterUID(ctx, kubeClient)
	if err != nil {
		// Non-fatal: continue without UID
		clusterUID = ""
	}

	// Get cluster version
	version, err := f.getClusterVersion(ctx, kubeClient)
	if err != nil {
		version = "unknown"
	}

	info := &ClusterInfo{
		Name:             "local",
		UID:              clusterUID,
		IsLocal:          true,
		APIServerURL:     f.localConfig.Host,
		Version:          version,
		Platform:         "unknown", // Could detect from nodes
		AllowEnforcement: true,      // Always allow enforcement on local cluster
	}

	return kubeClient, dynamicClient, info, nil
}

// createRemoteClients creates clients for a remote cluster defined by ClusterTarget
func (f *ClusterClientFactory) createRemoteClients(
	ctx context.Context,
	target *kspecv1alpha1.ClusterTarget,
) (kubernetes.Interface, dynamic.Interface, *ClusterInfo, error) {
	// Build REST config from ClusterTarget
	config, err := f.buildRestConfigFromTarget(ctx, target)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build REST config: %w", err)
	}

	// Create clients
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create kube client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Get cluster UID
	clusterUID, err := f.getClusterUID(ctx, kubeClient)
	if err != nil {
		// Use status UID if available
		clusterUID = target.Status.UID
	}

	// Get cluster version
	version, err := f.getClusterVersion(ctx, kubeClient)
	if err != nil {
		// Use status version if available
		version = target.Status.Version
	}

	info := &ClusterInfo{
		Name:             target.Name,
		UID:              clusterUID,
		IsLocal:          false,
		APIServerURL:     target.Spec.APIServerURL,
		Version:          version,
		Platform:         target.Status.Platform,
		AllowEnforcement: target.Spec.AllowEnforcement,
	}

	return kubeClient, dynamicClient, info, nil
}

// buildRestConfigFromTarget builds a REST config from ClusterTarget
func (f *ClusterClientFactory) buildRestConfigFromTarget(
	ctx context.Context,
	target *kspecv1alpha1.ClusterTarget,
) (*rest.Config, error) {
	switch target.Spec.AuthMode {
	case "kubeconfig":
		return f.buildConfigFromKubeconfig(ctx, target)
	case "serviceAccount":
		return f.buildConfigFromServiceAccount(ctx, target)
	case "token":
		return f.buildConfigFromToken(ctx, target)
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", target.Spec.AuthMode)
	}
}

// buildConfigFromKubeconfig builds REST config from kubeconfig in Secret
func (f *ClusterClientFactory) buildConfigFromKubeconfig(
	ctx context.Context,
	target *kspecv1alpha1.ClusterTarget,
) (*rest.Config, error) {
	if target.Spec.KubeconfigSecretRef == nil {
		return nil, fmt.Errorf("kubeconfigSecretRef is required for authMode=kubeconfig")
	}

	// Get kubeconfig from Secret
	kubeconfigData, err := GetKubeconfigFromSecret(
		ctx,
		f.k8sClient,
		target.Spec.KubeconfigSecretRef,
		target.Namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Build config from kubeconfig
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	// Apply TLS settings
	f.applyTLSSettings(config, target)

	return config, nil
}

// buildConfigFromServiceAccount builds REST config from ServiceAccount token in Secret
func (f *ClusterClientFactory) buildConfigFromServiceAccount(
	ctx context.Context,
	target *kspecv1alpha1.ClusterTarget,
) (*rest.Config, error) {
	if target.Spec.ServiceAccountSecretRef == nil {
		return nil, fmt.Errorf("serviceAccountSecretRef is required for authMode=serviceAccount")
	}

	// Get token from Secret
	token, err := GetTokenFromSecret(
		ctx,
		f.k8sClient,
		target.Spec.ServiceAccountSecretRef,
		target.Namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Build config with token auth
	config := &rest.Config{
		Host:        target.Spec.APIServerURL,
		BearerToken: token,
	}

	// Apply TLS settings
	f.applyTLSSettings(config, target)

	return config, nil
}

// buildConfigFromToken builds REST config from raw bearer token in Secret
func (f *ClusterClientFactory) buildConfigFromToken(
	ctx context.Context,
	target *kspecv1alpha1.ClusterTarget,
) (*rest.Config, error) {
	if target.Spec.TokenSecretRef == nil {
		return nil, fmt.Errorf("tokenSecretRef is required for authMode=token")
	}

	// Get token from Secret
	token, err := GetTokenFromSecret(
		ctx,
		f.k8sClient,
		target.Spec.TokenSecretRef,
		target.Namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Build config with token auth
	config := &rest.Config{
		Host:        target.Spec.APIServerURL,
		BearerToken: token,
	}

	// Apply TLS settings
	f.applyTLSSettings(config, target)

	return config, nil
}

// applyTLSSettings applies TLS settings from ClusterTarget to REST config
func (f *ClusterClientFactory) applyTLSSettings(config *rest.Config, target *kspecv1alpha1.ClusterTarget) {
	// Set CA data if provided
	if len(target.Spec.CAData) > 0 {
		config.TLSClientConfig.CAData = target.Spec.CAData
	}

	// Set InsecureSkipTLSVerify if requested
	config.TLSClientConfig.Insecure = target.Spec.InsecureSkipTLSVerify

	// TODO: Add proxy support
	// if target.Spec.ProxyURL != "" {
	//     config.Proxy = ...
	// }
}

// getClusterTarget fetches a ClusterTarget resource
func (f *ClusterClientFactory) getClusterTarget(
	ctx context.Context,
	ref *kspecv1alpha1.ClusterReference,
	defaultNamespace string,
) (*kspecv1alpha1.ClusterTarget, error) {
	namespace := ref.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}

	clusterTarget := &kspecv1alpha1.ClusterTarget{}
	key := types.NamespacedName{
		Name:      ref.Name,
		Namespace: namespace,
	}

	if err := f.k8sClient.Get(ctx, key, clusterTarget); err != nil {
		return nil, fmt.Errorf("failed to get ClusterTarget %s/%s: %w", namespace, ref.Name, err)
	}

	return clusterTarget, nil
}

// getClusterUID gets the cluster UID from kube-system namespace
func (f *ClusterClientFactory) getClusterUID(ctx context.Context, kubeClient kubernetes.Interface) (string, error) {
	ns, err := kubeClient.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(ns.UID), nil
}

// getClusterVersion gets the Kubernetes version
func (f *ClusterClientFactory) getClusterVersion(ctx context.Context, kubeClient kubernetes.Interface) (string, error) {
	versionInfo, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return versionInfo.GitVersion, nil
}

// DetectPlatform detects the cluster platform (eks, gke, aks, etc.)
func DetectPlatform(ctx context.Context, kubeClient kubernetes.Interface) string {
	// Try to detect platform from nodes
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil || len(nodes.Items) == 0 {
		return "unknown"
	}

	node := nodes.Items[0]

	// Check provider ID
	providerID := node.Spec.ProviderID
	switch {
	case len(providerID) == 0:
		return "vanilla"
	case len(providerID) > 0 && providerID[:3] == "aws":
		return "eks"
	case len(providerID) > 0 && providerID[:3] == "gce":
		return "gke"
	case len(providerID) > 0 && providerID[:5] == "azure":
		return "aks"
	}

	// Check labels
	labels := node.Labels
	if _, ok := labels["eks.amazonaws.com/nodegroup"]; ok {
		return "eks"
	}
	if _, ok := labels["cloud.google.com/gke-nodepool"]; ok {
		return "gke"
	}
	if _, ok := labels["kubernetes.azure.com/cluster"]; ok {
		return "aks"
	}
	if _, ok := labels["node.openshift.io/os_id"]; ok {
		return "openshift"
	}

	return "vanilla"
}
