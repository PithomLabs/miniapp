// Package executor defines the execution-class dispatch boundary.
// Class A tools run synchronously inside the request transaction;
// Class D tools submit durable workflows and return run handles.
// River-backed classes arrive in a later phase; no executor here
// pretends to cover them.
package executor
