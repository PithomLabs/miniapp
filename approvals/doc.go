// Package approvals owns durable approval requests and decisions.
// Every consequential action gated by policy pauses here until an
// authorized principal decides; decisions bind to the exact requested
// operation so material changes invalidate outstanding approvals.
package approvals
