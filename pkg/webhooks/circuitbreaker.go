package webhooks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudcwfranck/kspec/pkg/alerts"
	"github.com/cloudcwfranck/kspec/pkg/metrics"
)

const (
	// ErrorRateThreshold is the error rate that triggers circuit breaker (50%)
	ErrorRateThreshold = 0.5

	// MinRequestsForBreaker is the minimum requests before circuit breaker activates
	MinRequestsForBreaker = 10

	// CircuitBreakerWindow is the time window for error rate calculation
	CircuitBreakerWindow = 1 * time.Minute

	// CircuitBreakerCooldown is the cooldown period before retrying after trip
	CircuitBreakerCooldown = 5 * time.Minute
)

// CircuitBreaker implements a circuit breaker pattern for webhooks
type CircuitBreaker struct {
	mu sync.RWMutex

	// Request tracking
	totalRequests   int
	errorRequests   int
	successRequests int

	// State
	isTripped     bool
	lastTripTime  time.Time
	lastResetTime time.Time

	// Windowed metrics (last N requests)
	requestWindow []requestResult
	windowSize    int

	// AlertManager for sending alerts
	alertManager *alerts.Manager
}

type requestResult struct {
	timestamp time.Time
	isError   bool
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(alertManager *alerts.Manager) *CircuitBreaker {
	return &CircuitBreaker{
		alertManager:  alertManager,
		windowSize:    100, // Track last 100 requests
		requestWindow: make([]requestResult, 0, 100),
		lastResetTime: time.Now(),
	}
}

// RecordSuccess records a successful webhook request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++
	cb.successRequests++

	cb.addToWindow(requestResult{
		timestamp: time.Now(),
		isError:   false,
	})

	cb.checkRecovery()
	cb.updateMetrics()
}

// RecordError records a failed webhook request
func (cb *CircuitBreaker) RecordError() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++
	cb.errorRequests++

	cb.addToWindow(requestResult{
		timestamp: time.Now(),
		isError:   true,
	})

	cb.checkTrip()
	cb.updateMetrics()
}

// IsTripped returns whether the circuit breaker is currently tripped
func (cb *CircuitBreaker) IsTripped() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Check if cooldown period has passed
	if cb.isTripped && time.Since(cb.lastTripTime) > CircuitBreakerCooldown {
		return false // Allow retry after cooldown
	}

	return cb.isTripped
}

// GetErrorRate returns the current error rate
func (cb *CircuitBreaker) GetErrorRate() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.calculateErrorRate()
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		TotalRequests:   cb.totalRequests,
		ErrorRequests:   cb.errorRequests,
		SuccessRequests: cb.successRequests,
		ErrorRate:       cb.calculateErrorRate(),
		IsTripped:       cb.isTripped,
		LastTripTime:    cb.lastTripTime,
	}
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests = 0
	cb.errorRequests = 0
	cb.successRequests = 0
	cb.isTripped = false
	cb.requestWindow = make([]requestResult, 0, cb.windowSize)
	cb.lastResetTime = time.Now()
}

// addToWindow adds a request result to the sliding window
func (cb *CircuitBreaker) addToWindow(result requestResult) {
	// Remove old entries outside the time window
	cutoff := time.Now().Add(-CircuitBreakerWindow)
	validResults := make([]requestResult, 0, cb.windowSize)
	for _, r := range cb.requestWindow {
		if r.timestamp.After(cutoff) {
			validResults = append(validResults, r)
		}
	}

	// Add new result
	validResults = append(validResults, result)

	// Keep only last N requests
	if len(validResults) > cb.windowSize {
		validResults = validResults[len(validResults)-cb.windowSize:]
	}

	cb.requestWindow = validResults
}

// calculateErrorRate calculates the current error rate
func (cb *CircuitBreaker) calculateErrorRate() float64 {
	if cb.totalRequests == 0 {
		return 0.0
	}

	// Calculate from windowed requests
	if len(cb.requestWindow) == 0 {
		return 0.0
	}

	errors := 0
	for _, r := range cb.requestWindow {
		if r.isError {
			errors++
		}
	}

	return float64(errors) / float64(len(cb.requestWindow))
}

// checkTrip checks if circuit breaker should trip
func (cb *CircuitBreaker) checkTrip() {
	// Don't trip if already tripped
	if cb.isTripped {
		return
	}

	// Need minimum requests before tripping
	if len(cb.requestWindow) < MinRequestsForBreaker {
		return
	}

	errorRate := cb.calculateErrorRate()
	if errorRate >= ErrorRateThreshold {

		// Send circuit breaker trip alert
		cb.sendTripAlert(errorRate)
	}
}

// checkRecovery checks if circuit breaker should recover
func (cb *CircuitBreaker) checkRecovery() {
	// Only check recovery if tripped and cooldown passed
	if !cb.isTripped {
		return
	}

	if time.Since(cb.lastTripTime) < CircuitBreakerCooldown {
		return
	}

	// Check if error rate has dropped below threshold
	errorRate := cb.calculateErrorRate()
	if errorRate < ErrorRateThreshold {
		cb.isTripped = false
	}
}

// CircuitBreakerStats contains circuit breaker statistics
type CircuitBreakerStats struct {
	TotalRequests   int
	ErrorRequests   int
	SuccessRequests int
	ErrorRate       float64
	IsTripped       bool
	LastTripTime    time.Time
}

// updateMetrics updates Prometheus metrics (must be called with lock held)
func (cb *CircuitBreaker) updateMetrics() {
	// Update circuit breaker status
	if cb.isTripped {
		metrics.CircuitBreakerTripped.Set(1)
	} else {
		metrics.CircuitBreakerTripped.Set(0)
	}

	// Update error rate
	metrics.CircuitBreakerErrorRate.Set(cb.calculateErrorRate())

	// Update total requests
	metrics.CircuitBreakerTotalRequests.Set(float64(cb.totalRequests))
}

// sendTripAlert sends an alert when the circuit breaker trips
func (cb *CircuitBreaker) sendTripAlert(errorRate float64) {
	if cb.alertManager == nil {
		return
	}

	alert := alerts.Alert{
		Level:       alerts.AlertLevelCritical,
		Title:       "Webhook circuit breaker tripped",
		Description: fmt.Sprintf("Circuit breaker has tripped due to high error rate (%.1f%%). Webhook validation is now in fail-open mode.", errorRate*100),
		Source:      "Webhook/CircuitBreaker",
		EventType:   "CircuitBreakerTripped",
		Labels: map[string]string{
			"component": "webhook",
		},
		Metadata: map[string]interface{}{
			"error_rate":      errorRate,
			"total_requests":  cb.totalRequests,
			"error_requests":  cb.errorRequests,
			"success_requests": cb.successRequests,
		},
	}

	// Send alert in background to avoid blocking
	go func() {
		if err := cb.alertManager.Send(context.Background(), alert); err != nil {
			// Log error but don't fail (alerting is best-effort)
			fmt.Printf("Failed to send circuit breaker alert: %v\n", err)
		}
	}()
}
