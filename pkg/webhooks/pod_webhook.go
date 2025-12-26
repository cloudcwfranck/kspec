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

package webhooks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/spec"
)

// +kubebuilder:webhook:path=/validate-v1-pod,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod.kspec.io,admissionReviewVersions=v1

// PodValidator validates Pods against active ClusterSpecifications
type PodValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

var podlog = logf.Log.WithName("pod-webhook")

// SetupWebhookWithManager registers the webhook with the manager
func (v *PodValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithValidator(v).
		Complete()
}

// ValidateCreate validates Pod creation
func (v *PodValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod but got %T", obj)
	}

	podlog.Info("Validating Pod creation",
		"name", pod.Name,
		"namespace", pod.Namespace)

	return v.validatePod(ctx, pod)
}

// ValidateUpdate validates Pod updates
func (v *PodValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newPod, ok := newObj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod but got %T", newObj)
	}

	podlog.Info("Validating Pod update",
		"name", newPod.Name,
		"namespace", newPod.Namespace)

	return v.validatePod(ctx, newPod)
}

// ValidateDelete validates Pod deletion (allow all)
func (v *PodValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// Allow all Pod deletions
	return nil, nil
}

// validatePod performs actual Pod validation against ClusterSpecifications
func (v *PodValidator) validatePod(ctx context.Context, pod *corev1.Pod) (admission.Warnings, error) {
	// Get active ClusterSpecification for local cluster
	clusterSpec, err := v.getActiveClusterSpec(ctx)
	if err != nil {
		// If no ClusterSpec exists, allow the Pod (fail-open for missing config)
		podlog.Info("No active ClusterSpecification found, allowing Pod",
			"error", err)
		return nil, nil
	}

	// Check if namespace is exempted
	if v.isExemptedNamespace(pod.Namespace, clusterSpec) {
		podlog.Info("Pod in exempted namespace, allowing",
			"namespace", pod.Namespace)
		return nil, nil
	}

	// Validate workload security requirements
	if clusterSpec.Spec.Workloads != nil {
		if err := v.validateWorkloadSecurity(pod, clusterSpec); err != nil {
			podlog.Info("Pod violates workload security requirements",
				"pod", pod.Name,
				"namespace", pod.Namespace,
				"error", err)
			return []string{"Pod violates cluster security requirements"}, err
		}
	}

	// Validate image requirements
	if clusterSpec.Spec.Workloads != nil && clusterSpec.Spec.Workloads.Images != nil {
		if err := v.validateImageRequirements(pod, clusterSpec); err != nil {
			podlog.Info("Pod violates image requirements",
				"pod", pod.Name,
				"namespace", pod.Namespace,
				"error", err)
			return []string{"Pod violates image requirements"}, err
		}
	}

	podlog.Info("Pod validation passed",
		"pod", pod.Name,
		"namespace", pod.Namespace)

	return nil, nil
}

// getActiveClusterSpec retrieves the active ClusterSpecification for local cluster
func (v *PodValidator) getActiveClusterSpec(ctx context.Context) (*kspecv1alpha1.ClusterSpecification, error) {
	var clusterSpecs kspecv1alpha1.ClusterSpecificationList
	if err := v.Client.List(ctx, &clusterSpecs); err != nil {
		return nil, err
	}

	// Find first ClusterSpec without clusterRef (local cluster)
	// or with Phase=Active
	for _, cs := range clusterSpecs.Items {
		if cs.Spec.ClusterRef == nil && cs.Status.Phase == "Active" {
			return &cs, nil
		}
	}

	// Fallback: return first ClusterSpec without clusterRef
	for _, cs := range clusterSpecs.Items {
		if cs.Spec.ClusterRef == nil {
			return &cs, nil
		}
	}

	return nil, fmt.Errorf("no active ClusterSpecification found for local cluster")
}

// isExemptedNamespace checks if namespace is exempted from validation
func (v *PodValidator) isExemptedNamespace(namespace string, clusterSpec *kspecv1alpha1.ClusterSpecification) bool {
	// Always exempt system namespaces
	systemNamespaces := []string{"kube-system", "kube-public", "kube-node-lease", "kspec-system"}
	for _, ns := range systemNamespaces {
		if namespace == ns {
			return true
		}
	}

	// Check PodSecurity exemptions
	if clusterSpec.Spec.PodSecurity != nil && len(clusterSpec.Spec.PodSecurity.Exemptions) > 0 {
		for _, exemption := range clusterSpec.Spec.PodSecurity.Exemptions {
			if namespace == exemption.Namespace {
				return true
			}
		}
	}

	return false
}

