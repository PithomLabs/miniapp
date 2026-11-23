// Package testkit holds reusable framework test infrastructure:
// migrated test databases, fixture helpers, and assertion support.
// Production code must never import testkit.
//
// Methodology (O-05): test packages sharing one database must run
// serially (`go test -p 1 ./...`), unless each package process has
// an isolated database — packages truncate shared tables, so
// parallel packages contend and fail or hang. This documents the
// supported invocation; it does not change harness semantics.
package testkit
