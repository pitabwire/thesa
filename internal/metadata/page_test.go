package metadata

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

func testPageDefinitions() []model.DomainDefinition {
	minLen := 3
	return []model.DomainDefinition{
		{
			Domain: "orders",
			Pages: []model.PageDefinition{
				{
					ID:           "orders-list",
					Title:        "Orders",
					Route:        "/orders",
					Layout:       "list",
					Capabilities: []string{"orders:list:view"},
					Breadcrumb: []model.BreadcrumbItem{
						{Label: "Home", Route: "/"},
						{Label: "Orders"},
					},
					Table: &model.TableDefinition{
						DataSource: model.DataSourceDefinition{
							OperationID: "listOrders",
							ServiceID:   "order-svc",
							Mapping: model.ResponseMappingDefinition{
								ItemsPath: "data.items",
								TotalPath: "data.total",
								FieldMap: map[string]string{
									"order_id":   "id",
									"created_at": "date",
								},
							},
						},
						Columns: []model.ColumnDefinition{
							{Field: "id", Label: "ID", Type: "text", Sortable: true, Width: "100px"},
							{Field: "status", Label: "Status", Type: "status", StatusMap: map[string]string{
								"active":    "success",
								"cancelled": "danger",
							}},
							{Field: "customer", Label: "Customer", Type: "text", Link: &model.LinkDefinition{
								Route:  "/customers/{id}",
								Params: map[string]string{"id": "{row.customer_id}"},
							}},
						},
						Filters: []model.FilterDefinition{
							{
								Field:    "status",
								Label:    "Status",
								Type:     "select",
								Operator: "eq",
								Options: &model.FilterOptionsDefinition{
									Static: []model.StaticOption{
										{Label: "Active", Value: "active"},
										{Label: "Cancelled", Value: "cancelled"},
									},
								},
							},
						},
						RowActions: []model.ActionDefinition{
							{ID: "view-order", Label: "View", Type: "navigate", NavigateTo: "/orders/{id}"},
							{ID: "cancel-order", Label: "Cancel", Type: "command", CommandID: "cancel-order",
								Capabilities: []string{"orders:cancel"}},
						},
						BulkActions: []model.ActionDefinition{
							{ID: "export", Label: "Export", Type: "command", CommandID: "export-orders",
								Capabilities: []string{"orders:export"}},
						},
						DefaultSort: "created_at",
						SortDir:     "desc",
						PageSize:    20,
						Selectable:  true,
					},
					Sections: []model.SectionDefinition{
						{
							ID:           "summary",
							Title:        "Summary",
							Layout:       "grid",
							Columns:      2,
							Capabilities: []string{"orders:summary"},
							Fields: []model.FieldDefinition{
								{Field: "total_orders", Label: "Total Orders", Type: "number"},
								{Field: "total_revenue", Label: "Revenue", Type: "currency", Format: "USD"},
							},
						},
					},
					Actions: []model.ActionDefinition{
						{ID: "create-order", Label: "New Order", Type: "navigate", NavigateTo: "/orders/new",
							Capabilities: []string{"orders:create"}},
					},
				},
				{
					ID:     "orders-detail",
					Title:  "Order Detail",
					Route:  "/orders/:id",
					Layout: "detail",
					Sections: []model.SectionDefinition{
						{
							ID:     "info",
							Title:  "Information",
							Layout: "form",
							Fields: []model.FieldDefinition{
								{Field: "name", Label: "Name", Type: "text", Required: true,
									Validation: &model.ValidationDefinition{MinLength: &minLen}},
								{Field: "notes", Label: "Notes", Type: "textarea", ReadOnly: "true"},
								{Field: "hidden_field", Label: "Hidden", Type: "text", Visibility: "hidden"},
							},
						},
					},
				},
			},
		},
	}
}

