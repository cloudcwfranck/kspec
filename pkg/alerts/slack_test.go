package alerts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSlackNotifier_Send(t *testing.T) {
	var receivedPayload map[string]interface{}
	receivedHeaders := make(map[string]string)

	// Mock Slack webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request details
		receivedHeaders["Content-Type"] = r.Header.Get("Content-Type")

		// Decode payload
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, "#kspec-alerts", "kspec-bot", ":shield:")

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

	// Verify Content-Type header
	if receivedHeaders["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", receivedHeaders["Content-Type"])
	}

	// Verify basic payload structure
	if receivedPayload["username"] != "kspec-bot" {
		t.Errorf("Expected username 'kspec-bot', got '%v'", receivedPayload["username"])
	}

	if receivedPayload["icon_emoji"] != ":shield:" {
		t.Errorf("Expected icon_emoji ':shield:', got '%v'", receivedPayload["icon_emoji"])
	}

	if receivedPayload["channel"] != "#kspec-alerts" {
		t.Errorf("Expected channel '#kspec-alerts', got '%v'", receivedPayload["channel"])
	}

	// Verify attachments exist
	attachments, ok := receivedPayload["attachments"].([]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("Expected attachments array in payload")
	}

	attachment := attachments[0].(map[string]interface{})

	// Verify attachment fields
	if attachment["color"] != "danger" {
		t.Errorf("Expected color 'danger' for critical alert, got '%v'", attachment["color"])
	}

	if attachment["title"] != "Configuration drift detected" {
		t.Errorf("Expected title to match alert title, got '%v'", attachment["title"])
	}

	if attachment["text"] != "Cluster configuration has drifted from specification" {
		t.Errorf("Expected text to match alert description, got '%v'", attachment["text"])
	}
}

func TestSlackNotifier_PayloadFormat(t *testing.T) {
	notifier := NewSlackNotifier("https://hooks.slack.com/test", "#kspec-alerts", "kspec-bot", ":shield:")

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

	payload := notifier.buildPayload(alert)

	// Convert to JSON to verify structure
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Load golden file
	goldenPath := filepath.Join("testdata", "slack_payload.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		// Update golden file
		os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err := os.WriteFile(goldenPath, jsonData, 0644); err != nil {
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
	if err := json.Unmarshal(jsonData, &actual); err != nil {
		t.Fatalf("Failed to unmarshal actual payload: %v", err)
	}

	// Compare structures (deep equal)
	if !equalJSON(expected, actual) {
		t.Errorf("Payload does not match golden file.\nExpected:\n%s\nActual:\n%s", expectedData, jsonData)
	}
}

func TestSlackNotifier_AlertColors(t *testing.T) {
	notifier := NewSlackNotifier("https://hooks.slack.com/test", "#test", "bot", ":shield:")

	tests := []struct {
		level         AlertLevel
		expectedColor string
	}{
		{AlertLevelCritical, "danger"},
		{AlertLevelWarning, "warning"},
		{AlertLevelInfo, "good"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			color := notifier.alertColor(tt.level)
			if color != tt.expectedColor {
				t.Errorf("Expected color %s for level %s, got %s", tt.expectedColor, tt.level, color)
			}
		})
	}
}

func TestSlackNotifier_EventFilter(t *testing.T) {
	notifier := NewSlackNotifier("https://hooks.slack.com/test", "#test", "bot", ":shield:")
	notifier.EventFilter = []string{"DriftDetected", "ComplianceFailure"}

	tests := []struct {
		eventType  string
		shouldSend bool
	}{
		{"DriftDetected", true},
		{"ComplianceFailure", true},
		{"RemediationPerformed", false},
		{"CircuitBreakerTripped", false},
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

func TestSlackNotifier_Disabled(t *testing.T) {
	notifier := NewSlackNotifier("https://hooks.slack.com/test", "#test", "bot", ":shield:")
	notifier.Enabled_ = false

	if notifier.Enabled() {
		t.Error("Expected notifier to be disabled")
	}
}

// equalJSON compares two JSON objects for equality
func equalJSON(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
