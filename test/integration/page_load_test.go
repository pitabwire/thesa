package integration

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ==========================================================================
// Navigation Tests
// ==========================================================================

func TestNavigation_CapabilityFiltering(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("user with orders:view sees orders domain", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var tree map[string]any
		h.ParseJSON(resp, &tree)

		items, _ := tree["items"].([]any)
		if len(items) == 0 {
			t.Fatal("expected at least one navigation item")
		}

		node := items[0].(map[string]any)
		assertEqual(t, node["id"], "orders", "domain id")
		assertEqual(t, node["label"], "Orders", "domain label")
		assertEqual(t, node["icon"], "shopping_cart", "domain icon")
	})

	t.Run("user without orders:view sees empty navigation", func(t *testing.T) {
		token := h.GenerateToken(TestClaims{
			SubjectID: "user-nobody",
			TenantID:  "acme-corp",
			Email:     "nobody@acme.example.com",
			Roles:     []string{"nonexistent_role"},
		})
		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var tree map[string]any
		h.ParseJSON(resp, &tree)

		items, _ := tree["items"].([]any)
		if len(items) != 0 {
			t.Errorf("expected empty navigation, got %d items", len(items))
		}
	})
}

func TestNavigation_ChildrenSorted(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/navigation", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var tree map[string]any
	h.ParseJSON(resp, &tree)

	items := tree["items"].([]any)
	children := items[0].(map[string]any)["children"].([]any)

	if len(children) < 2 {
		t.Fatalf("expected >= 2 children, got %d", len(children))
	}

	// All Orders (order=1) should come before Create Order (order=2).
	assertEqual(t, children[0].(map[string]any)["label"], "All Orders", "first child")
	assertEqual(t, children[1].(map[string]any)["label"], "Create Order", "second child")
}

func TestNavigation_ChildCapabilityFiltering(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("viewer sees only All Orders", func(t *testing.T) {
		// Viewer has orders:view but NOT orders:create.
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var tree map[string]any
		h.ParseJSON(resp, &tree)

		items := tree["items"].([]any)
		children := items[0].(map[string]any)["children"].([]any)

		if len(children) != 1 {
			t.Fatalf("expected 1 child for viewer, got %d", len(children))
		}
		assertEqual(t, children[0].(map[string]any)["label"], "All Orders", "child label")
	})

	t.Run("manager sees both children", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.GET("/ui/navigation", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var tree map[string]any
		h.ParseJSON(resp, &tree)

		items := tree["items"].([]any)
		children := items[0].(map[string]any)["children"].([]any)

		if len(children) != 2 {
			t.Fatalf("expected 2 children for manager, got %d", len(children))
		}
	})
}

