package fault_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp/testkit/fault"
)

var errBoom = errors.New("boom")

// TestExactNthOneShot: FailAt fires exactly once, on the Nth
// invocation, and never again.
func TestExactNthOneShot(t *testing.T) {
	inj := fault.New()
	inj.FailAt("op", 3, errBoom)
	var got []bool
	for i := 0; i < 5; i++ {
		got = append(got, inj.Fire("op") != nil)
	}
	want := []bool{false, false, true, false, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d fired = %v, want %v (full %v)", i+1, got[i], want[i], got)
		}
	}
	if n := inj.Count("op"); n != 5 {
		t.Fatalf("count = %d, want 5", n)
	}
}

// TestPersistentFromNth: FailFrom fires on the Nth and every later
// invocation.
func TestPersistentFromNth(t *testing.T) {
	inj := fault.New()
	inj.FailFrom("op", 2, errBoom)
	var got []bool
	for i := 0; i < 4; i++ {
		got = append(got, inj.Fire("op") != nil)
	}
	want := []bool{false, true, true, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d fired = %v, want %v (full %v)", i+1, got[i], want[i], got)
		}
	}
}

// TestIndependentInstances: separate injectors never share counters
// or rules, even for the same operation name.
func TestIndependentInstances(t *testing.T) {
	a, b := fault.New(), fault.New()
	a.FailAt("op", 1, errBoom)
	if err := a.Fire("op"); !errors.Is(err, errBoom) {
		t.Fatalf("a.Fire = %v, want boom", err)
	}
	if err := b.Fire("op"); err != nil {
		t.Fatalf("b.Fire = %v, want nil (independent)", err)
	}
	if n := b.Count("op"); n != 1 {
		t.Fatalf("b.Count = %d, want 1", n)
	}
	if n := a.Count("op"); n != 1 {
		t.Fatalf("a.Count = %d, want 1", n)
	}
}

// TestUnconfiguredNeverFires: unknown operations record counts but
// return nil.
func TestUnconfiguredNeverFires(t *testing.T) {
	inj := fault.New()
	for i := 0; i < 3; i++ {
		if err := inj.Fire("ghost"); err != nil {
			t.Fatalf("Fire(ghost) = %v, want nil", err)
		}
	}
	if n := inj.Count("ghost"); n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
}

// TestNilInjectorIsNoOp: a nil injector is safe to consult and
// never fires (the production state).
func TestNilInjectorIsNoOp(t *testing.T) {
	var inj *fault.Injector
	if err := inj.Fire("op"); err != nil {
		t.Fatalf("nil Fire = %v, want nil", err)
	}
	if n := inj.Count("op"); n != 0 {
		t.Fatalf("nil Count = %d, want 0", n)
	}
}

// TestFireReturnsExactError: propagation preserves error identity
// (callers match with errors.Is on sentinels).
func TestFireReturnsExactError(t *testing.T) {
	inj := fault.New()
	inj.FailAt("op", 1, errBoom)
	if err := inj.Fire("op"); err != errBoom {
		t.Fatalf("Fire = %v, want the exact configured error", err)
	}
}

// TestBadConfigPanics: invalid rules fail fast at configuration,
// never silently at fire time.
func TestBadConfigPanics(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func()
	}{
		{"nth-zero", func() { fault.New().FailAt("op", 0, errBoom) }},
		{"empty-op", func() { fault.New().FailFrom("", 1, errBoom) }},
		{"nil-err", func() { fault.New().FailAt("op", 1, nil) }},
		{"nil-injector", func() { var i *fault.Injector; i.FailAt("op", 1, errBoom) }},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s: no panic", tc.name)
				}
			}()
			tc.fn()
		}()
	}
}

// TestConcurrentUse: hammering from parallel goroutines keeps exact
// counts (run under -race in CI); per-test injectors stay isolated.
func TestConcurrentUse(t *testing.T) {
	const workers = 8
	const perWorker = 50
	shared := fault.New()
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				_ = shared.Fire("shared")
			}
		}()
	}
	wg.Wait()
	if n := shared.Count("shared"); n != workers*perWorker {
		t.Fatalf("shared count = %d, want %d", n, workers*perWorker)
	}

	isolated := make([]*fault.Injector, workers)
	for w := range isolated {
		isolated[w] = fault.New()
		isolated[w].FailAt("op", perWorker, errBoom)
	}
	var mu sync.Mutex
	fired := make([]int, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			n := 0
			for i := 0; i < perWorker; i++ {
				if isolated[w].Fire("op") != nil {
					n++
				}
			}
			mu.Lock()
			fired[w] = n
			mu.Unlock()
		}(w)
	}
	wg.Wait()
	for w, n := range fired {
		if n != 1 {
			t.Fatalf("injector %d fired %d times, want exactly 1", w, n)
		}
	}
}
