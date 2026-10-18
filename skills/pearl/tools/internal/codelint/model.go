package codelint

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Import paths of the Charm libraries, every major version and vanity domain.
var (
	teaPaths      = []string{"charm.land/bubbletea/v2", "github.com/charmbracelet/bubbletea", "github.com/charmbracelet/bubbletea/v2"}
	lipglossPaths = []string{"charm.land/lipgloss/v2", "github.com/charmbracelet/lipgloss", "github.com/charmbracelet/lipgloss/v2"}
	keyPaths      = []string{"charm.land/bubbles/v2/key", "github.com/charmbracelet/bubbles/key", "github.com/charmbracelet/bubbles/v2/key"}
)

func in(p string, set []string) bool {
	for _, s := range set {
		if p == s {
			return true
		}
	}
	return false
}

// Root kinds: the model methods the event loop calls.
const (
	rootUpdate = 1 << iota
	rootView
	rootInit
)

// Reach says how a function is reached from the model: which roots, and the first path.
type Reach struct {
	Roots int
	Root  string   // the root's name, as written
	Via   []string // functions between the root and this one
}

// Program is what ModelAnalyzer learns about a package: its non-test files, how names
// resolve (with or without full types), and which code runs inside the event loop.
type Program struct {
	Pass  *analysis.Pass
	Files []*ast.File

	imports map[*ast.File]map[string]string
	decls   map[*types.Func]*ast.FuncDecl
	msgType types.Type // tea.Msg with aliases removed, if types are known

	// Sync holds the functions that run on the update goroutine from Update or View:
	// static calls and references to non-Cmd functions, never Cmd bodies or goroutines.
	Sync map[*ast.FuncDecl]*Reach
	// All adds Init and everything the model starts: Cmds, Tick callbacks, goroutines.
	All map[*ast.FuncDecl]*Reach
	// NewPrograms are the tea.NewProgram calls in the package.
	NewPrograms []*ast.CallExpr
}

// ModelAnalyzer finds the Bubble Tea model in a package and the code reachable from it.
var ModelAnalyzer = &analysis.Analyzer{
	Name:             "tuimodel",
	Doc:              "finds Bubble Tea models and the functions their Init, Update and View reach",
	Run:              runModel,
	ResultType:       reflect.TypeOf((*Program)(nil)),
	RunDespiteErrors: true,
}

func runModel(pass *analysis.Pass) (any, error) {
	p := &Program{Pass: pass, imports: map[*ast.File]map[string]string{}, decls: map[*types.Func]*ast.FuncDecl{}}
	for _, f := range pass.Files {
		name := pass.Fset.File(f.Pos()).Name()
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		p.Files = append(p.Files, f)
		m := map[string]string{}
		for _, s := range f.Imports {
			ip := strings.Trim(s.Path.Value, "`\"")
			n := guessName(ip)
			if s.Name != nil {
				n = s.Name.Name
			}
			m[n] = ip
		}
		p.imports[f] = m
	}
	for _, imp := range pass.Pkg.Imports() {
		if in(imp.Path(), teaPaths) {
			if o, ok := imp.Scope().Lookup("Msg").(*types.TypeName); ok {
				p.msgType = types.Unalias(o.Type())
			}
		}
	}
	// methods by receiver type name, for the syntax-only model test
	methods := map[string]map[string]*ast.FuncDecl{}
	var fns []*ast.FuncDecl
	for _, f := range p.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			fns = append(fns, fd)
			if fn, ok := pass.TypesInfo.Defs[fd.Name].(*types.Func); ok {
				p.decls[fn] = fd
			}
			if r := recvName(fd); r != "" {
				if methods[r] == nil {
					methods[r] = map[string]*ast.FuncDecl{}
				}
				methods[r][fd.Name.Name] = fd
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if r := p.Ref(c.Fun); in(r.Pkg, teaPaths) && r.Name == "NewProgram" {
					p.NewPrograms = append(p.NewPrograms, c)
				}
			}
			return true
		})
	}
	iface := p.teaModel()
	isModel := func(name string) bool {
		if iface != nil {
			obj, ok := pass.Pkg.Scope().Lookup(name).(*types.TypeName)
			if !ok {
				return false
			}
			t := obj.Type()
			return types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface)
		}
		ms := methods[name]
		return ms["Init"] != nil && ms["Update"] != nil && ms["View"] != nil
	}
	type root struct {
		fd   *ast.FuncDecl
		kind int
	}
	var roots []root
	updaters := map[string]bool{}
	for _, fd := range fns {
		r := recvName(fd)
		switch fd.Name.Name {
		case "Update", "update":
			if (r != "" && isModel(r) && fd.Name.Name == "Update") || p.takesMsg(fd.Type) {
				roots = append(roots, root{fd, rootUpdate})
				if r != "" {
					updaters[r] = true
				}
			}
		case "Init":
			if r != "" && isModel(r) {
				roots = append(roots, root{fd, rootInit})
			}
		}
	}
	for _, fd := range fns {
		r := recvName(fd)
		if r != "" && (fd.Name.Name == "View" && (isModel(r) || updaters[r]) || fd.Name.Name == "view" && updaters[r]) {
			roots = append(roots, root{fd, rootView})
		}
	}
	p.Sync = map[*ast.FuncDecl]*Reach{}
	p.All = map[*ast.FuncDecl]*Reach{}
	for _, rt := range roots {
		if rt.kind != rootInit {
			p.reach(p.Sync, rt.fd, rt.kind, false)
		}
		p.reach(p.All, rt.fd, rt.kind, true)
	}
	return p, nil
}

