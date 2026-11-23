// Command preflight validates architectural-contract consistency:
// registered miniapps against ADR-001 decision records and the
// evidence register. It checks structured facts only and exits
// nonzero on any failure. Run from the repository root:
//
//	preflight --check-all
//	preflight --miniapp crm-read
//
// Exit codes: 0 pass (warnings allowed), 1 failed check,
// 2 usage error or no checks evaluated.
package main

import (
	"flag"
	"fmt"
	"os"

	crmread "github.com/PithomLabs/miniapp/miniapps/crm-read"
	"github.com/PithomLabs/miniapp/preflight"
)

func main() {
	os.Exit(run())
}

func run() int {
	checkAll := flag.Bool("check-all", false, "check every ADR file and registered miniapp")
	miniapp := flag.String("miniapp", "", "check one registered miniapp")
	adrDir := flag.String("adr-dir", "docs/adr/miniapps", "decision records directory")
	evidence := flag.String("evidence", "docs/evidence/README.md", "evidence register")
	flag.Parse()

	if (*checkAll && *miniapp != "") || (!*checkAll && *miniapp == "") {
		fmt.Fprintln(os.Stderr, "usage: preflight (--check-all | --miniapp <id>) [--adr-dir DIR] [--evidence FILE]")
		return 2
	}

	// Shipped miniapps only. Test fixtures (such as the Class D
	// conformance probe) are intentionally absent: they carry no
	// workload decision and must never be mistaken for one.
	// lead-followup registers here only after its ADR is approved
	// and its implementation lands, per the hard-stop rule.
	r := &preflight.Runner{
		ADRDir:   *adrDir,
		Evidence: *evidence,
		Miniapps: map[string]string{string(crmread.ID): string(crmread.Version)},
		IsClassD: map[string]bool{string(crmread.ID): false},
	}

	var findings []preflight.Finding
	if *checkAll {
		findings = r.RunAll()
	} else {
		findings = r.RunOne(*miniapp)
	}
	if len(findings) == 0 {
		fmt.Fprintln(os.Stderr, "preflight: no checks evaluated (nothing silently passes)")
		return 2
	}
	pass, warn, fail, skip := preflight.Summarize(findings)
	for _, f := range findings {
		fmt.Printf("%s %s: %s\n", f.Verdict, f.Check, f.Detail)
	}
	fmt.Printf("preflight: %d pass, %d warn, %d fail, %d skip\n", pass, warn, fail, skip)
	if fail > 0 {
		return 1
	}
	return 0
}
