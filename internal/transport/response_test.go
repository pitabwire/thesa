package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pitabwire/thesa/model"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if xct := w.Header().Get("X-Content-Type-Options"); xct != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", xct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["hello"] != "world" {
		t.Errorf("body = %v", body)
	}
}

func TestWriteError_envelope(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, model.NewNotFoundError("page not found"))

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}

	var resp struct {
		Error model.ErrorEnvelope `json:"error"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", resp.Error.Code)
	}
}

func TestWriteError_non_envelope(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, fmt.Errorf("something went wrong"))

	if w.Code != 500 {
		t.Errorf("status = %d, want 500 for non-envelope error", w.Code)
	}
}

func TestWriteNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	WriteNotFound(w, "resource missing")
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestWriteForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	WriteForbidden(w, "access denied")
	if w.Code != 403 {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteValidationError(w, []model.FieldError{
		{Field: "email", Code: "REQUIRED", Message: "email is required"},
	})
	if w.Code != 422 {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestStatusForCode_coverage(t *testing.T) {
	codes := []struct {
		code   string
		status int
	}{
		{model.ErrBadRequest, 400},
		{model.ErrUnauthorized, 401},
		{model.ErrForbidden, 403},
		{model.ErrNotFound, 404},
		{model.ErrConflict, 409},
		{model.ErrValidationError, 422},
		{model.ErrRateLimited, 429},
		{model.ErrInternalError, 500},
		{model.ErrBackendUnavailable, 502},
		{model.ErrBackendTimeout, 504},
	}
	for _, tc := range codes {
		t.Run(tc.code, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, &model.ErrorEnvelope{Code: tc.code, Message: "test"})
			if w.Code != tc.status {
				t.Errorf("status for %s = %d, want %d", tc.code, w.Code, tc.status)
			}
		})
	}
}
