package invoker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/model"
)

// serviceClient holds the HTTP client, circuit breaker, and retry config
// for a single backend service.
type serviceClient struct {
	cfg     config.ServiceConfig
	client  *http.Client
	breaker *CircuitBreaker
}

// OpenAPIOperationInvoker dynamically builds and executes HTTP requests
// against backend services using indexed OpenAPI specifications.
type OpenAPIOperationInvoker struct {
	index   *openapi.Index
	clients map[string]*serviceClient
}

// NewOpenAPIOperationInvoker creates an invoker with per-service HTTP clients,
// circuit breakers, and retry policies.
func NewOpenAPIOperationInvoker(idx *openapi.Index, services map[string]config.ServiceConfig) *OpenAPIOperationInvoker {
	clients := make(map[string]*serviceClient, len(services))
	for id, svcCfg := range services {
		timeout := svcCfg.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		transport := &http.Transport{
			MaxIdleConns:        100,
			MaxConnsPerHost:     50,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}
		cbCfg := svcCfg.CircuitBreaker
		clients[id] = &serviceClient{
			cfg: svcCfg,
			client: &http.Client{
				Timeout:   timeout,
				Transport: transport,
			},
			breaker: NewCircuitBreaker(
				cbCfg.FailureThreshold,
				cbCfg.SuccessThreshold,
				cbCfg.Timeout,
			),
		}
	}
	return &OpenAPIOperationInvoker{
		index:   idx,
		clients: clients,
	}
}

// Supports returns true for operation bindings with type "openapi".
func (inv *OpenAPIOperationInvoker) Supports(binding model.OperationBinding) bool {
	return binding.Type == "openapi"
}

// Invoke looks up the operation in the OpenAPI index, builds an HTTP request,
// and executes it with circuit breaker and retry support.
func (inv *OpenAPIOperationInvoker) Invoke(
	ctx context.Context,
	rctx *model.RequestContext,
	binding model.OperationBinding,
	input model.InvocationInput,
) (model.InvocationResult, error) {
	op, ok := inv.index.GetOperation(binding.ServiceID, binding.OperationID)
	if !ok {
		return model.InvocationResult{}, fmt.Errorf(
			"invoker: operation %s/%s not found in OpenAPI index",
			binding.ServiceID, binding.OperationID,
		)
	}

	svc, ok := inv.clients[binding.ServiceID]
	if !ok {
		return model.InvocationResult{}, fmt.Errorf(
			"invoker: service %q not configured", binding.ServiceID,
		)
	}

	reqURL := buildRequestURL(op, input)
	headers := buildRequestHeaders(rctx, input, op.Method)

	var bodyBytes []byte
	if input.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(input.Body)
		if err != nil {
			return model.InvocationResult{}, fmt.Errorf("invoker: marshal body: %w", err)
		}
	}

	return inv.executeWithRetry(ctx, svc, op.Method, reqURL, headers, bodyBytes)
}

