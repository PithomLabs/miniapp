package contract

import "encoding/json"

// Class is the Dimension A execution class of a tool.
type Class string

const (
	ClassA Class = "A"
	ClassB Class = "B"
	ClassC Class = "C"
	ClassD Class = "D"
)

// Substrate is the declared execution substrate for a tool.
type Substrate string

const (
	SubstrateGo    Substrate = "go"
	SubstrateRiver Substrate = "river"
	SubstrateDBOS  Substrate = "dbos"
)

// Placement is the Dimension B compute placement for a tool.
type Placement string

const (
	PlacementMachine Placement = "machine"
	PlacementSprite  Placement = "sprite"
	PlacementHybrid  Placement = "hybrid"
	PlacementLocal   Placement = "local"
)

// CapabilityRef pins one approved capability a tool may call.
type CapabilityRef struct {
	Provider  string  `json:"provider"`
	Operation string  `json:"operation"`
	Version   Version `json:"version"`
}

func (r CapabilityRef) validate(field string) *MultiError {
	var m MultiError
	if err := Name(r.Provider).Validate(field + ".provider"); err != nil {
		m.Add(err.(*Error))
	}
	if err := Name(r.Operation).Validate(field + ".operation"); err != nil {
		m.Add(err.(*Error))
	}
	if err := r.Version.Validate(field + ".version"); err != nil {
		m.Add(err.(*Error))
	}
	return &m
}

// ApprovalRequirement declares when a human or authorized principal
// must approve a tool invocation.
type ApprovalRequirement struct {
	Required      bool     `json:"required"`
	ApproverScope string   `json:"approverScope,omitempty"`
	SelfApproval  bool     `json:"selfApproval,omitempty"`
	Expiry        string   `json:"expiry,omitempty"`
	Thresholds    []string `json:"thresholds,omitempty"`
}

// IdempotencyRequirement declares the idempotency contract of a tool.
type IdempotencyRequirement struct {
	Required bool   `json:"required"`
	KeyScope string `json:"keyScope,omitempty"`
}

// RetryPolicy declares bounded retry behavior. Empty means no retries
// beyond the substrate default; MaxAttempts 0 with Backoff set is invalid.
type RetryPolicy struct {
	MaxAttempts int    `json:"maxAttempts,omitempty"`
	Backoff     string `json:"backoff,omitempty"`
}

// SagaDeclaration names the compensation contract for multi-effect tools.
type SagaDeclaration struct {
	Compensations []string `json:"compensations,omitempty"`
	Irreversible  []string `json:"irreversible,omitempty"`
}

// BlastRadiusLimits declares enforceable server-side bounds for a tool.
// Limits are declared in the manifest and enforced centrally so no
// horizontal worker can bypass them.
type BlastRadiusLimits struct {
	MaxRunsPerHour   int      `json:"maxRunsPerHour,omitempty"`
	MaxRecordsPerRun int      `json:"maxRecordsPerRun,omitempty"`
	MaxAmount        string   `json:"maxAmount,omitempty"`
	RecipientAllow   []string `json:"recipientAllowlist,omitempty"`
	DomainAllow      []string `json:"domainAllowlist,omitempty"`
	KillSwitch       string   `json:"killSwitch,omitempty"`
}

func (l BlastRadiusLimits) validate(field string) *MultiError {
	var m MultiError
	if l.MaxRunsPerHour < 0 {
		m.Add(Terminal(field+".maxRunsPerHour", "must not be negative"))
	}
	if l.MaxRecordsPerRun < 0 {
		m.Add(Terminal(field+".maxRecordsPerRun", "must not be negative"))
	}
	return &m
}

// AuditRequirement declares audit retention and redaction needs.
type AuditRequirement struct {
	Events    []string `json:"events,omitempty"`
	Retention string   `json:"retention,omitempty"`
	Redact    []string `json:"redact,omitempty"`
}

