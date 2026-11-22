// Package store owns Postgres access shapes: pool configuration,
// transaction-boundary helpers, and tenant-context injection for
// row-level security. Live behavior is proven by Phase 0 spikes;
// this package holds only the frozen shapes and pure helpers.
package store