// executeWithRetry wraps executeOnce with retry logic and exponential backoff.
func (inv *OpenAPIOperationInvoker) executeWithRetry(
	ctx context.Context,
	svc *serviceClient,
	method, reqURL string,
	headers http.Header,
	bodyBytes []byte,
) (model.InvocationResult, error) {
	retryCfg := svc.cfg.Retry
	maxAttempts := retryCfg.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	canRetry := isIdempotentMethod(method) || !retryCfg.IdempotentOnly

	var lastErr error
	var lastResult model.InvocationResult

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := calculateBackoff(retryCfg, attempt)
			select {
			case <-ctx.Done():
				return model.InvocationResult{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := inv.executeOnce(ctx, svc, method, reqURL, headers, bodyBytes)
		if err != nil {
			lastErr = err
			if !canRetry || !isRetryableError(err) {
				return model.InvocationResult{}, err
			}
			slog.Debug("invoker: retrying after error",
				"attempt", attempt+1,
				"max", maxAttempts,
				"error", err,
			)
			continue
		}

		if isRetryableStatus(result.StatusCode) && canRetry && attempt < maxAttempts-1 {
			lastResult = result
			slog.Debug("invoker: retrying after status",
				"attempt", attempt+1,
				"max", maxAttempts,
				"status", result.StatusCode,
			)
			continue
		}

		return result, nil
	}

	if lastErr != nil {
		return model.InvocationResult{}, lastErr
	}
	return lastResult, nil
}

// executeOnce performs a single HTTP request with circuit breaker protection.
func (inv *OpenAPIOperationInvoker) executeOnce(
	ctx context.Context,
	svc *serviceClient,
	method, reqURL string,
	headers http.Header,
	bodyBytes []byte,
) (model.InvocationResult, error) {
	if err := svc.breaker.Allow(); err != nil {
		return model.InvocationResult{}, model.NewBackendUnavailableError()
	}

	var body io.Reader
	if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return model.InvocationResult{}, fmt.Errorf("invoker: build request: %w", err)
	}
	req.Header = headers

	resp, err := svc.client.Do(req)
	if err != nil {
		svc.breaker.RecordFailure()
		if isConnectionError(err) {
			return model.InvocationResult{}, model.NewBackendUnavailableError()
		}
		if ctx.Err() != nil {
			return model.InvocationResult{}, model.NewBackendTimeoutError()
		}
		return model.InvocationResult{}, fmt.Errorf("invoker: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		svc.breaker.RecordFailure()
		return model.InvocationResult{}, fmt.Errorf("invoker: read response: %w", err)
	}

	// Record circuit breaker outcome.
	if isServerError(resp.StatusCode) {
		svc.breaker.RecordFailure()
	} else if !isClientError(resp.StatusCode) {
		// Only record success for 2xx/3xx; 4xx are not infrastructure failures.
		svc.breaker.RecordSuccess()
	}

	result := model.InvocationResult{
		StatusCode: resp.StatusCode,
		Headers:    extractResponseHeaders(resp),
	}

	// Parse JSON response body if present.
	if len(respBody) > 0 {
		var parsed any
		if err := json.Unmarshal(respBody, &parsed); err == nil {
			result.Body = parsed
		}
	}

	return result, nil
}

// --- URL and header building ---

func buildRequestURL(op openapi.IndexedOperation, input model.InvocationInput) string {
	path := op.PathTemplate

	// Substitute path parameters.
	for name, value := range input.PathParams {
		path = strings.ReplaceAll(path, "{"+name+"}", url.PathEscape(value))
	}

	result := op.BaseURL + path

	// Append query parameters.
	if len(input.QueryParams) > 0 {
		params := url.Values{}
		for k, v := range input.QueryParams {
			params.Set(k, v)
		}
		result += "?" + params.Encode()
	}

	return result
}

func buildRequestHeaders(rctx *model.RequestContext, input model.InvocationInput, method string) http.Header {
	h := make(http.Header)

	h.Set("Accept", "application/json")
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		h.Set("Content-Type", "application/json")
	}

	if rctx != nil {
		if rctx.Token != "" {
			h.Set("Authorization", "Bearer "+sanitizeHeader(rctx.Token))
		}
		h.Set("X-Tenant-Id", sanitizeHeader(rctx.TenantID))
		h.Set("X-Partition-Id", sanitizeHeader(rctx.PartitionID))
		h.Set("X-Correlation-Id", sanitizeHeader(rctx.CorrelationID))
		h.Set("X-Request-Subject", sanitizeHeader(rctx.SubjectID))
	}

	// Apply custom headers from input (after standard headers, so they can override).
	for k, v := range input.Headers {
		h.Set(sanitizeHeader(k), sanitizeHeader(v))
	}

	return h
}

// sanitizeHeader strips newlines and carriage returns to prevent header injection.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func extractResponseHeaders(resp *http.Response) map[string]string {
	headers := make(map[string]string)
	for _, key := range []string{
		"Content-Type", "X-Correlation-Id", "X-Trace-Id",
		"X-Request-Id", "Retry-After",
	} {
		if v := resp.Header.Get(key); v != "" {
			headers[key] = v
		}
	}
	return headers
}

// --- classification helpers ---

func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodDelete,
		http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func isServerError(code int) bool {
	return code >= 500
}

func isClientError(code int) bool {
	return code >= 400 && code < 500
}

func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Circuit breaker open errors are not retryable.
	if _, ok := err.(*model.ErrorEnvelope); ok {
		return false
	}
	return true
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return false
}

func calculateBackoff(cfg config.RetryConfig, attempt int) time.Duration {
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = 100 * time.Millisecond
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 2
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 2 * time.Second
	}

	delay := cfg.BackoffInitial
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * cfg.BackoffMultiplier)
		if delay > cfg.BackoffMax {
			delay = cfg.BackoffMax
			break
		}
	}
	return delay
}
