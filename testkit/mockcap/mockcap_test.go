package mockcap_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

func channel(t *testing.T) (*mockcap.Channel, *testkit.DB) {
	t.Helper()
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	return mockcap.New(db.Pool), db
}

func payload(mode, opID string) json.RawMessage {
	return json.RawMessage(`{"mock":{"mode":"` + mode + `","opId":"` + opID + `"}}`)
}

func TestModes(t *testing.T) {
	ch, db := channel(t)
	ctx := context.Background()

	rcpt, err := ch.Send(ctx, effects.Operation{CapabilityID: "mock", Operation: "send", Payload: payload(mockcap.ModeAccept, "op-a")})
	if err != nil || rcpt.ProviderRequestID == "" {
		t.Fatalf("accept = %+v, %v", rcpt, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-a"); n != 1 {
		t.Fatalf("accept log rows = %d, want 1", n)
	}

	if _, err := ch.Send(ctx, effects.Operation{CapabilityID: "mock", Operation: "send", Payload: payload(mockcap.ModeReject, "op-r")}); err == nil {
		t.Fatal("reject succeeded")
	}
	if n := mockcap.SentCount(t, db.Pool, "op-r"); n != 0 {
		t.Fatalf("reject log rows = %d, want 0", n)
	}

	_, err = ch.Send(ctx, effects.Operation{CapabilityID: "mock", Operation: "send", Payload: payload(mockcap.ModeTimeout, "op-t")})
	var unknown effects.UnknownError
	if !errors.As(err, &unknown) {
		t.Fatalf("timeout err = %v, want UnknownError", err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-t"); n != 0 {
		t.Fatalf("timeout log rows = %d, want 0", n)
	}

	_, err = ch.Send(ctx, effects.Operation{CapabilityID: "mock", Operation: "send", Payload: payload(mockcap.ModeLostAck, "op-l")})
	if !errors.As(err, &unknown) {
		t.Fatalf("lost-ack err = %v, want UnknownError", err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-l"); n != 1 {
		t.Fatalf("lost-ack log rows = %d, want 1 (provider has it)", n)
	}

	if _, err := ch.Send(ctx, effects.Operation{CapabilityID: "mock", Operation: "send", Payload: payload("bogus", "op-b")}); err == nil {
		t.Fatal("bogus mode succeeded")
	}

	_, err = ch.Send(ctx, effects.Operation{CapabilityID: "mock", Operation: "send", Payload: payload(mockcap.ModeDelayedAck, "op-d")})
	if !errors.As(err, &unknown) {
		t.Fatalf("delayed-ack err = %v, want UnknownError", err)
	}
	if n := mockcap.SentCount(t, db.Pool, "op-d"); n != 1 {
		t.Fatalf("delayed-ack log rows = %d, want 1", n)
	}
}

// TestMockAttemptRowsAttributed asserts the row-level shape behind
// the count: distinct serial IDs, insertion order, and correct
// tenant/operation attribution for each recorded send.
func TestMockAttemptRowsAttributed(t *testing.T) {
	ch, db := channel(t)
	ctx := context.Background()
	op := effects.Operation{CapabilityID: "mock", Operation: "send",
		Payload:  payload(mockcap.ModeAccept, "op-rows"),
		TenantID: "tenant-x"}

	for i := 0; i < 2; i++ {
		if _, err := ch.Send(ctx, op); err != nil {
			t.Fatalf("send %d = %v", i, err)
		}
	}
	rows := mockcap.Attempts(t, db.Pool, "op-rows")
	if len(rows) != 2 {
		t.Fatalf("attempt rows = %d, want 2", len(rows))
	}
	if rows[0].ID == rows[1].ID {
		t.Fatalf("serial IDs not distinct: %+v", rows)
	}
	for i, r := range rows {
		if r.OpID != "op-rows" || r.TenantID != "tenant-x" || r.Mode != mockcap.ModeAccept {
			t.Fatalf("row %d misattributed: %+v", i, r)
		}
	}
	if rows[0].ID > rows[1].ID {
		t.Fatalf("rows out of insertion order: %+v", rows)
	}
}

type allowCanceller struct{}

func (allowCanceller) CheckCancelled(_ context.Context, _, _ string) error { return nil }

type allowPolicy struct{}

func (allowPolicy) CheckPolicy(_ context.Context, _ string, _ effects.Intent) error { return nil }

// TestMockTenantAttribution proves provider attempts carry the
// tenant that authorized them (X-24e): two tenants performing
// through the real effects service with the same provider op ID
// produce two rows attributed to their respective tenants.
func TestMockTenantAttribution(t *testing.T) {
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	db.Truncate(t, "app.effect_records")
	ctx := context.Background()
	svc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: allowCanceller{}, Policy: allowPolicy{}})
	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		if _, err := svc.Perform(ctx, tenant, effects.Intent{
			BusinessOpID: tenant + ":shared-op", AttemptID: "a1",
			InvocationID: "inv-" + tenant,
			Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
				Payload: payload(mockcap.ModeAccept, "shared-provider-op")},
		}); err != nil {
			t.Fatalf("perform %s = %v", tenant, err)
		}
	}
	rows := mockcap.Attempts(t, db.Pool, "shared-provider-op")
	if len(rows) != 2 {
		t.Fatalf("attempt rows = %d, want 2", len(rows))
	}
	got := map[string]bool{rows[0].TenantID: true, rows[1].TenantID: true}
	if !got["tenant-a"] || !got["tenant-b"] {
		t.Fatalf("tenant attribution = %+v, want tenant-a and tenant-b", rows)
	}
}
