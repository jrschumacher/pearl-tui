package palette

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"tuikit/theme"
	"tuitools/internal/oklch"
	"tuitools/internal/pyfmt"
)

// Main runs the palette command on args (without the program name) and returns its
// exit status: 0 OK, 1 contrast problems, 2 usage or input errors. usage is printed for
// -h and for an unknown mode.
func Main(args []string, usage string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	switch args[0] {
	case "check":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "usage: palette check theme.json")
			return 2
		}
		return check(args[1], stdout, stderr)
	case "derive":
		return derive(args[1:], stdout, stderr)
	}
	fmt.Fprintln(stdout, usage)
	return 2
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "palette:", err)
	return 2
}

// Check reads a theme file (or a bare roles object) and reports every mode's contrast.
func check(path string, stdout, stderr io.Writer) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return fail(stderr, err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return fail(stderr, fmt.Errorf("%s: %w", path, err))
	}
	// theme.CheckGlyphs: a glyph -ascii can't draw, a twin that isn't ASCII, or two
	// states that share a twin. theme.Load lets them through so an app always starts;
	// this is the gate
	if raw, ok := top["glyphs"]; ok {
		var g theme.Glyphs
		if err := json.Unmarshal(raw, &g); err != nil {
			return fail(stderr, fmt.Errorf("%s: glyphs: %w", path, err))
		}
		if err := theme.CheckGlyphs(g); err != nil {
			return fail(stderr, fmt.Errorf("%s: %w", path, err))
		}
	}
	// expressions: a key the kit doesn't read, or a value it doesn't know, renders the
	// default silently (theme.Load loads it so an app starts); this is the gate
	if raw, ok := top["expressions"]; ok {
		if err := checkExpressions(raw); err != nil {
			return fail(stderr, fmt.Errorf("%s: %w", path, err))
		}
	}
	// the theme as the engine reads it, for its ansi16 pins and the 16-colour clashes
	var tf theme.File
	_ = json.Unmarshal(data, &tf) // a shape error surfaces below, with the role that has it
	if _, ok := top["roles"]; !ok {
		tf.Roles = nil
		_ = json.Unmarshal(data, &tf.Roles)
	}
	if err := theme.CheckANSI16(tf); err != nil {
		return fail(stderr, fmt.Errorf("%s: %w", path, err))
	}
	accept := map[string]string{}
	if raw, ok := top["contrastAccept"]; ok {
		if v, err := pyfmt.Decode(raw); err == nil {
			if m, ok := v.(map[string]any); ok {
				for k, why := range m {
					if pyfmt.Truthy(why) {
						accept[k] = pyfmt.Str(why)
					}
				}
			}
		}
	}
	var extra []Check
	var skip map[[2]string]bool
	if raw, ok := top["expressions"]; ok {
		if v, err := pyfmt.Decode(raw); err == nil {
			if m, ok := v.(map[string]any); ok {
				extra, skip = ExpressionChecks(m), ExpressionSkips(m)
			}
		}
	}
	rolesRaw := json.RawMessage(data)
	if raw, ok := top["roles"]; ok {
		rolesRaw = raw
	}
	modes, err := pyfmt.Keys(rolesRaw)
	if err != nil {
		return fail(stderr, fmt.Errorf("%s: roles: %w", path, err))
	}
	var byMode map[string]json.RawMessage
	if err := json.Unmarshal(rolesRaw, &byMode); err != nil {
		return fail(stderr, fmt.Errorf("%s: roles: %w", path, err))
	}
	bad := 0
	for _, mode := range modes {
		v, err := pyfmt.Decode(byMode[mode])
		if err != nil {
			return fail(stderr, err)
		}
		obj, ok := v.(map[string]any)
		if !ok {
			continue
		}
		r := map[string]string{}
		for _, k := range sortedKeys(obj) {
			val := obj[k]
			s, isStr := val.(string)
			if isStr && s == "" {
				continue // derived by the engine, as theme.Load does with a missing role
			}
			if !referenced[k] && !roleKeys[k] {
				if isStr {
					r[k] = s
				}
				continue
			}
			if !isStr {
				return fail(stderr, fmt.Errorf("%s: %s.%s: %s is not a hex colour", path, mode, k, pyfmt.Repr(val)))
			}
			// The engine's rule (theme.Load), not oklch.Parse's looser one: a role Lip
			// Gloss would misread ("6B50FF", "#6B50FFAA") must not pass as OK.
			if err := theme.CheckHex(s); err != nil {
				return fail(stderr, fmt.Errorf("%s: %s.%s: %w", path, mode, k, err))
			}
			r[k] = s
		}
		if err := FillOnColours(r, mode != "light"); err != nil {
			return fail(stderr, err)
		}
		b, txt := ReportSkipping(r, mode, accept, skip, extra...)
		bad += b
		fmt.Fprintln(stdout, txt)
	}
	clashes := Clashes16(tf, path, stderr)
	if clashes > 0 {
		fmt.Fprintf(stdout, "FAIL (%d contrast problems, %d state colour clashes at 16 colours)\n", bad, clashes)
		return 1
	}
	fmt.Fprintln(stdout, verdict(bad))
	if bad > 0 {
		return 1
	}
	return 0
}

