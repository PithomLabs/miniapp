package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp/testkit"
)

func testEvent() Event {
	return Event{
		TenantID: "t-audit", AgentID: "agent-1",
		AuthorityKind: "user", AuthorityID: "u1",
		MiniappID: "crm-read", Version: "1.0.0", Tool: "rpa.crm.read_contact",
		InvocationID: "00000000-0000-0000-0000-000000000001",
		Action:       "invoke",
		Outcome:      "allowed",
		Details:      json.RawMessage(`{"key":"k1"}`),
	}
}

func TestAuditRoundTrip(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.audit_events")
	svc := New(db.Pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := svc.Record(ctx, testEvent()); err != nil {
		t.Fatal(err)
	}
	events, err := svc.Query(ctx, "t-audit", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("queried %d events, want 1", len(events))
	}
	got := events[0]
	if got.AgentID != "agent-1" || got.Action != "invoke" || got.Outcome != "allowed" {
		t.Fatalf("attribution lost: %+v", got)
	}
	var details map[string]string
	if err := json.Unmarshal(got.Details, &details); err != nil {
		t.Fatalf("details not valid JSON: %s", got.Details)
	}
	if details["key"] != "k1" {
		t.Fatalf("details lost: %s", got.Details)
	}
	// Tenant isolation on reads.
	other, err := svc.Query(ctx, "t-other", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatalf("cross-tenant audit read returned %d events", len(other))
	}
}

func TestAuditRejectsUnattributed(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.audit_events")
	svc := New(db.Pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	bad := testEvent()
	bad.AgentID = ""
	if err := svc.Record(ctx, bad); err == nil {
		t.Fatal("unattributed event accepted")
	}
	if _, err := svc.Query(ctx, "t-audit", 0); err == nil {
		t.Fatal("unbounded query accepted")
	}
}
