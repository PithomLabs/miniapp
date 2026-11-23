package classd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	dbosexec "github.com/PithomLabs/miniapp/executor/dbos"
	"github.com/PithomLabs/miniapp/registry"
)

// SignalProceed is the external-event topic the sequence waits on.
const SignalProceed = "probe.proceed"

// StaleGrace bounds clock skew and scheduling gaps around wake_at.
// A sequence is supersedeable only after its declared wake time
// plus this grace has passed with no heartbeat.
const StaleGrace = 5 * time.Minute

// probeDS is the family datasource, bound once at executor setup.
var probeDS *dbos.DataSource

// Effects is the shared ledger client, wired by whoever launches
// the probe (tests; production wiring would do the same). Steps
// fail closed when it is unset.
var Effects *effects.Service

// seqInput is the translated workflow input. All waits and modes
// come from the invocation, so tests drive timing deterministically.
type seqInput struct {
	TenantID          string
	InvocationID      string
	SubjectID         string
	EffectToken       string
	ApprovalID        string
	SleepMs           int
	WaitMs            int
	ApprovalTimeoutMs int
	PoisonStage       int
}

func translate(req registry.WorkflowRequest, approvalRequired bool) (seqInput, error) {
	var body struct {
		SubjectID         string `json:"subject_id"`
		SleepMs           int    `json:"sleep_ms"`
		WaitMs            int    `json:"wait_ms"`
		ApprovalTimeoutMs int    `json:"approval_timeout_ms"`
		PoisonStage       int    `json:"poison_stage"`
	}
	if err := json.Unmarshal(req.Input, &body); err != nil {
		return seqInput{}, err
	}
	if body.SubjectID == "" {
		return seqInput{}, fmt.Errorf("classd: subject_id is required")
	}
	// Workflow-side fail-closed (§5.6): a gated registration never
	// starts without its approval linkage, even if a future caller
	// bypasses both the service choke and the executor gate.
	if approvalRequired && req.ApprovalID == "" {
		return seqInput{}, fmt.Errorf("classd: gated run requires approval linkage")
	}
	return seqInput{
		TenantID: req.TenantID, InvocationID: req.InvocationID,
		SubjectID: body.SubjectID, EffectToken: req.EffectToken,
		ApprovalID: req.ApprovalID, SleepMs: body.SleepMs,
		WaitMs: body.WaitMs, ApprovalTimeoutMs: body.ApprovalTimeoutMs,
		PoisonStage: body.PoisonStage,
	}, nil
}

func registerRun(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, sequenceWorkflowGated)
	return nil
}

func registerOpen(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, sequenceWorkflowOpen)
	return nil
}

func startRun(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translate(req, true)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, sequenceWorkflowGated, in, dbos.WithWorkflowID(runID))
	return err
}

func startOpen(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translate(req, false)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, sequenceWorkflowOpen, in, dbos.WithWorkflowID(runID))
	return err
}

// sequenceWorkflowGated runs the shared sequence logic behind an
// approval wait. sequenceWorkflowOpen runs it without one; both are
// separate registrations over one implementation.
func sequenceWorkflowGated(ctx dbos.Context, in seqInput) (string, error) {
	return sequenceLogic(ctx, in)
}

func sequenceWorkflowOpen(ctx dbos.Context, in seqInput) (string, error) {
	return sequenceLogic(ctx, in)
}

type claimOut struct {
	Generation int
	Attached   bool
	Retry      bool
}

