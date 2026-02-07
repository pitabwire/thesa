package model

import "fmt"

// Standard error codes.
const (
	ErrBadRequest         = "BAD_REQUEST"
	ErrUnauthorized       = "UNAUTHORIZED"
	ErrForbidden          = "FORBIDDEN"
	ErrNotFound           = "NOT_FOUND"
	ErrConflict           = "CONFLICT"
	ErrValidationError    = "VALIDATION_ERROR"
	ErrInvalidTransition  = "INVALID_TRANSITION"
	ErrRateLimited        = "RATE_LIMITED"
	ErrInternalError      = "INTERNAL_ERROR"
	ErrBackendUnavailable = "BACKEND_UNAVAILABLE"
	ErrBackendTimeout     = "BACKEND_TIMEOUT"
)

// Workflow-specific error codes.
const (
	ErrWorkflowNotFound   = "WORKFLOW_NOT_FOUND"
	ErrWorkflowNotActive  = "WORKFLOW_NOT_ACTIVE"
	ErrStepUnauthorized   = "STEP_UNAUTHORIZED"
	ErrWorkflowExpired    = "WORKFLOW_EXPIRED"
	ErrWorkflowChainLimit = "WORKFLOW_CHAIN_LIMIT"
)

// ErrorEnvelope is the standard error response envelope returned by the BFF.
// It implements the error interface.
type ErrorEnvelope struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
	TraceID string       `json:"trace_id"`
}

// Error implements the error interface.
func (e *ErrorEnvelope) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// FieldError describes a field-level validation error.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewBadRequestError returns a BAD_REQUEST error.
func NewBadRequestError(msg string) *ErrorEnvelope {
	return &ErrorEnvelope{Code: ErrBadRequest, Message: msg}
}

// NewUnauthorizedError returns an UNAUTHORIZED error.
func NewUnauthorizedError(msg string) *ErrorEnvelope {
	return &ErrorEnvelope{Code: ErrUnauthorized, Message: msg}
}

// NewForbiddenError returns a FORBIDDEN error.
func NewForbiddenError(msg string) *ErrorEnvelope {
	return &ErrorEnvelope{Code: ErrForbidden, Message: msg}
}

// NewNotFoundError returns a NOT_FOUND error.
func NewNotFoundError(msg string) *ErrorEnvelope {
	return &ErrorEnvelope{Code: ErrNotFound, Message: msg}
}

// NewConflictError returns a CONFLICT error.
func NewConflictError(msg string) *ErrorEnvelope {
	return &ErrorEnvelope{Code: ErrConflict, Message: msg}
}

// NewValidationError returns a VALIDATION_ERROR with field-level details.
func NewValidationError(details []FieldError) *ErrorEnvelope {
	return &ErrorEnvelope{
		Code:    ErrValidationError,
		Message: "One or more fields are invalid",
		Details: details,
	}
}

// NewInternalError returns an INTERNAL_ERROR.
func NewInternalError() *ErrorEnvelope {
	return &ErrorEnvelope{
		Code:    ErrInternalError,
		Message: "An unexpected error occurred",
	}
}

// NewBackendUnavailableError returns a BACKEND_UNAVAILABLE error.
func NewBackendUnavailableError() *ErrorEnvelope {
	return &ErrorEnvelope{
		Code:    ErrBackendUnavailable,
		Message: "The backend service is temporarily unavailable",
	}
}

// NewBackendTimeoutError returns a BACKEND_TIMEOUT error.
func NewBackendTimeoutError() *ErrorEnvelope {
	return &ErrorEnvelope{
		Code:    ErrBackendTimeout,
		Message: "The backend service did not respond in time",
	}
}

// NewRateLimitedError returns a RATE_LIMITED error.
func NewRateLimitedError() *ErrorEnvelope {
	return &ErrorEnvelope{
		Code:    ErrRateLimited,
		Message: "Rate limit exceeded. Please try again later.",
	}
}
