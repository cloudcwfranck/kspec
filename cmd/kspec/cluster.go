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

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/cloudcwfranck/kspec/pkg/discovery"
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage multi-cluster configurations",
	Long:  `Discover, register, and manage Kubernetes clusters for multi-cluster compliance scanning.`,
}

var clusterDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover clusters from kubeconfig",
	Long:  `Discover Kubernetes clusters from your kubeconfig file and show which can be registered.`,
	RunE:  runClusterDiscover,
}

var clusterAddCmd = &cobra.Command{
	Use:   "add <context-name>",
	Short: "Add a cluster from kubeconfig",
	Long:  `Generate ClusterTarget and Secret manifests for a cluster from your kubeconfig.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runClusterAdd,
}

var (
	kubeconfigPath  string
	outputFormat    string
	targetNamespace string
)

func init() {
	clusterCmd.AddCommand(clusterDiscoverCmd)
	clusterCmd.AddCommand(clusterAddCmd)

	// Flags for discover command
	clusterDiscoverCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (default: $HOME/.kube/config)")

	// Flags for add command
	clusterAddCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (default: $HOME/.kube/config)")
	clusterAddCmd.Flags().StringVarP(&outputFormat, "output", "o", "yaml", "Output format: yaml or json")
	clusterAddCmd.Flags().StringVarP(&targetNamespace, "namespace", "n", "kspec-system", "Namespace for ClusterTarget and Secret")
}

func runClusterDiscover(cmd *cobra.Command, args []string) error {
	disc := discovery.NewClusterDiscovery(kubeconfigPath)

	clusters, err := disc.DiscoverClusters()
	if err != nil {
		return fmt.Errorf("failed to discover clusters: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Println("No clusters found in kubeconfig")
		return nil
	}

	fmt.Printf("Discovered %d cluster(s) from kubeconfig:\n\n", len(clusters))

	// Use tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tCONTEXT\tAPI SERVER\tAUTH")

	for _, cluster := range clusters {
		authType := "unknown"
		if cluster.HasToken {
			authType = "token"
		} else if cluster.HasClientCert {
			authType = "client-cert"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			cluster.Name,
			cluster.ContextName,
			cluster.APIServerURL,
			authType,
		)
	}

	w.Flush()

	fmt.Printf("\nTo add a cluster, run:\n")
	fmt.Printf("  kspec cluster add <context-name> > cluster-manifest.yaml\n")
	fmt.Printf("  kubectl apply -f cluster-manifest.yaml\n")

	return nil
}

func runClusterAdd(cmd *cobra.Command, args []string) error {
	contextName := args[0]

	disc := discovery.NewClusterDiscovery(kubeconfigPath)

	// Discover all clusters to find the requested one
	clusters, err := disc.DiscoverClusters()
	if err != nil {
		return fmt.Errorf("failed to discover clusters: %w", err)
	}

	var targetCluster *discovery.DiscoveredCluster
	for i, cluster := range clusters {
		if cluster.ContextName == contextName || cluster.Name == contextName {
			targetCluster = &clusters[i]
			break
		}
	}

	if targetCluster == nil {
		return fmt.Errorf("cluster context %s not found in kubeconfig", contextName)
	}

	// Generate ClusterTarget
	clusterTarget, err := disc.GenerateClusterTarget(*targetCluster, targetNamespace)
	if err != nil {
		return fmt.Errorf("failed to generate ClusterTarget: %w", err)
	}

	clusterTarget.APIVersion = "kspec.io/v1alpha1"
	clusterTarget.Kind = "ClusterTarget"

	// Extract kubeconfig for this context
	kubeconfigData, err := disc.ExtractKubeconfigContext(contextName)
	if err != nil {
		return fmt.Errorf("failed to extract kubeconfig: %w", err)
	}

	// Serialize kubeconfig
	kubeconfigBytes, err := clientcmd.Write(*kubeconfigData)
	if err != nil {
		return fmt.Errorf("failed to serialize kubeconfig: %w", err)
	}

	// Create Secret manifest
	secret := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      clusterTarget.Spec.KubeconfigSecretRef.Name,
			"namespace": targetNamespace,
		},
		"type": "Opaque",
		"stringData": map[string]string{
			"kubeconfig": string(kubeconfigBytes),
		},
	}

	// Output manifests
	if outputFormat == "json" {
		// JSON output (not fully implemented for brevity)
		fmt.Fprintf(os.Stderr, "JSON output not yet implemented, using YAML\n")
	}

	// YAML output
	secretYAML, err := yaml.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal Secret: %w", err)
	}

	clusterTargetYAML, err := yaml.Marshal(clusterTarget)
	if err != nil {
		return fmt.Errorf("failed to marshal ClusterTarget: %w", err)
	}

	// Print Secret first
	fmt.Println("---")
	fmt.Printf("# Secret containing kubeconfig for cluster: %s\n", targetCluster.Name)
	fmt.Print(string(secretYAML))

	fmt.Println("---")
	fmt.Printf("# ClusterTarget for cluster: %s\n", targetCluster.Name)
	fmt.Printf("# API Server: %s\n", targetCluster.APIServerURL)
	fmt.Printf("# Context: %s\n", targetCluster.ContextName)
	fmt.Print(string(clusterTargetYAML))

	fmt.Fprintf(os.Stderr, "\nGenerated manifests for cluster: %s\n", targetCluster.Name)
	fmt.Fprintf(os.Stderr, "Apply with: kubectl apply -f <filename>\n")

	return nil
}
