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

package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/alerts"
)

func TestAlertConfigReconciler_Reconcile_SlackNotifier(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kspecv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create AlertConfig with Slack configuration
	enabled := true
	alertConfig := &kspecv1alpha1.AlertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: kspecv1alpha1.AlertConfigSpec{
			Enabled: &enabled,
			Slack: &kspecv1alpha1.SlackConfig{
				Enabled:    true,
				WebhookURL: "https://hooks.slack.com/test",
				Channel:    "#alerts",
				Username:   "test-bot",
				IconEmoji:  ":robot:",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(alertConfig).
		WithStatusSubresource(alertConfig).
		Build()

	alertManager := alerts.NewManager(logr.Discard())
	reconciler := NewAlertConfigReconciler(fakeClient, scheme, alertManager)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	// Reconcile
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Slack notifier was added
	notifiers := alertManager.ListNotifiers()
	if len(notifiers) != 1 {
		t.Errorf("Expected 1 notifier, got %d", len(notifiers))
	}

	if len(notifiers) > 0 && notifiers[0] != "slack" {
		t.Errorf("Expected 'slack' notifier, got '%s'", notifiers[0])
	}

	// Verify status was updated
	var updatedConfig kspecv1alpha1.AlertConfig
	if err := fakeClient.Get(context.Background(), req.NamespacedName, &updatedConfig); err != nil {
		t.Fatalf("Failed to get updated AlertConfig: %v", err)
	}

	// Check conditions
	if len(updatedConfig.Status.Conditions) == 0 {
		t.Error("Expected status conditions to be set")
	} else {
		cond := updatedConfig.Status.Conditions[0]
		if cond.Type != ConditionTypeConfigured {
			t.Errorf("Expected condition type '%s', got '%s'", ConditionTypeConfigured, cond.Type)
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("Expected condition status True, got %s", cond.Status)
		}
	}
}

func TestAlertConfigReconciler_Reconcile_WebhookNotifier(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kspecv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	enabled := true
	alertConfig := &kspecv1alpha1.AlertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: kspecv1alpha1.AlertConfigSpec{
			Enabled: &enabled,
			Webhooks: []kspecv1alpha1.WebhookConfig{
				{
					Name:   "test-webhook",
					URL:    "https://example.com/webhook",
					Method: "POST",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(alertConfig).
		WithStatusSubresource(alertConfig).
		Build()

	alertManager := alerts.NewManager(logr.Discard())
	reconciler := NewAlertConfigReconciler(fakeClient, scheme, alertManager)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	// Reconcile
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify webhook notifier was added
	notifiers := alertManager.ListNotifiers()
	if len(notifiers) != 1 {
		t.Errorf("Expected 1 notifier, got %d", len(notifiers))
	}

	if len(notifiers) > 0 && notifiers[0] != "test-webhook" {
		t.Errorf("Expected 'test-webhook' notifier, got '%s'", notifiers[0])
	}
}

func TestAlertConfigReconciler_Reconcile_WithSecretRef(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kspecv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create secret with webhook URL
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slack-webhook-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"url": []byte("https://hooks.slack.com/secret-webhook"),
		},
	}

	enabled := true
	alertConfig := &kspecv1alpha1.AlertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: kspecv1alpha1.AlertConfigSpec{
			Enabled: &enabled,
			Slack: &kspecv1alpha1.SlackConfig{
				Enabled: true,
				WebhookURLSecretRef: &kspecv1alpha1.SecretReference{
					Name: "slack-webhook-secret",
					Key:  "url",
				},
				Channel:   "#alerts",
				Username:  "test-bot",
				IconEmoji: ":robot:",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, alertConfig).
		WithStatusSubresource(alertConfig).
		Build()

	alertManager := alerts.NewManager(logr.Discard())
	reconciler := NewAlertConfigReconciler(fakeClient, scheme, alertManager)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	// Reconcile
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Slack notifier was added with secret URL
	notifiers := alertManager.ListNotifiers()
	if len(notifiers) != 1 {
		t.Errorf("Expected 1 notifier, got %d", len(notifiers))
	}

	// Verify the notifier is configured (it should exist)
	if _, exists := alertManager.GetNotifier("slack"); !exists {
		t.Error("Expected slack notifier to be configured with secret URL")
	}
}

func TestAlertConfigReconciler_Reconcile_Disabled(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kspecv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	disabled := false
	alertConfig := &kspecv1alpha1.AlertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: kspecv1alpha1.AlertConfigSpec{
			Enabled: &disabled,
			Slack: &kspecv1alpha1.SlackConfig{
				Enabled:    true,
				WebhookURL: "https://hooks.slack.com/test",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(alertConfig).
		WithStatusSubresource(alertConfig).
		Build()

	alertManager := alerts.NewManager(logr.Discard())
	reconciler := NewAlertConfigReconciler(fakeClient, scheme, alertManager)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	// Reconcile
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no notifiers were added
	notifiers := alertManager.ListNotifiers()
	if len(notifiers) != 0 {
		t.Errorf("Expected 0 notifiers when disabled, got %d", len(notifiers))
	}

	// Verify status shows disabled
	var updatedConfig kspecv1alpha1.AlertConfig
	if err := fakeClient.Get(context.Background(), req.NamespacedName, &updatedConfig); err != nil {
		t.Fatalf("Failed to get updated AlertConfig: %v", err)
	}

	if len(updatedConfig.Status.Conditions) == 0 {
		t.Error("Expected status conditions to be set")
	} else {
		cond := updatedConfig.Status.Conditions[0]
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("Expected condition status False when disabled, got %s", cond.Status)
		}
	}
}

func TestAlertConfigReconciler_Reconcile_MultipleWebhooks(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kspecv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	enabled := true
	alertConfig := &kspecv1alpha1.AlertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: kspecv1alpha1.AlertConfigSpec{
			Enabled: &enabled,
			Webhooks: []kspecv1alpha1.WebhookConfig{
				{
					Name: "webhook-1",
					URL:  "https://example.com/webhook1",
				},
				{
					Name: "webhook-2",
					URL:  "https://example.com/webhook2",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(alertConfig).
		WithStatusSubresource(alertConfig).
		Build()

	alertManager := alerts.NewManager(logr.Discard())
	reconciler := NewAlertConfigReconciler(fakeClient, scheme, alertManager)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	// Reconcile
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify both webhooks were added
	notifiers := alertManager.ListNotifiers()
	if len(notifiers) != 2 {
		t.Errorf("Expected 2 notifiers, got %d", len(notifiers))
	}

	// Verify both webhook names exist
	notifierMap := make(map[string]bool)
	for _, name := range notifiers {
		notifierMap[name] = true
	}

	if !notifierMap["webhook-1"] || !notifierMap["webhook-2"] {
		t.Error("Expected both webhook-1 and webhook-2 to be configured")
	}
}

func TestAlertConfigReconciler_Reconcile_Deletion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kspecv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Start with an empty client (simulating deletion)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	alertManager := alerts.NewManager(logr.Discard())

	// Add a notifier first
	alertManager.AddNotifier(alerts.NewSlackNotifier("https://hooks.slack.com/test", "#test", "bot", ":shield:"))

	// Verify notifier exists
	if len(alertManager.ListNotifiers()) != 1 {
		t.Fatal("Expected 1 notifier before reconcile")
	}

	reconciler := NewAlertConfigReconciler(fakeClient, scheme, alertManager)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	// Reconcile (config doesn't exist, simulating deletion)
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify all notifiers were cleared
	notifiers := alertManager.ListNotifiers()
	if len(notifiers) != 0 {
		t.Errorf("Expected 0 notifiers after deletion, got %d", len(notifiers))
	}
}
