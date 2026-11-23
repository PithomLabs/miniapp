// Package determinism implements the Phase 1 DBOS workflow
// determinism analyzer: the seven rule families of
// docs/phase0/determinism-rules.md, enforced as a go/analysis
// checker. It recognizes workflow bodies through
// dbos.RegisterWorkflow call sites and step bodies through
// dbos.RunAsStep / dbos.RunAsTransaction call arguments.
package determinism

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// dbosPath identifies the pinned DBOS Go SDK package.
const dbosPath = "github.com/dbos-inc/dbos-transact-golang/dbos"

// Analyzer flags nondeterministic constructs inside DBOS workflow
// bodies.
var Analyzer = &analysis.Analyzer{
	Name: "determinism",
	Doc:  "flags nondeterministic constructs in DBOS workflow bodies",
	Run:  run,
}

// nondeterministicFuncs lists time and randomness entry points that
// must not appear in workflow bodies. Pure constructors and constants
// stay allowed; anything wall-clock or random is listed.
var nondeterministicFuncs = map[string]map[string]bool{
	"time": {
		"Now": true, "Since": true, "Until": true, "After": true,
		"AfterFunc": true, "NewTimer": true, "NewTicker": true,
		"Tick": true, "Sleep": true,
	},
	"math/rand":   nil, // all entry points
	"crypto/rand": nil, // all entry points
}

// ioPackages lists packages whose calls are direct I/O (rule 6).
// Pure subpackages (net/url, net/netip) are carved out below.
var ioPackages = map[string]bool{
	"database/sql": true, "net": true, "net/http": true,
	"os": true, "io": true,
}

var ioPackagePrefixes = []string{
	"github.com/jackc/pgx",
	"github.com/PithomLabs/miniapp/internal/store",
}

var ioPureCarveouts = map[string]bool{
	"net/url": true, "net/netip": true,
}

// allowedVars exempts well-known immutable sentinel comparisons.
var allowedVars = map[string]bool{
	"io.EOF":                 true,
	"database/sql.ErrNoRows": true,
}

// stepOnlyAPIs must not be called from inside step bodies (rule 7).
var stepOnlyAPIs = map[string]bool{
	"RunAsTransaction": true, "Enqueue": true, "Send": true, "Recv": true,
}

func run(pass *analysis.Pass) (any, error) {
	decls := map[*types.Func]*ast.FuncDecl{}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil || pass.TypesInfo.Defs[fn.Name] == nil {
				continue
			}
			if obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func); ok {
				decls[obj] = fn
			}
		}
	}
	registered := map[*types.Func]bool{}
	var roots []*types.Func
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if !isDBOSFunc(pass, call.Fun, "RegisterWorkflow") {
				return true
			}
			if len(call.Args) < 2 {
				return true
			}
			if fn := calleeFunc(pass, call.Args[1]); fn != nil {
				if _, ok := decls[fn]; ok {
					registered[fn] = true
					roots = append(roots, fn)
				}
			}
			return true
		})
	}
	for _, fn := range roots {
		v := &visitor{pass: pass, decls: decls, registered: registered, visited: map[*types.Func]bool{fn: true}}
		v.walkFunc(decls[fn], false, 0)
	}
	return nil, nil
}

// visitor walks one workflow body (and one level of same-package
// helper calls) flagging rule violations.
type visitor struct {
	pass       *analysis.Pass
	decls      map[*types.Func]*ast.FuncDecl
	registered map[*types.Func]bool
	visited    map[*types.Func]bool
}

func (v *visitor) walkFunc(decl *ast.FuncDecl, inStep bool, depth int) {
	writes := collectWriteTargets(v.pass, decl.Body)
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GoStmt:
			v.report(n.Pos(), "determinism(goroutine): go statements are forbidden in workflow bodies; use the durable dbos.Go helper")
			return true
		case *ast.SelectStmt:
			v.report(n.Pos(), "determinism(select): select on channels is forbidden in workflow bodies; use the durable dbos.Select helper")
			return true
		case *ast.RangeStmt:
			if t := v.pass.TypesInfo.TypeOf(n.X); t != nil {
				if _, ok := t.Underlying().(*types.Map); ok {
					v.report(n.Pos(), "determinism(maporder): ranging over a map depends on iteration order")
				}
			}
			return true
		case *ast.CallExpr:
			return v.walkCall(n, inStep, depth)
		case *ast.Ident:
			v.checkGlobal(n, writes[n])
			return true
		}
		return true
	})
}

// collectWriteTargets finds idents assigned within a body: plain and
// compound assignment targets, inc/dec statements, and the mutated
// bases of index and selector assignments.
func collectWriteTargets(pass *analysis.Pass, body *ast.BlockStmt) map[*ast.Ident]bool {
	targets := map[*ast.Ident]bool{}
	addBase := func(e ast.Expr) {
		for {
			switch x := ast.Unparen(e).(type) {
			case *ast.Ident:
				if _, ok := pass.TypesInfo.Uses[x].(*types.Var); ok {
					targets[x] = true
				}
				return
			case *ast.IndexExpr:
				e = x.X
			case *ast.SelectorExpr:
				e = x.X
			case *ast.StarExpr:
				e = x.X
			default:
				return
			}
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				addBase(lhs)
			}
		case *ast.IncDecStmt:
			addBase(n.X)
		}
		return true
	})
	return targets
}

