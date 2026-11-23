// Package faultwrap adapts production interfaces to fault
// injection for tests: each wrapper consults its Injector
// immediately before delegating to the wrapped implementation,
// so a firing fault is raised at the claimed operation boundary.
// Wrappers live apart from testkit/fault (which stays
// dependency-free) to avoid import cycles with the services.
package faultwrap

import (
	"context"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

// Operation names for interface boundaries.
const (
	OpAsyncSubmit  = "async-submit"
	OpAsyncStatus  = "async-status"
	OpAsyncSignal  = "async-signal"
	OpAsyncCancel  = "async-cancel"
	OpProviderSend = "provider-send"
	OpPolicyCheck  = "policy-check"
)

// Async wraps an executor.AsyncExecutor with fault injection.
type Async struct {
	Inner executor.AsyncExecutor
	Inj   *fault.Injector
}

func (a Async) Submit(ctx context.Context, req executor.Request) (contract.RunHandle, error) {
	if err := a.Inj.Fire(OpAsyncSubmit); err != nil {
		return contract.RunHandle{}, err
	}
	return a.Inner.Submit(ctx, req)
}

func (a Async) Status(ctx context.Context, runID string) (contract.RunStatus, error) {
	if err := a.Inj.Fire(OpAsyncStatus); err != nil {
		return "", err
	}
	return a.Inner.Status(ctx, runID)
}

func (a Async) Signal(ctx context.Context, runID string, sig contract.Signal) error {
	if err := a.Inj.Fire(OpAsyncSignal); err != nil {
		return err
	}
	return a.Inner.Signal(ctx, runID, sig)
}

func (a Async) Cancel(ctx context.Context, runID string, reason string) error {
	if err := a.Inj.Fire(OpAsyncCancel); err != nil {
		return err
	}
	return a.Inner.Cancel(ctx, runID, reason)
}

// Channel wraps an effects.Channel with fault injection.
type Channel struct {
	Inner effects.Channel
	Inj   *fault.Injector
}

func (c Channel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	if err := c.Inj.Fire(OpProviderSend); err != nil {
		return effects.Receipt{}, err
	}
	return c.Inner.Send(ctx, op)
}

// Policy wraps an effects.PolicyChecker with fault injection.
type Policy struct {
	Inner effects.PolicyChecker
	Inj   *fault.Injector
}

func (p Policy) CheckPolicy(ctx context.Context, tenantID string, intent effects.Intent) error {
	if err := p.Inj.Fire(OpPolicyCheck); err != nil {
		return err
	}
	return p.Inner.CheckPolicy(ctx, tenantID, intent)
}
