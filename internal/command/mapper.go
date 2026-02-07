package command

import (
	"fmt"
	"strings"

	"github.com/pitabwire/thesa/model"
)

// InputMapper resolves an InputMapping definition into a concrete
// InvocationInput by evaluating source expressions.
type InputMapper struct{}

// NewInputMapper creates a new InputMapper.
func NewInputMapper() *InputMapper {
	return &InputMapper{}
}

// MapInput resolves the mapping definition using the provided sources and
// returns a ready-to-invoke InvocationInput.
func (m *InputMapper) MapInput(
	mapping model.InputMapping,
	input model.CommandInput,
	rctx *model.RequestContext,
	workflowState map[string]any,
) (model.InvocationInput, error) {

	resolver := &ExpressionResolver{
		Input:         input.Input,
		RouteParams:   input.RouteParams,
		Context:       rctx,
		WorkflowState: workflowState,
	}

	result := model.InvocationInput{}

	// Resolve path params.
	if len(mapping.PathParams) > 0 {
		result.PathParams = make(map[string]string, len(mapping.PathParams))
		for param, expr := range mapping.PathParams {
			val, err := resolver.Resolve(expr)
			if err != nil {
				return model.InvocationInput{}, fmt.Errorf("path_params[%s]: %w", param, err)
			}
			result.PathParams[param] = fmt.Sprint(val)
		}
	}

	// Resolve query params.
	if len(mapping.QueryParams) > 0 {
		result.QueryParams = make(map[string]string, len(mapping.QueryParams))
		for param, expr := range mapping.QueryParams {
			val, err := resolver.Resolve(expr)
			if err != nil {
				return model.InvocationInput{}, fmt.Errorf("query_params[%s]: %w", param, err)
			}
			result.QueryParams[param] = fmt.Sprint(val)
		}
	}

	// Resolve header params.
	if len(mapping.HeaderParams) > 0 {
		result.Headers = make(map[string]string, len(mapping.HeaderParams))
		for header, expr := range mapping.HeaderParams {
			val, err := resolver.Resolve(expr)
			if err != nil {
				return model.InvocationInput{}, fmt.Errorf("header_params[%s]: %w", header, err)
			}
			result.Headers[header] = fmt.Sprint(val)
		}
	}

	// Build body based on strategy.
	body, err := m.buildBody(mapping, resolver, input)
	if err != nil {
		return model.InvocationInput{}, err
	}
	result.Body = body

	return result, nil
}

// buildBody constructs the request body using the configured strategy.
func (m *InputMapper) buildBody(
	mapping model.InputMapping,
	resolver *ExpressionResolver,
	input model.CommandInput,
) (any, error) {
	switch strings.ToLower(mapping.BodyMapping) {
	case "passthrough", "":
		return input.Input, nil

	case "template":
		return m.resolveTemplate(mapping.BodyTemplate, resolver)

	case "projection":
		return m.resolveProjection(mapping.FieldProjection, resolver)

	default:
		return nil, fmt.Errorf("unknown body_mapping strategy %q", mapping.BodyMapping)
	}
}

// resolveTemplate recursively resolves expression values in a template map.
// Leaf string values that look like expressions are resolved; non-expression
// strings are passed through as literals.
func (m *InputMapper) resolveTemplate(
	template map[string]string,
	resolver *ExpressionResolver,
) (map[string]any, error) {
	if len(template) == 0 {
		return map[string]any{}, nil
	}

	result := make(map[string]any, len(template))
	for key, expr := range template {
		val, err := resolver.Resolve(expr)
		if err != nil {
			return nil, fmt.Errorf("body_template[%s]: %w", key, err)
		}
		result[key] = val
	}
	return result, nil
}

// resolveProjection resolves a field projection, mapping output field names
// to expression-resolved values.
func (m *InputMapper) resolveProjection(
	projection map[string]string,
	resolver *ExpressionResolver,
) (map[string]any, error) {
	if len(projection) == 0 {
		return map[string]any{}, nil
	}

	result := make(map[string]any, len(projection))
	for outField, expr := range projection {
		val, err := resolver.Resolve(expr)
		if err != nil {
			return nil, fmt.Errorf("field_projection[%s]: %w", outField, err)
		}
		result[outField] = val
	}
	return result, nil
}

// ReverseFieldMap builds a reverse mapping from backend field names to UI
// field names. This is used for translating validation errors from the
// backend back to the frontend field names.
func ReverseFieldMap(projection map[string]string) map[string]string {
	if len(projection) == 0 {
		return nil
	}

	reverse := make(map[string]string, len(projection))
	for uiField, expr := range projection {
		// Only reverse mappings that reference input fields.
		if strings.HasPrefix(expr, "input.") {
			backendField := uiField // The output key IS the backend field name.
			origField := expr[len("input."):]
			reverse[backendField] = origField
		}
	}
	return reverse
}
