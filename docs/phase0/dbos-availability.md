# DBOS Availability and Recovery Ownership (Phase 0, A11)

The single dedicated executor is the initial topology, not an HA
design. This note assigns the procedures that keep it honest.

## Owned procedures

- Fly restart and health-check replacement behavior for the `dbos`
  process group, verified in Phase 1 (worker health/restart baseline)
  and Phase 6 (restart/drain with active workflows).
- Graceful drain: stop claiming new work, finish in-flight steps,
  then terminate. Documented runbook, drilled in Phase 6.
- Executor-down alerting and stuck-workflow detection with named
  recovery ownership (ADR-001 §7.7 policy per workflow).
- Recovery ownership: on restart the executor recovers its PENDING
  workflows from the last completed step; Conductor, if ever adopted,
  owns multi-executor coordination instead.

## Measured scale thresholds (define exact numbers in Phase 6)

Add a standby, multiple executors, or Conductor when any threshold
trips: sustained executor CPU/memory saturation, recovery time after
restart exceeding the workflow SLO, stuck-workflow rate above zero
for longer than the alert window, or multi-region requirements.

## Explicit non-claims

- "Always on" never means "highly available" in any plan, doc, or
  status output.
- Wake behavior for durable sleeps is proven by the Phase 6 wake
  test, not assumed from this topology.
