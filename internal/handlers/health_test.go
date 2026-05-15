package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BinHsu/aegis-greeter/internal/handlers"
)

// TestHealthz_Always200 asserts the liveness probe is unconditional —
// the handler exists, the process is up, the kernel routes traffic, that
// is what 200 here means. Anything more clever creates a fragile probe.
func TestHealthz_Always200(t *testing.T) {
	t.Parallel()
	h := handlers.Healthz()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestReadiness_StateTransitions_BVA walks the readiness state machine
// through its three meaningful states: pre-listen (503), listening
// (200), post-SIGTERM (503). Same-API check on the same probe — the
// state mutation is the test target.
func TestReadiness_StateTransitions_BVA(t *testing.T) {
	t.Parallel()
	r := handlers.NewReadiness()

	cases := []struct {
		name       string
		setReady   bool
		wantStatus int
	}{
		{"pre-listen (default)", false, http.StatusServiceUnavailable},
		{"listening", true, http.StatusOK},
		{"post-SIGTERM (drain)", false, http.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r.SetReady(tc.setReady)

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("got %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}
