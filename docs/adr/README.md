# Architecture Decision Records

Four thematic records covering the decisions behind aegis-greeter. Each
consolidates one coherent area of the project into a single narrative;
the rejected alternatives and the deferral triggers are part of every
record, not an afterthought.

| ADR | Theme | Covers |
|---|---|---|
| [AG-01](AG-01-toolchain-and-layout.md) | Toolchain & project layout | Project-local toolchain, newest-stable Go pin, `cmd/` + `internal/` layout |
| [AG-02](AG-02-http-service-design.md) | HTTP service design | Bounded `?name=` input, drain-aware readiness, graceful shutdown |
| [AG-03](AG-03-container-and-delivery.md) | Container image & delivery | Distroless multi-stage build, immutable SHA tags, OIDC CI/CD, cross-repo commit-back |
| [AG-04](AG-04-observability.md) | Observability | `log/slog` + Loki contract, OpenTelemetry metrics & traces, Pyroscope, fail-soft exporters |

App-side records use the `AG-` prefix; the sibling infrastructure repo
(`aegis-stateless`) uses `AS-`. The two-digit numbering marks these as
the consolidated set — each record is a coherent topic, not a single
granular decision.
