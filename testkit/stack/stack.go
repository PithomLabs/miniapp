// Package stack assembles a complete canonical service stack for
// integration tests: registry, real DBOS executor, effects ledger
// with strict policy, approvals, audit, and the runtime service.
// Suites own their database, migrations, tables, and fixtures; this
// package owns only the wiring (and the DBOS-before-pool shutdown
// order) so every suite exercises the same production-shaped
// assembly without drifting.
package stack

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor/classa"
	dbosexec "github.com/PithomLabs/miniapp/executor/dbos"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
)

// Assembled is one wired stack. DB is the caller's database (pool
// close stays caller-ordered); Shutdown stops DBOS workers and must
// run before that close — Assemble registers it for you.
type Assembled struct {
	Svc      *runtime.Service
	Approver *approvals.Service
	Auditor  *audit.Service
	Effects  *effects.Service
	Exec     *dbosexec.Executor
	DB       *testkit.DB
	Policy   auth.Policy
	Registry *registry.Registry
}

// Config supplies the suite-specific pieces.
type Config struct {
	// DB is a migrated database; truncation and fixture tables
	// stay the caller's job.
	DB *testkit.DB
	// Register admits the suite's miniapps.
	Register func(t *testing.T, reg *registry.Registry)
	// Policy carries keys and grants. Maps are shared, so tests
	// that mutate them observe live revalidation.
	Policy auth.Policy
	// Channels dispatches effect capabilities (usually mocks).
	Channels map[string]effects.Channel
	// Logger receives DBOS output; nil discards it.
	Logger *slog.Logger
}

// Assemble wires and launches the stack.
func Assemble(t *testing.T, cfg Config) *Assembled {
	t.Helper()
	reg := registry.New()
	cfg.Register(t, reg)
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	ex, err := dbosexec.New(dbosexec.Config{Pool: cfg.DB.Pool, Registry: reg, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	eff := effects.New(effects.Config{Pool: cfg.DB.Pool,
		Channels:  cfg.Channels,
		Canceller: runtime.EffectCanceller(cfg.DB.Pool),
		Policy:    effects.StrictPolicy{Pool: cfg.DB.Pool, Registry: reg, Grants: cfg.Policy},
	})
	approver := approvals.New(cfg.DB.Pool)
	aud := audit.New(cfg.DB.Pool, nil)
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: cfg.Policy, Pool: cfg.DB.Pool,
		Sync: classa.New(cfg.DB.Pool, reg), Async: ex,
		Approver: approver, Auditor: aud, Effects: eff,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Shutdown before pool close: the pool close was registered by
	// NewDB (earlier, so it runs later); this runs first.
	t.Cleanup(func() {
		if err := ex.Shutdown(30 * time.Second); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	return &Assembled{Svc: svc, Approver: approver, Auditor: aud,
		Effects: eff, Exec: ex, DB: cfg.DB, Policy: cfg.Policy, Registry: reg}
}