// sequenceLogic is the conformance sequence: claim a generation
// (attach when one is live), progress checkpointed stages with an
// approval wait, a durable sleep, an eligibility gate, one ledgered
// mock effect, and a signal wait. Nothing consequential happens
// before approval and eligibility; waits precede effects so
// cancellation and eligibility tests land in clean windows. Every
// branch keys off recorded step outputs, so replay follows the same
// path.
func sequenceLogic(ctx dbos.Context, in seqInput) (string, error) {
	runID, err := dbos.GetWorkflowID(ctx)
	if err != nil {
		return "", err
	}
	var generation int
	for attempt := 0; attempt < 3; attempt++ {
		out, err := claimStep(ctx, in, runID)
		if err != nil {
			return "", err
		}
		if out.Retry {
			continue
		}
		if out.Attached {
			if err := observeStep(ctx, in, "attached"); err != nil {
				return "", err
			}
			return finishJSON(in, out.Generation, "attached", 0)
		}
		generation = out.Generation
		break
	}
	if generation == 0 {
		return "", fmt.Errorf("classd: sequence claim did not settle")
	}
	if err := bumpStep(ctx, in, generation, 1); err != nil {
		return "", err
	}
	if in.ApprovalID != "" {
		outcome, done, err := approvalStep(ctx, in)
		if err != nil {
			return "", err
		}
		if done {
			return finishSequence(ctx, in, generation, outcome, 1)
		}
	}
	if err := heartbeatStep(ctx, in, generation, in.SleepMs); err != nil {
		return "", err
	}
	if in.SleepMs > 0 {
		if _, err := dbos.Sleep(ctx, time.Duration(in.SleepMs)*time.Millisecond); err != nil {
			return "", err
		}
	}
	if err := bumpStep(ctx, in, generation, 2); err != nil {
		return "", err
	}
	eligible, err := eligibilityStep(ctx, in, generation)
	if err != nil {
		return "", err
	}
	if !eligible {
		return finishSequence(ctx, in, generation, "ineligible", 2)
	}
	state, err := effectStep(ctx, in, generation)
	if err != nil {
		return "", err
	}
	if state == contract.EffectUnknown {
		return finishSequence(ctx, in, generation, "quarantined", 2)
	}
	if err := heartbeatStep(ctx, in, generation, in.WaitMs); err != nil {
		return "", err
	}
	signaled := false
	if _, err := dbos.Recv[string](ctx, SignalProceed, time.Duration(in.WaitMs)*time.Millisecond); err == nil {
		signaled = true
	}
	outcome := "completed-timeout"
	if signaled {
		outcome = "completed-signaled"
	}
	if err := bumpStep(ctx, in, generation, 3); err != nil {
		return "", err
	}
	return finishSequence(ctx, in, generation, outcome, 3)
}

// observeStep records a run-local observation (such as an attach)
// for tests to assert on. Attached runs own no sequence row, so
// observations live in their own table.
func observeStep(ctx dbos.Context, in seqInput, detail string) error {
	_, err := dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_observations(invocation_id, tenant_id, detail)
			VALUES ($1, $2, $3)`, in.InvocationID, in.TenantID, detail)
		return "", err
	})
	return err
}

type outcomeJSON struct {
	Sequence   string `json:"sequence"`
	Generation int    `json:"generation"`
	Outcome    string `json:"outcome"`
	Stages     int    `json:"stages"`
}

func finishJSON(in seqInput, generation int, outcome string, stages int) (string, error) {
	raw, err := json.Marshal(outcomeJSON{
		Sequence:   in.TenantID + ":" + in.SubjectID,
		Generation: generation, Outcome: outcome, Stages: stages,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func finishSequence(ctx dbos.Context, in seqInput, generation int, outcome string, stages int) (string, error) {
	if _, err := dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `UPDATE probe_sequences SET status = 'done', outcome = $1,
			updated_at = now() WHERE tenant_id = $2 AND subject_id = $3 AND generation = $4`,
			outcome, in.TenantID, in.SubjectID, generation)
		return outcome, err
	}); err != nil {
		return "", err
	}
	return finishJSON(in, generation, outcome, stages)
}

// claimStep supersedes stale active rows, then inserts this run's
// generation. A unique conflict means a live sequence won the race
// and this run attaches to it. Retry covers the abort race where a
// conflict resolves to no visible row.
func claimStep(ctx dbos.Context, in seqInput, runID string) (claimOut, error) {
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (claimOut, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return claimOut{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE probe_sequences SET status = 'superseded', updated_at = now()
			WHERE tenant_id = $1 AND subject_id = $2 AND status = 'active' AND wake_at < now()`,
			in.TenantID, in.SubjectID); err != nil {
			return claimOut{}, err
		}
		var generation int
		err := tx.QueryRow(ctx, `INSERT INTO probe_sequences
			(tenant_id, subject_id, generation, status, stage, run_id, wake_at)
			VALUES ($1, $2, nextval('probe_generation_seq'), 'active', 0, $3, now())
			ON CONFLICT (tenant_id, subject_id) WHERE status = 'active' DO NOTHING
			RETURNING generation`, in.TenantID, in.SubjectID, runID).Scan(&generation)
		if err == nil {
			return claimOut{Generation: generation}, nil
		}
		var active int
		if err := tx.QueryRow(ctx, `SELECT generation FROM probe_sequences
			WHERE tenant_id = $1 AND subject_id = $2 AND status = 'active'`,
			in.TenantID, in.SubjectID).Scan(&active); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return claimOut{Retry: true}, nil
			}
			return claimOut{}, err
		}
		return claimOut{Generation: active, Attached: true}, nil
	})
}

