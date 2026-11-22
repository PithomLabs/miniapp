package contract

import (
	"encoding/json"
	"net/url"
)

// RiskClass classifies the blast radius of a capability operation.
type RiskClass string

const (
	RiskRead          RiskClass = "read"
	RiskReversible    RiskClass = "reversible-write"
	RiskConsequential RiskClass = "consequential-write"
)

// AttributionMode declares whose SaaS identity an action is attributed
// to: the delegated user or an approved service identity.
type AttributionMode string

const (
	AttributionUser    AttributionMode = "delegated-user"
	AttributionService AttributionMode = "service-identity"
)

// CapabilitySpec is the frozen contract for one approved operation
// against a SaaS or custom application.
type CapabilitySpec struct {
	Provider       string                 `json:"provider"`
	Operation      string                 `json:"operation"`
	APIVersion     string                 `json:"apiVersion"`
	InputSchema    json.RawMessage        `json:"inputSchema"`
	OutputSchema   json.RawMessage        `json:"outputSchema"`
	AuthMethod     string                 `json:"authMethod"`
	ConnectionRef  string                 `json:"connectionRef"`
	Scopes         []string               `json:"scopes,omitempty"`
	TenantScope    string                 `json:"tenantScope,omitempty"`
	ResourceScope  string                 `json:"resourceScope,omitempty"`
	ReadOnly       bool                   `json:"readOnly,omitempty"`
	Risk           RiskClass              `json:"risk"`
	RateLimit      string                 `json:"rateLimit,omitempty"`
	Timeout        string                 `json:"timeout,omitempty"`
	Retry          RetryPolicy            `json:"retry,omitempty"`
	Idempotency    IdempotencyRequirement `json:"idempotency,omitempty"`
	ErrorContract  string                 `json:"errorContract,omitempty"`
	Audit          AuditRequirement       `json:"audit,omitempty"`
	Fixtures       []string               `json:"fixtures,omitempty"`
	Compensation   string                 `json:"compensation,omitempty"`
	Reconciliation string                 `json:"reconciliation,omitempty"`
	Attribution    AttributionMode        `json:"attribution,omitempty"`
	// Origin is the operator-approved custom-REST destination
	// (https://host[:port], no path). Tenant-supplied URLs are never
	// fetched directly; unset means provider-default routing.
	Origin           string `json:"origin,omitempty"`
	OriginApprovedBy string `json:"originApprovedBy,omitempty"`
	MaxResponseBytes int64  `json:"maxResponseBytes,omitempty"`
}

// Validate checks the capability spec structurally.
func (c CapabilitySpec) Validate() error {
	var m MultiError
	if err := Name(c.Provider).Validate("provider"); err != nil {
		m.Add(err.(*Error))
	}
	if err := Name(c.Operation).Validate("operation"); err != nil {
		m.Add(err.(*Error))
	}
	if c.APIVersion == "" {
		m.Add(Terminal("apiVersion", "must pin the provider API version"))
	}
	if !validJSON(c.InputSchema) {
		m.Add(Terminal("inputSchema", "must be a JSON object schema"))
	}
	if !validJSON(c.OutputSchema) {
		m.Add(Terminal("outputSchema", "must be a JSON object schema"))
	}
	if c.AuthMethod == "" {
		m.Add(Terminal("authMethod", "must not be empty"))
	}
	if c.ConnectionRef == "" {
		m.Add(Terminal("connectionRef", "must reference a brokered connection, never inline credentials"))
	}
	switch c.Risk {
	case RiskRead, RiskReversible, RiskConsequential:
	default:
		m.Add(Terminal("risk", "must be read, reversible-write, or consequential-write"))
	}
	if c.ReadOnly && c.Risk != RiskRead {
		m.Add(Terminal("risk", "readOnly capabilities must use risk read"))
	}
	if !c.ReadOnly && !c.Idempotency.Required {
		m.Add(Terminal("idempotency", "write capabilities must require idempotency"))
	}
	switch c.Attribution {
	case "", AttributionUser, AttributionService:
	default:
		m.Add(Terminal("attribution", "must be delegated-user or service-identity"))
	}
	if c.Origin != "" {
		u, err := url.Parse(c.Origin)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" || u.RawQuery != "" {
			m.Add(Terminal("origin", "must be an operator-approved https://host[:port] with no path"))
		}
		if c.OriginApprovedBy == "" {
			m.Add(Terminal("originApprovedBy", "custom origins require a named operator approver"))
		}
	}
	if c.MaxResponseBytes < 0 {
		m.Add(Terminal("maxResponseBytes", "must not be negative"))
	}
	return m.OrNil()
}