// recvName is the base type name of a method's receiver, or "".
func recvName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	t := fd.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.ParenExpr:
			t = x.X
		case *ast.IndexExpr:
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

// teaModel is the tea.Model interface, if the package imports a type-checked Bubble Tea.
func (p *Program) teaModel() *types.Interface {
	for _, imp := range p.Pass.Pkg.Imports() {
		if !in(imp.Path(), teaPaths) {
			continue
		}
		if o, ok := imp.Scope().Lookup("Model").(*types.TypeName); ok {
			if it, ok := o.Type().Underlying().(*types.Interface); ok {
				return it
			}
		}
	}
	return nil
}

// isMsg reports whether a type expression denotes tea.Msg.
func (p *Program) isMsg(e ast.Expr) bool {
	if p.msgType != nil {
		if t := p.Pass.TypesInfo.TypeOf(e); t != nil && types.Identical(types.Unalias(t), p.msgType) {
			return true
		}
	}
	r := p.Ref(e)
	return in(r.Pkg, teaPaths) && r.Name == "Msg"
}

// takesMsg reports whether a function has a tea.Msg parameter.
func (p *Program) takesMsg(ft *ast.FuncType) bool {
	for _, f := range ft.Params.List {
		if p.isMsg(f.Type) {
			return true
		}
	}
	return false
}

// returnsMsg reports whether a function's only result is a tea.Msg: a Cmd, or a Tick
// callback. Its body runs off the update goroutine.
func (p *Program) returnsMsg(ft *ast.FuncType) bool {
	if ft.Results == nil || len(ft.Results.List) != 1 || len(ft.Results.List[0].Names) > 1 {
		return false
	}
	return p.isMsg(ft.Results.List[0].Type)
}

// Ref names what an expression refers to: package path, receiver type name (methods) and
// object name. It uses types when they are known, then the file's imports.
type Ref struct{ Pkg, Recv, Name string }

// Ref resolves e (an identifier, selector, or either wrapped in parens or type arguments).
func (p *Program) Ref(e ast.Expr) Ref {
	info := p.Pass.TypesInfo
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
			continue
		case *ast.IndexExpr:
			e = x.X
			continue
		case *ast.IndexListExpr:
			e = x.X
			continue
		}
		break
	}
	switch x := e.(type) {
	case *ast.Ident:
		return objRef(info.Uses[x])
	case *ast.SelectorExpr:
		if sel := info.Selections[x]; sel != nil {
			r := objRef(sel.Obj())
			if r.Recv == "" {
				r.Recv = namedOf(sel.Recv())
			}
			return r
		}
		if o := info.Uses[x.Sel]; o != nil && o.Pkg() != nil {
			return objRef(o)
		}
		if id, ok := x.X.(*ast.Ident); ok {
			if pn, ok := info.Uses[id].(*types.PkgName); ok {
				return Ref{Pkg: pn.Imported().Path(), Name: x.Sel.Name}
			}
			if info.Uses[id] == nil && info.Defs[id] == nil {
				if ip := p.importFor(x, id.Name); ip != "" {
					return Ref{Pkg: ip, Name: x.Sel.Name}
				}
			}
		}
		// method on an expression whose type is unknown: recognise the usual sources
		if t := info.TypeOf(x.X); t == nil || t == types.Typ[types.Invalid] {
			switch src := p.Ref(callFun(x.X)); {
			case src.Pkg == "os/exec" && (src.Name == "Command" || src.Name == "CommandContext"):
				return Ref{Pkg: "os/exec", Recv: "Cmd", Name: x.Sel.Name}
			case src.Pkg == "net/http" && src.Name == "DefaultClient":
				return Ref{Pkg: "net/http", Recv: "Client", Name: x.Sel.Name}
			case src.Pkg == "os" && (src.Name == "Open" || src.Name == "Create" || src.Name == "OpenFile"):
				return Ref{Pkg: "os", Recv: "File", Name: x.Sel.Name}
			}
		}
	}
	return Ref{}
}

// callFun is the function of a call expression, or e itself.
func callFun(e ast.Expr) ast.Expr {
	if c, ok := e.(*ast.CallExpr); ok {
		return c.Fun
	}
	return e
}