// validateWorkloadSecurity validates Pod security context requirements
func (v *PodValidator) validateWorkloadSecurity(pod *corev1.Pod, clusterSpec *kspecv1alpha1.ClusterSpecification) error {
	workloads := clusterSpec.Spec.Workloads

	// Check required container fields
	if workloads.Containers != nil {
		for _, req := range workloads.Containers.Required {
			if err := v.validateContainerRequirement(pod, req); err != nil {
				return err
			}
		}
	}

	// Check forbidden container fields
	if workloads.Containers != nil {
		for _, forbidden := range workloads.Containers.Forbidden {
			if err := v.validateContainerForbidden(pod, forbidden); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateContainerRequirement checks if required field is set
func (v *PodValidator) validateContainerRequirement(pod *corev1.Pod, req spec.FieldRequirement) error {
	for i, container := range pod.Spec.Containers {
		switch req.Key {
		case "securityContext.runAsNonRoot":
			if container.SecurityContext == nil || container.SecurityContext.RunAsNonRoot == nil ||
				!*container.SecurityContext.RunAsNonRoot {
				return fmt.Errorf("container %d (%s) must have securityContext.runAsNonRoot=true",
					i, container.Name)
			}

		case "securityContext.allowPrivilegeEscalation":
			if container.SecurityContext == nil || container.SecurityContext.AllowPrivilegeEscalation == nil {
				return fmt.Errorf("container %d (%s) must set securityContext.allowPrivilegeEscalation",
					i, container.Name)
			}
			// If a value is specified, check if it matches "false"
			if req.Value != "" && req.Value == "false" {
				actualValue := container.SecurityContext.AllowPrivilegeEscalation != nil &&
					!*container.SecurityContext.AllowPrivilegeEscalation

				if !actualValue {
					return fmt.Errorf("container %d (%s) must have securityContext.allowPrivilegeEscalation=false",
						i, container.Name)
				}
			}

		case "securityContext.readOnlyRootFilesystem":
			if container.SecurityContext == nil || container.SecurityContext.ReadOnlyRootFilesystem == nil ||
				!*container.SecurityContext.ReadOnlyRootFilesystem {
				return fmt.Errorf("container %d (%s) must have securityContext.readOnlyRootFilesystem=true",
					i, container.Name)
			}
		}
	}

	return nil
}

// validateContainerForbidden checks if forbidden field is not set
func (v *PodValidator) validateContainerForbidden(pod *corev1.Pod, forbidden spec.FieldRequirement) error {
	for i, container := range pod.Spec.Containers {
		switch forbidden.Key {
		case "securityContext.privileged":
			if container.SecurityContext != nil && container.SecurityContext.Privileged != nil &&
				*container.SecurityContext.Privileged {
				return fmt.Errorf("container %d (%s) must not be privileged", i, container.Name)
			}
		}
	}

	// Check pod-level forbidden fields
	switch forbidden.Key {
	case "hostNetwork":
		if pod.Spec.HostNetwork {
			return fmt.Errorf("pod must not use hostNetwork")
		}
	case "hostPID":
		if pod.Spec.HostPID {
			return fmt.Errorf("pod must not use hostPID")
		}
	case "hostIPC":
		if pod.Spec.HostIPC {
			return fmt.Errorf("pod must not use hostIPC")
		}
	}

	return nil
}

// validateImageRequirements checks image registry and digest requirements
func (v *PodValidator) validateImageRequirements(pod *corev1.Pod, clusterSpec *kspecv1alpha1.ClusterSpecification) error {
	images := clusterSpec.Spec.Workloads.Images

	for i, container := range pod.Spec.Containers {
		image := container.Image

		// Check if image uses digest (required)
		if images.RequireDigests {
			if !containsDigest(image) {
				return fmt.Errorf("container %d (%s) image must use digest (sha256:...): %s",
					i, container.Name, image)
			}
		}

		// Check allowed registries
		if len(images.AllowedRegistries) > 0 {
			if !matchesRegistryPattern(image, images.AllowedRegistries) {
				return fmt.Errorf("container %d (%s) image not from allowed registries: %s",
					i, container.Name, image)
			}
		}

		// Check blocked registries
		if len(images.BlockedRegistries) > 0 {
			if matchesRegistryPattern(image, images.BlockedRegistries) {
				return fmt.Errorf("container %d (%s) image from blocked registry: %s",
					i, container.Name, image)
			}
		}
	}

	return nil
}

// Helper functions

func containsDigest(image string) bool {
	return len(image) > 0 && (image[len(image)-1:] != ":" && len(image) > 7 &&
		(image[len(image)-65:len(image)-64] == "@" || image[len(image)-72:len(image)-71] == "@"))
}

func matchesRegistryPattern(image string, patterns []string) bool {
	// Simple registry matching - can be enhanced with glob patterns
	for _, pattern := range patterns {
		if len(image) >= len(pattern) && image[:len(pattern)] == pattern {
			return true
		}
		// Handle wildcard patterns like "*.gcr.io"
		if pattern[0] == '*' {
			suffix := pattern[1:]
			if len(image) >= len(suffix) && image[len(image)-len(suffix):] == suffix {
				return true
			}
		}
	}
	return false
}
