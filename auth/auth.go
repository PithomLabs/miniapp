package auth

import (
	"fmt"

	"github.com/PithomLabs/miniapp/contract"
)

// Binding ties one presented credential to an agent principal and its
// authority, both resolved server-side. Phase 1 loads bindings from
// trusted configuration; nothing about the authority is accepted from
// the caller. Roles are owner-defined authorization strings; approval
// scopes match against them.
type Binding struct {
	AgentID   string
	Authority contract.Authority
	Roles     []string
}

// Policy is the static Phase 1 authorization policy. Keys maps a
// presented credential to its binding; Grants maps tenant to miniapp
// to the maximum trust mode granted. Unlisted pairs are denied.
type Policy struct {
	Keys   map[string]Binding
	Grants map[string]map[string]contract.TrustMode
}

// Authenticate resolves a presented credential to its binding. Empty
// and unknown credentials fail identically so callers learn nothing
// about which credentials exist.
func (p Policy) Authenticate(credential string) (Binding, error) {
	if credential == "" {
		return Binding{}, fmt.Errorf("auth: authentication required")
	}
	b, ok := p.Keys[credential]
	if !ok {
		return Binding{}, fmt.Errorf("auth: authentication required")
	}
	if err := validBinding(b); err != nil {
		return Binding{}, err
	}
	return b, nil
}

func validBinding(b Binding) error {
	if b.AgentID == "" {
		return fmt.Errorf("auth: binding has no agent identity")
	}
	c := contract.Caller{AgentID: b.AgentID, Authority: b.Authority}
	if err := c.Validate(); err != nil {
		return fmt.Errorf("auth: invalid binding: %w", err)
	}
	return nil
}

// Decision is the outcome of an authorization check.
type Decision int

const (
	// Deny refuses the invocation with a reason.
	Deny Decision = iota
	// Allow permits immediate execution.
	Allow
	// NeedsApproval permits execution only after a recorded approval.
	NeedsApproval
)

// AuthorizeRequest carries everything decided server-side about one
// invocation. TenantID must come from the resolved authority, never
// from client input; the runtime guarantees this by construction.
type AuthorizeRequest struct {
	Binding   Binding
	TenantID  string
	Manifest  *contract.MiniappManifest
	Tool      contract.ToolManifest
	Requested contract.TrustMode
}

// Authorize enforces, in order: tenant binding, tool membership,
// trust ceiling, and approval gating. It returns the effective trust
// mode alongside the decision. Requesting more trust than granted is
// denied rather than silently downgraded.
func (p Policy) Authorize(req AuthorizeRequest) (Decision, contract.TrustMode, error) {
	if req.Manifest == nil {
		return Deny, "", fmt.Errorf("auth: manifest is required")
	}
	if req.TenantID == "" || req.TenantID != req.Binding.Authority.TenantID {
		return Deny, "", fmt.Errorf("auth: tenant mismatch")
	}
	grants, ok := p.Grants[req.TenantID]
	if !ok {
		return Deny, "", fmt.Errorf("auth: tenant %q has no grants", req.TenantID)
	}
	maxTrust, ok := grants[string(req.Manifest.ID)]
	if !ok {
		return Deny, "", fmt.Errorf("auth: miniapp %q is not granted to tenant", req.Manifest.ID)
	}
	if trustRank(req.Requested) < 0 {
		return Deny, "", fmt.Errorf("auth: unknown trust mode %q", req.Requested)
	}
	if trustRank(req.Requested) > trustRank(maxTrust) {
		return Deny, "", fmt.Errorf("auth: requested trust %q exceeds grant %q", req.Requested, maxTrust)
	}
	effective := req.Requested
	if effective == contract.TrustDryRun && len(req.Tool.Effects) > 0 {
		return Deny, "", fmt.Errorf("auth: dry-run forbids tools with external effects")
	}
	if req.Tool.Approval.Required && effective != contract.TrustAutonomous {
		return NeedsApproval, effective, nil
	}
	return Allow, effective, nil
}

// GrantAllows reports whether the tenant's current grant for the
// miniapp covers the stated trust mode. Effect-time revalidation
// calls it against live policy maps, so revocations take effect
// without a redeploy.
func (p Policy) GrantAllows(tenant, miniapp string, want contract.TrustMode) bool {
	grants, ok := p.Grants[tenant]
	if !ok {
		return false
	}
	have, ok := grants[miniapp]
	if !ok {
		return false
	}
	return trustRank(want) >= 0 && trustRank(want) <= trustRank(have)
}

func trustRank(t contract.TrustMode) int {
	switch t {
	case contract.TrustDryRun:
		return 0
	case contract.TrustConfirm:
		return 1
	case contract.TrustAutonomous:
		return 2
	default:
		return -1
	}
}
