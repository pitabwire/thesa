package invoker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

// testSpec is a minimal valid OpenAPI 3.0 spec for testing.
const testSpec = `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: listUsers
      parameters:
        - name: page
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
              properties:
                name:
                  type: string
      responses:
        "201":
          description: Created
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
    put:
      operationId: updateUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        "200":
          description: OK
    delete:
      operationId: deleteUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: No Content
`

func writeSpecFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(path, []byte(testSpec), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func loadTestIndex(t *testing.T, baseURL string) *openapi.Index {
	t.Helper()
	specPath := writeSpecFile(t)
	idx := openapi.NewIndex()
	err := idx.Load([]openapi.SpecSource{{
		ServiceID: "test-svc",
		BaseURL:   baseURL,
		SpecPath:  specPath,
	}})
	if err != nil {
		t.Fatalf("loading test spec: %v", err)
	}
	return idx
}

func defaultServiceConfig() config.ServiceConfig {
	return config.ServiceConfig{
		Timeout: 5 * time.Second,
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
		Retry: config.RetryConfig{
			MaxAttempts:    1,
			IdempotentOnly: true,
		},
	}
}

func newTestInvoker(t *testing.T, serverURL string, svcCfg config.ServiceConfig) *OpenAPIOperationInvoker {
	t.Helper()
	idx := loadTestIndex(t, serverURL)
	return NewOpenAPIOperationInvoker(idx, map[string]config.ServiceConfig{
		"test-svc": svcCfg,
	})
}

// --- Supports ---

func TestOpenAPIOperationInvoker_Supports(t *testing.T) {
	inv := newTestInvoker(t, "http://localhost", defaultServiceConfig())

	if !inv.Supports(model.OperationBinding{Type: "openapi"}) {
		t.Error("Supports(openapi) = false, want true")
	}
	if inv.Supports(model.OperationBinding{Type: "sdk"}) {
		t.Error("Supports(sdk) = true, want false")
	}
	if inv.Supports(model.OperationBinding{Type: ""}) {
		t.Error("Supports('') = true, want false")
	}
}

// --- Successful invocations ---

func TestOpenAPIOperationInvoker_Invoke_GETSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/users" {
			t.Errorf("path = %s, want /users", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{"alice", "bob"},
			"total": 2,
		})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		&model.RequestContext{Token: "tok123", TenantID: "t1", CorrelationID: "c1"},
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	body, ok := result.Body.(map[string]any)
	if !ok {
		t.Fatalf("Body type = %T, want map[string]any", result.Body)
	}
	if total, _ := body["total"].(float64); total != 2 {
		t.Errorf("Body.total = %v, want 2", body["total"])
	}
}

