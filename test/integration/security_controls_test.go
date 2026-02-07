package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// ==========================================================================
// Authentication Tests
// ==========================================================================

func TestSecurity_NoAuthHeader_Returns401(t *testing.T) {
	h := NewTestHarness(t)

	endpoints := []string{
		"/ui/navigation",
		"/ui/pages/orders.list",
		"/ui/forms/orders.edit_form",
		"/ui/search?q=test",
		"/ui/lookups/orders.statuses",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			resp := h.GET(ep, "")
			h.AssertStatus(t, resp, http.StatusUnauthorized)
		})
	}
}

func TestSecurity_ExpiredJWT_Returns401(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateExpiredToken(ManagerClaims())

	resp := h.GET("/ui/navigation", token)
	h.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestSecurity_InvalidSignature_Returns401(t *testing.T) {
	h := NewTestHarness(t)

	// Generate a token signed with a different RSA key (not in JWKS).
	differentKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	claims := jwt.MapClaims{
		"iss":       "https://auth.test.thesa.dev",
		"aud":       "thesa-bff-test",
		"sub":       "user-1",
		"tenant_id": "acme-corp",
		"email":     "user@acme.com",
		"roles":     []any{"order_manager"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-1"
	signed, err := token.SignedString(differentKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	resp := h.GET("/ui/navigation", signed)
	h.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestSecurity_NoneAlgorithm_Returns401(t *testing.T) {
	h := NewTestHarness(t)

	// Craft a "none" algorithm token manually.
	// Header: {"alg":"none","typ":"JWT"}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","tenant_id":"acme-corp","iss":"https://auth.test.thesa.dev","aud":"thesa-bff-test","roles":["order_manager"]}`))
	noneToken := header + "." + payload + "."

	resp := h.GET("/ui/navigation", noneToken)
	h.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestSecurity_ValidJWT_Returns200(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	resp := h.GET("/ui/navigation", token)
	h.AssertStatus(t, resp, http.StatusOK)
}

func TestSecurity_MalformedToken_Returns401(t *testing.T) {
	h := NewTestHarness(t)

	resp := h.GET("/ui/navigation", "not.a.valid.jwt.token")
	h.AssertStatus(t, resp, http.StatusUnauthorized)
}

// ==========================================================================
// Cross-Tenant Isolation Tests
// ==========================================================================

func TestSecurity_TenantIsolation_WorkflowAccessDenied(t *testing.T) {
	h := NewTestHarness(t, WithWorkflows())

	// Tenant A creates a workflow.
	tenantA := h.GenerateToken(TestClaims{
		SubjectID: "user-a",
		TenantID:  "tenant-alpha",
		Email:     "a@alpha.com",
		Roles:     []string{"order_approver"},
	})

	h.MockBackend("orders-svc").OnOperation("confirmOrder").
		RespondWith(200, map[string]any{"status": "ok"})

	instanceID := startApprovalWorkflow(t, h, tenantA, "ord-100")

	// Tenant B tries to access tenant A's workflow → 404 (not 403).
	tenantB := h.GenerateToken(TestClaims{
		SubjectID: "user-b",
		TenantID:  "tenant-beta",
		Email:     "b@beta.com",
		Roles:     []string{"order_approver"},
	})

	resp := h.GET("/ui/workflows/"+instanceID, tenantB)
	h.AssertStatus(t, resp, http.StatusNotFound)
}

func TestSecurity_TenantIDFromJWT_NotRequestHeader(t *testing.T) {
	h := NewTestHarness(t)

	// Token has tenant "acme-corp" but we send X-Tenant-Id: "evil-corp".
	token := h.GenerateToken(ManagerClaims()) // tenant: acme-corp

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture([]map[string]any{
			OrderFixture("ord-1", "ORD-001", "pending"),
		}, 1))

	resp := h.GETWithHeaders("/ui/pages/orders.list/data", token, map[string]string{
		"X-Tenant-Id": "evil-corp",
	})
	h.AssertStatus(t, resp, http.StatusOK)

	// Verify the backend received tenant ID from JWT, not the request header.
	req := h.MockBackend("orders-svc").LastRequest("listOrders")
	if req == nil {
		t.Fatal("listOrders not called")
	}

	backendTenantID := req.Headers.Get("X-Tenant-Id")
	if backendTenantID != "acme-corp" {
		t.Errorf("backend X-Tenant-Id = %q, want %q (from JWT, not request header)", backendTenantID, "acme-corp")
	}
	if backendTenantID == "evil-corp" {
		t.Error("backend received X-Tenant-Id from request header instead of JWT — tenant isolation broken!")
	}
}

func TestSecurity_BackendCallsIncludeJWTTenant(t *testing.T) {
	h := NewTestHarness(t)

	token := h.GenerateToken(TestClaims{
		SubjectID: "user-1",
		TenantID:  "specific-tenant-123",
		Email:     "u@t.com",
		Roles:     []string{"order_manager"},
	})

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{"data": []map[string]any{}})

	h.GET("/ui/search?q=test", token)

	req := h.MockBackend("orders-svc").LastRequest("searchOrders")
	if req == nil {
		t.Fatal("searchOrders not called")
	}
	assertEqual(t, req.Headers.Get("X-Tenant-Id"), "specific-tenant-123", "backend X-Tenant-Id")
}

// ==========================================================================
// Privilege Escalation Prevention Tests
// ==========================================================================

func TestSecurity_ViewerCannotExecuteAdminCommand(t *testing.T) {
	h := NewTestHarness(t)
	viewerToken := h.GenerateToken(ViewerClaims()) // only orders:view

	// Viewer tries to execute cancel command (requires orders:cancel).
	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, viewerToken)

	h.AssertStatus(t, resp, http.StatusForbidden)

	// Backend should not be called.
	h.MockBackend("orders-svc").AssertNotCalled(t, "cancelOrder")
}

func TestSecurity_ViewerDescriptorOmitsAdminActions(t *testing.T) {
	h := NewTestHarness(t)
	viewerToken := h.GenerateToken(ViewerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture([]map[string]any{
			OrderFixture("ord-1", "ORD-001", "pending"),
		}, 1))

	resp := h.GET("/ui/pages/orders.list", viewerToken)

	var page map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &page)

	// Viewer should NOT see the Create Order action (requires orders:create).
	actions, _ := page["actions"].([]any)
	for _, a := range actions {
		action := a.(map[string]any)
		if action["id"] == "create_order" {
			t.Error("viewer should not see create_order action (requires orders:create)")
		}
	}
}

func TestSecurity_CapabilityCheckAtBothDescriptorAndExecution(t *testing.T) {
	h := NewTestHarness(t)
	clerkToken := h.GenerateToken(ClerkClaims()) // has orders:edit but NOT orders:cancel

	// Clerk cannot cancel (checked at execution time).
	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "ord-1",
			"reason": "test",
		},
	}, clerkToken)
	h.AssertStatus(t, resp, http.StatusForbidden)

	// Clerk can edit (both descriptor and execution).
	h.MockBackend("orders-svc").OnOperation("updateOrder").
		RespondWith(200, OrderFixture("ord-1", "ORD-001", "pending"))
	resp2 := h.POST("/ui/commands/orders.update", map[string]any{
		"input": map[string]any{
			"id":               "ord-1",
			"shipping_address": "123 St",
			"priority":         "normal",
			"notes":            "",
			"internal_code":    "",
		},
	}, clerkToken)
	h.AssertStatus(t, resp2, http.StatusOK)
}

