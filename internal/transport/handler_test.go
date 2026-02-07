package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/internal/workflow"
	"github.com/pitabwire/thesa/model"
)

// --- Test helpers ---

// contextMiddleware injects a RequestContext and CapabilitySet into the request.
func contextMiddleware(rctx *model.RequestContext, caps model.CapabilitySet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := model.WithRequestContext(r.Context(), rctx)
			ctx = context.WithValue(ctx, capabilitiesKey{}, caps)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func testRequestContext() *model.RequestContext {
	return &model.RequestContext{
		SubjectID: "user-1",
		TenantID:  "tenant-1",
		Email:     "user@example.com",
	}
}

func testCaps() model.CapabilitySet {
	return model.CapabilitySet{
		"orders:*":   true,
		"forms:*":    true,
		"workflow:*": true,
	}
}

// makeRouterRequest creates a chi-routed request with URL params and context injected.
func makeRouterRequest(method, pattern, path string, body []byte, handler http.HandlerFunc, rctx *model.RequestContext, caps model.CapabilitySet) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Use(contextMiddleware(rctx, caps))
	switch method {
	case "GET":
		r.Get(pattern, handler)
	case "POST":
		r.Post(pattern, handler)
	}

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// newRegistry creates a definition.Registry from DomainDefinitions.
func newRegistry(defs ...model.DomainDefinition) *definition.Registry {
	return definition.NewRegistry(defs)
}

// --- stub invoker ---

type stubInvoker struct {
	result model.InvocationResult
	err    error
}

func (s *stubInvoker) Invoke(_ context.Context, _ *model.RequestContext, _ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
	return s.result, s.err
}

func (s *stubInvoker) Supports(_ model.OperationBinding) bool { return true }

func newTestInvokerRegistry(inv model.OperationInvoker) *invoker.Registry {
	reg := invoker.NewRegistry()
	reg.Register(inv)
	return reg
}

// --- stub cap resolver ---

type stubCapResolver struct {
	caps model.CapabilitySet
}

func (s *stubCapResolver) Resolve(_ *model.RequestContext) (model.CapabilitySet, error) {
	return s.caps, nil
}
func (s *stubCapResolver) Invalidate(_, _ string) {}

// --- Navigation handler tests ---

func TestHandleNavigation_success(t *testing.T) {
	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Navigation: model.NavigationDefinition{
			Label: "Orders",
			Icon:  "shopping-cart",
			Order: 1,
			Children: []model.NavigationChildDefinition{
				{PageID: "orders.list", Label: "Order List", Route: "/orders", Order: 1},
			},
		},
	})

	menu := metadata.NewMenuProvider(reg, nil)
	handler := handleNavigation(menu)

	w := makeRouterRequest("GET", "/ui/navigation", "/ui/navigation", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var tree model.NavigationTree
	json.NewDecoder(w.Body).Decode(&tree)
	if len(tree.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(tree.Items))
	}
	if tree.Items[0].Label != "Orders" {
		t.Errorf("label = %q, want Orders", tree.Items[0].Label)
	}
}