func TestOpenAPIOperationInvoker_Invoke_pathParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/abc-123" {
			t.Errorf("path = %s, want /users/abc-123", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "abc-123", "name": "Alice"})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "getUser"},
		model.InvocationInput{PathParams: map[string]string{"id": "abc-123"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	body := result.Body.(map[string]any)
	if body["id"] != "abc-123" {
		t.Errorf("Body.id = %v, want abc-123", body["id"])
	}
}

func TestOpenAPIOperationInvoker_Invoke_pathParamsEscaped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL-encoded "hello world" → "hello%20world"
		if r.URL.RawPath != "" && r.URL.RawPath != "/users/hello%20world" {
			t.Errorf("raw path = %s, want /users/hello%%20world", r.URL.RawPath)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "getUser"},
		model.InvocationInput{PathParams: map[string]string{"id": "hello world"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

func TestOpenAPIOperationInvoker_Invoke_queryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "3" {
			t.Errorf("query page = %s, want 3", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("size") != "25" {
			t.Errorf("query size = %s, want 25", r.URL.Query().Get("size"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"page": 3})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{QueryParams: map[string]string{"page": "3", "size": "25"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
}

func TestOpenAPIOperationInvoker_Invoke_POSTWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if body["name"] != "Alice" {
			t.Errorf("body.name = %v, want Alice", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": "new-1", "name": "Alice"})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "createUser"},
		model.InvocationInput{Body: map[string]any{"name": "Alice"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want 201", result.StatusCode)
	}
}

func TestOpenAPIOperationInvoker_Invoke_PUTWithPathAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/users/u99" {
			t.Errorf("path = %s, want /users/u99", r.URL.Path)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		var body map[string]any
		json.Unmarshal(bodyBytes, &body)
		if body["name"] != "Updated" {
			t.Errorf("body.name = %v, want Updated", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "u99", "name": "Updated"})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "updateUser"},
		model.InvocationInput{
			PathParams: map[string]string{"id": "u99"},
			Body:       map[string]any{"name": "Updated"},
		},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
}

func TestOpenAPIOperationInvoker_Invoke_DELETE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/users/del-1" {
			t.Errorf("path = %s, want /users/del-1", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "deleteUser"},
		model.InvocationInput{PathParams: map[string]string{"id": "del-1"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusNoContent {
		t.Errorf("StatusCode = %d, want 204", result.StatusCode)
	}
}

// --- Header handling ---

func TestOpenAPIOperationInvoker_Invoke_forwardsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer my-jwt-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer my-jwt-token")
		}
		if got := r.Header.Get("X-Tenant-Id"); got != "tenant-42" {
			t.Errorf("X-Tenant-Id = %q, want %q", got, "tenant-42")
		}
		if got := r.Header.Get("X-Partition-Id"); got != "part-7" {
			t.Errorf("X-Partition-Id = %q, want %q", got, "part-7")
		}
		if got := r.Header.Get("X-Correlation-Id"); got != "corr-abc" {
			t.Errorf("X-Correlation-Id = %q, want %q", got, "corr-abc")
		}
		if got := r.Header.Get("X-Request-Subject"); got != "sub-1" {
			t.Errorf("X-Request-Subject = %q, want %q", got, "sub-1")
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	rctx := &model.RequestContext{
		Token:         "my-jwt-token",
		TenantID:      "tenant-42",
		PartitionID:   "part-7",
		CorrelationID: "corr-abc",
		SubjectID:     "sub-1",
	}

	_, err := inv.Invoke(
		context.Background(),
		rctx,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

func TestOpenAPIOperationInvoker_Invoke_customInputHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom-Header"); got != "custom-value" {
			t.Errorf("X-Custom-Header = %q, want %q", got, "custom-value")
		}
		// Custom headers can override standard ones.
		if got := r.Header.Get("Accept"); got != "text/plain" {
			t.Errorf("Accept = %q, want %q (overridden by input)", got, "text/plain")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"Accept":          "text/plain",
		}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

func TestOpenAPIOperationInvoker_Invoke_sanitizesHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Newlines should be stripped from header values.
		if got := r.Header.Get("Authorization"); got != "Bearer injectedvalue" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer injectedvalue")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	// Token with injected newlines.
	rctx := &model.RequestContext{
		Token: "injected\r\nvalue",
	}

	_, err := inv.Invoke(
		context.Background(),
		rctx,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

func TestOpenAPIOperationInvoker_Invoke_nilRequestContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// With nil rctx, no Authorization header should be set.
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

// --- Response header extraction ---

func TestOpenAPIOperationInvoker_Invoke_extractsResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Correlation-Id", "resp-corr-1")
		w.Header().Set("X-Trace-Id", "trace-abc")
		w.Header().Set("Retry-After", "30")
		// This header should NOT be extracted (not in allowed list).
		w.Header().Set("X-Internal-Debug", "debug-info")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", result.Headers["Content-Type"])
	}
	if result.Headers["X-Correlation-Id"] != "resp-corr-1" {
		t.Errorf("X-Correlation-Id = %q, want resp-corr-1", result.Headers["X-Correlation-Id"])
	}
	if result.Headers["X-Trace-Id"] != "trace-abc" {
		t.Errorf("X-Trace-Id = %q, want trace-abc", result.Headers["X-Trace-Id"])
	}
	if result.Headers["Retry-After"] != "30" {
		t.Errorf("Retry-After = %q, want 30", result.Headers["Retry-After"])
	}
	if _, exists := result.Headers["X-Internal-Debug"]; exists {
		t.Error("X-Internal-Debug should not be in extracted headers")
	}
}

// --- Non-JSON and empty responses ---

func TestOpenAPIOperationInvoker_Invoke_nonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	// Non-JSON body should not be parsed.
	if result.Body != nil {
		t.Errorf("Body = %v, want nil (non-JSON)", result.Body)
	}
}

func TestOpenAPIOperationInvoker_Invoke_emptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	inv := newTestInvoker(t, server.URL, defaultServiceConfig())

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "deleteUser"},
		model.InvocationInput{PathParams: map[string]string{"id": "1"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusNoContent {
		t.Errorf("StatusCode = %d, want 204", result.StatusCode)
	}
	if result.Body != nil {
		t.Errorf("Body = %v, want nil", result.Body)
	}
}

// --- Error cases ---

func TestOpenAPIOperationInvoker_Invoke_operationNotFound(t *testing.T) {
	inv := newTestInvoker(t, "http://localhost", defaultServiceConfig())

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "nonExistent"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
}

func TestOpenAPIOperationInvoker_Invoke_serviceNotConfigured(t *testing.T) {
	inv := newTestInvoker(t, "http://localhost", defaultServiceConfig())

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "unknown-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for unconfigured service")
	}
}

// --- Circuit breaker ---

func TestOpenAPIOperationInvoker_Invoke_circuitBreakerRejectsWhenOpen(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 2
	inv := newTestInvoker(t, server.URL, cfg)

	binding := model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"}

	// Trip the circuit breaker with server errors.
	for i := 0; i < 2; i++ {
		inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	}

	// Next call should be rejected by the circuit breaker without hitting the server.
	countBefore := callCount.Load()
	_, err := inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	if err == nil {
		t.Fatal("expected error when circuit breaker is open")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrBackendUnavailable {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrBackendUnavailable)
	}
	if callCount.Load() != countBefore {
		t.Error("server was called despite open circuit breaker")
	}
}

func TestOpenAPIOperationInvoker_Invoke_serverErrorsRecordFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 3
	inv := newTestInvoker(t, server.URL, cfg)

	binding := model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"}
	svc := inv.clients["test-svc"]

	// After 2 failures, still closed.
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	if s := svc.breaker.State(); s != BreakerClosed {
		t.Errorf("state after 2 failures = %v, want Closed", s)
	}

	// 3rd failure trips the breaker.
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	if s := svc.breaker.State(); s != BreakerOpen {
		t.Errorf("state after 3 failures = %v, want Open", s)
	}
}

func TestOpenAPIOperationInvoker_Invoke_clientErrorsDoNotTripBreaker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 2
	inv := newTestInvoker(t, server.URL, cfg)

	binding := model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"}
	svc := inv.clients["test-svc"]

	// Many 4xx errors should not trip the breaker.
	for i := 0; i < 5; i++ {
		inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	}
	if s := svc.breaker.State(); s != BreakerClosed {
		t.Errorf("state after 5 client errors = %v, want Closed", s)
	}
}

func TestOpenAPIOperationInvoker_Invoke_successResetsBreaker(t *testing.T) {
	var respondWithError atomic.Bool
	respondWithError.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if respondWithError.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 3
	inv := newTestInvoker(t, server.URL, cfg)

	binding := model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"}
	svc := inv.clients["test-svc"]

	// Record 2 failures.
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})

	// Switch to success, which resets failure count.
	respondWithError.Store(false)
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})

	// 2 more failures should NOT trip (count was reset).
	respondWithError.Store(true)
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})
	inv.Invoke(context.Background(), nil, binding, model.InvocationInput{})

	if s := svc.breaker.State(); s != BreakerClosed {
		t.Errorf("state = %v, want Closed (failures reset by success)", s)
	}
}

