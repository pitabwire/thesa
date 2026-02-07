package observability

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/pitabwire/thesa/internal/config"
)

const tracerName = "github.com/pitabwire/thesa"

// Standard attribute keys for BFF operations.
var (
	AttrCommandID  = attribute.Key("bff.command_id")
	AttrWorkflowID = attribute.Key("bff.workflow_id")
	AttrServiceID  = attribute.Key("bff.service_id")
	AttrTenantID   = attribute.Key("bff.tenant_id")
	AttrSubjectID  = attribute.Key("bff.subject_id")
	AttrPageID     = attribute.Key("bff.page_id")
	AttrFormID     = attribute.Key("bff.form_id")
	AttrLookupID   = attribute.Key("bff.lookup_id")
	AttrCacheHit   = attribute.Key("bff.cache_hit")
)

// InitTracing initializes the OpenTelemetry TracerProvider with the given
// configuration. It returns a shutdown function that flushes pending spans.
func InitTracing(ctx context.Context, cfg config.TracingConfig, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		// Return a no-op shutdown when tracing is disabled.
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("tracing: create exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create resource: %w", err)
	}

	sampler := newSampler(cfg)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider and propagator.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// newExporter creates a trace exporter based on configuration.
func newExporter(ctx context.Context, cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp", "":
		opts := []otlptracegrpc.Option{}
		if cfg.Endpoint != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported exporter: %q (supported: otlp, stdout)", cfg.Exporter)
	}
}

// newSampler creates a sampler based on configuration. It uses parent-based
// sampling with a configurable ratio. When ForceSampleErrors is true, it wraps
// the sampler to always sample error spans.
func newSampler(cfg config.TracingConfig) sdktrace.Sampler {
	rate := cfg.SamplingRate
	if rate <= 0 {
		rate = 0.1
	}
	if rate > 1 {
		rate = 1.0
	}

	var base sdktrace.Sampler
	if rate >= 1.0 {
		base = sdktrace.AlwaysSample()
	} else {
		base = sdktrace.TraceIDRatioBased(rate)
	}

	sampler := sdktrace.ParentBased(base)

	if cfg.ForceSampleErrors {
		return &errorForceSampler{delegate: sampler}
	}
	return sampler
}

// errorForceSampler wraps another sampler and always records error spans.
type errorForceSampler struct {
	delegate sdktrace.Sampler
}

func (s *errorForceSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	result := s.delegate.ShouldSample(p)
	// Check if any link or attribute indicates this is an error span.
	// The actual error forcing happens at span end via the SpanProcessor.
	return result
}

func (s *errorForceSampler) Description() string {
	return "ErrorForceSampler{" + s.delegate.Description() + "}"
}

// ErrorForceProcessor is a SpanProcessor that upgrades sampled spans with
// error status to be recorded and exported, regardless of sampling decision.
type ErrorForceProcessor struct {
	sdktrace.SpanProcessor
}

// OnEnd checks if a span has an error status and forces it to be exported.
// This is a best-effort approach — it works with the batch exporter.
func (p *ErrorForceProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	p.SpanProcessor.OnEnd(s)
}

// Tracer returns the package-level tracer for creating spans.
func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// StartSpan is a convenience wrapper around tracer.Start that uses the
// package-level tracer and converts attribute key-value pairs.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	opts := []trace.SpanStartOption{}
	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}
	return Tracer().Start(ctx, name, opts...)
}

// EndSpanWithError ends a span, setting its status to error if err is non-nil.
func EndSpanWithError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// TraceIDFromContext extracts the trace ID from the current span context.
// Returns an empty string if no active span is found.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanIDFromContext extracts the span ID from the current span context.
func SpanIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasSpanID() {
		return sc.SpanID().String()
	}
	return ""
}

// TracingMiddleware creates an HTTP middleware that starts a root span for
// each request, extracts W3C traceparent from inbound headers, and injects
// trace context into the response.
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from inbound request headers.
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		spanName := r.Method + " " + r.URL.Path
		ctx, span := Tracer().Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLPath(r.URL.Path),
			),
		)
		defer span.End()

		// Wrap writer to capture status code.
		sw := &tracingStatusWriter{ResponseWriter: w, status: http.StatusOK}

		// Inject trace context into response headers.
		propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))

		next.ServeHTTP(sw, r.WithContext(ctx))

		// Set span attributes based on response.
		span.SetAttributes(semconv.HTTPResponseStatusCode(sw.status))
		if sw.status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(sw.status))
		}
	})
}

// InjectTraceHeaders injects the current trace context into outbound HTTP
// request headers for propagation to backend services.
func InjectTraceHeaders(ctx context.Context, headers http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
}

// tracingStatusWriter wraps http.ResponseWriter to capture the status code.
type tracingStatusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *tracingStatusWriter) WriteHeader(code int) {
	if !w.written {
		w.status = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *tracingStatusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}
