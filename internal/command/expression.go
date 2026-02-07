package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pitabwire/thesa/model"
)

// ExpressionResolver resolves source expressions against the available
// sources: user input, route params, request context, and optional
// workflow state.
type ExpressionResolver struct {
	Input         map[string]any
	RouteParams   map[string]string
	Context       *model.RequestContext
	WorkflowState map[string]any
}

// Resolve evaluates a source expression string and returns the resolved value.
// Supported expressions:
//   - input.field_name         — value from user input
//   - input.address.city       — nested field access
//   - route.param_name         — value from route parameters
//   - context.subject_id       — from RequestContext
//   - context.tenant_id        — from RequestContext
//   - context.partition_id     — from RequestContext
//   - context.email            — from RequestContext
//   - workflow.field_name      — from workflow state
//   - 'literal'                — single-quoted literal string
//   - 123 / 99.99              — numeric literal
func (r *ExpressionResolver) Resolve(expr string) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// Literal string: single-quoted.
	if len(expr) >= 2 && expr[0] == '\'' && expr[len(expr)-1] == '\'' {
		return expr[1 : len(expr)-1], nil
	}

	// Numeric literal.
	if isNumericLiteral(expr) {
		return parseNumeric(expr)
	}

	// Source expressions: prefix.path
	dotIdx := strings.IndexByte(expr, '.')
	if dotIdx < 0 {
		return nil, fmt.Errorf("invalid expression %q: missing source prefix", expr)
	}

	prefix := expr[:dotIdx]
	path := expr[dotIdx+1:]
	if path == "" {
		return nil, fmt.Errorf("invalid expression %q: empty path after prefix", expr)
	}

	switch prefix {
	case "input":
		return r.resolveInput(path)
	case "route":
		return r.resolveRoute(path)
	case "context":
		return r.resolveContext(path)
	case "workflow":
		return r.resolveWorkflow(path)
	default:
		return nil, fmt.Errorf("unknown expression prefix %q in %q", prefix, expr)
	}
}

// resolveInput resolves a dotted path in the user input map.
func (r *ExpressionResolver) resolveInput(path string) (any, error) {
	if r.Input == nil {
		return nil, fmt.Errorf("input source is nil, cannot resolve %q", "input."+path)
	}
	val := navigatePath(r.Input, path)
	if val == nil {
		return nil, fmt.Errorf("input field %q not found", path)
	}
	return val, nil
}

// resolveRoute resolves a route parameter.
func (r *ExpressionResolver) resolveRoute(param string) (any, error) {
	if r.RouteParams == nil {
		return nil, fmt.Errorf("route params is nil, cannot resolve %q", "route."+param)
	}
	val, ok := r.RouteParams[param]
	if !ok {
		return nil, fmt.Errorf("route param %q not found", param)
	}
	return val, nil
}

// resolveContext resolves a request context field.
func (r *ExpressionResolver) resolveContext(field string) (any, error) {
	if r.Context == nil {
		return nil, fmt.Errorf("request context is nil, cannot resolve %q", "context."+field)
	}
	switch field {
	case "subject_id":
		return r.Context.SubjectID, nil
	case "tenant_id":
		return r.Context.TenantID, nil
	case "partition_id":
		return r.Context.PartitionID, nil
	case "email":
		return r.Context.Email, nil
	default:
		return nil, fmt.Errorf("unknown context field %q", field)
	}
}

// resolveWorkflow resolves a field from workflow state.
func (r *ExpressionResolver) resolveWorkflow(path string) (any, error) {
	if r.WorkflowState == nil {
		return nil, fmt.Errorf("workflow state is nil, cannot resolve %q", "workflow."+path)
	}
	val := navigatePath(r.WorkflowState, path)
	if val == nil {
		return nil, fmt.Errorf("workflow field %q not found", path)
	}
	return val, nil
}

// navigatePath navigates a dot-separated path through nested maps.
func navigatePath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

// isNumericLiteral returns true if the string looks like a number.
func isNumericLiteral(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
		if start >= len(s) {
			return false
		}
	}
	hasDot := false
	for i := start; i < len(s); i++ {
		if s[i] == '.' {
			if hasDot {
				return false
			}
			hasDot = true
		} else if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseNumeric parses a numeric string literal.
func parseNumeric(s string) (any, error) {
	if strings.ContainsRune(s, '.') {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric literal %q: %w", s, err)
		}
		return v, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid numeric literal %q: %w", s, err)
	}
	return v, nil
}