// --- Retry logic ---

func TestOpenAPIOperationInvoker_Invoke_retriesOnServerError(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"attempt": n})
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 10 // Don't trip during retries.
	cfg.Retry = config.RetryConfig{
		MaxAttempts:       3,
		BackoffInitial:    1 * time.Millisecond,
		BackoffMultiplier: 1,
		BackoffMax:        5 * time.Millisecond,
		IdempotentOnly:    true,
	}
	inv := newTestInvoker(t, server.URL, cfg)

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if callCount.Load() != 3 {
		t.Errorf("server called %d times, want 3", callCount.Load())
	}
}

func TestOpenAPIOperationInvoker_Invoke_noRetryPOSTWhenIdempotentOnly(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 10
	cfg.Retry = config.RetryConfig{
		MaxAttempts:       3,
		BackoffInitial:    1 * time.Millisecond,
		BackoffMultiplier: 1,
		BackoffMax:        5 * time.Millisecond,
		IdempotentOnly:    true,
	}
	inv := newTestInvoker(t, server.URL, cfg)

	// POST is not idempotent → should NOT retry.
	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "createUser"},
		model.InvocationInput{Body: map[string]any{"name": "Alice"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want 503", result.StatusCode)
	}
	if callCount.Load() != 1 {
		t.Errorf("server called %d times, want 1 (no retry for POST)", callCount.Load())
	}
}

