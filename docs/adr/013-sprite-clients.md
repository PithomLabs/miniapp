# ADR-013: Sprite SDK vs Framework REST Client

Status: Adopted (pending re-verification at the pinned version in G0).

## Context

The inspected sprites-go checkout covers lifecycle, exec, filesystem,
services, checkpoints, and network policy, but exposes no usage, logs,
connector, or privilege APIs.

## Decision

Use pinned sprites-go (wrapped) where it is authoritative and a
framework-owned typed REST client only for verified API gaps. Both sit
behind framework interfaces with fakes. Never code against imagined
APIs; re-verify coverage at the pinned version.

## Consequences

- SDK upgrades re-run the coverage check before adoption.
- Gap-client endpoints require live-API verification, not doc rumor.
