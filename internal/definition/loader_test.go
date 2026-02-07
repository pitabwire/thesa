package definition

import (
	"testing"
)

func TestLoader_LoadFile(t *testing.T) {
	l := NewLoader()
	def, err := l.LoadFile("testdata/orders/definition.yaml")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if def.Domain != "orders" {
		t.Errorf("Domain = %q, want orders", def.Domain)
	}
	if def.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", def.Version)
	}
	if def.Navigation.Label != "Orders" {
		t.Errorf("Navigation.Label = %q, want Orders", def.Navigation.Label)
	}
	if len(def.Navigation.Children) != 1 {
		t.Fatalf("Navigation.Children = %d, want 1", len(def.Navigation.Children))
	}
	if def.Navigation.Children[0].PageID != "orders.list" {
		t.Errorf("Child.PageID = %q, want orders.list", def.Navigation.Children[0].PageID)
	}
	if len(def.Pages) != 1 {
		t.Fatalf("Pages = %d, want 1", len(def.Pages))
	}
	if def.Pages[0].ID != "orders.list" {
		t.Errorf("Page.ID = %q, want orders.list", def.Pages[0].ID)
	}
	if len(def.Commands) != 1 {
		t.Fatalf("Commands = %d, want 1", len(def.Commands))
	}
	if def.Commands[0].ID != "orders.update" {
		t.Errorf("Command.ID = %q, want orders.update", def.Commands[0].ID)
	}
	if def.Checksum == "" {
		t.Error("Checksum should not be empty")
	}
	if def.SourceFile != "testdata/orders/definition.yaml" {
		t.Errorf("SourceFile = %q", def.SourceFile)
	}
}

func TestLoader_LoadFile_not_found(t *testing.T) {
	l := NewLoader()
	_, err := l.LoadFile("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("LoadFile() with missing file should return error")
	}
}

func TestLoader_LoadFile_invalid_yaml(t *testing.T) {
	l := NewLoader()
	_, err := l.LoadFile("testdata/invalid/bad.yaml")
	if err == nil {
		t.Fatal("LoadFile() with invalid YAML should return error")
	}
}

func TestLoader_LoadAll(t *testing.T) {
	l := NewLoader()
	defs, err := l.LoadAll([]string{"testdata/orders"})
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("LoadAll() returned %d definitions, want 1", len(defs))
	}
	if defs[0].Domain != "orders" {
		t.Errorf("Domain = %q, want orders", defs[0].Domain)
	}
}

func TestLoader_LoadAll_invalid_dir(t *testing.T) {
	l := NewLoader()
	_, err := l.LoadAll([]string{"testdata/nonexistent"})
	if err == nil {
		t.Fatal("LoadAll() with missing directory should return error")
	}
}

func TestLoader_LoadAll_invalid_yaml(t *testing.T) {
	l := NewLoader()
	_, err := l.LoadAll([]string{"testdata/invalid"})
	if err == nil {
		t.Fatal("LoadAll() with invalid YAML should return error")
	}
}

func TestLoader_Checksum_deterministic(t *testing.T) {
	l := NewLoader()
	def1, _ := l.LoadFile("testdata/orders/definition.yaml")
	def2, _ := l.LoadFile("testdata/orders/definition.yaml")
	if def1.Checksum != def2.Checksum {
		t.Error("Checksum should be deterministic")
	}
}
