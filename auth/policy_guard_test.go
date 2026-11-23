// Guard test: grant/key maps are never touched directly outside
// the auth package. All reads go through accessors (pinned under
// RLock where required); all writes go through the setter API.
// Any new direct access fails loudly here instead of racing
// silently in production.
//
// AST matching (mirrors the INFO-2 precedent): subscript,
// whole-map and element assignment, delete/range/len over the
// map fields, and bare auth.Policy literals. Pure aliasing
// (`m := policy.Grants`, used only via the alias) is invisible
// without type information — it is indistinguishable from
// legitimate *Policy-pointer plumbing (`cfg.Grants`) — and is
// the documented residual limitation; no such shape exists
// (verified by grep), and every mutation or element observation
// is covered.
package auth

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// policyViolation is one direct map access outside auth.
type policyViolation struct {
	line int
	kind string
}

func isMapField(sel *ast.SelectorExpr) bool {
	return sel.Sel.Name == "Grants" || sel.Sel.Name == "Keys"
}

// authLocalName resolves the file-local name bound to the auth
// package import (default "auth"; honors explicit aliases).
// Dot imports report dot=true.
func authLocalName(f *ast.File) (name string, dot bool) {
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != "github.com/PithomLabs/miniapp/auth" {
			continue
		}
		if imp.Name != nil {
			if imp.Name.Name == "." {
				return "", true
			}
			return imp.Name.Name, false
		}
		return "auth", false
	}
	return "", false
}

// policyViolationsInFile parses one file and reports every direct
// policy-map access it contains.
func policyViolationsInFile(path string) ([]policyViolation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	authName, authDot := authLocalName(f)
	var out []policyViolation
	flagged := map[int]bool{}
	flag := func(n ast.Node, kind string) {
		line := fset.Position(n.Pos()).Line
		if flagged[line] {
			return
		}
		flagged[line] = true
		out = append(out, policyViolation{line: line, kind: kind})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.IndexExpr:
			if sel, ok := n.X.(*ast.SelectorExpr); ok && isMapField(sel) {
				flag(n, "subscript on policy map field")
			}
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				if sel, ok := lhs.(*ast.SelectorExpr); ok && isMapField(sel) {
					flag(lhs, "whole-map assignment bypasses the setter and hash discipline")
					continue
				}
				if idx, ok := lhs.(*ast.IndexExpr); ok {
					if sel, ok := idx.X.(*ast.SelectorExpr); ok && isMapField(sel) {
						flag(lhs, "element assignment bypasses the setter and hash discipline")
					}
				}
			}
		case *ast.CompositeLit:
			if sel, ok := n.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Policy" {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == authName && authName != "" {
					flag(n, "bare auth.Policy literal outside auth package (use auth.NewPolicy)")
				}
			}
			if id, ok := n.Type.(*ast.Ident); ok && authDot && id.Name == "Policy" {
				flag(n, "bare auth.Policy literal outside auth package (use auth.NewPolicy)")
			}
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok && len(n.Args) > 0 &&
				(id.Name == "delete" || id.Name == "len") {
				if sel, ok := n.Args[0].(*ast.SelectorExpr); ok && isMapField(sel) {
					flag(n, id.Name+" over policy map field")
				}
			}
		case *ast.RangeStmt:
			if sel, ok := n.X.(*ast.SelectorExpr); ok && isMapField(sel) {
				flag(n, "range over policy map field")
			}
		}
		return true
	})
	return out, nil
}

func walkGoFiles(t *testing.T, root string, fn func(rel, path string)) {
	t.Helper()
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
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			t.Fatalf("rel %s: %v", path, rerr)
		}
		if strings.HasPrefix(rel, "auth"+string(filepath.Separator)) {
			return nil
		}
		fn(rel, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestP23NoDirectPolicyMapAccess(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(self))
	walkGoFiles(t, root, func(rel, path string) {
		found, err := policyViolationsInFile(path)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, v := range found {
			t.Errorf("%s:%d: direct policy map access outside auth package: %s", rel, v.line, v.kind)
		}
	})
}

// TestP23GuardNegativeControl proves the guard is not blind: every
// planted violation in the testdata fixture must be flagged. If a
// refactor blinds the matcher, this fails instead of passing
// vacuously.
func TestP23GuardNegativeControl(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(self))
	fixture := filepath.Join(root, "auth", "testdata", "policy_guard_violations.go")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var want []int
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "// want:") {
			want = append(want, i+1)
		}
	}
	if len(want) == 0 {
		t.Fatal("fixture has no planted violations; the control is vacuous")
	}
	found, err := policyViolationsInFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	seen := map[int]bool{}
	for _, v := range found {
		seen[v.line] = true
	}
	for _, line := range want {
		if !seen[line] {
			t.Errorf("planted violation at fixture line %d was NOT flagged; the guard is blind", line)
		}
	}
}
