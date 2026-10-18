package codelint

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// C-HUHTHEME

// HuhThemeAnalyzer reports Huh forms, Huh fields run on their own, Huh spinners and
// Bubbles spinners that are never given the brand's styles (C-HUHTHEME). Each one is
// followed from where it is made, through builder chains, variables, struct fields,
// helper parameters and helper results, to a WithTheme (or a spinner style). A value
// that leaves the package's sight (an unknown call, a slice, an exported name in a
// library package) is given the benefit of the doubt.
var HuhThemeAnalyzer = newAnalyzer("tuihuhtheme",
	"C-HUHTHEME: a Huh form, field or spinner, or a Bubbles spinner, with no theme or style; it renders the library's default colours, not the brand",
	runHuhTheme)

var (
	huhPaths            = []string{"charm.land/huh/v2", "github.com/charmbracelet/huh", "github.com/charmbracelet/huh/v2"}
	huhSpinnerPaths     = []string{"charm.land/huh/v2/spinner", "github.com/charmbracelet/huh/spinner", "github.com/charmbracelet/huh/v2/spinner"}
	bubblesSpinnerPaths = []string{"charm.land/bubbles/v2/spinner", "github.com/charmbracelet/bubbles/spinner", "github.com/charmbracelet/bubbles/v2/spinner"}
	// charmPrefixes are libraries that take a value without restyling it: handing a
	// form to tea.NewProgram is not theming it, and not losing sight of it either.
	charmPrefixes = []string{"charm.land/", "github.com/charmbracelet/"}

	huhFields = map[string]bool{"NewInput": true, "NewText": true, "NewSelect": true, "NewMultiSelect": true,
		"NewConfirm": true, "NewNote": true, "NewFilePicker": true}
	huhStock = map[string]bool{"ThemeCharm": true, "ThemeDracula": true, "ThemeBase16": true, "ThemeCatppuccin": true,
		"ThemeBase": true, "ThemeDefault": true}
	// huhStop are the Form, Field and Spinner methods that do not return the receiver:
	// a chain through them no longer holds the value being followed.
	huhStop = map[string]bool{"Init": true, "Update": true, "View": true, "Run": true, "RunWithContext": true,
		"RunAccessible": true, "Errors": true, "Help": true, "KeyBinds": true, "Get": true, "GetString": true,
		"GetInt": true, "GetBool": true, "NextGroup": true, "PrevGroup": true, "NextField": true, "PrevField": true,
		"GetFocusedField": true, "Focus": true, "Blur": true, "Error": true, "String": true, "GetKey": true,
		"GetValue": true, "GetFiltering": true, "Hovered": true, "Skip": true, "Zoom": true}
)

type huhKind int

const (
	huhForm huhKind = iota
	huhField
	huhSpinner
	bubblesSpinner
)

// huhTrace is what following one value found.
type huhTrace struct {
	kind    huhKind
	themed  bool        // given a theme or style that is not a stock one
	stock   []token.Pos // WithTheme calls given a stock Huh theme
	stockAs string
	ran     bool // a field run on its own: .Run() or huh.Run(field)
	escaped bool // handed somewhere this package cannot see
}

type huhPass struct {
	p      *Program
	parent map[ast.Node]ast.Node
	occ    map[any][]ast.Expr
	main   bool
}

