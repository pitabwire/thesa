package invoker

import (
	"testing"
	"time"
)

func TestCircuitBreaker_startsClosedPassesThrough(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond, 0, 0)

	if s := cb.State(); s != BreakerClosed {
		t.Errorf("initial state = %v, want Closed", s)
	}
	if err := cb.Allow(); err != nil {
		t.Errorf("Allow() error = %v, want nil", err)
	}
}

func TestCircuitBreaker_opensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond, 0, 0)

	cb.RecordFailure()
	cb.RecordFailure()
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state after 2 failures = %v, want Closed", s)
	}

	cb.RecordFailure() // 3rd failure → Open
	if s := cb.State(); s != BreakerOpen {
		t.Errorf("state after 3 failures = %v, want Open", s)
	}
	if err := cb.Allow(); err == nil {
		t.Error("Allow() should return error when Open")
	}
}

func TestCircuitBreaker_successResetsFailureCount(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond, 0, 0)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // resets failure count

	cb.RecordFailure()
	cb.RecordFailure()
	// Only 2 failures since reset, should still be Closed.
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state = %v, want Closed after reset", s)
	}
}

func TestCircuitBreaker_transitionsToHalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(1, 1, 10*time.Millisecond, 0, 0)

	cb.RecordFailure() // Open
	if s := cb.State(); s != BreakerOpen {
		t.Fatalf("state = %v, want Open", s)
	}

	time.Sleep(20 * time.Millisecond)

	if s := cb.State(); s != BreakerHalfOpen {
		t.Errorf("state after timeout = %v, want HalfOpen", s)
	}
	if err := cb.Allow(); err != nil {
		t.Errorf("Allow() in HalfOpen should return nil, got %v", err)
	}
}

func TestCircuitBreaker_halfOpenToClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond, 0, 0)

	cb.RecordFailure() // Open
	time.Sleep(20 * time.Millisecond)
	cb.Allow() // transitions to HalfOpen

	cb.RecordSuccess()
	if s := cb.State(); s != BreakerHalfOpen {
		t.Errorf("state after 1 success = %v, want HalfOpen", s)
	}

	cb.RecordSuccess() // 2nd success → Closed
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state after 2 successes = %v, want Closed", s)
	}
}

func TestCircuitBreaker_halfOpenToOpenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond, 0, 0)

	cb.RecordFailure() // Open
	time.Sleep(20 * time.Millisecond)
	cb.Allow() // transitions to HalfOpen

	cb.RecordFailure() // immediately reopens
	if s := cb.State(); s != BreakerOpen {
		t.Errorf("state = %v, want Open after HalfOpen failure", s)
	}
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := NewCircuitBreaker(5, 2, time.Minute, 0, 0)

	cb.RecordFailure()
	cb.RecordFailure()
	f, s := cb.Counts()
	if f != 2 || s != 0 {
		t.Errorf("Counts() = (%d, %d), want (2, 0)", f, s)
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	if BreakerClosed.String() != "closed" {
		t.Error("Closed string mismatch")
	}
	if BreakerOpen.String() != "open" {
		t.Error("Open string mismatch")
	}
	if BreakerHalfOpen.String() != "half-open" {
		t.Error("HalfOpen string mismatch")
	}
}

func TestCircuitBreaker_defaultValues(t *testing.T) {
	cb := NewCircuitBreaker(0, 0, 0, 0, 0) // all zeros → defaults applied

	// Should default to 5 failures, 2 successes, 30s timeout.
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state after 4 failures = %v, want Closed (default threshold=5)", s)
	}
	cb.RecordFailure() // 5th → Open
	if s := cb.State(); s != BreakerOpen {
		t.Errorf("state after 5 failures = %v, want Open", s)
	}
}

// --- Error rate tracking ---

