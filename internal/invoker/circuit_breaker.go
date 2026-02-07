package invoker

import (
	"fmt"
	"sync"
	"time"
)

// BreakerState represents the current state of a circuit breaker.
type BreakerState int

const (
	// BreakerClosed allows all requests through. Failures are counted.
	BreakerClosed BreakerState = iota
	// BreakerOpen rejects all requests immediately.
	BreakerOpen
	// BreakerHalfOpen allows a single probe request through.
	BreakerHalfOpen
)

func (s BreakerState) String() string {
	switch s {
	case BreakerClosed:
		return "closed"
	case BreakerOpen:
		return "open"
	case BreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern with three states:
// Closed → Open → HalfOpen. It is safe for concurrent use.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            BreakerState
	failures         int
	successes        int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	openedAt         time.Time
}

// NewCircuitBreaker creates a circuit breaker with the given thresholds.
// failureThreshold: consecutive failures to trip from Closed → Open.
// successThreshold: consecutive successes in HalfOpen to return to Closed.
// timeout: duration to stay Open before transitioning to HalfOpen.
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	if failureThreshold < 1 {
		failureThreshold = 5
	}
	if successThreshold < 1 {
		successThreshold = 2
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &CircuitBreaker{
		state:            BreakerClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// Allow checks whether a request should be allowed through.
// Returns nil if allowed, or an error if the circuit is open.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case BreakerClosed:
		return nil
	case BreakerOpen:
		if time.Since(cb.openedAt) > cb.timeout {
			cb.state = BreakerHalfOpen
			cb.successes = 0
			return nil
		}
		return fmt.Errorf("circuit breaker is open")
	case BreakerHalfOpen:
		return nil
	}
	return nil
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case BreakerClosed:
		cb.failures = 0
	case BreakerHalfOpen:
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = BreakerClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case BreakerClosed:
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			cb.state = BreakerOpen
			cb.openedAt = time.Now()
		}
	case BreakerHalfOpen:
		// Any failure in half-open immediately reopens.
		cb.state = BreakerOpen
		cb.openedAt = time.Now()
		cb.successes = 0
	}
}

// State returns the current breaker state.
func (cb *CircuitBreaker) State() BreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == BreakerOpen && time.Since(cb.openedAt) > cb.timeout {
		cb.state = BreakerHalfOpen
		cb.successes = 0
	}
	return cb.state
}

// Counts returns the current failure and success counts (for diagnostics).
func (cb *CircuitBreaker) Counts() (failures, successes int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures, cb.successes
}
