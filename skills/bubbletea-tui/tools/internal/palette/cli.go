package palette

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
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
	fmt.Fprintln(stdout, verdict(bad))
	if bad > 0 {
		return 1
	}
	return 0
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
