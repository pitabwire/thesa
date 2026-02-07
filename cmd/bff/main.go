// Package main is the entry point for the Thesa BFF server.
// It wires all dependencies together and starts the HTTP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/pitabwire/thesa/internal/capability"
	"github.com/pitabwire/thesa/internal/command"
	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/internal/invoker"
	"github.com/pitabwire/thesa/internal/metadata"
	"github.com/pitabwire/thesa/internal/observability"
	"github.com/pitabwire/thesa/internal/openapi"
	"github.com/pitabwire/thesa/internal/search"
	"github.com/pitabwire/thesa/internal/transport"
	"github.com/pitabwire/thesa/internal/workflow"
)

// Build-time variables set via ldflags:
//
//	go build -ldflags "-X main.version=1.0.0 -X main.commit=abc1234"
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Step 1: Parse CLI flags.
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Step 2: Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		return 1
	}

	// Step 3: Initialize telemetry (logger, tracer, metrics).
	observability.Version = version
	observability.Commit = commit

	logger, err := observability.NewLogger(cfg.Observability)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		return 1
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	tracingShutdown, err := observability.InitTracing(ctx, cfg.Observability.Tracing, "thesa-bff", version)
	if err != nil {
		logger.Fatal("tracing initialization failed", zap.Error(err))
		return 1
	}

	metrics := observability.InitMetrics(prometheus.DefaultRegisterer)

	// Step 4: Load OpenAPI specs and build index.
	oaIndex := openapi.NewIndex()
	specSources := buildSpecSources(cfg.Specs)
	if err := oaIndex.Load(specSources); err != nil {
		logger.Fatal("OpenAPI index load failed", zap.Error(err))
		return 1
	}

	// Step 5: Load definitions, validate, build registry.
	loader := definition.NewLoader()
	defs, err := loader.LoadAll(cfg.Definitions.Directories)
	if err != nil {
		logger.Fatal("definition loading failed", zap.Error(err))
		return 1
	}

	validator := definition.NewValidator()
	verrs := validator.Validate(defs, oaIndex)
	if len(verrs) > 0 {
		for _, ve := range verrs {
			logger.Error("definition validation error", zap.String("error", ve.Error()))
		}
		logger.Fatal("definition validation failed", zap.Int("errors", len(verrs)))
		return 1
	}

	registry := definition.NewRegistry(defs)

	// Step 6: Initialize capability resolver.
	capResolver, err := buildCapabilityResolver(cfg.Capability)
	if err != nil {
		logger.Fatal("capability resolver initialization failed", zap.Error(err))
		return 1
	}

	// Step 7: Initialize workflow store (optional).
	wfStore, wfStoreCloser, err := buildWorkflowStore(ctx, cfg.Workflow, logger)
	if err != nil {
		logger.Fatal("workflow store initialization failed", zap.Error(err))
		return 1
	}

	// Step 8: Initialize idempotency store (optional).
	idempotencyStore, idempotencyCloser := buildIdempotencyStore(cfg.Idempotency, logger)

	// Step 9-10: Build invoker registry.
	sdkHandlers := invoker.NewSDKHandlerRegistry()
	invokerReg := invoker.NewRegistry()
	invokerReg.Register(invoker.NewOpenAPIOperationInvoker(oaIndex, cfg.Services))
	invokerReg.Register(invoker.NewSDKOperationInvoker(sdkHandlers))

	// Step 11: Build providers.
	var cmdOpts []command.CommandExecutorOption
	if idempotencyStore != nil {
		cmdOpts = append(cmdOpts, command.WithIdempotencyStore(idempotencyStore))
	}
	cmdExecutor := command.NewCommandExecutor(registry, invokerReg, oaIndex, cmdOpts...)

	var wfEngine *workflow.Engine
	if cfg.Workflow.Enabled && wfStore != nil {
		wfEngine = workflow.NewEngine(registry, wfStore, invokerReg, capResolver)
	}

	actionProvider := metadata.NewActionProvider()
	menuProvider := metadata.NewMenuProvider(registry, invokerReg)
	pageProvider := metadata.NewPageProvider(registry, invokerReg, actionProvider)
	formProvider := metadata.NewFormProvider(registry, invokerReg, actionProvider)
	searchProvider := search.NewSearchProvider(
		registry, invokerReg,
		cfg.Search.TimeoutPerProvider,
		cfg.Search.MaxResultsPerProvider,
	)
	lookupProvider := search.NewLookupProvider(
		registry, invokerReg,
		cfg.Lookup.Cache.TTL,
		cfg.Lookup.Cache.MaxEntries,
	)

	// Step 12: Build HTTP router.
	jwks := transport.NewJWKSClient(cfg.Identity.JWKSURL, cfg.Identity.JWKSCacheTTL)

	// Build readiness checks using data known at startup.
	specServiceIDs := make([]string, 0, len(specSources))
	for _, s := range specSources {
		specServiceIDs = append(specServiceIDs, s.ServiceID)
	}
	readinessChecks := observability.ReadinessChecks{
		DefinitionsLoaded: func() bool { return len(registry.AllDomains()) > 0 },
		OpenAPILoaded: func() bool {
			for _, svcID := range specServiceIDs {
				if len(oaIndex.AllOperationIDs(svcID)) > 0 {
					return true
				}
			}
			return len(specServiceIDs) == 0 // OK if no specs configured
		},
	}
	if wfStore != nil {
		if hc, ok := wfStore.(observability.HealthChecker); ok {
			readinessChecks.WorkflowStore = hc
		}
	}

	router := transport.NewRouter(transport.Dependencies{
		Config:             cfg,
		Authenticate:       transport.JWTAuthenticator(cfg.Identity, jwks),
		CapabilityResolver: capResolver,
		MenuProvider:       menuProvider,
		PageProvider:       pageProvider,
		FormProvider:       formProvider,
		CommandExecutor:    cmdExecutor,
		WorkflowEngine:     wfEngine,
		SearchProvider:     searchProvider,
		LookupProvider:     lookupProvider,
		HealthHandler:      observability.HandleHealth(),
		ReadyHandler:       observability.HandleReady(readinessChecks),
		MetricsHandler:     observability.Handler(),
	})

	// Wrap router with metrics middleware.
	handler := metrics.MetricsMiddleware(observability.TracingMiddleware(router))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Step 13: Start background tasks.
	bgCtx, bgCancel := context.WithCancel(ctx)
	defer bgCancel()

	if cfg.Workflow.Enabled && wfEngine != nil {
		go runWorkflowTimeoutProcessor(bgCtx, wfEngine, cfg.Workflow.TimeoutCheckInterval, logger)
	}

	// Step 14: Start HTTP server.
	logger.Info("server started",
		zap.Int("port", cfg.Server.Port),
		zap.String("version", version),
		zap.String("commit", commit),
		zap.Int("definitions", len(defs)),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for shutdown signal or server error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown initiated")
	case err := <-errCh:
		logger.Error("server error", zap.Error(err))
		return 1
	}

	// Graceful shutdown sequence.
	shutdownTimeout := cfg.Server.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Stop accepting new connections and drain in-flight requests.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Cancel background tasks.
	bgCancel()

	// Close stores.
	if wfStoreCloser != nil {
		wfStoreCloser()
	}
	if idempotencyCloser != nil {
		idempotencyCloser()
	}

	// Flush telemetry.
	if err := tracingShutdown(shutdownCtx); err != nil {
		logger.Error("tracing shutdown error", zap.Error(err))
	}

	_ = metrics // metrics are scraped by Prometheus, no explicit flush needed.

	logger.Info("shutdown complete")
	return 0
}

