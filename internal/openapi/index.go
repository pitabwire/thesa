// Package openapi loads and indexes OpenAPI specifications, providing
// operation lookup by operationId with request/response schema resolution.
package openapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// SpecSource describes an OpenAPI spec file to load.
type SpecSource struct {
	ServiceID string
	BaseURL   string
	SpecPath  string
}

// IndexedOperation holds a resolved OpenAPI operation with its context.
type IndexedOperation struct {
	ServiceID    string
	OperationID  string
	Method       string
	PathTemplate string
	Parameters   []*openapi3.Parameter
	RequestBody  *openapi3.RequestBody
	Responses    *openapi3.Responses
	BaseURL      string
}

// ValidationError describes a schema validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Index is an in-memory index of OpenAPI operations keyed by (serviceID, operationID).
type Index struct {
	operations map[string]IndexedOperation // key: "serviceID:operationID"
	byService  map[string][]string         // serviceID → []operationID
}

// NewIndex creates an empty OpenAPI index.
func NewIndex() *Index {
	return &Index{
		operations: make(map[string]IndexedOperation),
		byService:  make(map[string][]string),
	}
}

func operationKey(serviceID, operationID string) string {
	return serviceID + ":" + operationID
}

// Load parses OpenAPI specs from the given sources and indexes all operations.
func (idx *Index) Load(specs []SpecSource) error {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	for _, src := range specs {
		doc, err := loader.LoadFromFile(src.SpecPath)
		if err != nil {
			return fmt.Errorf("openapi: loading %s (%s): %w", src.ServiceID, src.SpecPath, err)
		}

		if err := doc.Validate(context.Background()); err != nil {
			return fmt.Errorf("openapi: validating %s: %w", src.ServiceID, err)
		}

		baseURL := src.BaseURL
		if baseURL == "" && len(doc.Servers) > 0 {
			baseURL = doc.Servers[0].URL
		}

		for path, pathItem := range doc.Paths.Map() {
			for method, op := range pathItem.Operations() {
				if op.OperationID == "" {
					continue
				}

				// Merge path-level and operation-level parameters.
				params := make([]*openapi3.Parameter, 0)
				for _, ref := range pathItem.Parameters {
					if ref.Value != nil {
						params = append(params, ref.Value)
					}
				}
				for _, ref := range op.Parameters {
					if ref.Value != nil {
						params = append(params, ref.Value)
					}
				}

				var reqBody *openapi3.RequestBody
				if op.RequestBody != nil && op.RequestBody.Value != nil {
					reqBody = op.RequestBody.Value
				}

				indexed := IndexedOperation{
					ServiceID:    src.ServiceID,
					OperationID:  op.OperationID,
					Method:       method,
					PathTemplate: path,
					Parameters:   params,
					RequestBody:  reqBody,
					Responses:    op.Responses,
					BaseURL:      baseURL,
				}

				key := operationKey(src.ServiceID, op.OperationID)
				idx.operations[key] = indexed
				idx.byService[src.ServiceID] = append(idx.byService[src.ServiceID], op.OperationID)
			}
		}
	}

	return nil
}

// GetOperation returns the indexed operation for the given service and operation ID.
func (idx *Index) GetOperation(serviceID, operationID string) (IndexedOperation, bool) {
	op, ok := idx.operations[operationKey(serviceID, operationID)]
	return op, ok
}

// AllOperationIDs returns all operation IDs for the given service, sorted.
func (idx *Index) AllOperationIDs(serviceID string) []string {
	ids := make([]string, len(idx.byService[serviceID]))
	copy(ids, idx.byService[serviceID])
	sort.Strings(ids)
	return ids
}

// ValidateRequest validates a request body against the operation's request schema.
// Returns an empty slice if valid, or a list of validation errors.
func (idx *Index) ValidateRequest(serviceID, operationID string, body map[string]any) []ValidationError {
	op, ok := idx.operations[operationKey(serviceID, operationID)]
	if !ok {
		return []ValidationError{{Message: fmt.Sprintf("operation %s/%s not found", serviceID, operationID)}}
	}

	if op.RequestBody == nil {
		return nil
	}

	ct := op.RequestBody.Content.Get("application/json")
	if ct == nil || ct.Schema == nil || ct.Schema.Value == nil {
		return nil
	}

	schema := ct.Schema.Value
	var errs []ValidationError

	// Validate required fields.
	for _, req := range schema.Required {
		if _, exists := body[req]; !exists {
			errs = append(errs, ValidationError{
				Field:   req,
				Message: fmt.Sprintf("%s is required", req),
			})
		}
	}

	return errs
}