func (p *Program) importFor(n ast.Node, name string) string {
	for _, f := range p.Files {
		if f.FileStart <= n.Pos() && n.Pos() <= f.FileEnd {
			return p.imports[f][name]
		}
	}
	return ""
}

func objRef(o types.Object) Ref {
	if o == nil || o.Pkg() == nil {
		return Ref{}
	}
	r := Ref{Pkg: o.Pkg().Path(), Name: o.Name()}
	if fn, ok := o.(*types.Func); ok {
		if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
			r.Recv = namedOf(sig.Recv().Type())
		}
	}
	return r
}

func namedOf(t types.Type) string {
	if pt, ok := t.(*types.Pointer); ok {
		t = pt.Elem()
	}
	if n, ok := types.Unalias(t).(*types.Named); ok {
		return n.Obj().Name()
	}
	return ""
}

// Walk visits every node under n, telling visit whether the node runs off the update
// goroutine: inside a function literal that returns a tea.Msg, or a go statement.
func (p *Program) Walk(n ast.Node, async bool, visit func(n ast.Node, async bool)) {
	ast.Inspect(n, func(c ast.Node) bool {
		if c == nil {
			return false
		}
		if c != n {
			switch c := c.(type) {
			case *ast.FuncLit:
				visit(c, async)
				p.Walk(c.Body, async || p.returnsMsg(c.Type), visit)
				return false
			case *ast.GoStmt:
				visit(c, async)
				p.Walk(c.Call, true, visit)
				return false
			}
		}
		visit(c, async)
		return true
	})
}

// edges lists the package's own functions that fd calls or refers to, and whether each
// runs off the update goroutine.
func (p *Program) edges(fd *ast.FuncDecl) []edge {
	var out []edge
	called := map[*ast.Ident]bool{}
	p.Walk(fd.Body, false, func(n ast.Node, async bool) {
		switch n := n.(type) {
		case *ast.CallExpr:
			switch f := ast.Unparen(n.Fun).(type) {
			case *ast.Ident:
				called[f] = true
			case *ast.SelectorExpr:
				called[f.Sel] = true
			case *ast.IndexExpr:
				if id, ok := f.X.(*ast.Ident); ok {
					called[id] = true
				}
			}
		case *ast.Ident:
			fn, ok := p.Pass.TypesInfo.Uses[n].(*types.Func)
			if !ok {
				return
			}
			callee := p.decls[fn.Origin()]
			if callee == nil {
				return
			}
			// a function value handed on (a Cmd, a callback) runs later if it returns a Msg
			out = append(out, edge{callee, async || (!called[n] && p.returnsMsg(callee.Type))})
		}
	})
	return out
}

type edge struct {
	to    *ast.FuncDecl
	async bool
}

// reach marks everything reachable from root in m, through async edges only if all.
func (p *Program) reach(m map[*ast.FuncDecl]*Reach, root *ast.FuncDecl, kind int, all bool) {
	if r := m[root]; r != nil && r.Roots&kind != 0 {
		return
	}
	if m[root] == nil {
		m[root] = &Reach{Root: root.Name.Name}
	}
	m[root].Roots |= kind
	queue := []*ast.FuncDecl{root}
	for len(queue) > 0 {
		fd := queue[0]
		queue = queue[1:]
		if fd.Body == nil {
			continue
		}
		from := m[fd]
		for _, e := range p.edges(fd) {
			if e.async && !all {
				continue
			}
			r := m[e.to]
			if r == nil {
				r = &Reach{Root: from.Root, Via: append(append([]string(nil), from.Via...), e.to.Name.Name)}
				m[e.to] = r
			}
			if r.Roots&kind == 0 {
				r.Roots |= kind
				queue = append(queue, e.to)
			}
		}
	}
}

// Sorted returns the functions of m in source order.
func Sorted(m map[*ast.FuncDecl]*Reach) []*ast.FuncDecl {
	out := make([]*ast.FuncDecl, 0, len(m))
	for fd := range m {
		out = append(out, fd)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pos() < out[j].Pos() })
	return out
}

// Where describes how a function is reached, for messages: "Update" or "Update (via load)".
func (r *Reach) Where() string {
	if len(r.Via) == 0 {
		return r.Root
	}
	return r.Root + " (via " + strings.Join(r.Via, " → ") + ")"
}

// lineOnce reports each (rule, file, line) once.
type lineOnce map[string]bool

func (l lineOnce) first(fset *token.FileSet, rule string, pos token.Pos) bool {
	p := fset.Position(pos)
	k := rule + "\x00" + p.Filename + "\x00" + strconv.Itoa(p.Line)
	if l[k] {
		return false
	}
	l[k] = true
	return true
}

// report files a diagnostic whose message carries the rule ID, so go vet shows it too.
func report(pass *analysis.Pass, pos token.Pos, rule, msg string) {
	pass.Report(analysis.Diagnostic{Pos: pos, Category: rule, Message: rule + ": " + msg})
}
