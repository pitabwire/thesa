package invoker

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/pitabwire/thesa/model"
)

// mockSDKHandler is a test double for SDKHandler.
type mockSDKHandler struct {
	name      string
	invokeFn func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error)
}

func (m *mockSDKHandler) Name() string { return m.name }

func (m *mockSDKHandler) Invoke(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, rctx, input)
	}
	return model.InvocationResult{StatusCode: http.StatusOK}, nil
}

// --- SDKHandlerRegistry ---

func TestSDKHandlerRegistry_RegisterAndGet(t *testing.T) {
	r := NewSDKHandlerRegistry()
	h := &mockSDKHandler{name: "test-handler"}

	r.Register("test-handler", h)

	got, ok := r.Get("test-handler")
	if !ok {
		t.Fatal("Get(test-handler) returned false")
	}
	if got.Name() != "test-handler" {
		t.Errorf("Name() = %q, want test-handler", got.Name())
	}
}

func TestSDKHandlerRegistry_GetNotFound(t *testing.T) {
	r := NewSDKHandlerRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) = true, want false")
	}
}

func TestSDKHandlerRegistry_RegisterDuplicatePanics(t *testing.T) {
	r := NewSDKHandlerRegistry()
	h := &mockSDKHandler{name: "dup-handler"}
	r.Register("dup-handler", h)

	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	r.Register("dup-handler", h) // should panic
}

func TestSDKHandlerRegistry_Names(t *testing.T) {
	r := NewSDKHandlerRegistry()
	r.Register("bravo", &mockSDKHandler{name: "bravo"})
	r.Register("alpha", &mockSDKHandler{name: "alpha"})
	r.Register("charlie", &mockSDKHandler{name: "charlie"})

	names := r.Names()
	if len(names) != 3 {
		t.Fatalf("len(Names()) = %d, want 3", len(names))
	}
	// Should be sorted.
	if names[0] != "alpha" || names[1] != "bravo" || names[2] != "charlie" {
		t.Errorf("Names() = %v, want [alpha bravo charlie]", names)
	}
}

func TestSDKHandlerRegistry_NamesEmpty(t *testing.T) {
	r := NewSDKHandlerRegistry()
	names := r.Names()
	if len(names) != 0 {
		t.Errorf("Names() = %v, want empty", names)
	}
}

// --- SDKOperationInvoker ---

func TestSDKOperationInvoker_Supports(t *testing.T) {
	inv := NewSDKOperationInvoker(NewSDKHandlerRegistry())

	if !inv.Supports(model.OperationBinding{Type: "sdk"}) {
		t.Error("Supports(sdk) = false, want true")
	}
	if inv.Supports(model.OperationBinding{Type: "openapi"}) {
		t.Error("Supports(openapi) = true, want false")
	}
	if inv.Supports(model.OperationBinding{Type: ""}) {
		t.Error("Supports('') = true, want false")
	}
}

func TestSDKOperationInvoker_Invoke_success(t *testing.T) {
	registry := NewSDKHandlerRegistry()
	registry.Register("my-handler", &mockSDKHandler{
		name: "my-handler",
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{
				StatusCode: http.StatusOK,
				Body:       map[string]any{"result": "success"},
			}, nil
		},
	})

	inv := NewSDKOperationInvoker(registry)

	result, err := inv.Invoke(
		context.Background(),
		&model.RequestContext{SubjectID: "user-1"},
		model.OperationBinding{Type: "sdk", Handler: "my-handler"},
		model.InvocationInput{Body: map[string]any{"key": "value"}},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	body := result.Body.(map[string]any)
	if body["result"] != "success" {
		t.Errorf("Body.result = %v, want success", body["result"])
	}
}

func TestSDKOperationInvoker_Invoke_handlerNotFound(t *testing.T) {
	inv := NewSDKOperationInvoker(NewSDKHandlerRegistry())

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "sdk", Handler: "missing"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for missing handler")
	}
}

