package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

// MaxNameLen is the byte-length cap on the ?name= URL parameter. Requests
// above the cap are rejected with 400 Bad Request rather than truncated
// silently — see ADR AG-0003.
const MaxNameLen = 256

// Greeter answers GET / with "Hello, <addressee>! I'm <hostname>", where
// addressee is the ?name= query parameter when present and the caller IP
// otherwise (preserves the baseline behavior of curl / with no params).
type Greeter struct {
	Hostname string
}

// ServeHTTP implements http.Handler.
func (g *Greeter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if len(name) > MaxNameLen {
		http.Error(w, "name parameter exceeds maximum length", http.StatusBadRequest)
		return
	}

	addressee := name
	if addressee == "" {
		addressee = GetIPFromRequest(r)
	}

	fmt.Fprintf(w, "Hello, %s! I'm %s\n", addressee, g.Hostname)
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
