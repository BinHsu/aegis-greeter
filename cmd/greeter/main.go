// Command greeter is the aegis-greeter HTTP service. It answers GET /
// with a greeting that echoes the optional ?name= parameter (or the
// caller IP), exposes /healthz (liveness) and /readyz (drain-aware
// readiness), and drains in-flight requests on SIGTERM within the
// graceful shutdown deadline.
//
// Observability instrumentation (OTel SDK + Pyroscope) lands in the
// next commit per backlog Step A3b — this commit is stdlib-only.
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

	"github.com/BinHsu/aegis-greeter/internal/handlers"
)

const (
	// shutdownDeadline is the budget for in-flight requests to complete
	// after SIGTERM. K8s default terminationGracePeriodSeconds is 30 s;
	// 25 s leaves headroom for the OTel provider flushes that Step A3b
	// will add to the shutdown path. See ADR AG-0004.
	shutdownDeadline = 25 * time.Second

	// HTTP server timeouts. ReadHeaderTimeout is the slowloris defense;
	// the rest are sensible defaults so the server cannot be tied up
	// indefinitely by a slow or stuck peer.
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second

	defaultAddr = ":8080"
)

func main() {
	slog.SetDefault(initLogger())

	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("os.Hostname failed; using fallback", "err", err)
		hostname = "unknown"
	}

	slog.Info("starting aegis-greeter",
		"tag", os.Getenv("HELLO_TAG"),
		"hostname", hostname,
	)

	addr := defaultAddr
	if a := os.Getenv("LISTEN_ADDR"); a != "" {
		addr = a
	}

	readiness := handlers.NewReadiness()
	mux := http.NewServeMux()
	mux.Handle("/", &handlers.Greeter{Hostname: hostname})
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

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
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
		os.Exit(1)
	}

	slog.Info("aegis-greeter stopped")
}

// initLogger returns a slog.Logger that emits JSON to stdout. Level is
// taken from LOG_LEVEL (DEBUG / INFO / WARN / ERROR); INFO by default.
// The handler is plain stdlib for now; Step A3b wraps it to inject
// OTel trace_id / span_id from the request context.
func initLogger() *slog.Logger {
	level := slog.LevelInfo
	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(raw)); err == nil {
			level = parsed
		}
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
