// Package integration provides a reusable test harness for end-to-end
// integration testing of the Thesa BFF server. It starts a full HTTP server
// with mock backend services, in-memory stores, and a test JWT issuer.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/capability"
	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/internal/transport"
	"github.com/pitabwire/thesa/internal/workflow"
	"github.com/pitabwire/thesa/model"
)

// TestHarness encapsulates a fully wired BFF instance with mock backends
// for integration testing.
type TestHarness struct {
	t      *testing.T
	server *httptest.Server
	issuer *tokenIssuer

	// Internal components exposed for advanced test scenarios.
	Registry        *definition.Registry
	OAIndex         *openapi.Index
	InvokerRegistry *invoker.Registry
	WorkflowStore   *workflow.MemoryWorkflowStore
	WorkflowEngine  *workflow.Engine
	IdempotencyStore *command.MemoryIdempotencyStore
	CommandExecutor *command.CommandExecutor
	CapResolver     model.CapabilityResolver

	backends map[string]*MockBackend
	cfg      *config.Config
}

// HarnessOption configures the test harness.
type HarnessOption func(*harnessConfig)

type harnessConfig struct {
	definitionDirs []string
	specSources    []specSourceConfig
	policyFile     string
	workflowEnabled bool
	idempotencyEnabled bool
	handlerTimeout time.Duration
	sdkHandlers    map[string]invoker.SDKHandler
}

type specSourceConfig struct {
	serviceID string
	specFile  string
}

// WithDefinitions sets the definition directories to load. Relative paths are
// resolved from the testdata directory.
func WithDefinitions(dirs ...string) HarnessOption {
	return func(c *harnessConfig) {
		c.definitionDirs = dirs
	}
}

// WithSpec adds an OpenAPI spec source to load.
func WithSpec(serviceID, specFile string) HarnessOption {
	return func(c *harnessConfig) {
		c.specSources = append(c.specSources, specSourceConfig{
			serviceID: serviceID,
			specFile:  specFile,
		})
	}
}

// WithPolicyFile sets the static policy YAML file for capability resolution.
func WithPolicyFile(path string) HarnessOption {
	return func(c *harnessConfig) {
		c.policyFile = path
	}
}

// WithWorkflows enables the workflow engine with an in-memory store.
func WithWorkflows() HarnessOption {
	return func(c *harnessConfig) {
		c.workflowEnabled = true
	}
}

// WithIdempotency enables idempotency checking with an in-memory store.
func WithIdempotency() HarnessOption {
	return func(c *harnessConfig) {
		c.idempotencyEnabled = true
	}
}

// WithHandlerTimeout sets the per-request handler timeout.
func WithHandlerTimeout(d time.Duration) HarnessOption {
	return func(c *harnessConfig) {
		c.handlerTimeout = d
	}
}

// WithSDKHandler registers an SDK handler for workflow system steps.
func WithSDKHandler(name string, handler invoker.SDKHandler) HarnessOption {
	return func(c *harnessConfig) {
		if c.sdkHandlers == nil {
			c.sdkHandlers = make(map[string]invoker.SDKHandler)
		}
		c.sdkHandlers[name] = handler
	}
}

