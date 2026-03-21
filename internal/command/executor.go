package command

import (
	"context"
	"fmt"

	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	openapiIndex "github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

// CommandExecutor implements the command execution pipeline.
type CommandExecutor struct {
	registry *definition.Registry
	invokers *invoker.Registry
	index    *openapiIndex.Index
	mapper   *InputMapper
}

// NewCommandExecutor creates a CommandExecutor with its required dependencies.
func NewCommandExecutor(
	registry *definition.Registry,
	invokers *invoker.Registry,
	index *openapiIndex.Index,
) *CommandExecutor {
	return &CommandExecutor{
		registry: registry,
		invokers: invokers,
		index:    index,
		mapper:   NewInputMapper(),
	}
}

// Execute runs the full 10-step command pipeline.
func (e *CommandExecutor) Execute(
	ctx context.Context,
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	commandID string,
	input model.CommandInput,
) (model.CommandResponse, error) {
	// Step 1: Lookup command definition.
	cmdDef, ok := e.registry.GetCommand(commandID)
	if !ok {
		return model.CommandResponse{}, model.NewNotFoundError(
			fmt.Sprintf("command %q not found", commandID),
		)
	}

	// Step 2: Check capabilities.
	if len(cmdDef.Capabilities) > 0 && !caps.HasAll(cmdDef.Capabilities...) {
		return model.CommandResponse{}, model.NewForbiddenError(
			fmt.Sprintf("insufficient capabilities for command %q", commandID),
		)
	}

	// Step 5: Apply input mapping.
	invInput, err := e.mapper.MapInput(cmdDef.Input, input, rctx, nil)
	if err != nil {
		return model.CommandResponse{}, model.NewBadRequestError(
			fmt.Sprintf("input mapping error: %v", err),
		)
	}

	// Step 6: Validate constructed body against OpenAPI schema.
	if e.index != nil && cmdDef.Operation.Type == "openapi" && cmdDef.Operation.ServiceID != "" {
		if bodyMap, ok := invInput.Body.(map[string]any); ok {
			valErrs := e.index.ValidateRequest(cmdDef.Operation.ServiceID, cmdDef.Operation.OperationID, bodyMap)
			if len(valErrs) > 0 {
				reverseMap := ReverseFieldMap(cmdDef.Input.FieldProjection)
				fieldErrors := translateValidationErrors(valErrs, reverseMap)
				return model.CommandResponse{
					Success: false,
					Errors:  fieldErrors,
				}, model.NewValidationError(fieldErrors)
			}
		}
	}

	// Step 7: Invoke backend.
	result, err := e.invokers.Invoke(ctx, rctx, cmdDef.Operation, invInput)
	if err != nil {
		return model.CommandResponse{}, err
	}

	// Step 8: Handle response.
	resp := e.handleResponse(result, cmdDef)

	if !resp.Success {
		return resp, model.NewBadRequestError(resp.Message)
	}

	return resp, nil
}

// Validate performs dry-run validation of a command's input against the
// OpenAPI schema without invoking the backend.
func (e *CommandExecutor) Validate(
	rctx *model.RequestContext,
	caps model.CapabilitySet,
	commandID string,
	input model.CommandInput,
) []model.FieldError {
	cmdDef, ok := e.registry.GetCommand(commandID)
	if !ok {
		return []model.FieldError{{Field: "", Code: "NOT_FOUND", Message: fmt.Sprintf("command %q not found", commandID)}}
	}

	if len(cmdDef.Capabilities) > 0 && !caps.HasAll(cmdDef.Capabilities...) {
		return []model.FieldError{{Field: "", Code: "FORBIDDEN", Message: "insufficient capabilities"}}
	}

	// Apply input mapping to get the backend body.
	invInput, err := e.mapper.MapInput(cmdDef.Input, input, rctx, nil)
	if err != nil {
		return []model.FieldError{{Field: "", Code: "MAPPING_ERROR", Message: err.Error()}}
	}

	if e.index == nil || cmdDef.Operation.Type != "openapi" || cmdDef.Operation.ServiceID == "" {
		return nil
	}

	bodyMap, ok := invInput.Body.(map[string]any)
	if !ok {
		return nil
	}

	valErrs := e.index.ValidateRequest(cmdDef.Operation.ServiceID, cmdDef.Operation.OperationID, bodyMap)
	if len(valErrs) == 0 {
		return nil
	}

	reverseMap := ReverseFieldMap(cmdDef.Input.FieldProjection)
	return translateValidationErrors(valErrs, reverseMap)
}

// handleResponse processes the backend response and builds a CommandResponse.
func (e *CommandExecutor) handleResponse(
	result model.InvocationResult,
	cmdDef model.CommandDefinition,
) model.CommandResponse {
	statusCode := result.StatusCode

	// 2xx: Success.
	if statusCode >= 200 && statusCode < 300 {
		resp := model.CommandResponse{
			Success: true,
			Message: cmdDef.Output.SuccessMessage,
		}

		// Apply output mapping.
		if body, ok := result.Body.(map[string]any); ok {
			resp.Result = applyOutputMapping(body, cmdDef.Output)
		}

		return resp
	}

	// 4xx: Client error — translate via error_map.
	if statusCode >= 400 && statusCode < 500 {
		return e.handleClientError(result, cmdDef)
	}

	// 5xx: Server error — generic error.
	return model.CommandResponse{
		Success: false,
		Message: "An internal error occurred. Please try again later.",
	}
}

// handleClientError translates backend 4xx errors using the error_map.
func (e *CommandExecutor) handleClientError(
	result model.InvocationResult,
	cmdDef model.CommandDefinition,
) model.CommandResponse {
	body, ok := result.Body.(map[string]any)
	if !ok {
		return model.CommandResponse{
			Success: false,
			Message: fmt.Sprintf("Backend returned status %d", result.StatusCode),
		}
	}

	// Extract error code and message from backend response.
	errorCode := extractString(body, "error.code")
	errorMsg := extractString(body, "error.message")
	if errorCode == "" {
		errorCode = extractString(body, "code")
	}
	if errorMsg == "" {
		errorMsg = extractString(body, "message")
	}

	// Translate error code via error_map.
	if translated, ok := cmdDef.Output.ErrorMap[errorCode]; ok {
		errorMsg = translated
	}

	resp := model.CommandResponse{
		Success: false,
		Message: errorMsg,
	}

	// Extract field errors from backend response and reverse field names.
	if details := extractFieldErrors(body); len(details) > 0 {
		reverseMap := ReverseFieldMap(cmdDef.Input.FieldProjection)
		for i, fe := range details {
			if uiField, ok := reverseMap[fe.Field]; ok {
				details[i].Field = uiField
			}
		}
		resp.Errors = details
	}

	return resp
}

// --- helpers ---

// applyOutputMapping extracts and renames fields from the response body.
func applyOutputMapping(body map[string]any, output model.OutputMapping) map[string]any {
	if len(output.Fields) == 0 {
		return body
	}
	result := make(map[string]any, len(output.Fields))
	for uiField, backendPath := range output.Fields {
		result[uiField] = navigatePath(body, backendPath)
	}
	return result
}

// translateValidationErrors converts OpenAPI validation errors to model.FieldError,
// reversing field names using the reverse map.
func translateValidationErrors(valErrs []openapiIndex.ValidationError, reverseMap map[string]string) []model.FieldError {
	fieldErrors := make([]model.FieldError, 0, len(valErrs))
	for _, ve := range valErrs {
		field := ve.Field
		if uiField, ok := reverseMap[field]; ok {
			field = uiField
		}
		code := "INVALID_VALUE"
		if ve.Field != "" {
			// If the message mentions "required", use REQUIRED code.
			if len(ve.Message) > 0 && containsWord(ve.Message, "required") {
				code = "REQUIRED"
			}
		}
		fieldErrors = append(fieldErrors, model.FieldError{
			Field:   field,
			Code:    code,
			Message: ve.Message,
		})
	}
	return fieldErrors
}

// containsWord checks if a string contains a specific word (case-insensitive substring).
func containsWord(s, word string) bool {
	for i := 0; i <= len(s)-len(word); i++ {
		match := true
		for j := 0; j < len(word); j++ {
			sc := s[i+j]
			wc := word[j]
			// Simple ASCII case-insensitive comparison.
			if sc != wc && sc != wc+32 && sc != wc-32 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// extractString extracts a string value from a nested map using a dot path.
func extractString(data map[string]any, path string) string {
	val := navigatePath(data, path)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprint(val)
}

// extractFieldErrors tries to extract field-level errors from a backend error response.
// Common patterns: error.details (array of {field, code, message}).
func extractFieldErrors(body map[string]any) []model.FieldError {
	// Try error.details pattern.
	details := navigatePath(body, "error.details")
	if details == nil {
		details = navigatePath(body, "details")
	}

	slice, ok := details.([]any)
	if !ok {
		return nil
	}

	var result []model.FieldError
	for _, item := range slice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fe := model.FieldError{}
		if v, ok := m["field"].(string); ok {
			fe.Field = v
		}
		if v, ok := m["code"].(string); ok {
			fe.Code = v
		}
		if v, ok := m["message"].(string); ok {
			fe.Message = v
		}
		if fe.Field != "" || fe.Message != "" {
			result = append(result, fe)
		}
	}
	return result
}
