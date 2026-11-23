package registry

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dbos-inc/dbos-transact-golang/dbos"

	"github.com/PithomLabs/miniapp/contract"
)

func testManifest() *contract.MiniappManifest {
	return &contract.MiniappManifest{
		ID:          "crm-read",
		Version:     "1.0.0",
		Description: "Read-only CRM contact access.",
		Tools: []contract.ToolManifest{{
			Name:         "rpa.crm.read_contact",
			Description:  "Read one contact.",
			Class:        contract.ClassA,
			Substrate:    contract.SubstrateGo,
			InputSchema:  json.RawMessage(`{"type":"object"}`),
			OutputSchema: json.RawMessage(`{"type":"object"}`),
		}},
		Deployment: contract.DeploymentSection{
			Tools: map[string]contract.ToolDeployment{
				"rpa.crm.read_contact": {
					Placement: contract.PlacementMachine,
					Profile: contract.ProfileRef{
						Name:   "prod-machine",
						Digest: contract.Digest("sha256:" + strings.Repeat("0", 64)),
					},
				},
			},
		},
	}
}

func testImpl() *Implementation {
	return &Implementation{
		Build: "test-build-1",
		ClassATools: map[string]ToolFunc{
			"rpa.crm.read_contact": func(ctx context.Context, call ToolCall) (json.RawMessage, error) {
				return json.RawMessage(`{}`), nil
			},
		},
	}
}

func envelopeFor(t *testing.T, mf *contract.MiniappManifest, build string) contract.ReleaseEnvelope {
	t.Helper()
	d, err := contract.ProfileDigest(mf)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ReleaseEnvelope{Manifest: d, Implementation: build}
}

func TestRegisterResolveRoundTrip(t *testing.T) {
	r := New()
	mf := testManifest()
	if err := r.Register(mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err != nil {
		t.Fatal(err)
	}
	gotMf, gotImpl, err := r.Resolve("crm-read", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if gotMf != mf || gotImpl.Build != "test-build-1" {
		t.Fatal("resolve returned the wrong registration")
	}
}

func TestRegisterRejects(t *testing.T) {
	newReg := func(t *testing.T, mf *contract.MiniappManifest, impl *Implementation, env contract.ReleaseEnvelope) error {
		t.Helper()
		return New().Register(mf, impl, env)
	}
	t.Run("duplicate", func(t *testing.T) {
		r := New()
		mf := testManifest()
		env := envelopeFor(t, mf, "test-build-1")
		if err := r.Register(mf, testImpl(), env); err != nil {
			t.Fatal(err)
		}
		if err := r.Register(mf, testImpl(), env); err == nil {
			t.Fatal("duplicate registration accepted")
		}
	})
	t.Run("invalid-manifest", func(t *testing.T) {
		mf := testManifest()
		mf.Version = "1.0"
		if err := newReg(t, mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("invalid manifest accepted")
		}
	})
	t.Run("effectful-without-entity-key", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Effects = []string{"mock.send"}
		mf.Tools[0].NoEntityWhy = "broadcast has no entity (reviewed)"
		if err := newReg(t, mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("effectful tool without entityKey admitted")
		}
	})

	t.Run("effectful-with-entity-key", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Effects = []string{"mock.send"}
		mf.Tools[0].EntityKey = "contact_id"
		if err := newReg(t, mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err != nil {
			t.Fatalf("effectful tool with entityKey rejected: %v", err)
		}
	})

	t.Run("envelope-manifest-mismatch", func(t *testing.T) {
		mf := testManifest()
		env := envelopeFor(t, mf, "test-build-1")
		mf.Description = "tampered after envelope issuance"
		if err := newReg(t, mf, testImpl(), env); err == nil {
			t.Fatal("tampered manifest accepted against stale envelope")
		}
	})
	t.Run("envelope-build-mismatch", func(t *testing.T) {
		mf := testManifest()
		if err := newReg(t, mf, testImpl(), envelopeFor(t, mf, "other-build")); err == nil {
			t.Fatal("mismatched build identity accepted")
		}
	})
	t.Run("missing-binding", func(t *testing.T) {
		mf := testManifest()
		impl := testImpl()
		impl.ClassATools = nil
		if err := newReg(t, mf, impl, envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("unbound tool accepted")
		}
	})
	t.Run("stray-binding", func(t *testing.T) {
		mf := testManifest()
		impl := testImpl()
		impl.ClassATools["rpa.crm.ghost"] = impl.ClassATools["rpa.crm.read_contact"]
		if err := newReg(t, mf, impl, envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("undeclared binding accepted")
		}
	})
	t.Run("class-mismatch", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Class = contract.ClassD
		mf.Tools[0].Substrate = contract.SubstrateDBOS
		if err := newReg(t, mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("class D tool with Class A binding accepted")
		}
	})
	t.Run("class-b-rejected", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Class = contract.ClassB
		mf.Tools[0].Substrate = contract.SubstrateRiver
		if err := newReg(t, mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("Class B tool accepted in Phase 1")
		}
	})
	t.Run("workflow-needs-version", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Class = contract.ClassD
		mf.Tools[0].Substrate = contract.SubstrateDBOS
		impl := &Implementation{Build: "test-build-1", Workflows: map[string]WorkflowDef{
			"rpa.crm.read_contact": {Name: "wf", Register: func(env WorkflowEnv) error { return nil }},
		}}
		if err := newReg(t, mf, impl, envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("unversioned workflow accepted")
		}
	})

	t.Run("workflow-needs-start", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Class = contract.ClassD
		mf.Tools[0].Substrate = contract.SubstrateDBOS
		impl := &Implementation{Build: "test-build-1", Workflows: map[string]WorkflowDef{
			"rpa.crm.read_contact": {Name: "wf", Version: "1", Register: func(env WorkflowEnv) error { return nil }},
		}}
		if err := newReg(t, mf, impl, envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("workflow without Start accepted")
		}
	})

	t.Run("family-version-agreement", func(t *testing.T) {
		mf := testManifest()
		mf.Tools[0].Class = contract.ClassD
		mf.Tools[0].Substrate = contract.SubstrateDBOS
		mf.Tools = append(mf.Tools, mf.Tools[0])
		mf.Tools[1].Name = "rpa.crm.read_contact2"
		mf.Deployment.Tools["rpa.crm.read_contact2"] = mf.Deployment.Tools["rpa.crm.read_contact"]
		impl := &Implementation{Build: "test-build-1", Workflows: map[string]WorkflowDef{
			"rpa.crm.read_contact":  {Name: "wf", Version: "1", Register: func(env WorkflowEnv) error { return nil }, Start: func(ctx dbos.Context, req WorkflowRequest, runID string) error { return nil }},
			"rpa.crm.read_contact2": {Name: "wf", Version: "2", Register: func(env WorkflowEnv) error { return nil }, Start: func(ctx dbos.Context, req WorkflowRequest, runID string) error { return nil }},
		}}
		if err := newReg(t, mf, impl, envelopeFor(t, mf, "test-build-1")); err == nil {
			t.Fatal("family with mixed workflow versions admitted")
		}
	})
}

func TestResolveRejects(t *testing.T) {
	r := New()
	mf := testManifest()
	if err := r.Register(mf, testImpl(), envelopeFor(t, mf, "test-build-1")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range [][2]string{{"", "1.0.0"}, {"crm-read", ""}, {"nope", "1.0.0"}, {"crm-read", "9.9.9"}} {
		if _, _, err := r.Resolve(tc[0], tc[1]); err == nil {
			t.Fatalf("resolve(%q, %q) accepted", tc[0], tc[1])
		}
	}
}
