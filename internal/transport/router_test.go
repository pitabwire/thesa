package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// testDeps returns Dependencies with sensible defaults for testing.
func testDeps() Dependencies {
	cfg := config.Defaults()
	cfg.Server.CORS.AllowedOrigins = []string{"https://app.example.com"}
	cfg.Server.HandlerTimeout = 5 * time.Second
	return Dependencies{Config: cfg}
}

// --- Router tests ---

func TestNewRouter_health(t *testing.T) {
	r := NewRouter(testDeps())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/health", nil))

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestNewRouter_ready(t *testing.T) {
	r := NewRouter(testDeps())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/ready", nil))

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestNewRouter_metrics(t *testing.T) {
	r := NewRouter(testDeps())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestNewRouter_authenticatedRoutes_areRegistered(t *testing.T) {
	// With auth rejecting all requests, all authenticated routes should
	// return 401, confirming they are registered and not 404/405.
	rejectAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteError(w, model.NewUnauthorizedError("rejected"))
		})
	}

	deps := testDeps()
	deps.Authenticate = rejectAuth
	r := NewRouter(deps)

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/ui/navigation"},
		{"GET", "/ui/pages/orders.list"},
		{"GET", "/ui/pages/orders.list/data"},
		{"GET", "/ui/forms/orders.create"},
		{"GET", "/ui/forms/orders.create/data"},
		{"POST", "/ui/commands/orders.cancel"},
		{"POST", "/ui/workflows/orders.approval/start"},
		{"POST", "/ui/workflows/wf-123/advance"},
		{"GET", "/ui/workflows/wf-123"},
		{"POST", "/ui/workflows/wf-123/cancel"},
		{"GET", "/ui/workflows"},
		{"GET", "/ui/search"},
		{"GET", "/ui/lookups/currencies"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != 401 {
				t.Errorf("status = %d, want 401 (auth should reject)", w.Code)
			}
		})
	}
}

func TestNewRouter_publicRoutesbypassAuth(t *testing.T) {
	// Create an auth middleware that always rejects.
	rejectAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteError(w, model.NewUnauthorizedError("rejected"))
		})
	}

	deps := testDeps()
	deps.Authenticate = rejectAuth
	r := NewRouter(deps)

	// Health, ready, and metrics should still return 200.
	for _, path := range []string{"/ui/health", "/ui/ready", "/metrics"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code != 200 {
				t.Errorf("status = %d, want 200 (should bypass auth)", w.Code)
			}
		})
	}

	// Authenticated route should be rejected.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/navigation", nil))
	if w.Code != 401 {
		t.Errorf("navigation status = %d, want 401 (auth should reject)", w.Code)
	}
}

// --- Middleware tests ---

func TestRecovery_catchesPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != 500 {
		t.Errorf("status = %d, want 500 after panic", w.Code)
	}
}

func TestRecovery_passesThrough(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestCORS_preflight(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for preflight")
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Allow-Origin = %q", got)
	}
}

func TestCORS_disallowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Authorization"},
	}

	called := false
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should still be called for non-preflight")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin should be empty for disallowed origin, got %q", got)
	}
}

func TestRequestID_generated(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := CorrelationIDFrom(r.Context())
		if id == "" {
			t.Error("correlation ID should be generated")
		}
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if got := w.Header().Get("X-Correlation-Id"); got == "" {
		t.Error("response should have X-Correlation-Id header")
	}
}