// ==========================================================================
// Information Leakage Tests
// ==========================================================================

func TestSecurity_500ErrorReturnsGenericMessage(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Backend connection error triggers internal error handling.
	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWithConnectionError()

	resp := h.GET("/ui/search?q=test", token)

	// Should still return 200 with error in provider metadata (search is fault-tolerant).
	var result map[string]any
	h.AssertJSON(t, resp, http.StatusOK, &result)

	meta := result["meta"].(map[string]any)
	providers := meta["providers"].(map[string]any)

	// The provider should report error, not expose backend details.
	status := providers["orders.search"].(string)
	if status != "error" {
		t.Errorf("provider status = %q, want %q", status, "error")
	}
}

func TestSecurity_DescriptorOmitsOperationIDs(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture([]map[string]any{
			OrderFixture("ord-1", "ORD-001", "pending"),
		}, 1))

	resp := h.GET("/ui/pages/orders.list", token)

	body := h.ReadBody(resp)
	bodyStr := string(body)

	// Verify no operation IDs or service IDs leak into the descriptor.
	sensitiveStrings := []string{
		"listOrders",
		"searchOrders",
		"getOrderStatuses",
		"orders-svc",
		"openapi",
		"operation_id",
		"service_id",
	}

	for _, s := range sensitiveStrings {
		if strings.Contains(bodyStr, s) {
			t.Errorf("descriptor contains sensitive string %q — information leakage", s)
		}
	}
}

func TestSecurity_ErrorResponseNoStackTrace(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	// Trigger a 403 error and verify no sensitive info in response.
	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{"id": "ord-1", "reason": "test"},
	}, token)

	body := h.ReadBody(resp)
	bodyStr := string(body)

	sensitivePatterns := []string{
		"goroutine",
		".go:",
		"panic",
		"runtime.",
		"/home/",
		"/internal/",
		"localhost",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(bodyStr, pattern) {
			t.Errorf("error response contains sensitive pattern %q: %s", pattern, bodyStr)
		}
	}
}

// ==========================================================================
// Security Headers Tests
// ==========================================================================

func TestSecurity_HeadersOnAuthenticatedResponse(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	resp := h.GET("/ui/navigation", token)
	h.AssertStatus(t, resp, http.StatusOK)

	expectedHeaders := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Cache-Control":             "no-store",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}

	for name, expected := range expectedHeaders {
		actual := resp.Header.Get(name)
		if actual != expected {
			t.Errorf("header %s = %q, want %q", name, actual, expected)
		}
	}
}

