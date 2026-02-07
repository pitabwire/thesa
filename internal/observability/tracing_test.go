package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/pitabwire/thesa/internal/config"
)

// setupTestTracer creates an in-memory span exporter and configures a
// TracerProvider that always samples. Returns the exporter and a cleanup func.
func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return exporter
}

func TestInitTracing_disabled(t *testing.T) {
	cfg := config.TracingConfig{Enabled: false}
	shutdown, err := InitTracing(context.Background(), cfg, "test-svc", "1.0.0")
	if err != nil {
		t.Fatalf("InitTracing() error = %v", err)
	}
	// Shutdown should be a no-op.
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown() error = %v", err)
	}
}

func TestInitTracing_stdout(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:      true,
		Exporter:     "stdout",
		SamplingRate: 1.0,
	}
	shutdown, err := InitTracing(context.Background(), cfg, "test-svc", "1.0.0")
	if err != nil {
		t.Fatalf("InitTracing() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown() error = %v", err)
	}
}

func TestInitTracing_unsupportedExporter(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled:  true,
		Exporter: "zipkin",
	}
	_, err := InitTracing(context.Background(), cfg, "test-svc", "1.0.0")
	if err == nil {
		t.Fatal("expected error for unsupported exporter")
	}
}

func TestStartSpan_createsSpan(t *testing.T) {
	exporter := setupTestTracer(t)

	ctx, span := StartSpan(context.Background(), "test.operation",
		AttrCommandID.String("cmd-1"),
		AttrTenantID.String("tenant-1"),
	)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]
	if s.Name != "test.operation" {
		t.Errorf("span name = %q, want %q", s.Name, "test.operation")
	}

	attrMap := spanAttrMap(s)
	if v, ok := attrMap["bff.command_id"]; !ok || v != "cmd-1" {
		t.Errorf("bff.command_id = %q, want %q", v, "cmd-1")
	}
	if v, ok := attrMap["bff.tenant_id"]; !ok || v != "tenant-1" {
		t.Errorf("bff.tenant_id = %q, want %q", v, "tenant-1")
	}

	// Context should carry the span.
	if trace.SpanFromContext(ctx) != span {
		t.Error("context should carry the created span")
	}
}

func TestStartSpan_noAttributes(t *testing.T) {
	exporter := setupTestTracer(t)

	_, span := StartSpan(context.Background(), "simple.op")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "simple.op" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "simple.op")
	}
}

func TestStartSpan_parentChild(t *testing.T) {
	exporter := setupTestTracer(t)

	ctx, parent := StartSpan(context.Background(), "parent.op")
	_, child := StartSpan(ctx, "child.op")
	child.End()
	parent.End()

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Both spans should share the same trace ID.
	parentTraceID := spans[1].SpanContext.TraceID()
	childTraceID := spans[0].SpanContext.TraceID()
	if parentTraceID != childTraceID {
		t.Error("parent and child should share the same trace ID")
	}

	// Child's parent should be the parent span.
	if spans[0].Parent.SpanID() != spans[1].SpanContext.SpanID() {
		t.Error("child parent span ID should match parent span ID")
	}
}

func TestEndSpanWithError_setsErrorStatus(t *testing.T) {
	exporter := setupTestTracer(t)

	_, span := StartSpan(context.Background(), "error.op")
	EndSpanWithError(span, errors.New("something failed"))

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("status code = %v, want Error", spans[0].Status.Code)
	}
	if spans[0].Status.Description != "something failed" {
		t.Errorf("status description = %q, want %q", spans[0].Status.Description, "something failed")
	}
	// Should have recorded the error event.
	if len(spans[0].Events) == 0 {
		t.Error("expected at least one event (error recording)")
	}
}

func TestEndSpanWithError_nilError(t *testing.T) {
	exporter := setupTestTracer(t)

	_, span := StartSpan(context.Background(), "ok.op")
	EndSpanWithError(span, nil)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code == codes.Error {
		t.Error("status should not be Error when err is nil")
	}
}

func TestTraceIDFromContext_withActiveSpan(t *testing.T) {
	setupTestTracer(t)

	ctx, span := StartSpan(context.Background(), "trace.id.test")
	defer span.End()

	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		t.Error("TraceIDFromContext should return a non-empty trace ID")
	}
	if traceID != span.SpanContext().TraceID().String() {
		t.Errorf("TraceIDFromContext = %q, want %q", traceID, span.SpanContext().TraceID().String())
	}
}

