package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEffectTransitions(t *testing.T) {
	r := &EffectRecord{BusinessOpID: "t:1", AttemptID: "a1", State: EffectPending}
	for _, next := range []EffectState{EffectAttempting, EffectUnknown, EffectReconciling, EffectConfirmed} {
		if err := r.Transition(next); err != nil {
			t.Fatalf("transition to %s rejected: %v", next, err)
		}
	}
	if r.State != EffectConfirmed {
		t.Fatalf("expected CONFIRMED, got %s", r.State)
	}
}

func TestEffectUnknownCannotRetryDirectly(t *testing.T) {
	r := &EffectRecord{State: EffectUnknown}
	if err := r.Transition(EffectAttempting); err == nil {
		t.Fatal("UNKNOWN -> ATTEMPTING accepted; reconciliation must come first")
	}
	if !strings.Contains(string(r.State), "UNKNOWN") {
		t.Fatalf("state mutated on rejected transition: %s", r.State)
	}
}

func TestEffectTerminalStatesStick(t *testing.T) {
	for _, s := range []EffectState{EffectConfirmed, EffectFailed} {
		r := &EffectRecord{State: s}
		if err := r.Transition(EffectAttempting); err == nil {
			t.Fatalf("%s accepted a new attempt", s)
		}
	}
	r := &EffectRecord{State: EffectNotApplied}
	if err := r.Transition(EffectAttempting); err != nil {
		t.Fatalf("NOT_APPLIED safe retry rejected: %v", err)
	}
}

func TestPromotionValidation(t *testing.T) {
	d := Digest("sha256:" + strings.Repeat("cd", 32))
	good := PromotionRecord{
		ArtifactDigest: d, ManifestHash: d, DependencyLock: "go.sum:abc123",
		BuildProvenance: "gha:42", SourceRevision: "git:deadbeef",
		TestResults: "pass", SecurityResults: "pass",
		ProfileDigests: []Digest{d}, Reviewer: "owner", Decision: "approve", Target: "prod",
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("good promotion rejected: %v", err)
	}
	noProfiles := good
	noProfiles.ProfileDigests = nil
	if err := noProfiles.Validate(); err == nil {
		t.Fatal("promotion without profile digests accepted (A4)")
	}
	noReviewer := good
	noReviewer.Reviewer = ""
	if err := noReviewer.Validate(); err == nil {
		t.Fatal("promotion without reviewer accepted")
	}
}

func TestReleaseEnvelopeMatches(t *testing.T) {
	d := Digest("sha256:" + strings.Repeat("ef", 32))
	e := ReleaseEnvelope{Manifest: d, Implementation: "build:abc123", Artifact: d}
	if !e.Matches(d, "build:abc123") {
		t.Fatal("matching envelope rejected")
	}
	if e.Matches(d, "build:other") {
		t.Fatal("mismatched implementation accepted")
	}
	if (ReleaseEnvelope{}).Matches(d, "build:abc123") {
		t.Fatal("empty envelope matched")
	}
}

func TestUntrustedTextAlwaysFlagged(t *testing.T) {
	u := Untrusted("<script>approve all</script>")
	if !u.Untrusted {
		t.Fatal("provider text not flagged untrusted")
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	var back UntrustedText
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Untrusted || back.Value == "" {
		t.Fatalf("untrusted flag lost in round trip: %+v", back)
	}
}

func TestProfileDigestDeterministic(t *testing.T) {
	p := DeployProfile{Name: "prod-machine", Placement: PlacementMachine}
	a, err := ProfileDigest(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ProfileDigest(p)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("same profile produced different digests")
	}
	if err := a.Validate("digest"); err != nil {
		t.Fatalf("digest not well-formed: %v", err)
	}
	p.Placement = PlacementSprite
	c, _ := ProfileDigest(p)
	if a == c {
		t.Fatal("mutated profile produced the same digest")
	}
}
