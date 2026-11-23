// Package audit owns durable, attributable audit records. Every
// security-relevant transition in the runtime appends an event; there
// is no update or delete API. Structured log lines accompany durable
// writes; external trace export arrives in a later phase.
package audit
