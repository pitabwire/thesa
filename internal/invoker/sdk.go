package invoker

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/pitabwire/thesa/model"
)

// SDKHandler is the interface for typed backend handlers that are registered
// at startup and invoked by name through definition bindings.
type SDKHandler interface {
	// Name returns the unique handler name used in definition bindings.
	Name() string
	// Invoke executes the handler with the given request context and input.
	Invoke(ctx context.Context, rctx *model.RequestContext, input model.InvocationInput) (model.InvocationResult, error)
}

// SDKHandlerRegistry stores named SDK handlers and provides lookup by name.
// It is safe for concurrent use after initial registration.
type SDKHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]SDKHandler
}

// NewSDKHandlerRegistry creates a new empty handler registry.
func NewSDKHandlerRegistry() *SDKHandlerRegistry {
	return &SDKHandlerRegistry{
		handlers: make(map[string]SDKHandler),
	}
}

// Register adds a handler to the registry under its Name(). Panics if a
// handler with the same name is already registered, since this indicates
// a wiring mistake at startup.
func (r *SDKHandlerRegistry) Register(name string, handler SDKHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[name]; exists {
		panic(fmt.Sprintf("invoker: SDK handler %q already registered", name))
	}
	r.handlers[name] = handler
}

// Get returns the handler registered under the given name, or false if not found.
func (r *SDKHandlerRegistry) Get(name string) (SDKHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// Names returns all registered handler names, sorted alphabetically.
func (r *SDKHandlerRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SDKOperationInvoker dispatches invocations to registered SDK handlers
// based on the binding's Handler field. It implements model.OperationInvoker.
type SDKOperationInvoker struct {
	registry *SDKHandlerRegistry
}

// NewSDKOperationInvoker creates an invoker backed by the given handler registry.
func NewSDKOperationInvoker(registry *SDKHandlerRegistry) *SDKOperationInvoker {
	return &SDKOperationInvoker{registry: registry}
}

// Supports returns true for bindings with type "sdk".
func (inv *SDKOperationInvoker) Supports(binding model.OperationBinding) bool {
	return binding.Type == "sdk"
}

// Invoke looks up the handler by binding.Handler and delegates the call.
func (inv *SDKOperationInvoker) Invoke(
	ctx context.Context,
	rctx *model.RequestContext,
	binding model.OperationBinding,
	input model.InvocationInput,
) (model.InvocationResult, error) {
	handler, ok := inv.registry.Get(binding.Handler)
	if !ok {
		return model.InvocationResult{}, fmt.Errorf(
			"invoker: SDK handler %q not found", binding.Handler,
		)
	}
	return handler.Invoke(ctx, rctx, input)
}