// NewTestHarness creates and starts a full BFF test instance. The server is
// automatically cleaned up when the test completes.
func NewTestHarness(t *testing.T, opts ...HarnessOption) *TestHarness {
	t.Helper()

	hc := &harnessConfig{
		handlerTimeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(hc)
	}

	testdataDir := testdataDir()

	// Defaults: use testdata fixtures if nothing specified.
	if len(hc.definitionDirs) == 0 {
		hc.definitionDirs = []string{filepath.Join(testdataDir, "definitions")}
	}
	if len(hc.specSources) == 0 {
		hc.specSources = []specSourceConfig{
			{serviceID: "orders-svc", specFile: filepath.Join(testdataDir, "specs", "orders-svc.yaml")},
		}
	}
	if hc.policyFile == "" {
		hc.policyFile = filepath.Join(testdataDir, "policies.yaml")
	}

	h := &TestHarness{
		t:        t,
		backends: make(map[string]*MockBackend),
	}

	// Step 1: Create mock backends and get their URLs.
	// We create the backend first so we can rewrite spec files with the real URLs.
	for _, src := range hc.specSources {
		routes := parseOperationRoutesFromSpec(src.specFile)
		mb := newMockBackend(t, src.serviceID, routes)
		h.backends[src.serviceID] = mb
	}

	// Step 2: Create temporary spec files with mock backend URLs.
	tmpSpecDir := t.TempDir()
	specSources := make([]openapi.SpecSource, len(hc.specSources))
	for i, src := range hc.specSources {
		mb := h.backends[src.serviceID]

		// Read spec template and replace placeholder URL.
		data, err := os.ReadFile(src.specFile)
		if err != nil {
			t.Fatalf("read spec %s: %v", src.specFile, err)
		}
		specContent := strings.ReplaceAll(string(data), "{{ORDERS_SVC_URL}}", mb.URL())
		// Also handle the original URL for example specs.
		specContent = strings.ReplaceAll(specContent, "https://orders.internal", mb.URL())

		tmpPath := filepath.Join(tmpSpecDir, filepath.Base(src.specFile))
		if err := os.WriteFile(tmpPath, []byte(specContent), 0644); err != nil {
			t.Fatalf("write temp spec: %v", err)
		}

		specSources[i] = openapi.SpecSource{
			ServiceID: src.serviceID,
			BaseURL:   mb.URL(),
			SpecPath:  tmpPath,
		}
	}

	// Step 3: Load OpenAPI index.
	h.OAIndex = openapi.NewIndex()
	if err := h.OAIndex.Load(specSources); err != nil {
		t.Fatalf("load OpenAPI specs: %v", err)
	}

	// Step 4: Load definitions.
	loader := definition.NewLoader()
	defs, err := loader.LoadAll(hc.definitionDirs)
	if err != nil {
		t.Fatalf("load definitions: %v", err)
	}
	h.Registry = definition.NewRegistry(defs)

	// Step 5: Build capability resolver.
	evaluator, err := capability.NewStaticPolicyEvaluator(hc.policyFile)
	if err != nil {
		t.Fatalf("load policy file: %v", err)
	}
	h.CapResolver = capability.NewResolver(evaluator, 0) // no caching in tests

	// Step 6: Build in-memory stores.
	h.WorkflowStore = workflow.NewMemoryWorkflowStore()
	h.IdempotencyStore = command.NewMemoryIdempotencyStore()

	// Step 7: Build invoker registry.
	sdkHandlers := invoker.NewSDKHandlerRegistry()
	for name, handler := range hc.sdkHandlers {
		sdkHandlers.Register(name, handler)
	}

	// Build service configs pointing to mock backends.
	serviceConfigs := make(map[string]config.ServiceConfig, len(h.backends))
	for svcID, mb := range h.backends {
		serviceConfigs[svcID] = config.ServiceConfig{
			BaseURL: mb.URL(),
			Timeout: 5 * time.Second,
			Retry: config.RetryConfig{
				MaxAttempts:    1,
				IdempotentOnly: true,
			},
		}
	}

	h.InvokerRegistry = invoker.NewRegistry()
	h.InvokerRegistry.Register(invoker.NewOpenAPIOperationInvoker(h.OAIndex, serviceConfigs))
	h.InvokerRegistry.Register(invoker.NewSDKOperationInvoker(sdkHandlers))

	// Step 8: Build providers.
	var cmdOpts []command.CommandExecutorOption
	if hc.idempotencyEnabled {
		cmdOpts = append(cmdOpts, command.WithIdempotencyStore(h.IdempotencyStore))
	}
	h.CommandExecutor = command.NewCommandExecutor(h.Registry, h.InvokerRegistry, h.OAIndex, cmdOpts...)

	if hc.workflowEnabled {
		h.WorkflowEngine = workflow.NewEngine(h.Registry, h.WorkflowStore, h.InvokerRegistry, h.CapResolver)
	}

	actionProvider := metadata.NewActionProvider()
	menuProvider := metadata.NewMenuProvider(h.Registry, h.InvokerRegistry)
	pageProvider := metadata.NewPageProvider(h.Registry, h.InvokerRegistry, actionProvider)
	formProvider := metadata.NewFormProvider(h.Registry, h.InvokerRegistry, actionProvider)
	searchProvider := search.NewSearchProvider(h.Registry, h.InvokerRegistry, 3*time.Second, 50)
	lookupProvider := search.NewLookupProvider(h.Registry, h.InvokerRegistry, 5*time.Minute, 1000)

	// Step 9: Create JWT issuer.
	h.issuer = newTokenIssuer(t)

	// Step 10: Build config.
	h.cfg = &config.Config{
		Server: config.ServerConfig{
			Port:           0, // unused, httptest picks a port
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			HandlerTimeout: hc.handlerTimeout,
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Authorization", "Content-Type", "X-Partition-Id",
					"X-Correlation-Id", "X-Idempotency-Key"},
				MaxAge: 86400,
			},
		},
		Identity: config.IdentityConfig{
			Issuer:     h.issuer.Issuer(),
			Audience:   h.issuer.Audience(),
			JWKSURL:    h.issuer.JWKSURL(),
			Algorithms: []string{"RS256"},
			ClaimPaths: map[string]string{
				"subject_id": "sub",
				"tenant_id":  "tenant_id",
				"email":      "email",
				"roles":      "roles",
			},
		},
	}

	// Step 11: Build router with full middleware chain.
	jwks := transport.NewJWKSClient(h.issuer.JWKSURL(), 1*time.Hour)

	router := transport.NewRouter(transport.Dependencies{
		Config:             h.cfg,
		Authenticate:       transport.JWTAuthenticator(h.cfg.Identity, jwks),
		CapabilityResolver: h.CapResolver,
		MenuProvider:       menuProvider,
		PageProvider:       pageProvider,
		FormProvider:       formProvider,
		CommandExecutor:    h.CommandExecutor,
		WorkflowEngine:     h.WorkflowEngine,
		SearchProvider:     searchProvider,
		LookupProvider:     lookupProvider,
		HealthHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		},
		ReadyHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ready"}`))
		},
	})

	// Step 12: Start test server.
	h.server = httptest.NewServer(router)
	t.Cleanup(func() {
		h.server.Close()
	})

	return h
}

