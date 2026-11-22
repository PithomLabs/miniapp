# DBOS Go Determinism Rules (Phase 0 spec; analyzer in Phase 1, F17)

Workflow bodies must replay deterministically against the pinned DBOS
Go SDK. The Phase 1 `go/analysis` linter flags these constructs inside
functions registered via `RegisterWorkflow`, and CI strict mode fails
the build on any hit before a Class D production release.

## Forbidden in workflow bodies

- `go` statements and unbounded goroutines (use durable `Go` helpers).
- `select` on channels (use durable `Select` helpers).
- Map iteration order dependence.
- `time.Now`, `time.Since`, `rand.*`, and other nondeterministic stdlib.
- Reads/writes of mutable package globals.
- Direct DB, HTTP, or clock I/O (belongs in steps/transactions).
- `RunAsTransaction`, `Enqueue`, `Send`/`Recv` calls inside step bodies
  (per pinned-SDK step rules).

## Required alongside

- Deterministic replay tests per workflow: same input twice yields the
  same step sequence and result.
- Step-name stability review on every workflow-code change.

Rule wording is verified against the pinned SDK's determinism docs at
implementation time; this spec freezes the rule set, not the prose.