func (v *visitor) walkCall(call *ast.CallExpr, inStep bool, depth int) bool {
	if isDBOSFunc(v.pass, call.Fun, "RunAsStep") && len(call.Args) >= 2 {
		v.walkStepArg(call.Args[1], depth)
		return false
	}
	if isDBOSFunc(v.pass, call.Fun, "RunAsTransaction") && len(call.Args) >= 3 {
		v.walkStepArg(call.Args[2], depth)
		return false
	}
	if isDBOSFunc(v.pass, call.Fun, "Go") && len(call.Args) >= 2 {
		v.walkStepArg(call.Args[1], depth)
		return false
	}
	name, path := calleeIdentity(v.pass, call.Fun)
	if name == "" {
		return true
	}
	if inStep && path == dbosPath && stepOnlyAPIs[name] {
		v.report(call.Pos(), "determinism(stepapi): dbos."+name+" must not be called inside a step body")
		return true
	}
	if !inStep {
		if deny, ok := nondeterministicFuncs[path]; ok && (deny == nil || deny[name]) {
			v.report(call.Pos(), "determinism(nondet): "+path+"."+name+" is nondeterministic in workflow bodies; move it into a step")
			return true
		}
		if isIOPackage(path) {
			v.report(call.Pos(), "determinism(directio): "+path+"."+name+" is direct I/O in a workflow body; move it into a step or transaction")
			return true
		}
	}
	if depth < 1 {
		if fn := calleeFunc(v.pass, call.Fun); fn != nil && !v.visited[fn] && !v.registered[fn] {
			if decl, ok := v.decls[fn]; ok && fn.Pkg() == v.pass.Pkg {
				v.visited[fn] = true
				v.walkFunc(decl, inStep, depth+1)
			}
		}
	}
	return true
}

// walkStepArg visits a step function body in step mode: direct I/O
// and nondeterminism are allowed there (steps run once with
// recorded results), but nested durable APIs are not.
func (v *visitor) walkStepArg(arg ast.Expr, depth int) {
	switch fn := ast.Unparen(arg).(type) {
	case *ast.FuncLit:
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, path := calleeIdentity(v.pass, call.Fun)
			if path == dbosPath && stepOnlyAPIs[name] {
				v.report(call.Pos(), "determinism(stepapi): dbos."+name+" must not be called inside a step body")
			}
			return true
		})
	case *ast.Ident:
		if obj := calleeFunc(v.pass, fn); obj != nil && !v.visited[obj] {
			if decl, ok := v.decls[obj]; ok && obj.Pkg() == v.pass.Pkg {
				v.visited[obj] = true
				v.walkFunc(decl, true, depth+1)
			}
		}
	}
}

func (v *visitor) checkGlobal(id *ast.Ident, isWrite bool) {
	obj, ok := v.pass.TypesInfo.Uses[id].(*types.Var)
	if !ok || obj.Pkg() == nil {
		return
	}
	// Function parameters and locals also carry a package; only
	// vars declared at package scope (in any package) count.
	if obj.Parent() != obj.Pkg().Scope() {
		return
	}
	key := obj.Pkg().Path() + "." + obj.Name()
	if allowedVars[key] && !isWrite {
		return
	}
	if !isWrite && isDBOSTyped(obj) {
		// Framework handles (datasources) are setup-stable by
		// construction: set once before launch, never mutated
		// mid-workflow. Reads are replay-safe; writes still flag.
		return
	}
	v.report(id.Pos(), "determinism(global): package variable "+key+" must not be read or written in a workflow body; pass values through inputs and steps")
}

// isDBOSTyped reports whether a variable's type comes from the DBOS
// SDK package (through any pointer indirection).
func isDBOSTyped(v *types.Var) bool {
	t := v.Type()
	for {
		p, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = p.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == dbosPath
}

func (v *visitor) report(pos token.Pos, msg string) {
	v.pass.Report(analysis.Diagnostic{Pos: pos, Message: msg})
}

// isDBOSFunc reports whether fun resolves to dbos.<name>.
func isDBOSFunc(pass *analysis.Pass, fun ast.Expr, name string) bool {
	n, p := calleeIdentity(pass, fun)
	return n == name && p == dbosPath
}

// calleeIdentity resolves a called function to (name, package path).
func calleeIdentity(pass *analysis.Pass, fun ast.Expr) (string, string) {
	fn := calleeFunc(pass, fun)
	if fn == nil {
		return "", ""
	}
	pkg := ""
	if fn.Pkg() != nil {
		pkg = fn.Pkg().Path()
	}
	return fn.Name(), pkg
}

// calleeFunc resolves the called function object through idents,
// selectors, and instantiated generics.
func calleeFunc(pass *analysis.Pass, fun ast.Expr) *types.Func {
	switch f := ast.Unparen(fun).(type) {
	case *ast.Ident:
		if obj, ok := pass.TypesInfo.Uses[f].(*types.Func); ok {
			return obj
		}
	case *ast.SelectorExpr:
		if obj, ok := pass.TypesInfo.Uses[f.Sel].(*types.Func); ok {
			return obj
		}
		if sel, ok := pass.TypesInfo.Selections[f]; ok {
			if obj, ok := sel.Obj().(*types.Func); ok {
				return obj
			}
		}
	case *ast.IndexExpr, *ast.IndexListExpr:
		var x ast.Expr
		switch f := f.(type) {
		case *ast.IndexExpr:
			x = f.X
		case *ast.IndexListExpr:
			x = f.X
		}
		return calleeFunc(pass, x)
	}
	return nil
}

func isIOPackage(path string) bool {
	if ioPureCarveouts[path] {
		return false
	}
	if ioPackages[path] {
		return true
	}
	for _, p := range ioPackagePrefixes {
		if path == p || len(path) > len(p)+1 && path[:len(p)+1] == p+"/" {
			return true
		}
	}
	return false
}