func TestNavigation_BadgeResolution(t *testing.T) {
	// Badge resolution uses domain.Domain ("orders") as the ServiceID.
	// Register a spec source with ID "orders" so the invoker can resolve it.
	specFile := filepath.Join(testdataDir(), "specs", "orders-svc.yaml")
	h := NewTestHarness(t,
		WithSpec("orders-svc", specFile),
		WithSpec("orders", specFile),
	)

	h.MockBackend("orders").OnOperation("getOrderCounts").
		RespondWith(200, map[string]any{
			"pending_count":    5,
			"processing_count": 3,
			"total_count":      42,
		})

	token := h.GenerateToken(ManagerClaims())
	resp := h.GET("/ui/navigation", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var tree map[string]any
	h.ParseJSON(resp, &tree)

	items := tree["items"].([]any)
	children := items[0].(map[string]any)["children"].([]any)
	allOrders := children[0].(map[string]any)

	badge, ok := allOrders["badge"].(map[string]any)
	if !ok {
		t.Fatal("expected badge on All Orders navigation item")
	}
	assertFloatEqual(t, badge["count"], 5, "badge count")
	assertEqual(t, badge["style"], "warning", "badge style")

	h.MockBackend("orders").AssertCalled(t, "getOrderCounts", 1)
}

func TestNavigation_BadgeOmittedOnFailure(t *testing.T) {
	// Default harness: badge tries service "orders" which is not configured.
	// Badge should be gracefully omitted.
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/navigation", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var tree map[string]any
	h.ParseJSON(resp, &tree)

	items := tree["items"].([]any)
	children := items[0].(map[string]any)["children"].([]any)
	allOrders := children[0].(map[string]any)

	if allOrders["badge"] != nil {
		t.Errorf("expected no badge when service unavailable, got %v", allOrders["badge"])
	}
}

// ==========================================================================
// Page Descriptor Tests
// ==========================================================================

func TestPageDescriptor_CompleteStructure(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/pages/orders.list", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var page map[string]any
	h.ParseJSON(resp, &page)

	// Basic fields.
	assertEqual(t, page["id"], "orders.list", "id")
	assertEqual(t, page["title"], "All Orders", "title")
	assertEqual(t, page["route"], "/orders", "route")
	assertEqual(t, page["layout"], "table", "layout")

	// Breadcrumb.
	breadcrumb, _ := page["breadcrumb"].([]any)
	if len(breadcrumb) != 2 {
		t.Fatalf("expected 2 breadcrumb items, got %d", len(breadcrumb))
	}
	assertEqual(t, breadcrumb[0].(map[string]any)["label"], "Home", "breadcrumb[0].label")
	assertEqual(t, breadcrumb[0].(map[string]any)["route"], "/", "breadcrumb[0].route")
	assertEqual(t, breadcrumb[1].(map[string]any)["label"], "Orders", "breadcrumb[1].label")
	assertEqual(t, breadcrumb[1].(map[string]any)["route"], "/orders", "breadcrumb[1].route")

	// Table.
	table, ok := page["table"].(map[string]any)
	if !ok {
		t.Fatal("expected table in page descriptor")
	}

	// 5 columns.
	columns := table["columns"].([]any)
	assertLen(t, columns, 5, "columns")

	// order_number column with link.
	col0 := columns[0].(map[string]any)
	assertEqual(t, col0["field"], "order_number", "columns[0].field")
	assertEqual(t, col0["label"], "Order #", "columns[0].label")
	assertEqual(t, col0["type"], "text", "columns[0].type")
	if col0["sortable"] != true {
		t.Error("columns[0] should be sortable")
	}
	link, hasLink := col0["link"].(map[string]any)
	if !hasLink {
		t.Fatal("expected link on order_number column")
	}
	assertEqual(t, link["route"], "/orders/{id}", "columns[0].link.route")

	// currency column.
	col2 := columns[2].(map[string]any)
	assertEqual(t, col2["field"], "total_amount", "columns[2].field")
	assertEqual(t, col2["type"], "currency", "columns[2].type")
	assertEqual(t, col2["format"], "USD", "columns[2].format")

	// badge column with status_map.
	col3 := columns[3].(map[string]any)
	assertEqual(t, col3["field"], "status", "columns[3].field")
	assertEqual(t, col3["type"], "badge", "columns[3].type")
	statusMap := col3["status_map"].(map[string]any)
	assertEqual(t, statusMap["pending"], "warning", "status_map.pending")
	assertEqual(t, statusMap["processing"], "info", "status_map.processing")
	assertEqual(t, statusMap["shipped"], "success", "status_map.shipped")
	assertEqual(t, statusMap["cancelled"], "error", "status_map.cancelled")

	// datetime column.
	col4 := columns[4].(map[string]any)
	assertEqual(t, col4["field"], "created_at", "columns[4].field")
	assertEqual(t, col4["type"], "datetime", "columns[4].type")

	// 3 filters: status (select+lookup), priority (select+static), created_at (date_range).
	filters := table["filters"].([]any)
	assertLen(t, filters, 3, "filters")

	f0 := filters[0].(map[string]any)
	assertEqual(t, f0["field"], "status", "filters[0].field")
	assertEqual(t, f0["type"], "select", "filters[0].type")

	f1 := filters[1].(map[string]any)
	assertEqual(t, f1["field"], "priority", "filters[1].field")
	priorityOpts := f1["options"].([]any)
	assertLen(t, priorityOpts, 3, "priority options")
	assertEqual(t, priorityOpts[0].(map[string]any)["value"], "normal", "priority[0].value")
	assertEqual(t, priorityOpts[1].(map[string]any)["value"], "high", "priority[1].value")
	assertEqual(t, priorityOpts[2].(map[string]any)["value"], "urgent", "priority[2].value")

	f2 := filters[2].(map[string]any)
	assertEqual(t, f2["field"], "created_at", "filters[2].field")
	assertEqual(t, f2["type"], "date_range", "filters[2].type")

	// Table metadata.
	assertEqual(t, table["default_sort"], "created_at", "default_sort")
	assertEqual(t, table["sort_dir"], "desc", "sort_dir")
	assertFloatEqual(t, table["page_size"], 25, "page_size")
	if table["selectable"] != true {
		t.Error("expected selectable = true")
	}
	assertEqual(t, table["data_endpoint"], "/api/pages/orders.list/data", "data_endpoint")

	// Row actions for manager: view_order + cancel_order.
	rowActions := table["row_actions"].([]any)
	assertLen(t, rowActions, 2, "row_actions")

	// Bulk actions for manager: bulk_cancel.
	bulkActions := table["bulk_actions"].([]any)
	assertLen(t, bulkActions, 1, "bulk_actions")

	// Page-level actions for manager: create_order.
	actions, _ := page["actions"].([]any)
	assertLen(t, actions, 1, "page actions")
	assertEqual(t, actions[0].(map[string]any)["id"], "create_order", "action.id")
	assertEqual(t, actions[0].(map[string]any)["type"], "navigate", "action.type")
	assertEqual(t, actions[0].(map[string]any)["navigate_to"], "/orders/new", "action.navigate_to")
}

func TestPageDescriptor_RowActionConfirmationAndConditions(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/pages/orders.list", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var page map[string]any
	h.ParseJSON(resp, &page)

	table := page["table"].(map[string]any)
	rowActions := table["row_actions"].([]any)

	// Find cancel_order action.
	var cancelAction map[string]any
	for _, a := range rowActions {
		action := a.(map[string]any)
		if action["id"] == "cancel_order" {
			cancelAction = action
			break
		}
	}
	if cancelAction == nil {
		t.Fatal("cancel_order row action not found")
	}

	// Confirmation dialog.
	conf, ok := cancelAction["confirmation"].(map[string]any)
	if !ok {
		t.Fatal("expected confirmation on cancel_order")
	}
	assertEqual(t, conf["title"], "Cancel Order", "confirmation.title")
	assertEqual(t, conf["confirm"], "Yes, Cancel", "confirmation.confirm")
	assertEqual(t, conf["cancel"], "No, Keep", "confirmation.cancel")
	assertEqual(t, conf["style"], "danger", "confirmation.style")

	// Conditions (passed through for client-side evaluation since no resource data).
	conditions, _ := cancelAction["conditions"].([]any)
	if len(conditions) == 0 {
		t.Fatal("expected conditions on cancel_order action")
	}
	cond := conditions[0].(map[string]any)
	assertEqual(t, cond["field"], "status", "condition.field")
	assertEqual(t, cond["operator"], "in", "condition.operator")
	assertEqual(t, cond["effect"], "show", "condition.effect")
}

func TestPageDescriptor_ColumnsNotFilteredByCapability(t *testing.T) {
	h := NewTestHarness(t)

	// Viewer has only orders:view — should still see all 5 columns.
	// Column visibility is client-side, not server-side.
	token := h.GenerateToken(ViewerClaims())
	resp := h.GET("/ui/pages/orders.list", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var page map[string]any
	h.ParseJSON(resp, &page)

	table := page["table"].(map[string]any)
	columns := table["columns"].([]any)
	assertLen(t, columns, 5, "viewer columns")
}

func TestPageDescriptor_RowActionsFilteredByCapability(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("manager sees view and cancel", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var page map[string]any
		h.ParseJSON(resp, &page)

		table := page["table"].(map[string]any)
		ids := collectActionIDs(table["row_actions"].([]any))

		assertMapContains(t, ids, "view_order", "manager row actions")
		assertMapContains(t, ids, "cancel_order", "manager row actions")
	})

	t.Run("viewer sees only view", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var page map[string]any
		h.ParseJSON(resp, &page)

		table := page["table"].(map[string]any)
		ids := collectActionIDs(table["row_actions"].([]any))

		assertMapContains(t, ids, "view_order", "viewer row actions")
		assertMapNotContains(t, ids, "cancel_order", "viewer row actions")
	})
}

