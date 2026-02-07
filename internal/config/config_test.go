package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_valid(t *testing.T) {
	cfg, err := Load("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 15s", cfg.Server.ReadTimeout)
	}
	if cfg.Identity.Issuer != "https://auth.example.com" {
		t.Errorf("Identity.Issuer = %q", cfg.Identity.Issuer)
	}
	if cfg.Identity.JWKSURL != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("Identity.JWKSURL = %q", cfg.Identity.JWKSURL)
	}
	if cfg.Identity.Audience != "thesa-bff" {
		t.Errorf("Identity.Audience = %q", cfg.Identity.Audience)
	}
	if len(cfg.Identity.Algorithms) != 2 {
		t.Errorf("Identity.Algorithms = %v, want 2 entries", cfg.Identity.Algorithms)
	}
	if !cfg.Definitions.HotReload {
		t.Error("Definitions.HotReload = false, want true")
	}
	if len(cfg.Specs.Sources) != 1 {
		t.Errorf("Specs.Sources = %d entries, want 1", len(cfg.Specs.Sources))
	}

	svc, ok := cfg.Services["orders-svc"]
	if !ok {
		t.Fatal("Services[orders-svc] not found")
	}
	if svc.BaseURL != "https://orders.internal" {
		t.Errorf("orders-svc.BaseURL = %q", svc.BaseURL)
	}
	if svc.Timeout != 10*time.Second {
		t.Errorf("orders-svc.Timeout = %v, want 10s", svc.Timeout)
	}
	if svc.CircuitBreaker.FailureThreshold != 5 {
		t.Errorf("orders-svc.CircuitBreaker.FailureThreshold = %d, want 5", svc.CircuitBreaker.FailureThreshold)
	}
}

func TestLoad_missing_file(t *testing.T) {
	_, err := Load("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("Load() with missing file should return error")
	}
}

func TestLoad_missing_identity(t *testing.T) {
	_, err := Load("testdata/missing_identity.yaml")
	if err == nil {
		t.Fatal("Load() with missing identity should return error")
	}
}

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Server.Port != 8080 {
		t.Errorf("default Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Capability.Cache.TTL != 5*time.Minute {
		t.Errorf("default Capability.Cache.TTL = %v, want 5m", cfg.Capability.Cache.TTL)
	}
	if cfg.Observability.LogLevel != "info" {
		t.Errorf("default LogLevel = %q, want info", cfg.Observability.LogLevel)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("THESA_SERVER_PORT", "3000")
	t.Setenv("THESA_IDENTITY_ISSUER", "https://env-issuer.com")
	t.Setenv("THESA_IDENTITY_JWKS_URL", "https://env-issuer.com/.well-known/jwks.json")
	t.Setenv("THESA_IDENTITY_AUDIENCE", "env-audience")
	t.Setenv("THESA_OBSERVABILITY_LOG_LEVEL", "error")

	cfg, err := Load("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want 3000 (env override)", cfg.Server.Port)
	}
	if cfg.Identity.Issuer != "https://env-issuer.com" {
		t.Errorf("Identity.Issuer = %q, want env override", cfg.Identity.Issuer)
	}
	if cfg.Identity.Audience != "env-audience" {
		t.Errorf("Identity.Audience = %q, want env override", cfg.Identity.Audience)
	}
	if cfg.Observability.LogLevel != "error" {
		t.Errorf("LogLevel = %q, want error (env override)", cfg.Observability.LogLevel)
	}
}

func TestValidate_invalid_port(t *testing.T) {
	cfg := Defaults()
	cfg.Identity.Issuer = "https://auth.example.com"
	cfg.Identity.JWKSURL = "https://auth.example.com/.well-known/jwks.json"
	cfg.Identity.Audience = "thesa-bff"
	cfg.Server.Port = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() with port 0 should return error")
	}
}

func TestLoad_env_priority_over_file(t *testing.T) {
	// File sets port 9090, env sets 5555 — env wins
	t.Setenv("THESA_SERVER_PORT", "5555")
	// Ensure identity fields are set so validation passes
	_ = os.Setenv("THESA_IDENTITY_ISSUER", "")
	_ = os.Setenv("THESA_IDENTITY_JWKS_URL", "")
	_ = os.Setenv("THESA_IDENTITY_AUDIENCE", "")

	cfg, err := Load("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Port != 5555 {
		t.Errorf("Server.Port = %d, want 5555 (env override beats file)", cfg.Server.Port)
	}
}