// ToolManifest is the frozen per-tool contract: identity, class,
// schemas, capabilities, policy, and recovery declarations.
type ToolManifest struct {
	Name         ToolName               `json:"name"`
	Description  string                 `json:"description"`
	Class        Class                  `json:"class"`
	Substrate    Substrate              `json:"substrate"`
	InputSchema  json.RawMessage        `json:"inputSchema"`
	OutputSchema json.RawMessage        `json:"outputSchema"`
	Capabilities []CapabilityRef        `json:"capabilities,omitempty"`
	TrustModes   []TrustMode            `json:"trustModes,omitempty"`
	Approval     ApprovalRequirement    `json:"approval,omitempty"`
	Idempotency  IdempotencyRequirement `json:"idempotency,omitempty"`
	Timeout      string                 `json:"timeout,omitempty"`
	Retry        RetryPolicy            `json:"retry,omitempty"`
	Cancellable  bool                   `json:"cancellable,omitempty"`
	Effects      []string               `json:"effects,omitempty"`
	Saga         *SagaDeclaration       `json:"saga,omitempty"`
	Audit        AuditRequirement       `json:"audit,omitempty"`
	Fixtures     []string               `json:"fixtures,omitempty"`
	Examples     []string               `json:"examples,omitempty"`
	EntityKey    string                 `json:"entityKey,omitempty"`
	NoEntityWhy  string                 `json:"noEntityReason,omitempty"`
	Limits       BlastRadiusLimits      `json:"limits,omitempty"`
}

func validJSON(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	_, ok := v.(map[string]any)
	return ok
}

func (t ToolManifest) validate(field string) *MultiError {
	var m MultiError
	if err := t.Name.Validate(field + ".name"); err != nil {
		m.Add(err.(*Error))
	}
	if t.Description == "" {
		m.Add(Terminal(field+".description", "must not be empty"))
	}
	switch t.Class {
	case ClassA, ClassB, ClassC, ClassD:
	default:
		m.Add(Terminal(field+".class", "must be one of A, B, C, D"))
	}
	if err := checkSubstrate(t.Class, t.Substrate); err != nil {
		m.Add(Terminal(field+".substrate", "%s", err.Error()))
	}
	if !validJSON(t.InputSchema) {
		m.Add(Terminal(field+".inputSchema", "must be a JSON object schema"))
	}
	if !validJSON(t.OutputSchema) {
		m.Add(Terminal(field+".outputSchema", "must be a JSON object schema"))
	}
	for i, c := range t.Capabilities {
		sub := c.validate(field + ".capabilities")
		m.Errs = append(m.Errs, sub.Errs...)
		_ = i
	}
	for _, tm := range t.TrustModes {
		if err := tm.Validate(field + ".trustModes"); err != nil {
			m.Add(err.(*Error))
		}
	}
	if t.Retry.Backoff != "" && t.Retry.MaxAttempts <= 0 {
		m.Add(Terminal(field+".retry", "backoff without maxAttempts is invalid"))
	}
	// Tools with declared external effects are consequential writes at
	// the structural level and must serialize on an entity key or carry
	// an explicit reviewed justification. Risk-class-aware enforcement
	// lands with registry admission in Phase 1.
	if len(t.Effects) > 0 && t.EntityKey == "" && t.NoEntityWhy == "" {
		m.Add(Terminal(field, "tools with effects must declare entityKey or noEntityReason"))
	}
	sub := t.Limits.validate(field + ".limits")
	m.Errs = append(m.Errs, sub.Errs...)
	return &m
}

// checkSubstrate enforces the class-to-substrate matrix. Class C may
// declare DBOS only when its promotion is justified elsewhere; the
// admission check (not this structural validation) enforces the rubric.
func checkSubstrate(class Class, s Substrate) error {
	switch class {
	case ClassA:
		if s != SubstrateGo {
			return Terminal("", "class A requires substrate go")
		}
	case ClassB:
		if s != SubstrateRiver {
			return Terminal("", "class B requires substrate river")
		}
	case ClassC:
		if s != SubstrateRiver && s != SubstrateDBOS {
			return Terminal("", "class C requires substrate river or justified dbos")
		}
	case ClassD:
		if s != SubstrateDBOS {
			return Terminal("", "class D requires substrate dbos")
		}
	}
	switch s {
	case SubstrateGo, SubstrateRiver, SubstrateDBOS:
		return nil
	default:
		return Terminal("", "must be one of go, river, dbos")
	}
}

// ProfileRef binds a deployment profile by name and content digest.
// The digest is mandatory: validate rejects unbound references (A4).
type ProfileRef struct {
	Name   string `json:"name"`
	Digest Digest `json:"digest"`
}

