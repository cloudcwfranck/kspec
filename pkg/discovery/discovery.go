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

package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

// ClusterDiscovery discovers Kubernetes clusters from kubeconfig
type ClusterDiscovery struct {
	kubeconfigPath string
}

// NewClusterDiscovery creates a new ClusterDiscovery
func NewClusterDiscovery(kubeconfigPath string) *ClusterDiscovery {
	if kubeconfigPath == "" {
		// Use default kubeconfig location
		if home := os.Getenv("HOME"); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	return &ClusterDiscovery{
		kubeconfigPath: kubeconfigPath,
	}
}

// DiscoveredCluster represents a discovered cluster from kubeconfig
type DiscoveredCluster struct {
	Name          string
	APIServerURL  string
	ContextName   string
	ClusterName   string
	AuthInfoName  string
	HasCAData     bool
	HasClientCert bool
	HasToken      bool
	Namespace     string
}

// DiscoverClusters discovers all clusters from the kubeconfig file
func (d *ClusterDiscovery) DiscoverClusters() ([]DiscoveredCluster, error) {
	config, err := clientcmd.LoadFromFile(d.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	var discovered []DiscoveredCluster

	for contextName, context := range config.Contexts {
		cluster, ok := config.Clusters[context.Cluster]
		if !ok {
			continue
		}

		authInfo, hasAuth := config.AuthInfos[context.AuthInfo]

		dc := DiscoveredCluster{
			Name:         sanitizeName(contextName),
			APIServerURL: cluster.Server,
			ContextName:  contextName,
			ClusterName:  context.Cluster,
			AuthInfoName: context.AuthInfo,
			HasCAData:    len(cluster.CertificateAuthorityData) > 0 || cluster.CertificateAuthority != "",
			Namespace:    context.Namespace,
		}

		if hasAuth {
			dc.HasClientCert = len(authInfo.ClientCertificateData) > 0 || authInfo.ClientCertificate != ""
			dc.HasToken = authInfo.Token != ""
		}

		discovered = append(discovered, dc)
	}

	return discovered, nil
}

// GenerateClusterTarget generates a ClusterTarget resource for a discovered cluster
func (d *ClusterDiscovery) GenerateClusterTarget(cluster DiscoveredCluster, namespace string) (*kspecv1alpha1.ClusterTarget, error) {
	config, err := clientcmd.LoadFromFile(d.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	kubeCluster, ok := config.Clusters[cluster.ClusterName]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found in kubeconfig", cluster.ClusterName)
	}

	target := &kspecv1alpha1.ClusterTarget{}
	target.Name = cluster.Name
	target.Namespace = namespace
	target.Spec.APIServerURL = kubeCluster.Server
	target.Spec.InsecureSkipTLSVerify = kubeCluster.InsecureSkipTLSVerify

	// Set CA data if available
	if len(kubeCluster.CertificateAuthorityData) > 0 {
		target.Spec.CAData = kubeCluster.CertificateAuthorityData
	}

	// Determine auth mode
	// For auto-discovery, we'll use kubeconfig auth mode and expect
	// users to create a Secret with the kubeconfig data
	target.Spec.AuthMode = "kubeconfig"
	target.Spec.KubeconfigSecretRef = &kspecv1alpha1.SecretReference{
		Name:      fmt.Sprintf("%s-kubeconfig", cluster.Name),
		Namespace: namespace,
		Key:       "kubeconfig",
	}

	return target, nil
}

// ExtractKubeconfigContext extracts a specific context from kubeconfig as a standalone config
func (d *ClusterDiscovery) ExtractKubeconfigContext(contextName string) (*clientcmdapi.Config, error) {
	config, err := clientcmd.LoadFromFile(d.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	context, ok := config.Contexts[contextName]
	if !ok {
		return nil, fmt.Errorf("context %s not found in kubeconfig", contextName)
	}

	// Create a new config with only this context
	newConfig := clientcmdapi.NewConfig()

	// Copy cluster
	if cluster, ok := config.Clusters[context.Cluster]; ok {
		newConfig.Clusters[context.Cluster] = cluster
	} else {
		return nil, fmt.Errorf("cluster %s not found", context.Cluster)
	}

	// Copy auth info
	if authInfo, ok := config.AuthInfos[context.AuthInfo]; ok {
		newConfig.AuthInfos[context.AuthInfo] = authInfo
	} else {
		return nil, fmt.Errorf("auth info %s not found", context.AuthInfo)
	}

	// Copy context
	newConfig.Contexts[contextName] = context
	newConfig.CurrentContext = contextName

	return newConfig, nil
}

// sanitizeName converts a context name to a valid Kubernetes resource name
func sanitizeName(name string) string {
	// Replace invalid characters with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "@", "-at-")
	name = strings.ToLower(name)

	// Truncate to 63 characters (Kubernetes name limit)
	if len(name) > 63 {
		name = name[:63]
	}

	return name
}
