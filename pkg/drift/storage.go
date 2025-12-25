package drift

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Storage stores drift history.
type Storage interface {
	// Store stores a drift event
	Store(event DriftEvent) error

	// GetHistory returns drift history
	GetHistory(since time.Time) (*DriftHistory, error)

	// Clear clears all history
	Clear() error
}

// MemoryStorage stores drift history in memory.
type MemoryStorage struct {
	mu     sync.RWMutex
	events []DriftEvent
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		events: []DriftEvent{},
	}
}

// Store stores a drift event.
func (s *MemoryStorage) Store(event DriftEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

// GetHistory returns drift history since a given time.
func (s *MemoryStorage) GetHistory(since time.Time) (*DriftHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := []DriftEvent{}
	for _, event := range s.events {
		if event.Timestamp.After(since) || event.Timestamp.Equal(since) {
			filtered = append(filtered, event)
		}
	}

	history := &DriftHistory{
		Events: filtered,
		Stats:  s.calculateStats(filtered),
	}

	return history, nil
}

// Clear clears all stored events.
func (s *MemoryStorage) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = []DriftEvent{}
	return nil
}

// calculateStats calculates statistics from events.
func (s *MemoryStorage) calculateStats(events []DriftEvent) DriftStats {
	stats := DriftStats{
		TotalEvents:      len(events),
		EventsByType:     make(map[DriftType]int),
		EventsBySeverity: make(map[DriftSeverity]int),
	}

	if len(events) == 0 {
		return stats
	}

	stats.FirstEvent = events[0].Timestamp
	stats.LastEvent = events[len(events)-1].Timestamp

	remediatedCount := 0
	for _, event := range events {
		stats.EventsByType[event.Type]++
		stats.EventsBySeverity[event.Severity]++

		if event.Remediation != nil && event.Remediation.Status == DriftStatusRemediated {
			remediatedCount++
		}
	}

	if len(events) > 0 {
		stats.RemediationSuccessRate = float64(remediatedCount) / float64(len(events))
	}

	return stats
}

// FileStorage stores drift history in a JSON file.
type FileStorage struct {
	filePath string
	mu       sync.RWMutex
}

// NewFileStorage creates a new file-based storage.
func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{
		filePath: filePath,
	}
}

// Store stores a drift event.
func (s *FileStorage) Store(event DriftEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load existing history
	history, err := s.loadHistory()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if history == nil {
		history = &DriftHistory{Events: []DriftEvent{}}
	}

	// Append new event
	history.Events = append(history.Events, event)

	// Update stats
	memStorage := &MemoryStorage{events: history.Events}
	history.Stats = memStorage.calculateStats(history.Events)

	// Save to file
	return s.saveHistory(history)
}

// GetHistory returns drift history since a given time.
func (s *FileStorage) GetHistory(since time.Time) (*DriftHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadHistory()
	if err != nil {
		if os.IsNotExist(err) {
			return &DriftHistory{Events: []DriftEvent{}}, nil
		}
		return nil, err
	}

	// Filter events
	filtered := []DriftEvent{}
	for _, event := range history.Events {
		if event.Timestamp.After(since) || event.Timestamp.Equal(since) {
			filtered = append(filtered, event)
		}
	}

	history.Events = filtered
	memStorage := &MemoryStorage{events: filtered}
	history.Stats = memStorage.calculateStats(filtered)

	return history, nil
}

// Clear clears all history.
func (s *FileStorage) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.filePath)
}

// loadHistory loads history from file.
func (s *FileStorage) loadHistory() (*DriftHistory, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var history DriftHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return &history, nil
}

// saveHistory saves history to file.
func (s *FileStorage) saveHistory(history *DriftHistory) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// NewStorage creates a storage based on configuration.
func NewStorage(config *StorageConfig) (Storage, error) {
	if config == nil {
		return NewMemoryStorage(), nil
	}

	switch config.Type {
	case "memory", "":
		return NewMemoryStorage(), nil
	case "file":
		if config.Path == "" {
			return nil, fmt.Errorf("file storage requires path")
		}
		return NewFileStorage(config.Path), nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}