// buildSpecSources converts config spec sources to openapi.SpecSource.
func buildSpecSources(specsCfg config.SpecsConfig) []openapi.SpecSource {
	sources := make([]openapi.SpecSource, len(specsCfg.Sources))
	for i, s := range specsCfg.Sources {
		specPath := s.SpecFile
		if specsCfg.Directory != "" && !filepath.IsAbs(specPath) {
			specPath = filepath.Join(specsCfg.Directory, specPath)
		}
		sources[i] = openapi.SpecSource{
			ServiceID: s.ServiceID,
			SpecPath:  specPath,
		}
	}
	return sources
}

// buildCapabilityResolver creates the appropriate resolver based on config.
func buildCapabilityResolver(cfg config.CapabilityConfig) (*capability.Resolver, error) {
	switch cfg.Evaluator {
	case "static", "":
		evaluator, err := capability.NewStaticPolicyEvaluator(cfg.StaticPolicyFile)
		if err != nil {
			return nil, fmt.Errorf("static policy: %w", err)
		}
		return capability.NewResolver(evaluator, cfg.Cache.TTL), nil
	default:
		return nil, fmt.Errorf("unsupported capability evaluator: %q", cfg.Evaluator)
	}
}

// buildWorkflowStore creates the workflow store based on config.
// Returns nil store and closer if workflows are disabled.
func buildWorkflowStore(ctx context.Context, cfg config.WorkflowConfig, logger *zap.Logger) (workflow.WorkflowStore, func(), error) {
	if !cfg.Enabled {
		return nil, nil, nil
	}

	switch cfg.Store.Driver {
	case "memory":
		logger.Info("using in-memory workflow store")
		return workflow.NewMemoryWorkflowStore(), nil, nil
	case "postgres", "":
		dsn := os.Getenv(cfg.Store.DSNEnv)
		if dsn == "" && cfg.Store.DSNEnv != "" {
			return nil, nil, fmt.Errorf("workflow store: %s environment variable not set", cfg.Store.DSNEnv)
		}
		if dsn == "" {
			logger.Warn("workflow store DSN not configured, using in-memory store")
			return workflow.NewMemoryWorkflowStore(), nil, nil
		}

		poolCfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return nil, nil, fmt.Errorf("workflow store: parse DSN: %w", err)
		}
		poolCfg.MaxConns = int32(cfg.Store.MaxOpenConns)
		poolCfg.MinConns = int32(cfg.Store.MaxIdleConns)
		poolCfg.MaxConnLifetime = cfg.Store.ConnMaxLifetime

		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("workflow store: connect: %w", err)
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("workflow store: ping: %w", err)
		}

		store := workflow.NewPgWorkflowStore(pool)
		return store, pool.Close, nil
	default:
		return nil, nil, fmt.Errorf("unsupported workflow store driver: %q", cfg.Store.Driver)
	}
}

// buildIdempotencyStore creates the idempotency store based on config.
func buildIdempotencyStore(cfg config.IdempotencyConfig, logger *zap.Logger) (command.IdempotencyStore, func()) {
	if !cfg.Enabled {
		return nil, nil
	}

	switch cfg.Store.Driver {
	case "memory":
		logger.Info("using in-memory idempotency store")
		return command.NewMemoryIdempotencyStore(), nil
	default:
		logger.Info("using in-memory idempotency store (redis not yet supported)")
		return command.NewMemoryIdempotencyStore(), nil
	}
}

// runWorkflowTimeoutProcessor periodically processes expired workflow steps.
func runWorkflowTimeoutProcessor(ctx context.Context, engine *workflow.Engine, interval time.Duration, logger *zap.Logger) {
	if interval == 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := engine.ProcessTimeouts(ctx); err != nil {
				logger.Error("workflow timeout processing failed", zap.Error(err))
			}
		}
	}
}
