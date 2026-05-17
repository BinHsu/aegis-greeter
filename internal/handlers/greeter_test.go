package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BinHsu/aegis-greeter/internal/handlers"
)

// TestGreeter_URLParam_BVA exercises the ?name= length boundary: empty
// (fallback to IP), max-allowed (echoes), and max+1 (400). Per the BVA
// guardrail in CLAUDE.md, equivalence-class-only tests are insufficient
// here — the off-by-one is the failure mode worth catching.
func TestGreeter_URLParam_BVA(t *testing.T) {
	t.Parallel()
	g := &handlers.Greeter{Hostname: "test-host"}

	cases := []struct {
		name       string
		param      string
		wantStatus int
		wantBody   string
	}{
		{"empty: falls back to caller IP", "", http.StatusOK, "Hello, 1.2.3.4! I'm test-host\n"},
		{"single char", "a", http.StatusOK, "Hello, a! I'm test-host\n"},
		{"max allowed (256)", strings.Repeat("a", handlers.MaxNameLen), http.StatusOK, "Hello, " + strings.Repeat("a", handlers.MaxNameLen) + "! I'm test-host\n"},
		{"max+1 (257) rejected", strings.Repeat("a", handlers.MaxNameLen+1), http.StatusBadRequest, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/?name="+tc.param, nil)
			req.RemoteAddr = "1.2.3.4"
			rr := httptest.NewRecorder()

			g.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusOK && rr.Body.String() != tc.wantBody {
				t.Errorf("body: got %q, want %q", rr.Body.String(), tc.wantBody)
			}
		})
	}
}

// TestGetIPFromRequest_BVA covers the three boundary states of the
// X-Forwarded-For header: unset, single value, multi-value comma list.
func TestGetIPFromRequest_BVA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		fwd        string
		remoteAddr string
		want       string
	}{
		{"unset: returns RemoteAddr", "", "1.2.3.4:5678", "1.2.3.4:5678"},
		{"single value", "203.0.113.7", "10.0.0.1:5678", "203.0.113.7"},
		{"multi value: returns leftmost (original client)", "203.0.113.7, 198.51.100.1, 10.0.0.42", "10.0.0.1:5678", "203.0.113.7"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.fwd != "" {
				req.Header.Set("X-Forwarded-For", tc.fwd)
			}

			got := handlers.GetIPFromRequest(req)

			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGreeter_Tag covers the HELLO_TAG-in-response behaviour: a
// non-empty tag appears as a "[tag]" suffix after the hostname; an
// empty tag — the boundary case — omits the suffix entirely.
func TestGreeter_Tag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		tag  string
		want string
	}{
		{"tag present", "eu-central-1", "Hello, Operator! I'm test-host [eu-central-1]\n"},
		{"tag empty: suffix omitted", "", "Hello, Operator! I'm test-host\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &handlers.Greeter{Hostname: "test-host", Tag: tc.tag}
			req := httptest.NewRequest(http.MethodGet, "/?name=Operator", nil)
			rr := httptest.NewRecorder()

			g.ServeHTTP(rr, req)

			if rr.Body.String() != tc.want {
				t.Errorf("body: got %q, want %q", rr.Body.String(), tc.want)
			}
		})
	}
}