func newTestPageProvider(invokeFn func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error)) *PageProvider {
	reg := definition.NewRegistry(testPageDefinitions())
	ap := NewActionProvider()

	var invokerReg *invoker.Registry
	if invokeFn != nil {
		invokerReg = invoker.NewRegistry()
		invokerReg.Register(&mockInvokerForMenu{invokeFn: invokeFn})
	} else {
		invokerReg = invoker.NewRegistry()
	}

	return NewPageProvider(reg, invokerReg, ap)
}

// --- GetPage ---

func TestPageProvider_GetPage_success(t *testing.T) {
	p := newTestPageProvider(nil)
	caps := model.CapabilitySet{"orders:list:view": true, "orders:create": true}

	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if desc.ID != "orders-list" {
		t.Errorf("ID = %q, want orders-list", desc.ID)
	}
	if desc.Title != "Orders" {
		t.Errorf("Title = %q, want Orders", desc.Title)
	}
	if desc.Route != "/orders" {
		t.Errorf("Route = %q, want /orders", desc.Route)
	}
	if desc.Layout != "list" {
		t.Errorf("Layout = %q, want list", desc.Layout)
	}
}

func TestPageProvider_GetPage_notFound(t *testing.T) {
	p := newTestPageProvider(nil)

	_, err := p.GetPage(context.Background(), nil, model.CapabilitySet{}, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent page")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrNotFound {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrNotFound)
	}
}

func TestPageProvider_GetPage_forbidden(t *testing.T) {
	p := newTestPageProvider(nil)

	_, err := p.GetPage(context.Background(), nil, model.CapabilitySet{}, "orders-list")
	if err == nil {
		t.Fatal("expected error for insufficient capabilities")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrForbidden {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrForbidden)
	}
}

func TestPageProvider_GetPage_breadcrumb(t *testing.T) {
	p := newTestPageProvider(nil)
	caps := model.CapabilitySet{"orders:list:view": true}

	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Breadcrumb) != 2 {
		t.Fatalf("len(Breadcrumb) = %d, want 2", len(desc.Breadcrumb))
	}
	if desc.Breadcrumb[0].Label != "Home" || desc.Breadcrumb[0].Route != "/" {
		t.Errorf("Breadcrumb[0] = %+v", desc.Breadcrumb[0])
	}
	if desc.Breadcrumb[1].Label != "Orders" {
		t.Errorf("Breadcrumb[1] = %+v", desc.Breadcrumb[1])
	}
}

func TestPageProvider_GetPage_table(t *testing.T) {
	p := newTestPageProvider(nil)
	caps := model.CapabilitySet{"orders:list:view": true, "orders:cancel": true}

	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if desc.Table == nil {
		t.Fatal("Table is nil")
	}
	if desc.Table.DataEndpoint != "/api/pages/orders-list/data" {
		t.Errorf("DataEndpoint = %q", desc.Table.DataEndpoint)
	}
	if desc.Table.DefaultSort != "created_at" {
		t.Errorf("DefaultSort = %q", desc.Table.DefaultSort)
	}
	if desc.Table.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", desc.Table.PageSize)
	}
	if !desc.Table.Selectable {
		t.Error("Selectable = false, want true")
	}
}

func TestPageProvider_GetPage_tableColumns(t *testing.T) {
	p := newTestPageProvider(nil)
	caps := model.CapabilitySet{"orders:list:view": true}

	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Table.Columns) != 3 {
		t.Fatalf("len(Columns) = %d, want 3", len(desc.Table.Columns))
	}
	// Check link resolution.
	custCol := desc.Table.Columns[2]
	if custCol.Link == nil {
		t.Fatal("customer column Link is nil")
	}
	if custCol.Link.Route != "/customers/{id}" {
		t.Errorf("Link.Route = %q", custCol.Link.Route)
	}
	// Check status map.
	statusCol := desc.Table.Columns[1]
	if statusCol.StatusMap["active"] != "success" {
		t.Errorf("StatusMap[active] = %q", statusCol.StatusMap["active"])
	}
}

