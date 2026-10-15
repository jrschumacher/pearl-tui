package codelint

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/width"
	"golang.org/x/tools/go/analysis"
)

func program(pass *analysis.Pass) *Program { return pass.ResultOf[ModelAnalyzer].(*Program) }

func newAnalyzer(name, doc string, run func(*analysis.Pass) (any, error)) *analysis.Analyzer {
	return &analysis.Analyzer{Name: name, Doc: doc, Run: run, Requires: []*analysis.Analyzer{ModelAnalyzer}, RunDespiteErrors: true}
}

// C-LEN

// LenAnalyzer reports byte or rune counts of text used to size or pad (C-LEN).
var LenAnalyzer = newAnalyzer("tuilen",
	"C-LEN: len() or utf8.RuneCountInString of text used to size or pad; a cell is not a byte or a rune, so use lipgloss.Width",
	runLen)

var (
	widthNameRe = regexp.MustCompile(`(?i)width|cols|columns|pad|gap|indent|margin`)
	widthVarRe  = regexp.MustCompile(`^w\d*$|[a-z0-9]W\d*$`)
	widthSinkRe = regexp.MustCompile(`^(?i:repeat|width|maxwidth|pad\w*|padding\w*|margin\w*|truncate\w*|place\w*|wrap\w*|fit\w*|indent\w*)$`)
	// names tuilint.py took for slices when it could not see types
	sliceNames = map[string]bool{"Banner": true, "top": true, "pre": true, "post": true, "src": true, "grid": true, "m": true, "b": true,
		"buf": true, "data": true, "out": true, "res": true, "parts": true, "args": true, "frames": true}
)

func widthName(s string) bool { return widthNameRe.MatchString(s) || widthVarRe.MatchString(s) }

func runLen(pass *analysis.Pass) (any, error) {
	p := program(pass)
	once := lineOnce{}
	for _, f := range p.Files {
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			stack = append(stack, n)
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			what, arg, unit := p.measures(call)
			if what == "" || !p.widthContext(stack) || !once.first(pass.Fset, "C-LEN", call.Pos()) {
				return true
			}
			if t := pass.TypesInfo.TypeOf(arg); t == nil || t == types.Typ[types.Invalid] {
				unit = "if " + types.ExprString(arg) + " is text, " + unit
			}
			report(pass, call.Pos(), "C-LEN", fmt.Sprintf("%s in width maths: %s, not cells; use lipgloss.Width, and ansi.Truncate to cut it", what, unit))
			return true
		})
	}
	return nil, nil
}

// measures reports whether call counts the bytes or runes of a string: the call as
// written, the string, and what it counts.
func (p *Program) measures(call *ast.CallExpr) (string, ast.Expr, string) {
	info := p.Pass.TypesInfo
	if len(call.Args) != 1 {
		return "", nil, ""
	}
	arg := ast.Unparen(call.Args[0])
	if r := p.Ref(call.Fun); r.Pkg == "unicode/utf8" && (r.Name == "RuneCountInString" || r.Name == "RuneCount") {
		return "utf8." + r.Name + "(" + types.ExprString(arg) + ")", arg, "that counts runes"
	}
	id, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok || id.Name != "len" {
		return "", nil, ""
	}
	if b, ok := info.Uses[id].(*types.Builtin); !ok && info.Uses[id] != nil || ok && b.Name() != "len" {
		return "", nil, ""
	}
	what := "len(" + types.ExprString(arg) + ")"
	if conv, ok := arg.(*ast.CallExpr); ok && len(conv.Args) == 1 {
		if at, ok := conv.Fun.(*ast.ArrayType); ok && at.Len == nil {
			if el, ok := at.Elt.(*ast.Ident); ok && (el.Name == "rune" || el.Name == "byte") {
				return what, conv.Args[0], "that counts " + el.Name + "s"
			}
		}
	}
	t := info.TypeOf(arg)
	if t == nil || t == types.Typ[types.Invalid] {
		// no types: fall back to tuilint.py's guess that plural names are slices
		last := types.ExprString(arg)
		if i := strings.LastIndexAny(last, ".["); i >= 0 {
			last = last[i+1:]
		}
		if strings.HasSuffix(last, "s") || sliceNames[last] {
			return "", nil, ""
		}
		return what, arg, "that counts bytes"
	}
	if b, ok := t.Underlying().(*types.Basic); !ok || b.Info()&types.IsString == 0 || p.asciiOnly(arg) {
		return "", nil, ""
	}
	return what, arg, "that counts bytes"
}