func TestHandleNavigation_noRequestContext(t *testing.T) {
	menu := metadata.NewMenuProvider(newRegistry(), nil)
	handler := handleNavigation(menu)

	r := chi.NewRouter()
	r.Get("/ui/navigation", handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/navigation", nil))

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Page handler tests ---

func TestHandleGetPage_success(t *testing.T) {
	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Pages: []model.PageDefinition{
			{ID: "orders.list", Title: "Orders", Route: "/orders", Layout: "table"},
		},
	})

	actions := metadata.NewActionProvider()
	pages := metadata.NewPageProvider(reg, nil, actions)
	handler := handleGetPage(pages)

	w := makeRouterRequest("GET", "/ui/pages/{pageId}", "/ui/pages/orders.list", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var desc model.PageDescriptor
	json.NewDecoder(w.Body).Decode(&desc)
	if desc.Title != "Orders" {
		t.Errorf("title = %q, want Orders", desc.Title)
	}
}

func TestHandleGetPage_notFound(t *testing.T) {
	reg := newRegistry()
	actions := metadata.NewActionProvider()
	pages := metadata.NewPageProvider(reg, nil, actions)
	handler := handleGetPage(pages)

	w := makeRouterRequest("GET", "/ui/pages/{pageId}", "/ui/pages/nonexistent", nil, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleGetPageData_success(t *testing.T) {
	inv := &stubInvoker{
		result: model.InvocationResult{
			StatusCode: 200,
			Body: map[string]any{
				"data":  []any{map[string]any{"id": "1", "name": "Order A"}},
				"total": float64(1),
			},
		},
	}

	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Pages: []model.PageDefinition{
			{
				ID: "orders.list", Title: "Orders", Layout: "table",
				Table: &model.TableDefinition{
					DataSource: model.DataSourceDefinition{
						ServiceID:   "orders-svc",
						OperationID: "listOrders",
						Mapping: model.ResponseMappingDefinition{
							ItemsPath: "data",
							TotalPath: "total",
						},
					},
				},
			},
		},
	})

	actions := metadata.NewActionProvider()
	pages := metadata.NewPageProvider(reg, newTestInvokerRegistry(inv), actions)
	handler := handleGetPageData(pages)

	w := makeRouterRequest("GET", "/ui/pages/{pageId}/data", "/ui/pages/orders.list/data?page=1&page_size=10", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp model.DataResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data.Items) != 1 {
		t.Errorf("items = %d, want 1", len(resp.Data.Items))
	}
}

// --- Form handler tests ---

func TestHandleGetForm_success(t *testing.T) {
	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Forms: []model.FormDefinition{
			{
				ID:    "orders.create",
				Title: "Create Order",
				Sections: []model.SectionDefinition{
					{
						ID:    "main",
						Title: "Details",
						Fields: []model.FieldDefinition{
							{Field: "name", Label: "Name", Type: "text"},
						},
					},
				},
			},
		},
	})

	actions := metadata.NewActionProvider()
	forms := metadata.NewFormProvider(reg, nil, actions)
	handler := handleGetForm(forms)

	w := makeRouterRequest("GET", "/ui/forms/{formId}", "/ui/forms/orders.create", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var desc model.FormDescriptor
	json.NewDecoder(w.Body).Decode(&desc)
	if desc.Title != "Create Order" {
		t.Errorf("title = %q, want Create Order", desc.Title)
	}
}

