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

// minErrorRateSamples is the minimum number of requests in a window before
// the error rate threshold is evaluated. This prevents tripping on very
// few requests (e.g. 1 failure out of 1 total = 100% but not meaningful).
const minErrorRateSamples = 10

// CircuitBreaker implements the circuit breaker pattern with three states:
// Closed → Open → HalfOpen. It trips on either consecutive failure count
// or error rate within a sliding window. It is safe for concurrent use.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            BreakerState
	failures         int
	successes        int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	openedAt         time.Time

	// Error rate tracking (tumbling window).
	errorRateThreshold float64
	errorRateWindow    time.Duration
	windowStart        time.Time
	windowTotal        int
	windowFailures     int
}

// NewCircuitBreaker creates a circuit breaker with the given thresholds.
// failureThreshold: consecutive failures to trip from Closed → Open.
// successThreshold: consecutive successes in HalfOpen to return to Closed.
// timeout: duration to stay Open before transitioning to HalfOpen.
// errorRateThreshold: error rate (0.0–1.0) to trip; 0 disables rate-based tripping.
// errorRateWindow: time window for computing the error rate; 0 disables.
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration,
	errorRateThreshold float64, errorRateWindow time.Duration) *CircuitBreaker {
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
		state:              BreakerClosed,
		failureThreshold:   failureThreshold,
		successThreshold:   successThreshold,
		timeout:            timeout,
		errorRateThreshold: errorRateThreshold,
		errorRateWindow:    errorRateWindow,
		windowStart:        time.Now(),
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
		cb.recordWindowCall(false)
	case BreakerHalfOpen:
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = BreakerClosed
			cb.failures = 0
			cb.successes = 0
			cb.resetWindow()
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
		cb.recordWindowCall(true)

		// Trip on consecutive failure threshold OR error rate threshold.
		if cb.failures >= cb.failureThreshold || cb.errorRateExceeded() {
			cb.state = BreakerOpen
			cb.openedAt = time.Now()
			cb.resetWindow()
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

// ErrorRate returns the current error rate and total requests in the window.
func (cb *CircuitBreaker) ErrorRate() (rate float64, total int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.maybeResetWindow()
	if cb.windowTotal == 0 {
		return 0, 0
	}
	return float64(cb.windowFailures) / float64(cb.windowTotal), cb.windowTotal
}

// recordWindowCall tracks a call in the tumbling window. Must be called with lock held.
func (cb *CircuitBreaker) recordWindowCall(isFailure bool) {
	if cb.errorRateWindow <= 0 {
		return
	}
	cb.maybeResetWindow()
	cb.windowTotal++
	if isFailure {
		cb.windowFailures++
	}
}

// maybeResetWindow resets the tumbling window if it has expired. Must be called with lock held.
func (cb *CircuitBreaker) maybeResetWindow() {
	if cb.errorRateWindow <= 0 {
		return
	}
	if time.Since(cb.windowStart) > cb.errorRateWindow {
		cb.windowStart = time.Now()
		cb.windowTotal = 0
		cb.windowFailures = 0
	}
}

// resetWindow clears the window counters. Must be called with lock held.
func (cb *CircuitBreaker) resetWindow() {
	cb.windowStart = time.Now()
	cb.windowTotal = 0
	cb.windowFailures = 0
}

// errorRateExceeded checks if the error rate in the current window exceeds the
// threshold. Requires at least minErrorRateSamples requests. Must be called with lock held.
func (cb *CircuitBreaker) errorRateExceeded() bool {
	if cb.errorRateThreshold <= 0 || cb.errorRateWindow <= 0 {
		return false
	}
	if cb.windowTotal < minErrorRateSamples {
		return false
	}
	rate := float64(cb.windowFailures) / float64(cb.windowTotal)
	return rate >= cb.errorRateThreshold
}
