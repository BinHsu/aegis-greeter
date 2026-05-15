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
	// the kubelet SIGKILL. See ADR AG-0004.
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
	// Phase 1: bootstrap logger — plain JSON, no trace context yet.
	// Used for any error before OTel providers are up.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel()})))

	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("os.Hostname failed; using fallback", "err", err)
		hostname = "unknown"
	}

	serviceName := envOrDefault("OTEL_SERVICE_NAME", defaultServiceName)
	pod := os.Getenv("POD_NAME")
	node := os.Getenv("NODE_NAME")

	// Phase 2: OTel providers.
	ctx := context.Background()
	providers, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:  serviceName,
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		SamplerRatio: samplerRatio,
	})
	if err != nil {
		slog.Error("telemetry init failed", "err", err)
		os.Exit(1)
	}
	otel.SetTracerProvider(providers.Tracer)
	otel.SetMeterProvider(providers.Meter)

	// Phase 3: swap to trace-context-aware logger now that the OTel
	// providers exist. From here on, slog records emitted with a
	// request ctx will carry trace_id + span_id.
	slog.SetDefault(slog.New(telemetry.NewTraceContextHandler(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel()}),
		pod, node,
	)))

	// Phase 4: Pyroscope. Fail-soft — a missing profile pipeline must
	// not block the request path.
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
		slog.Error("metrics init failed", "err", err)
		os.Exit(1)
	}

	slog.Info("starting aegis-greeter",
		"tag", os.Getenv("HELLO_TAG"),
		"hostname", hostname,
		"version", Version,
		"commit", Commit,
		"service", serviceName,
	)

	addr := envOrDefault("LISTEN_ADDR", defaultAddr)

	readiness := handlers.NewReadiness()
	greeter := &handlers.Greeter{Hostname: hostname, Recorder: instruments}

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
		slog.Error("listen failed", "addr", addr, "err", err)
		os.Exit(1)
	}
	readiness.SetReady(true)
	slog.Info("listening", "addr", ln.Addr().String())

	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case <-signalCtx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		slog.Error("server error", "err", err)
		os.Exit(1)
	}

	readiness.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownDeadline)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
	if err := providers.Shutdown(shutdownCtx); err != nil {
		slog.Error("provider flush failed", "err", err)
	}
	if err := profiler.Stop(); err != nil {
		slog.Warn("pyroscope stop failed", "err", err)
	}

	slog.Info("aegis-greeter stopped")
}

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
