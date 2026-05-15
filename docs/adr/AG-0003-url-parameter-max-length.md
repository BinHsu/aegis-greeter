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