// BaseURL returns the test server's base URL.
func (h *TestHarness) BaseURL() string {
	return h.server.URL
}

// MockBackend returns the mock backend for the given service ID.
// Panics if the service ID was not configured.
func (h *TestHarness) MockBackend(serviceID string) *MockBackend {
	mb, ok := h.backends[serviceID]
	if !ok {
		h.t.Fatalf("mock backend %q not configured", serviceID)
	}
	return mb
}

// GenerateToken creates a valid JWT token with the given claims.
func (h *TestHarness) GenerateToken(claims TestClaims) string {
	return h.issuer.GenerateToken(claims)
}

// GenerateExpiredToken creates a JWT that has already expired.
func (h *TestHarness) GenerateExpiredToken(claims TestClaims) string {
	return h.issuer.GenerateExpiredToken(claims)
}

// --- HTTP client helpers ---

// GET performs an authenticated GET request.
func (h *TestHarness) GET(path, token string) *http.Response {
	h.t.Helper()
	return h.doRequest("GET", path, nil, token, nil)
}

// GETWithHeaders performs an authenticated GET request with additional headers.
func (h *TestHarness) GETWithHeaders(path, token string, headers map[string]string) *http.Response {
	h.t.Helper()
	return h.doRequest("GET", path, nil, token, headers)
}

// POST performs an authenticated POST request with a JSON body.
func (h *TestHarness) POST(path string, body any, token string) *http.Response {
	h.t.Helper()
	return h.doRequest("POST", path, body, token, nil)
}

