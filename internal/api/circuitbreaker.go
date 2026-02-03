package api

import (
	"sync"
	"time"
)

const (
	// CircuitBreakerThreshold is the number of failures to open the circuit.
	CircuitBreakerThreshold = 5
	// CircuitBreakerResetTime is how long to wait before attempting again.
	CircuitBreakerResetTime = 30 * time.Second
)

// CircuitBreaker prevents cascading failures by tracking consecutive errors.
type CircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	open        bool
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{}
}

// IsOpen returns true if the circuit is open (too many failures).
// Automatically resets after CircuitBreakerResetTime.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		return false
	}

	// Auto-reset after timeout (half-open state)
	if time.Since(cb.lastFailure) > CircuitBreakerResetTime {
		cb.open = false
		cb.failures = 0
		return false
	}

	return true
}

// RecordSuccess resets the failure count and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.open = false
}

// RecordFailure increments failure count.
// Returns true if the circuit is now open.
func (cb *CircuitBreaker) RecordFailure() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= CircuitBreakerThreshold {
		cb.open = true
		return true
	}

	return false
}

// Failures returns current failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}
