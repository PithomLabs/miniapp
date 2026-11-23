package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
)

// ToolCall is everything a Class A handler needs. Tx is the request's
// tenant-scoped transaction, already armed with the RLS tenant
// context; handlers must not open their own connections.
type ToolCall struct {
	Context      context.Context
	TenantID     string
	Principal    contract.Caller
	Input        json.RawMessage
	Tx           pgx.Tx
	InvocationID string
	// EntityKey is the tenant-scoped serialization entity derived
	// by the runtime, or empty when the tool declares none. The
	// Class A executor already holds the entity lock when set.
	EntityKey string
	// EffectToken authorizes effect claims under this invocation.
	EffectToken string
}

// ToolFunc executes one Class A tool synchronously and returns its
// typed result payload.
type ToolFunc func(ctx context.Context, call ToolCall) (json.RawMessage, error)

// WorkflowEnv is what a Class D miniapp needs at setup: the shared
// DBOS context to register workflows on, the shared datasource to
// run durable transactions against, and the shared application pool
// for framework service clients (such as the effects ledger, whose
// calls live outside DBOS transactions). The executor binds the
// datasource to the pool (the same handle it gave the DBOS system
// database), which selects single-transaction durability. Register
// runs once per tool binding; each Register function registers only
// its own tool's workflow.
type WorkflowEnv struct {
	Context dbos.Context
	Data    *dbos.DataSource
	Pool    *pgxpool.Pool
}

// WorkflowRequest is one validated, authorized Class D invocation.
// The miniapp's Start function translates it into its typed workflow
// input; steps re-derive the tenant from the translated input, never
// from ambient state.
type WorkflowRequest struct {
	InvocationID   string
	TenantID       string
	Principal      contract.Caller
	Input          json.RawMessage
	IdempotencyKey string
	// EntityKey is the tenant-scoped serialization entity derived
	// by the runtime, or empty when the tool declares none.
	EntityKey string
	// EffectToken authorizes effect claims under this invocation.
	EffectToken string
	// ApprovalID names the approval this run waits on, or empty
	// when the invocation was not approval-gated.
	ApprovalID string
}

// WorkflowDef binds a Class D tool to its DBOS workflow. Register is
// invoked once at executor setup; Start launches one run under the
// given run ID; Version is the workflow compatibility line used by
// recovery and deployment guards. Register and Start are plain
// package-level functions, never closures: workflow code must replay
// deterministically.
type WorkflowDef struct {
	Name     string
	Version  string
	Register func(env WorkflowEnv) error
	Start    func(ctx dbos.Context, req WorkflowRequest, runID string) error
}

// Implementation is a miniapp's executable side. Build identifies the
// exact implementation artifact (commit SHA or equivalent); the
// release envelope must bind it to the manifest digest. ClassATools
// and Workflows are keyed by tool name; every manifest tool needs
// exactly one binding of matching class.
type Implementation struct {
	Build       string
	ClassATools map[string]ToolFunc
	Workflows   map[string]WorkflowDef
}

// Registry maps immutable miniapp identity to manifest plus
// implementation. It is safe for concurrent use.
type Registry struct {
	mu   sync.RWMutex
	apps map[string]*entry
}

type entry struct {
	manifest *contract.MiniappManifest
	impl     *Implementation
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{apps: map[string]*entry{}}
}

func key(id, version string) string { return id + "@" + version }

// Register admits one miniapp version, failing closed on any defect:
// invalid manifest, identity or class mismatch, duplicate identity,
// unbound tools, or a release envelope that does not bind this exact
// manifest to this exact build. Envelope issuance (trusted builds)
// arrives in Phase 5; enforcement of the binding starts here.
func (r *Registry) Register(manifest *contract.MiniappManifest, impl *Implementation, env contract.ReleaseEnvelope) error {
	if manifest == nil || impl == nil {
		return fmt.Errorf("registry: manifest and implementation are required")
	}
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("registry: invalid manifest: %w", err)
	}
	digest, err := contract.ProfileDigest(manifest)
	if err != nil {
		return fmt.Errorf("registry: cannot digest manifest: %w", err)
	}
	if impl.Build == "" {
		return fmt.Errorf("registry: implementation build identity is required")
	}
	if !env.Matches(digest, impl.Build) {
		return fmt.Errorf("registry: release envelope does not bind manifest %s to build %q", digest, impl.Build)
	}
	for _, tool := range manifest.Tools {
		name := string(tool.Name)
		_, inA := impl.ClassATools[name]
		_, inD := impl.Workflows[name]
		if len(tool.Effects) > 0 && tool.EntityKey == "" {
			return fmt.Errorf("registry: consequential tool %q must declare entityKey (a reviewed noEntityReason variance is represented in the ADR preflight, not at admission)", name)
		}
		switch tool.Class {
		case contract.ClassA:
			if !inA || inD {
				return fmt.Errorf("registry: class A tool %q needs exactly one Class A binding", name)
			}
		case contract.ClassD:
			if !inD || inA {
				return fmt.Errorf("registry: class D tool %q needs exactly one workflow binding", name)
			}
			if impl.Workflows[name].Version == "" {
				return fmt.Errorf("registry: workflow %q needs an explicit version", name)
			}
			if impl.Workflows[name].Register == nil {
				return fmt.Errorf("registry: workflow %q needs a Register function", name)
			}
			if impl.Workflows[name].Start == nil {
				return fmt.Errorf("registry: workflow %q needs a Start function", name)
			}
		default:
			return fmt.Errorf("registry: tool %q class %s is not served in Phase 1 (Class A/D only)", name, tool.Class)
		}
	}
	versions := map[string]bool{}
	for _, def := range impl.Workflows {
		versions[def.Version] = true
	}
	if len(versions) > 1 {
		return fmt.Errorf("registry: one family, one workflow-compatibility version (miniapp %q declares %d)", manifest.ID, len(versions))
	}
	for name := range impl.ClassATools {
		if !hasTool(manifest, name) {
			return fmt.Errorf("registry: Class A binding %q has no manifest tool", name)
		}
	}
	for name := range impl.Workflows {
		if !hasTool(manifest, name) {
			return fmt.Errorf("registry: workflow binding %q has no manifest tool", name)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(string(manifest.ID), string(manifest.Version))
	if _, dup := r.apps[k]; dup {
		return fmt.Errorf("registry: duplicate registration of %s (refusing silent overwrite)", k)
	}
	r.apps[k] = &entry{manifest: manifest, impl: impl}
	return nil
}

// Resolve returns the manifest and implementation for an exact
// id@version identity. Empty versions never resolve to "latest".
func (r *Registry) Resolve(id, version string) (*contract.MiniappManifest, *Implementation, error) {
	if id == "" || version == "" {
		return nil, nil, fmt.Errorf("registry: miniapp id and version are both required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.apps[key(id, version)]
	if !ok {
		return nil, nil, fmt.Errorf("registry: unknown miniapp %s", key(id, version))
	}
	return e.manifest, e.impl, nil
}

// Range visits every registration. The executor uses it once at
// setup to register Class D workflows; iteration order is
// unspecified.
func (r *Registry) Range(fn func(manifest *contract.MiniappManifest, impl *Implementation) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.apps {
		if !fn(e.manifest, e.impl) {
			return
		}
	}
}

func hasTool(m *contract.MiniappManifest, name string) bool {
	for _, t := range m.Tools {
		if string(t.Name) == name {
			return true
		}
	}
	return false
}