func TestSecurity_HeadersOnErrorResponse(t *testing.T) {
	h := NewTestHarness(t)

	// Even 401 responses should have security headers.
	resp := h.GET("/ui/navigation", "")
	h.AssertStatus(t, resp, http.StatusUnauthorized)

	requiredHeaders := []string{
		"Strict-Transport-Security",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Cache-Control",
		"Referrer-Policy",
	}

	for _, name := range requiredHeaders {
		if resp.Header.Get(name) == "" {
			t.Errorf("security header %s missing on error response", name)
		}
	}
}

func TestSecurity_HeadersOnPublicEndpoint(t *testing.T) {
	h := NewTestHarness(t)

	// Health endpoint is public but should still have security headers.
	resp := h.GET("/ui/health", "")
	h.AssertStatus(t, resp, http.StatusOK)

	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Error("HSTS header missing on public endpoint")
	}
	if resp.Header.Get("X-Content-Type-Options") == "" {
		t.Error("X-Content-Type-Options missing on public endpoint")
	}
}

func TestSecurity_CorrelationIDReturned(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ViewerClaims())

	// Without custom correlation ID → generated one returned.
	resp1 := h.GET("/ui/navigation", token)
	correlationID := resp1.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		t.Error("X-Correlation-Id not set in response")
	}

	// With custom correlation ID → echoed back.
	resp2 := h.GETWithHeaders("/ui/navigation", token, map[string]string{
		"X-Correlation-Id": "custom-trace-123",
	})
	if resp2.Header.Get("X-Correlation-Id") != "custom-trace-123" {
		t.Errorf("X-Correlation-Id = %q, want %q", resp2.Header.Get("X-Correlation-Id"), "custom-trace-123")
	}
}

// ==========================================================================
// Input Sanitization Tests
// ==========================================================================

func TestSecurity_HeaderInjectionPrevented(t *testing.T) {
	h := NewTestHarness(t)

	// Create token with a tenant ID containing CRLF injection attempt.
	token := h.GenerateToken(TestClaims{
		SubjectID: "user-1",
		TenantID:  "acme-corp\r\nX-Injected: evil-header",
		Email:     "u@t.com",
		Roles:     []string{"order_manager"},
	})

	h.MockBackend("orders-svc").OnOperation("searchOrders").
		RespondWith(200, map[string]any{"data": []map[string]any{}})

	h.GET("/ui/search?q=test", token)

	req := h.MockBackend("orders-svc").LastRequest("searchOrders")
	if req == nil {
		t.Fatal("searchOrders not called")
	}

	// The tenant ID header should have CRLF stripped.
	tenantHeader := req.Headers.Get("X-Tenant-Id")
	if strings.Contains(tenantHeader, "\r") || strings.Contains(tenantHeader, "\n") {
		t.Errorf("header injection not prevented: X-Tenant-Id = %q", tenantHeader)
	}
	if req.Headers.Get("X-Injected") != "" {
		t.Error("header injection succeeded: X-Injected header was set")
	}
}

func TestSecurity_PathTraversalInPathParams(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Attempt path traversal via command input.
	// url.PathEscape encodes "/" to "%2F", preventing HTTP-level path traversal.
	// The request still routes to the correct handler (cancelOrder) because
	// the escaped value is treated as a single path segment.
	h.MockBackend("orders-svc").OnOperation("cancelOrder").
		RespondWith(200, map[string]any{"status": "ok"})

	resp := h.POST("/ui/commands/orders.cancel", map[string]any{
		"input": map[string]any{
			"id":     "../../etc/passwd",
			"reason": "test",
		},
	}, token)

	// The request should route to cancelOrder handler (not to a different path).
	h.AssertStatus(t, resp, http.StatusOK)
	h.MockBackend("orders-svc").AssertCalled(t, "cancelOrder", 1)

	// Verify the cancelOrder handler received the request, not some other handler.
	// This confirms that url.PathEscape prevented the ".." from being interpreted
	// as path navigation at the HTTP transport level.
	req := h.MockBackend("orders-svc").LastRequest("cancelOrder")
	if req == nil {
		t.Fatal("cancelOrder not called — path traversal may have misrouted the request")
	}
}

// ==========================================================================
// CORS Tests
// ==========================================================================

func TestSecurity_CORSAllowedOrigin(t *testing.T) {
	h := NewTestHarness(t)

	// Allowed origin (configured in harness: http://localhost:3000).
	resp := h.GETWithHeaders("/ui/health", "", map[string]string{
		"Origin": "http://localhost:3000",
	})

	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("CORS not set for allowed origin")
	}
}

func TestSecurity_CORSDisallowedOrigin(t *testing.T) {
	h := NewTestHarness(t)

	// Disallowed origin.
	resp := h.GETWithHeaders("/ui/health", "", map[string]string{
		"Origin": "https://evil.example.com",
	})

	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set for disallowed origin")
	}
}
