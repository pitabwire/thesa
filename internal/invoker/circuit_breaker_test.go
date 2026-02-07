package invoker

import (
	"testing"
	"time"
)

func TestCircuitBreaker_startsClosedPassesThrough(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

	if s := cb.State(); s != BreakerClosed {
		t.Errorf("initial state = %v, want Closed", s)
	}
	if err := cb.Allow(); err != nil {
		t.Errorf("Allow() error = %v, want nil", err)
	}
}

func TestCircuitBreaker_opensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

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
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)

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
	cb := NewCircuitBreaker(1, 1, 10*time.Millisecond)

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
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond)

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
	cb := NewCircuitBreaker(1, 2, 10*time.Millisecond)

	cb.RecordFailure() // Open
	time.Sleep(20 * time.Millisecond)
	cb.Allow() // transitions to HalfOpen

	cb.RecordFailure() // immediately reopens
	if s := cb.State(); s != BreakerOpen {
		t.Errorf("state = %v, want Open after HalfOpen failure", s)
	}
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := NewCircuitBreaker(5, 2, time.Minute)

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
	cb := NewCircuitBreaker(0, 0, 0) // all zeros → defaults applied

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
