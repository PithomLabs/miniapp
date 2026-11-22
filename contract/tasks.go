package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// NetworkAction is allow or deny for one Sprite egress rule.
type NetworkAction string

const (
	NetworkAllow NetworkAction = "allow"
	NetworkDeny  NetworkAction = "deny"
)

// NetworkRule is one domain egress rule. Profiles ship default-deny
// templates; generic exfiltration-capable domains are rejected by
// policy review, not by this structural shape.
type NetworkRule struct {
	Domain  string        `json:"domain,omitempty"`
	Action  NetworkAction `json:"action"`
	Include string        `json:"include,omitempty"`
}

// ResourceLimits bounds one Sprite workspace or task.
type ResourceLimits struct {
	RAMMB     int `json:"ramMb,omitempty"`
	CPUs      int `json:"cpus,omitempty"`
	StorageGB int `json:"storageGb,omitempty"`
}

// SpritePolicy is the frozen per-profile Sprite policy: network rules,
// resources, and lifetime. Its digest is embedded in published
// artifacts (A4).
type SpritePolicy struct {
	Network   []NetworkRule  `json:"network"`
	Resources ResourceLimits `json:"resources,omitempty"`
	Lifetime  string         `json:"lifetime,omitempty"`
}

// TaskAuthority is proxy-issued scoped task identity. It carries TTL,
// minimal scope, and per-task budgets — never provider bearer tokens.
type TaskAuthority struct {
	TaskID    string    `json:"taskId"`
	Audience  string    `json:"audience"`
	ExpiresAt time.Time `json:"expiresAt"`
	Budget    string    `json:"budget,omitempty"`
}

// DeployProfile is the frozen named deployment profile.
type DeployProfile struct {
	Name      string          `json:"name"`
	Placement Placement       `json:"placement"`
	Sprite    *SpritePolicy   `json:"sprite,omitempty"`
	Promotion PromotionPolicy `json:"promotion,omitempty"`
}

// ProfileDigest returns the canonical content digest of a profile.
// Canonical form is encoding/json output of the struct value, which is
// deterministic (fixed field order, sorted map keys). The same input
// always yields the same digest; any content change yields a new one.
func ProfileDigest(v any) (Digest, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", Terminal("profile", "cannot marshal profile: %v", err)
	}
	sum := sha256.Sum256(data)
	return Digest("sha256:" + hex.EncodeToString(sum[:])), nil
}

// TaskState is the lifecycle state of one Sprite task registry row.
type TaskState string

const (
	TaskStaged        TaskState = "STAGED"
	TaskProvisioning  TaskState = "PROVISIONING"
	TaskPolicyApplied TaskState = "POLICY_APPLIED"
	TaskRunning       TaskState = "RUNNING"
	TaskSucceeded     TaskState = "SUCCEEDED"
	TaskFailed        TaskState = "FAILED"
	TaskTimedOut      TaskState = "TIMED_OUT"
	TaskCancelled     TaskState = "CANCELLED"
)

// CleanupState tracks workspace teardown separately from task outcome
// so the reaper can distinguish "result recorded" from "compute freed".
type CleanupState string

const (
	CleanupPending CleanupState = "PENDING"
	CleanupDone    CleanupState = "DONE"
	CleanupLost    CleanupState = "LOST"
)

// TaskRecord is the durable Neon task-registry row (A2). Provision and
// reconcile operations are idempotent on TaskID.
type TaskRecord struct {
	TaskID         string       `json:"taskId"`
	ParentRunID    string       `json:"parentRunId"`
	WorkflowID     string       `json:"workflowId,omitempty"`
	SpriteID       string       `json:"spriteId,omitempty"`
	State          TaskState    `json:"state"`
	PolicyDigest   Digest       `json:"policyDigest,omitempty"`
	LeaseExpiresAt time.Time    `json:"leaseExpiresAt,omitempty"`
	FencingToken   string       `json:"fencingToken,omitempty"`
	ResultRef      string       `json:"resultRef,omitempty"`
	Cleanup        CleanupState `json:"cleanup"`
	LastError      string       `json:"lastError,omitempty"`
	UpdatedAt      time.Time    `json:"updatedAt,omitempty"`
}

// ReleaseEnvelope binds a manifest digest to the exact implementation
// and build that serve it, without circularity: the envelope lives
// outside the self-hashed manifest payload. The registry refuses an
// implementation whose envelope does not match the manifest's declared
// implementation identity (F18).
type ReleaseEnvelope struct {
	Manifest       Digest `json:"manifest"`
	Implementation string `json:"implementation"`
	Build          string `json:"build"`
	SourceRevision string `json:"sourceRevision"`
	Artifact       Digest `json:"artifact"`
}

// Matches reports whether the envelope binds the given manifest digest
// to the given implementation identity. Empty fields never match.
func (e ReleaseEnvelope) Matches(manifest Digest, implementation string) bool {
	if e.Manifest == "" || e.Implementation == "" {
		return false
	}
	return e.Manifest == manifest && e.Implementation == implementation
}

// PromotionRecord binds an immutable release artifact to its review.
// The artifact is a trusted-build OCI image (or equivalent); a Sprite
// snapshot ID may appear only as workspace provenance, never as the
// promoted artifact (A6).
type PromotionRecord struct {
	ArtifactDigest  Digest    `json:"artifactDigest"`
	ManifestHash    Digest    `json:"manifestHash"`
	DependencyLock  string    `json:"dependencyLock"`
	SBOMRef         string    `json:"sbomRef,omitempty"`
	BuildProvenance string    `json:"buildProvenance"`
	SourceRevision  string    `json:"sourceRevision"`
	TestResults     string    `json:"testResults"`
	SecurityResults string    `json:"securityResults"`
	ProfileDigests  []Digest  `json:"profileDigests"`
	Reviewer        string    `json:"reviewer"`
	Decision        string    `json:"decision"`
	Target          string    `json:"target"`
	DecidedAt       time.Time `json:"decidedAt,omitempty"`
}

// Validate checks the promotion record structurally, including that
// profile digests are bound (A4).
func (p PromotionRecord) Validate() error {
	var m MultiError
	if err := p.ArtifactDigest.Validate("artifactDigest"); err != nil {
		m.Add(err.(*Error))
	}
	if err := p.ManifestHash.Validate("manifestHash"); err != nil {
		m.Add(err.(*Error))
	}
	if p.DependencyLock == "" {
		m.Add(Terminal("dependencyLock", "must not be empty"))
	}
	if p.BuildProvenance == "" {
		m.Add(Terminal("buildProvenance", "must identify the trusted build"))
	}
	if p.SourceRevision == "" {
		m.Add(Terminal("sourceRevision", "must identify reviewed source"))
	}
	if len(p.ProfileDigests) == 0 {
		m.Add(Terminal("profileDigests", "must bind at least one profile digest"))
	}
	for _, d := range p.ProfileDigests {
		if err := d.Validate("profileDigests"); err != nil {
			m.Add(err.(*Error))
		}
	}
	if p.Reviewer == "" {
		m.Add(Terminal("reviewer", "must name the approver"))
	}
	if p.Decision == "" {
		m.Add(Terminal("decision", "must record the release decision"))
	}
	if p.Target == "" {
		m.Add(Terminal("target", "must name the deployment target"))
	}
	return m.OrNil()
}
