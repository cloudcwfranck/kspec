package checks

import (
	"context"
	"testing"

	"github.com/cloudcwfranck/kspec/pkg/scanner"
	"github.com/cloudcwfranck/kspec/pkg/spec"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAdmissionCheck_Pass(t *testing.T) {
	// Create required webhook
	webhook := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyverno-resource-validating-webhook-cfg",
		},
	}

	client := fake.NewSimpleClientset(webhook)
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "ValidatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    1,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestAdmissionCheck_FailMissingWebhook(t *testing.T) {
	// No webhooks installed
	client := fake.NewSimpleClientset()
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "ValidatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    1,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Contains(t, result.Evidence, "violations")
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "found 0, need 1")
}

func TestAdmissionCheck_FailInsufficientWebhookCount(t *testing.T) {
	// Only one webhook, but need 2
	webhook := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyverno-webhook-1",
		},
	}

	client := fake.NewSimpleClientset(webhook)
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "ValidatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    2,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "found 1, need 2")
}

func TestAdmissionCheck_MutatingWebhook(t *testing.T) {
	// MutatingWebhookConfiguration
	webhook := &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyverno-policy-mutating-webhook",
		},
	}

	client := fake.NewSimpleClientset(webhook)
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "MutatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    1,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestAdmissionCheck_MultipleWebhooks(t *testing.T) {
	// Multiple webhooks matching pattern
	webhook1 := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyverno-resource-webhook",
		},
	}
	webhook2 := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyverno-policy-webhook",
		},
	}
	webhook3 := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "other-webhook",
		},
	}

	client := fake.NewSimpleClientset(webhook1, webhook2, webhook3)
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "ValidatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    2,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusPass, result.Status)
}

func TestAdmissionCheck_WrongWebhookPattern(t *testing.T) {
	// Webhook exists but doesn't match pattern
	webhook := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "other-webhook",
		},
	}

	client := fake.NewSimpleClientset(webhook)
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "ValidatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    1,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
}

func TestAdmissionCheck_Skip(t *testing.T) {
	client := fake.NewSimpleClientset()
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			// No admission spec
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusSkip, result.Status)
}

func TestAdmissionCheck_Name(t *testing.T) {
	check := &AdmissionCheck{}
	assert.Equal(t, "admission.controllers", check.Name())
}

func TestAdmissionCheck_PolicyRequirements(t *testing.T) {
	// Test with policy requirements (will fail gracefully since we can't mock dynamic client easily)
	webhook := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyverno-webhook",
		},
	}

	client := fake.NewSimpleClientset(webhook)
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "ValidatingWebhookConfiguration",
						NamePattern: "kyverno-.*",
						MinCount:    1,
					},
				},
				Policies: &spec.PolicySpec{
					MinCount: 5,
					RequiredPolicies: []spec.RequiredPolicy{
						{Name: "disallow-privileged-containers"},
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	// Will fail because dynamic client can't connect (expected in unit tests)
	assert.Equal(t, scanner.StatusFail, result.Status)
	assert.Contains(t, result.Evidence, "violations")
}

func TestAdmissionCheck_UnknownWebhookType(t *testing.T) {
	client := fake.NewSimpleClientset()
	check := &AdmissionCheck{}

	clusterSpec := &spec.ClusterSpecification{
		Spec: spec.SpecFields{
			Admission: &spec.AdmissionSpec{
				Required: []spec.AdmissionRequirement{
					{
						Type:        "UnknownWebhookType",
						NamePattern: "test-.*",
						MinCount:    1,
					},
				},
			},
		},
	}

	result, err := check.Run(context.Background(), client, clusterSpec)
	assert.NoError(t, err)
	assert.Equal(t, scanner.StatusFail, result.Status)
	violations := result.Evidence["violations"].([]string)
	assert.Contains(t, violations[0], "Unknown webhook type")
}
