package metadata

import (
	"context"
	"fmt"
	"testing"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/model"
)

func testDomains() []model.DomainDefinition {
	return []model.DomainDefinition{
		{
			Domain: "orders",
			Navigation: model.NavigationDefinition{
				Label:        "Orders",
				Icon:         "shopping-cart",
				Order:        2,
				Capabilities: []string{"orders:view"},
				Children: []model.NavigationChildDefinition{
					{
						Label:        "Order List",
						Icon:         "list",
						Route:        "/orders",
						PageID:       "orders-list",
						Capabilities: []string{"orders:list:view"},
						Order:        1,
					},
					{
						Label:        "Pending Approval",
						Icon:         "clock",
						Route:        "/orders/pending",
						PageID:       "orders-pending",
						Capabilities: []string{"orders:pending:view"},
						Order:        2,
						Badge: &model.BadgeDefinition{
							OperationID: "countPendingOrders",
							Field:       "count",
							Style:       "warning",
						},
					},
				},
			},
		},
		{
			Domain: "users",
			Navigation: model.NavigationDefinition{
				Label:        "Users",
				Icon:         "users",
				Order:        1,
				Capabilities: []string{"users:view"},
				Children: []model.NavigationChildDefinition{
					{
						Label:        "User List",
						Icon:         "list",
						Route:        "/users",
						PageID:       "users-list",
						Capabilities: []string{},
						Order:        1,
					},
				},
			},
		},
		{
			Domain: "admin",
			Navigation: model.NavigationDefinition{
				Label:        "Admin",
				Icon:         "settings",
				Order:        3,
				Capabilities: []string{"admin:access"},
				Children: []model.NavigationChildDefinition{
					{
						Label:        "Settings",
						Route:        "/admin/settings",
						PageID:       "admin-settings",
						Capabilities: []string{"admin:settings"},
						Order:        1,
					},
					{
						Label:        "Audit Log",
						Route:        "/admin/audit",
						PageID:       "admin-audit",
						Capabilities: []string{"admin:audit"},
						Order:        2,
					},
				},
			},
		},
	}
}

func TestMenuProvider_GetMenu_allCapabilities(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	caps := model.CapabilitySet{
		"orders:view":         true,
		"orders:list:view":    true,
		"orders:pending:view": true,
		"users:view":          true,
		"admin:access":        true,
		"admin:settings":      true,
		"admin:audit":         true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	// Should have 3 domains, sorted by order: users(1), orders(2), admin(3).
	if len(tree.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(tree.Items))
	}
	if tree.Items[0].ID != "users" {
		t.Errorf("Items[0].ID = %q, want users (order 1)", tree.Items[0].ID)
	}
	if tree.Items[1].ID != "orders" {
		t.Errorf("Items[1].ID = %q, want orders (order 2)", tree.Items[1].ID)
	}
	if tree.Items[2].ID != "admin" {
		t.Errorf("Items[2].ID = %q, want admin (order 3)", tree.Items[2].ID)
	}
}

func TestMenuProvider_GetMenu_filtersDomainsByCapability(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	// Only has users:view, not orders:view or admin:access.
	caps := model.CapabilitySet{
		"users:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	if len(tree.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(tree.Items))
	}
	if tree.Items[0].ID != "users" {
		t.Errorf("Items[0].ID = %q, want users", tree.Items[0].ID)
	}
}

