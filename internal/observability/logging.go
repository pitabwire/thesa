package observability

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// Context key for the logger.
type loggerKey struct{}

// NewLogger creates a zap.Logger configured for JSON output to stdout.
//
// Log level usage conventions:
//   - error: Infrastructure failures (DB down, unhandled panics), 5xx responses
//   - warn:  Client errors (4xx), degraded operation (circuit breaker open), slow queries
//   - info:  Request start/end, command execution, workflow transitions, definition reload
//   - debug: Cache operations, input/output mapping details, schema validation
func NewLogger(cfg config.ObservabilityConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	zapCfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return zapCfg.Build()
}

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFrom returns the logger stored in the context, or the provided
// fallback if none is found.
func LoggerFrom(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return fallback
}

// RequestLogger returns a logger enriched with RequestContext fields.
// If no logger is in the context, the fallback is used.
func RequestLogger(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	logger := LoggerFrom(ctx, fallback)

	rctx := model.RequestContextFrom(ctx)
	if rctx == nil {
		return logger
	}

	fields := []zap.Field{
		zap.String("tenant_id", rctx.TenantID),
		zap.String("subject_id", rctx.SubjectID),
		zap.String("partition_id", rctx.PartitionID),
		zap.String("correlation_id", rctx.CorrelationID),
	}

	// Include trace_id if present.
	if rctx.TraceID != "" {
		fields = append(fields, zap.String("trace_id", rctx.TraceID))
	}

	return logger.With(fields...)
}

// defaultSensitiveFields is the default set of field names that should be
// redacted in debug logging output.
var defaultSensitiveFields = map[string]bool{
	"password":      true,
	"secret":        true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"api_key":       true,
	"authorization": true,
	"credit_card":   true,
	"ssn":           true,
	"pin":           true,
}

// RedactBody returns a copy of body with sensitive fields replaced by
// "[REDACTED]". The sensitiveFields list is merged with default sensitive
// field names. This is intended for debug-level logging only.
func RedactBody(body map[string]any, sensitiveFields []string) map[string]any {
	if body == nil {
		return nil
	}

	redactSet := make(map[string]bool, len(defaultSensitiveFields)+len(sensitiveFields))
	for k, v := range defaultSensitiveFields {
		redactSet[k] = v
	}
	for _, f := range sensitiveFields {
		redactSet[f] = true
	}

	result := make(map[string]any, len(body))
	for k, v := range body {
		if redactSet[k] {
			result[k] = "[REDACTED]"
		} else if nested, ok := v.(map[string]any); ok {
			result[k] = RedactBody(nested, sensitiveFields)
		} else {
			result[k] = v
		}
	}
	return result
}
