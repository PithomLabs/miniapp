package dbosexec

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

// Config wires one executor. Pool is the shared application pool,
// which doubles as every family's DBOS system pool so application
// writes and step checkpoints commit in a single transaction (proven
// by the shared-handle spike). The pool stays caller-owned: Shutdown
// stops DBOS workers, and the owner closes the pool after Shutdown
// returns.
type Config struct {
	Pool     *pgxpool.Pool
	Registry *registry.Registry
	Logger   *slog.Logger
}

// family is one DBOS application: one miniapp's workflows under one
// AppName and one workflow-compatibility version.
type family struct {
	ctx     dbos.Context
	version string
}

// Executor serves Class D tools durably, one DBOS application per
// registered miniapp family.
//
// TestFault injects failures at named executor boundaries for
// tests (see testkit/fault). It is set only by tests, after
// construction; production constructors leave it nil, and a nil
// injector never fires, so production behavior is unchanged.
type Executor struct {
	mu        sync.Mutex
	reg       *registry.Registry
	logger    *slog.Logger
	families  map[string]*family // appName -> family
	runs      map[string]string  // runID -> appName (submit-time routing)
	TestFault *fault.Injector
}

// AppNameFor derives a family's stable DBOS application name from
// its miniapp ID. Dots are not valid in application names, so they
// map to dashes.
func AppNameFor(miniappID string) string {
	return "miniapp-" + strings.ReplaceAll(miniappID, ".", "-")
}

