// INFO-2 mechanical guard: production advisory-lock acquisition
// stays pinned to the Class A framework path. The §4 no-cycle
// proof assumes miniapps (same module, exported functions) never
// take advisory locks directly; this test makes that assumption
// structural instead of tribal. Guard-only: no defect exists, so
// no red run applies — it fails loudly if anyone adds a second
// acquisition site or renames the calls into silence.
package store_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestP20AdvisoryAcquisitionSingleSite asserts every production
// call to the advisory primitives lives in
// executor/classa/classa.go, and that both known calls are still
// present (so a rename cannot pass vacuously).
func TestP20AdvisoryAcquisitionSingleSite(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(self)))
	want := map[string]bool{"LockEntity": false, "TryClaimExecution": false}
	found := 0
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
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			// Match by selector name alone (not the qualifier):
			// an aliased import must not slip through.
			if _, watched := want[sel.Sel.Name]; !watched {
				return true
			}
			found++
			want[sel.Sel.Name] = true
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				t.Fatalf("rel %s: %v", path, rerr)
			}
			if rel != filepath.Join("executor", "classa", "classa.go") {
				t.Errorf("advisory acquisition %s at %s; production acquisition is restricted to executor/classa/classa.go",
					sel.Sel.Name, rel)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == 0 {
		t.Fatal("no advisory acquisitions found; the guard is blind (renamed primitives?)")
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("expected acquisition %s not found; update the guard deliberately, not by accident", name)
		}
	}
}
