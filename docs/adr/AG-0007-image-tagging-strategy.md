# AG-0007: Image tag strategy — immutable short-SHA tag only

- **Status**: Accepted (revised 2026-05-16)
- **Date**: 2026-05-15

## Context

Each CI build produces a container image that needs a tag. The tag has
to give a deterministic, traceable mapping from a running container
back to the source commit it was built from.

The ECR repository, provisioned by the sibling `aegis-stateless`
`platform/` environment, is created with `image_tag_mutability =
IMMUTABLE` — a deliberate supply-chain choice so a tag can never be
silently repointed at different bytes.

## Decision

CI tags and pushes exactly one tag to ECR: the **7-character commit
SHA**. The cross-repo commit-back pins the sibling's kustomization to
that SHA tag — it is the tag ArgoCD deploys.

A moving `latest` tag is **not** pushed. It was in the original plan as
an ad-hoc `docker pull` convenience, but `latest` is mutable by
definition and cannot be re-pushed to an `IMMUTABLE` ECR repository —
the second build's `latest` push fails with `tag invalid ... cannot be
overwritten`. Immutability is the more valuable property, so `latest`
is dropped rather than weakening the repository to `MUTABLE`.

The local `make image` target still applies a `:latest` tag for
developer convenience — a local Docker daemon has no immutability
constraint. Only the ECR push is SHA-only.

## Consequences

- Every deployed container maps to exactly one commit; rollback is
  "pin the previous SHA".
- No tag in ECR is ever overwritten — immutability holds end to end.
- No `latest` in the registry, so nothing can accidentally deploy a
  floating tag.
- No semantic version tag on the image. Releases are tracked as git
  tags on the repository, not as image tags — sufficient, since the
  image is an internal artifact, not a published distribution.

## Alternatives considered

- **A moving `latest` tag** — covered in the Decision: incompatible
  with an `IMMUTABLE` ECR repository, and immutability is worth more
  than the pull-convenience `latest` offered.
- **Semantic-version image tags** (`v1.2.3`) — meaningful for a
  published, externally-consumed image; this is an internal deploy
  artifact, and the git repo already carries semver tags.
- **Date / build-number tags** — sortable, but not traceable back to a
  commit without a lookup. The commit SHA is both unique and the
  pointer to the source.
- **Full 40-character SHA** — unambiguous but unwieldy in `kubectl`
  and dashboard output; the 7-character short SHA is the git
  convention and collision-safe at this repo's scale.

## Out of scope / when to revisit

- Semantic-version image tags — revisit only if the image stops being
  an internal artifact and becomes something external consumers pull
  by version.