func TestOpenAPIOperationInvoker_Invoke_retryPOSTWhenNotIdempotentOnly(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": "new-1"})
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 10
	cfg.Retry = config.RetryConfig{
		MaxAttempts:       3,
		BackoffInitial:    1 * time.Millisecond,
		BackoffMultiplier: 1,
		BackoffMax:        5 * time.Millisecond,
		IdempotentOnly:    false, // Allow retry for any method.
	}
	inv := newTestInvoker(t, server.URL, cfg)

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "createUser"},
		model.InvocationInput{Body: map[string]any{"name": "Alice"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want 201", result.StatusCode)
	}
	if callCount.Load() != 2 {
		t.Errorf("server called %d times, want 2", callCount.Load())
	}
}

func TestOpenAPIOperationInvoker_Invoke_retryExhaustedReturnsLastResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 10
	cfg.Retry = config.RetryConfig{
		MaxAttempts:       3,
		BackoffInitial:    1 * time.Millisecond,
		BackoffMultiplier: 1,
		BackoffMax:        5 * time.Millisecond,
		IdempotentOnly:    true,
	}
	inv := newTestInvoker(t, server.URL, cfg)

	result, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	// After exhausting retries, returns last result.
	if result.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d, want 502", result.StatusCode)
	}
}

// --- Context cancellation ---

