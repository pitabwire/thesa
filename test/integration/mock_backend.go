package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// MockBackend is a configurable HTTP test server that simulates a backend service.
// It allows configuring per-operation responses and records all received requests
// for later assertion.
type MockBackend struct {
	t         *testing.T
	serviceID string
	server    *httptest.Server

	mu             sync.RWMutex
	operations     map[string]*operationConfig
	receivedByOp   map[string][]*RecordedRequest
	defaultHandler http.HandlerFunc
}

// RecordedRequest captures the details of a request received by the mock backend.
type RecordedRequest struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     http.Header
	Body        map[string]any
	RawBody     []byte
	ReceivedAt  time.Time
}

// operationConfig holds the configured response for a single operation.
type operationConfig struct {
	mu        sync.Mutex
	responses []*mockResponse
	current   int
}

type mockResponse struct {
	status     int
	body       any
	delay      time.Duration
	connError  bool
	headerFunc func(http.Header)
}

// OperationMock is a builder for configuring mock responses for a specific operation.
type OperationMock struct {
	backend *MockBackend
	opID    string
}

// newMockBackend creates a new mock backend and starts the HTTP test server.
func newMockBackend(t *testing.T, serviceID string, operationPaths map[string]operationRoute) *MockBackend {
	t.Helper()

	mb := &MockBackend{
		t:            t,
		serviceID:    serviceID,
		operations:   make(map[string]*operationConfig),
		receivedByOp: make(map[string][]*RecordedRequest),
	}

	mux := http.NewServeMux()
	for opID, route := range operationPaths {
		pattern := route.method + " " + route.pathPattern
		mux.HandleFunc(pattern, mb.handleOperation(opID))
	}

	// Fallback for unregistered paths.
	mb.defaultHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("mock: no operation registered for %s %s", r.Method, r.URL.Path),
		})
	}

	mb.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try mux first; if no match, use default handler.
		// We wrap because ServeMux returns 404 for unmatched patterns.
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(mb.server.Close)

	return mb
}

// operationRoute maps an operation ID to its HTTP method and path pattern.
type operationRoute struct {
	method      string
	pathPattern string
}

// URL returns the base URL of the mock backend server.
func (mb *MockBackend) URL() string {
	return mb.server.URL
}

// OnOperation returns a builder for configuring responses for the named operation.
func (mb *MockBackend) OnOperation(operationID string) *OperationMock {
	return &OperationMock{
		backend: mb,
		opID:    operationID,
	}
}

// RespondWith configures the operation to respond with the given status and body.
func (om *OperationMock) RespondWith(status int, body any) *OperationMock {
	om.backend.addResponse(om.opID, &mockResponse{
		status: status,
		body:   body,
	})
	return om
}

// RespondWithError configures the operation to respond with an error envelope.
func (om *OperationMock) RespondWithError(status int, code, message string) *OperationMock {
	om.backend.addResponse(om.opID, &mockResponse{
		status: status,
		body: map[string]any{
			"code":    code,
			"message": message,
		},
	})
	return om
}

// RespondWithDelay configures a delayed response to simulate slow backends.
func (om *OperationMock) RespondWithDelay(delay time.Duration, status int, body any) *OperationMock {
	om.backend.addResponse(om.opID, &mockResponse{
		status: status,
		body:   body,
		delay:  delay,
	})
	return om
}

// RespondWithConnectionError configures the operation to close the connection
// to simulate a backend failure.
func (om *OperationMock) RespondWithConnectionError() *OperationMock {
	om.backend.addResponse(om.opID, &mockResponse{
		connError: true,
	})
	return om
}

// RespondWithHeaders configures additional response headers.
func (om *OperationMock) RespondWithHeaders(status int, body any, headerFunc func(http.Header)) *OperationMock {
	om.backend.addResponse(om.opID, &mockResponse{
		status:     status,
		body:       body,
		headerFunc: headerFunc,
	})
	return om
}

func (mb *MockBackend) addResponse(opID string, resp *mockResponse) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	cfg, ok := mb.operations[opID]
	if !ok {
		cfg = &operationConfig{}
		mb.operations[opID] = cfg
	}
	cfg.responses = append(cfg.responses, resp)
}

