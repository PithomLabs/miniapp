// Pinned-snapshot semantics: hash stability/content-sensitivity,
// setter ordering (fully precedes-or-follows a pinned read), and
// race-clean concurrent access.
package auth

import (
	"strings"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
)

func snapshotGrants() map[string]map[string]contract.TrustMode {
	return map[string]map[string]contract.TrustMode{
		"t1": {"m1": contract.TrustAutonomous},
	}
}

func TestP23ConfigHashStableAndSensitive(t *testing.T) {
	p := NewPolicy(map[string]Binding{}, snapshotGrants())
	h1 := p.ConfigHash()
	h2 := p.ConfigHash()
	if h1 == "" || h1 != h2 {
		t.Fatalf("hash unstable or empty: %q vs %q", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") || len(h1) != len("sha256:")+64 {
		t.Fatalf("hash %q is not a sha256 hex digest", h1)
	}
	// Same grants, different keys: identical hash (credentials
	// never enter the digest).
	q := NewPolicy(map[string]Binding{"k": {AgentID: "a"}}, snapshotGrants())
	if q.ConfigHash() != h1 {
		t.Fatal("hash depends on keys; must cover grants only")
	}
	// Any grant change flips the hash.
	p.SetGrants(map[string]map[string]contract.TrustMode{
		"t1": {"m1": contract.TrustConfirm},
	})
	if p.ConfigHash() == h1 {
		t.Fatal("hash unchanged after grant change")
	}
	if !p.GrantAllows("t1", "m1", contract.TrustConfirm) || p.GrantAllows("t1", "m1", contract.TrustAutonomous) {
		t.Fatal("reads do not observe the swapped snapshot")
	}
}

// TestP23SnapshotPinning proves setter/revocation ordering without
// timing: while RLock is held, the observable snapshot cannot
// change (the setter fully follows); after release it lands
// atomically (a coherent map, never a torn one).
func TestP23SnapshotPinning(t *testing.T) {
	p := NewPolicy(nil, snapshotGrants())
	h1 := p.ConfigHash()
	p.RLock()
	done := make(chan struct{})
	go func() {
		p.SetGrants(map[string]map[string]contract.TrustMode{
			"t1": {"m1": contract.TrustConfirm},
		})
		close(done)
	}()
	// Still under OUR read lock: the setter cannot have landed,
	// so every observation must match the pinned snapshot. No
	// sleep: any interleaving satisfies this or the mutex is
	// broken (a hang fails loudly via the test timeout). Only
	// non-locking reads here: self-locking accessors would
	// deadlock against the pending writer (the exact hazard the
	// Locked variants exist to avoid).
	for i := 0; i < 100; i++ {
		if !p.GrantAllowsLocked("t1", "m1", contract.TrustAutonomous) {
			p.RUnlock()
			t.Fatal("pinned read lost autonomous grant under RLock")
		}
	}
	select {
	case <-done:
		p.RUnlock()
		t.Fatal("setter completed while RLock held")
	default:
	}
	p.RUnlock()
	<-done
	if p.ConfigHash() == h1 {
		t.Fatal("setter never landed after release")
	}
	if p.GrantAllows("t1", "m1", contract.TrustAutonomous) || !p.GrantAllows("t1", "m1", contract.TrustConfirm) {
		t.Fatal("post-set reads do not observe the swapped snapshot")
	}
}

// TestP23ConcurrentSnapshotAccess exercises readers against a
// setter for race-cleanliness (run with -race).
func TestP23ConcurrentSnapshotAccess(t *testing.T) {
	p := NewPolicy(nil, snapshotGrants())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = p.GrantAllows("t1", "m1", contract.TrustConfirm)
				_ = p.ConfigHash()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 20; j++ {
			p.SetGrants(snapshotGrants())
		}
	}()
	wg.Wait()
}

// TestP28ConfigHashOrderInvariant proves map-order invariance
// deterministically: rotated insertion orders across separate
// policies plus repeated reads (Go map iteration is random per
// read) must all agree bit-for-bit.
func TestP28ConfigHashOrderInvariant(t *testing.T) {
	build := func(order []string) map[string]map[string]contract.TrustMode {
		g := map[string]map[string]contract.TrustMode{}
		for _, tenant := range order {
			g[tenant] = map[string]contract.TrustMode{}
		}
		modes := map[string]contract.TrustMode{"a": contract.TrustAutonomous, "b": contract.TrustConfirm, "c": contract.TrustDryRun}
		for _, tenant := range order {
			for m, mode := range modes {
				g[tenant]["m-"+m] = mode
			}
		}
		return g
	}
	want := NewPolicy(nil, build([]string{"t1", "t2", "t3"})).ConfigHash()
	rots := [][]string{{"t2", "t3", "t1"}, {"t3", "t1", "t2"}, {"t1", "t3", "t2"}}
	for _, r := range rots {
		if got := NewPolicy(nil, build(r)).ConfigHash(); got != want {
			t.Fatalf("rotated-insertion hash = %q, want %q", got, want)
		}
	}
	p := NewPolicy(nil, build([]string{"t1", "t2", "t3"}))
	for i := 0; i < 100; i++ {
		if got := p.ConfigHash(); got != want {
			t.Fatalf("read %d hash = %q, want %q (iteration order leaked)", i, got, want)
		}
	}
}