func TestPageDescriptor_BulkActionsFilteredByCapability(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("manager sees bulk cancel", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var page map[string]any
		h.ParseJSON(resp, &page)

		table := page["table"].(map[string]any)
		bulkActions := table["bulk_actions"].([]any)
		assertLen(t, bulkActions, 1, "manager bulk actions")
		assertEqual(t, bulkActions[0].(map[string]any)["id"], "bulk_cancel", "bulk action id")
	})

	t.Run("viewer sees no bulk actions", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var page map[string]any
		h.ParseJSON(resp, &page)

		table := page["table"].(map[string]any)
		bulkActions, _ := table["bulk_actions"].([]any)
		assertLen(t, bulkActions, 0, "viewer bulk actions")
	})
}

func TestPageDescriptor_PageActionsFilteredByCapability(t *testing.T) {
	h := NewTestHarness(t)

	t.Run("manager sees create action", func(t *testing.T) {
		token := h.GenerateToken(ManagerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var page map[string]any
		h.ParseJSON(resp, &page)

		actions, _ := page["actions"].([]any)
		assertLen(t, actions, 1, "manager page actions")
		assertEqual(t, actions[0].(map[string]any)["id"], "create_order", "action.id")
	})

	t.Run("viewer sees no page actions", func(t *testing.T) {
		token := h.GenerateToken(ViewerClaims())
		resp := h.GET("/ui/pages/orders.list", token)
		h.AssertStatus(t, resp, http.StatusOK)

		var page map[string]any
		h.ParseJSON(resp, &page)

		actions, _ := page["actions"].([]any)
		assertLen(t, actions, 0, "viewer page actions")
	})
}

func TestPageDescriptor_NotFound(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/pages/nonexistent.page", token)
	h.AssertStatus(t, resp, http.StatusNotFound)
}

func TestPageDescriptor_Forbidden(t *testing.T) {
	h := NewTestHarness(t)

	// User with no valid capabilities.
	token := h.GenerateToken(TestClaims{
		SubjectID: "user-nobody",
		TenantID:  "acme-corp",
		Email:     "nobody@acme.example.com",
		Roles:     []string{},
	})
	// orders.list requires orders:view, which this user lacks.
	resp := h.GET("/ui/pages/orders.list", token)
	h.AssertStatus(t, resp, http.StatusForbidden)
}

func TestPageDescriptor_NoInternalMetadata(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/pages/orders.list", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	raw := h.ReadBody(resp)
	bodyStr := string(raw)

	// Verify internal metadata is NOT present in the JSON response.
	prohibited := []string{
		`"operation_id"`,
		`"service_id"`,
		`"capabilities"`,
		`"lookup_id"`,
	}
	for _, p := range prohibited {
		if strings.Contains(bodyStr, p) {
			t.Errorf("response contains internal metadata %s", p)
		}
	}

	// Verify it's still valid JSON with expected structure.
	var page map[string]any
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if page["id"] != "orders.list" {
		t.Errorf("expected valid page descriptor, got id = %v", page["id"])
	}
}

// ==========================================================================
// Page Data Tests
// ==========================================================================

func TestPageData_InvokesCorrectBackendOperation(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	orders := []map[string]any{
		OrderFixture("ord-1", "ORD-001", "pending"),
		OrderFixture("ord-2", "ORD-002", "processing"),
	}
	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture(orders, 2))

	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusOK)

	h.MockBackend("orders-svc").AssertCalled(t, "listOrders", 1)

	// Verify headers forwarded to backend.
	req := h.MockBackend("orders-svc").LastRequest("listOrders")
	if req == nil {
		t.Fatal("expected recorded request")
	}
	if req.Headers.Get("Authorization") == "" {
		t.Error("backend request missing Authorization header")
	}
	if req.Headers.Get("X-Tenant-Id") != "acme-corp" {
		t.Errorf("X-Tenant-Id = %q, want acme-corp", req.Headers.Get("X-Tenant-Id"))
	}
	if req.Headers.Get("X-Request-Subject") != "user-manager" {
		t.Errorf("X-Request-Subject = %q, want user-manager", req.Headers.Get("X-Request-Subject"))
	}
}

