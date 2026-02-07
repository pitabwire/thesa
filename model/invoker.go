package model

import "context"

// OperationInvoker is the unified interface for backend invocation.
type OperationInvoker interface {
	// Invoke calls the backend operation described by the binding with the given input.
	Invoke(ctx context.Context, rctx *RequestContext, binding OperationBinding, input InvocationInput) (InvocationResult, error)

	// Supports returns true if this invoker can handle the given binding type.
	Supports(binding OperationBinding) bool
}

// InvocationInput is the constructed backend request.
type InvocationInput struct {
	PathParams  map[string]string `json:"path_params,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        any               `json:"body,omitempty"`
}

// InvocationResult is the backend response.
type InvocationResult struct {
	StatusCode int               `json:"status_code"`
	Body       any               `json:"body,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// CommandInput is the frontend command request payload.
type CommandInput struct {
	Input          map[string]any    `json:"input"`
	RouteParams    map[string]string `json:"route_params,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// DataParams describes parameters for data-fetching endpoints.
type DataParams struct {
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Sort     string            `json:"sort,omitempty"`
	SortDir  string            `json:"sort_dir,omitempty"`
	Filters  map[string]string `json:"filters,omitempty"`
	Query    string            `json:"query,omitempty"`
}

// Pagination describes pagination parameters for search.
type Pagination struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Domain   string `json:"domain,omitempty"`
}

// WorkflowFilters describes filters for listing workflow instances.
type WorkflowFilters struct {
	Status     string `json:"status,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	SubjectID  string `json:"subject_id,omitempty"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}
