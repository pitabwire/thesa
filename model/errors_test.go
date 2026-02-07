package model

import "testing"

func TestErrorEnvelope_Error(t *testing.T) {
	e := &ErrorEnvelope{Code: ErrNotFound, Message: "Page not found"}
	want := "NOT_FOUND: Page not found"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorEnvelope_implements_error(t *testing.T) {
	var _ error = (*ErrorEnvelope)(nil)
}

func TestNewNotFoundError(t *testing.T) {
	e := NewNotFoundError("resource missing")
	if e.Code != ErrNotFound {
		t.Errorf("Code = %q, want %q", e.Code, ErrNotFound)
	}
	if e.Message != "resource missing" {
		t.Errorf("Message = %q, want %q", e.Message, "resource missing")
	}
}

func TestNewForbiddenError(t *testing.T) {
	e := NewForbiddenError("access denied")
	if e.Code != ErrForbidden {
		t.Errorf("Code = %q, want %q", e.Code, ErrForbidden)
	}
}

func TestNewValidationError(t *testing.T) {
	details := []FieldError{
		{Field: "email", Code: "REQUIRED", Message: "Email is required"},
	}
	e := NewValidationError(details)
	if e.Code != ErrValidationError {
		t.Errorf("Code = %q, want %q", e.Code, ErrValidationError)
	}
	if len(e.Details) != 1 {
		t.Fatalf("Details length = %d, want 1", len(e.Details))
	}
	if e.Details[0].Field != "email" {
		t.Errorf("Details[0].Field = %q, want %q", e.Details[0].Field, "email")
	}
}

func TestNewInternalError(t *testing.T) {
	e := NewInternalError()
	if e.Code != ErrInternalError {
		t.Errorf("Code = %q, want %q", e.Code, ErrInternalError)
	}
}

func TestNewBackendUnavailableError(t *testing.T) {
	e := NewBackendUnavailableError()
	if e.Code != ErrBackendUnavailable {
		t.Errorf("Code = %q, want %q", e.Code, ErrBackendUnavailable)
	}
}

func TestNewBackendTimeoutError(t *testing.T) {
	e := NewBackendTimeoutError()
	if e.Code != ErrBackendTimeout {
		t.Errorf("Code = %q, want %q", e.Code, ErrBackendTimeout)
	}
}

func TestNewRateLimitedError(t *testing.T) {
	e := NewRateLimitedError()
	if e.Code != ErrRateLimited {
		t.Errorf("Code = %q, want %q", e.Code, ErrRateLimited)
	}
}

func TestNewBadRequestError(t *testing.T) {
	e := NewBadRequestError("bad json")
	if e.Code != ErrBadRequest {
		t.Errorf("Code = %q, want %q", e.Code, ErrBadRequest)
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	e := NewUnauthorizedError("missing token")
	if e.Code != ErrUnauthorized {
		t.Errorf("Code = %q, want %q", e.Code, ErrUnauthorized)
	}
}

func TestNewConflictError(t *testing.T) {
	e := NewConflictError("duplicate key")
	if e.Code != ErrConflict {
		t.Errorf("Code = %q, want %q", e.Code, ErrConflict)
	}
}
