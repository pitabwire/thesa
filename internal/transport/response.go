// Package transport contains the HTTP router, middleware chain, and all
// request handlers for the BFF API.
package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/pitabwire/thesa/model"
)

// statusForCode maps ErrorEnvelope codes to HTTP status codes.
var statusForCode = map[string]int{
	model.ErrBadRequest:         http.StatusBadRequest,
	model.ErrUnauthorized:       http.StatusUnauthorized,
	model.ErrForbidden:          http.StatusForbidden,
	model.ErrNotFound:           http.StatusNotFound,
	model.ErrConflict:           http.StatusConflict,
	model.ErrValidationError:    http.StatusUnprocessableEntity,
	model.ErrRateLimited:        http.StatusTooManyRequests,
	model.ErrInternalError:      http.StatusInternalServerError,
	model.ErrBackendUnavailable: http.StatusBadGateway,
	model.ErrBackendTimeout:     http.StatusGatewayTimeout,
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if body != nil {
		json.NewEncoder(w).Encode(body)
	}
}

// WriteError writes an ErrorEnvelope as a JSON response with the correct
// HTTP status code. If err is not an *ErrorEnvelope, a generic 500 is returned.
func WriteError(w http.ResponseWriter, err error) {
	ee, ok := err.(*model.ErrorEnvelope)
	if !ok {
		ee = model.NewInternalError()
	}

	// Populate trace ID if the ResponseWriter carries context (set by traceWriter middleware).
	if tw, ok := w.(*traceWriter); ok && ee.TraceID == "" {
		if span := trace.SpanFromContext(tw.ctx); span.SpanContext().HasTraceID() {
			ee.TraceID = span.SpanContext().TraceID().String()
		}
	}

	status := statusForCode[ee.Code]
	if status == 0 {
		status = http.StatusInternalServerError
	}

	type errorResponse struct {
		Error *model.ErrorEnvelope `json:"error"`
	}
	WriteJSON(w, status, errorResponse{Error: ee})
}

// traceWriter wraps http.ResponseWriter to carry request context for trace ID extraction.
type traceWriter struct {
	http.ResponseWriter
	ctx context.Context
}

// InjectTraceContext is middleware that wraps the ResponseWriter with request context,
// enabling WriteError to automatically include trace IDs in error responses.
func InjectTraceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&traceWriter{ResponseWriter: w, ctx: r.Context()}, r)
	})
}

// WriteNotFound writes a 404 error response.
func WriteNotFound(w http.ResponseWriter, msg string) {
	WriteError(w, model.NewNotFoundError(msg))
}

// WriteForbidden writes a 403 error response.
func WriteForbidden(w http.ResponseWriter, msg string) {
	WriteError(w, model.NewForbiddenError(msg))
}

// WriteValidationError writes a 422 error response with field-level details.
func WriteValidationError(w http.ResponseWriter, details []model.FieldError) {
	WriteError(w, model.NewValidationError(details))
}
