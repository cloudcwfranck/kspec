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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

// GetSecretData retrieves data from a Secret referenced by SecretReference
func GetSecretData(
	ctx context.Context,
	k8sClient client.Client,
	secretRef *kspecv1alpha1.SecretReference,
	defaultNamespace string,
) ([]byte, error) {
	if secretRef == nil {
		return nil, fmt.Errorf("secret reference is nil")
	}

	// Determine namespace
	namespace := secretRef.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}

	// Fetch the secret
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}

	if err := k8sClient.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretRef.Name, err)
	}

	// Determine key
	key := secretRef.Key
	if key == "" {
		// Use sensible defaults based on common patterns
		key = "data"
	}

	// Extract data
	data, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s does not contain key %s", namespace, secretRef.Name, key)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("secret %s/%s key %s is empty", namespace, secretRef.Name, key)
	}

	return data, nil
}

// GetKubeconfigFromSecret retrieves kubeconfig data from a Secret
func GetKubeconfigFromSecret(
	ctx context.Context,
	k8sClient client.Client,
	secretRef *kspecv1alpha1.SecretReference,
	defaultNamespace string,
) ([]byte, error) {
	// Use "kubeconfig" as default key for kubeconfig secrets
	if secretRef.Key == "" {
		secretRef.Key = "kubeconfig"
	}

	return GetSecretData(ctx, k8sClient, secretRef, defaultNamespace)
}

// GetTokenFromSecret retrieves a bearer token from a Secret
func GetTokenFromSecret(
	ctx context.Context,
	k8sClient client.Client,
	secretRef *kspecv1alpha1.SecretReference,
	defaultNamespace string,
) (string, error) {
	// Use "token" as default key for token secrets
	if secretRef.Key == "" {
		secretRef.Key = "token"
	}

	data, err := GetSecretData(ctx, k8sClient, secretRef, defaultNamespace)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// RedactedSecretRef returns a safe string representation of a SecretReference
// This is safe to use in logs and status updates
func RedactedSecretRef(ref *kspecv1alpha1.SecretReference) string {
	if ref == nil {
		return "<nil>"
	}

	namespace := ref.Namespace
	if namespace == "" {
		namespace = "<default>"
	}

	key := ref.Key
	if key == "" {
		key = "<default>"
	}

	return fmt.Sprintf("%s/%s[%s]", namespace, ref.Name, key)
}
