# ADR-017: Lifecycle Golden Path (A16)

Status: Adopted.

## Context

A lifecycle described only as stages and commands is not a product
capability until one complete journey is binding and tested.

## Decision

One golden path — INIT → DEVELOP → VALIDATE → TEST → PREVIEW → PUBLISH
→ DEPLOY → OPERATE → UPGRADE/ROLLBACK/RETIRE — across the CLI and the
standard GitHub Actions CI/CD on shared services and policies, is the
binding end-to-end acceptance requirement. It is defined in Phase 0,
implemented incrementally across Phases 1–6, and verified from a clean
checkout in Phase 6.

## Consequences

- CLI/CI drift is a release-blocking defect, not a cosmetic gap.
- Operators get inspect/upgrade/recover/rollback/retire without a
  second, undocumented procedure set.
