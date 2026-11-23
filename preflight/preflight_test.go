package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

const approvedADR = `# ADR-001 decision — widget v1.0.0

Status: APPROVED.

- Business promise: widgets processed.
- Score (0/1/2): lifetime 2  stages 2  waiting 2  recovery 2  effects 1  coordination 1  human 1  = 11/14
- Gate 1 durable need: multi-day waits.
- Gate 2 characteristics (need ≥2): waiting, recovery.
- Gate 3 failure consequence: lost widgets.
- Decision: ADOPT
- Reviewer (human): Some Owner   Date: 2026-11-23
`

const draftADR = `# ADR-001 decision — widget v1.0.0

Status: **DRAFT — NOT APPROVED.**

- Business promise: widgets processed.
- Score (0/1/2): lifetime _  stages _  waiting _  recovery _  effects _  coordination _  human _  = __/14
- Gate 1 durable need: tbd.
- Gate 2 characteristics (need ≥2): tbd.
- Gate 3 failure consequence: tbd.
- Decision: PROPOSED ADOPT
- Reviewer (human): <awaiting owner>   Date: <pending>
`

const evidenceWithWidget = `# Evidence

| Gate | Claim | Artifact | Test | Revision | Environment | Command | Result | Date |
| P1 | widget works | console | TestWidget | abc | local | go test | PASS 2026-11-23 | 2026-11-23 |
`

func fixture(t *testing.T, adr, evidence string) Runner {
	t.Helper()
	dir := t.TempDir()
	adrDir := filepath.Join(dir, "adr")
	writeFile(t, filepath.Join(adrDir, "widget-adr001.md"), adr)
	evPath := filepath.Join(dir, "evidence.md")
	writeFile(t, evPath, evidence)
	return Runner{ADRDir: adrDir, Evidence: evPath,
		Miniapps: map[string]string{"widget": "1.0.0"},
		IsClassD: map[string]bool{"widget": true}}
}

func verdicts(findings []Finding) map[Verdict]int {
	out := map[Verdict]int{}
	for _, f := range findings {
		out[f.Verdict]++
	}
	return out
}

func TestApprovedPasses(t *testing.T) {
	r := fixture(t, approvedADR, evidenceWithWidget)
	v := verdicts(r.RunOne("widget"))
	if v[Fail] != 0 {
		t.Fatalf("failures: %+v", r.RunOne("widget"))
	}
}

func TestDraftRegisteredFails(t *testing.T) {
	r := fixture(t, draftADR, evidenceWithWidget)
	found := false
	for _, f := range r.RunOne("widget") {
		if f.Verdict == Fail && f.Check == "adr-state" {
			found = true
		}
	}
	if !found {
		t.Fatal("draft with registered implementation not blocked")
	}
}

func TestDraftUnregisteredSkips(t *testing.T) {
	r := fixture(t, draftADR, evidenceWithWidget)
	r.Miniapps = map[string]string{}
	r.IsClassD = map[string]bool{}
	found := false
	for _, f := range r.RunAll() {
		if f.Verdict == Skip {
			found = true
		}
		if f.Verdict == Fail {
			t.Fatalf("unexpected failure: %+v", f)
		}
	}
	if !found {
		t.Fatal("pre-implementation draft not acknowledged with SKIP")
	}
}

func TestApprovedOrphanFails(t *testing.T) {
	r := fixture(t, approvedADR, evidenceWithWidget)
	r.Miniapps = map[string]string{}
	r.IsClassD = map[string]bool{}
	found := false
	for _, f := range r.RunAll() {
		if f.Verdict == Fail && f.Check == "adr-orphan" {
			found = true
		}
	}
	if !found {
		t.Fatal("approved decision without implementation not flagged")
	}
}

func TestScopeMismatchFails(t *testing.T) {
	adr := strings.Replace(approvedADR, "widget v1.0.0", "widget v9.9.9", 1)
	r := fixture(t, adr, evidenceWithWidget)
	found := false
	for _, f := range r.RunOne("widget") {
		if f.Verdict == Fail && f.Check == "scope-mismatch" {
			found = true
		}
	}
	if !found {
		t.Fatal("stale ADR version not flagged")
	}
}

func TestEvidenceMissingFails(t *testing.T) {
	r := fixture(t, approvedADR, "# Evidence\n\nNothing here.\n")
	found := false
	for _, f := range r.RunOne("widget") {
		if f.Verdict == Fail && f.Check == "evidence-missing" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing evidence not flagged")
	}
}

func TestOverclaimWarns(t *testing.T) {
	adr := approvedADR + "\nDelivery is guaranteed delivery in all cases.\n"
	r := fixture(t, adr, evidenceWithWidget)
	var warns, fails int
	for _, f := range r.RunOne("widget") {
		if f.Verdict == Warn && f.Check == "overclaim" {
			warns++
		}
		if f.Verdict == Fail {
			fails++
		}
	}
	if warns != 1 || fails != 0 {
		t.Fatalf("warns=%d fails=%d; overclaim must warn, never fail", warns, fails)
	}
}

func TestNoChecksFound(t *testing.T) {
	r := fixture(t, approvedADR, evidenceWithWidget)
	found := false
	for _, f := range r.RunOne("nonexistent") {
		if f.Verdict == Fail {
			found = true
		}
	}
	if !found {
		t.Fatal("unknown target silently passed")
	}
}

func TestUnknownTargetMiniapp(t *testing.T) {
	r := fixture(t, approvedADR, evidenceWithWidget)
	r.Miniapps["plain"] = "1.0.0"
	r.IsClassD["plain"] = false
	found := false
	for _, f := range r.RunOne("plain") {
		if f.Verdict == Pass && f.Check == "adr-scope" {
			found = true
		}
		if f.Verdict == Fail {
			t.Fatalf("unexpected failure: %+v", f)
		}
	}
	if !found {
		t.Fatal("Class A miniapp not passed without an ADR")
	}
}