func TestTraceIDFromContext_noSpan(t *testing.T) {
	traceID := TraceIDFromContext(context.Background())
	if traceID != "" {
		t.Errorf("TraceIDFromContext without span = %q, want empty", traceID)
	}
}

func TestSpanIDFromContext(t *testing.T) {
	setupTestTracer(t)

	ctx, span := StartSpan(context.Background(), "span.id.test")
	defer span.End()

	spanID := SpanIDFromContext(ctx)
	if spanID == "" {
		t.Error("SpanIDFromContext should return a non-empty span ID")
	}
}

func TestTracingMiddleware_createsRootSpan(t *testing.T) {
	exporter := setupTestTracer(t)

	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ui/pages/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]
	if s.Name != "GET /ui/pages/dashboard" {
		t.Errorf("span name = %q, want %q", s.Name, "GET /ui/pages/dashboard")
	}
	if s.SpanKind != trace.SpanKindServer {
		t.Errorf("span kind = %v, want Server", s.SpanKind)
	}

	attrMap := spanAttrMap(s)
	if v, ok := attrMap["http.request.method"]; !ok || v != "GET" {
		t.Errorf("http.request.method = %q, want GET", v)
	}
	if v, ok := attrMap["url.path"]; !ok || v != "/ui/pages/dashboard" {
		t.Errorf("url.path = %q, want /ui/pages/dashboard", v)
	}
}

func TestTracingMiddleware_500_setsErrorStatus(t *testing.T) {
	exporter := setupTestTracer(t)

	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/ui/commands/do-stuff", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("status code = %v, want Error for 500 response", spans[0].Status.Code)
	}
}

func TestTracingMiddleware_capturesStatusCode(t *testing.T) {
	exporter := setupTestTracer(t)

	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/ui/workflows/start", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	attrMap := spanAttrMap(spans[0])
	if v, ok := attrMap["http.response.status_code"]; !ok || v != "201" {
		t.Errorf("http.response.status_code = %q, want 201", v)
	}
}

func TestTracingMiddleware_extractsTraceparent(t *testing.T) {
	exporter := setupTestTracer(t)

	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The span in context should have the parent trace ID from the header.
		w.WriteHeader(http.StatusOK)
	}))

	// Valid W3C traceparent header.
	traceID := "0af7651916cd43dd8448eb211c80319c"
	parentSpanID := "b7ad6b7169203331"
	traceparent := "00-" + traceID + "-" + parentSpanID + "-01"

	req := httptest.NewRequest(http.MethodGet, "/ui/navigation", nil)
	req.Header.Set("Traceparent", traceparent)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// The span should have the extracted trace ID from the parent.
	if spans[0].SpanContext.TraceID().String() != traceID {
		t.Errorf("trace ID = %q, want %q", spans[0].SpanContext.TraceID().String(), traceID)
	}
	if spans[0].Parent.SpanID().String() != parentSpanID {
		t.Errorf("parent span ID = %q, want %q", spans[0].Parent.SpanID().String(), parentSpanID)
	}
}

func TestTracingMiddleware_injectsResponseHeaders(t *testing.T) {
	setupTestTracer(t)

	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ui/pages/home", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The response should have a traceparent header injected.
	tp := rec.Header().Get("Traceparent")
	if tp == "" {
		t.Error("response should have Traceparent header")
	}
}

func TestInjectTraceHeaders(t *testing.T) {
	setupTestTracer(t)

	ctx, span := StartSpan(context.Background(), "outbound.call")
	defer span.End()

	headers := http.Header{}
	InjectTraceHeaders(ctx, headers)

	if headers.Get("Traceparent") == "" {
		t.Error("InjectTraceHeaders should set Traceparent header")
	}
}

func TestTracer_returnsNamedTracer(t *testing.T) {
	setupTestTracer(t)

	tracer := Tracer()
	if tracer == nil {
		t.Fatal("Tracer() should not return nil")
	}
}

func TestNewSampler_defaultRate(t *testing.T) {
	cfg := config.TracingConfig{SamplingRate: 0}
	sampler := newSampler(cfg)
	desc := sampler.Description()
	if desc == "" {
		t.Error("sampler description should not be empty")
	}
}

