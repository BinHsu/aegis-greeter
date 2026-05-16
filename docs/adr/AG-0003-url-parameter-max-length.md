# AG-0003: URL parameter capped at 256 bytes, reject over

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

`GET /?name=<value>` echoes caller-supplied input into the response.
Unbounded input is a latent problem — oversized values waste bandwidth
and make the response size attacker-influenced. A boundary is needed,
and the behavior at the boundary must be defined.

## Decision

The `name` parameter is capped at **256 bytes**. A request whose `name`
exceeds the cap is **rejected with 400 Bad Request** — not silently
truncated. 256 bytes is generous for any real name and small enough to
keep the response bounded.

The boundary is tested with explicit Boundary Value Analysis: empty,
exactly 256, and 257 (`MaxNameLen` and `MaxNameLen+1`).

## Consequences

- Response size is bounded and not attacker-controlled.
- Rejection over truncation means the caller learns their input was
  invalid rather than receiving a silently altered greeting.
- 256 is a chosen constant, not derived from a spec. It is a single
  named constant (`handlers.MaxNameLen`), trivial to revise.

## Alternatives considered

- **Silently truncate at 256 bytes** — the caller would receive a
  greeting addressed to a name they did not send, with no signal
  anything was wrong. A wrong-but-200 response is worse than an honest
  400.
- **No application-level limit** — rely on `net/http`'s header / URL
  size limits. That bounds the request but not the `name` field
  specifically, and the effective limit becomes a standard-library
  implementation detail rather than a stated contract.
- **413 Payload Too Large instead of 400** — 413 is about the request
  body; an over-long query parameter is a malformed *request*, so 400
  is the accurate status.

## Out of scope / when to revisit

- The 256-byte cap is a self-chosen default, not a spec value.
  Revisit if a downstream system defines an authoritative maximum for
  the name field — at which point `handlers.MaxNameLen` aligns to it.
