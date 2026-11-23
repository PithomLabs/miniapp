// Package classd is the Phase 1 Class D conformance probe: a neutral,
// test-only miniapp exercising the framework's reusable durable
// execution contracts through the canonical invocation path. It
// carries no business semantics: subjects, stages, waits, signals,
// and mock effects exist only to prove framework invariants.
//
// Vocabulary is deliberately non-sales: subject (entity), sequence,
// generation, stage, eligibility flag, mock effect. Nothing here
// resembles, previews, or stands in for lead-followup; the ADR-001
// hard stop is unaffected.
//
// What this probe proves (each covered by a test in probe_test.go):
// canonical authenticated invocation; server-derived identity;
// stable run handles; multi-stage checkpointed progress with a
// durable per-entity sequence record (active-uniqueness,
// generation lifecycle, attach-on-conflict); durable sleep;
// signal wait/resume; eligibility recheck; approval wait with
// table re-read; mock effects through the shared ledger with
// UNKNOWN quarantine; cancellation with in-flight settlement;
// crash recovery without repeated confirmed effects; tenant
// isolation; audit reconstruction; analyzer-clean workflow code;
// deterministic replay equivalence.
//
// What it does not prove: version transitions (covered at executor
// level), production channels or reconciliation (Phase 2/4),
// volumetric limits (OSI-4), or that any business workload
// deserves DBOS. Passing this probe establishes only the tested
// conformance envelope.
package classd