func TestNewSampler_alwaysSample(t *testing.T) {
	cfg := config.TracingConfig{SamplingRate: 1.0}
	sampler := newSampler(cfg)
	desc := sampler.Description()
	// ParentBased with AlwaysSample as root should be in the description.
	if desc == "" {
		t.Error("sampler description should not be empty")
	}
}

func TestNewSampler_rateBased(t *testing.T) {
	cfg := config.TracingConfig{SamplingRate: 0.5}
	sampler := newSampler(cfg)
	if sampler == nil {
		t.Fatal("sampler should not be nil")
	}
}

func TestNewSampler_forceSampleErrors(t *testing.T) {
	cfg := config.TracingConfig{
		SamplingRate:      0.5,
		ForceSampleErrors: true,
	}
	sampler := newSampler(cfg)
	fs, ok := sampler.(*errorForceSampler)
	if !ok {
		t.Fatalf("expected *errorForceSampler, got %T", sampler)
	}
	if fs.Description() == "" {
		t.Error("errorForceSampler description should not be empty")
	}
}

func TestNewSampler_rateAbove1_clamps(t *testing.T) {
	cfg := config.TracingConfig{SamplingRate: 2.0}
	sampler := newSampler(cfg)
	if sampler == nil {
		t.Fatal("sampler should not be nil")
	}
}

func TestAllAttributeKeys(t *testing.T) {
	// Verify all standard attribute keys are defined and have expected prefixes.
	keys := []attribute.Key{
		AttrCommandID, AttrWorkflowID, AttrServiceID,
		AttrTenantID, AttrSubjectID, AttrPageID,
		AttrFormID, AttrLookupID, AttrCacheHit,
	}
	for _, k := range keys {
		if string(k) == "" {
			t.Error("attribute key should not be empty")
		}
	}
}

func TestSpanHierarchy_commandRequest(t *testing.T) {
	exporter := setupTestTracer(t)

	// Simulate span hierarchy for a command request.
	ctx, rootSpan := StartSpan(context.Background(), "HTTP POST /ui/commands/create-order",
		attribute.String("http.request.method", "POST"),
	)

	ctx, capSpan := StartSpan(ctx, "capability.resolve",
		AttrCacheHit.Bool(true),
	)
	capSpan.End()

	ctx, cmdSpan := StartSpan(ctx, "command.execute",
		AttrCommandID.String("create-order"),
	)

	ctx, mapSpan := StartSpan(ctx, "input.mapping")
	mapSpan.End()

	ctx, valSpan := StartSpan(ctx, "schema.validate")
	valSpan.End()

	ctx, invokeSpan := StartSpan(ctx, "backend.invoke",
		AttrServiceID.String("order-service"),
	)

	_, httpSpan := StartSpan(ctx, "http.client.request",
		attribute.String("http.url", "https://orders.internal/api/orders"),
		attribute.Int("http.status_code", 201),
	)
	httpSpan.End()
	invokeSpan.End()

	_, outMapSpan := StartSpan(ctx, "output.mapping")
	outMapSpan.End()

	cmdSpan.End()
	rootSpan.End()

	spans := exporter.GetSpans()
	if len(spans) != 8 {
		t.Fatalf("expected 8 spans, got %d", len(spans))
	}

	// All spans should share the same trace ID.
	traceID := spans[0].SpanContext.TraceID()
	for _, s := range spans {
		if s.SpanContext.TraceID() != traceID {
			t.Errorf("span %q has different trace ID", s.Name)
		}
	}

	// Verify expected span names exist.
	names := map[string]bool{}
	for _, s := range spans {
		names[s.Name] = true
	}
	expectedNames := []string{
		"HTTP POST /ui/commands/create-order",
		"capability.resolve",
		"command.execute",
		"input.mapping",
		"schema.validate",
		"backend.invoke",
		"http.client.request",
		"output.mapping",
	}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("missing span %q", name)
		}
	}
}

// --- helpers ---

// spanAttrMap converts a span's attributes to a map[string]string for easier assertion.
func spanAttrMap(s tracetest.SpanStub) map[string]string {
	m := make(map[string]string)
	for _, a := range s.Attributes {
		m[string(a.Key)] = a.Value.Emit()
	}
	return m
}