func TestPageData_PaginationParamsMapped(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture(nil, 0))

	resp := h.GET("/ui/pages/orders.list/data?page=3&page_size=10", token)
	h.AssertStatus(t, resp, http.StatusOK)

	req := h.MockBackend("orders-svc").LastRequest("listOrders")
	if req == nil {
		t.Fatal("expected recorded request")
	}
	if req.QueryParams["page"] != "3" {
		t.Errorf("backend page = %q, want 3", req.QueryParams["page"])
	}
	if req.QueryParams["page_size"] != "10" {
		t.Errorf("backend page_size = %q, want 10", req.QueryParams["page_size"])
	}
}

func TestPageData_SortParamsForwarded(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture(nil, 0))

	resp := h.GET("/ui/pages/orders.list/data?sort=total_amount&sort_dir=desc", token)
	h.AssertStatus(t, resp, http.StatusOK)

	req := h.MockBackend("orders-svc").LastRequest("listOrders")
	if req == nil {
		t.Fatal("expected recorded request")
	}
	if req.QueryParams["sort"] != "total_amount" {
		t.Errorf("backend sort = %q, want total_amount", req.QueryParams["sort"])
	}
	if req.QueryParams["sort_dir"] != "desc" {
		t.Errorf("backend sort_dir = %q, want desc", req.QueryParams["sort_dir"])
	}
}

