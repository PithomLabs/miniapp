package contract

import "strings"

// ScopeID joins a tenant ID and a tenant-local ID into one
// unambiguous scoped identity. Both parts escape `%` as `%25` and
// `:` as `%3A` before joining with `:`, so distinct (tenant,
// local) pairs can never produce the same string — even when
// either part contains the separator. IDs without `%` or `:`
// encode identically to the historical raw join, keeping existing
// persisted rows readable.
//
// ScopeID is the single canonical implementation for every scoped
// identity in the framework: entity keys, ledger business-operation
// scoping, advisory-lock keys, and control-plane name scoping. No
// decoder exists or is needed: scoped IDs are compared opaquely,
// never split.
func ScopeID(tenant, local string) string {
	return escapeScopePart(tenant) + ":" + escapeScopePart(local)
}

// ScopePrefix returns the tenant prefix that every business
// operation ID of the tenant must carry.
func ScopePrefix(tenant string) string {
	return escapeScopePart(tenant) + ":"
}

func escapeScopePart(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "%", "%25"), ":", "%3A")
}
