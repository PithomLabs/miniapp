package detbad

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
)

var badCounter int

func registerBad(ctx dbos.Context) {
	dbos.RegisterWorkflow(ctx, badGo)
	dbos.RegisterWorkflow(ctx, badSelect)
	dbos.RegisterWorkflow(ctx, badMap)
	dbos.RegisterWorkflow(ctx, badTime)
	dbos.RegisterWorkflow(ctx, badRand)
	dbos.RegisterWorkflow(ctx, badGlobal)
	dbos.RegisterWorkflow(ctx, badGlobalRead)
	dbos.RegisterWorkflow(ctx, badHTTP)
	dbos.RegisterWorkflow(ctx, badEnv)
	dbos.RegisterWorkflow(ctx, badStepSend)
	dbos.RegisterWorkflow(ctx, badStepRecv)
	dbos.RegisterWorkflow(ctx, badHelperCaller)
}

func badGo(ctx dbos.Context, in string) (string, error) {
	go func() {}() // want "go statements are forbidden"
	_ = ctx
	return in, nil
}

func badSelect(ctx dbos.Context, in string) (string, error) {
	ch := make(chan int, 1)
	ch <- 1
	select { // want "select on channels is forbidden"
	case <-ch:
	}
	_ = ctx
	return in, nil
}

func badMap(ctx dbos.Context, in string) (string, error) {
	m := map[string]int{"a": 1}
	for k := range m { // want "depends on iteration order"
		_ = k
	}
	_ = ctx
	return in, nil
}

func badTime(ctx dbos.Context, in string) (string, error) {
	_ = time.Now() // want "is nondeterministic"
	_ = ctx
	return in, nil
}

func badRand(ctx dbos.Context, in string) (string, error) {
	_ = rand.Int() // want "is nondeterministic"
	_ = ctx
	return in, nil
}

func badGlobal(ctx dbos.Context, in string) (string, error) {
	badCounter++ // want "must not be read or written"
	_ = ctx
	return in, nil
}

func badGlobalRead(ctx dbos.Context, in string) (string, error) {
	_ = badCounter // want "must not be read or written"
	_ = ctx
	return in, nil
}

func badHTTP(ctx dbos.Context, in string) (string, error) {
	_, _ = http.Get("http://example.invalid") // want "is direct I/O"
	_ = ctx
	return in, nil
}

func badEnv(ctx dbos.Context, in string) (string, error) {
	_ = os.Getenv("DET_BAD") // want "is direct I/O"
	_ = ctx
	return in, nil
}

func badStepSend(wctx dbos.Context, in string) (string, error) {
	return dbos.RunAsStep(wctx, func(ctx context.Context) (string, error) {
		err := dbos.Send(wctx, "wf", "m", "t") // want "must not be called inside a step body"
		_ = ctx
		return "", err
	})
}

func badStepRecv(wctx dbos.Context, in string) (string, error) {
	return dbos.RunAsStep(wctx, func(ctx context.Context) (string, error) {
		v, err := dbos.Recv[string](wctx, "t", time.Second) // want "must not be called inside a step body"
		_ = ctx
		return v, err
	})
}

func nowHelper() time.Time {
	return time.Now() // want "is nondeterministic"
}

func badHelperCaller(ctx dbos.Context, in string) (string, error) {
	_ = nowHelper()
	_ = ctx
	return in, nil
}

// unregistered is never passed to RegisterWorkflow, so its
// violations must stay silent: the analyzer only governs workflow
// bodies.
func unregistered() {
	_ = time.Now()
}
