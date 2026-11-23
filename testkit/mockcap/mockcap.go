// Package mockcap is a deterministic mock provider for effect tests.
// Behavior is driven by the operation payload, so it survives process
// restarts without shared mutable state: {"mock":{"mode":...}}.
// External acceptance is recorded in a provider-log table written in
// autocommit, simulating a provider whose state is independent of the
// application's database transactions.
package mockcap

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/effects"
)

// Modes select provider behavior.
const (
	// ModeAccept records acceptance and returns a receipt.
	ModeAccept = "accept"
	// ModeReject returns a terminal provider error (FAILED).
	ModeReject = "reject"
	// ModeTimeout returns UnknownError without recording acceptance.
	ModeTimeout = "timeout"
	// ModeLostAck records acceptance, then returns UnknownError:
	// the poisoned case (provider has it, we cannot prove it).
	ModeLostAck = "lost-ack"
	// ModeSlowAccept sleeps briefly, then behaves as accept.
	// Used to widen race windows deterministically.
	ModeSlowAccept = "slow-accept"
	// ModeDelayedAck records acceptance, sleeps to widen the crash
	// window, then returns UnknownError. Kill tests strand a
	// provider-accepted send in ATTEMPTING with this mode.
	ModeDelayedAck = "delayed-ack"
)

// Channel implements effects.Channel against the provider log.
type Channel struct {
	pool *pgxpool.Pool
}

// New returns a Channel logging to pool.
func New(pool *pgxpool.Pool) *Channel {
	return &Channel{pool: pool}
}

// EnsureLog creates the provider-log table. Provider state is
// deliberately outside every framework migration: it belongs to the
// simulated external system, not the application. One row is one
// actual send attempt (serial PK, no dedup): the log measures
// provider calls, never reservations. A legacy table without the
// serial PK is dropped and recreated; the log holds test-only data.
func EnsureLog(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var hasID bool
	err := pool.QueryRow(ctx, `SELECT count(*) > 0 FROM information_schema.columns
		WHERE table_name = 'mock_provider_log' AND column_name = 'id'`).Scan(&hasID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID {
		if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS mock_provider_log`); err != nil {
			t.Fatal(err)
		}
	}
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS mock_provider_log (
		id BIGSERIAL PRIMARY KEY, op_id TEXT NOT NULL, tenant_id TEXT NOT NULL,
		mode TEXT NOT NULL, receipt TEXT NOT NULL DEFAULT '',
		sent_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS mock_provider_log_op
		ON mock_provider_log (op_id)`); err != nil {
		t.Fatal(err)
	}
}

// TruncateLog empties the provider log for test isolation.
func TruncateLog(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := pool.Exec(ctx, `TRUNCATE mock_provider_log`); err != nil {
		t.Fatal(err)
	}
}

// SentCount returns how many actual provider sends were recorded for
// opID. Every Send invocation inserts its own row, so the count is
// the send count — never a deduplicated unique-operation count.
func SentCount(t *testing.T, pool *pgxpool.Pool, opID string) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM mock_provider_log WHERE op_id = $1`, opID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Attempt is one recorded provider send.
type Attempt struct {
	ID       int64
	OpID     string
	TenantID string
	Mode     string
	Receipt  string
}

// Attempts returns every recorded send for opID in insertion order.
// Tests asserting exactly-once provider behavior use these rows:
// one row per send, distinct serial IDs, explicit attribution.
func Attempts(t *testing.T, pool *pgxpool.Pool, opID string) []Attempt {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	rows, err := pool.Query(ctx, `SELECT id, op_id, tenant_id, mode, receipt
		FROM mock_provider_log WHERE op_id = $1 ORDER BY id`, opID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []Attempt
	for rows.Next() {
		var a Attempt
		if err := rows.Scan(&a.ID, &a.OpID, &a.TenantID, &a.Mode, &a.Receipt); err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

type directive struct {
	Mock struct {
		Mode    string `json:"mode"`
		Receipt string `json:"receipt"`
		OpID    string `json:"opId"`
	} `json:"mock"`
}

func parseDirective(payload json.RawMessage) directive {
	var d directive
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &d)
	}
	if d.Mock.Mode == "" {
		d.Mock.Mode = ModeAccept
	}
	return d
}

// Send executes one scripted provider call.
func (c *Channel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	d := parseDirective(op.Payload)
	opID := d.Mock.OpID
	if opID == "" {
		opID = op.Operation
	}
	receipt := d.Mock.Receipt
	if receipt == "" {
		receipt = "rcpt-" + opID
	}
	switch d.Mock.Mode {
	case ModeReject:
		return effects.Receipt{}, fmt.Errorf("mock: provider rejected %s", opID)
	case ModeTimeout:
		return effects.Receipt{}, effects.UnknownError{Err: fmt.Errorf("mock: timeout on %s", opID)}
	case ModeSlowAccept:
		select {
		case <-ctx.Done():
			return effects.Receipt{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		fallthrough
	case ModeAccept, ModeLostAck, ModeDelayedAck:
		if _, err := c.pool.Exec(ctx, `INSERT INTO mock_provider_log (op_id, tenant_id, mode, receipt)
			VALUES ($1, $2, $3, $4)`,
			opID, op.TenantID, d.Mock.Mode, receipt); err != nil {
			return effects.Receipt{}, err
		}
		if d.Mock.Mode == ModeDelayedAck {
			select {
			case <-ctx.Done():
				return effects.Receipt{}, ctx.Err()
			case <-time.After(2 * time.Second):
			}
			return effects.Receipt{}, effects.UnknownError{Err: fmt.Errorf("mock: delayed ack lost for %s", opID)}
		}
		if d.Mock.Mode == ModeLostAck {
			return effects.Receipt{}, effects.UnknownError{Err: fmt.Errorf("mock: ack lost for %s", opID)}
		}
		return effects.Receipt{ProviderRequestID: receipt, ProviderReply: "ok:" + opID}, nil
	default:
		return effects.Receipt{}, fmt.Errorf("mock: unknown mode %q", d.Mock.Mode)
	}
}