// asciiOnly reports whether a string expression can only hold ASCII: a constant, or a
// number formatted by strconv.
func (p *Program) asciiOnly(e ast.Expr) bool {
	if tv, ok := p.Pass.TypesInfo.Types[e]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		s := constant.StringVal(tv.Value)
		for i := 0; i < len(s); i++ {
			if s[i] >= 0x80 {
				return false
			}
		}
		return true
	}
	if c, ok := e.(*ast.CallExpr); ok {
		r := p.Ref(c.Fun)
		return r.Pkg == "strconv" && (r.Name == "Itoa" || strings.HasPrefix(r.Name, "Format"))
	}
	return false
}

// widthContext reports whether the count at the top of stack feeds layout: it is (through
// arithmetic) padded with, compared to, or assigned to a width, or passed to a sizing call.
func (p *Program) widthContext(stack []ast.Node) bool {
	mentions := func(e ast.Node) bool {
		found := false
		ast.Inspect(e, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && widthName(id.Name) {
				found = true
			}
			return !found
		})
		return found
	}
	cur := stack[len(stack)-1]
	for i := len(stack) - 2; i >= 0; i-- {
		switch x := stack[i].(type) {
		case *ast.ParenExpr, *ast.UnaryExpr:
		case *ast.BinaryExpr:
			other := x.X
			if other == cur {
				other = x.Y
			}
			switch x.Op {
			case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
				if mentions(other) {
					return true
				}
			case token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL, token.NEQ:
				return mentions(other)
			default:
				return false
			}
		case *ast.CallExpr:
			if x.Fun == cur {
				return false
			}
			if tv, ok := p.Pass.TypesInfo.Types[x.Fun]; ok && tv.IsType() || isBuiltin(p.Pass.TypesInfo, x.Fun, "max", "min") {
				for _, a := range x.Args {
					if a != cur && mentions(a) {
						return true
					}
				}
				break
			}
			return widthSinkRe.MatchString(funName(x.Fun))
		case *ast.AssignStmt:
			for j, r := range x.Rhs {
				if r == cur && j < len(x.Lhs) {
					return widthName(types.ExprString(lastName(x.Lhs[j])))
				}
			}
			return false
		case *ast.ValueSpec:
			for j, v := range x.Values {
				if v == cur && j < len(x.Names) {
					return widthName(x.Names[j].Name)
				}
			}
			return false
		case *ast.KeyValueExpr:
			return x.Value == cur && widthName(types.ExprString(x.Key))
		case *ast.ReturnStmt:
			for k := i - 1; k >= 0; k-- {
				switch fn := stack[k].(type) {
				case *ast.FuncLit:
					return false
				case *ast.FuncDecl:
					return widthName(fn.Name.Name)
				}
			}
			return false
		default:
			return false
		}
		cur = stack[i]
	}
	return false
}

func lastName(e ast.Expr) ast.Expr {
	if s, ok := e.(*ast.SelectorExpr); ok {
		return s.Sel
	}
	return e
}

func funName(e ast.Expr) string {
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.IndexExpr:
		return funName(x.X)
	}
	return ""
}

func isBuiltin(info *types.Info, e ast.Expr, names ...string) bool {
	id, ok := ast.Unparen(e).(*ast.Ident)
	if !ok {
		return false
	}
	b, ok := info.Uses[id].(*types.Builtin)
	if !ok {
		return false
	}
	for _, n := range names {
		if b.Name() == n {
			return true
		}
	}
	return false
}

// C-SLEEP, C-VIEWIO

// BlockingAnalyzer reports time.Sleep and blocking I/O on the update goroutine (C-SLEEP,
// C-VIEWIO): anything Update or View reaches through direct calls, but not Cmd bodies,
// Tick callbacks or goroutines, which is where blocking work belongs.
var BlockingAnalyzer = newAnalyzer("tuiblocking",
	"C-SLEEP / C-VIEWIO: time.Sleep or blocking I/O reachable from Update or View; return a tea.Cmd instead",
	runBlocking)

// blockingIO are the calls that wait on the disk, the network or another process.
var blockingIO = map[string]bool{}

