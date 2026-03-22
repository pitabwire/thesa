package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/search"
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

// makeRouterRequest creates a stdlib mux-routed request with URL params and context injected.
func makeRouterRequest(method, pattern, path string, body []byte, handler http.HandlerFunc, rctx *model.RequestContext, caps model.CapabilitySet) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.Handle(method+" "+pattern, contextMiddleware(rctx, caps)(handler))

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// newRegistry creates a definition.Registry from DomainDefinitions.
func newRegistry(defs ...model.DomainDefinition) *definition.Registry {
	return definition.NewRegistry(defs)
}

// --- fake invoker (test double returning canned responses) ---

type fakeInvoker struct {
	result model.InvocationResult
	err    error
}

func (f *fakeInvoker) Invoke(_ context.Context, _ *model.RequestContext, _ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
	return f.result, f.err
}

func (f *fakeInvoker) Supports(_ model.OperationBinding) bool { return true }

func newTestInvokerRegistry(inv model.OperationInvoker) *invoker.Registry {
	reg := invoker.NewRegistry()
	reg.Register(inv)
	return reg
}

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ui/navigation", handler)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/ui/navigation", nil))

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
	inv := &fakeInvoker{
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
	inv := &fakeInvoker{
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ui/forms/{formId}", handler)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/ui/forms/test", nil))
	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Command handler tests ---

func TestHandleCommand_success(t *testing.T) {
	inv := &fakeInvoker{
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
	executor := command.NewCommandExecutor(reg, newTestInvokerRegistry(&fakeInvoker{}), nil)
	handler := handleCommand(executor)

	w := makeRouterRequest("POST", "/ui/commands/{commandId}", "/ui/commands/orders.create", []byte("not json"), handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCommand_notFound(t *testing.T) {
	reg := newRegistry()
	executor := command.NewCommandExecutor(reg, newTestInvokerRegistry(&fakeInvoker{}), nil)
	handler := handleCommand(executor)

	body, _ := json.Marshal(model.CommandInput{Input: map[string]any{}})
	w := makeRouterRequest("POST", "/ui/commands/{commandId}", "/ui/commands/nonexistent", body, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleCommand_noRequestContext(t *testing.T) {
	executor := command.NewCommandExecutor(newRegistry(), newTestInvokerRegistry(&fakeInvoker{}), nil)
	handler := handleCommand(executor)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /ui/commands/{commandId}", handler)
	body, _ := json.Marshal(model.CommandInput{Input: map[string]any{}})
	req := httptest.NewRequest("POST", "/ui/commands/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Search handler tests ---

func TestHandleSearch_success(t *testing.T) {
	inv := &fakeInvoker{
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
	provider := search.NewSearchProvider(reg, newTestInvokerRegistry(&fakeInvoker{}), 3*time.Second, 50)
	handler := handleSearch(provider)

	w := makeRouterRequest("GET", "/ui/search", "/ui/search?q=a", nil, handler, testRequestContext(), testCaps())
	if w.Code != 400 {
		t.Errorf("status = %d, want 400 (query too short)", w.Code)
	}
}

func TestHandleSearch_noRequestContext(t *testing.T) {
	provider := search.NewSearchProvider(newRegistry(), newTestInvokerRegistry(&fakeInvoker{}), 3*time.Second, 50)
	handler := handleSearch(provider)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ui/search", handler)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/ui/search?q=test", nil))

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- Lookup handler tests ---

func TestHandleLookup_success(t *testing.T) {
	inv := &fakeInvoker{
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
	provider := search.NewLookupProvider(reg, newTestInvokerRegistry(&fakeInvoker{}), 5*time.Minute, 100)
	handler := handleLookup(provider)

	w := makeRouterRequest("GET", "/ui/lookups/{lookupId}", "/ui/lookups/nonexistent", nil, handler, testRequestContext(), testCaps())
	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleLookup_noRequestContext(t *testing.T) {
	provider := search.NewLookupProvider(newRegistry(), newTestInvokerRegistry(&fakeInvoker{}), 5*time.Minute, 100)
	handler := handleLookup(provider)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ui/lookups/{lookupId}", handler)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/ui/lookups/test", nil))

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
