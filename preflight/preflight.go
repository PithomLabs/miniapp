package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Verdict is one check outcome.
type Verdict string

const (
	Pass Verdict = "PASS"
	Warn Verdict = "WARN"
	Fail Verdict = "FAIL"
	Skip Verdict = "SKIP"
)

// Finding is one structural check result.
type Finding struct {
	Verdict Verdict
	Check   string
	Detail  string
}

// Runner holds preflight inputs.
type Runner struct {
	// ADRDir holds per-workload decision records.
	ADRDir string
	// Evidence is the evidence register README.
	Evidence string
	// Miniapps maps registered miniapp IDs to their versions. Only
	// miniapps the binary registers are visible; anything else is
	// undeployable and out of scope.
	Miniapps map[string]string
	// IsClassD reports whether a registered miniapp has Class D tools.
	IsClassD map[string]bool
}

// RunAll checks every ADR file plus every registered miniapp, in
// deterministic order.
func (r *Runner) RunAll() []Finding {
	files, err := filepath.Glob(filepath.Join(r.ADRDir, "*.md"))
	if err != nil {
		return []Finding{{Fail, "adr-glob", err.Error()}}
	}
	var out []Finding
	for _, f := range files {
		base := filepath.Base(f)
		if base == "adr001-template.md" || base == "inventory.md" {
			continue
		}
		out = append(out, r.checkADRFile(f)...)
	}
	var ids []string
	for id := range r.Miniapps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		out = append(out, r.checkMiniapp(id)...)
	}
	return out
}

// RunOne checks a single registered miniapp and its ADR record.
func (r *Runner) RunOne(id string) []Finding {
	version, ok := r.Miniapps[id]
	if !ok {
		return []Finding{{Fail, "target", fmt.Sprintf("miniapp %q is not registered", id)}}
	}
	out := r.checkMiniapp(id)
	if r.IsClassD[id] {
		path := filepath.Join(r.ADRDir, id+"-adr001.md")
		if _, err := os.Stat(path); err != nil {
			return append(out, Finding{Fail, "adr-missing",
				fmt.Sprintf("Class D miniapp %q has no %s", id, path)})
		}
		out = append(out, r.checkADRFile(path)...)
	}
	_ = version
	return out
}

func (r *Runner) checkMiniapp(id string) []Finding {
	if !r.IsClassD[id] {
		return []Finding{{Pass, "adr-scope",
			fmt.Sprintf("%s is not Class D; no decision record required", id)}}
	}
	path := filepath.Join(r.ADRDir, id+"-adr001.md")
	if _, err := os.Stat(path); err != nil {
		return []Finding{{Fail, "adr-missing",
			fmt.Sprintf("Class D miniapp %q has no %s", id, path)}}
	}
	return nil
}

var (
	titleRe  = regexp.MustCompile(`(?m)^# ADR-001 decision [—-] (\S+) v(\S+)`)
	statusRe = regexp.MustCompile(`(?m)^Status:\s*\**([A-Z]+)`)
	// Decision lines look like "- Decision: PROPOSED ADOPT ...".
	decisionWords = []string{"REJECT", "DEFER", "PILOT", "ADOPT", "PROPOSED"}
	reviewerRe    = regexp.MustCompile(`(?m)^- Reviewer \(human\):\s*(.*?)(?:\s{2,}Date:\s*(.*))?$`)
	scoreBlankRe  = regexp.MustCompile(`(?m)^- Score.*__`)
)

var (
	validStatuses  = map[string]bool{"DRAFT": true, "APPROVED": true, "SUPERSEDED": true, "REJECTED": true}
	validDecisions = map[string]bool{"REJECT": true, "DEFER": true, "PILOT": true, "ADOPT": true, "PROPOSED": true}
)

