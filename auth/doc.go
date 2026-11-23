// Package auth owns principal authentication and authorization
// enforcement. Phase 1 uses static credential bindings loaded from
// trusted configuration; the token broker and OAuth flows arrive in
// Phase 2. The authorization rules enforced here — two principals,
// tenant binding, trust ceilings, approval gating — are the same
// rules later phases keep.
package auth
