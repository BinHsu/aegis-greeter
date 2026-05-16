# AG-03: Container image and delivery pipeline

- **Status**: Accepted
- **Date**: 2026-05-16

## Context

This record covers how the service travels from a commit to a running
pod: the container artifact, the tag it carries, the CI pipeline that
builds and ships it, and the hand-off to GitOps. One invariant frames
all of it — CI produces exactly two things, a container image and a
git commit, and never talks to the cluster. ArgoCD does all
reconciliation.

## Decisions

### 1. A minimal-surface runtime image

The image is a multi-stage build: a `golang:1.26-alpine` builder
produces a fully static binary (`CGO_ENABLED=0`, `-trimpath`,
stripped), and the runtime stage is `gcr.io/distroless/static-debian12`
— no shell, no package manager, no libc. Both base images are pinned
to multi-arch index digests, not floating tags. The container runs as
`USER nonroot`. The result is roughly 7 MB, with a minimal CVE surface
and no shell for an attacker to reach.

### 2. Immutable, SHA-only image tags

The ECR repository is created `IMMUTABLE` — a tag can never be
repointed at different bytes. CI therefore pushes exactly one tag: the
7-character commit SHA. There is no moving `latest` tag in the
registry; `latest` is mutable by definition and cannot be re-pushed to
an immutable repository, and immutability is worth more than the
pull-time convenience `latest` offered. Every running container maps
to exactly one commit; rollback is "pin the previous SHA".

### 3. GitHub Actions, OIDC, no static keys

CI is GitHub Actions. The publish workflow authenticates to AWS with
GitHub OIDC — there is no static AWS key anywhere in the repo. Every
third-party action is pinned to a commit SHA, not a floating tag.
Quality gates are layered shift-left: git hooks (pre-commit, pre-push)
run the same `make` targets the CI workflow runs, so "passes locally"
predicts "passes CI"; CI then adds the container build and the Trivy
image scan.

### 4. Cross-repo commit-back closes the GitOps loop

After pushing the image, CI updates the image tag in the sibling
infrastructure repo so ArgoCD can reconcile it. The write uses a
fine-grained Personal Access Token scoped to the sibling repo with
`contents: write` only, carrying a 90-day expiry. CI rebases onto the
sibling's `main` before pushing a single linear commit. CI never runs
`kubectl`; that commit is the entire hand-off.

## Consequences

- The deployed artifact is traceable to a commit and immutable end to
  end; nothing can accidentally deploy a floating tag.
- No credential with standing access lives in the repo — OIDC is
  minted per run, the PAT is short-lived and narrowly scoped.
- Cost: the PAT needs rotation every 90 days (a calendar reminder
  covers it); the cross-repo write is a second credential, since the
  default `GITHUB_TOKEN` is scoped to one repo and cannot reach the
  sibling.

## Alternatives considered

- **`alpine` / `scratch` / `debian-slim` runtime** — `alpine` ships a
  shell and a package manager the static binary does not need;
  `scratch` lacks the CA certificates and `/etc/passwd` the service
  does need; `debian-slim` is a full userland for nothing.
- **A moving `latest` tag, or semantic-version image tags** — `latest`
  is incompatible with an immutable repository; semver tags belong on a
  published artifact, and the git repo already carries semver tags.
- **A GitHub App or an SSH deploy key for the cross-repo write** — a
  GitHub App removes rotation but is heavier than one repo pair
  warrants; a deploy key cannot be scoped to `contents` alone.
- **ArgoCD Image Updater** — moves tag-bumping into the cluster, but
  adds a cluster component and inverts the "CI owns the git commit"
  shape this project demonstrates.
- **Static AWS access keys** — a standing credential in CI secrets;
  OIDC removes the standing secret entirely.

## Out of scope / when to revisit

- `gcr.io/distroless/base` (with glibc) — required only if the build
  ever needs `CGO_ENABLED=1`; no such dependency is on the horizon.
- A GitHub App for the cross-repo write — revisit when more than one
  repo needs to write to the infra repo, or when PAT rotation becomes
  a recurring cost rather than a footnote.
- Branch protection enforcing the CI gate — unavailable on a private
  repo on the GitHub Free plan; it activates when the repo is made
  public. See the README's Known limitations.
