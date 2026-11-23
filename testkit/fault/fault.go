// Package fault provides deterministic fault injection for
// lifecycle tests: named operation boundaries that fail on a
// configured invocation with a configured error.
//
// Semantics are exact: FailAt fires once, on the Nth invocation of
// the named operation; FailFrom fires on the Nth and every later
// invocation. Invocation counters increment on every Fire call for
// the operation, configured or not. Each Injector is independent —
// there is no package-global state — and safe for concurrent use.
//
// A nil *Injector is a valid no-op: Fire returns nil and Count
// returns 0. Production code consults its TestFault field
// unconditionally; with the field nil (the only production state),
// behavior is unchanged.
package fault

import (
	"fmt"
	"sync"
)

// Mode selects one-shot or persistent failure.
type Mode int

const (
	// Once fires only on the configured Nth invocation.
	Once Mode = iota
	// Persistent fires on the Nth and every later invocation.
	Persistent
)

// rule is one configured failure.
type rule struct {
	mode Mode
	nth  int
	err  error
}

// Injector injects configured failures at named boundaries.
// Construct with New; do not copy after use.
type Injector struct {
	mu     sync.Mutex
	counts map[string]int
	rules  map[string][]rule
}

// New returns an isolated injector with no configured failures.
func New() *Injector {
	return &Injector{counts: map[string]int{}, rules: map[string][]rule{}}
}

// FailAt configures op to fail with err exactly once, on its nth
// invocation (1-based). nth < 1 fails the configuration itself.
func (i *Injector) FailAt(op string, nth int, err error) {
	i.add(op, rule{mode: Once, nth: nth, err: err})
}

// FailFrom configures op to fail with err on its nth and every
// later invocation (1-based).
func (i *Injector) FailFrom(op string, nth int, err error) {
	i.add(op, rule{mode: Persistent, nth: nth, err: err})
}

func (i *Injector) add(op string, r rule) {
	if i == nil {
		panic("fault: configure a non-nil Injector")
	}
	if op == "" {
		panic("fault: operation name is required")
	}
	if r.nth < 1 {
		panic(fmt.Sprintf("fault: nth must be >= 1 for op %q", op))
	}
	if r.err == nil {
		panic(fmt.Sprintf("fault: error is required for op %q", op))
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.rules[op] = append(i.rules[op], r)
}

// Fire records one invocation of op and returns the injected error
// when a rule matches, or nil. Rules evaluate in configuration
// order; the first match wins. A nil injector never fires.
func (i *Injector) Fire(op string) error {
	if i == nil {
		return nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.counts[op]++
	n := i.counts[op]
	for idx, r := range i.rules[op] {
		switch r.mode {
		case Once:
			if n == r.nth {
				// Consume one-shot rules so they fire exactly once.
				i.rules[op] = append(i.rules[op][:idx], i.rules[op][idx+1:]...)
				return r.err
			}
		case Persistent:
			if n >= r.nth {
				return r.err
			}
		}
	}
	return nil
}

// Count reports how many invocations of op have been recorded.
// A nil injector reports 0.
func (i *Injector) Count(op string) int {
	if i == nil {
		return 0
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.counts[op]
}
