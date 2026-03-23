// Package config loads and validates application configuration from YAML files
// and environment variables.
package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	frameconfig "github.com/pitabwire/frame/config"
	"gopkg.in/yaml.v3"
)

// Config is the root application configuration.
type Config struct {
	frameconfig.ConfigurationDefault `yaml:"-"` // Frame handles infrastructure config via env vars

	Server        ServerConfig             `yaml:"server"`
	Definitions   DefinitionsConfig        `yaml:"definitions"`
	Specs         SpecsConfig              `yaml:"specs"`
	Services      map[string]ServiceConfig `yaml:"services"`
	Capability    CapabilityConfig         `yaml:"capability"`
	Search        SearchConfig             `yaml:"search"`
	Lookup        LookupCacheConfig        `yaml:"lookup"`
	Observability ObservabilityConfig      `yaml:"observability"`
}

// ServerConfig describes HTTP server settings.
type ServerConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	HandlerTimeout  time.Duration `yaml:"handler_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	CORS            CORSConfig    `yaml:"cors"`
}

// CORSConfig describes Cross-Origin Resource Sharing settings.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	MaxAge         int      `yaml:"max_age"`
}

// DefinitionsConfig describes where to find definition YAML files.
type DefinitionsConfig struct {
	Directories     []string `yaml:"directories"`
	HotReload       bool     `yaml:"hot_reload"`
	StrictChecksums bool     `yaml:"strict_checksums"`
}

// SpecsConfig describes where to find OpenAPI specification files.
type SpecsConfig struct {
	Directory string       `yaml:"directory"`
	Sources   []SpecSource `yaml:"sources"`
}

// SpecSource maps a service ID to an OpenAPI spec file.
type SpecSource struct {
	ServiceID string `yaml:"service_id"`
	SpecFile  string `yaml:"spec_file"`
}

// ServiceConfig describes a backend service.
type ServiceConfig struct {
	BaseURL                string        `yaml:"base_url"`
	Timeout                time.Duration `yaml:"timeout"`
	Retry                  RetryConfig   `yaml:"retry"`
	AuthorizationNamespace string        `yaml:"authorization_namespace"`
}

// RetryConfig describes retry settings per service.
type RetryConfig struct {
	MaxAttempts       int           `yaml:"max_attempts"`
	BackoffInitial    time.Duration `yaml:"backoff_initial"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier"`
	BackoffMax        time.Duration `yaml:"backoff_max"`
	IdempotentOnly    bool          `yaml:"idempotent_only"`
}

// CapabilityConfig describes authorization cache settings.
type CapabilityConfig struct {
	Cache CacheConfig `yaml:"cache"`
}

// CacheConfig describes cache settings.
type CacheConfig struct {
	TTL        time.Duration `yaml:"ttl"`
	MaxEntries int           `yaml:"max_entries"`
}

// SearchConfig describes search settings.
type SearchConfig struct {
	TimeoutPerProvider    time.Duration `yaml:"timeout_per_provider"`
	MaxResultsPerProvider int           `yaml:"max_results_per_provider"`
}

// LookupCacheConfig describes lookup cache settings.
type LookupCacheConfig struct {
	Cache CacheConfig `yaml:"cache"`
}

// ObservabilityConfig describes logging, tracing, and metrics settings.
type ObservabilityConfig struct {
	LogLevel string        `yaml:"log_level"`
	Tracing  TracingConfig `yaml:"tracing"`
	Metrics  MetricsConfig `yaml:"metrics"`
}

// TracingConfig describes distributed tracing settings.
type TracingConfig struct {
	Enabled           bool    `yaml:"enabled"`
	Exporter          string  `yaml:"exporter"`
	Endpoint          string  `yaml:"endpoint"`
	SamplingRate      float64 `yaml:"sampling_rate"`
	ForceSampleErrors bool    `yaml:"force_sample_errors"`
}

// MetricsConfig describes Prometheus metrics settings.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// Defaults returns a Config with sensible default values.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			HandlerTimeout:  25 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			CORS: CORSConfig{
				AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Authorization", "Content-Type", "X-Partition-Id",
					"X-Correlation-Id", "X-Idempotency-Key"},
				MaxAge: 86400,
			},
		},
		Definitions: DefinitionsConfig{
			Directories:     []string{"/definitions"},
			StrictChecksums: true,
		},
		Specs: SpecsConfig{
			Directory: "/specs",
		},
		Capability: CapabilityConfig{
			Cache: CacheConfig{
				TTL:        5 * time.Minute,
				MaxEntries: 10000,
			},
		},
		Search: SearchConfig{
			TimeoutPerProvider:    3 * time.Second,
			MaxResultsPerProvider: 50,
		},
		Lookup: LookupCacheConfig{
			Cache: CacheConfig{
				TTL:        5 * time.Minute,
				MaxEntries: 1000,
			},
		},
		Observability: ObservabilityConfig{
			LogLevel: "info",
			Tracing: TracingConfig{
				Exporter:     "otlp",
				SamplingRate: 0.1,
			},
			Metrics: MetricsConfig{
				Enabled: true,
				Path:    "/metrics",
			},
		},
	}
}

// Load reads a YAML config file, populates Frame's embedded config from
// environment variables (OAUTH2_*, LOG_*, etc.), loads OIDC discovery,
// and validates required fields.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	// Populate Frame's embedded ConfigurationDefault from OAUTH2_*, LOG_*, etc. env vars.
	if err := frameconfig.FillEnv(&cfg.ConfigurationDefault); err != nil {
		return nil, fmt.Errorf("config: frame env: %w", err)
	}

	// Load OIDC discovery (fetches JWKS data from the OAuth2 provider).
	if cfg.GetOauth2ServiceURI() != "" {
		if err := cfg.LoadOauth2Config(context.Background()); err != nil {
			return nil, fmt.Errorf("config: oidc discovery: %w", err)
		}
	}

	applyEnvOverrides(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	return cfg, nil
}

// Validate checks that all required fields are present and valid.
func (c *Config) Validate() error {
	var errs []string

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, "server.port must be between 1 and 65535")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// applyEnvOverrides reads THESA_* environment variables and overrides config
// values. Only the most commonly overridden fields are supported.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("THESA_SERVER_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("THESA_OBSERVABILITY_LOG_LEVEL"); v != "" {
		cfg.Observability.LogLevel = v
	}
}
