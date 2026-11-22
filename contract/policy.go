package contract

// TrustMode bounds how much autonomy a miniapp version has per tenant.
// It is set by tenant policy, never by the calling agent.
type TrustMode string

const (
	TrustDryRun     TrustMode = "dry-run"
	TrustConfirm    TrustMode = "confirm"
	TrustAutonomous TrustMode = "autonomous"
)

// Validate reports whether t is a known trust mode.
func (t TrustMode) Validate(field string) error {
	switch t {
	case TrustDryRun, TrustConfirm, TrustAutonomous:
		return nil
	default:
		return Terminal(field, "must be dry-run, confirm, or autonomous")
	}
}

// AuthorityKind names which principal's authority permits an operation.
type AuthorityKind string

const (
	AuthorityUser    AuthorityKind = "user"
	AuthorityService AuthorityKind = "service"
	AuthorityTrigger AuthorityKind = "trigger"
)

// Authority is the frozen authority-principal shape: a delegated human
// user, an explicitly approved tenant service identity, or an authorized
// system/scheduled trigger.
type Authority struct {
	Kind     AuthorityKind `json:"kind"`
	ID       string        `json:"id"`
	TenantID string        `json:"tenantId"`
}

// Caller carries both principals of every invocation: the agent client
// and the authority under which the operation is permitted.
type Caller struct {
	AgentID   string    `json:"agentId"`
	Authority Authority `json:"authority"`
	Scopes    []string  `json:"scopes,omitempty"`
}

// Validate checks caller shape. Authorization decisions belong to the
// auth package; this validates structure only.
func (c Caller) Validate() error {
	var m MultiError
	if c.AgentID == "" {
		m.Add(Terminal("agentId", "must identify the calling agent principal"))
	}
	switch c.Authority.Kind {
	case AuthorityUser, AuthorityService, AuthorityTrigger:
	default:
		m.Add(Terminal("authority.kind", "must be user, service, or trigger"))
	}
	if c.Authority.ID == "" {
		m.Add(Terminal("authority.id", "must not be empty"))
	}
	if c.Authority.TenantID == "" {
		m.Add(Terminal("authority.tenantId", "must not be empty"))
	}
	return m.OrNil()
}
