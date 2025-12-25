package drift

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryStorage_Store(t *testing.T) {
	storage := NewMemoryStorage()

	event := DriftEvent{
		Timestamp: time.Now(),
		Type:      DriftTypePolicy,
		Severity:  SeverityHigh,
		Resource: DriftResource{
			Kind: "ClusterPolicy",
			Name: "test-policy",
		},
		DriftKind: "missing",
		Message:   "Policy is missing",
	}

	err := storage.Store(event)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Retrieve history
	history, err := storage.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(history.Events))
	}

	if history.Stats.TotalEvents != 1 {
		t.Errorf("Expected total events 1, got %d", history.Stats.TotalEvents)
	}
}

func TestMemoryStorage_GetHistorySince(t *testing.T) {
	storage := NewMemoryStorage()

	// Store events at different times
	now := time.Now()
	oldEvent := DriftEvent{
		Timestamp: now.Add(-2 * time.Hour),
		Type:      DriftTypePolicy,
		Severity:  SeverityLow,
	}
	recentEvent := DriftEvent{
		Timestamp: now.Add(-30 * time.Minute),
		Type:      DriftTypeCompliance,
		Severity:  SeverityHigh,
	}

	storage.Store(oldEvent)
	storage.Store(recentEvent)

	// Get history since 1 hour ago
	history, err := storage.GetHistory(now.Add(-1 * time.Hour))
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	// Should only return recent event
	if len(history.Events) != 1 {
		t.Errorf("Expected 1 recent event, got %d", len(history.Events))
	}

	if history.Events[0].Type != DriftTypeCompliance {
		t.Errorf("Expected compliance event, got %s", history.Events[0].Type)
	}
}

func TestMemoryStorage_Stats(t *testing.T) {
	storage := NewMemoryStorage()

	// Store multiple events with different types and severities
	events := []DriftEvent{
		{
			Timestamp: time.Now(),
			Type:      DriftTypePolicy,
			Severity:  SeverityHigh,
			Remediation: &RemediationResult{
				Status: DriftStatusRemediated,
			},
		},
		{
			Timestamp: time.Now(),
			Type:      DriftTypePolicy,
			Severity:  SeverityMedium,
			Remediation: &RemediationResult{
				Status: DriftStatusFailed,
			},
		},
		{
			Timestamp: time.Now(),
			Type:      DriftTypeCompliance,
			Severity:  SeverityCritical,
			Remediation: &RemediationResult{
				Status: DriftStatusManualRequired,
			},
		},
	}

	for _, event := range events {
		storage.Store(event)
	}

	history, err := storage.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	// Verify stats
	if history.Stats.TotalEvents != 3 {
		t.Errorf("Expected total events 3, got %d", history.Stats.TotalEvents)
	}

	if history.Stats.EventsByType[DriftTypePolicy] != 2 {
		t.Errorf("Expected 2 policy events, got %d", history.Stats.EventsByType[DriftTypePolicy])
	}

	if history.Stats.EventsByType[DriftTypeCompliance] != 1 {
		t.Errorf("Expected 1 compliance event, got %d", history.Stats.EventsByType[DriftTypeCompliance])
	}

	if history.Stats.EventsBySeverity[SeverityHigh] != 1 {
		t.Errorf("Expected 1 high severity event, got %d", history.Stats.EventsBySeverity[SeverityHigh])
	}

	if history.Stats.EventsBySeverity[SeverityCritical] != 1 {
		t.Errorf("Expected 1 critical severity event, got %d", history.Stats.EventsBySeverity[SeverityCritical])
	}

	// Remediation success rate: 1 remediated out of 3 total = 33.3%
	expectedRate := 1.0 / 3.0
	if history.Stats.RemediationSuccessRate < expectedRate-0.01 || history.Stats.RemediationSuccessRate > expectedRate+0.01 {
		t.Errorf("Expected remediation success rate ~%.2f, got %.2f", expectedRate, history.Stats.RemediationSuccessRate)
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	storage := NewMemoryStorage()

	// Store an event
	storage.Store(DriftEvent{
		Timestamp: time.Now(),
		Type:      DriftTypePolicy,
	})

	// Clear storage
	err := storage.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify storage is empty
	history, err := storage.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history.Events) != 0 {
		t.Errorf("Expected 0 events after clear, got %d", len(history.Events))
	}
}

func TestFileStorage_Store(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "drift-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storagePath := filepath.Join(tmpDir, "drift-events.json")
	storage := NewFileStorage(storagePath)

	event := DriftEvent{
		Timestamp: time.Now(),
		Type:      DriftTypePolicy,
		Severity:  SeverityHigh,
		Resource: DriftResource{
			Kind: "ClusterPolicy",
			Name: "test-policy",
		},
		DriftKind: "missing",
		Message:   "Policy is missing",
	}

	err = storage.Store(event)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		t.Error("Storage file was not created")
	}

	// Retrieve history
	history, err := storage.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(history.Events))
	}

	if history.Events[0].Resource.Name != "test-policy" {
		t.Errorf("Expected policy name 'test-policy', got '%s'", history.Events[0].Resource.Name)
	}
}

func TestFileStorage_Persistence(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "drift-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storagePath := filepath.Join(tmpDir, "drift-events.json")

	// Create storage and store event
	storage1 := NewFileStorage(storagePath)

	event := DriftEvent{
		Timestamp: time.Now(),
		Type:      DriftTypePolicy,
		Severity:  SeverityHigh,
		Message:   "Test event",
	}

	err = storage1.Store(event)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Create new storage instance pointing to same file
	storage2 := NewFileStorage(storagePath)

	// Retrieve history from second instance
	history, err := storage2.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	// Should still have the event from first instance
	if len(history.Events) != 1 {
		t.Errorf("Expected 1 persisted event, got %d", len(history.Events))
	}
}

func TestFileStorage_Clear(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "drift-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storagePath := filepath.Join(tmpDir, "drift-events.json")
	storage := NewFileStorage(storagePath)

	// Store an event
	storage.Store(DriftEvent{
		Timestamp: time.Now(),
		Type:      DriftTypePolicy,
	})

	// Clear storage
	err = storage.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify storage is empty
	history, err := storage.GetHistory(time.Time{})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history.Events) != 0 {
		t.Errorf("Expected 0 events after clear, got %d", len(history.Events))
	}

	// Verify file still exists (but is empty)
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		t.Error("Storage file should exist after clear")
	}
}

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name        string
		config      *StorageConfig
		expectError bool
	}{
		{
			name: "memory storage",
			config: &StorageConfig{
				Type: "memory",
			},
			expectError: false,
		},
		{
			name: "file storage with path",
			config: &StorageConfig{
				Type: "file",
				Path: "/tmp/test-drift.json",
			},
			expectError: false,
		},
		{
			name: "file storage without path",
			config: &StorageConfig{
				Type: "file",
			},
			expectError: true,
		},
		{
			name: "unsupported storage type",
			config: &StorageConfig{
				Type: "sqlite",
			},
			expectError: true,
		},
		{
			name:        "nil config defaults to memory",
			config:      nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewStorage(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if storage == nil {
					t.Error("Expected non-nil storage")
				}
			}
		})
	}
}
