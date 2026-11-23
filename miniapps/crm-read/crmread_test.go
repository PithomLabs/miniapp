package crmread

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
)

func TestManifestValid(t *testing.T) {
	if err := Manifest().Validate(); err != nil {
		t.Fatalf("manifest invalid: %v", err)
	}
}

func TestRegister(t *testing.T) {
	reg := registry.New()
	if err := Register(reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	mf, impl, err := reg.Resolve(ID, Version)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if mf.Tools[0].Name != ToolSearch {
		t.Fatalf("tool = %q, want %q", mf.Tools[0].Name, ToolSearch)
	}
	if _, ok := impl.ClassATools[ToolSearch]; !ok {
		t.Fatal("Class A binding missing")
	}
	if err := Register(reg); err == nil {
		t.Fatal("duplicate registration accepted")
	}
}

func seed(t *testing.T, db *testkit.DB, tenant, id, name, email string) {
	t.Helper()
	ctx := context.Background()
	_, err := store.WithTenant(ctx, db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(ctx, `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
			VALUES ($1, $2, $3, $4)`, id, tenant, name, email)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func search(t *testing.T, db *testkit.DB, tenant, input string) []contact {
	t.Helper()
	reg := registry.New()
	if err := Register(reg); err != nil {
		t.Fatal(err)
	}
	ex := classa.New(db.Pool, reg)
	raw, err := ex.Execute(context.Background(), executor.Request{
		InvocationID: "test-invocation", TenantID: tenant,
		MiniappID: ID, Version: Version, Tool: ToolSearch,
		Input: json.RawMessage(input),
		Claim: &executor.ClaimHooks{
			Recheck: func(tx pgx.Tx) (bool, error) { return true, nil },
			Complete: func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
				return true, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var out struct {
		Contacts []contact `json:"contacts"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out.Contacts
}

func TestSearchRoundTrip(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.crm_contacts")
	seed(t, db, "tenant-a", "a1", "Ada Lovelace", "ada@example.com")
	seed(t, db, "tenant-a", "a2", "Alan Turing", "alan@example.com")
	seed(t, db, "tenant-b", "b1", "Ada Byron", "byron@example.com")

	got := search(t, db, "tenant-a", `{"query":"ada"}`)
	if len(got) != 1 || got[0].ID != "a1" {
		t.Fatalf("tenant-a search = %+v, want [a1]", got)
	}
	got = search(t, db, "tenant-b", `{"query":"ada"}`)
	if len(got) != 1 || got[0].ID != "b1" {
		t.Fatalf("tenant-b search = %+v, want [b1]", got)
	}
	got = search(t, db, "tenant-a", `{"query":""}`)
	if len(got) != 2 {
		t.Fatalf("empty query = %d contacts, want 2", len(got))
	}
	got = search(t, db, "tenant-a", `{"query":"","limit":1}`)
	if len(got) != 1 {
		t.Fatalf("limit 1 = %d contacts, want 1", len(got))
	}
}
