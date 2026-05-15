# AG-0007: Image tag strategy — short SHA + latest

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

Each CI build produces a container image that needs a tag. The tag has
to give a deterministic, traceable mapping from a running container
back to the source commit it was built from.

## Decision

Every build is tagged twice: the **7-character commit SHA** and
**`latest`**. The cross-repo commit-back pins the sibling's
kustomization to the **SHA tag** — that is the tag ArgoCD deploys.
`latest` exists only as a convenience pointer for ad-hoc `docker pull`.

## Consequences

- Every deployed container maps to exactly one commit; rollback is
  "pin the previous SHA".
- `latest` is never deployed — it is mutable and unsafe for that, and
  is kept only for human convenience.
- No semantic version tag on the image. Releases are tracked as git
  tags on the repository, not as image tags. This is sufficient: the
  image is an internal artifact, not a published distribution.