func (mb *MockBackend) handleOperation(opID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Record the request.
		rec := &RecordedRequest{
			Method:      r.Method,
			Path:        r.URL.Path,
			QueryParams: make(map[string]string),
			Headers:     r.Header.Clone(),
			ReceivedAt:  time.Now(),
		}
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				rec.QueryParams[key] = values[0]
			}
		}
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			rec.RawBody = body
			if len(body) > 0 {
				var parsed map[string]any
				if err := json.Unmarshal(body, &parsed); err == nil {
					rec.Body = parsed
				}
			}
		}

		mb.mu.Lock()
		mb.receivedByOp[opID] = append(mb.receivedByOp[opID], rec)
		mb.mu.Unlock()

		// Get configured response.
		resp := mb.getNextResponse(opID)
		if resp == nil {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		if resp.connError {
			// Hijack the connection and close it to simulate a connection error.
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				if conn != nil {
					conn.Close()
				}
			}
			return
		}

		if resp.delay > 0 {
			time.Sleep(resp.delay)
		}

		if resp.headerFunc != nil {
			resp.headerFunc(w.Header())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.status)
		if resp.body != nil {
			json.NewEncoder(w).Encode(resp.body)
		}
	}
}

func (mb *MockBackend) getNextResponse(opID string) *mockResponse {
	mb.mu.RLock()
	cfg, ok := mb.operations[opID]
	mb.mu.RUnlock()
	if !ok || cfg == nil {
		return nil
	}

	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	if len(cfg.responses) == 0 {
		return nil
	}

	idx := cfg.current
	if idx >= len(cfg.responses) {
		// Repeat the last response for subsequent calls.
		idx = len(cfg.responses) - 1
	} else {
		cfg.current++
	}
	return cfg.responses[idx]
}

// AssertCalled verifies that the operation was called the expected number of times.
func (mb *MockBackend) AssertCalled(t *testing.T, operationID string, expectedCount int) {
	t.Helper()
	mb.mu.RLock()
	actual := len(mb.receivedByOp[operationID])
	mb.mu.RUnlock()
	if actual != expectedCount {
		t.Errorf("mock %s: operation %q called %d times, want %d", mb.serviceID, operationID, actual, expectedCount)
	}
}

// AssertNotCalled verifies that the operation was never called.
func (mb *MockBackend) AssertNotCalled(t *testing.T, operationID string) {
	t.Helper()
	mb.AssertCalled(t, operationID, 0)
}

// LastRequest returns the last request received for the given operation.
// Returns nil if no requests were recorded.
func (mb *MockBackend) LastRequest(operationID string) *RecordedRequest {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	reqs := mb.receivedByOp[operationID]
	if len(reqs) == 0 {
		return nil
	}
	return reqs[len(reqs)-1]
}

// AllRequests returns all requests received for the given operation.
func (mb *MockBackend) AllRequests(operationID string) []*RecordedRequest {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	reqs := mb.receivedByOp[operationID]
	copied := make([]*RecordedRequest, len(reqs))
	copy(copied, reqs)
	return copied
}

// Reset clears all recorded requests and configured responses for the backend.
func (mb *MockBackend) Reset() {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.operations = make(map[string]*operationConfig)
	mb.receivedByOp = make(map[string][]*RecordedRequest)
}

// ResetOperation clears recorded requests and configured responses for one operation.
func (mb *MockBackend) ResetOperation(operationID string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	delete(mb.operations, operationID)
	delete(mb.receivedByOp, operationID)
}

// DefaultOrdersRoutes returns the operation routes for the orders-svc test OpenAPI spec.
// This avoids importing kin-openapi in the test package, keeping the harness lightweight.
func DefaultOrdersRoutes() map[string]operationRoute {
	return map[string]operationRoute{
		"listOrders":       {method: "GET", pathPattern: "/api/orders"},
		"searchOrders":     {method: "GET", pathPattern: "/api/orders/search"},
		"getOrderStatuses": {method: "GET", pathPattern: "/api/orders/statuses"},
		"getOrderCounts":   {method: "GET", pathPattern: "/api/orders/counts"},
		"getOrder":         {method: "GET", pathPattern: "/api/orders/{id}"},
		"updateOrder":      {method: "PATCH", pathPattern: "/api/orders/{id}"},
		"cancelOrder":      {method: "POST", pathPattern: "/api/orders/{id}/cancel"},
		"confirmOrder":     {method: "POST", pathPattern: "/api/orders/{id}/confirm"},
	}
}

// parseOperationRoutesFromSpec builds operation routes from the known test spec.
func parseOperationRoutesFromSpec(specPath string) map[string]operationRoute {
	_ = specPath
	return DefaultOrdersRoutes()
}