func TestPageProvider_GetPage_tableFilters(t *testing.T) {
	p := newTestPageProvider(nil)
	caps := model.CapabilitySet{"orders:list:view": true}

	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Table.Filters) != 1 {
		t.Fatalf("len(Filters) = %d, want 1", len(desc.Table.Filters))
	}
	f := desc.Table.Filters[0]
	if f.Field != "status" {
		t.Errorf("Filter.Field = %q", f.Field)
	}
	if len(f.Options) != 2 {
		t.Fatalf("len(Options) = %d, want 2", len(f.Options))
	}
}

func TestPageProvider_GetPage_rowActionsFilteredByCapability(t *testing.T) {
	p := newTestPageProvider(nil)

	// Without orders:cancel capability.
	caps := model.CapabilitySet{"orders:list:view": true}
	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Table.RowActions) != 1 {
		t.Fatalf("len(RowActions) = %d, want 1 (cancel filtered)", len(desc.Table.RowActions))
	}
	if desc.Table.RowActions[0].ID != "view-order" {
		t.Errorf("RowActions[0].ID = %q, want view-order", desc.Table.RowActions[0].ID)
	}

	// With orders:cancel capability.
	caps = model.CapabilitySet{"orders:list:view": true, "orders:cancel": true}
	desc, err = p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Table.RowActions) != 2 {
		t.Errorf("len(RowActions) = %d, want 2", len(desc.Table.RowActions))
	}
}

func TestPageProvider_GetPage_bulkActionsFilteredByCapability(t *testing.T) {
	p := newTestPageProvider(nil)

	caps := model.CapabilitySet{"orders:list:view": true}
	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	// No orders:export capability → bulk action filtered out.
	if len(desc.Table.BulkActions) != 0 {
		t.Errorf("len(BulkActions) = %d, want 0", len(desc.Table.BulkActions))
	}
}

func TestPageProvider_GetPage_sectionsFilteredByCapability(t *testing.T) {
	p := newTestPageProvider(nil)

	// Without orders:summary capability → section omitted.
	caps := model.CapabilitySet{"orders:list:view": true}
	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Sections) != 0 {
		t.Errorf("len(Sections) = %d, want 0", len(desc.Sections))
	}

	// With orders:summary capability → section included.
	caps = model.CapabilitySet{"orders:list:view": true, "orders:summary": true}
	desc, err = p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(desc.Sections))
	}
	if desc.Sections[0].ID != "summary" {
		t.Errorf("Sections[0].ID = %q, want summary", desc.Sections[0].ID)
	}
}

func TestPageProvider_GetPage_hiddenFieldsOmitted(t *testing.T) {
	p := newTestPageProvider(nil)
	caps := model.CapabilitySet{} // No caps needed for orders-detail.

	desc, err := p.GetPage(context.Background(), nil, caps, "orders-detail")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if len(desc.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(desc.Sections))
	}
	// 3 fields defined, but hidden_field has visibility="hidden" → 2 visible.
	if len(desc.Sections[0].Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2 (hidden field omitted)", len(desc.Sections[0].Fields))
	}
}

func TestPageProvider_GetPage_fieldProperties(t *testing.T) {
	p := newTestPageProvider(nil)

	desc, err := p.GetPage(context.Background(), nil, model.CapabilitySet{}, "orders-detail")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	fields := desc.Sections[0].Fields

	// Field: name
	name := fields[0]
	if name.Field != "name" {
		t.Errorf("Field = %q, want name", name.Field)
	}
	if !name.Required {
		t.Error("Required = false, want true")
	}
	if name.Validation == nil {
		t.Fatal("Validation is nil")
	}
	if *name.Validation.MinLength != 3 {
		t.Errorf("MinLength = %d, want 3", *name.Validation.MinLength)
	}

	// Field: notes (read_only)
	notes := fields[1]
	if !notes.ReadOnly {
		t.Error("notes.ReadOnly = false, want true")
	}
}

