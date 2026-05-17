package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// MaxNameLen is the byte-length cap on the ?name= URL parameter. Requests
// above the cap are rejected with 400 Bad Request rather than truncated
// silently — see ADR AG-02.
const MaxNameLen = 256

// ResponseRecorder is the metric sink the Greeter calls after each
// successful response. The handlers package depends on this interface,
// not on the metrics package directly — decoupling lets unit tests
// construct a Greeter without dragging the OTel SDK in.
type ResponseRecorder interface {
	RecordResponse(ctx context.Context, personalized bool)
}

// Greeter answers GET / with "Hello, <addressee>! I'm <hostname> [<tag>]",
// where addressee is the ?name= query parameter when present and the
// caller IP otherwise, and <tag> is the HELLO_TAG value — the brief's
// "unique tag". The "[<tag>]" suffix is omitted when Tag is empty.
// Recorder may be nil, in which case the metric emission is skipped —
// useful in tests and in degraded telemetry configurations.
type Greeter struct {
	Hostname string
	Tag      string
	Recorder ResponseRecorder
}

// ServeHTTP implements http.Handler.
func (g *Greeter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if len(name) > MaxNameLen {
		http.Error(w, "name parameter exceeds maximum length", http.StatusBadRequest)
		return
	}

	personalized := name != ""
	addressee := name
	if !personalized {
		addressee = GetIPFromRequest(r)
	}

	identity := g.Hostname
	if g.Tag != "" {
		identity += " [" + g.Tag + "]"
	}
	fmt.Fprintf(w, "Hello, %s! I'm %s\n", addressee, identity)

	if g.Recorder != nil {
		g.Recorder.RecordResponse(r.Context(), personalized)
	}
}

// GetIPFromRequest extracts the caller IP. When X-Forwarded-For is present
// (typical behind L7 load balancers including ALB and Envoy), the leftmost
// value is taken — the original client. Otherwise RemoteAddr is used.
func GetIPFromRequest(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := strings.IndexByte(fwd, ','); idx >= 0 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	return r.RemoteAddr
}
