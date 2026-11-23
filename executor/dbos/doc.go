// Package dbosexec is the Phase 1 Class D executor: one DBOS
// application per registered miniapp family, each pinned to its
// declared workflow-compatibility version. The executor owns the
// DBOS contexts and the shared-pool durability configuration;
// miniapps own their workflow functions. The application pool is
// shared with the runtime and stays caller-owned: Shutdown stops
// DBOS workers so the owner can close the pool after.
//
// Version policy (ADR-016): the SDK recovers and dequeues only rows
// matching a worker's application version, so overlapping releases
// never steal each other's runs. A worker that finds pending runs
// under another version reports them at startup; they stay pending
// for their owning version to drain. Retiring an old version means
// keeping its worker up until its pending set empties — in Phase 1
// (single process) that is a restart of the same version; across
// releases it is the deployment overlap the G6 drill proves.
package dbosexec
