package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// newTestLogger creates a logger that writes JSON to a buffer for assertion.
func newTestLogger(buf *bytes.Buffer) *zap.Logger {
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	})
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	return zap.New(core)
}

func TestNewLogger_defaultLevel(t *testing.T) {
	cfg := config.ObservabilityConfig{LogLevel: "info"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Sync()

	// Info should be enabled, Debug should not.
	if !logger.Core().Enabled(zapcore.InfoLevel) {
		t.Error("info level should be enabled")
	}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		t.Error("debug level should NOT be enabled at info level")
	}
}

func TestNewLogger_debugLevel(t *testing.T) {
	cfg := config.ObservabilityConfig{LogLevel: "debug"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Sync()

	if !logger.Core().Enabled(zapcore.DebugLevel) {
		t.Error("debug level should be enabled")
	}
}

func TestNewLogger_invalidLevel_defaultsToInfo(t *testing.T) {
	cfg := config.ObservabilityConfig{LogLevel: "bogus"}
	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Sync()

	if !logger.Core().Enabled(zapcore.InfoLevel) {
		t.Error("should default to info level")
	}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		t.Error("debug should NOT be enabled with invalid level (defaults to info)")
	}
}

func TestWithLogger_and_LoggerFrom(t *testing.T) {
	logger := zap.NewNop()
	ctx := WithLogger(context.Background(), logger)

	got := LoggerFrom(ctx, nil)
	if got != logger {
		t.Error("LoggerFrom should return the stored logger")
	}
}

func TestLoggerFrom_fallback(t *testing.T) {
	fallback := zap.NewNop()
	got := LoggerFrom(context.Background(), fallback)
	if got != fallback {
		t.Error("LoggerFrom should return fallback when no logger in context")
	}
}

func TestRequestLogger_enrichesWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	rctx := &model.RequestContext{
		TenantID:      "tenant-1",
		SubjectID:     "user-42",
		PartitionID:   "part-1",
		CorrelationID: "corr-abc",
		TraceID:       "trace-xyz",
	}
	ctx := model.WithRequestContext(context.Background(), rctx)
	ctx = WithLogger(ctx, logger)

	rl := RequestLogger(ctx, logger)
	rl.Info("test message")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	checks := map[string]string{
		"tenant_id":      "tenant-1",
		"subject_id":     "user-42",
		"partition_id":   "part-1",
		"correlation_id": "corr-abc",
		"trace_id":       "trace-xyz",
		"msg":            "test message",
		"level":          "info",
	}

	for key, want := range checks {
		got, ok := entry[key].(string)
		if !ok {
			t.Errorf("missing field %q in log entry", key)
			continue
		}
		if got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestRequestLogger_noTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	rctx := &model.RequestContext{
		TenantID:      "tenant-1",
		SubjectID:     "user-42",
		CorrelationID: "corr-abc",
	}
	ctx := model.WithRequestContext(context.Background(), rctx)

	rl := RequestLogger(ctx, logger)
	rl.Info("no trace")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	if _, exists := entry["trace_id"]; exists {
		t.Error("trace_id should not be present when empty")
	}
}

func TestRequestLogger_noRequestContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	rl := RequestLogger(context.Background(), logger)
	rl.Info("no context")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	// Should still log, just without context fields.
	if entry["msg"] != "no context" {
		t.Errorf("msg = %q, want no context", entry["msg"])
	}
	if _, exists := entry["tenant_id"]; exists {
		t.Error("tenant_id should not be present without RequestContext")
	}
}

func TestRedactBody_defaultFields(t *testing.T) {
	body := map[string]any{
		"name":     "John",
		"password": "secret123",
		"email":    "john@example.com",
		"token":    "abc.def.ghi",
	}

	redacted := RedactBody(body, nil)
	if redacted["name"] != "John" {
		t.Errorf("name = %v, want John", redacted["name"])
	}
	if redacted["email"] != "john@example.com" {
		t.Errorf("email = %v, should not be redacted by default", redacted["email"])
	}
	if redacted["password"] != "[REDACTED]" {
		t.Errorf("password = %v, want [REDACTED]", redacted["password"])
	}
	if redacted["token"] != "[REDACTED]" {
		t.Errorf("token = %v, want [REDACTED]", redacted["token"])
	}
}

func TestRedactBody_customFields(t *testing.T) {
	body := map[string]any{
		"name":  "John",
		"email": "john@example.com",
		"phone": "555-1234",
	}

	redacted := RedactBody(body, []string{"email", "phone"})
	if redacted["name"] != "John" {
		t.Errorf("name = %v, want John", redacted["name"])
	}
	if redacted["email"] != "[REDACTED]" {
		t.Errorf("email = %v, want [REDACTED]", redacted["email"])
	}
	if redacted["phone"] != "[REDACTED]" {
		t.Errorf("phone = %v, want [REDACTED]", redacted["phone"])
	}
}

func TestRedactBody_nested(t *testing.T) {
	body := map[string]any{
		"user": map[string]any{
			"name":     "John",
			"password": "secret123",
		},
		"metadata": "some value",
	}

	redacted := RedactBody(body, nil)
	nested, ok := redacted["user"].(map[string]any)
	if !ok {
		t.Fatal("user should be a nested map")
	}
	if nested["name"] != "John" {
		t.Errorf("user.name = %v, want John", nested["name"])
	}
	if nested["password"] != "[REDACTED]" {
		t.Errorf("user.password = %v, want [REDACTED]", nested["password"])
	}
}

func TestRedactBody_nil(t *testing.T) {
	if result := RedactBody(nil, nil); result != nil {
		t.Errorf("RedactBody(nil) = %v, want nil", result)
	}
}

func TestRedactBody_doesNotMutateOriginal(t *testing.T) {
	body := map[string]any{
		"password": "secret123",
		"name":     "John",
	}

	_ = RedactBody(body, nil)

	if body["password"] != "secret123" {
		t.Errorf("original body was mutated: password = %v", body["password"])
	}
}

func TestNewLogger_allLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := config.ObservabilityConfig{LogLevel: level}
			logger, err := NewLogger(cfg)
			if err != nil {
				t.Fatalf("NewLogger(%q) error = %v", level, err)
			}
			defer logger.Sync()

			expected, _ := zapcore.ParseLevel(level)
			if !logger.Core().Enabled(expected) {
				t.Errorf("level %q should be enabled", level)
			}
		})
	}
}
