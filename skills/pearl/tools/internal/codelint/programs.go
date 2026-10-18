package codelint

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// The program-wide rules ask whether something happens anywhere in the program. A package
// fact carries the answer up the import graph, within the module: a package has it if it
// or a package of its module that it imports has it.

type moduleFact interface {
	analysis.Fact
	module() string
	setModule(string)
}

type sizeFact struct{ Module string }

func (*sizeFact) AFact()                {}
func (*sizeFact) String() string        { return "handles tea.WindowSizeMsg" }
func (f *sizeFact) module() string      { return f.Module }
func (f *sizeFact) setModule(m string)  { f.Module = m }
func (*colorFact) AFact()               {}
func (*colorFact) String() string       { return "handles colour profile" }
func (f *colorFact) module() string     { return f.Module }
func (f *colorFact) setModule(m string) { f.Module = m }
func (*keysFact) AFact()                {}
func (*keysFact) String() string        { return "uses key.Binding" }
func (f *keysFact) module() string      { return f.Module }
func (f *keysFact) setModule(m string)  { f.Module = m }

type colorFact struct{ Module string }

type keysFact struct{ Module string }

// aware reports (and exports as a fact) whether the package, or a package of the same
// module it imports, has the property.
func aware(pass *analysis.Pass, local bool, mk func() moduleFact) bool {
	mod := ""
	if pass.Module != nil {
		mod = pass.Module.Path
	}
	ok := local
	for _, imp := range pass.Pkg.Imports() {
		if ok {
			break
		}
		f := mk()
		ok = pass.ImportPackageFact(imp, f) && f.module() == mod
	}
	if ok {
		f := mk()
		f.setModule(mod)
		pass.ExportPackageFact(f)
	}
	return ok
}

// refs calls fn for every qualified name in the package's non-test files.
func (p *Program) refs(fn func(Ref)) {
	for _, f := range p.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				if r := p.Ref(s); r.Pkg != "" {
					fn(r)
				}
			}
			return true
		})
	}
}

// C-SIZE

// SizeAnalyzer reports programs whose code never handles tea.WindowSizeMsg (C-SIZE), at
// the tea.NewProgram call.
var SizeAnalyzer = &analysis.Analyzer{
	Name:             "tuisize",
	Doc:              "C-SIZE: a program that never handles tea.WindowSizeMsg cannot lay out for the terminal it is in",
	Requires:         []*analysis.Analyzer{ModelAnalyzer},
	Run:              runSize,
	FactTypes:        []analysis.Fact{new(sizeFact)},
	RunDespiteErrors: true,
}

func runSize(pass *analysis.Pass) (any, error) {
	p := program(pass)
	local := false
	p.refs(func(r Ref) { local = local || in(r.Pkg, teaPaths) && r.Name == "WindowSizeMsg" })
	if aware(pass, local, func() moduleFact { return new(sizeFact) }) {
		return nil, nil
	}
	for _, c := range p.NewPrograms {
		report(pass, c.Pos(), "C-SIZE", "no tea.WindowSizeMsg handled anywhere: layout cannot respond to the terminal")
	}
	return nil, nil
}

// C-NOCOLOR

// NoColorAnalyzer reports programs with no colour-profile, background or NO_COLOR handling
// (C-NOCOLOR), at the tea.NewProgram call.
var NoColorAnalyzer = &analysis.Analyzer{
	Name:             "tuinocolor",
	Doc:              "C-NOCOLOR: no colour-profile / dark-background / NO_COLOR handling anywhere in the program",
	Requires:         []*analysis.Analyzer{ModelAnalyzer},
	Run:              runNoColor,
	FactTypes:        []analysis.Fact{new(colorFact)},
	RunDespiteErrors: true,
}

var (
	teaColor      = map[string]bool{"BackgroundColorMsg": true, "ColorProfileMsg": true, "RequestBackgroundColor": true, "WithColorProfile": true}
	lipglossColor = map[string]bool{"HasDarkBackground": true, "LightDark": true, "Complete": true, "CompleteColor": true, "AdaptiveColor": true,
		"CompleteAdaptiveColor": true, "BackgroundColor": true, "ColorProfile": true, "SetColorProfile": true, "SetHasDarkBackground": true}
)

func colorEvidence(r Ref) bool {
	switch {
	case r.Pkg == "github.com/charmbracelet/colorprofile", r.Pkg == "github.com/muesli/termenv", strings.HasSuffix(r.Pkg, "lipgloss/v2/compat"):
		return true
	case in(r.Pkg, teaPaths):
		return teaColor[r.Name]
	case in(r.Pkg, lipglossPaths):
		return lipglossColor[r.Name] && r.Recv == ""
	}
	return false
}

func runNoColor(pass *analysis.Pass) (any, error) {
	p := program(pass)
	local := false
	p.refs(func(r Ref) { local = local || colorEvidence(r) })
	for _, f := range p.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if v, _ := strconv.Unquote(lit.Value); v == "NO_COLOR" {
					local = true
				}
			}
			return !local
		})
	}
	if aware(pass, local, func() moduleFact { return new(colorFact) }) {
		return nil, nil
	}
	for _, c := range p.NewPrograms {
		report(pass, c.Pos(), "C-NOCOLOR", "no colour-profile / dark-background handling found: check light terminals, 256 and 16 colours, NO_COLOR")
	}
	return nil, nil
}

// C-KEYS