func TestPageProvider_GetPage_pageActionsFiltered(t *testing.T) {
	p := newTestPageProvider(nil)

	caps := model.CapabilitySet{"orders:list:view": true}
	desc, err := p.GetPage(context.Background(), nil, caps, "orders-list")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	// No orders:create → action filtered.
	if len(desc.Actions) != 0 {
		t.Errorf("len(Actions) = %d, want 0", len(desc.Actions))
	}
}

func TestPageProvider_GetPage_noCapabilitiesRequired(t *testing.T) {
	p := newTestPageProvider(nil)

	// orders-detail has no capabilities requirement.
	desc, err := p.GetPage(context.Background(), nil, model.CapabilitySet{}, "orders-detail")
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}
	if desc.ID != "orders-detail" {
		t.Errorf("ID = %q, want orders-detail", desc.ID)
	}
}

// --- GetPageData ---

func TestPageProvider_GetPageData_success(t *testing.T) {
	p := newTestPageProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		if binding.OperationID != "listOrders" {
			return model.InvocationResult{}, fmt.Errorf("unexpected operation: %s", binding.OperationID)
		}
		if binding.ServiceID != "order-svc" {
			return model.InvocationResult{}, fmt.Errorf("unexpected service: %s", binding.ServiceID)
		}
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body: map[string]any{
				"data": map[string]any{
					"items": []any{
						map[string]any{"order_id": "1", "status": "active", "created_at": "2024-01-01"},
						map[string]any{"order_id": "2", "status": "cancelled", "created_at": "2024-01-02"},
					},
					"total": float64(50),
				},
			},
		}, nil
	})

	caps := model.CapabilitySet{"orders:list:view": true}
	params := model.DataParams{Page: 1, PageSize: 20}

	resp, err := p.GetPageData(context.Background(), nil, caps, "orders-list", params)
	if err != nil {
		t.Fatalf("GetPageData error: %v", err)
	}
	if resp.Data.TotalCount != 50 {
		t.Errorf("TotalCount = %d, want 50", resp.Data.TotalCount)
	}
	if len(resp.Data.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(resp.Data.Items))
	}
	// Field map should rename order_id → id and created_at → date.
	if resp.Data.Items[0]["id"] != "1" {
		t.Errorf("Items[0].id = %v, want 1", resp.Data.Items[0]["id"])
	}
	if resp.Data.Items[0]["date"] != "2024-01-01" {
		t.Errorf("Items[0].date = %v, want 2024-01-01", resp.Data.Items[0]["date"])
	}
	if resp.Data.Page != 1 {
		t.Errorf("Page = %d, want 1", resp.Data.Page)
	}
	if resp.Data.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", resp.Data.PageSize)
	}
}

func TestPageProvider_GetPageData_notFound(t *testing.T) {
	p := newTestPageProvider(nil)

	_, err := p.GetPageData(context.Background(), nil, model.CapabilitySet{}, "nonexistent", model.DataParams{})
	if err == nil {
		t.Fatal("expected error for nonexistent page")
	}
}

func TestPageProvider_GetPageData_forbidden(t *testing.T) {
	p := newTestPageProvider(nil)

	_, err := p.GetPageData(context.Background(), nil, model.CapabilitySet{}, "orders-list", model.DataParams{})
	if err == nil {
		t.Fatal("expected error for insufficient capabilities")
	}
}

func TestPageProvider_GetPageData_noDataSource(t *testing.T) {
	p := newTestPageProvider(nil)

	// orders-detail has no table/data source.
	_, err := p.GetPageData(context.Background(), nil, model.CapabilitySet{}, "orders-detail", model.DataParams{})
	if err == nil {
		t.Fatal("expected error for page without data source")
	}
	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrBadRequest {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrBadRequest)
	}
}

