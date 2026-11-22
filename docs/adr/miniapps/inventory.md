# Workload Inventory (ADR-001 §8)

One authoritative execution owner per workload pattern. Seeded in
Phase 0; entries gain decision-record links as miniapps are admitted.

| Workload pattern | Execution owner | Decision record |
|---|---|---|
| Sync read / validated command | Go handler + Postgres tx | n/a (Class A, no engine) |
| Approval-gated single-step command | Approval table + one-shot River job | n/a until first use |
| One-shot job, webhook processing, scheduled report | River | n/a until first use |
| Bounded batch with per-record effects | PILOT: River vs DBOS queue (per workload) | required before Class C enablement |
| Multi-day, waits, approvals inside process, saga | DBOS | required before Class D enablement |
| Per-entity serialization | Row lock (A/B/C); partitioned DBOS queue (D) | with owning miniapp record |
| Effect reconciliation sweeper | River periodic job | n/a until first use |

River and DBOS coexist permanently by design (not as a migration
state); this table is reviewed whenever either's seat is questioned.