func TestCircuitBreaker_errorRateTripsBreaker(t *testing.T) {
	// High failure threshold (100) so consecutive failures alone won't trip.
	// Error rate threshold at 50% with a long window.
	cb := NewCircuitBreaker(100, 2, time.Minute, 0.5, time.Minute)

	// Record 6 successes and 4 failures → 4/10 = 40% < 50%, still closed.
	for i := 0; i < 6; i++ {
		cb.RecordSuccess()
	}
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state at 40%% error rate = %v, want Closed", s)
	}

	// Record more failures to push rate above 50%.
	// Current: 10 total (6 success, 4 fail). Adding failures:
	// 11 total, 5 fail = 45% → still closed
	cb.RecordFailure()
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state at ~45%% error rate = %v, want Closed", s)
	}

	// 12 total, 6 fail = 50% → trips
	cb.RecordFailure()
	if s := cb.State(); s != BreakerOpen {
		t.Errorf("state at 50%% error rate = %v, want Open", s)
	}
}

func TestCircuitBreaker_errorRateRequiresMinSamples(t *testing.T) {
	// Error rate threshold at 10% but min samples is 10.
	cb := NewCircuitBreaker(100, 2, time.Minute, 0.1, time.Minute)

	// 1 failure out of 1 = 100% but below min samples.
	cb.RecordFailure()
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state = %v, want Closed (below min samples)", s)
	}

	// Get to 9 total (still below minErrorRateSamples=10).
	for i := 0; i < 8; i++ {
		cb.RecordFailure()
	}
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state = %v, want Closed (9 samples < min 10)", s)
	}

	// 10th request as failure → 10/10 = 100% > 10%, trips.
	cb.RecordFailure()
	if s := cb.State(); s != BreakerOpen {
		t.Errorf("state at 100%% with 10 samples = %v, want Open", s)
	}
}

func TestCircuitBreaker_errorRateDisabledWhenZero(t *testing.T) {
	// Error rate disabled (threshold=0).
	cb := NewCircuitBreaker(100, 2, time.Minute, 0, time.Minute)

	// Record 20 failures — would exceed any rate threshold but rate checking is disabled.
	for i := 0; i < 20; i++ {
		cb.RecordFailure()
	}
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state = %v, want Closed (error rate disabled)", s)
	}
}

func TestCircuitBreaker_errorRateWindowExpiry(t *testing.T) {
	// Short window so we can test expiry.
	cb := NewCircuitBreaker(100, 2, time.Minute, 0.5, 20*time.Millisecond)

	// Fill the window with mostly failures.
	for i := 0; i < 4; i++ {
		cb.RecordSuccess()
	}
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	// 9 total, 5 failures = 55% but below min samples (10), still closed.

	// Wait for window to expire.
	time.Sleep(30 * time.Millisecond)

	// After expiry, window resets. New requests start fresh.
	rate, total := cb.ErrorRate()
	if total != 0 {
		t.Errorf("window total after expiry = %d, want 0 (rate=%f)", total, rate)
	}
}

func TestCircuitBreaker_ErrorRate(t *testing.T) {
	cb := NewCircuitBreaker(100, 2, time.Minute, 0.5, time.Minute)

	rate, total := cb.ErrorRate()
	if rate != 0 || total != 0 {
		t.Errorf("initial ErrorRate() = (%f, %d), want (0, 0)", rate, total)
	}

	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordFailure()

	rate, total = cb.ErrorRate()
	if total != 3 {
		t.Errorf("ErrorRate() total = %d, want 3", total)
	}
	wantRate := 1.0 / 3.0
	if rate < wantRate-0.01 || rate > wantRate+0.01 {
		t.Errorf("ErrorRate() rate = %f, want ~%f", rate, wantRate)
	}
}

func TestCircuitBreaker_windowResetsAfterHalfOpenRecovery(t *testing.T) {
	// Use a high failure threshold, low error rate threshold.
	cb := NewCircuitBreaker(3, 1, 10*time.Millisecond, 0.5, time.Minute)

	// Trip via consecutive failures.
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	if s := cb.State(); s != BreakerOpen {
		t.Fatalf("state = %v, want Open", s)
	}

	// Wait for half-open.
	time.Sleep(20 * time.Millisecond)
	cb.Allow()

	// Recover with success → back to Closed.
	cb.RecordSuccess()
	if s := cb.State(); s != BreakerClosed {
		t.Errorf("state = %v, want Closed", s)
	}

	// Window should be reset — ErrorRate should show 0 total.
	_, total := cb.ErrorRate()
	if total != 0 {
		t.Errorf("window total after recovery = %d, want 0", total)
	}
}
