# aegis-greeter

A stateless Go HTTP greeter, packaged and operated like a production
service. It is the application side of a two-repository GitOps system:
this repo holds the service, its tests, the container build, and the
CI/CD that ships images; the sibling repo
[`aegis-stateless`](https://github.com/BinHsu/aegis-stateless) holds the
Terraform, the EKS cluster, ArgoCD, and the Kubernetes manifests.

The service itself is deliberately small. The point of the project is
everything *around* the service — host-isolated tooling, boundary-tested
code, a minimal-surface container, layered quality gates, and a CI
pipeline that builds images and hands off to GitOps without ever
touching the cluster.

## What it does

`GET /` answers `Hello, <name>! I'm <hostname>`. The `name` comes from
the `?name=` query parameter; with no parameter it falls back to the
caller IP (read from `X-Forwarded-For` when present, else `RemoteAddr`).

| Path | Purpose | Behavior |
|---|---|---|
| `GET /?name=<value>` | Greeter | 200 with the greeting. `name` is capped at 256 bytes; longer → 400. |
| `GET /healthz` | Liveness probe | Always 200 once the process is up. |
| `GET /readyz` | Readiness probe | 200 once the listener is bound; 503 before that and after `SIGTERM` so the pod drains before termination. |

## Architecture

```
┌─────────────────────────────┐         ┌──────────────────────────────────┐
│ aegis-greeter (this repo)   │         │ aegis-stateless (sibling)        │
│                             │  push   │                                  │
│  cmd/greeter, internal/     │── ECR ─→│  k8s/overlays/prod/              │
│  Dockerfile                 │         │  └── kustomization.yaml          │
│  .github/workflows/         │  commit │      ← image tag bumped by       │
│                             │── PAT ─→│        this repo's CI            │
│                             │ to repo │           ↓                      │
│                             │         │       ArgoCD sync → EKS          │
└─────────────────────────────┘         └──────────────────────────────────┘
```

CI produces exactly two things: a container image in ECR, and a git
commit in the sibling repo that bumps the image tag. ArgoCD inside the
cluster reconciles from there. The application CI never runs `kubectl`.

## Observability

The service emits all four signal types to a Grafana Cloud stack via a
local Grafana Alloy DaemonSet:

| Signal | How | Backend |
|---|---|---|
| Metrics | OpenTelemetry SDK; `otelhttp` middleware auto-emits HTTP RED metrics; two custom instruments (`greeter_responses_total{personalized}`, `greeter_build_info`) | Grafana Cloud Mimir |
| Traces | OpenTelemetry SDK; OTLP gRPC; one span per request | Grafana Cloud Tempo |
| Logs | `log/slog` JSON to stdout, with `trace_id` / `span_id` / `pod` / `node` injected | Grafana Cloud Loki |
| Profiles | `grafana/pyroscope-go` — CPU, alloc, goroutines | Grafana Cloud Pyroscope |

Every exporter is fail-soft: an empty or unreachable endpoint degrades
that subsystem to a no-op and never blocks the request path.

## Reviewer setup

The project owns its toolchain. Nothing it installs lands in your
`~/go/bin`, `/usr/local`, or shell profile — the Go compiler is pinned
in `go.mod` and dev tools install into `./bin/`.

```sh
make hooks-install   # activate the git hooks (optional but recommended)
make dev-setup       # install golangci-lint, govulncheck, actionlint into ./bin/
make test            # run the suite with the race detector
```

Your host Go version does not matter: `go.mod` declares a `toolchain`
directive and `GOTOOLCHAIN=auto` (Go's default since 1.21) fetches the
pinned compiler on demand.

If you would rather not install Go at all, the container build is fully
self-contained:

```sh
make image           # multi-stage docker build, no host Go needed
```

## Build

```sh
make build           # static binary → ./bin/greeter
make image           # production container image (distroless, ~7 MB)
```

## Quality gates

Checks are layered shift-left — the earlier a defect is caught, the
cheaper it is to fix:

| Stage | Trigger | Checks |
|---|---|---|
| `pre-commit` hook | every `git commit` | `gofmt`, `go vet`, `go build` |
| `pre-push` hook | every `git push` | the above + `go test -race`, `golangci-lint`, `govulncheck`, `actionlint`, `hadolint` |
| GitHub Actions `ci.yml` | every PR and push | the full pre-push suite + container build + Trivy image scan |
| GitHub Actions `codeql.yml` | push, PR | CodeQL static analysis (runs once the repo is public — see CI/CD) |
| GitHub Actions `dependency-review.yml` | every PR | blocks PRs adding HIGH+ vulnerable dependencies |

Local hooks and CI run the *same* `make` targets, so "passes locally"
genuinely predicts "passes CI". Run the full local gate by hand with
`make prepush`.

## CI/CD

| Workflow | Runs on | Does |
|---|---|---|
| `ci.yml` | PR, push | Verification gate; secrets-free, expected always green. |
| `codeql.yml` | push, PR | SAST. Code scanning needs a public repo or GitHub Advanced Security, so the job skips cleanly while the repo is private and activates when it goes public. |
| `dependency-review.yml` | PR | Dependency-diff vulnerability gate. |
| `publish.yml` | push to `main` | Build → push to ECR over OIDC → commit the image tag back to the sibling repo. |

`publish.yml` authenticates to AWS with GitHub OIDC — no static keys —
and writes back to the infra repo with a fine-grained, short-lived PAT.
Every third-party action is pinned to a commit SHA.

## Configuration

All configuration is environment variables; there is no config file.

| Variable | Default | Purpose |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Listen address. |
| `LOG_LEVEL` | `INFO` | `DEBUG` / `INFO` / `WARN` / `ERROR`. |
| `HELLO_TAG` | — | Free-form release tag, logged at startup. |
| `OTEL_SERVICE_NAME` | `aegis-greeter` | OpenTelemetry service name. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP gRPC endpoint; empty disables trace/metric export. |
| `PYROSCOPE_ENDPOINT` | — | Pyroscope endpoint; empty disables profiling. |
| `POD_NAME`, `NODE_NAME` | — | Downward API values; surfaced as `pod` / `node` log fields. |

## Decisions

Architecture decision records live in [`docs/adr/`](docs/adr/). They
cover the toolchain, the logging stack, the container base, the CI
mechanics, and the boundary-value choices.

## License

See [LICENSE](LICENSE).