// New builds, wires, and launches the executor: one DBOS application
// per registered miniapp family, each pinned to its declared
// workflow-compatibility version. Pending runs under any other
// version are reported (never silently adopted or abandoned); the
// SDK's version affinity keeps each worker on its own rows.
func New(cfg Config) (*Executor, error) {
	if cfg.Pool == nil {
		return nil, fmt.Errorf("dbosexec: shared pool is required")
	}
	if cfg.Registry == nil {
		return nil, fmt.Errorf("dbosexec: registry is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	e := &Executor{
		reg: cfg.Registry, logger: logger,
		families: map[string]*family{}, runs: map[string]string{},
	}
	var setupErr error
	cfg.Registry.Range(func(manifest *contract.MiniappManifest, impl *registry.Implementation) bool {
		if len(impl.Workflows) == 0 {
			return true
		}
		if err := e.addFamily(cfg.Pool, logger, manifest, impl); err != nil {
			setupErr = err
			return false
		}
		return true
	})
	if setupErr != nil {
		e.shutdownBestEffort()
		return nil, setupErr
	}
	for name, fam := range e.families {
		if err := fam.ctx.Launch(); err != nil {
			e.shutdownBestEffort()
			return nil, fmt.Errorf("dbosexec: launch %s: %w", name, err)
		}
	}
	e.reportForeignPending()
	return e, nil
}

func (e *Executor) addFamily(pool *pgxpool.Pool, logger *slog.Logger, manifest *contract.MiniappManifest, impl *registry.Implementation) error {
	appName := AppNameFor(string(manifest.ID))
	if _, dup := e.families[appName]; dup {
		return fmt.Errorf("dbosexec: duplicate family %s", appName)
	}
	var version string
	for _, def := range impl.Workflows {
		version = def.Version
	}
	if version == "" {
		return fmt.Errorf("dbosexec: family %s has no workflow-compatibility version", appName)
	}
	// Application versions share one namespace per system database
	// across all AppNames, so the family name qualifies the
	// compatibility line. Affinity still compares equality; the
	// prefix only prevents cross-family collisions.
	version = appName + "/" + version
	dctx, err := dbos.NewContext(context.Background(), dbos.Config{
		AppName:            appName,
		ApplicationVersion: version,
		SystemDBPool:       pool,
		Logger:             logger,
	})
	if err != nil {
		return fmt.Errorf("dbosexec: %s: %w", appName, err)
	}
	ds, err := dbos.NewDataSource(dctx, pool, dbos.WithDataSourceName("app"))
	if err != nil {
		return fmt.Errorf("dbosexec: %s datasource: %w", appName, err)
	}
	env := registry.WorkflowEnv{Context: dctx, Data: ds, Pool: pool}
	for name, def := range impl.Workflows {
		if err := def.Register(env); err != nil {
			return fmt.Errorf("dbosexec: register %s %s: %w", manifest.ID, name, err)
		}
	}
	e.families[appName] = &family{ctx: dctx, version: version}
	return nil
}

// reportForeignPending logs pending runs under versions this worker
// does not serve. They stay pending for their owning version to
// drain; this warning is what makes that visible instead of silent.
func (e *Executor) reportForeignPending() {
	for name, fam := range e.families {
		statuses, err := dbos.ListWorkflows(fam.ctx, dbos.WithFilterStatus(dbos.WorkflowStatusPending))
		if err != nil {
			e.logger.Warn("cannot list pending workflows", "family", name, "error", err)
			continue
		}
		foreign := map[string]int{}
		for _, st := range statuses {
			if st.ApplicationName != "" && st.ApplicationName != name {
				continue
			}
			if st.ApplicationVersion != fam.version {
				foreign[st.ApplicationVersion]++
			}
		}
		for v, n := range foreign {
			e.logger.Warn("pending workflows under another version; draining worker must stay up",
				"family", name, "serving", fam.version, "pending_version", v, "count", n)
		}
	}
}

func (e *Executor) shutdownBestEffort() {
	for _, fam := range e.families {
		_ = dbos.Shutdown(fam.ctx, 10*time.Second)
	}
}

// Shutdown stops all family workers gracefully. Call it before
// closing the shared pool; reversing the order deadlocks.
func (e *Executor) Shutdown(timeout time.Duration) error {
	if err := e.TestFault.Fire("executor-shutdown"); err != nil {
		return err
	}
	var first error
	for name, fam := range e.families {
		if err := dbos.Shutdown(fam.ctx, timeout); err != nil && first == nil {
			first = fmt.Errorf("dbosexec: shutdown %s: %w", name, err)
		}
	}
	return first
}

// Submit starts one durable run in its miniapp's family. The run ID
// is the DBOS workflow ID; the miniapp's Start function translates
// the request into its typed workflow input.
func (e *Executor) Submit(_ context.Context, req executor.Request) (contract.RunHandle, error) {
	manifest, impl, err := e.reg.Resolve(req.MiniappID, req.Version)
	if err != nil {
		return contract.RunHandle{}, err
	}
	def, ok := impl.Workflows[req.Tool]
	if !ok {
		return contract.RunHandle{}, fmt.Errorf("dbosexec: tool %q has no workflow binding", req.Tool)
	}
	if req.RunID == "" {
		return contract.RunHandle{}, fmt.Errorf("dbosexec: runtime-minted run ID is required")
	}
	// Defense in depth behind the service choke point (§5.6): a
	// gated tool must arrive with its approval linkage, so even a
	// future bypassing caller cannot run gated work unapproved.
	// The service choke is authoritative; this gate never admits
	// what the choke refuses.
	for _, tm := range manifest.Tools {
		if string(tm.Name) == req.Tool && tm.Approval.Required &&
			req.TrustMode != contract.TrustAutonomous && req.ApprovalID == "" {
			return contract.RunHandle{}, fmt.Errorf("dbosexec: tool %q requires approval linkage", req.Tool)
		}
	}
	fam, ok := e.familyFor(req.MiniappID)
	if !ok {
		return contract.RunHandle{}, fmt.Errorf("dbosexec: no family serves %q", req.MiniappID)
	}
	runID := req.RunID
	wreq := registry.WorkflowRequest{
		InvocationID: req.InvocationID, TenantID: req.TenantID,
		Principal: req.Principal, Input: req.Input, IdempotencyKey: req.IdempotencyKey,
		EntityKey: req.EntityKey, EffectToken: req.EffectToken, ApprovalID: req.ApprovalID,
	}
	if err := def.Start(fam.ctx, wreq, runID); err != nil {
		return contract.RunHandle{}, fmt.Errorf("dbosexec: submit: %w", err)
	}
	e.mu.Lock()
	e.runs[runID] = AppNameFor(req.MiniappID)
	e.mu.Unlock()
	return contract.RunHandle{
		RunID: runID, MiniappID: req.MiniappID, Version: contract.Version(req.Version),
		Status:   contract.RunRunning,
		PollPath: "runs.status", SignalPath: "runs.signal", CancelPath: "runs.cancel",
	}, nil
}

func (e *Executor) familyFor(miniappID string) (*family, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fam, ok := e.families[AppNameFor(miniappID)]
	return fam, ok
}

// contextsForRun returns the owning family context first when the
// run was submitted here, then every family for fan-out (restart
// recovery must find runs submitted before this process started).
func (e *Executor) contextsForRun(runID string) []dbos.Context {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []dbos.Context
	if name, ok := e.runs[runID]; ok {
		if fam, ok := e.families[name]; ok {
			out = append(out, fam.ctx)
		}
	}
	for name, fam := range e.families {
		if name == e.runs[runID] {
			continue
		}
		out = append(out, fam.ctx)
	}
	return out
}

// Status maps the durable workflow state onto run status. PENDING,
// ENQUEUED, and DELAYED are running; SUSPENDED (blocked on an event)
// is waiting_event; SUCCESS, ERROR (including exhausted recovery),
// and CANCELLED map directly.
func (e *Executor) Status(_ context.Context, runID string) (contract.RunStatus, error) {
	var lastErr error
	tried, unknown := false, true
	for _, ctx := range e.contextsForRun(runID) {
		tried = true
		h, err := dbos.RetrieveWorkflow[any](ctx, runID)
		if err != nil {
			if !errors.Is(err, dbos.ErrNonExistentWorkflow) {
				unknown = false
			}
			lastErr = err
			continue
		}
		st, err := h.GetStatus()
		if err != nil {
			unknown = false
			lastErr = err
			continue
		}
		switch st.Status {
		case dbos.WorkflowStatusSuccess:
			return contract.RunSucceeded, nil
		case dbos.WorkflowStatusError, dbos.WorkflowStatusMaxRecoveryAttemptsExceeded:
			return contract.RunFailed, nil
		case dbos.WorkflowStatusCancelled:
			return contract.RunCancelled, nil
		case dbos.WorkflowStatusSuspended:
			return contract.RunWaitingEvent, nil
		case dbos.WorkflowStatusPending, dbos.WorkflowStatusEnqueued, dbos.WorkflowStatusDelayed:
			return contract.RunRunning, nil
		default:
			return "", fmt.Errorf("dbosexec: unknown workflow status %q", st.Status)
		}
	}
	if tried && unknown {
		return "", fmt.Errorf("dbosexec: unknown run %s: %w", runID, executor.ErrUnknownRun)
	}
	if lastErr != nil {
		return "", fmt.Errorf("dbosexec: %w", lastErr)
	}
	return "", fmt.Errorf("dbosexec: unknown run %s: %w", runID, executor.ErrUnknownRun)
}

// Signal delivers an external input to a live run. The signal type
// becomes the DBOS topic and the payload travels as UTF-8 JSON
// bytes; workflows receive it with Recv[string] on that topic.
func (e *Executor) Signal(_ context.Context, runID string, sig contract.Signal) error {
	if sig.Type == "" {
		return fmt.Errorf("dbosexec: signal type is required")
	}
	var lastErr error
	for _, ctx := range e.contextsForRun(runID) {
		if err := dbos.Send(ctx, runID, string(sig.Payload), sig.Type); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("dbosexec: unknown run %s: %w", runID, executor.ErrUnknownRun)
}

// Cancel stops a live run. The reason is audit-owned by the runtime;
// DBOS records the cancellation itself. Cancelling a nonexistent
// run reports executor.ErrUnknownRun so the runtime can distinguish
// "never submitted" (converge) from "executor unavailable" (retry).
func (e *Executor) Cancel(_ context.Context, runID string, _ string) error {
	var lastErr error
	tried, unknown := false, true
	for _, ctx := range e.contextsForRun(runID) {
		tried = true
		if err := dbos.CancelWorkflow(ctx, runID); err != nil {
			if !errors.Is(err, dbos.ErrNonExistentWorkflow) {
				unknown = false
			}
			lastErr = err
			continue
		}
		return nil
	}
	if tried && unknown {
		return fmt.Errorf("dbosexec: unknown run %s: %w", runID, executor.ErrUnknownRun)
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("dbosexec: unknown run %s: %w", runID, executor.ErrUnknownRun)
}
