# Class D Framework Operations (Phase 1)

How the framework behaves under failure, and what operators own.
Every claim below traces to implemented code or a pinned-SDK
default verified in `dbos-transact-golang@v1.9.0` source.

## Retry limits and terminal failure

- Steps do not retry by default (SDK `WithStepMaxRetries`
  unset → first error fails the step). The framework sets no step
  retries in Phase 1: a failing step fails its workflow visibly
  rather than amplifying an unknown provider state.
- Interrupted workflows recover up to 100 times (SDK default
  `_DEFAULT_MAX_RECOVERY_ATTEMPTS`), then dead-letter as
  `MAX_RECOVERY_ATTEMPTS_EXCEEDED`, which the executor maps to
  `RunFailed`. Per-workflow caps (`WithMaxRecoveryAttempts`) are
  available when a family needs them.
- Effect sends never retry blindly: `UNKNOWN` quarantines until
  reconcile evidence or operator settlement (`effects.Settle`).

## Cancellation

- `runtime.Cancel` records the cancel durably first, settles
  pending approvals, then stops the executor run. A second Cancel
  (or any Status read) redelivers a stop that failed while the
  executor was unreachable.
- Cancellation cannot recall an accepted provider send. The
  cancel response names in-flight operations
  (`CancelResult.InFlight`); each settles by evidence to
  `CONFIRMED`, `FAILED`, or `UNKNOWN`.

## Workflow inspection

- `Status` returns fresh executor state, or the persisted record
  with `Stale` set when the executor is unreachable. Terminal
  records are authoritative without a live query.
- `dbos.workflow_status` holds per-run state (status, version,
  attempts, executor); operators query it directly for fleets.
- Every invocation, signal, cancel, decision, and effect attempt
  lands in `app.audit_events` attributed to tenant, agent, and
  authority.

## Recovery ownership

- On launch each family worker recovers its own executor's
  `PENDING` rows at its own application version. Rows under any
  other version are left for their owning worker and reported in
  the startup log with version and count.
- Retiring a version means keeping its worker up until its
  pending set empties. Same-version restarts are proven by
  `TestRestartRecovery`; cross-release drain rides the G6
  deployment-overlap drill.

## History and checkpoint retention

- The SDK keeps workflow history until explicitly deleted
  (`DeleteWorkflows` API exists; no automatic GC was found in
  v1.9.0 source). Phase 1 sets no retention job: volumes are
  test-scale and every row is evidence. A retention policy and
  job are required before any production workload.

## Stuck-workflow and stale-status alerts

- Quarantined effects surface through `ListUnknown`
  (oldest-first); alerting on age/count is an operator integration
  on top of this listing, not yet built.
- Foreign-version pending runs surface through the startup
  warning; alerting on the log pattern is likewise future work.
- `Stale` status responses are the client-visible signal of
  executor unreachability.

## Safe operator intervention

- Quarantine: investigate the provider, then `Settle` with the
  evidence cited in the note. Never re-drive `UNKNOWN` by
  resubmitting the same business operation.
- Stuck runs: `Cancel` (durable intent + redelivery), then
  verify via `Status`. Never delete workflow rows to "unstick"
  a run; deletion destroys the audit trail and orphans signals.
- Version incidents: identify the owning version from the
  startup warning or `workflow_status`, bring its worker up to
  drain, and only then retire it.

## DBOS edition and version verification

- Pinned SDK: `dbos-transact-golang v1.9.0` (see `go.mod` and
  the version register). Suspend-mode sleep/recv (an Enterprise
  feature) is refused without `SuspendMode` configuration —
  proven by `TestSleepSuspendRequiresEnterprise`, which records
  the gating error. Phase 1 therefore runs fully on
  open-source behavior: sleeps hold worker goroutines (see the
  sleeper envelope evidence), and no Enterprise module may be
  imported without explicit owner approval (`scripts/checks.sh`
  gate 3).
