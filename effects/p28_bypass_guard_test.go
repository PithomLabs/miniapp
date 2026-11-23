package effects_test

// Prompt28 workstream F — provider Send-site inventory guard.
// Exactly one production call site may initiate provider work
// (sendAndRecord), through exactly two Channel implementations
// (mockcap; faultwrap is test-only). DBOS workflow signaling
// (dbos.Send) is allowlisted as non-provider by name. Any new
// Send implementation or call site fails loudly here instead
// of silently widening the provider surface. Mirrors the AST
// guard precedent (P24-009).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// p28SendSite is one provider-Send implementation or call site
// found in a single file. Both the inventory guard and the
// negative control run this same matcher, so a refactor that
// blinds it fails the control instead of passing vacuously.
type p28SendSite struct {
	line int
	kind string // "impl" or "call"
}

// p28SendSitesInFile parses one file and reports every Send
// implementation and non-dbos Send call site it contains.
func p28SendSitesInFile(path string) ([]p28SendSite, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	var out []p28SendSite
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Recv != nil && n.Name.Name == "Send" {
				out = append(out, p28SendSite{line: fset.Position(n.Pos()).Line, kind: "impl"})
			}
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Send" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "dbos" {
				return true // SDK workflow signaling, not provider work
			}
			out = append(out, p28SendSite{line: fset.Position(n.Pos()).Line, kind: "call"})
		}
		return true
	})
	return out, nil
}

func TestP28ProviderSendInventory(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(self))
	var impls, calls []string
	sawCanonical := false
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			t.Fatalf("rel %s: %v", path, rerr)
		}
		sites, err := p28SendSitesInFile(path)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, s := range sites {
			if s.kind == "impl" {
				impls = append(impls, rel)
				continue
			}
			calls = append(calls, rel)
			if rel == filepath.Join("effects", "effects.go") {
				sawCanonical = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantImpls := map[string]bool{
		filepath.Join("testkit", "mockcap", "mockcap.go"):     false,
		filepath.Join("testkit", "faultwrap", "faultwrap.go"): false,
	}
	for _, rel := range impls {
		seen, ok := wantImpls[rel]
		if !ok {
			t.Errorf("unexpected Channel.Send implementation at %s", rel)
			continue
		}
		if seen {
			t.Errorf("duplicate Send implementation at %s", rel)
		}
		wantImpls[rel] = true
	}
	for rel, seen := range wantImpls {
		if !seen {
			t.Errorf("expected Send implementation missing at %s (guard blind?)", rel)
		}
	}
	for _, rel := range calls {
		if rel != filepath.Join("effects", "effects.go") &&
			rel != filepath.Join("testkit", "faultwrap", "faultwrap.go") {
			t.Errorf("unexpected provider Send call site at %s", rel)
		}
	}
	if !sawCanonical {
		t.Error("canonical sendAndRecord dispatch not found in effects/effects.go (guard blind?)")
	}
}

// TestP28GuardNegativeControl proves the guard is not blind: it
// runs the same p28SendSitesInFile matcher the inventory test
// uses over the planted fixture, and requires the flagged line
// set to equal the planted // want: line set exactly — every
// planted violation flagged, and the dbos allowlist plus the
// bare-selector line silent.
func TestP28GuardNegativeControl(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(self))
	fixture := filepath.Join(root, "effects", "testdata", "bypass_guard_violations.go")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]bool{}
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "// want:") {
			want[i+1] = true
		}
	}
	if len(want) == 0 {
		t.Fatal("fixture has no planted violations; the control is vacuous")
	}
	found, err := p28SendSitesInFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	seen := map[int]bool{}
	for _, s := range found {
		seen[s.line] = true
	}
	for line := range want {
		if !seen[line] {
			t.Errorf("planted violation at fixture line %d was NOT flagged; the guard is blind", line)
		}
	}
	for _, s := range found {
		if !want[s.line] {
			t.Errorf("fixture line %d flagged without a // want: marker (%s); the guard over-matches", s.line, s.kind)
		}
	}
}