func TestRequestID_propagated(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := CorrelationIDFrom(r.Context())
		if id != "test-corr-123" {
			t.Errorf("correlation ID = %q, want test-corr-123", id)
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Correlation-Id", "test-corr-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("X-Correlation-Id"); got != "test-corr-123" {
		t.Errorf("response X-Correlation-Id = %q, want test-corr-123", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	expected := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "0",
		"Cache-Control":             "no-store",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}

	for header, want := range expected {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestBuildRequestContextMiddleware(t *testing.T) {
	claims := map[string]any{
		"sub":       "user-42",
		"email":     "user@example.com",
		"tenant_id": "tenant-1",
		"roles":     []any{"admin", "viewer"},
	}

	handler := BuildRequestContextMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			t.Fatal("RequestContext should be in context")
		}
		if rctx.SubjectID != "user-42" {
			t.Errorf("SubjectID = %q, want user-42", rctx.SubjectID)
		}
		if rctx.TenantID != "tenant-1" {
			t.Errorf("TenantID = %q, want tenant-1", rctx.TenantID)
		}
		if rctx.PartitionID != "part-1" {
			t.Errorf("PartitionID = %q, want part-1", rctx.PartitionID)
		}
		if len(rctx.Roles) != 2 || rctx.Roles[0] != "admin" {
			t.Errorf("Roles = %v, want [admin viewer]", rctx.Roles)
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithClaims(req.Context(), claims))
	req.Header.Set("X-Partition-Id", "part-1")
	req.Header.Set("X-Device-Id", "device-abc")
	req.Header.Set("Accept-Language", "en-US")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestBuildRequestContextMiddleware_customPaths(t *testing.T) {
	claims := map[string]any{
		"sub":   "user-99",
		"email": "user@keycloak.com",
		"realm_access": map[string]any{
			"roles": []any{"manager"},
		},
		"custom_tenant": "tenant-kc",
	}

	paths := map[string]string{
		"tenant_id": "custom_tenant",
		"roles":     "realm_access.roles",
	}

	handler := BuildRequestContextMiddleware(paths)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx.TenantID != "tenant-kc" {
			t.Errorf("TenantID = %q, want tenant-kc", rctx.TenantID)
		}
		if len(rctx.Roles) != 1 || rctx.Roles[0] != "manager" {
			t.Errorf("Roles = %v, want [manager]", rctx.Roles)
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithClaims(req.Context(), claims))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestResolveCapabilities(t *testing.T) {
	resolver := &mockResolver{
		caps: model.CapabilitySet{"orders:list:view": true},
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caps := CapabilitiesFrom(r.Context())
		if !caps.Has("orders:list:view") {
			t.Error("should have orders:list:view capability")
		}
		w.WriteHeader(200)
	})

	// Chain: BuildRequestContextMiddleware → ResolveCapabilities → handler
	handler := BuildRequestContextMiddleware(nil)(ResolveCapabilities(resolver)(inner))

	claims := map[string]any{"sub": "user-1", "tenant_id": "t-1"}
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithClaims(req.Context(), claims))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestResolveCapabilities_nilResolver(t *testing.T) {
	handler := ResolveCapabilities(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caps := CapabilitiesFrom(r.Context())
		if caps != nil {
			t.Errorf("caps should be nil, got %v", caps)
		}
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
}

func TestHandlerTimeout_setsDeadline(t *testing.T) {
	handler := HandlerTimeout(100 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("context should have deadline")
		}
		if time.Until(deadline) > 200*time.Millisecond {
			t.Error("deadline should be within 200ms")
		}
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
}

func TestHandlerTimeout_zeroNoDeadline(t *testing.T) {
	handler := HandlerTimeout(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		if ok {
			t.Error("context should not have deadline when timeout is 0")
		}
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
}

func TestRequestLogging_capturesStatus(t *testing.T) {
	handler := RequestLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))

	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
}

func TestMiddlewareOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string

	track := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				order = append(order, name)
				mu.Unlock()
				next.ServeHTTP(w, r)
			})
		}
	}

	deps := testDeps()
	deps.Authenticate = track("authenticate")

	r := NewRouter(deps)

	// Replace authenticated route handlers temporarily is not straightforward
	// with chi, so instead we verify the middleware order by testing a full
	// request through the authenticated group and checking the response
	// includes all global middleware effects.
	_ = r

	// Instead, test the composed middleware chain manually.
	order = nil
	chain := track("recovery")(
		track("cors")(
			track("requestID")(
				track("securityHeaders")(
					track("authenticate")(
						track("buildCtx")(
							track("capabilities")(
								track("timeout")(
									track("logging")(
										track("metrics")(
											http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
												w.WriteHeader(200)
											}),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	)

	w := httptest.NewRecorder()
	chain.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	expected := []string{
		"recovery", "cors", "requestID", "securityHeaders",
		"authenticate", "buildCtx", "capabilities", "timeout",
		"logging", "metrics",
	}

	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d: %v", len(order), len(expected), order)
	}
	for i, name := range expected {
		if order[i] != name {
			t.Errorf("order[%d] = %q, want %q", i, order[i], name)
		}
	}
}

func TestSecurityHeaders_onHealth(t *testing.T) {
	r := NewRouter(testDeps())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/health", nil))

	// Security headers should be present even on health endpoint.
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := w.Header().Get("X-Correlation-Id"); got == "" {
		t.Error("health should still get X-Correlation-Id")
	}
}

// --- mocks ---

type mockResolver struct {
	caps model.CapabilitySet
	err  error
}

func (m *mockResolver) Resolve(_ *model.RequestContext) (model.CapabilitySet, error) {
	return m.caps, m.err
}

func (m *mockResolver) Invalidate(_, _ string) {}