// Clashes16 reports on stderr how the theme's state roles read at 16 colours, after its
// ansi16 pins, each with the pin that fixes it, and returns how many fail:
//
//   - error off red (theme.ANSI16Red): a grey error beside a red warning reverses them.
//     Fails.
//   - another state role on red, error's slot: it reads as an error. Fails for primary
//     under focus "fg" (the theme's one state hue); a warning for the rest, primary under
//     focus "primary" included (there focus and selection draw in error's red), since the
//     glyph and the word still differ. The kit's own default warning lands on red, so
//     failing it would fail every theme that leaves warning unset.
//   - two state roles on one colour (theme.ANSI16Clashes): under focus "fg" a clash with
//     primary fails, since the theme promised its hue means one state (amber = new); any
//     other is a warning, as the glyph and the word still tell the two apart.
//
// Which slot reads well on the page is the terminal's palette, so a pin is a suggestion:
// a status role to its own slot, else the nearest free hue, never red for a non-error and
// never a colour another suggestion took.
func Clashes16(f theme.File, path string, stderr io.Writer) int {
	type key struct{ mode, role string }
	type finding struct {
		key
		col, what string
		fail, red bool
	}
	fg := f.Expressions.Focus == "fg"
	var found []finding
	covered := map[key]bool{}
	for _, r := range theme.ANSI16Red(f) {
		k := key{r.Mode, r.Role}
		covered[k] = true
		what := fmt.Sprintf("%s: %s %s draws as ANSI %d at 16 colours (-profile ansi), error's red, so it reads as an error there", r.Mode, r.Role, hexOf(r.Col), r.Index)
		switch {
		case r.Role == "error":
			what = fmt.Sprintf("%s: error %s draws as ANSI %d at 16 colours (-profile ansi), not red (1 or 9), so a red state beside it reads as the error", r.Mode, hexOf(r.Col), r.Index)
		case r.Role == "primary" && !fg:
			what = fmt.Sprintf("%s: primary %s draws as ANSI %d at 16 colours (-profile ansi), error's red, so focus and selection read as errors there", r.Mode, hexOf(r.Col), r.Index)
		}
		found = append(found, finding{k, hexOf(r.Col), what, r.Role == "error" || (r.Role == "primary" && fg), true})
	}
	clash := map[key][]string{}
	col := map[key]string{}
	var order []key
	for _, c := range theme.ANSI16Clashes(f) {
		move, keep, colMove, colKeep := c.B, c.A, c.ColB, c.ColA
		if keepRank[c.B] < keepRank[c.A] {
			move, keep, colMove, colKeep = c.A, c.B, c.ColA, c.ColB
		}
		k := key{c.Mode, move}
		if covered[k] {
			continue // its red finding's pin moves it off this colour too
		}
		if _, seen := clash[k]; !seen {
			order = append(order, k)
			col[k] = hexOf(colMove)
		}
		clash[k] = append(clash[k], keep+" "+hexOf(colKeep))
	}
	for _, k := range order {
		others, verb := strings.Join(clash[k], " and "), "do"
		if len(clash[k]) == 1 {
			verb = "does"
		}
		withPrimary := k.role == "primary" || slices.ContainsFunc(clash[k], func(o string) bool { return strings.HasPrefix(o, "primary ") })
		what := fmt.Sprintf("%s: %s %s draws as ANSI %d at 16 colours (-profile ansi), as %s %s", k.mode, k.role, col[k], theme.ANSI16Index(f, k.mode, k.role), others, verb)
		found = append(found, finding{k, col[k], what, fg && withPrimary, false})
	}
	// failures first, and each suggestion taken before the next, so following them all
	// never makes a new problem
	sort.SliceStable(found, func(i, j int) bool { return found[i].fail && !found[j].fail })
	used := map[string]map[int]bool{}
	failed := 0
	for _, x := range found {
		if used[x.mode] == nil {
			used[x.mode] = map[int]bool{1: true, 9: true} // red is error's alone
			for _, r := range theme.StateRoles {
				used[x.mode][theme.ANSI16Index(f, x.mode, r)] = true
			}
		}
		n := Pin16(x.role, x.col, x.mode != "light", used[x.mode])
		if x.role == "error" {
			n = 1
			if x.mode != "light" {
				n = 9
			}
		}
		used[x.mode][n] = true
		pin := fmt.Sprintf(`"ansi16": {%q: {%q: %d}}`, x.mode, x.role, n)
		switch {
		case x.fail && !x.red:
			failed++
			fmt.Fprintf(stderr, "palette: %s: %s; under focus \"fg\" primary is a state, so they read as one: pin %s with %s\n", path, x.what, x.role, pin)
		case x.fail:
			failed++
			fmt.Fprintf(stderr, "palette: %s: %s: pin it with %s\n", path, x.what, pin)
		case x.red:
			fmt.Fprintf(stderr, "palette: warning: %s: %s; %s moves it\n", path, x.what, pin)
		default:
			fmt.Fprintf(stderr, "palette: warning: %s: %s, so those states differ there only by glyph and word; %s separates them\n", path, x.what, pin)
		}
	}
	return failed
}

