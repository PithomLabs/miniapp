// Package runtime is the canonical invocation service: the single
// governed execution path for every transport. It owns request
// validation, authentication, authorization-before-dispatch,
// idempotency, execution-class dispatch, run handles, approvals
// orchestration, and audit. Transports translate; they never decide.
package runtime