func TestMenuProvider_GetMenu_filtersChildrenByCapability(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	// Has admin domain access and audit but not settings.
	caps := model.CapabilitySet{
		"admin:access": true,
		"admin:audit":  true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	if len(tree.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1 (admin)", len(tree.Items))
	}
	admin := tree.Items[0]
	if len(admin.Children) != 1 {
		t.Fatalf("len(admin.Children) = %d, want 1 (audit only)", len(admin.Children))
	}
	if admin.Children[0].ID != "admin-audit" {
		t.Errorf("admin.Children[0].ID = %q, want admin-audit", admin.Children[0].ID)
	}
}

func TestMenuProvider_GetMenu_childrenSortedByOrder(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	caps := model.CapabilitySet{
		"orders:view":         true,
		"orders:list:view":    true,
		"orders:pending:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	if len(tree.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(tree.Items))
	}
	orders := tree.Items[0]
	if len(orders.Children) != 2 {
		t.Fatalf("len(orders.Children) = %d, want 2", len(orders.Children))
	}
	// Order 1 first, order 2 second.
	if orders.Children[0].ID != "orders-list" {
		t.Errorf("Children[0].ID = %q, want orders-list (order 1)", orders.Children[0].ID)
	}
	if orders.Children[1].ID != "orders-pending" {
		t.Errorf("Children[1].ID = %q, want orders-pending (order 2)", orders.Children[1].ID)
	}
}

func TestMenuProvider_GetMenu_noCapabilitiesRequiredOnChild(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	// Users domain has no capabilities on children.
	caps := model.CapabilitySet{
		"users:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	if len(tree.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(tree.Items))
	}
	users := tree.Items[0]
	// Child has empty capabilities → should be included.
	if len(users.Children) != 1 {
		t.Fatalf("len(users.Children) = %d, want 1", len(users.Children))
	}
	if users.Children[0].Label != "User List" {
		t.Errorf("Children[0].Label = %q, want User List", users.Children[0].Label)
	}
}

func TestMenuProvider_GetMenu_emptyCapabilities(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	tree, err := provider.GetMenu(context.Background(), nil, model.CapabilitySet{})
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	// No capabilities → no domains visible.
	if len(tree.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0 (no capabilities)", len(tree.Items))
	}
}

func TestMenuProvider_GetMenu_wildcardCapability(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	// Wildcard grants all capabilities.
	caps := model.CapabilitySet{
		"*": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	if len(tree.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3 (wildcard grants all)", len(tree.Items))
	}
}

func TestMenuProvider_GetMenu_noDomainsRequiringCapabilities(t *testing.T) {
	// Domain with no capability requirements.
	domains := []model.DomainDefinition{
		{
			Domain: "public",
			Navigation: model.NavigationDefinition{
				Label: "Public",
				Order: 1,
				Children: []model.NavigationChildDefinition{
					{Label: "Home", Route: "/", PageID: "home", Order: 1},
				},
			},
		},
	}
	reg := definition.NewRegistry(domains)
	provider := NewMenuProvider(reg, nil)

	tree, err := provider.GetMenu(context.Background(), nil, model.CapabilitySet{})
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	// No capabilities required → domain should appear.
	if len(tree.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(tree.Items))
	}
	if tree.Items[0].Children[0].Label != "Home" {
		t.Errorf("child label = %q, want Home", tree.Items[0].Children[0].Label)
	}
}

func TestMenuProvider_GetMenu_nodeProperties(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil)

	caps := model.CapabilitySet{
		"orders:view":      true,
		"orders:list:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	if len(tree.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(tree.Items))
	}

	orders := tree.Items[0]
	if orders.Label != "Orders" {
		t.Errorf("Label = %q, want Orders", orders.Label)
	}
	if orders.Icon != "shopping-cart" {
		t.Errorf("Icon = %q, want shopping-cart", orders.Icon)
	}

	if len(orders.Children) != 1 {
		t.Fatalf("len(Children) = %d, want 1", len(orders.Children))
	}
	child := orders.Children[0]
	if child.Label != "Order List" {
		t.Errorf("child.Label = %q, want Order List", child.Label)
	}
	if child.Route != "/orders" {
		t.Errorf("child.Route = %q, want /orders", child.Route)
	}
	if child.Icon != "list" {
		t.Errorf("child.Icon = %q, want list", child.Icon)
	}
}

// --- Badge resolution ---

// mockInvoker is a test double for invoker.Registry.
type mockInvokerForMenu struct {
	invokeFn func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error)
}

func (m *mockInvokerForMenu) Supports(binding model.OperationBinding) bool {
	return true
}

func (m *mockInvokerForMenu) Invoke(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, rctx, binding, input)
	}
	return model.InvocationResult{}, nil
}

