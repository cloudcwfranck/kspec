package alerts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebhookNotifier_Send(t *testing.T) {
	var receivedPayload map[string]interface{}
	receivedHeaders := make(map[string]string)
	requestCount := 0

	// Mock webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Capture headers
		receivedHeaders["Content-Type"] = r.Header.Get("Content-Type")
		receivedHeaders["Authorization"] = r.Header.Get("Authorization")

		// Decode payload
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"Authorization": "Bearer test-token",
	}

	notifier := NewWebhookNotifier("test-webhook", server.URL, "POST", headers, "")

	alert := Alert{
		Level:       AlertLevelCritical,
		Title:       "Configuration drift detected",
		Description: "Cluster configuration has drifted from specification",
		Source:      "ClusterSpec/prod-cluster",
		Timestamp:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EventType:   "DriftDetected",
		Labels: map[string]string{
			"cluster": "prod-cluster",
		},
		Metadata: map[string]interface{}{
			"cluster": "prod-cluster",
			"count":   3,
		},
	}

	// Send alert
	err := notifier.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	// Verify headers
	if receivedHeaders["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", receivedHeaders["Content-Type"])
	}

	if receivedHeaders["Authorization"] != "Bearer test-token" {
		t.Errorf("Expected Authorization header to be set, got %s", receivedHeaders["Authorization"])
	}

	// Verify payload structure
	if receivedPayload["level"] != "critical" {
		t.Errorf("Expected level 'critical', got '%v'", receivedPayload["level"])
	}

	if receivedPayload["title"] != "Configuration drift detected" {
		t.Errorf("Expected title to match, got '%v'", receivedPayload["title"])
	}

	// Verify request count (no retries on success)
	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}
}

func TestWebhookNotifier_PayloadFormat(t *testing.T) {
	notifier := NewWebhookNotifier("test-webhook", "https://example.com/webhook", "POST", nil, "")

	alert := Alert{
		Level:       AlertLevelCritical,
		Title:       "Configuration drift detected",
		Description: "Cluster configuration has drifted from specification",
		Source:      "ClusterSpec/prod-cluster",
		Timestamp:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EventType:   "DriftDetected",
		Labels: map[string]string{
			"cluster": "prod-cluster",
		},
		Metadata: map[string]interface{}{
			"cluster": "prod-cluster",
			"count":   3,
		},
	}

	payload, err := notifier.renderPayload(alert)
	if err != nil {
		t.Fatalf("renderPayload() failed: %v", err)
	}

	// Load golden file
	goldenPath := filepath.Join("testdata", "webhook_payload.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		// Update golden file
		os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err := os.WriteFile(goldenPath, payload, 0644); err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
		t.Log("Updated golden file")
	}

	// Compare with golden file
	expectedData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	var expected, actual map[string]interface{}
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatalf("Failed to unmarshal golden file: %v", err)
	}
	if err := json.Unmarshal(payload, &actual); err != nil {
		t.Fatalf("Failed to unmarshal actual payload: %v", err)
	}

	// Compare structures
	if !equalJSONWebhook(expected, actual) {
		t.Errorf("Payload does not match golden file.\nExpected:\n%s\nActual:\n%s", expectedData, payload)
	}
}

func TestWebhookNotifier_CustomTemplate(t *testing.T) {
	template := `{"message": "{{.Title}}", "severity": "{{.Level}}", "source": "{{.Source}}"}`
	notifier := NewWebhookNotifier("test-webhook", "https://example.com/webhook", "POST", nil, template)

	alert := Alert{
		Level:  AlertLevelCritical,
		Title:  "Test Alert",
		Source: "TestSource",
	}

	payload, err := notifier.renderPayload(alert)
	if err != nil {
		t.Fatalf("renderPayload() failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if result["message"] != "Test Alert" {
		t.Errorf("Expected message 'Test Alert', got '%v'", result["message"])
	}

	if result["severity"] != "critical" {
		t.Errorf("Expected severity 'critical', got '%v'", result["severity"])
	}

	if result["source"] != "TestSource" {
		t.Errorf("Expected source 'TestSource', got '%v'", result["source"])
	}
}

func TestWebhookNotifier_Retry(t *testing.T) {
	requestCount := 0
	failCount := 2 // Fail first 2 requests, succeed on 3rd

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= failCount {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier("test-webhook", server.URL, "POST", nil, "")
	notifier.RetryAttempts = 3

	alert := Alert{
		Level: AlertLevelInfo,
		Title: "Test",
	}

	// Should succeed after retries
	err := notifier.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() failed after retries: %v", err)
	}

	// Should have made 3 requests (initial + 2 retries)
	if requestCount != 3 {
		t.Errorf("Expected 3 requests (initial + 2 retries), got %d", requestCount)
	}
}

func TestWebhookNotifier_RetryExhausted(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier("test-webhook", server.URL, "POST", nil, "")
	notifier.RetryAttempts = 2

	alert := Alert{
		Level: AlertLevelInfo,
		Title: "Test",
	}

	// Should fail after exhausting retries
	err := notifier.Send(context.Background(), alert)
	if err == nil {
		t.Fatal("Expected error after retry exhaustion")
	}

	if !strings.Contains(err.Error(), "failed after") {
		t.Errorf("Expected retry exhaustion error, got: %v", err)
	}

	// Should have made 3 requests (initial + 2 retries)
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

func TestWebhookNotifier_EventFilter(t *testing.T) {
	notifier := NewWebhookNotifier("test-webhook", "https://example.com/webhook", "POST", nil, "")
	notifier.EventFilter = []string{"DriftDetected"}

	tests := []struct {
		eventType  string
		shouldSend bool
	}{
		{"DriftDetected", true},
		{"ComplianceFailure", false},
		{"RemediationPerformed", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			alert := Alert{EventType: tt.eventType}
			result := notifier.ShouldSend(alert)
			if result != tt.shouldSend {
				t.Errorf("Expected ShouldSend=%v for event %s, got %v", tt.shouldSend, tt.eventType, result)
			}
		})
	}
}

func TestWebhookNotifier_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than timeout
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier("test-webhook", server.URL, "POST", nil, "")
	notifier.Timeout = 100 * time.Millisecond
	notifier.RetryAttempts = 0 // No retries for faster test

	alert := Alert{
		Level: AlertLevelInfo,
		Title: "Test",
	}

	// Should timeout
	err := notifier.Send(context.Background(), alert)
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// equalJSONWebhook compares two JSON objects for equality
func equalJSONWebhook(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