func (r ProfileRef) validate(field string) *MultiError {
	var m MultiError
	if err := Name(r.Name).Validate(field + ".name"); err != nil {
		m.Add(err.(*Error))
	}
	if err := r.Digest.Validate(field + ".digest"); err != nil {
		m.Add(err.(*Error))
	}
	return &m
}

// ToolDeployment is the Dimension B declaration for one tool.
type ToolDeployment struct {
	Placement Placement  `json:"placement"`
	Profile   ProfileRef `json:"profile"`
}

// PromotionPolicy declares the review gate for releases.
type PromotionPolicy struct {
	ReviewRequired bool     `json:"reviewRequired"`
	RequiredChecks []string `json:"requiredChecks,omitempty"`
}

// DeploymentSection is the manifest's Dimension B block.
type DeploymentSection struct {
	Tools     map[string]ToolDeployment `json:"tools"`
	Promotion PromotionPolicy           `json:"promotion,omitempty"`
}

// DependencyManifest pins every input needed to reproduce a miniapp.
type DependencyManifest struct {
	DBOSSDK            string `json:"dbosSdk,omitempty"`
	River              string `json:"river,omitempty"`
	MCPSDK             string `json:"mcpSdk,omitempty"`
	SpriteSDK          string `json:"spriteSdk,omitempty"`
	CapabilityRegistry Digest `json:"capabilityRegistry,omitempty"`
	Policy             string `json:"policy,omitempty"`
	Fixtures           Digest `json:"fixtures,omitempty"`
}

// MiniappManifest is the versioned artifact of record.
type MiniappManifest struct {
	ID            Name               `json:"id"`
	Version       Version            `json:"version"`
	Description   string             `json:"description"`
	Examples      []string           `json:"examples,omitempty"`
	Tools         []ToolManifest     `json:"tools"`
	Deployment    DeploymentSection  `json:"deployment"`
	Dependencies  DependencyManifest `json:"dependencies,omitempty"`
	Compatibility string             `json:"compatibility,omitempty"`
}

// Validate checks the full manifest structurally, including that the
// deployment section covers every declared tool (A4/A12 admission input).
func (mf MiniappManifest) Validate() error {
	var m MultiError
	if err := mf.ID.Validate("id"); err != nil {
		m.Add(err.(*Error))
	}
	if err := mf.Version.Validate("version"); err != nil {
		m.Add(err.(*Error))
	}
	if mf.Description == "" {
		m.Add(Terminal("description", "must not be empty"))
	}
	if len(mf.Tools) == 0 {
		m.Add(Terminal("tools", "must declare at least one tool"))
	}
	seen := map[string]bool{}
	for i, t := range mf.Tools {
		field := "tools"
		sub := t.validate(field)
		m.Errs = append(m.Errs, sub.Errs...)
		name := string(t.Name)
		if seen[name] {
			m.Add(Terminal(field, "duplicate tool %q", name))
		}
		seen[name] = true
		dep, ok := mf.Deployment.Tools[name]
		if !ok {
			m.Add(Terminal("deployment.tools", "tool %q (index %d) has no placement", name, i))
			continue
		}
		switch dep.Placement {
		case PlacementMachine, PlacementSprite, PlacementHybrid, PlacementLocal:
		default:
			m.Add(Terminal("deployment.tools", "tool %q has invalid placement", name))
		}
		sub2 := dep.Profile.validate("deployment.tools." + name + ".profile")
		m.Errs = append(m.Errs, sub2.Errs...)
	}
	for name := range mf.Deployment.Tools {
		if !seen[name] {
			m.Add(Terminal("deployment.tools", "placement for unknown tool %q", name))
		}
	}
	if mf.Dependencies.CapabilityRegistry != "" {
		if err := mf.Dependencies.CapabilityRegistry.Validate("dependencies.capabilityRegistry"); err != nil {
			m.Add(err.(*Error))
		}
	}
	if mf.Dependencies.Fixtures != "" {
		if err := mf.Dependencies.Fixtures.Validate("dependencies.fixtures"); err != nil {
			m.Add(err.(*Error))
		}
	}
	return m.OrNil()
}
