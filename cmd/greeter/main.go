// Command greeter is the aegis-greeter HTTP service. It answers GET /
// with a greeting that echoes the optional ?name= parameter (or the
// caller IP), exposes /healthz (liveness) and /readyz (drain-aware
// readiness), drains in-flight requests on SIGTERM, and emits OTel
// traces + metrics to a local Grafana Alloy DaemonSet plus continuous
// profiles to Pyroscope. Cross-repo observability contract:
// aegis-stateless issue #1.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"github.com/BinHsu/aegis-greeter/internal/handlers"
	"github.com/BinHsu/aegis-greeter/internal/metrics"
	"github.com/BinHsu/aegis-greeter/internal/telemetry"
)

const (
	// shutdownDeadline is the budget for graceful shutdown: in-flight
	// requests draining + OTel provider flushes. K8s default
	// terminationGracePeriodSeconds is 30 s; 25 s leaves headroom for
	// the kubelet SIGKILL. See ADR AG-02.
	shutdownDeadline = 25 * time.Second

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second

	defaultAddr        = ":8080"
	defaultServiceName = "aegis-greeter"

	// samplerRatio at 1.0 is intentional for take-home scope (low
	// volume, every trace is interesting). Production tradeoff would
	// be 0.01 + 100% errors via parentbased(traceidratio(0.01)) with
	// an error sampler; documented as a follow-up, not implemented.
	samplerRatio = 1.0
)

// Version and Commit are populated at build time via
// -ldflags="-X main.Version=$TAG -X main.Commit=$SHA". Defaults match
// "ran from a dev shell" state.
var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("aegis-greeter exited with error", "err", err)
		os.Exit(1)
	}
}

// run wires the service and serves until ctx is cancelled (SIGINT /
// SIGTERM) or the server fails, then drains gracefully. It is the
// testable seam beneath main: a caller drives shutdown by cancelling
// ctx instead of delivering a signal.
func run(ctx context.Context) error {
	// Phase 1: bootstrap logger — plain JSON, no trace context yet.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel()})))

	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("os.Hostname failed; using fallback", "err", err)
		hostname = "unknown"
	}

	serviceName := envOrDefault("OTEL_SERVICE_NAME", defaultServiceName)
	helloTag := os.Getenv("HELLO_TAG")
	pod := os.Getenv("POD_NAME")
	node := os.Getenv("NODE_NAME")

	// Phase 2: OTel providers.
	providers, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:  serviceName,
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		SamplerRatio: samplerRatio,
	})
	if err != nil {
		return fmt.Errorf("telemetry init: %w", err)
	}
	otel.SetTracerProvider(providers.Tracer)
	otel.SetMeterProvider(providers.Meter)

	// Phase 3: swap to the trace-context-aware logger.
	slog.SetDefault(slog.New(telemetry.NewTraceContextHandler(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel()}),
		pod, node,
	)))

	// Phase 4: Pyroscope. Fail-soft — a missing profile pipeline must
	// not block startup.
	profiler, err := telemetry.StartProfiler(telemetry.ProfilerConfig{
		ApplicationName: serviceName,
		Endpoint:        os.Getenv("PYROSCOPE_ENDPOINT"),
	})
	if err != nil {
		slog.Warn("pyroscope start failed; continuing without profiles", "err", err)
	}

	// Phase 5: custom metric instruments.
	instruments, err := metrics.New(providers.Meter.Meter(serviceName), Version, Commit)
	if err != nil {
		return fmt.Errorf("metrics init: %w", err)
	}

	slog.Info("starting aegis-greeter",
		"tag", helloTag,
		"hostname", hostname,
		"version", Version,
		"commit", Commit,
		"service", serviceName,
	)

	addr := envOrDefault("LISTEN_ADDR", defaultAddr)

	readiness := handlers.NewReadiness()
	greeter := &handlers.Greeter{Hostname: hostname, Tag: helloTag, Recorder: instruments}

	mux := http.NewServeMux()
	mux.Handle("/", otelhttp.NewHandler(greeter, "greeter"))
	mux.Handle("/healthz", handlers.Healthz())
	mux.Handle("/readyz", readiness)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	return serve(ctx, srv, ln, readiness, shutdownDeadline, func(shutdownCtx context.Context) {
		if err := providers.Shutdown(shutdownCtx); err != nil {
			slog.Error("provider flush failed", "err", err)
		}
		if err := profiler.Stop(); err != nil {
			slog.Warn("pyroscope stop failed", "err", err)
		}
	})
}

// serve runs srv on ln until ctx is cancelled or the server fails. On
// ctx cancellation it drains: readiness flips to 503 (the orchestrator
// stops routing), then http.Server.Shutdown lets in-flight requests
// finish — refusing new connections — within gracePeriod. cleanup runs
// after the HTTP drain (provider flush, profiler stop).
//
// serve returns nil on a clean drain, a wrapped error if the server
// failed outright, or a wrapped error if the drain exceeded gracePeriod
// (in-flight requests still running when the budget expired).
func serve(
	ctx context.Context,
	srv *http.Server,
	ln net.Listener,
	readiness *handlers.Readiness,
	gracePeriod time.Duration,
	cleanup func(context.Context),
) error {
	readiness.SetReady(true)
	slog.Info("listening", "addr", ln.Addr().String())

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("server error: %w", err)
	}

	readiness.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracePeriod)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if cleanup != nil {
		cleanup(shutdownCtx)
	}
	if err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	slog.Info("aegis-greeter stopped")
	return nil
}

// parseLogLevel reads LOG_LEVEL (DEBUG / INFO / WARN / ERROR); INFO by
// default.
func parseLogLevel() slog.Level {
	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(raw)); err == nil {
			return parsed
		}
	}
	return slog.LevelInfo
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