func TestHandleGetForm_notFound(t *testing.T) {
	reg := newRegistry()
	actions := metadata.NewActionProvider()
	forms := metadata.NewFormProvider(reg, nil, actions)
	handler := handleGetForm(forms)

	w := makeRouterRequest("GET", "/ui/forms/{formId}", "/ui/forms/nonexistent", nil, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleGetFormData_noLoadSource(t *testing.T) {
	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Forms: []model.FormDefinition{
			{ID: "orders.create", Title: "Create Order"},
		},
	})

	actions := metadata.NewActionProvider()
	forms := metadata.NewFormProvider(reg, nil, actions)
	handler := handleGetFormData(forms)

	w := makeRouterRequest("GET", "/ui/forms/{formId}/data", "/ui/forms/orders.create/data", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandleGetFormData_withLoadSource(t *testing.T) {
	inv := &stubInvoker{
		result: model.InvocationResult{
			StatusCode: 200,
			Body:       map[string]any{"name": "Existing Order", "amount": float64(100)},
		},
	}

	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Forms: []model.FormDefinition{
			{
				ID:    "orders.edit",
				Title: "Edit Order",
				LoadSource: &model.DataSourceDefinition{
					ServiceID:   "orders-svc",
					OperationID: "getOrder",
				},
				Sections: []model.SectionDefinition{
					{
						ID: "main",
						Fields: []model.FieldDefinition{
							{Field: "name", Label: "Name", Type: "text"},
							{Field: "amount", Label: "Amount", Type: "number"},
						},
					},
				},
			},
		},
	})

	actions := metadata.NewActionProvider()
	forms := metadata.NewFormProvider(reg, newTestInvokerRegistry(inv), actions)
	handler := handleGetFormData(forms)

	w := makeRouterRequest("GET", "/ui/forms/{formId}/data", "/ui/forms/orders.edit/data?id=order-1", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandleGetForm_noRequestContext(t *testing.T) {
	reg := newRegistry()
	actions := metadata.NewActionProvider()
	forms := metadata.NewFormProvider(reg, nil, actions)
	handler := handleGetForm(forms)

	r := chi.NewRouter()
	r.Get("/ui/forms/{formId}", handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/forms/test", nil))
	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Command handler tests ---

func TestHandleCommand_success(t *testing.T) {
	inv := &stubInvoker{
		result: model.InvocationResult{
			StatusCode: 200,
			Body:       map[string]any{"id": "order-1"},
		},
	}

	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Commands: []model.CommandDefinition{
			{
				ID: "orders.create",
				Operation: model.OperationBinding{
					Type:        "openapi",
					ServiceID:   "orders-svc",
					OperationID: "createOrder",
				},
				Output: model.OutputMapping{
					SuccessMessage: "Order created",
				},
			},
		},
	})

	executor := command.NewCommandExecutor(reg, newTestInvokerRegistry(inv), nil)
	handler := handleCommand(executor)

	body, _ := json.Marshal(model.CommandInput{
		Input: map[string]any{"name": "Test Order"},
	})

	w := makeRouterRequest("POST", "/ui/commands/{commandId}", "/ui/commands/orders.create", body, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp model.CommandResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Errorf("success = false, want true")
	}
}

func TestHandleCommand_invalidJSON(t *testing.T) {
	reg := newRegistry()
	executor := command.NewCommandExecutor(reg, newTestInvokerRegistry(&stubInvoker{}), nil)
	handler := handleCommand(executor)

	w := makeRouterRequest("POST", "/ui/commands/{commandId}", "/ui/commands/orders.create", []byte("not json"), handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCommand_notFound(t *testing.T) {
	reg := newRegistry()
	executor := command.NewCommandExecutor(reg, newTestInvokerRegistry(&stubInvoker{}), nil)
	handler := handleCommand(executor)

	body, _ := json.Marshal(model.CommandInput{Input: map[string]any{}})
	w := makeRouterRequest("POST", "/ui/commands/{commandId}", "/ui/commands/nonexistent", body, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleCommand_idempotencyKeyFromHeader(t *testing.T) {
	inv := &stubInvoker{
		result: model.InvocationResult{
			StatusCode: 200,
			Body:       map[string]any{},
		},
	}

	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Commands: []model.CommandDefinition{
			{
				ID: "orders.create",
				Operation: model.OperationBinding{
					Type:        "openapi",
					ServiceID:   "orders-svc",
					OperationID: "createOrder",
				},
				Output: model.OutputMapping{SuccessMessage: "OK"},
			},
		},
	})

	executor := command.NewCommandExecutor(reg, newTestInvokerRegistry(inv), nil)
	handler := handleCommand(executor)

	body, _ := json.Marshal(model.CommandInput{Input: map[string]any{"name": "test"}})

	r := chi.NewRouter()
	r.Use(contextMiddleware(testRequestContext(), testCaps()))
	r.Post("/ui/commands/{commandId}", handler)

	req := httptest.NewRequest("POST", "/ui/commands/orders.create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", "idem-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHandleCommand_noRequestContext(t *testing.T) {
	executor := command.NewCommandExecutor(newRegistry(), newTestInvokerRegistry(&stubInvoker{}), nil)
	handler := handleCommand(executor)

	r := chi.NewRouter()
	r.Post("/ui/commands/{commandId}", handler)
	body, _ := json.Marshal(model.CommandInput{Input: map[string]any{}})
	req := httptest.NewRequest("POST", "/ui/commands/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Workflow handler tests ---

func newTestWorkflowEngine(inv model.OperationInvoker, caps model.CapabilitySet) *workflow.Engine {
	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Workflows: []model.WorkflowDefinition{
			{
				ID:          "approval",
				Name:        "Approval Flow",
				InitialStep: "submit",
				Steps: []model.StepDefinition{
					{ID: "submit", Name: "Submit", Type: "human"},
					{ID: "done", Name: "Done", Type: "terminal"},
				},
				Transitions: []model.TransitionDefinition{
					{From: "submit", Event: "approve", To: "done"},
				},
			},
		},
	})

	store := workflow.NewMemoryWorkflowStore()
	resolver := &stubCapResolver{caps: caps}
	return workflow.NewEngine(reg, store, newTestInvokerRegistry(inv), resolver)
}

func TestHandleWorkflowStart_success(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowStart(engine)

	body, _ := json.Marshal(map[string]any{
		"input": map[string]any{"name": "Test"},
	})

	w := makeRouterRequest("POST", "/ui/workflows/{workflowId}/start", "/ui/workflows/approval/start", body, handler, testRequestContext(), testCaps())
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	var inst model.WorkflowInstance
	json.NewDecoder(w.Body).Decode(&inst)
	if inst.WorkflowID != "approval" {
		t.Errorf("workflow_id = %q, want approval", inst.WorkflowID)
	}
	if inst.Status != "active" {
		t.Errorf("status = %q, want active", inst.Status)
	}
}

func TestHandleWorkflowStart_invalidJSON(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowStart(engine)

	w := makeRouterRequest("POST", "/ui/workflows/{workflowId}/start", "/ui/workflows/approval/start", []byte("bad"), handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleWorkflowStart_notFound(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowStart(engine)

	body, _ := json.Marshal(map[string]any{"input": map[string]any{}})
	w := makeRouterRequest("POST", "/ui/workflows/{workflowId}/start", "/ui/workflows/nonexistent/start", body, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleWorkflowAdvance_success(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	rctx := testRequestContext()

	inst, err := engine.Start(context.Background(), rctx, "approval", map[string]any{"name": "Test"})
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	handler := handleWorkflowAdvance(engine)
	body, _ := json.Marshal(map[string]any{
		"event": "approve",
		"input": map[string]any{"approved_by": "manager"},
	})

	w := makeRouterRequest("POST", "/ui/workflows/{instanceId}/advance", fmt.Sprintf("/ui/workflows/%s/advance", inst.ID), body, handler, rctx, testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var advanced model.WorkflowInstance
	json.NewDecoder(w.Body).Decode(&advanced)
	if advanced.Status != "completed" {
		t.Errorf("status = %q, want completed", advanced.Status)
	}
}

func TestHandleWorkflowAdvance_invalidJSON(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowAdvance(engine)

	w := makeRouterRequest("POST", "/ui/workflows/{instanceId}/advance", "/ui/workflows/inst-1/advance", []byte("bad"), handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleWorkflowGet_success(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	rctx := testRequestContext()

	inst, err := engine.Start(context.Background(), rctx, "approval", nil)
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	handler := handleWorkflowGet(engine)
	w := makeRouterRequest("GET", "/ui/workflows/{instanceId}", fmt.Sprintf("/ui/workflows/%s", inst.ID), nil, handler, rctx, testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var desc model.WorkflowDescriptor
	json.NewDecoder(w.Body).Decode(&desc)
	if desc.Name != "Approval Flow" {
		t.Errorf("name = %q, want Approval Flow", desc.Name)
	}
}

func TestHandleWorkflowGet_notFound(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowGet(engine)

	w := makeRouterRequest("GET", "/ui/workflows/{instanceId}", "/ui/workflows/nonexistent", nil, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleWorkflowCancel_success(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	rctx := testRequestContext()

	inst, err := engine.Start(context.Background(), rctx, "approval", nil)
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	handler := handleWorkflowCancel(engine)
	body, _ := json.Marshal(map[string]any{"reason": "no longer needed"})
	w := makeRouterRequest("POST", "/ui/workflows/{instanceId}/cancel", fmt.Sprintf("/ui/workflows/%s/cancel", inst.ID), body, handler, rctx, testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestHandleWorkflowCancel_invalidJSON(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowCancel(engine)

	w := makeRouterRequest("POST", "/ui/workflows/{instanceId}/cancel", "/ui/workflows/inst-1/cancel", []byte("bad"), handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleWorkflowList_success(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	rctx := testRequestContext()

	_, _ = engine.Start(context.Background(), rctx, "approval", map[string]any{"name": "A"})
	_, _ = engine.Start(context.Background(), rctx, "approval", map[string]any{"name": "B"})

	handler := handleWorkflowList(engine)
	w := makeRouterRequest("GET", "/ui/workflows", "/ui/workflows?page=1&page_size=10", nil, handler, rctx, testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data       []model.WorkflowSummary `json:"data"`
		TotalCount int                     `json:"total_count"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(resp.Data))
	}
	if resp.TotalCount != 2 {
		t.Errorf("total_count = %d, want 2", resp.TotalCount)
	}
}

func TestHandleWorkflowList_noRequestContext(t *testing.T) {
	engine := newTestWorkflowEngine(&stubInvoker{}, testCaps())
	handler := handleWorkflowList(engine)

	r := chi.NewRouter()
	r.Get("/ui/workflows", handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/workflows", nil))

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Search handler tests ---

func TestHandleSearch_success(t *testing.T) {
	inv := &stubInvoker{
		result: model.InvocationResult{
			StatusCode: 200,
			Body: []any{
				map[string]any{"id": "1", "title": "Order 1"},
			},
		},
	}

	reg := newRegistry(model.DomainDefinition{
		Domain: "orders",
		Searches: []model.SearchDefinition{
			{
				ID:     "orders",
				Domain: "orders",
				Operation: model.OperationBinding{
					Type:        "openapi",
					ServiceID:   "orders-svc",
					OperationID: "searchOrders",
				},
				ResultMapping: model.SearchResultMapping{
					IDField:    "id",
					TitleField: "title",
					Route:      "/orders/{id}",
				},
				Weight: 10,
			},
		},
	})

	provider := search.NewSearchProvider(reg, newTestInvokerRegistry(inv), 3*time.Second, 50)
	handler := handleSearch(provider)

	w := makeRouterRequest("GET", "/ui/search", "/ui/search?q=order&page=1&page_size=10", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp model.SearchResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data.Results) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestHandleSearch_queryTooShort(t *testing.T) {
	reg := newRegistry()
	provider := search.NewSearchProvider(reg, newTestInvokerRegistry(&stubInvoker{}), 3*time.Second, 50)
	handler := handleSearch(provider)

	w := makeRouterRequest("GET", "/ui/search", "/ui/search?q=a", nil, handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400 (query too short)", w.Code)
	}
}

func TestHandleSearch_noRequestContext(t *testing.T) {
	provider := search.NewSearchProvider(newRegistry(), newTestInvokerRegistry(&stubInvoker{}), 3*time.Second, 50)
	handler := handleSearch(provider)

	r := chi.NewRouter()
	r.Get("/ui/search", handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/search?q=test", nil))

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Lookup handler tests ---

func TestHandleLookup_success(t *testing.T) {
	inv := &stubInvoker{
		result: model.InvocationResult{
			StatusCode: 200,
			Body:       []any{map[string]any{"name": "USD", "code": "USD"}},
		},
	}

	reg := newRegistry(model.DomainDefinition{
		Domain: "reference",
		Lookups: []model.LookupDefinition{
			{
				ID: "currencies",
				Operation: model.OperationBinding{
					Type:        "openapi",
					ServiceID:   "ref-svc",
					OperationID: "getCurrencies",
				},
				LabelField: "name",
				ValueField: "code",
			},
		},
	})

	provider := search.NewLookupProvider(reg, newTestInvokerRegistry(inv), 5*time.Minute, 100)
	handler := handleLookup(provider)

	w := makeRouterRequest("GET", "/ui/lookups/{lookupId}", "/ui/lookups/currencies?q=usd", nil, handler, testRequestContext(), testCaps())
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp model.LookupResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data.Options) == 0 {
		t.Error("expected at least one option")
	}
}

func TestHandleLookup_notFound(t *testing.T) {
	reg := newRegistry()
	provider := search.NewLookupProvider(reg, newTestInvokerRegistry(&stubInvoker{}), 5*time.Minute, 100)
	handler := handleLookup(provider)

	w := makeRouterRequest("GET", "/ui/lookups/{lookupId}", "/ui/lookups/nonexistent", nil, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleLookup_noRequestContext(t *testing.T) {
	provider := search.NewLookupProvider(newRegistry(), newTestInvokerRegistry(&stubInvoker{}), 5*time.Minute, 100)
	handler := handleLookup(provider)

	r := chi.NewRouter()
	r.Get("/ui/lookups/{lookupId}", handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ui/lookups/test", nil))

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- queryInt and queryMap tests ---

func TestQueryInt_default(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=", nil)
	if got := queryInt(req, "page", 1); got != 1 {
		t.Errorf("queryInt empty = %d, want 1", got)
	}
}

func TestQueryInt_valid(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=5", nil)
	if got := queryInt(req, "page", 1); got != 5 {
		t.Errorf("queryInt = %d, want 5", got)
	}
}

func TestQueryInt_invalid(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=abc", nil)
	if got := queryInt(req, "page", 1); got != 1 {
		t.Errorf("queryInt invalid = %d, want 1", got)
	}
}

func TestQueryMap_success(t *testing.T) {
	req := httptest.NewRequest("GET", "/?filter[status]=active&filter[type]=urgent", nil)
	result := queryMap(req, "filter")
	if result["status"] != "active" {
		t.Errorf("status = %q, want active", result["status"])
	}
	if result["type"] != "urgent" {
		t.Errorf("type = %q, want urgent", result["type"])
	}
}

func TestQueryMap_empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/?other=value", nil)
	result := queryMap(req, "filter")
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}