func (r *Runner) checkADRFile(path string) []Finding {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []Finding{{Fail, "adr-read", err.Error()}}
	}
	text := string(raw)
	id := strings.TrimSuffix(filepath.Base(path), "-adr001.md")
	var out []Finding

	m := titleRe.FindStringSubmatch(text)
	if m == nil {
		out = append(out, Finding{Fail, "adr-title",
			fmt.Sprintf("%s: title must match '# ADR-001 decision — <id> v<version>'", id)})
		return out
	}
	if m[1] != id {
		out = append(out, Finding{Fail, "scope-mismatch",
			fmt.Sprintf("%s: title names miniapp %q", id, m[1])})
	}
	version, registered := r.Miniapps[m[1]]
	if registered && m[2] != version {
		out = append(out, Finding{Fail, "scope-mismatch",
			fmt.Sprintf("%s: ADR targets v%s but registered v%s", id, m[2], version)})
	}

	sm := statusRe.FindStringSubmatch(text)
	if sm == nil {
		out = append(out, Finding{Fail, "adr-status", id + ": no Status line"})
		return out
	}
	status := sm[1]
	if !validStatuses[status] {
		out = append(out, Finding{Fail, "adr-status",
			fmt.Sprintf("%s: status %q outside controlled vocabulary", id, status)})
	}

	decision := ""
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "- Decision:") {
			continue
		}
		upper := strings.ToUpper(line)
		for _, w := range decisionWords {
			if strings.Contains(upper, w) {
				decision = w
				break
			}
		}
	}
	if decision == "" {
		out = append(out, Finding{Fail, "adr-decision", id + ": no decision in controlled vocabulary"})
	} else if !validDecisions[decision] {
		out = append(out, Finding{Fail, "adr-decision", id + ": decision outside controlled vocabulary"})
	}

	out = append(out, lintOverclaims(id, text)...)

	switch {
	case status == "DRAFT" && !registered:
		out = append(out, Finding{Skip, "adr-state",
			fmt.Sprintf("%s: draft decision precedes implementation (hard-stop pre-state)", id)})
		return out
	case status == "DRAFT" && registered:
		out = append(out, Finding{Fail, "adr-state",
			fmt.Sprintf("%s: DRAFT decision with registered implementation is blocked until human approval", id)})
	case status == "APPROVED" && !registered:
		out = append(out, Finding{Fail, "adr-orphan",
			fmt.Sprintf("%s: approved decision with no registered implementation", id)})
	case status == "APPROVED" && registered:
		if decision != "ADOPT" && decision != "PILOT" {
			out = append(out, Finding{Fail, "adr-state",
				fmt.Sprintf("%s: approved record holds non-enabling decision %s", id, decision)})
		}
	}

	rm := reviewerRe.FindStringSubmatch(text)
	reviewer, date := "", ""
	if rm != nil {
		reviewer, date = strings.TrimSpace(rm[1]), strings.TrimSpace(rm[2])
	}
	reviewed := reviewer != "" && !strings.Contains(reviewer, "awaiting") && !strings.Contains(reviewer, "pending") &&
		!strings.Contains(reviewer, "<")
	if !reviewed && (registered || status == "APPROVED") {
		out = append(out, Finding{Fail, "adr-reviewer",
			fmt.Sprintf("%s: no reviewer recorded", id)})
	}
	if scoreBlankRe.MatchString(text) && !reviewed {
		out = append(out, Finding{Fail, "adr-score",
			fmt.Sprintf("%s: score unconfirmed and no reviewer recorded", id)})
	}

	out = append(out, r.checkEvidenceRefs(id, text, date)...)
	return out
}

// checkEvidenceRefs requires evidence-register rows for the miniapp
// and warns on review-state drift (wording and temporal).
func (r *Runner) checkEvidenceRefs(id, adrText, adrDate string) []Finding {
	raw, err := os.ReadFile(r.Evidence)
	if err != nil {
		return []Finding{{Fail, "evidence-read", err.Error()}}
	}
	text := string(raw)
	if !strings.Contains(text, id) {
		return []Finding{{Fail, "evidence-missing",
			fmt.Sprintf("%s: no evidence-register row names this miniapp", id)}}
	}
	var out []Finding
	if strings.Contains(strings.ToUpper(adrText), "PLANNED") {
		out = append(out, Finding{Warn, "recheck-wording",
			fmt.Sprintf("%s: ADR still marks items planned while evidence exists; recheck wording", id)})
	}
	if latest := latestEvidenceDate(text, id); latest != "" && adrDate != "" && adrDate < latest {
		out = append(out, Finding{Warn, "recheck-temporal",
			fmt.Sprintf("%s: evidence dated %s is newer than ADR review date %s; recheck", id, latest, adrDate)})
	}
	return out
}

var evidenceDateRe = regexp.MustCompile(`\| (\d{4}-\d{2}-\d{2}) \|`)

func latestEvidenceDate(register, id string) string {
	latest := ""
	for _, line := range strings.Split(register, "\n") {
		if !strings.Contains(line, id) {
			continue
		}
		for _, m := range evidenceDateRe.FindAllStringSubmatch(line, -1) {
			if m[1] > latest {
				latest = m[1]
			}
		}
	}
	return latest
}

// overclaimPhrases warn (never fail): natural-language guarantee
// claims a structural check cannot judge.
var overclaimPhrases = []string{
	"guaranteed delivery", "highly available", "exactly once",
	"exactly-once", "zero downtime", "never fails",
}

// alwaysFailPhrases fail the run. The list is intentionally empty:
// no phrase has yet been judged severe enough to fail mechanically.
var alwaysFailPhrases []string

func lintOverclaims(id, text string) []Finding {
	var out []Finding
	lower := strings.ToLower(text)
	for _, p := range alwaysFailPhrases {
		if strings.Contains(lower, strings.ToLower(p)) {
			out = append(out, Finding{Fail, "overclaim",
				fmt.Sprintf("%s: banned guarantee phrase %q", id, p)})
		}
	}
	for i, line := range strings.Split(text, "\n") {
		for _, p := range overclaimPhrases {
			if strings.Contains(strings.ToLower(line), p) {
				out = append(out, Finding{Warn, "overclaim",
					fmt.Sprintf("%s:%d: guarantee phrase %q needs reviewer judgment", id, i+1, p)})
			}
		}
	}
	return out
}

// Summarize counts findings by verdict.
func Summarize(findings []Finding) (pass, warn, fail, skip int) {
	for _, f := range findings {
		switch f.Verdict {
		case Pass:
			pass++
		case Warn:
			warn++
		case Fail:
			fail++
		case Skip:
			skip++
		}
	}
	return pass, warn, fail, skip
}