// KeysAnalyzer reports key strings matched inline when the program has no key.Binding
// (C-KEYS): with no bindings there is no help view that can stay true.
var KeysAnalyzer = &analysis.Analyzer{
	Name:             "tuikeys",
	Doc:              "C-KEYS: key strings matched inline and no key.Binding in the program; the help view will drift from reality",
	Requires:         []*analysis.Analyzer{ModelAnalyzer},
	Run:              runKeys,
	FactTypes:        []analysis.Fact{new(keysFact)},
	RunDespiteErrors: true,
}

// keyLike are the strings tuilint.py took for key names; used only without types.
var keyLike = regexp.MustCompile(`^((ctrl|alt|shift)\+.+|.|enter|esc|tab|up|down|left|right|space|backspace|delete|home|end|pgup|pgdown)$`)

func runKeys(pass *analysis.Pass) (any, error) {
	p := program(pass)
	local := false
	p.refs(func(r Ref) { local = local || in(r.Pkg, keyPaths) })
	if aware(pass, local, func() moduleFact { return new(keysFact) }) {
		return nil, nil
	}
	var hits []token.Pos
	for _, f := range p.Files {
		hits = append(hits, p.inlineKeys(f)...)
	}
	if len(hits) == 0 {
		return nil, nil
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i] < hits[j] })
	report(pass, hits[0], "C-KEYS", fmt.Sprintf("%d keys matched as raw strings and no key.Binding: the help view will drift from reality", len(hits)))
	return nil, nil
}

// inlineKeys returns the string constants compared with a key's String() in f: switch
// cases and == / != operands.
func (p *Program) inlineKeys(f *ast.File) []token.Pos {
	info := p.Pass.TypesInfo
	defs := map[types.Object]ast.Expr{} // k := msg.String()
	var hits []token.Pos
	// without types, only trust key-like strings inside the update path, as tuilint.py did
	inUpdate := map[ast.Node]bool{}
	for fd, r := range p.Sync {
		if r.Roots&rootUpdate != 0 {
			inUpdate[fd] = true
		}
	}
	var keyString func(e ast.Expr) (bool, bool)
	keyString = func(e ast.Expr) (ok, guessed bool) {
		e = ast.Unparen(e)
		if id, isID := e.(*ast.Ident); isID {
			if rhs := defs[info.Uses[id]]; rhs != nil {
				return keyString(rhs)
			}
			return false, false
		}
		call, isCall := e.(*ast.CallExpr)
		if !isCall || len(call.Args) != 0 {
			return false, false
		}
		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "String" && sel.Sel.Name != "Keystroke" {
			return false, false
		}
		t := info.TypeOf(sel.X)
		if t == nil || t == types.Typ[types.Invalid] {
			return true, true
		}
		if pt, isPtr := t.(*types.Pointer); isPtr {
			t = pt.Elem()
		}
		n, isNamed := types.Unalias(t).(*types.Named)
		if !isNamed || n.Obj().Pkg() == nil || !in(n.Obj().Pkg().Path(), teaPaths) {
			return false, false
		}
		switch n.Obj().Name() {
		case "Key", "KeyMsg", "KeyPressMsg", "KeyReleaseMsg":
			return true, false
		}
		return false, false
	}
	str := func(e ast.Expr, guessed bool) bool {
		v := ""
		if tv, ok := info.Types[e]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
			v = constant.StringVal(tv.Value)
		} else if lit, ok := ast.Unparen(e).(*ast.BasicLit); ok && lit.Kind == token.STRING {
			v, _ = strconv.Unquote(lit.Value)
		} else {
			return false
		}
		return !guessed || keyLike.MatchString(v)
	}
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		stack = append(stack, n)
		trusted := func(guessed bool) bool {
			if !guessed {
				return true
			}
			for _, s := range stack {
				if inUpdate[s] {
					return true
				}
			}
			return false
		}
		switch x := n.(type) {
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE && len(x.Lhs) == 1 && len(x.Rhs) == 1 {
				if id, ok := x.Lhs[0].(*ast.Ident); ok && info.Defs[id] != nil {
					defs[info.Defs[id]] = x.Rhs[0]
				}
			}
		case *ast.SwitchStmt:
			if x.Tag == nil {
				break
			}
			if x.Init != nil {
				ast.Inspect(x.Init, func(m ast.Node) bool {
					if a, ok := m.(*ast.AssignStmt); ok && a.Tok == token.DEFINE && len(a.Lhs) == 1 && len(a.Rhs) == 1 {
						if id, ok := a.Lhs[0].(*ast.Ident); ok && info.Defs[id] != nil {
							defs[info.Defs[id]] = a.Rhs[0]
						}
					}
					return true
				})
			}
			ok, guessed := keyString(x.Tag)
			if !ok || !trusted(guessed) {
				break
			}
			for _, s := range x.Body.List {
				for _, e := range s.(*ast.CaseClause).List {
					if str(e, guessed) {
						hits = append(hits, e.Pos())
					}
				}
			}
		case *ast.BinaryExpr:
			if x.Op != token.EQL && x.Op != token.NEQ {
				break
			}
			for _, pair := range [2][2]ast.Expr{{x.X, x.Y}, {x.Y, x.X}} {
				if ok, guessed := keyString(pair[0]); ok && trusted(guessed) && str(pair[1], guessed) {
					hits = append(hits, pair[1].Pos())
				}
			}
		}
		return true
	})
	return hits
}