func runHuhTheme(pass *analysis.Pass) (any, error) {
	p := program(pass)
	uses := false
	for _, m := range p.imports {
		for _, ip := range m {
			uses = uses || in(ip, huhPaths) || in(ip, huhSpinnerPaths) || in(ip, bubblesSpinnerPaths)
		}
	}
	if !uses {
		return nil, nil
	}
	h := &huhPass{p: p, parent: map[ast.Node]ast.Node{}, occ: map[any][]ast.Expr{}, main: pass.Pkg.Name() == "main"}
	type made struct {
		call *ast.CallExpr
		kind huhKind
		name string
	}
	var all []made
	for _, f := range p.Files {
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			if len(stack) > 0 {
				h.parent[n] = stack[len(stack)-1]
			}
			stack = append(stack, n)
			switch x := n.(type) {
			case *ast.CallExpr:
				r := p.Ref(x.Fun)
				switch {
				case in(r.Pkg, huhPaths) && r.Name == "NewForm" && r.Recv == "":
					all = append(all, made{x, huhForm, "NewForm"})
				case in(r.Pkg, huhPaths) && huhFields[r.Name] && r.Recv == "":
					all = append(all, made{x, huhField, r.Name})
				case in(r.Pkg, huhSpinnerPaths) && r.Name == "New" && r.Recv == "":
					all = append(all, made{x, huhSpinner, "New"})
				case in(r.Pkg, bubblesSpinnerPaths) && r.Name == "New" && r.Recv == "":
					all = append(all, made{x, bubblesSpinner, "New"})
				}
			case *ast.Ident, *ast.SelectorExpr:
				if s, ok := h.parent[n].(*ast.SelectorExpr); ok && s.Sel == n {
					break // the selector as a whole is the occurrence
				}
				if id, ok := x.(*ast.Ident); ok && pass.TypesInfo.Defs[id] != nil {
					break // a declaration, not a use
				}
				if k := h.key(x.(ast.Expr)); k != nil {
					h.occ[k] = append(h.occ[k], x.(ast.Expr))
				}
			}
			return true
		})
	}
	once := lineOnce{}
	for _, m := range all {
		t := &huhTrace{kind: m.kind}
		if m.kind == bubblesSpinner {
			t.themed = h.styledByOption(m.call)
		}
		if !t.themed {
			h.trace(t, m.call)
		}
		var msg string
		pos := m.call.Pos()
		switch {
		case t.themed:
			continue
		case len(t.stock) > 0:
			pos = t.stock[0]
			what := map[huhKind]string{huhForm: "huh form", huhField: "huh field", huhSpinner: "huh spinner"}[m.kind]
			msg = what + " themed with " + t.stockAs + ", a stock theme: it renders Huh's colours, not the brand; build the theme from the brand's roles"
		case t.escaped:
			continue
		case m.kind == huhForm:
			msg = "huh form without WithTheme: it renders Huh's default colours, not the brand"
		case m.kind == huhField && t.ran:
			msg = "huh." + m.name + " run without WithTheme: it renders Huh's default colours, not the brand"
		case m.kind == huhSpinner:
			msg = "huh spinner without WithTheme: it renders Huh's default fuchsia #F780E2, not the brand"
		case m.kind == bubblesSpinner:
			msg = "spinner.New without spinner.WithStyle or a .Style: it renders unstyled, not in the brand's accent"
		default:
			continue
		}
		if once.first(pass.Fset, "C-HUHTHEME", pos) {
			report(pass, pos, "C-HUHTHEME", msg)
		}
	}
	return nil, nil
}

// styledByOption reports whether a Bubbles spinner.New call is given a style, or options
// this package cannot see into.
func (h *huhPass) styledByOption(call *ast.CallExpr) bool {
	if call.Ellipsis.IsValid() {
		return true
	}
	for _, a := range call.Args {
		c, ok := ast.Unparen(a).(*ast.CallExpr)
		if !ok {
			return true // an Option held in a variable
		}
		r := h.p.Ref(c.Fun)
		if !in(r.Pkg, bubblesSpinnerPaths) || r.Name == "WithStyle" {
			return true
		}
	}
	return false
}

// key identifies what an expression names, so its other occurrences can be found: the
// object when it resolves (a variable, struct field, parameter or function), else the
// expression as written.
func (h *huhPass) key(e ast.Expr) any {
	info := h.p.Pass.TypesInfo
	var obj types.Object
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		if x.Name == "_" {
			return nil
		}
		obj = info.Uses[x]
		if obj == nil {
			obj = info.Defs[x]
		}
		if obj == nil {
			return "id:" + x.Name
		}
	case *ast.SelectorExpr:
		if sel := info.Selections[x]; sel != nil {
			obj = sel.Obj()
		} else if o := info.Uses[x.Sel]; o != nil {
			obj = o
		} else {
			return "sel:" + types.ExprString(x)
		}
	default:
		return nil
	}
	switch o := obj.(type) {
	case *types.PkgName, *types.TypeName, *types.Const, *types.Builtin, *types.Nil, *types.Label:
		return nil
	case *types.Func:
		return o.Origin()
	case *types.Var:
		return o.Origin()
	}
	return obj
}