func TestOpenAPIOperationInvoker_Invoke_contextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.Timeout = 5 * time.Second
	inv := newTestInvoker(t, server.URL, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := inv.Invoke(
		ctx,
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestOpenAPIOperationInvoker_Invoke_contextDeadlineExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.Timeout = 5 * time.Second
	inv := newTestInvoker(t, server.URL, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := inv.Invoke(
		ctx,
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for deadline exceeded")
	}
}

func TestOpenAPIOperationInvoker_Invoke_contextCancelDuringRetryBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := defaultServiceConfig()
	cfg.CircuitBreaker.FailureThreshold = 10
	cfg.Retry = config.RetryConfig{
		MaxAttempts:       5,
		BackoffInitial:    500 * time.Millisecond, // Long backoff.
		BackoffMultiplier: 1,
		BackoffMax:        1 * time.Second,
		IdempotentOnly:    false,
	}
	inv := newTestInvoker(t, server.URL, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := inv.Invoke(
		ctx,
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error when context expires during backoff")
	}
}

// --- Connection errors ---

func TestOpenAPIOperationInvoker_Invoke_connectionError(t *testing.T) {
	// Create a server and immediately close it to produce a connection error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	cfg := defaultServiceConfig()
	inv := newTestInvoker(t, serverURL, cfg)

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "openapi", ServiceID: "test-svc", OperationID: "listUsers"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrBackendUnavailable {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrBackendUnavailable)
	}
}

// --- Direct helper tests ---

func TestBuildRequestURL_basic(t *testing.T) {
	op := openapi.IndexedOperation{
		BaseURL:      "http://api.example.com",
		PathTemplate: "/items",
		Method:       "GET",
	}
	got := buildRequestURL(op, model.InvocationInput{})
	want := "http://api.example.com/items"
	if got != want {
		t.Errorf("buildRequestURL = %q, want %q", got, want)
	}
}

func TestBuildRequestURL_pathParams(t *testing.T) {
	op := openapi.IndexedOperation{
		BaseURL:      "http://api.example.com",
		PathTemplate: "/orgs/{orgId}/users/{userId}",
		Method:       "GET",
	}
	got := buildRequestURL(op, model.InvocationInput{
		PathParams: map[string]string{"orgId": "o1", "userId": "u2"},
	})
	want := "http://api.example.com/orgs/o1/users/u2"
	if got != want {
		t.Errorf("buildRequestURL = %q, want %q", got, want)
	}
}

func TestBuildRequestURL_queryParams(t *testing.T) {
	op := openapi.IndexedOperation{
		BaseURL:      "http://api.example.com",
		PathTemplate: "/search",
		Method:       "GET",
	}
	got := buildRequestURL(op, model.InvocationInput{
		QueryParams: map[string]string{"q": "hello"},
	})
	// url.Values.Encode() will produce q=hello
	want := "http://api.example.com/search?q=hello"
	if got != want {
		t.Errorf("buildRequestURL = %q, want %q", got, want)
	}
}

func TestBuildRequestHeaders_GETNoBody(t *testing.T) {
	h := buildRequestHeaders(nil, model.InvocationInput{}, http.MethodGet)
	if h.Get("Accept") != "application/json" {
		t.Errorf("Accept = %q, want application/json", h.Get("Accept"))
	}
	if h.Get("Content-Type") != "" {
		t.Errorf("Content-Type = %q, want empty for GET", h.Get("Content-Type"))
	}
}

func TestBuildRequestHeaders_POSTWithContentType(t *testing.T) {
	h := buildRequestHeaders(nil, model.InvocationInput{}, http.MethodPost)
	if h.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", h.Get("Content-Type"))
	}
}

func TestBuildRequestHeaders_PUTWithContentType(t *testing.T) {
	h := buildRequestHeaders(nil, model.InvocationInput{}, http.MethodPut)
	if h.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", h.Get("Content-Type"))
	}
}

