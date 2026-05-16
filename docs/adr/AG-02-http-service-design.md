# AG-02: HTTP service design

- **Status**: Accepted
- **Date**: 2026-05-16

## Context

The greeter is a small service, but it is operated like a production
one: it sits behind a load balancer, takes caller-supplied input, and
is started and stopped by an orchestrator. Two behaviours decide
whether it survives that environment — how it treats hostile input,
and how it behaves when told to shut down.

## Decisions

### 1. Bounded input, rejected not truncated

`GET /?name=<value>` echoes a caller-supplied value into the response.
The `name` parameter is capped at 256 bytes; a request over the cap is
**rejected with 400 Bad Request**, not silently truncated. 256 bytes is
generous for any real name and small enough to keep the response size
bounded and not attacker-influenced. The boundary is covered by
explicit Boundary Value Analysis — empty, exactly 256, and 257.

A wrong-but-`200` response (truncation) is worse than an honest `400`:
the caller learns their input was invalid instead of receiving a
greeting addressed to a name they did not send.

### 2. Drain-aware lifecycle and graceful shutdown

`/healthz` (liveness) is unconditionally 200 once the process is up.
`/readyz` (readiness) is *drain-aware*: 503 before the listener binds,
503 from the moment `SIGTERM` arrives, and 200 only while the service
can actually serve. That readiness flip is what makes an orchestrator
stop routing new traffic before the listener closes.

On `SIGTERM` the shutdown order is fixed: flip `/readyz` to 503, drain
in-flight requests via `http.Server.Shutdown`, flush the OpenTelemetry
providers, exit. The deadline is **25 seconds** — the Kubernetes
default `terminationGracePeriodSeconds` of 30, minus a 5-second margin
before the kubelet `SIGKILL`. A request still running past the deadline
is abandoned so shutdown cannot hang.

## Consequences

- Response size is bounded, and the caller gets honest status codes.
- In-flight requests and the final telemetry batch complete before
  `SIGKILL`; a rolling deploy does not drop connections already being
  served.
- The 25-second deadline is coupled to the Kubernetes
  `terminationGracePeriodSeconds`. The coupling is stated here so that
  if the sibling repo lowers that value, this constant drops in step
  rather than silently.

## Alternatives considered

- **Silently truncate the over-long name** — a wrong-but-200 response;
  rejected.
- **No application-level input limit** — rely on `net/http`'s header /
  URL size limits; that bounds the request but not the `name` field
  specifically, leaving the contract a standard-library implementation
  detail.
- **413 Payload Too Large instead of 400** — 413 is about the request
  body; an over-long query parameter is a malformed request, so 400 is
  the accurate status.
- **No graceful shutdown — exit immediately on `SIGTERM`** — drops
  in-flight requests and the final telemetry batch.
- **A configurable shutdown deadline** — flexibility nobody asked for;
  the deadline is derived from one fact, so a documented constant is
  honest where an env var would invite an inconsistent pair.

## Out of scope / when to revisit

- A short pre-shutdown sleep between the readiness flip and
  `http.Server.Shutdown`, to let endpoint controllers observe the 503
  before the listener closes — revisit if a rolling deploy is observed
  to drop requests.
- The 256-byte cap is a self-chosen default — revisit if a downstream
  system defines an authoritative maximum for the name field.
