# ADR-001: Selective Adoption of DBOS (Go Port)

Status: Adopted. The governing principles of `docs/DBOS.md` are ported
to this stack; its TypeScript/Hono/pg-boss examples are not adopted
verbatim. See `miniapps/adr001-template.md` for the per-workload record.

## Context

Durable orchestration adds programming-model, operational, and versioning
cost. Adopting it by default would tax every miniapp for guarantees most
do not need. The original policy text targets OpenLevel's TypeScript
stack, so its business questions stay authoritative while its examples
and alternative-technology references are rewritten for Go, PostgreSQL,
River, and DBOS.

## Decision

DBOS must earn adoption per workload through Gates 1–4 and the complexity
score, judged against this stack. Async, scheduled, or multi-step
execution alone never justifies DBOS. Every Class C/D miniapp records a
workload-specific decision (`REJECT`/`DEFER`/`PILOT`/`ADOPT`) with human
review of business consequence and net complexity; automated admission
checks record presence, score arithmetic, and approval state only.

## Consequences

- The executor admission check rejects DBOS substrates for tools without
  an approved decision record.
- The workload inventory in `miniapps/inventory.md` keeps one
  authoritative execution owner per pattern.
- §7 obligations (determinism, idempotency, tx boundaries, identity,
  versioning, payload discipline, recovery, operational cost) bind every
  adopted workflow.
