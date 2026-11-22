// Package contract defines the frozen public contracts of the miniapp
// framework: manifests, capability specs, identity and policy shapes,
// run envelopes, effect records, task and promotion records, and
// deployment profiles.
//
// These types are the Phase 0 contract freeze. Go structs are the source
// of truth; JSON Schemas under schemas/ are generated from them (see
// schema.go) and drift tests fail the build on any mismatch. Behavioral
// enforcement lives in the owning packages (registry, runtime, auth,
// effects, deploy); this package owns shapes and structural validation.
package contract
