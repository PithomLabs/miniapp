// Package effects owns the external-effect protocol: intent ledger,
// attempt recording, outcome states, and quarantine. Attempts are
// at-least-once; confirmed effects are at-most-once per business
// operation; ambiguous outcomes quarantine as UNKNOWN and are never
// blindly retried. Reconciliation sweepers arrive in a later phase;
// this package quarantines unknown outcomes durably, exposes them
// for operators, and accepts explicit operator settlement instead of
// guessing.
package effects