func TestPageData_FilterParamsForwarded(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture(nil, 0))

	resp := h.GET("/ui/pages/orders.list/data?filter[status]=pending&filter[priority]=high", token)
	h.AssertStatus(t, resp, http.StatusOK)

	req := h.MockBackend("orders-svc").LastRequest("listOrders")
	if req == nil {
		t.Fatal("expected recorded request")
	}
	if req.QueryParams["status"] != "pending" {
		t.Errorf("backend status = %q, want pending", req.QueryParams["status"])
	}
	if req.QueryParams["priority"] != "high" {
		t.Errorf("backend priority = %q, want high", req.QueryParams["priority"])
	}
}

func TestPageData_ResponseFieldMapping(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Backend returns camelCase field names.
	orders := []map[string]any{
		{
			"orderNumber":  "ORD-001",
			"customerName": "Test Customer",
			"totalAmount":  99.99,
			"status":       "pending",
		},
	}
	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, map[string]any{
			"data":        orders,
			"total_count": 1,
		})

	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var body map[string]any
	h.ParseJSON(resp, &body)

	data := body["data"].(map[string]any)
	items := data["items"].([]any)
	assertLen(t, items, 1, "items")

	// Verify field_map renamed camelCase to snake_case.
	item := items[0].(map[string]any)
	assertEqual(t, item["order_number"], "ORD-001", "order_number (mapped from orderNumber)")
	assertEqual(t, item["customer_name"], "Test Customer", "customer_name (mapped from customerName)")

	// totalAmount should be mapped to total_amount.
	if _, ok := item["total_amount"]; !ok {
		t.Error("expected total_amount (mapped from totalAmount)")
	}

	// Original camelCase keys should no longer be present.
	if _, ok := item["orderNumber"]; ok {
		t.Error("orderNumber should have been renamed to order_number")
	}
	if _, ok := item["customerName"]; ok {
		t.Error("customerName should have been renamed to customer_name")
	}

	// status has no mapping, so it passes through unchanged.
	assertEqual(t, item["status"], "pending", "status (unchanged)")
}

func TestPageData_PaginationMetadata(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	orders := []map[string]any{
		OrderFixture("ord-1", "ORD-001", "pending"),
	}
	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, map[string]any{
			"data":        orders,
			"total_count": 42,
		})

	resp := h.GET("/ui/pages/orders.list/data?page=2&page_size=10", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var body map[string]any
	h.ParseJSON(resp, &body)

	data := body["data"].(map[string]any)
	assertFloatEqual(t, data["total_count"], 42, "total_count")
	assertFloatEqual(t, data["page"], 2, "page")
	assertFloatEqual(t, data["page_size"], 10, "page_size")
}

func TestPageData_DefaultPagination(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWith(200, OrderListFixture(nil, 0))

	// No explicit page/page_size params — should use defaults.
	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusOK)

	req := h.MockBackend("orders-svc").LastRequest("listOrders")
	if req == nil {
		t.Fatal("expected recorded request")
	}
	// Default: page=1, page_size=25.
	if req.QueryParams["page"] != "1" {
		t.Errorf("backend page = %q, want 1", req.QueryParams["page"])
	}
	if req.QueryParams["page_size"] != "25" {
		t.Errorf("backend page_size = %q, want 25", req.QueryParams["page_size"])
	}
}

func TestPageData_BackendConnectionError_502(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	// Close the mock server to simulate an unreachable backend.
	// This produces a "connection refused" error (net.OpError).
	h.MockBackend("orders-svc").Close()

	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusBadGateway)

	var body map[string]any
	h.ParseJSON(resp, &body)

	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response")
	}
	assertEqual(t, errObj["code"], "BACKEND_UNAVAILABLE", "error.code")
}

