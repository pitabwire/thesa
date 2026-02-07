// Package config loads and validates application configuration from YAML files
// and environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root application configuration.
type Config struct {
	Server        ServerConfig               `yaml:"server"`
	Identity      IdentityConfig             `yaml:"identity"`
	Definitions   DefinitionsConfig          `yaml:"definitions"`
	Specs         SpecsConfig                `yaml:"specs"`
	Services      map[string]ServiceConfig   `yaml:"services"`
	Capability    CapabilityConfig           `yaml:"capability"`
	Workflow      WorkflowConfig             `yaml:"workflow"`
	Idempotency   IdempotencyConfig          `yaml:"idempotency"`
	Search        SearchConfig               `yaml:"search"`
	Lookup        LookupCacheConfig          `yaml:"lookup"`
	Observability ObservabilityConfig        `yaml:"observability"`
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

// IdentityConfig describes JWT and identity provider settings.
type IdentityConfig struct {
	Issuer      string            `yaml:"issuer"`
	Audience    string            `yaml:"audience"`
	JWKSURL     string            `yaml:"jwks_url"`
	JWKSCacheTTL time.Duration   `yaml:"jwks_cache_ttl"`
	Algorithms  []string          `yaml:"algorithms"`
	ClaimPaths  map[string]string `yaml:"claim_paths"`
}

// DefinitionsConfig describes where to find definition YAML files.
type DefinitionsConfig struct {
	Directories      []string `yaml:"directories"`
	HotReload        bool     `yaml:"hot_reload"`
	StrictChecksums  bool     `yaml:"strict_checksums"`
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
	BaseURL        string              `yaml:"base_url"`
	Timeout        time.Duration       `yaml:"timeout"`
	Auth           ServiceAuthConfig   `yaml:"auth"`
	Pagination     PaginationConfig    `yaml:"pagination"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Retry          RetryConfig         `yaml:"retry"`
}

// ServiceAuthConfig describes authentication for backend calls.
type ServiceAuthConfig struct {
	Strategy      string `yaml:"strategy"`
	ClientID      string `yaml:"client_id"`
	TokenEndpoint string `yaml:"token_endpoint"`
}

// PaginationConfig describes how a backend service paginates.
type PaginationConfig struct {
	Style        string `yaml:"style"`
	PageParam    string `yaml:"page_param"`
	SizeParam    string `yaml:"size_param"`
	SortParam    string `yaml:"sort_param"`
	SortDirParam string `yaml:"sort_dir_param"`
	CursorParam  string `yaml:"cursor_param"`
}

// CircuitBreakerConfig describes circuit breaker settings per service.
type CircuitBreakerConfig struct {
	FailureThreshold   int           `yaml:"failure_threshold"`
	SuccessThreshold   int           `yaml:"success_threshold"`
	Timeout            time.Duration `yaml:"timeout"`
	ErrorRateThreshold float64       `yaml:"error_rate_threshold"`
	ErrorRateWindow    time.Duration `yaml:"error_rate_window"`
}

// RetryConfig describes retry settings per service.
type RetryConfig struct {
	MaxAttempts       int           `yaml:"max_attempts"`
	BackoffInitial    time.Duration `yaml:"backoff_initial"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier"`
	BackoffMax        time.Duration `yaml:"backoff_max"`
	IdempotentOnly    bool          `yaml:"idempotent_only"`
}

// CapabilityConfig describes authorization settings.
type CapabilityConfig struct {
	Evaluator        string          `yaml:"evaluator"`
	StaticPolicyFile string          `yaml:"static_policy_file"`
	Cache            CacheConfig     `yaml:"cache"`
	OPA              OPAConfig       `yaml:"opa"`
}

// OPAConfig describes OPA policy engine settings.
type OPAConfig struct {
	URL        string        `yaml:"url"`
	PolicyPath string        `yaml:"policy_path"`
	Timeout    time.Duration `yaml:"timeout"`
}

// CacheConfig describes cache settings.
type CacheConfig struct {
	TTL        time.Duration `yaml:"ttl"`
	MaxEntries int           `yaml:"max_entries"`
}

// WorkflowConfig describes workflow engine settings.
type WorkflowConfig struct {
	Enabled              bool               `yaml:"enabled"`
	Store                WorkflowStoreConfig `yaml:"store"`
	TimeoutCheckInterval time.Duration      `yaml:"timeout_check_interval"`
}

// WorkflowStoreConfig describes workflow persistence settings.
type WorkflowStoreConfig struct {
	Driver          string        `yaml:"driver"`
	DSNEnv          string        `yaml:"dsn_env"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// IdempotencyConfig describes idempotency store settings.
type IdempotencyConfig struct {
	Enabled bool                 `yaml:"enabled"`
	Store   IdempotencyStoreConfig `yaml:"store"`
}

// IdempotencyStoreConfig describes idempotency persistence settings.
type IdempotencyStoreConfig struct {
	Driver     string        `yaml:"driver"`
	AddrEnv    string        `yaml:"addr_env"`
	DB         int           `yaml:"db"`
	DefaultTTL time.Duration `yaml:"default_ttl"`
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
		Identity: IdentityConfig{
			JWKSCacheTTL: 1 * time.Hour,
			Algorithms:   []string{"RS256"},
			ClaimPaths: map[string]string{
				"subject_id": "sub",
				"tenant_id":  "tenant_id",
				"email":      "email",
				"roles":      "roles",
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
			Evaluator: "static",
			Cache: CacheConfig{
				TTL:        5 * time.Minute,
				MaxEntries: 10000,
			},
		},
		Workflow: WorkflowConfig{
			TimeoutCheckInterval: 60 * time.Second,
			Store: WorkflowStoreConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
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

// Load reads a YAML config file, applies environment variable overrides,
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
	if c.Identity.Issuer == "" {
		errs = append(errs, "identity.issuer is required")
	}
	if c.Identity.JWKSURL == "" {
		errs = append(errs, "identity.jwks_url is required")
	}
	if c.Identity.Audience == "" {
		errs = append(errs, "identity.audience is required")
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
	if v := os.Getenv("THESA_IDENTITY_ISSUER"); v != "" {
		cfg.Identity.Issuer = v
	}
	if v := os.Getenv("THESA_IDENTITY_JWKS_URL"); v != "" {
		cfg.Identity.JWKSURL = v
	}
	if v := os.Getenv("THESA_IDENTITY_AUDIENCE"); v != "" {
		cfg.Identity.Audience = v
	}
	if v := os.Getenv("THESA_OBSERVABILITY_LOG_LEVEL"); v != "" {
		cfg.Observability.LogLevel = v
	}
	if v := os.Getenv("THESA_CAPABILITY_EVALUATOR"); v != "" {
		cfg.Capability.Evaluator = v
	}
}