func TestBuildRequestHeaders_PATCHWithContentType(t *testing.T) {
	h := buildRequestHeaders(nil, model.InvocationInput{}, http.MethodPatch)
	if h.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", h.Get("Content-Type"))
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"has\rnewline", "hasnewline"},
		{"has\nnewline", "hasnewline"},
		{"has\r\nboth", "hasboth"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeHeader(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeHeader(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsIdempotentMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{http.MethodGet, true},
		{http.MethodPut, true},
		{http.MethodDelete, true},
		{http.MethodHead, true},
		{http.MethodOptions, true},
		{http.MethodPost, false},
		{http.MethodPatch, false},
	}
	for _, tt := range tests {
		if got := isIdempotentMethod(tt.method); got != tt.want {
			t.Errorf("isIdempotentMethod(%q) = %v, want %v", tt.method, got, tt.want)
		}
	}
}

func TestIsRetryableStatus(t *testing.T) {
	retryable := []int{500, 502, 503, 504}
	for _, code := range retryable {
		if !isRetryableStatus(code) {
			t.Errorf("isRetryableStatus(%d) = false, want true", code)
		}
	}
	nonRetryable := []int{200, 201, 400, 401, 403, 404, 409, 501}
	for _, code := range nonRetryable {
		if isRetryableStatus(code) {
			t.Errorf("isRetryableStatus(%d) = true, want false", code)
		}
	}
}

func TestIsServerError(t *testing.T) {
	if !isServerError(500) {
		t.Error("isServerError(500) = false")
	}
	if !isServerError(503) {
		t.Error("isServerError(503) = false")
	}
	if isServerError(499) {
		t.Error("isServerError(499) = true")
	}
}

func TestIsClientError(t *testing.T) {
	if !isClientError(400) {
		t.Error("isClientError(400) = false")
	}
	if !isClientError(404) {
		t.Error("isClientError(404) = false")
	}
	if isClientError(500) {
		t.Error("isClientError(500) = true")
	}
	if isClientError(399) {
		t.Error("isClientError(399) = true")
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := config.RetryConfig{
		BackoffInitial:    100 * time.Millisecond,
		BackoffMultiplier: 2,
		BackoffMax:        2 * time.Second,
	}

	// attempt 1 → initial (100ms)
	if d := calculateBackoff(cfg, 1); d != 100*time.Millisecond {
		t.Errorf("backoff(1) = %v, want 100ms", d)
	}
	// attempt 2 → 200ms
	if d := calculateBackoff(cfg, 2); d != 200*time.Millisecond {
		t.Errorf("backoff(2) = %v, want 200ms", d)
	}
	// attempt 3 → 400ms
	if d := calculateBackoff(cfg, 3); d != 400*time.Millisecond {
		t.Errorf("backoff(3) = %v, want 400ms", d)
	}
}

func TestCalculateBackoff_defaults(t *testing.T) {
	cfg := config.RetryConfig{} // All zeros → defaults applied.
	d := calculateBackoff(cfg, 1)
	if d != 100*time.Millisecond {
		t.Errorf("backoff(1) with defaults = %v, want 100ms", d)
	}
}

func TestCalculateBackoff_cappedAtMax(t *testing.T) {
	cfg := config.RetryConfig{
		BackoffInitial:    100 * time.Millisecond,
		BackoffMultiplier: 10,
		BackoffMax:        500 * time.Millisecond,
	}
	d := calculateBackoff(cfg, 5)
	if d != 500*time.Millisecond {
		t.Errorf("backoff(5) = %v, want 500ms (capped)", d)
	}
}

func TestIsRetryableError(t *testing.T) {
	if isRetryableError(nil) {
		t.Error("isRetryableError(nil) = true")
	}
	// ErrorEnvelope (circuit breaker open) is not retryable.
	if isRetryableError(model.NewBackendUnavailableError()) {
		t.Error("isRetryableError(ErrorEnvelope) = true, want false")
	}
	// Generic errors are retryable.
	if !isRetryableError(context.DeadlineExceeded) {
		t.Error("isRetryableError(DeadlineExceeded) = false, want true")
	}
}

func TestExtractResponseHeaders(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"Content-Type":     {"application/json"},
			"X-Correlation-Id": {"c-1"},
			"X-Trace-Id":       {"t-1"},
			"X-Request-Id":     {"r-1"},
			"Retry-After":      {"60"},
			"X-Custom":         {"should-be-ignored"},
		},
	}
	headers := extractResponseHeaders(resp)
	if headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q", headers["Content-Type"])
	}
	if headers["X-Correlation-Id"] != "c-1" {
		t.Errorf("X-Correlation-Id = %q", headers["X-Correlation-Id"])
	}
	if headers["X-Trace-Id"] != "t-1" {
		t.Errorf("X-Trace-Id = %q", headers["X-Trace-Id"])
	}
	if headers["X-Request-Id"] != "r-1" {
		t.Errorf("X-Request-Id = %q", headers["X-Request-Id"])
	}
	if headers["Retry-After"] != "60" {
		t.Errorf("Retry-After = %q", headers["Retry-After"])
	}
	if _, exists := headers["X-Custom"]; exists {
		t.Error("X-Custom should not be extracted")
	}
}