func TestPageData_BackendTimeout_504(t *testing.T) {
	// Short handler timeout to trigger context cancellation.
	h := NewTestHarness(t, WithHandlerTimeout(500*time.Millisecond))
	token := h.GenerateToken(ManagerClaims())

	// Mock backend delays longer than the handler timeout.
	h.MockBackend("orders-svc").OnOperation("listOrders").
		RespondWithDelay(3*time.Second, 200, OrderListFixture(nil, 0))

	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusGatewayTimeout)

	var body map[string]any
	h.ParseJSON(resp, &body)

	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response")
	}
	assertEqual(t, errObj["code"], "BACKEND_TIMEOUT", "error.code")
}

func TestPageData_NotFound(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/pages/nonexistent.page/data", token)
	h.AssertStatus(t, resp, http.StatusNotFound)
}

func TestPageData_Forbidden(t *testing.T) {
	h := NewTestHarness(t)

	token := h.GenerateToken(TestClaims{
		SubjectID: "user-nobody",
		TenantID:  "acme-corp",
		Email:     "nobody@acme.example.com",
		Roles:     []string{},
	})
	resp := h.GET("/ui/pages/orders.list/data", token)
	h.AssertStatus(t, resp, http.StatusForbidden)
}

// ==========================================================================
// Detail Page Tests
// ==========================================================================

func TestPageDescriptor_DetailPage(t *testing.T) {
	h := NewTestHarness(t)
	token := h.GenerateToken(ManagerClaims())

	resp := h.GET("/ui/pages/orders.detail", token)
	h.AssertStatus(t, resp, http.StatusOK)

	var page map[string]any
	h.ParseJSON(resp, &page)

	assertEqual(t, page["id"], "orders.detail", "id")
	assertEqual(t, page["layout"], "detail", "layout")

	// Breadcrumb should have 3 levels.
	breadcrumb, _ := page["breadcrumb"].([]any)
	assertLen(t, breadcrumb, 3, "breadcrumb")

	// Sections.
	sections, _ := page["sections"].([]any)
	if len(sections) == 0 {
		t.Fatal("expected sections on detail page")
	}
	sec := sections[0].(map[string]any)
	assertEqual(t, sec["id"], "order_info", "section.id")
	assertFloatEqual(t, sec["columns"], 2, "section.columns")

	// Fields in the section.
	fields := sec["fields"].([]any)
	if len(fields) == 0 {
		t.Fatal("expected fields in order_info section")
	}

	// Verify field properties.
	fieldMap := make(map[string]map[string]any)
	for _, f := range fields {
		fd := f.(map[string]any)
		fieldMap[fd["field"].(string)] = fd
	}
	if f, ok := fieldMap["order_number"]; ok {
		if f["read_only"] != true {
			t.Error("order_number should be read_only")
		}
	} else {
		t.Error("expected order_number field")
	}
	if f, ok := fieldMap["total_amount"]; ok {
		assertEqual(t, f["type"], "currency", "total_amount type")
		assertEqual(t, f["format"], "USD", "total_amount format")
	} else {
		t.Error("expected total_amount field")
	}

	// Page-level actions for manager: edit + cancel.
	actions, _ := page["actions"].([]any)
	ids := collectActionIDs(actions)
	assertMapContains(t, ids, "edit_order", "detail actions")
	assertMapContains(t, ids, "cancel_order", "detail actions")
}

// ==========================================================================
// Test helpers
// ==========================================================================

func assertEqual(t *testing.T, got, want any, name string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertFloatEqual(t *testing.T, got any, wantInt int, name string) {
	t.Helper()
	f, ok := got.(float64)
	if !ok {
		t.Errorf("%s: expected float64, got %T (%v)", name, got, got)
		return
	}
	if int(f) != wantInt {
		t.Errorf("%s = %v, want %d", name, got, wantInt)
	}
}

func assertLen(t *testing.T, s []any, expected int, name string) {
	t.Helper()
	if len(s) != expected {
		t.Errorf("%s: got %d items, want %d", name, len(s), expected)
	}
}

func collectActionIDs(actions []any) map[string]bool {
	ids := make(map[string]bool)
	for _, a := range actions {
		action, ok := a.(map[string]any)
		if !ok {
			continue
		}
		id, ok := action["id"].(string)
		if ok {
			ids[id] = true
		}
	}
	return ids
}

func assertMapContains(t *testing.T, m map[string]bool, key, context string) {
	t.Helper()
	if !m[key] {
		t.Errorf("%s: missing %q", context, key)
	}
}

func assertMapNotContains(t *testing.T, m map[string]bool, key, context string) {
	t.Helper()
	if m[key] {
		t.Errorf("%s: should not contain %q", context, key)
	}
}
