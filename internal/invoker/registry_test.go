package invoker

import (
	"context"
	"testing"

	"github.com/pitabwire/thesa/model"
)

type mockInvoker struct {
	supportType string
	result      model.InvocationResult
	err         error
}

func (m *mockInvoker) Supports(binding model.OperationBinding) bool {
	return binding.Type == m.supportType
}

func (m *mockInvoker) Invoke(_ context.Context, _ *model.RequestContext, _ model.OperationBinding, _ model.InvocationInput) (model.InvocationResult, error) {
	return m.result, m.err
}

func TestRegistry_Invoke_dispatches(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockInvoker{
		supportType: "openapi",
		result:      model.InvocationResult{StatusCode: 200, Body: map[string]any{"ok": true}},
	})
	r.Register(&mockInvoker{
		supportType: "sdk",
		result:      model.InvocationResult{StatusCode: 201},
	})

	result, err := r.Invoke(context.Background(), &model.RequestContext{},
		model.OperationBinding{Type: "openapi"}, model.InvocationInput{})
	if err != nil {
		t.Fatalf("Invoke(openapi) error = %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}

	result, err = r.Invoke(context.Background(), &model.RequestContext{},
		model.OperationBinding{Type: "sdk"}, model.InvocationInput{})
	if err != nil {
		t.Fatalf("Invoke(sdk) error = %v", err)
	}
	if result.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", result.StatusCode)
	}
}

func TestRegistry_Invoke_no_support(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockInvoker{supportType: "openapi"})

	_, err := r.Invoke(context.Background(), &model.RequestContext{},
		model.OperationBinding{Type: "unknown"}, model.InvocationInput{})
	if err == nil {
		t.Fatal("Invoke(unknown) should return error")
	}
}

func TestRegistry_Invoke_empty(t *testing.T) {
	r := NewRegistry()
	_, err := r.Invoke(context.Background(), &model.RequestContext{},
		model.OperationBinding{Type: "openapi"}, model.InvocationInput{})
	if err == nil {
		t.Fatal("Invoke on empty registry should return error")
	}
}