// keepRank orders the state roles by how firmly the 16 colours own them: error is red
// everywhere, success green, warning yellow; primary and info are the brand's and
// move first.
var keepRank = map[string]int{"error": 0, "success": 1, "warning": 2, "primary": 3, "info": 4}

// checkExpressions fails an expression key the kit doesn't read, then a value it
// doesn't know (theme.CheckExpressions), each with what it does know.
func checkExpressions(raw json.RawMessage) error {
	keys, err := pyfmt.Keys(raw)
	if err != nil {
		return fmt.Errorf("expressions: %w", err)
	}
	for _, k := range keys {
		if !theme.ExpressionKey(k) {
			return fmt.Errorf("expressions.%s is not a key the kit reads (%s): a typo, or a key from a newer kit, is drawn as if unset", k, strings.Join(exprKeys(), ", "))
		}
	}
	var x theme.Expressions
	if err := json.Unmarshal(raw, &x); err != nil {
		return fmt.Errorf("expressions: %w", err)
	}
	return theme.CheckExpressions(x)
}

// exprKeys are the expression keys the kit reads, in the contract's order.
func exprKeys() []string {
	t := reflect.TypeOf(theme.Expressions{})
	out := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		k, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		out = append(out, k)
	}
	return out
}

// roleKeys are the role names the engine reads (theme.Roles' JSON keys). check
// validates each of them, not only the ones a contrast check reads, so it fails
// whatever theme.Load fails.
var roleKeys = func() map[string]bool {
	m := map[string]bool{}
	t := reflect.TypeOf(theme.Roles{})
	for i := range t.NumField() {
		k, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		m[k] = true
	}
	return m
}()

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func verdict(bad int) string {
	if bad > 0 {
		return fmt.Sprintf("FAIL (%d contrast problems)", bad)
	}
	return "OK (0 contrast problems)"
}

func derive(args []string, stdout, stderr io.Writer) int {
	var pos []string
	opts := map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		if k, v, ok := strings.Cut(a[2:], "="); ok {
			opts[k] = v
			continue
		}
		if i+1 >= len(args) {
			return fail(stderr, fmt.Errorf("%s needs a value", a))
		}
		opts[a[2:]] = args[i+1]
		i++
	}
	if len(pos) == 0 {
		fmt.Fprintln(stderr, `usage: palette derive "#primary" ["#secondary"] [--accent ..] [--bg ..] [--light-bg ..] [--tint 0.012] [--name ..] [--out ..] [--swatch ..]`)
		return 2
	}
	secondary := ""
	if len(pos) > 1 {
		secondary = pos[1]
	}
	in := []struct{ what, hex string }{
		{"primary", pos[0]}, {"secondary", secondary}, {"--accent", opts["accent"]},
		{"--bg", opts["bg"]}, {"--light-bg", opts["light-bg"]},
	}
	for i, c := range in {
		if (i == 0 || c.hex != "") && !oklch.Valid(c.hex) {
			return fail(stderr, fmt.Errorf("%s: %q is not a hex colour", c.what, c.hex))
		}
	}
	tint := 0.012
	if s, ok := opts["tint"]; ok {
		t, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return fail(stderr, fmt.Errorf("--tint: %q is not a number", s))
		}
		tint = t
	}
	name := "brand"
	if s, ok := opts["name"]; ok {
		name = s
	}
	modes := Derive(pos[0], secondary, opts["accent"], opts["bg"], opts["light-bg"], tint)
	js := JSON(name, modes)
	if out, ok := opts["out"]; ok {
		if err := os.WriteFile(out, []byte(js+"\n"), 0o644); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "wrote %s\n", out)
	} else {
		fmt.Fprintln(stdout, js)
	}
	bad := 0
	for _, m := range modes {
		b, txt := Report(m.Roles, m.Name, nil)
		bad += b
		fmt.Fprintln(stderr, txt)
	}
	if sw, ok := opts["swatch"]; ok {
		if err := os.WriteFile(sw, []byte(Swatch(modes)), 0o644); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stderr, "wrote %s (render: cellshot %s swatch.png)\n", sw, sw)
	}
	fmt.Fprintln(stderr, verdict(bad))
	if bad > 0 {
		return 1
	}
	return 0
}
