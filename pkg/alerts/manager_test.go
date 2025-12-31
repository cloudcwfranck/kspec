package alerts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

// mockNotifier is a test notifier
type mockNotifier struct {
	name        string
	enabled     bool
	eventFilter []string
	sendFunc    func(ctx context.Context, alert Alert) error
	sendCalls   []Alert
	mu          sync.Mutex // Protect sendCalls for concurrent access
}

func (m *mockNotifier) Send(ctx context.Context, alert Alert) error {
	m.mu.Lock()
	m.sendCalls = append(m.sendCalls, alert)
	m.mu.Unlock()

	if m.sendFunc != nil {
		return m.sendFunc(ctx, alert)
	}
	return nil
}

func (m *mockNotifier) Name() string {
	return m.name
}

func (m *mockNotifier) Enabled() bool {
	return m.enabled
}

func (m *mockNotifier) ShouldSend(alert Alert) bool {
	if len(m.eventFilter) == 0 {
		return true
	}
	for _, eventType := range m.eventFilter {
		if eventType == alert.EventType {
			return true
		}
	}
	return false
}

// getSendCallCount returns the number of send calls (thread-safe)
func (m *mockNotifier) getSendCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sendCalls)
}

func TestManager_AddNotifier(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier := &mockNotifier{
		name:    "test-notifier",
		enabled: true,
	}

	err := manager.AddNotifier(notifier)
	if err != nil {
		t.Fatalf("AddNotifier() failed: %v", err)
	}

	// Verify notifier was added
	if _, exists := manager.GetNotifier("test-notifier"); !exists {
		t.Error("Notifier was not added to manager")
	}

	// Test adding notifier with empty name
	badNotifier := &mockNotifier{name: ""}
	err = manager.AddNotifier(badNotifier)
	if err == nil {
		t.Error("Expected error when adding notifier with empty name")
	}
}

func TestManager_RemoveNotifier(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier := &mockNotifier{
		name:    "test-notifier",
		enabled: true,
	}

	manager.AddNotifier(notifier)
	manager.RemoveNotifier("test-notifier")

	if _, exists := manager.GetNotifier("test-notifier"); exists {
		t.Error("Notifier was not removed from manager")
	}
}

func TestManager_Send(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier1 := &mockNotifier{
		name:    "notifier-1",
		enabled: true,
	}
	notifier2 := &mockNotifier{
		name:    "notifier-2",
		enabled: true,
	}

	manager.AddNotifier(notifier1)
	manager.AddNotifier(notifier2)

	alert := Alert{
		Level:       AlertLevelCritical,
		Title:       "Test Alert",
		Description: "Test description",
		Source:      "test",
		EventType:   "TestEvent",
	}

	err := manager.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	// Verify both notifiers received the alert
	if notifier1.getSendCallCount() != 1 {
		t.Errorf("Expected 1 send call to notifier1, got %d", notifier1.getSendCallCount())
	}
	if notifier2.getSendCallCount() != 1 {
		t.Errorf("Expected 1 send call to notifier2, got %d", notifier2.getSendCallCount())
	}

	// Verify timestamp was set
	notifier1.mu.Lock()
	timestamp := notifier1.sendCalls[0].Timestamp
	notifier1.mu.Unlock()
	if timestamp.IsZero() {
		t.Error("Alert timestamp was not set")
	}
}

func TestManager_Send_WithFilter(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	// Notifier that only receives DriftDetected events
	notifier1 := &mockNotifier{
		name:        "drift-notifier",
		enabled:     true,
		eventFilter: []string{"DriftDetected"},
	}

	// Notifier that receives all events
	notifier2 := &mockNotifier{
		name:    "all-notifier",
		enabled: true,
	}

	manager.AddNotifier(notifier1)
	manager.AddNotifier(notifier2)

	// Send a DriftDetected alert
	driftAlert := Alert{
		Level:     AlertLevelCritical,
		Title:     "Drift detected",
		EventType: "DriftDetected",
	}

	manager.Send(context.Background(), driftAlert)

	if notifier1.getSendCallCount() != 1 {
		t.Errorf("Expected drift-notifier to receive drift alert, got %d calls", notifier1.getSendCallCount())
	}
	if notifier2.getSendCallCount() != 1 {
		t.Errorf("Expected all-notifier to receive drift alert, got %d calls", notifier2.getSendCallCount())
	}

	// Send a ComplianceFailure alert
	complianceAlert := Alert{
		Level:     AlertLevelWarning,
		Title:     "Compliance failure",
		EventType: "ComplianceFailure",
	}

	manager.Send(context.Background(), complianceAlert)

	// drift-notifier should not receive this (still at 1 call)
	if notifier1.getSendCallCount() != 1 {
		t.Errorf("Expected drift-notifier to filter out compliance alert, got %d calls", notifier1.getSendCallCount())
	}
	// all-notifier should receive it (now at 2 calls)
	if notifier2.getSendCallCount() != 2 {
		t.Errorf("Expected all-notifier to receive both alerts, got %d calls", notifier2.getSendCallCount())
	}
}

func TestManager_Send_DisabledNotifier(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	disabledNotifier := &mockNotifier{
		name:    "disabled-notifier",
		enabled: false,
	}

	manager.AddNotifier(disabledNotifier)

	alert := Alert{
		Level: AlertLevelInfo,
		Title: "Test",
	}

	err := manager.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() failed: %v", err)
	}

	// Disabled notifier should not receive alerts
	if disabledNotifier.getSendCallCount() != 0 {
		t.Errorf("Expected disabled notifier to not receive alerts, got %d calls", disabledNotifier.getSendCallCount())
	}
}