func init() {
	for _, s := range strings.Fields(`
		os.ReadFile os.WriteFile os.ReadDir os.Open os.OpenFile os.Create os.Stat os.Lstat os.Mkdir
		os.MkdirAll os.MkdirTemp os.CreateTemp os.Remove os.RemoveAll os.Rename os.Truncate os.Readlink
		os.Symlink os.Link os.Chmod os.Chown os.Chtimes os.CopyFS os.DirFS
		os.File.Read os.File.ReadAt os.File.ReadDir os.File.Readdir os.File.Readdirnames os.File.ReadFrom
		os.File.Write os.File.WriteAt os.File.WriteString os.File.WriteTo os.File.Sync os.File.Stat
		io/ioutil.ReadFile io/ioutil.WriteFile io/ioutil.ReadDir io/ioutil.TempFile io/ioutil.TempDir
		net/http.Get net/http.Post net/http.PostForm net/http.Head
		net/http.Client.Do net/http.Client.Get net/http.Client.Post net/http.Client.PostForm net/http.Client.Head
		net.Dial net.DialTimeout net.LookupHost net.LookupIP net.LookupAddr net.Listen
		os/exec.Cmd.Run os/exec.Cmd.Output os/exec.Cmd.CombinedOutput os/exec.Cmd.Wait`) {
		blockingIO[s] = true
	}
}

func refKey(r Ref) string {
	if r.Recv != "" {
		return r.Pkg + "." + r.Recv + "." + r.Name
	}
	return r.Pkg + "." + r.Name
}

func runBlocking(pass *analysis.Pass) (any, error) {
	p := program(pass)
	seen := map[token.Pos]bool{}
	for _, fd := range Sorted(p.Sync) {
		reach := p.Sync[fd]
		p.Walk(fd.Body, false, func(n ast.Node, async bool) {
			call, ok := n.(*ast.CallExpr)
			if !ok || async || seen[call.Pos()] {
				return
			}
			r := p.Ref(call.Fun)
			sleep := r.Pkg == "time" && r.Name == "Sleep"
			if !sleep && !blockingIO[refKey(r)] {
				return
			}
			seen[call.Pos()] = true
			rule := "C-SLEEP"
			if !sleep && reach.Roots&rootUpdate == 0 {
				rule = "C-VIEWIO"
			}
			report(pass, call.Pos(), rule, fmt.Sprintf("blocking call %s in %s: move it into a tea.Cmd and send the result back as a Msg",
				types.ExprString(call.Fun), reach.Where()))
		})
	}
	return nil, nil
}

// C-HEX

// HexAnalyzer reports colour literals outside the theme package (C-HEX).
var HexAnalyzer = newAnalyzer("tuihex",
	"C-HEX: lipgloss.Color literal or \"#rrggbb\" outside the theme package; screens take roles (t.Primary, t.FgMuted)",
	runHex)

var themePkg string

func init() {
	HexAnalyzer.Flags.StringVar(&themePkg, "theme-pkg", "", "path fragment of the theme package (always exempt: theme, style(s), palette, color(s))")
}

const hexMsg = "hex colour outside the theme: screens take roles (t.Primary, t.FgMuted), never literals"

var (
	themeRe = regexp.MustCompile(`(^|/)(theme|styles?|palette|colors?)(/|\.go$|_)`)
	hexRe   = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// relPath is a file's path inside its module, as tuilint.py saw it from the root.
func relPath(pass *analysis.Pass, pos token.Pos) string {
	pkg := pass.Pkg.Path()
	mod := ""
	if pass.Module != nil {
		mod = pass.Module.Path
	}
	switch {
	case pkg == "command-line-arguments" || pkg == mod:
		pkg = ""
	case mod != "":
		pkg = strings.TrimPrefix(pkg, mod+"/")
	}
	return path.Join(pkg, path.Base(strings.ReplaceAll(pass.Fset.File(pos).Name(), "\\", "/")))
}

func isTheme(pass *analysis.Pass, f *ast.File) bool {
	rel := relPath(pass, f.Pos())
	if themePkg != "" && (strings.Contains(rel, themePkg) || strings.Contains(pass.Pkg.Path(), themePkg)) {
		return true
	}
	return themeRe.MatchString(rel)
}

func runHex(pass *analysis.Pass) (any, error) {
	p := program(pass)
	once := lineOnce{}
	for _, f := range p.Files {
		if isTheme(pass, f) {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ImportSpec:
				return false
			case *ast.CallExpr:
				r := p.Ref(x.Fun)
				if !in(r.Pkg, lipglossPaths) || r.Name != "Color" || len(x.Args) != 1 {
					return true
				}
				lit, ok := ast.Unparen(x.Args[0]).(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				msg := hexMsg
				if v, _ := strconv.Unquote(lit.Value); !strings.HasPrefix(v, "#") {
					msg = "colour literal lipgloss.Color(" + lit.Value + ") outside the theme: screens take roles (t.Primary, t.FgMuted), never literals"
				}
				if once.first(pass.Fset, "C-HEX", x.Pos()) {
					report(pass, x.Pos(), "C-HEX", msg)
				}
			case *ast.BasicLit:
				if x.Kind != token.STRING {
					return true
				}
				if v, err := strconv.Unquote(x.Value); err == nil && hexRe.MatchString(v) && once.first(pass.Fset, "C-HEX", x.Pos()) {
					report(pass, x.Pos(), "C-HEX", hexMsg)
				}
			}
			return true
		})
	}
	return nil, nil
}

