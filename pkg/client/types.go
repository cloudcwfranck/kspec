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

// ClusterInfo contains metadata about a cluster
type ClusterInfo struct {
	// Name is the cluster name
	// For local clusters this is "local", for remote clusters it's the ClusterTarget name
	Name string

	// UID is the unique identifier of the cluster
	// This is the kube-system namespace UID
	UID string

	// IsLocal indicates if this is the local cluster
	IsLocal bool

	// APIServerURL is the cluster's API server endpoint
	APIServerURL string

	// Version is the Kubernetes version
	Version string

	// Platform describes the cluster platform (e.g., "eks", "gke", "aks", "vanilla")
	Platform string

	// AllowEnforcement indicates if policy enforcement and drift remediation is allowed
	AllowEnforcement bool
}
