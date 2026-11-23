// Package preflight validates architectural-contract consistency
// between registered miniapps, ADR-001 decision records, and the
// evidence register. It checks structured facts only — presence,
// vocabulary, references, and cross-file agreement — never business
// judgment, which stays with the named human reviewer.
package preflight