func TestMenuProvider_GetMenu_badgeResolved(t *testing.T) {
	reg := definition.NewRegistry(testDomains())

	// Create invoker registry with mock that returns badge count.
	invokerReg := invoker.NewRegistry()
	invokerReg.Register(&mockInvokerForMenu{
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
			if binding.OperationID == "countPendingOrders" {
				return model.InvocationResult{
					StatusCode: 200,
					Body:       map[string]any{"count": float64(5)},
				}, nil
			}
			return model.InvocationResult{}, nil
		},
	})

	provider := NewMenuProvider(reg, invokerReg)

	caps := model.CapabilitySet{
		"orders:view":         true,
		"orders:list:view":    true,
		"orders:pending:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	orders := tree.Items[0]
	pending := orders.Children[1] // order 2 = pending
	if pending.Badge == nil {
		t.Fatal("Badge is nil, want badge with count 5")
	}
	if pending.Badge.Count != 5 {
		t.Errorf("Badge.Count = %d, want 5", pending.Badge.Count)
	}
	if pending.Badge.Style != "warning" {
		t.Errorf("Badge.Style = %q, want warning", pending.Badge.Style)
	}
}

func TestMenuProvider_GetMenu_badgeFailureOmitsBadge(t *testing.T) {
	reg := definition.NewRegistry(testDomains())

	invokerReg := invoker.NewRegistry()
	invokerReg.Register(&mockInvokerForMenu{
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{}, fmt.Errorf("backend error")
		},
	})

	provider := NewMenuProvider(reg, invokerReg)

	caps := model.CapabilitySet{
		"orders:view":         true,
		"orders:list:view":    true,
		"orders:pending:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	orders := tree.Items[0]
	pending := orders.Children[1]
	// Badge should be omitted on failure, not error.
	if pending.Badge != nil {
		t.Errorf("Badge = %+v, want nil (failure should omit badge)", pending.Badge)
	}
}

func TestMenuProvider_GetMenu_badgeZeroCountOmitted(t *testing.T) {
	reg := definition.NewRegistry(testDomains())

	invokerReg := invoker.NewRegistry()
	invokerReg.Register(&mockInvokerForMenu{
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{
				StatusCode: 200,
				Body:       map[string]any{"count": float64(0)},
			}, nil
		},
	})

	provider := NewMenuProvider(reg, invokerReg)

	caps := model.CapabilitySet{
		"orders:view":         true,
		"orders:list:view":    true,
		"orders:pending:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	orders := tree.Items[0]
	pending := orders.Children[1]
	// Zero count → badge omitted.
	if pending.Badge != nil {
		t.Errorf("Badge = %+v, want nil (zero count)", pending.Badge)
	}
}

func TestMenuProvider_GetMenu_nilInvokerNoBadge(t *testing.T) {
	reg := definition.NewRegistry(testDomains())
	provider := NewMenuProvider(reg, nil) // nil invoker

	caps := model.CapabilitySet{
		"orders:view":         true,
		"orders:list:view":    true,
		"orders:pending:view": true,
	}

	tree, err := provider.GetMenu(context.Background(), nil, caps)
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}

	orders := tree.Items[0]
	pending := orders.Children[1]
	// No invoker → badge is nil.
	if pending.Badge != nil {
		t.Errorf("Badge = %+v, want nil (nil invoker)", pending.Badge)
	}
}

// --- extractBadgeCount ---

func TestExtractBadgeCount(t *testing.T) {
	tests := []struct {
		name  string
		body  any
		field string
		want  int
	}{
		{"nil body", nil, "count", 0},
		{"empty field", map[string]any{"count": float64(5)}, "", 0},
		{"float64", map[string]any{"count": float64(42)}, "count", 42},
		{"int", map[string]any{"count": 7}, "count", 7},
		{"int64", map[string]any{"count": int64(99)}, "count", 99},
		{"string value", map[string]any{"count": "abc"}, "count", 0},
		{"missing field", map[string]any{"other": float64(5)}, "count", 0},
		{"non-map body", "string", "count", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBadgeCount(tt.body, tt.field)
			if got != tt.want {
				t.Errorf("extractBadgeCount(%v, %q) = %d, want %d", tt.body, tt.field, got, tt.want)
			}
		})
	}
}

func TestMenuProvider_GetMenu_emptyRegistry(t *testing.T) {
	reg := definition.NewRegistry(nil)
	provider := NewMenuProvider(reg, nil)

	tree, err := provider.GetMenu(context.Background(), nil, model.CapabilitySet{"*": true})
	if err != nil {
		t.Fatalf("GetMenu error: %v", err)
	}
	if len(tree.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(tree.Items))
	}
}