// C-EMOJI

// EmojiAnalyzer reports emoji in string and rune literals (C-EMOJI).
var EmojiAnalyzer = newAnalyzer("tuiemoji",
	"C-EMOJI: emoji in a string or rune literal; emoji width varies by terminal, so use the brand glyph set",
	runEmoji)

// emojiOK are dingbats that render as one-cell text glyphs in most mono fonts.
var emojiOK = map[rune]bool{}

func init() {
	for _, r := range "✓✔✗✘★☆☐☑☒✦✧✱✳❯❮➜➤⚠" {
		emojiOK[r] = true
	}
}

// firstEmoji returns the first emoji in s: a wide pictograph, a flag letter, or any glyph
// asking for emoji presentation with U+FE0F.
func firstEmoji(s string) (string, bool) {
	rs := []rune(s)
	for i, r := range rs {
		if i+1 < len(rs) && rs[i+1] == 0xFE0F && !(r == ' ' || r == '\n') {
			return string(rs[i : i+2]), true
		}
		if r >= 0x1F1E6 && r <= 0x1F1FF {
			return string(r), true
		}
		if emojiOK[r] {
			continue
		}
		block := r >= 0x1F000 && r <= 0x1FAFF || r >= 0x2600 && r <= 0x27BF || r >= 0x2300 && r <= 0x23FF ||
			r >= 0x2B00 && r <= 0x2BFF || r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299
		if k := width.LookupRune(r).Kind(); block && (k == width.EastAsianWide || k == width.EastAsianFullwidth) {
			return string(r), true
		}
	}
	return "", false
}

func runEmoji(pass *analysis.Pass) (any, error) {
	p := program(pass)
	once := lineOnce{}
	for _, f := range p.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING && lit.Kind != token.CHAR {
				return true
			}
			v := lit.Value
			if lit.Kind == token.STRING {
				v, _ = strconv.Unquote(lit.Value)
			} else if r, _, _, err := strconv.UnquoteChar(strings.Trim(lit.Value, "'"), '\''); err == nil {
				v = string(r)
			}
			if em, ok := firstEmoji(v); ok && once.first(pass.Fset, "C-EMOJI", lit.Pos()) {
				report(pass, lit.Pos(), "C-EMOJI", fmt.Sprintf("emoji '%s' in a string: emoji width varies by terminal; use the brand glyph set", em))
			}
			return true
		})
	}
	return nil, nil
}

// C-V1

// V1Analyzer reports v1 import paths of the Charm libraries (C-V1).
var V1Analyzer = newAnalyzer("tuiv1",
	"C-V1: Bubble Tea / Lip Gloss / Bubbles / Huh v1 import; new code uses charm.land/*/v2",
	runV1)

var v1Re = regexp.MustCompile(`^github\.com/charmbracelet/(bubbletea|lipgloss|bubbles|huh)(/.*)?$`)

func runV1(pass *analysis.Pass) (any, error) {
	for _, f := range program(pass).Files {
		for _, s := range f.Imports {
			if ip, _ := strconv.Unquote(s.Path.Value); v1Re.MatchString(ip) {
				report(pass, s.Pos(), "C-V1", "v1 import: new code uses charm.land/bubbletea/v2, charm.land/lipgloss/v2, charm.land/bubbles/v2")
			}
		}
	}
	return nil, nil
}

// C-FASTTICK

