package definition

import (
	"sync"
	"testing"

	"github.com/pitabwire/thesa/model"
)

func testDefs() []model.DomainDefinition {
	return []model.DomainDefinition{
		{
			Domain:   "orders",
			Version:  "1.0.0",
			Checksum: "abc123",
			Pages: []model.PageDefinition{
				{ID: "orders.list", Title: "Orders", Layout: "list"},
				{ID: "orders.detail", Title: "Order Detail", Layout: "detail"},
			},
			Forms: []model.FormDefinition{
				{ID: "orders.edit_form", Title: "Edit Order"},
			},
			Commands: []model.CommandDefinition{
				{ID: "orders.update"},
				{ID: "orders.cancel"},
			},
			Workflows: []model.WorkflowDefinition{
				{ID: "orders.approval", Name: "Order Approval"},
			},
			Searches: []model.SearchDefinition{
				{ID: "orders.search", Domain: "orders"},
			},
			Lookups: []model.LookupDefinition{
				{ID: "orders.statuses"},
			},
		},
		{
			Domain:   "inventory",
			Version:  "1.0.0",
			Checksum: "def456",
			Pages: []model.PageDefinition{
				{ID: "inventory.list", Title: "Inventory", Layout: "list"},
			},
		},
	}
}

func TestRegistry_GetDomain(t *testing.T) {
	r := NewRegistry(testDefs())

	d, ok := r.GetDomain("orders")
	if !ok {
		t.Fatal("GetDomain(orders) not found")
	}
	if d.Domain != "orders" {
		t.Errorf("Domain = %q, want orders", d.Domain)
	}

	_, ok = r.GetDomain("unknown")
	if ok {
		t.Error("GetDomain(unknown) should return false")
	}
}

func TestRegistry_GetPage(t *testing.T) {
	r := NewRegistry(testDefs())

	p, ok := r.GetPage("orders.list")
	if !ok {
		t.Fatal("GetPage(orders.list) not found")
	}
	if p.Title != "Orders" {
		t.Errorf("Title = %q, want Orders", p.Title)
	}

	_, ok = r.GetPage("nonexistent")
	if ok {
		t.Error("GetPage(nonexistent) should return false")
	}
}

func TestRegistry_GetForm(t *testing.T) {
	r := NewRegistry(testDefs())
	f, ok := r.GetForm("orders.edit_form")
	if !ok {
		t.Fatal("GetForm(orders.edit_form) not found")
	}
	if f.Title != "Edit Order" {
		t.Errorf("Title = %q", f.Title)
	}
}

func TestRegistry_GetCommand(t *testing.T) {
	r := NewRegistry(testDefs())
	c, ok := r.GetCommand("orders.cancel")
	if !ok {
		t.Fatal("GetCommand(orders.cancel) not found")
	}
	if c.ID != "orders.cancel" {
		t.Errorf("ID = %q", c.ID)
	}
}

func TestRegistry_GetWorkflow(t *testing.T) {
	r := NewRegistry(testDefs())
	w, ok := r.GetWorkflow("orders.approval")
	if !ok {
		t.Fatal("GetWorkflow(orders.approval) not found")
	}
	if w.Name != "Order Approval" {
		t.Errorf("Name = %q", w.Name)
	}
}

func TestRegistry_GetSearch(t *testing.T) {
	r := NewRegistry(testDefs())
	s, ok := r.GetSearch("orders.search")
	if !ok {
		t.Fatal("GetSearch(orders.search) not found")
	}
	if s.Domain != "orders" {
		t.Errorf("Domain = %q", s.Domain)
	}
}

func TestRegistry_GetLookup(t *testing.T) {
	r := NewRegistry(testDefs())
	l, ok := r.GetLookup("orders.statuses")
	if !ok {
		t.Fatal("GetLookup(orders.statuses) not found")
	}
	if l.ID != "orders.statuses" {
		t.Errorf("ID = %q", l.ID)
	}
}

func TestRegistry_AllDomains(t *testing.T) {
	r := NewRegistry(testDefs())
	all := r.AllDomains()
	if len(all) != 2 {
		t.Errorf("AllDomains() returned %d, want 2", len(all))
	}
}

func TestRegistry_AllSearches(t *testing.T) {
	r := NewRegistry(testDefs())
	all := r.AllSearches()
	if len(all) != 1 {
		t.Errorf("AllSearches() returned %d, want 1", len(all))
	}
}

func TestRegistry_Checksum(t *testing.T) {
	r := NewRegistry(testDefs())
	cs := r.Checksum()
	if cs == "" {
		t.Error("Checksum should not be empty")
	}
}

func TestRegistry_Replace(t *testing.T) {
	r := NewRegistry(testDefs())

	// Initially has orders page.
	_, ok := r.GetPage("orders.list")
	if !ok {
		t.Fatal("before replace: orders.list not found")
	}

	// Replace with empty.
	r.Replace(nil)

	_, ok = r.GetPage("orders.list")
	if ok {
		t.Error("after replace with nil: orders.list should not be found")
	}
}

func TestRegistry_ConcurrentReads(t *testing.T) {
	r := NewRegistry(testDefs())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.GetPage("orders.list")
			r.GetCommand("orders.update")
			r.AllDomains()
			r.Checksum()
		}()
	}
	wg.Wait()
}

func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	r := NewRegistry(testDefs())

	var wg sync.WaitGroup

	// Concurrent readers.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				r.GetPage("orders.list")
				r.AllDomains()
			}
		}()
	}

	// Concurrent writer.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 10; j++ {
			r.Replace(testDefs())
		}
	}()

	wg.Wait()
}
