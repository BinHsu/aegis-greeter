package handlers

import (
	"net/http"
	"sync/atomic"
)

// Healthz returns an HTTP handler that always answers 200 once the process
// is up. K8s liveness probe target — failure here triggers a pod restart.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

// Readiness tracks whether the HTTP server is accepting traffic. Flipped
// to true once the listener binds, and back to false on SIGTERM so the
// readiness probe drains connections before pod termination.
//
// K8s readiness probe target — failure here removes the pod from the
// Service endpoints without restarting it.
type Readiness struct {
	ready atomic.Bool
}

// NewReadiness returns a Readiness in the not-ready state.
func NewReadiness() *Readiness {
	return &Readiness{}
}

// SetReady atomically flips the readiness state.
func (r *Readiness) SetReady(ready bool) {
	r.ready.Store(ready)
}

// ServeHTTP implements http.Handler — 200 when ready, 503 otherwise.
func (r *Readiness) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if r.ready.Load() {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}