func TestManager_Send_WithError(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	failingNotifier := &mockNotifier{
		name:    "failing-notifier",
		enabled: true,
		sendFunc: func(ctx context.Context, alert Alert) error {
			return errors.New("send failed")
		},
	}

	successNotifier := &mockNotifier{
		name:    "success-notifier",
		enabled: true,
	}

	manager.AddNotifier(failingNotifier)
	manager.AddNotifier(successNotifier)

	alert := Alert{
		Level: AlertLevelCritical,
		Title: "Test",
	}

	err := manager.Send(context.Background(), alert)
	// Should return error but not fail completely
	if err == nil {
		t.Error("Expected error when a notifier fails")
	}

	// Success notifier should still have received the alert
	if successNotifier.getSendCallCount() != 1 {
		t.Errorf("Expected success notifier to still receive alert, got %d calls", successNotifier.getSendCallCount())
	}

	// Check stats
	stats := manager.GetStats()
	if stats["failing-notifier"].Failed != 1 {
		t.Errorf("Expected failing notifier to have 1 failure, got %d", stats["failing-notifier"].Failed)
	}
	if stats["success-notifier"].Sent != 1 {
		t.Errorf("Expected success notifier to have 1 sent, got %d", stats["success-notifier"].Sent)
	}
}

func TestManager_SendToNotifier(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier1 := &mockNotifier{
		name:    "notifier-1",
		enabled: true,
	}
	notifier2 := &mockNotifier{
		name:    "notifier-2",
		enabled: true,
	}

	manager.AddNotifier(notifier1)
	manager.AddNotifier(notifier2)

	alert := Alert{
		Level: AlertLevelInfo,
		Title: "Test",
	}

	// Send to specific notifier
	err := manager.SendToNotifier(context.Background(), "notifier-1", alert)
	if err != nil {
		t.Fatalf("SendToNotifier() failed: %v", err)
	}

	// Only notifier-1 should have received it
	if notifier1.getSendCallCount() != 1 {
		t.Errorf("Expected notifier-1 to receive alert, got %d calls", notifier1.getSendCallCount())
	}
	if notifier2.getSendCallCount() != 0 {
		t.Errorf("Expected notifier-2 to not receive alert, got %d calls", notifier2.getSendCallCount())
	}

	// Test sending to non-existent notifier
	err = manager.SendToNotifier(context.Background(), "non-existent", alert)
	if err == nil {
		t.Error("Expected error when sending to non-existent notifier")
	}
}

func TestManager_GetStats(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier := &mockNotifier{
		name:    "test-notifier",
		enabled: true,
	}

	manager.AddNotifier(notifier)

	// Send some alerts
	for i := 0; i < 5; i++ {
		alert := Alert{
			Level: AlertLevelInfo,
			Title: "Test",
		}
		manager.Send(context.Background(), alert)
	}

	stats := manager.GetStats()
	if stats["test-notifier"].Sent != 5 {
		t.Errorf("Expected 5 sent alerts, got %d", stats["test-notifier"].Sent)
	}
}

func TestManager_ListNotifiers(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier1 := &mockNotifier{name: "notifier-1", enabled: true}
	notifier2 := &mockNotifier{name: "notifier-2", enabled: true}

	manager.AddNotifier(notifier1)
	manager.AddNotifier(notifier2)

	names := manager.ListNotifiers()
	if len(names) != 2 {
		t.Errorf("Expected 2 notifiers, got %d", len(names))
	}

	// Check both names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["notifier-1"] || !nameMap["notifier-2"] {
		t.Error("Expected both notifier-1 and notifier-2 in list")
	}
}

func TestManager_Clear(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier := &mockNotifier{name: "test-notifier", enabled: true}
	manager.AddNotifier(notifier)

	// Send an alert to create stats
	alert := Alert{Level: AlertLevelInfo, Title: "Test"}
	manager.Send(context.Background(), alert)

	// Clear
	manager.Clear()

	// Verify everything is cleared
	if len(manager.ListNotifiers()) != 0 {
		t.Error("Expected no notifiers after Clear()")
	}

	stats := manager.GetStats()
	if len(stats) != 0 {
		t.Error("Expected no stats after Clear()")
	}
}

func TestManager_ConcurrentSend(t *testing.T) {
	logger := logr.Discard()
	manager := NewManager(logger)

	notifier := &mockNotifier{
		name:    "test-notifier",
		enabled: true,
		sendFunc: func(ctx context.Context, alert Alert) error {
			time.Sleep(10 * time.Millisecond) // Simulate some work
			return nil
		},
	}

	manager.AddNotifier(notifier)

	// Send alerts concurrently
	const numAlerts = 10
	done := make(chan error, numAlerts)

	for i := 0; i < numAlerts; i++ {
		go func() {
			alert := Alert{
				Level: AlertLevelInfo,
				Title: "Concurrent test",
			}
			done <- manager.Send(context.Background(), alert)
		}()
	}

	// Wait for all to complete
	for i := 0; i < numAlerts; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent send failed: %v", err)
		}
	}

	// Verify all were sent
	stats := manager.GetStats()
	if stats["test-notifier"].Sent != numAlerts {
		t.Errorf("Expected %d sent alerts, got %d", numAlerts, stats["test-notifier"].Sent)
	}
}
