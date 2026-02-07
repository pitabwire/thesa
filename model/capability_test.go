package model

import "testing"

func TestCapabilitySet_Has_exact(t *testing.T) {
	cs := CapabilitySet{
		"orders:list:view":   true,
		"orders:detail:view": true,
	}
	if !cs.Has("orders:list:view") {
		t.Error("Has(orders:list:view) = false, want true")
	}
	if cs.Has("orders:cancel:execute") {
		t.Error("Has(orders:cancel:execute) = true, want false")
	}
}

func TestCapabilitySet_Has_wildcard_star(t *testing.T) {
	cs := CapabilitySet{"*": true}
	if !cs.Has("orders:list:view") {
		t.Error("wildcard * should match orders:list:view")
	}
	if !cs.Has("anything") {
		t.Error("wildcard * should match anything")
	}
}

func TestCapabilitySet_Has_wildcard_namespace(t *testing.T) {
	cs := CapabilitySet{"orders:*": true}
	if !cs.Has("orders:list:view") {
		t.Error("orders:* should match orders:list:view")
	}
	if !cs.Has("orders:cancel:execute") {
		t.Error("orders:* should match orders:cancel:execute")
	}
	if cs.Has("inventory:list:view") {
		t.Error("orders:* should not match inventory:list:view")
	}
}

func TestCapabilitySet_Has_wildcard_resource(t *testing.T) {
	cs := CapabilitySet{"orders:list:*": true}
	if !cs.Has("orders:list:view") {
		t.Error("orders:list:* should match orders:list:view")
	}
	if !cs.Has("orders:list:export") {
		t.Error("orders:list:* should match orders:list:export")
	}
	if cs.Has("orders:detail:view") {
		t.Error("orders:list:* should not match orders:detail:view")
	}
}

func TestCapabilitySet_Has_empty(t *testing.T) {
	cs := CapabilitySet{}
	if cs.Has("orders:list:view") {
		t.Error("empty set should not match anything")
	}
}

func TestCapabilitySet_Has_nil(t *testing.T) {
	var cs CapabilitySet
	if cs.Has("orders:list:view") {
		t.Error("nil set should not match anything")
	}
}

func TestCapabilitySet_HasAll(t *testing.T) {
	cs := CapabilitySet{
		"orders:list:view":   true,
		"orders:detail:view": true,
	}
	if !cs.HasAll("orders:list:view", "orders:detail:view") {
		t.Error("HasAll should be true when all present")
	}
	if cs.HasAll("orders:list:view", "orders:cancel:execute") {
		t.Error("HasAll should be false when one missing")
	}
}

func TestCapabilitySet_HasAll_empty(t *testing.T) {
	cs := CapabilitySet{"orders:list:view": true}
	if !cs.HasAll() {
		t.Error("HasAll with no args should be true")
	}
}

func TestCapabilitySet_HasAll_wildcard(t *testing.T) {
	cs := CapabilitySet{"orders:*": true}
	if !cs.HasAll("orders:list:view", "orders:detail:edit") {
		t.Error("HasAll with wildcard should match all under namespace")
	}
}

func TestCapabilitySet_HasAny(t *testing.T) {
	cs := CapabilitySet{
		"orders:list:view": true,
	}
	if !cs.HasAny("orders:cancel:execute", "orders:list:view") {
		t.Error("HasAny should be true when at least one present")
	}
	if cs.HasAny("orders:cancel:execute", "inventory:list:view") {
		t.Error("HasAny should be false when none present")
	}
}

func TestCapabilitySet_HasAny_empty(t *testing.T) {
	cs := CapabilitySet{"orders:list:view": true}
	if cs.HasAny() {
		t.Error("HasAny with no args should be false")
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		cap     string
		want    bool
	}{
		{"*", "orders:list:view", true},
		{"*", "anything", true},
		{"orders:*", "orders:list:view", true},
		{"orders:*", "orders:cancel:execute", true},
		{"orders:*", "inventory:list:view", false},
		{"orders:list:*", "orders:list:view", true},
		{"orders:list:*", "orders:list:export", true},
		{"orders:list:*", "orders:detail:view", false},
		{"orders:list:view", "orders:list:view", false}, // exact match handled by map lookup, not wildcard
		{"orders:list", "orders:list:view", false},       // no wildcard suffix
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.cap, func(t *testing.T) {
			if got := matchWildcard(tt.pattern, tt.cap); got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.cap, got, tt.want)
			}
		})
	}
}
