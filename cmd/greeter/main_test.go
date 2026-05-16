package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BinHsu/aegis-greeter/internal/handlers"
)

// Graceful-shutdown BVA. The boundary value B is the shutdown grace
// period. B-1 ("under") — a request that finishes inside the budget
// drains cleanly. B+1 ("over") — a request still running when the
// budget expires is abandoned and serve reports an error. B itself
// ("at", the deadline instant) is not separately observable: a request
// finishing exactly as the context deadline fires resolves to under or
// over by scheduler microseconds, so the two cases below bracket it.
// This closes the graceful-shutdown BVA deferred in backlog A4 item 10.

// TestServe_GracefulDrain_UnderDeadline — a request in flight when the
// shutdown signal arrives finishes cleanly because it completes within
// the grace period. Readiness flips to 503 the moment the drain starts,
// and serve returns nil.
func TestServe_GracefulDrain_UnderDeadline(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "drained")
	})

	readiness := handlers.NewReadiness()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- serve(ctx, srv, ln, readiness, 5*time.Second, nil) }()

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/slow")
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()
	<-entered // the request is now in flight inside the handler

	if got := readyCode(readiness); got != http.StatusOK {
		t.Fatalf("readiness before shutdown: got %d, want 200", got)
	}

	cancel() // SIGTERM equivalent

	// serve flips readiness to 503 before it starts draining.
	waitFor(t, 2*time.Second, func() bool {
		return readyCode(readiness) == http.StatusServiceUnavailable
	})

	close(release) // the in-flight request finishes — well under the 5s budget

	select {
	case resp := <-respCh:
		if resp.StatusCode != http.StatusOK {
			t.Errorf("in-flight response: got %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if string(body) != "drained" {
			t.Errorf("in-flight body: got %q, want %q", body, "drained")
		}
	case err := <-errCh:
		t.Fatalf("in-flight request errored instead of draining: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request did not complete")
	}

	select {
	case err := <-serveDone:
		if err != nil {
			t.Errorf("serve returned error on a clean drain: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not return after the drain")
	}
}

// TestServe_GracefulDrain_OverDeadline — a request still in flight past
// the grace period is abandoned: serve returns a non-nil error rather
// than blocking forever.
func TestServe_GracefulDrain_OverDeadline(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) }) // unblock the handler goroutine at the end

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	})

	readiness := handlers.NewReadiness()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- serve(ctx, srv, ln, readiness, 200*time.Millisecond, nil) }()

	go func() { _, _ = http.Get("http://" + addr + "/slow") }()
	<-entered // the request is in flight, and is held past the 200ms budget

	cancel() // SIGTERM equivalent

	select {
	case err := <-serveDone:
		if err == nil {
			t.Error("serve returned nil; expected a shutdown-deadline-exceeded error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not return after the grace period expired")
	}
}

// readyCode calls the readiness handler in-process and returns its
// status code — 200 ready, 503 draining/not-ready.
func readyCode(r *handlers.Readiness) int {
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	return rr.Code
}

// waitFor polls cond every 5ms until it is true or timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

// TestParseLogLevel covers LOG_LEVEL parsing: the four canonical
// levels, an empty value (treated the same as unset → INFO), and an
// unparseable value (falls back to INFO rather than erroring).
func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want slog.Level
	}{
		{"empty / unset", "", slog.LevelInfo},
		{"DEBUG", "DEBUG", slog.LevelDebug},
		{"INFO", "INFO", slog.LevelInfo},
		{"WARN", "WARN", slog.LevelWarn},
		{"ERROR", "ERROR", slog.LevelError},
		{"unparseable falls back to INFO", "loud", slog.LevelInfo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tc.env)
			if got := parseLogLevel(); got != tc.want {
				t.Errorf("LOG_LEVEL=%q: got %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}
