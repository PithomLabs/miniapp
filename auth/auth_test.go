package auth

import (
	"testing"

	"github.com/PithomLabs/miniapp/contract"
)

func testPolicy() Policy {
	return Policy{
		Keys: map[string]Binding{
			"key-agent-1": {
				AgentID:   "agent-1",
				Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "u1", TenantID: "t1"},
			},
		},
		Grants: map[string]map[string]contract.TrustMode{
			"t1": {"crm-read": contract.TrustConfirm},
		},
	}
}

func testManifest() *contract.MiniappManifest {
	return &contract.MiniappManifest{ID: "crm-read", Version: "1.0.0"}
}

func TestAuthenticate(t *testing.T) {
	p := testPolicy()
	b, err := p.Authenticate("key-agent-1")
	if err != nil || b.AgentID != "agent-1" {
		t.Fatalf("valid credential rejected: %v", err)
	}
	for _, bad := range []string{"", "nope"} {
		if _, err := p.Authenticate(bad); err == nil {
			t.Fatalf("credential %q accepted", bad)
		}
	}
	// Identical errors leak nothing about which credentials exist.
	_, errEmpty := p.Authenticate("")
	_, errUnknown := p.Authenticate("nope")
	if errEmpty.Error() != errUnknown.Error() {
		t.Fatal("empty and unknown credentials fail differently")
	}
}

func TestAuthorize(t *testing.T) {
	p := testPolicy()
	b, _ := p.Authenticate("key-agent-1")
	read := contract.ToolManifest{Name: "rpa.crm.read_contact"}
	write := contract.ToolManifest{
		Name:     "rpa.crm.update_contact",
		Approval: contract.ApprovalRequirement{Required: true},
		Effects:  []string{"crm.update_contact"},
	}
	req := func(tool contract.ToolManifest, trust contract.TrustMode) (Decision, contract.TrustMode, error) {
		return p.Authorize(AuthorizeRequest{
			Binding: b, TenantID: "t1", Manifest: testManifest(), Tool: tool, Requested: trust,
		})
	}
	if d, tm, err := req(read, contract.TrustConfirm); err != nil || d != Allow || tm != contract.TrustConfirm {
		t.Fatalf("read confirm = %v %v %v", d, tm, err)
	}
	if d, _, err := req(write, contract.TrustConfirm); err != nil || d != NeedsApproval {
		t.Fatalf("approval-gated write confirm = %v %v", d, err)
	}
	if _, _, err := req(read, contract.TrustAutonomous); err == nil {
		t.Fatal("trust above grant accepted")
	}
	if _, _, err := req(write, contract.TrustDryRun); err == nil {
		t.Fatal("dry-run with effects accepted")
	}
	if _, _, err := req(read, "yolo"); err == nil {
		t.Fatal("unknown trust mode accepted")
	}
	// Tenant binding: a different tenant is denied even with a valid key.
	other := b
	other.Authority.TenantID = "t2"
	_, _, err := p.Authorize(AuthorizeRequest{
		Binding: other, TenantID: "t2", Manifest: testManifest(), Tool: read, Requested: contract.TrustDryRun,
	})
	if err == nil {
		t.Fatal("ungranted tenant accepted")
	}
	// Tenant mismatch between binding and request is denied.
	_, _, err = p.Authorize(AuthorizeRequest{
		Binding: b, TenantID: "t2", Manifest: testManifest(), Tool: read, Requested: contract.TrustDryRun,
	})
	if err == nil {
		t.Fatal("tenant mismatch accepted")
	}
}