// trace follows the value made by call to everything that holds it.
func (h *huhPass) trace(t *huhTrace, call *ast.CallExpr) {
	seen := map[any]bool{}
	var keys []any
	add := func(k any) {
		if k == nil {
			t.escaped = true
			return
		}
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
		if o, ok := k.(types.Object); ok && !h.main && o.Exported() && o.Pkg() != nil {
			// an exported function, field or package variable of a library: callers in
			// other packages may theme it
			if v, ok := o.(*types.Var); !ok || v.IsField() || v.Parent() == o.Pkg().Scope() {
				t.escaped = true
			}
		}
	}
	h.follow(t, call, add)
	for i := 0; i < len(keys) && i < 64; i++ {
		k := keys[i]
		for _, e := range h.occ[k] {
			if _, isFn := k.(*types.Func); isFn {
				c, ok := h.parent[e].(*ast.CallExpr)
				if !ok || c.Fun != e {
					t.escaped = true // a function value handed on
					continue
				}
				h.follow(t, c, add)
				continue
			}
			h.follow(t, e, add)
		}
	}
}

// follow walks up from an expression holding the value: through builder chains to the
// statement that stores, returns or hands it on.
func (h *huhPass) follow(t *huhTrace, e ast.Expr, add func(any)) {
	info := h.p.Pass.TypesInfo
	var cur ast.Node = e
	for {
		switch p := h.parent[cur].(type) {
		case *ast.ParenExpr:
			cur = p
			continue
		case *ast.StarExpr:
			cur = p
			continue
		case *ast.UnaryExpr:
			if p.Op != token.AND {
				return
			}
			cur = p
			continue
		case *ast.SelectorExpr:
			if p.X != cur {
				return
			}
			name := p.Sel.Name
			c, called := h.parent[p].(*ast.CallExpr)
			if !called || c.Fun != p {
				// m.spin.Style = s
				if a, ok := h.parent[p].(*ast.AssignStmt); ok && name == "Style" && t.kind == bubblesSpinner {
					for _, l := range a.Lhs {
						if l == p {
							t.themed = true
						}
					}
				}
				return
			}
			if t.kind == bubblesSpinner {
				return // spinner.Model has no builder methods
			}
			switch {
			case name == "WithTheme" || t.kind == huhSpinner && name == "Style":
				if len(c.Args) == 1 {
					if s := h.stock(c.Args[0]); s != "" {
						t.stock = append(t.stock, p.Sel.Pos())
						t.stockAs = s
						break
					}
				}
				t.themed = true
			case name == "Run" || name == "RunWithContext":
				t.ran = true
			}
			if huhStop[name] {
				return
			}
			cur = c
			continue
		case *ast.CallExpr:
			if p.Fun == cur {
				return
			}
			if tv, ok := info.Types[p.Fun]; ok && tv.IsType() {
				cur = p // a conversion holds the same value
				continue
			}
			r := h.p.Ref(p.Fun)
			if in(r.Pkg, huhPaths) && r.Name == "Run" && r.Recv == "" {
				t.ran = true
				return
			}
			if fd := h.decl(p.Fun); fd != nil {
				h.param(t, fd, p, cur, add)
				return
			}
			for _, pre := range charmPrefixes {
				if strings.HasPrefix(r.Pkg, pre) {
					return
				}
			}
			t.escaped = true
			return
		case *ast.AssignStmt:
			if len(p.Lhs) != len(p.Rhs) {
				t.escaped = true // one of several results: which one is not tracked
				return
			}
			for i, r := range p.Rhs {
				if r == cur && !blankIdent(p.Lhs[i]) {
					add(h.key(p.Lhs[i]))
				}
			}
			return
		case *ast.ValueSpec:
			for i, v := range p.Values {
				if v == cur && i < len(p.Names) && p.Names[i].Name != "_" {
					add(h.key(p.Names[i]))
				}
			}
			return
		case *ast.KeyValueExpr:
			if p.Value != cur {
				return
			}
			if id, ok := p.Key.(*ast.Ident); ok {
				if v, ok := info.Uses[id].(*types.Var); ok && v.IsField() {
					add(v.Origin())
					return
				}
				if info.Uses[id] == nil && info.Defs[id] == nil {
					// field of a struct whose type is unknown
					add("field:" + id.Name)
					return
				}
			}
			t.escaped = true
			return
		case *ast.ReturnStmt:
			for n := h.parent[p]; n != nil; n = h.parent[n] {
				switch fn := n.(type) {
				case *ast.FuncLit:
					t.escaped = true
					return
				case *ast.FuncDecl:
					if o, ok := info.Defs[fn.Name].(*types.Func); ok {
						add(o.Origin())
					} else {
						t.escaped = true
					}
					return
				}
			}
			return
		case *ast.ExprStmt, *ast.IfStmt, *ast.SwitchStmt, *ast.BinaryExpr:
			return // used, not held
		default:
			t.escaped = true
			return
		}
	}
}

func blankIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}

// decl is the package's own declaration of the function a call calls, if any.
func (h *huhPass) decl(fun ast.Expr) *ast.FuncDecl {
	info := h.p.Pass.TypesInfo
	var obj types.Object
	switch x := ast.Unparen(fun).(type) {
	case *ast.Ident:
		obj = info.Uses[x]
	case *ast.SelectorExpr:
		if sel := info.Selections[x]; sel != nil {
			obj = sel.Obj()
		} else {
			obj = info.Uses[x.Sel]
		}
	case *ast.IndexExpr:
		return h.decl(x.X)
	case *ast.IndexListExpr:
		return h.decl(x.X)
	}
	if fn, ok := obj.(*types.Func); ok {
		return h.p.decls[fn.Origin()]
	}
	return nil
}

// param follows a value passed to one of the package's own functions into the parameter
// that receives it.
func (h *huhPass) param(t *huhTrace, fd *ast.FuncDecl, call *ast.CallExpr, arg ast.Node, add func(any)) {
	idx := -1
	for i, a := range call.Args {
		if a == arg {
			idx = i
		}
	}
	var names []*ast.Ident
	variadic := false
	for _, f := range fd.Type.Params.List {
		if _, ok := f.Type.(*ast.Ellipsis); ok {
			variadic = true
		}
		if len(f.Names) == 0 {
			names = append(names, nil)
		}
		names = append(names, f.Names...)
	}
	if idx < 0 || idx >= len(names) || variadic && idx >= len(names)-1 || call.Ellipsis.IsValid() {
		t.escaped = true
		return
	}
	if names[idx] == nil || names[idx].Name == "_" {
		return // the helper ignores it
	}
	add(h.key(names[idx]))
}

// stock names the stock Huh theme an argument to WithTheme is, or "".
func (h *huhPass) stock(e ast.Expr) string {
	e = ast.Unparen(e)
	if c, ok := e.(*ast.CallExpr); ok {
		r := h.p.Ref(c.Fun)
		switch {
		case (in(r.Pkg, huhPaths) || in(r.Pkg, huhSpinnerPaths)) && r.Name == "ThemeFunc" && len(c.Args) == 1:
			e = ast.Unparen(c.Args[0]) // huh.ThemeFunc(huh.ThemeCharm)
		case (in(r.Pkg, huhPaths) || in(r.Pkg, huhSpinnerPaths)) && huhStock[r.Name]:
			e = c.Fun // v1: huh.ThemeCharm()
		}
	}
	r := h.p.Ref(e)
	if (in(r.Pkg, huhPaths) || in(r.Pkg, huhSpinnerPaths)) && huhStock[r.Name] && r.Recv == "" {
		pkg := "huh."
		if in(r.Pkg, huhSpinnerPaths) {
			pkg = "spinner."
		}
		return pkg + r.Name
	}
	return ""
}
