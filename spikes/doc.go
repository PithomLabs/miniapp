// Package spikes holds the Phase 0 integration-spike harnesses: small,
// reproducible tests that prove load-bearing third-party semantics
// before production adapters are built on them.
//
// Env gates (all unset means skip, never fake a pass). The normal path
// is local Docker PostgreSQL via `task --taskfile Taskfile.dev.yml
// test:integration`, which exports these for you; a Neon branch URL is
// the alternative, not a prerequisite:
//
//	TEST_DATABASE_URL      River + RLS spikes (local Docker, or Neon branch, Q4)
//	TEST_DBOS_DATABASE_URL DBOS atomicity, isolation, and crash spikes (local Docker, or Neon branch, Q4)
//	SPRITE_TOKEN           Sprite API token for the lifecycle round-trip (Q4)
//
// The MCP round-trip runs in-process with no credentials. Crash
// orchestration beyond G0 scope (River worker kill/resume) belongs to
// Phase 3; the DBOS crash probe below covers the G0 pool-handle proof.
package spikes
