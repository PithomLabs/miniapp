package effects

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
)

// PolicyChecker revalidates authority at effect-claim time. Production
// wires StrictPolicy; tests that do not exercise policy use an
// explicit stub so the gap is visible, not accidental.
type PolicyChecker interface {
	CheckPolicy(ctx context.Context, tenantID string, intent Intent) error
}

// StrictPolicy resolves every policy input server-side: the tool and
// trust mode come from the invocation row (never the Intent), the
// grant from live policy maps, limits from the registered manifest,
// and kill switches from central state. A miniapp that forges any of
// these in its Intent gains nothing.
type StrictPolicy struct {
	Pool     *pgxpool.Pool
	Registry *registry.Registry
	Grants   *auth.Policy
}

// CheckPolicy enforces, in order: invocation binding (token match),
// live grant coverage, and kill switches (global, tenant, miniapp,
// capability, tool-declared).
func (p StrictPolicy) CheckPolicy(ctx context.Context, tenantID string, intent Intent) error {
	if p.Pool == nil || p.Registry == nil {
		return fmt.Errorf("effects: policy checker is not wired")
	}
	inv, err := p.loadInvocation(ctx, tenantID, intent.InvocationID)
	if err != nil {
		return err
	}
	if intent.EffectToken == "" || inv.token == "" || intent.EffectToken != inv.token {
		return fmt.Errorf("effects: effect token mismatch")
	}
	if !p.Grants.GrantAllows(tenantID, inv.miniapp, inv.trust) {
		return fmt.Errorf("effects: no live grant covers %s at this trust", inv.miniapp)
	}
	manifest, _, err := p.Registry.Resolve(inv.miniapp, inv.version)
	if err != nil {
		return fmt.Errorf("effects: %w", err)
	}
	tool, err := findTool(manifest, inv.tool)
	if err != nil {
		return fmt.Errorf("effects: %w", err)
	}
	names := killSwitchNames(tenantID, inv.miniapp, intent.Operation.CapabilityID, tool.Limits.KillSwitch)
	var hit string
	err = p.Pool.QueryRow(ctx, `SELECT name FROM app.kill_switches
		WHERE name = ANY($1) AND engaged LIMIT 1`, names).Scan(&hit)
	if err == nil {
		return fmt.Errorf("effects: kill switch %q is engaged", hit)
	}
	if !isNoRows(err) {
		return err
	}
	return nil
}

// killSwitchNames is the single source for applicable switch
// names, shared by the pre-claim coarse gate and the
// authorization transaction: global, tenant, miniapp,
// capability, plus the tool-declared switch when present.
func killSwitchNames(tenantID, miniapp, capabilityID, toolSwitch string) []string {
	names := []string{"global", contract.ScopeID("tenant", tenantID), "miniapp:" + miniapp,
		"capability:" + capabilityID}
	if toolSwitch != "" {
		names = append(names, toolSwitch)
	}
	return names
}

type invocationPolicy struct {
	miniapp string
	version string
	tool    string
	trust   contract.TrustMode
	token   string
}

func (p StrictPolicy) loadInvocation(ctx context.Context, tenantID, invocationID string) (invocationPolicy, error) {
	var inv invocationPolicy
	_, err := store.WithTenant(ctx, p.Pool, tenantID, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(ctx, `SELECT miniapp_id, miniapp_version, tool, trust_mode, effect_token
			FROM app.invocations WHERE id = $1::uuid AND tenant_id = $2`,
			invocationID, tenantID).Scan(&inv.miniapp, &inv.version, &inv.tool, &inv.trust, &inv.token)
	})
	if err != nil {
		if isNoRows(err) {
			return invocationPolicy{}, fmt.Errorf("effects: unknown invocation")
		}
		return invocationPolicy{}, err
	}
	return inv, nil
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func findTool(manifest *contract.MiniappManifest, name string) (contract.ToolManifest, error) {
	for _, t := range manifest.Tools {
		if string(t.Name) == name {
			return t, nil
		}
	}
	return contract.ToolManifest{}, fmt.Errorf("tool %q is not declared", name)
}
