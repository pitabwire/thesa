// Package invoker implements backend invocation through OpenAPI-driven HTTP calls
// and SDK handler dispatch, with circuit breaker and retry support.
package invoker

import (
	"context"
	"fmt"

	"github.com/pitabwire/thesa/model"
)

// Registry holds all OperationInvoker implementations and dispatches
// invocations to the appropriate one based on the operation binding type.
type Registry struct {
	invokers []model.OperationInvoker
}

// NewRegistry creates a new empty InvokerRegistry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds an invoker to the registry.
func (r *Registry) Register(invoker model.OperationInvoker) {
	r.invokers = append(r.invokers, invoker)
}

// Invoke finds the first registered invoker that supports the given binding
// and delegates the call. Returns an error if no invoker supports the binding.
func (r *Registry) Invoke(ctx context.Context, rctx *model.RequestContext, binding model.OperationBinding, input model.InvocationInput) (model.InvocationResult, error) {
	for _, inv := range r.invokers {
		if inv.Supports(binding) {
			return inv.Invoke(ctx, rctx, binding, input)
		}
	}
	return model.InvocationResult{}, fmt.Errorf("invoker: no invoker supports binding type %q", binding.Type)
}