func TestPageProvider_GetPageData_passesQueryParams(t *testing.T) {
	var capturedInput model.InvocationInput
	p := newTestPageProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		capturedInput = input
		return model.InvocationResult{
			StatusCode: http.StatusOK,
			Body:       map[string]any{"data": map[string]any{"items": []any{}, "total": float64(0)}},
		}, nil
	})

	caps := model.CapabilitySet{"orders:list:view": true}
	params := model.DataParams{
		Page:     2,
		PageSize: 10,
		Sort:     "name",
		SortDir:  "asc",
		Query:    "search",
		Filters:  map[string]string{"status": "active"},
	}

	_, err := p.GetPageData(context.Background(), nil, caps, "orders-list", params)
	if err != nil {
		t.Fatalf("GetPageData error: %v", err)
	}

	if capturedInput.QueryParams["page"] != "2" {
		t.Errorf("page = %q, want 2", capturedInput.QueryParams["page"])
	}
	if capturedInput.QueryParams["page_size"] != "10" {
		t.Errorf("page_size = %q, want 10", capturedInput.QueryParams["page_size"])
	}
	if capturedInput.QueryParams["sort"] != "name" {
		t.Errorf("sort = %q, want name", capturedInput.QueryParams["sort"])
	}
	if capturedInput.QueryParams["sort_dir"] != "asc" {
		t.Errorf("sort_dir = %q, want asc", capturedInput.QueryParams["sort_dir"])
	}
	if capturedInput.QueryParams["q"] != "search" {
		t.Errorf("q = %q, want search", capturedInput.QueryParams["q"])
	}
	if capturedInput.QueryParams["status"] != "active" {
		t.Errorf("status = %q, want active", capturedInput.QueryParams["status"])
	}
}

func TestPageProvider_GetPageData_backendError(t *testing.T) {
	p := newTestPageProvider(func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
		return model.InvocationResult{}, fmt.Errorf("backend error")
	})

	caps := model.CapabilitySet{"orders:list:view": true}
	_, err := p.GetPageData(context.Background(), nil, caps, "orders-list", model.DataParams{})
	if err == nil {
		t.Fatal("expected error from backend")
	}
}

// --- Helper tests ---

func TestExtractPath(t *testing.T) {
	data := map[string]any{
		"data": map[string]any{
			"items": []any{"a", "b"},
			"total": float64(2),
		},
	}

	items := extractPath(data, "data.items")
	slice, ok := items.([]any)
	if !ok || len(slice) != 2 {
		t.Errorf("extractPath(data.items) = %v", items)
	}

	total := extractPath(data, "data.total")
	if total != float64(2) {
		t.Errorf("extractPath(data.total) = %v", total)
	}

	if extractPath(data, "") != nil {
		t.Error("extractPath('') should be nil")
	}

	if extractPath(data, "nonexistent.path") != nil {
		t.Error("extractPath(nonexistent.path) should be nil")
	}
}

func TestApplyFieldMap(t *testing.T) {
	items := []map[string]any{
		{"old_name": "Alice", "age": float64(30)},
		{"old_name": "Bob", "age": float64(25)},
	}
	fieldMap := map[string]string{"old_name": "name"}

	result := applyFieldMap(items, fieldMap)
	if result[0]["name"] != "Alice" {
		t.Errorf("result[0].name = %v, want Alice", result[0]["name"])
	}
	if result[0]["age"] != float64(30) {
		t.Errorf("result[0].age = %v, want 30", result[0]["age"])
	}
	// old_name should not exist (renamed to name).
	if _, exists := result[0]["old_name"]; exists {
		t.Error("old_name should not exist after renaming")
	}
}

func TestBuildDataInput(t *testing.T) {
	params := model.DataParams{
		Page:     1,
		PageSize: 25,
		Sort:     "id",
		SortDir:  "desc",
		Filters:  map[string]string{"status": "active"},
	}
	input := buildDataInput(params)
	if input.QueryParams["page"] != "1" {
		t.Errorf("page = %q", input.QueryParams["page"])
	}
	if input.QueryParams["status"] != "active" {
		t.Errorf("status = %q", input.QueryParams["status"])
	}
}
