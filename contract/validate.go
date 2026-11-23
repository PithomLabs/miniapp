package contract

import "regexp"

var (
	// nameRe covers miniapp IDs, tool segments, capability names, and
	// profile names: lowercase ASCII, dot/underscore/dash separators.
	nameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
	// versionRe is strict semantic versioning without a "v" prefix, as
	// used by miniapp manifests ("1.4.2").
	versionRe = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	// digestRe pins content-addressed references to lowercase hex sha256.
	digestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	// toolNameRe allows dotted tool names such as
	// "rpa.crm.create_contact" as well as generic verbs.
	toolNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9_]*)+$`)
)

// MaxNameLen bounds every identifier validated by Name.Validate.
const MaxNameLen = 128

// Name is a validated manifest identifier.
type Name string

// Validate reports whether n is a well-formed identifier.
func (n Name) Validate(field string) error {
	if len(n) == 0 || len(n) > MaxNameLen {
		return Terminal(field, "must be 1-%d characters", MaxNameLen)
	}
	if !nameRe.MatchString(string(n)) {
		return Terminal(field, "must match %s", nameRe.String())
	}
	return nil
}

// Version is a validated semantic version without a "v" prefix.
type Version string

// Validate reports whether v is a well-formed version.
func (v Version) Validate(field string) error {
	if !versionRe.MatchString(string(v)) {
		return Terminal(field, "must be semantic version MAJOR.MINOR.PATCH")
	}
	return nil
}

// Digest is a validated content digest of the form "sha256:<hex>".
// Deployment-profile references must carry a digest (A4): an unbound
// name-only reference is rejected wherever digests are required.
type Digest string

// Validate reports whether d is a well-formed digest.
func (d Digest) Validate(field string) error {
	if !digestRe.MatchString(string(d)) {
		return Terminal(field, "must match sha256:<64 hex chars>")
	}
	return nil
}

// EntityKey declares how the framework derives a tool's serialization
// entity at dispatch: a dot-separated input field path whose string
// value is the tenant-local entity ID. The framework scopes the
// invoking tenant via contract.ScopeID (percent-escaping both parts
// before joining), so manifests never embed a tenant and distinct
// (tenant, local) pairs cannot collide even when either part
// contains the separator. An empty key means the tool declares no
// entity scope (allowed only with a reviewed NoEntityWhy
// justification on the manifest).
type EntityKey string

// Validate reports whether k is a well-formed entity-key path.
func (k EntityKey) Validate(field string) error {
	if len(k) == 0 || len(k) > MaxNameLen {
		return Terminal(field, "must be 1-%d characters", MaxNameLen)
	}
	if !nameRe.MatchString(string(k)) {
		return Terminal(field, "must be a lowercase input field path, e.g. lead_id")
	}
	return nil
}

// ToolName is a validated dotted tool name.
type ToolName string

// Validate reports whether t is a well-formed tool name.
func (t ToolName) Validate(field string) error {
	if len(t) == 0 || len(t) > MaxNameLen {
		return Terminal(field, "must be 1-%d characters", MaxNameLen)
	}
	if !toolNameRe.MatchString(string(t)) {
		return Terminal(field, "must be dotted lowercase, e.g. rpa.crm.create_contact")
	}
	return nil
}
