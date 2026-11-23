// Package dbos is a minimal API-shape stub of the pinned DBOS Go SDK
// for analyzer fixtures. analysistest loads fixture packages in
// GOPATH mode, where the real module is unreachable; the stub keeps
// the same import path and call shapes so boundary recognition
// behaves identically. Behavioral proof comes from running the built
// analyzer over the repository's real Class D code.
package dbos

import (
	"context"
	"time"
)

// Context is the durable execution context.
type Context interface{}

// Workflow is a durable workflow function.
type Workflow[P, R any] func(Context, P) (R, error)

// Step is a durable step function.
type Step[R any] func(context.Context) (R, error)

// StepOutcome is a durable task result.
type StepOutcome[R any] struct {
	Value R
}

// Tx is a step transaction.
type Tx interface{}

// Txn is a durable transaction function.
type Txn[R any] func(context.Context, Tx) (R, error)

// DataSource is an application datasource handle.
type DataSource struct{}

// RegisterWorkflow registers fn as a durable workflow.
func RegisterWorkflow[P, R any](ctx Context, fn Workflow[P, R]) {}

// RunAsStep runs fn as a durable step.
func RunAsStep[R any](ctx Context, fn Step[R]) (R, error) {
	var zero R
	return zero, nil
}

// RunAsTransaction runs fn as a durable transaction step.
func RunAsTransaction[R any](ctx Context, ds *DataSource, fn Txn[R]) (R, error) {
	var zero R
	return zero, nil
}

// Go runs fn as a durable task.
func Go[R any](ctx Context, fn Step[R]) (<-chan StepOutcome[R], error) {
	return nil, nil
}

// Select waits for durable task results.
func Select[R any](ctx Context, chs []<-chan StepOutcome[R]) (R, error) {
	var zero R
	return zero, nil
}

// Sleep sleeps durably.
func Sleep(ctx Context, d time.Duration) (time.Duration, error) {
	return 0, nil
}

// Send delivers a message to a workflow.
func Send[P any](ctx Context, dest string, msg P, topic string) error {
	return nil
}

// Recv awaits a message on a topic.
func Recv[R any](ctx Context, topic string, timeout time.Duration) (R, error) {
	var zero R
	return zero, nil
}
