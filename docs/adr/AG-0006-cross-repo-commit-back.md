# AG-0006: Cross-repo commit-back via fine-grained PAT

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

After pushing an image to ECR, the CI must update the image tag in the
sibling infrastructure repo (`aegis-stateless`) so ArgoCD can reconcile
the new version. Writing to another repository from GitHub Actions has
three common mechanisms: a fine-grained Personal Access Token, a deploy
key, or a GitHub App.

## Decision

Use a **fine-grained PAT** scoped to `aegis-stateless` with
`contents: write` only, stored as the `INFRA_REPO_PAT` secret in this
repo. The token carries a **90-day expiry**; a calendar reminder fires
at the 75-day mark and a rotation runbook lives in the sibling repo. A
token without an expiry is never used.

A GitHub App would remove the rotation burden but is operationally
heavier than one repo pair warrants. A deploy key is SSH-only and
awkward to scope to `contents` alone.

## Consequences

- Simplest mechanism that works; the token's blast radius is one repo,
  one permission.
- Cost: manual rotation every 90 days. This is accepted and made
  routine by the reminder and the runbook.
- The default `GITHUB_TOKEN` cannot do this — it is scoped to the
  current repo only — so a separate credential is unavoidable
  regardless of mechanism.

## Alternatives considered

- **GitHub App** — no token expiry to rotate, finer-grained
  installation scoping. Operationally heavier to set up and run than
  one repo pair justifies; the right answer once several repos need
  cross-writes.
- **Deploy key (SSH)** — repo-scoped, but SSH-only and cannot be
  scoped to `contents` alone; it grants more than the commit-back
  needs.
- **ArgoCD Image Updater** — moves image-tag bumping into the cluster
  and removes the cross-repo write entirely. Rejected for this POC: it
  adds a cluster component and inverts the "CI owns the git commit"
  GitOps shape this project demonstrates.

## Out of scope / when to revisit

- Migrate to a GitHub App when more than one repo needs to write to
  the infra repo, or when 90-day PAT rotation becomes a recurring
  operational cost rather than a calendar footnote.
