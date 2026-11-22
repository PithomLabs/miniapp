# DBOS Family, Version, and Drain Plan (Phase 0, F01)

Each Class D miniapp/workflow family gets a stable, distinct DBOS
`AppName` and an explicitly managed application version tied to the
workflow implementation compatibility line — never blindly to a
manifest version or the whole-repo version.

## Deployment strategy (default)

One worker process group per Class D family, all running the same Go
image with a family selector. A release may start a new family version
while the previous artifact stays available to drain or recover
old-version workflows. Old versions retire only after pending/running
workflows drain or migrate under a verified patch strategy.

## G0-Core proof obligations

1. Multi-`AppName` isolation on one Neon branch: same workflow name
   under two AppNames completes independently with correct results
   (`spikes/dbos_test.go` family test).
2. Version tagging: workflows record their application version; a
   version bump without a patch does not silently recover old
   workflows under new code (asserted via step metadata after a
   versioned relaunch).
3. If same-image process-group isolation cannot preserve per-family
   versions, revise the deployment unit to separate Fly apps for the
   affected families (no separate compiled binaries required).

Fly process-group retention and deploy-overlap fencing are proven by
the Phase 6 overlap drill, not claimed here.
