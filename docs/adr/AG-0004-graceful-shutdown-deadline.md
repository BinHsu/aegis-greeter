# AG-0004: 25-second graceful shutdown deadline

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

On `SIGTERM` the service must drain in-flight HTTP requests and flush
the OpenTelemetry trace and metric providers before the process exits.
Kubernetes sends `SIGTERM`, waits `terminationGracePeriodSeconds`
(default 30), then sends `SIGKILL`. Shutdown work must finish inside
that window with margin.

## Decision

The graceful shutdown deadline is **25 seconds** — 5 seconds of
headroom under the Kubernetes 30-second default. The sequence on
`SIGTERM`: flip readiness to 503 (so the pod leaves the Service
endpoints), `http.Server.Shutdown` to drain requests, then flush the
trace and metric providers.

## Consequences

- In-flight requests and telemetry flushes complete before `SIGKILL`.
- 5 seconds of margin absorbs a slow final flush.
- The deadline is coupled to the Kubernetes
  `terminationGracePeriodSeconds`. If the sibling repo lowers that
  value below 30, this constant must drop in step. The coupling is
  noted here so the dependency is not silent.

## Alternatives considered

- **No graceful shutdown — exit immediately on `SIGTERM`** — drops
  in-flight requests and loses the final telemetry batch. Unacceptable
  for a service behind a load balancer.
- **A configurable deadline (env var)** — flexibility nobody has
  asked for. The deadline is derived from one fact, the Kubernetes
  `terminationGracePeriodSeconds`; a hardcoded constant with a
  documented derivation is honest, where an env var would invite an
  inconsistent pair.
- **A pre-shutdown sleep before draining** — a common pattern to let
  endpoint removal propagate before the listener closes. Left out
  because the readiness flip to 503 already happens first; see below.

## Out of scope / when to revisit

- A short pre-shutdown sleep between the readiness flip and
  `http.Server.Shutdown`, to let kube-proxy / endpoint controllers
  observe the 503 before new connections are refused. Revisit if a
  rolling deploy is observed to drop a small number of requests.
- Making the deadline configurable — revisit only if the sibling repo
  sets a non-default `terminationGracePeriodSeconds`.