func bumpStep(ctx dbos.Context, in seqInput, generation, stage int) error {
	_, err := dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `UPDATE probe_sequences SET stage = $1, updated_at = now(),
			wake_at = now() + make_interval(secs => $2)
			WHERE tenant_id = $3 AND subject_id = $4 AND generation = $5`,
			stage, StaleGrace.Seconds(), in.TenantID, in.SubjectID, generation)
		return "", err
	})
	return err
}

func heartbeatStep(ctx dbos.Context, in seqInput, generation, waitMs int) error {
	_, err := dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `UPDATE probe_sequences SET updated_at = now(),
			wake_at = now() + make_interval(secs => $1) + make_interval(secs => $2)
			WHERE tenant_id = $3 AND subject_id = $4 AND generation = $5`,
			float64(waitMs)/1000.0, StaleGrace.Seconds(), in.TenantID, in.SubjectID, generation)
		return "", err
	})
	return err
}

func eligibilityStep(ctx dbos.Context, in seqInput, generation int) (bool, error) {
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (bool, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return false, err
		}
		var eligible bool
		err := tx.QueryRow(ctx, `SELECT eligible FROM probe_sequences
			WHERE tenant_id = $1 AND subject_id = $2 AND generation = $3`,
			in.TenantID, in.SubjectID, generation).Scan(&eligible)
		return eligible, err
	})
}

// approvalStep waits for the decision signal, then re-reads the
// authoritative table. The table — never the signal payload —
// decides. done reports early exit with the recorded outcome.
func approvalStep(ctx dbos.Context, in seqInput) (string, bool, error) {
	timeout := time.Duration(in.ApprovalTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	// Wait-entry marker (Rule 3 workflow-start evidence): recorded
	// immediately before the durable wait. Observation-only; the
	// wait semantics below are unchanged.
	if _, err := dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (bool, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return false, err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_wait_marks (invocation_id, tenant_id, approval_id)
			VALUES ($1, $2, $3)`, in.InvocationID, in.TenantID, in.ApprovalID)
		return true, err
	}); err != nil {
		return "", false, err
	}
	// The signal is only a wake-up hint; the table below decides.
	// Timeouts and spurious wakes all converge on the re-read.
	_, _ = dbos.Recv[string](ctx, contract.SignalApprovalDecided, timeout)
	read, err := dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (approvalRead, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return approvalRead{}, err
		}
		st, dec, err := dbosexec.ReadApprovalDecision(ctx, tx, in.TenantID, in.ApprovalID)
		return approvalRead{status: st, decision: dec}, err
	})
	if err != nil {
		return "", false, err
	}
	if read.status == "decided" && read.decision == contract.DecisionApprove {
		return "", false, nil
	}
	outcome := "timeout"
	switch {
	case read.status == "decided" && read.decision == contract.DecisionReject:
		outcome = "rejected"
	case read.status == "expired":
		outcome = "expired"
	case read.status == "cancelled":
		outcome = "cancelled"
	}
	return outcome, true, nil
}

type approvalRead struct {
	status   string
	decision contract.ApprovalDecision
}

// effectStep performs the sequence's single ledgered mock effect.
// Poison input selects the lost-acknowledgement mode, which must
// quarantine without a second send.
func effectStep(ctx dbos.Context, in seqInput, generation int) (contract.EffectState, error) {
	mode := "accept"
	if in.PoisonStage == 2 {
		mode = "lost-ack"
	}
	opID := contract.ScopeID(in.TenantID, fmt.Sprintf("%s:%d:effect1", in.SubjectID, generation))
	providerOp := fmt.Sprintf("%s:%d", in.SubjectID, generation)
	rec, err := dbos.RunAsStep(ctx, func(ctx context.Context) (contract.EffectRecord, error) {
		if Effects == nil {
			return contract.EffectRecord{}, fmt.Errorf("classd: effects service is not wired")
		}
		return Effects.Perform(ctx, in.TenantID, effects.Intent{
			BusinessOpID: opID, AttemptID: "a1", InvocationID: in.InvocationID,
			EffectToken: in.EffectToken,
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: json.RawMessage(`{"mock":{"mode":"` + mode + `","opId":"` + providerOp + `"}}`)},
		})
	})
	if err != nil {
		return "", err
	}
	return rec.State, nil
}
