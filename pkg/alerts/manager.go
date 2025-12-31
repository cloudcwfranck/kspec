package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// Manager manages alert notifiers and routes alerts to appropriate destinations
type Manager struct {
	notifiers map[string]Notifier
	stats     map[string]*NotifierStats
	mu        sync.RWMutex
	logger    logr.Logger
}

// NewManager creates a new alert manager
func NewManager(logger logr.Logger) *Manager {
	return &Manager{
		notifiers: make(map[string]Notifier),
		stats:     make(map[string]*NotifierStats),
		logger:    logger,
	}
}

// AddNotifier adds a notifier to the manager
func (m *Manager) AddNotifier(n Notifier) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := n.Name()
	if name == "" {
		return fmt.Errorf("notifier name cannot be empty")
	}

	m.notifiers[name] = n
	m.stats[name] = &NotifierStats{
		Name: name,
	}

	m.logger.Info("Added notifier", "name", name)
	return nil
}

// RemoveNotifier removes a notifier from the manager
func (m *Manager) RemoveNotifier(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.notifiers, name)
	delete(m.stats, name)

	m.logger.Info("Removed notifier", "name", name)
}

// Send sends an alert to all appropriate notifiers
func (m *Manager) Send(ctx context.Context, alert Alert) error {
	m.mu.RLock()
	notifiers := make(map[string]Notifier)
	for name, notifier := range m.notifiers {
		notifiers[name] = notifier
	}
	m.mu.RUnlock()

	if len(notifiers) == 0 {
		m.logger.V(1).Info("No notifiers configured, skipping alert", "title", alert.Title)
		return nil
	}

	// Set timestamp if not already set
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Send to all enabled notifiers that should receive this alert
	var errs []error
	sentCount := 0

	for name, notifier := range notifiers {
		// Skip if disabled
		if !notifier.Enabled() {
			m.logger.V(1).Info("Notifier disabled, skipping", "notifier", name)
			continue
		}

		// Check if this notifier should receive this alert
		if !notifier.ShouldSend(alert) {
			m.logger.V(1).Info("Notifier filtered alert", "notifier", name, "eventType", alert.EventType)
			continue
		}

		// Send the alert
		if err := notifier.Send(ctx, alert); err != nil {
			m.logger.Error(err, "Failed to send alert", "notifier", name, "title", alert.Title)
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			m.recordFailure(name, err)
		} else {
			m.logger.Info("Alert sent successfully", "notifier", name, "title", alert.Title)
			m.recordSuccess(name)
			sentCount++
		}
	}

	if len(errs) > 0 && sentCount == 0 {
		// All notifiers failed
		return fmt.Errorf("all notifiers failed: %v", errs)
	} else if len(errs) > 0 {
		// Some notifiers failed
		return fmt.Errorf("some notifiers failed: %v", errs)
	}

	return nil
}

// SendToNotifier sends an alert to a specific notifier
func (m *Manager) SendToNotifier(ctx context.Context, notifierName string, alert Alert) error {
	m.mu.RLock()
	notifier, exists := m.notifiers[notifierName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("notifier not found: %s", notifierName)
	}

	if !notifier.Enabled() {
		return fmt.Errorf("notifier is disabled: %s", notifierName)
	}

	// Set timestamp if not already set
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	if err := notifier.Send(ctx, alert); err != nil {
		m.recordFailure(notifierName, err)
		return fmt.Errorf("failed to send to %s: %w", notifierName, err)
	}

	m.recordSuccess(notifierName)
	return nil
}

// GetStats returns statistics for all notifiers
func (m *Manager) GetStats() map[string]NotifierStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]NotifierStats)
	for name, stat := range m.stats {
		// Copy struct to avoid race conditions
		stats[name] = *stat
	}

	return stats
}

// GetNotifier returns a notifier by name
func (m *Manager) GetNotifier(name string) (Notifier, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notifier, exists := m.notifiers[name]
	return notifier, exists
}

// ListNotifiers returns names of all registered notifiers
func (m *Manager) ListNotifiers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.notifiers))
	for name := range m.notifiers {
		names = append(names, name)
	}

	return names
}

// recordSuccess records a successful alert send
func (m *Manager) recordSuccess(notifierName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stat, exists := m.stats[notifierName]; exists {
		stat.Sent++
		stat.LastSent = time.Now()
		stat.LastError = nil
		stat.LastErrorMsg = ""
	}
}

// recordFailure records a failed alert send
func (m *Manager) recordFailure(notifierName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stat, exists := m.stats[notifierName]; exists {
		stat.Failed++
		stat.LastError = err
		if err != nil {
			stat.LastErrorMsg = err.Error()
		}
	}
}

// Clear removes all notifiers and resets stats
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifiers = make(map[string]Notifier)
	m.stats = make(map[string]*NotifierStats)

	m.logger.Info("Cleared all notifiers")
}
