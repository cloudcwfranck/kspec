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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterTargetSpec defines the desired state of ClusterTarget
type ClusterTargetSpec struct {
	// APIServerURL is the Kubernetes API server endpoint (e.g., https://k8s.example.com:6443)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://.*`
	APIServerURL string `json:"apiServerURL"`

	// AuthMode specifies the authentication method to use
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=kubeconfig;serviceAccount;token
	AuthMode string `json:"authMode"`

	// KubeconfigSecretRef references a Secret containing a kubeconfig file
	// Required when authMode is "kubeconfig"
	// +optional
	KubeconfigSecretRef *SecretReference `json:"kubeconfigSecretRef,omitempty"`

	// ServiceAccountSecretRef references a Secret containing ServiceAccount token
	// Required when authMode is "serviceAccount"
	// +optional
	ServiceAccountSecretRef *SecretReference `json:"serviceAccountSecretRef,omitempty"`

	// TokenSecretRef references a Secret containing a bearer token
	// Required when authMode is "token"
	// +optional
	TokenSecretRef *SecretReference `json:"tokenSecretRef,omitempty"`

	// CAData contains PEM-encoded certificate authority certificates
	// If specified, used to verify the cluster's API server certificate
	// +optional
	CAData []byte `json:"caData,omitempty"`

	// InsecureSkipTLSVerify skips the validity check for the server's certificate
	// This will make your connection insecure and should only be used for testing
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`

	// ProxyURL is the URL of the proxy to use for this cluster connection
	// +optional
	ProxyURL string `json:"proxyURL,omitempty"`

	// AllowEnforcement enables policy enforcement and drift remediation on this cluster
	// If false (default), the cluster will be scanned read-only
	// +optional
	// +kubebuilder:default=false
	AllowEnforcement bool `json:"allowEnforcement,omitempty"`

	// ScanInterval specifies how often to scan this cluster
	// If not specified, uses the default reconciliation interval
	// +optional
	ScanInterval *metav1.Duration `json:"scanInterval,omitempty"`
}

// SecretReference references a secret and optionally a specific key within it
type SecretReference struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the secret
	// If not specified, uses the same namespace as the ClusterTarget
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key is the key within the secret data
	// Defaults to "kubeconfig" for kubeconfig mode, "token" for token/serviceAccount modes
	// +optional
	Key string `json:"key,omitempty"`
}

// ClusterTargetStatus defines the observed state of ClusterTarget
type ClusterTargetStatus struct {
	// Reachable indicates whether the cluster is currently reachable
	Reachable bool `json:"reachable"`

	// LastChecked is the timestamp of the last health check
	// +optional
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`

	// Version is the Kubernetes version of the cluster
	// +optional
	Version string `json:"version,omitempty"`

	// UID is the unique identifier of the cluster
	// +optional
	UID string `json:"uid,omitempty"`

	// Platform describes the cluster platform (e.g., "eks", "gke", "aks", "openshift", "vanilla")
	// +optional
	Platform string `json:"platform,omitempty"`

	// NodeCount is the number of nodes in the cluster
	// +optional
	NodeCount int32 `json:"nodeCount,omitempty"`

	// Conditions represent the latest available observations of the ClusterTarget's state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration reflects the generation of the most recently observed ClusterTarget
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ClusterTarget defines a remote Kubernetes cluster that can be scanned by the operator
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ct
// +kubebuilder:printcolumn:name="API Server",type=string,JSONPath=`.spec.apiServerURL`
// +kubebuilder:printcolumn:name="Reachable",type=boolean,JSONPath=`.status.reachable`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Platform",type=string,JSONPath=`.status.platform`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ClusterTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTargetSpec   `json:"spec,omitempty"`
	Status ClusterTargetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterTargetList contains a list of ClusterTarget
type ClusterTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTarget{}, &ClusterTargetList{})
}
