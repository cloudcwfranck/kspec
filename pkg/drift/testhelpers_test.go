package drift

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// createTestClients creates properly configured fake clients for testing.
// This sets up the dynamic client with Kyverno GVR support.
func createTestClients(initialObjects ...runtime.Object) (*fake.Clientset, *dynamicfake.FakeDynamicClient) {
	client := fake.NewSimpleClientset()
	scheme := runtime.NewScheme()

	// Register Kyverno GroupVersionKind for list operations
	gvrToListKind := map[schema.GroupVersionResource]string{
		{
			Group:    "kyverno.io",
			Version:  "v1",
			Resource: "clusterpolicies",
		}: "ClusterPolicyList",
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, initialObjects...)

	return client, dynamicClient
}