// TickAnalyzer reports tea.Tick and tea.Every faster than 33ms (C-FASTTICK).
var TickAnalyzer = newAnalyzer("tuitick",
	"C-FASTTICK: tea.Tick/Every faster than 33ms; 30-80ms is plenty for motion and idle screens should not tick",
	runTick)

func runTick(pass *analysis.Pass) (any, error) {
	p := program(pass)
	for _, f := range p.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			if r := p.Ref(call.Fun); !in(r.Pkg, teaPaths) || r.Name != "Tick" && r.Name != "Every" {
				return true
			}
			d, ok := p.duration(call.Args[0])
			if !ok || d <= 0 || d >= 33*time.Millisecond {
				return true
			}
			ms := fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
			if d%time.Millisecond == 0 {
				ms = fmt.Sprintf("%dms", d/time.Millisecond)
			}
			report(pass, call.Pos(), "C-FASTTICK", ms+" tick: 30–80ms is plenty for motion; idle screens should not tick at all")
			return true
		})
	}
	return nil, nil
}

// duration evaluates a constant time.Duration, from types or, without them, from syntax.
func (p *Program) duration(e ast.Expr) (time.Duration, bool) {
	if tv, ok := p.Pass.TypesInfo.Types[e]; ok && tv.Value != nil {
		if v, ok := constant.Int64Val(constant.ToInt(tv.Value)); ok {
			return time.Duration(v), true
		}
		return 0, false
	}
	units := map[string]time.Duration{"Nanosecond": time.Nanosecond, "Microsecond": time.Microsecond,
		"Millisecond": time.Millisecond, "Second": time.Second, "Minute": time.Minute, "Hour": time.Hour}
	switch x := ast.Unparen(e).(type) {
	case *ast.BasicLit:
		if x.Kind == token.INT {
			v, err := strconv.ParseInt(x.Value, 0, 64)
			return time.Duration(v), err == nil
		}
	case *ast.SelectorExpr:
		if r := p.Ref(x); r.Pkg == "time" && units[r.Name] != 0 {
			return units[r.Name], true
		}
	case *ast.CallExpr:
		if r := p.Ref(x.Fun); r.Pkg == "time" && r.Name == "Duration" && len(x.Args) == 1 {
			return p.duration(x.Args[0])
		}
	case *ast.BinaryExpr:
		a, ok1 := p.duration(x.X)
		b, ok2 := p.duration(x.Y)
		if !ok1 || !ok2 {
			return 0, false
		}
		switch x.Op {
		case token.MUL:
			return a * b, true
		case token.QUO:
			if b != 0 {
				return a / b, true
			}
		case token.ADD:
			return a + b, true
		case token.SUB:
			return a - b, true
		}
	}
	return 0, false
}

// C-PRINT

// PrintAnalyzer reports printing to the terminal from a running model (C-PRINT): anything
// Init, Update or View reach, including the Cmds and callbacks they start.
var PrintAnalyzer = newAnalyzer("tuiprint",
	"C-PRINT: fmt.Print* while the program runs corrupts the frame; use tea.Println, a log file, or tea.LogToFile",
	runPrint)

func runPrint(pass *analysis.Pass) (any, error) {
	p := program(pass)
	seen := map[token.Pos]bool{}
	for _, fd := range Sorted(p.All) {
		reach := p.All[fd]
		p.Walk(fd.Body, false, func(n ast.Node, _ bool) {
			call, ok := n.(*ast.CallExpr)
			if !ok || seen[call.Pos()] {
				return
			}
			r := p.Ref(call.Fun)
			name := ""
			switch {
			case r.Pkg == "fmt" && (r.Name == "Print" || r.Name == "Printf" || r.Name == "Println"):
				name = "fmt." + r.Name
			case r.Pkg == "fmt" && strings.HasPrefix(r.Name, "Fprint") && len(call.Args) > 0:
				if w := p.Ref(call.Args[0]); w.Pkg == "os" && (w.Name == "Stdout" || w.Name == "Stderr") {
					name = "fmt." + r.Name + "(os." + w.Name + ")"
				}
			case isBuiltin(pass.TypesInfo, call.Fun, "print", "println"):
				name = funName(call.Fun)
			}
			if name == "" {
				return
			}
			seen[call.Pos()] = true
			report(pass, call.Pos(), "C-PRINT", fmt.Sprintf("%s in %s corrupts the frame: tea.Println, a log file, or tea.LogToFile", name, reach.Where()))
		})
	}
	return nil, nil
}