func TestSDKOperationInvoker_Invoke_handlerError(t *testing.T) {
	registry := NewSDKHandlerRegistry()
	registry.Register("failing-handler", &mockSDKHandler{
		name: "failing-handler",
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{}, fmt.Errorf("handler exploded")
		},
	})

	inv := NewSDKOperationInvoker(registry)

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "sdk", Handler: "failing-handler"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error from failing handler")
	}
	if err.Error() != "handler exploded" {
		t.Errorf("error = %q, want %q", err.Error(), "handler exploded")
	}
}

func TestSDKOperationInvoker_Invoke_receivesRequestContext(t *testing.T) {
	registry := NewSDKHandlerRegistry()
	registry.Register("ctx-check", &mockSDKHandler{
		name: "ctx-check",
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
			if rctx == nil {
				return model.InvocationResult{}, fmt.Errorf("rctx is nil")
			}
			if rctx.SubjectID != "user-42" {
				return model.InvocationResult{}, fmt.Errorf("SubjectID = %q, want user-42", rctx.SubjectID)
			}
			if rctx.TenantID != "tenant-7" {
				return model.InvocationResult{}, fmt.Errorf("TenantID = %q, want tenant-7", rctx.TenantID)
			}
			return model.InvocationResult{StatusCode: http.StatusOK}, nil
		},
	})

	inv := NewSDKOperationInvoker(registry)

	_, err := inv.Invoke(
		context.Background(),
		&model.RequestContext{SubjectID: "user-42", TenantID: "tenant-7"},
		model.OperationBinding{Type: "sdk", Handler: "ctx-check"},
		model.InvocationInput{},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

func TestSDKOperationInvoker_Invoke_receivesInput(t *testing.T) {
	registry := NewSDKHandlerRegistry()
	registry.Register("input-check", &mockSDKHandler{
		name: "input-check",
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
			if input.PathParams["id"] != "123" {
				return model.InvocationResult{}, fmt.Errorf("PathParams.id = %q, want 123", input.PathParams["id"])
			}
			body := input.Body.(map[string]any)
			if body["name"] != "test" {
				return model.InvocationResult{}, fmt.Errorf("Body.name = %v, want test", body["name"])
			}
			return model.InvocationResult{StatusCode: http.StatusOK}, nil
		},
	})

	inv := NewSDKOperationInvoker(registry)

	_, err := inv.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "sdk", Handler: "input-check"},
		model.InvocationInput{
			PathParams: map[string]string{"id": "123"},
			Body:       map[string]any{"name": "test"},
		},
	)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
}

func TestSDKOperationInvoker_Invoke_respectsContext(t *testing.T) {
	registry := NewSDKHandlerRegistry()
	registry.Register("ctx-handler", &mockSDKHandler{
		name: "ctx-handler",
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
			if err := ctx.Err(); err != nil {
				return model.InvocationResult{}, err
			}
			return model.InvocationResult{StatusCode: http.StatusOK}, nil
		},
	})

	inv := NewSDKOperationInvoker(registry)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before invoking.

	_, err := inv.Invoke(
		ctx,
		nil,
		model.OperationBinding{Type: "sdk", Handler: "ctx-handler"},
		model.InvocationInput{},
	)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- Integration with InvokerRegistry ---

func TestSDKOperationInvoker_worksWithRegistry(t *testing.T) {
	handlerRegistry := NewSDKHandlerRegistry()
	handlerRegistry.Register("echo", &mockSDKHandler{
		name: "echo",
		invokeFn: func(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error) {
			return model.InvocationResult{
				StatusCode: http.StatusOK,
				Body:       input.Body,
			}, nil
		},
	})

	sdkInvoker := NewSDKOperationInvoker(handlerRegistry)

	invokerRegistry := NewRegistry()
	invokerRegistry.Register(sdkInvoker)

	result, err := invokerRegistry.Invoke(
		context.Background(),
		nil,
		model.OperationBinding{Type: "sdk", Handler: "echo"},
		model.InvocationInput{Body: map[string]any{"ping": "pong"}},
	)
	if err != nil {
		t.Fatalf("Registry.Invoke error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	body := result.Body.(map[string]any)
	if body["ping"] != "pong" {
		t.Errorf("Body.ping = %v, want pong", body["ping"])
	}
}
