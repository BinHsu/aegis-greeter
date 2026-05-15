# Architecture Decision Records

Records of the significant decisions behind `aegis-greeter`. App-side
ADRs use the `AG-NNNN` prefix; the sibling infrastructure repo
(`aegis-stateless`) uses `AS-NNNN`.

| ADR | Decision | Status |
|-----|----------|--------|
| [AG-0001](AG-0001-project-local-toolchain.md) | Project-local toolchain via `tools.go` + `GOBIN` + `Makefile` | Accepted |
| [AG-0002](AG-0002-slog-over-zap.md) | `log/slog` over `zap` / `zerolog` | Accepted |
| [AG-0003](AG-0003-url-parameter-max-length.md) | URL parameter capped at 256 bytes, reject over | Accepted |
| [AG-0004](AG-0004-graceful-shutdown-deadline.md) | 25-second graceful shutdown deadline | Accepted |
| [AG-0005](AG-0005-distroless-static-debian12.md) | Distroless `static-debian12` runtime base | Accepted |
| [AG-0006](AG-0006-cross-repo-commit-back.md) | Cross-repo commit-back via fine-grained PAT | Accepted |
| [AG-0007](AG-0007-image-tagging-strategy.md) | Image tag strategy: short SHA + `latest` | Accepted |
| [AG-0008](AG-0008-grafana-log-format-contract.md) | Grafana-stack structured log format contract | Accepted |
| [AG-0009](AG-0009-otel-pyroscope-adoption.md) | OpenTelemetry SDK + `pyroscope-go` adoption | Accepted |
| [AG-0010](AG-0010-go-layout-cmd-internal.md) | Go layout: `cmd/greeter` + `internal/` | Accepted |
| [AG-0011](AG-0011-go-toolchain-pin-policy.md) | Go toolchain pin policy: newest stable | Accepted |

Each record states the context that forced a decision, the decision
itself, and the consequences accepted in taking it.