// POSTWithHeaders performs an authenticated POST request with additional headers.
func (h *TestHarness) POSTWithHeaders(path string, body any, token string, headers map[string]string) *http.Response {
	h.t.Helper()
	return h.doRequest("POST", path, body, token, headers)
}

func (h *TestHarness) doRequest(method, path string, body any, token string, headers map[string]string) *http.Response {
	h.t.Helper()

	url := h.server.URL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		h.t.Fatalf("create request: %v", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s failed: %v", method, path, err)
	}
	return resp
}

// ParseJSON reads the response body and unmarshals it into the target.
func (h *TestHarness) ParseJSON(resp *http.Response, target any) {
	h.t.Helper()
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatalf("read response body: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		h.t.Fatalf("unmarshal response body: %v\nbody: %s", err, string(data))
	}
}

// ReadBody reads and returns the response body as bytes.
func (h *TestHarness) ReadBody(resp *http.Response) []byte {
	h.t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatalf("read response body: %v", err)
	}
	return data
}

// AssertStatus checks that the response has the expected status code.
func (h *TestHarness) AssertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("status = %d, want %d\nbody: %s", resp.StatusCode, expected, string(body))
	}
}

// AssertJSON checks that the response has the expected status and parses the body.
func (h *TestHarness) AssertJSON(t *testing.T, resp *http.Response, expected int, target any) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status = %d, want %d\nbody: %s", resp.StatusCode, expected, string(body))
	}
	h.ParseJSON(resp, target)
}

// --- Default test claims ---

// ManagerClaims returns TestClaims for an order_manager user.
func ManagerClaims() TestClaims {
	return TestClaims{
		SubjectID: "user-manager",
		TenantID:  "acme-corp",
		Email:     "manager@acme.example.com",
		Roles:     []string{"order_manager"},
	}
}

// ViewerClaims returns TestClaims for an order_viewer user.
func ViewerClaims() TestClaims {
	return TestClaims{
		SubjectID: "user-viewer",
		TenantID:  "acme-corp",
		Email:     "viewer@acme.example.com",
		Roles:     []string{"order_viewer"},
	}
}

// ApproverClaims returns TestClaims for an order_approver user.
func ApproverClaims() TestClaims {
	return TestClaims{
		SubjectID: "user-approver",
		TenantID:  "acme-corp",
		Email:     "approver@acme.example.com",
		Roles:     []string{"order_approver"},
	}
}

// ClerkClaims returns TestClaims for an order_clerk user.
func ClerkClaims() TestClaims {
	return TestClaims{
		SubjectID: "user-clerk",
		TenantID:  "acme-corp",
		Email:     "clerk@acme.example.com",
		Roles:     []string{"order_clerk"},
	}
}

// --- Helpers ---

// testdataDir returns the absolute path to the testdata directory.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// OrderFixture returns a map representing a typical order for mock responses.
func OrderFixture(id, orderNum, status string) map[string]any {
	return map[string]any{
		"id":               id,
		"order_number":     orderNum,
		"customer_name":    "Test Customer",
		"total_amount":     99.99,
		"status":           status,
		"priority":         "normal",
		"shipping_address": "123 Test St",
		"notes":            "",
		"created_at":       "2024-01-15T10:30:00Z",
		"updated_at":       "2024-01-15T10:30:00Z",
	}
}

// OrderListFixture returns a paginated list response with the given orders.
func OrderListFixture(orders []map[string]any, total int) map[string]any {
	return map[string]any{
		"data":        orders,
		"total_count": float64(total),
		"page":        float64(1),
		"page_size":   float64(25),
	}
}

// ErrorFixture returns an error response matching the ErrorResponse schema.
func ErrorFixture(code, message string) map[string]any {
	return map[string]any{
		"code":    code,
		"message": message,
	}
}

// ErrorFixtureWithDetails returns an error response with field-level details.
func ErrorFixtureWithDetails(code, message string, details map[string]any) map[string]any {
	return map[string]any{
		"code":    code,
		"message": message,
		"details": details,
	}
}

// FormatJSON converts a value to indented JSON for test output.
func FormatJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}
