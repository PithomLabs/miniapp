# ADR-016: Binary and Registry Model (A15, refined by F01)

Status: Adopted as the Phase 0 decision basis; confirmed at G0.

## Context

One module plus a shared runtime still leaves open how miniapp code is
packaged, registered, and routed. DBOS tags workflows with the
application version that started them and does not by default recover
workflows across versions, so "independently versioned manifests" alone
cannot isolate multi-day workflows sharing one binary.

## Decision

One Go module, statically registered implementation catalog, shared
gateway by default, same image reused across families. Each Class D
miniapp/workflow family has its own DBOS `AppName` and explicit
workflow-compatibility version, with a separately selectable worker
process group/release that retains the old artifact while old
workflows drain. Separate Fly apps only if per-family release
isolation cannot be proven; separate compiled binaries are not
required. No plugin-style hot-loading.

## Consequences

- Manifest versions evolve independently of the host release train;
  registry entries plus the release envelope are the compatibility
  boundary.
- Family releases are independently drainable; the deployment guard
  refuses workflow-code changes without a version bump or patch.
