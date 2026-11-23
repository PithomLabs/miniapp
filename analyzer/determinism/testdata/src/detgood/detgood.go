package detgood

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
)

// okConst is read-only package state, which the analyzer permits.
const okConst = "fixed"

// goodDS models the established datasource-handle pattern: set once
// at setup, read (never written) in workflow bodies.
var goodDS *dbos.DataSource

func registerGood(ctx dbos.Context) {
	dbos.RegisterWorkflow(ctx, goodWorkflow)
	dbos.RegisterWorkflow(ctx, goodTxn)
}

func goodTxn(ctx dbos.Context, in string) (string, error) {
	return dbos.RunAsTransaction(ctx, goodDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		_ = time.Now()
		_ = tx
		return "tx-" + in, nil
	})
}

func pureHelper(in string) string {
	if in == "" {
		return okConst
	}
	return in + okConst
}

func goodWorkflow(ctx dbos.Context, in string) (string, error) {
	v := pureHelper(in)
	if _, err := dbos.Sleep(ctx, time.Millisecond); err != nil {
		return "", err
	}
	ch, err := dbos.Go(ctx, func(ctx context.Context) (string, error) {
		now := time.Now()
		_ = now
		return "stepped", nil
	})
	if err != nil {
		return "", err
	}
	out, err := dbos.Select(ctx, []<-chan dbos.StepOutcome[string]{ch})
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", err
		}
		return "", err
	}
	m := map[string]int{"a": 1}
	return out + v + string(rune(m["a"])), nil
}
