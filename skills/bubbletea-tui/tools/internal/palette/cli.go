package palette

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

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
	// theme.CheckKeys: a role, glyph or wordmark key the kit doesn't read ("primay",
	// "focusbar") is dropped by theme.Load and the role falls back to the derived colour
	// with nothing to say why; this is the gate, as for an expressions key below
	if err := theme.CheckKeys(data); err != nil {
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
		if d, near := NearPrimary(r); near {
			fmt.Fprintf(stderr, "palette: warning: %s: %s\n", path, nearPrimaryText(mode, r, d))
		}
	}
	clashes := Clashes16(tf, path, stderr)
	for _, x := range FarPins(tf) {
		fmt.Fprintf(stderr, "palette: warning: %s: %s\n", path, farPinText(x))
	}
	if clashes > 0 {
		fmt.Fprintf(stdout, "FAIL (%d contrast problems, %d problems at 16 colours)\n", bad, clashes)
		return 1
	}
	fmt.Fprintln(stdout, verdict(bad))
	if bad > 0 {
		return 1
	}
	return 0
}

// Finding16 is one thing a 16-colour terminal reads wrong in a theme, after its ansi16
// pins, with the pin that fixes it (Findings16).
type Finding16 struct {
	Mode, Role string
	Col        string // the role as the theme writes it, truecolour hex
	What       string // the finding, without the path or the pin
	Fail       bool   // check exits 1 on it
	Pin        int    // the ansi16 colour that fixes it
	kind       kind16
}

type kind16 int

const (
	red16   kind16 = iota // a state role on red, or error off it (theme.ANSI16Red)
	clash16               // two state roles on one colour (theme.ANSI16Clashes)
	hue16                 // a neutral on a hue (theme.ANSI16Neutrals)
	share16               // two levels of the text ramp on one grey (theme.ANSI16Neutrals)
	evict16               // primary on a status role's own slot while that role needs one
)

// PinJSON is the ansi16 block that applies the finding's pin.
func (x Finding16) PinJSON() string {
	return fmt.Sprintf(`"ansi16": {%q: {%q: %d}}`, x.Mode, x.Role, x.Pin)
}

// Findings16 is how the theme reads at 16 colours, after its ansi16 pins, each finding
// with the pin that fixes it:
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
//   - a neutral role on a hue (theme.ANSI16Neutrals): the engine puts every neutral on a
//     grey, the roles a dialog draws (border, borderFocus, bgPanel, bgOverlay) included,
//     so only a pin moves one off. On red it fails (hints read as errors, a dialog's edge
//     as an error's); on another hue it warns, except borderFocus pinned onto primary's
//     own slot, the kit's truecolour default.
//   - two levels of the text ramp on one grey (theme.ANSI16Neutrals): a warning; the pin
//     is the grey the engine would have used.
//   - primary sitting on the own slot of a status role that needs a pin (a yellow brand
//     natively on 3 while warning is on red): a warning that moves primary to the nearest
//     free hue, so warning takes its yellow instead of being sent to cyan.
//
// Which slot reads well on the page is the terminal's palette, so a pin is a suggestion:
// a status role to its own slot, else the nearest free hue, never red for a non-error,
// never a colour another suggestion took nor the other brightness of one another state
// role draws (10 beside success's 2 is a different slot and the same reading); a neutral
// to its grey (theme.GreySlot16). Failures come first, then primary, then the status
// roles (PinOrder): a brand whose amber and warning both land on red keeps the yellow
// nearest its hue and the warning moves, since a brand drawn green beside success's
// green reads as a success. Only a brand natively on a status role's own slot (the
// yellow brand above) yields it, in both modes, and the terminal's own order holds
// (keepRank: error, success, warning, then the brand's). Each suggestion is taken before
// the next, so following them all never makes a new problem.
func Findings16(f theme.File) []Finding16 {
	type key struct{ mode, role string }
	fg := f.Expressions.Focus == "fg"
	var found []Finding16
	covered := map[key]bool{}
	for _, r := range theme.ANSI16Red(f) {
		covered[key{r.Mode, r.Role}] = true
		what := fmt.Sprintf("%s: %s %s draws as ANSI %d at 16 colours (-profile ansi), error's red, so it reads as an error there", r.Mode, r.Role, hexOf(r.Col), r.Index)
		switch {
		case r.Role == "error":
			what = fmt.Sprintf("%s: error %s draws as ANSI %d at 16 colours (-profile ansi), not red (1 or 9), so a red state beside it reads as the error", r.Mode, hexOf(r.Col), r.Index)
		case r.Role == "primary" && !fg:
			what = fmt.Sprintf("%s: primary %s draws as ANSI %d at 16 colours (-profile ansi), error's red, so focus and selection read as errors there", r.Mode, hexOf(r.Col), r.Index)
		}
		found = append(found, Finding16{Mode: r.Mode, Role: r.Role, Col: hexOf(r.Col), What: what, Fail: r.Role == "error" || (r.Role == "primary" && fg), kind: red16})
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
		found = append(found, Finding16{Mode: k.mode, Role: k.role, Col: col[k], What: what, Fail: fg && withPrimary, kind: clash16})
	}
	for _, n := range theme.ANSI16Neutrals(f) {
		if n.Role == "borderFocus" && n.Index == theme.ANSI16Index(f, n.Mode, "primary") && n.Index != 1 && n.Index != 9 {
			continue // the dialog's edge in the brand's own hue is the kit's truecolour default, pinned on purpose
		}
		x := Finding16{Mode: n.Mode, Role: n.Role, Col: hexOf(n.Col), kind: hue16}
		switch {
		case n.With != "":
			x.kind = share16
			x.What = fmt.Sprintf("%s: %s %s draws as ANSI %d at 16 colours (-profile ansi), as %s %s does, so %s reads as %s there",
				n.Mode, n.Role, x.Col, n.Index, n.With, roleHex(f, n.Mode, n.With), neutralText[n.Role], neutralText[n.With])
		case n.Index == 1 || n.Index == 9:
			x.Fail = true
			x.What = fmt.Sprintf("%s: %s %s draws as ANSI %d at 16 colours (-profile ansi), error's red, so %s reads as an error there", n.Mode, n.Role, x.Col, n.Index, neutralText[n.Role])
		default:
			x.What = fmt.Sprintf("%s: %s %s draws as ANSI %d at 16 colours (-profile ansi), a hue, so %s reads as a state there", n.Mode, n.Role, x.Col, n.Index, neutralText[n.Role])
		}
		found = append(found, x)
	}
	// a status role whose own slot (warning 3/11, success 2/10, info 4/12) is held by
	// primary's native index: evict primary, since the terminal's palette owns that
	// meaning, rather than send the status role to the nearest free hue
	native := f
	native.ANSI16 = nil
	has := map[key]bool{}
	for _, x := range found {
		has[key{x.Mode, x.Role}] = true
	}
	for _, x := range found {
		own, isStatus := ownSlot[x.Role]
		if !isStatus || x.kind == hue16 || x.kind == share16 || has[key{x.Mode, "primary"}] {
			continue
		}
		shift := 0
		if x.Mode != "light" {
			shift = 8
		}
		p := theme.ANSI16Index(native, x.Mode, "primary") // where the brand lands by itself: a pin onto the slot is the theme's choice and holds
		for _, i := range own {
			if p == i+shift || p == i {
				tc := theme.Resolve(f, x.Mode != "light", false).P
				found = append(found, Finding16{Mode: x.Mode, Role: "primary", Col: hexOf(tc.Primary), kind: evict16,
					What: fmt.Sprintf("%s: primary %s draws as ANSI %d at 16 colours (-profile ansi), %s's own slot, which %s needs", x.Mode, hexOf(tc.Primary), p, x.Role, x.Role)})
				has[key{x.Mode, "primary"}] = true
				break
			}
		}
	}
	// failures first, then PinOrder: primary before the status roles when a slot near
	// its hue exists, unless the brand sits natively on one's own slot (evict16) and
	// yields it
	evicted := slices.ContainsFunc(found, func(x Finding16) bool { return x.kind == evict16 })
	rank := func(x Finding16) int {
		if x.kind == hue16 || x.kind == share16 {
			return len(keepRank)
		}
		return PinOrder(f, x.Mode, evicted)[x.Role]
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Fail != found[j].Fail {
			return found[i].Fail
		}
		return rank(found[i]) < rank(found[j])
	})
	// a slot is taken by red, and by every state role that stays where it is; a role
	// with a finding is being moved, so its slot is free for the next
	used := map[string]map[int]bool{}
	for i, x := range found {
		if used[x.Mode] == nil {
			used[x.Mode] = map[int]bool{1: true, 9: true} // red is error's alone
			for _, r := range theme.StateRoles {
				if !has[key{x.Mode, r}] {
					used[x.Mode][theme.ANSI16Index(f, x.Mode, r)] = true
				}
			}
		}
		dark := x.Mode != "light"
		switch {
		case x.kind == hue16 || x.kind == share16:
			found[i].Pin = greyPin(f, x.Mode, x.Role)
			continue
		case x.Role == "error":
			found[i].Pin = 1
			if dark {
				found[i].Pin = 9
			}
		default:
			found[i].Pin = Pin16(x.Role, x.Col, dark, used[x.Mode])
		}
		used[x.Mode][found[i].Pin] = true
	}
	return found
}

// neutralText is what each neutral role carries on screen, for a finding's sentence
// (singular, so "reads" follows each).
var neutralText = map[string]string{
	"bg": "the page", "bgPanel": "a panel", "bgSubtle": "the background step", "bgOverlay": "an overlay", "border": "every rule", "borderFocus": "the dialog's edge",
	"fg": "body text", "fgSoft": "prose", "fgMuted": "secondary text", "fgSubtle": "a hint",
}

// roleHex is a text ramp role of mode as the theme draws it at truecolour.
func roleHex(f theme.File, mode, role string) string {
	p := theme.Resolve(f, mode != "light", false).P
	return hexOf(map[string]color.Color{"fg": p.Fg, "fgMuted": p.FgMuted, "fgSubtle": p.FgSubtle}[role])
}

// greyPin is the grey to pin neutral role to in mode: its own (theme.GreySlot16), unless
// another level of the text ramp already draws there; then the other text grey, then
// fg's. The page's grey is never offered for text.
func greyPin(f theme.File, mode, role string) int {
	dark := mode != "light"
	own := theme.GreySlot16(role, dark)
	if !slices.Contains(theme.TextRamp, role) {
		return own
	}
	taken := map[int]bool{theme.GreySlot16("bg", dark): true}
	for _, other := range theme.TextRamp {
		if other != role {
			taken[theme.ANSI16Index(f, mode, other)] = true
		}
	}
	for _, n := range []int{own, theme.GreySlot16("fgMuted", dark), theme.GreySlot16("fgSubtle", dark), theme.GreySlot16("fg", dark)} {
		if !taken[n] {
			return n
		}
	}
	return own
}

// Clashes16 prints Findings16 on stderr, each with its pin, and returns how many fail.
func Clashes16(f theme.File, path string, stderr io.Writer) int {
	failed := 0
	for _, x := range Findings16(f) {
		pin := x.PinJSON()
		switch {
		case x.Fail && x.kind == clash16:
			failed++
			fmt.Fprintf(stderr, "palette: %s: %s; under focus \"fg\" primary is a state, so they read as one: pin %s with %s\n", path, x.What, x.Role, pin)
		case x.Fail:
			failed++
			fmt.Fprintf(stderr, "palette: %s: %s: pin it with %s\n", path, x.What, pin)
		case x.kind == clash16:
			fmt.Fprintf(stderr, "palette: warning: %s: %s, so those states differ there only by glyph and word; %s separates them\n", path, x.What, pin)
		case x.kind == share16:
			fmt.Fprintf(stderr, "palette: warning: %s: %s; %s separates them\n", path, x.What, pin)
		case x.kind == evict16:
			fmt.Fprintf(stderr, "palette: warning: %s: %s; %s moves primary off it\n", path, x.What, pin)
		default:
			fmt.Fprintf(stderr, "palette: warning: %s: %s; %s moves it\n", path, x.What, pin)
		}
	}
	return failed
}

// Pins16 are the ansi16 pins Findings16 recommends for f, applied until a 16-colour
// terminal reads it right (each round's pins are taken before the next looks again), on
// top of the pins f already has. nil when f needs none. palette derive writes them into
// its output, so its first check prints nothing.
func Pins16(f theme.File) map[string]map[string]int {
	pins := map[string]map[string]int{}
	for mode, m := range f.ANSI16 {
		pins[mode] = map[string]int{}
		for role, n := range m {
			pins[mode][role] = n
		}
	}
	added := false
	for range 4 {
		f.ANSI16 = pins
		found := Findings16(f)
		if len(found) == 0 {
			break
		}
		for _, x := range found {
			if pins[x.Mode] == nil {
				pins[x.Mode] = map[string]int{}
			}
			pins[x.Mode][x.Role] = x.Pin
			added = true
		}
	}
	if !added {
		return nil
	}
	return pins
}

// keepRank orders the state roles by how firmly the 16 colours own them: error is red
// everywhere, success green, warning yellow; primary and info are the brand's and
// move first. It decides which of a clashing pair moves, and the pin order when the
// brand yields a status role's own slot (PinOrder).
var keepRank = map[string]int{"error": 0, "success": 1, "warning": 2, "primary": 3, "info": 4}

// primaryFirst is the pin order when the brand is displaced too: after error, primary
// takes the free slot nearest its hue before the status roles take theirs, so an amber
// brand and its amber warning, both off red, leave the brand on yellow and move the
// warning, not the brand onto green or magenta.
var primaryFirst = map[string]int{"error": 0, "primary": 1, "success": 2, "warning": 3, "info": 4}

// PinOrder is the order the state roles of mode take their pins in: primaryFirst when
// a slot within farPinMax of the brand's hue exists (an amber, 36° from yellow), since
// the brand drawn in a far hue beside that slot's own role reads as that role; keepRank
// when none does (an orange, 71° from yellow and from magenta alike: the status roles
// keep their own slots and the brand takes the nearest free), when the brand has no
// hue, or when it sits natively on a status role's own slot (evicted) and yields it.
func PinOrder(f theme.File, mode string, evicted bool) map[string]int {
	if evicted || f.Roles[mode] == (theme.Roles{}) {
		return keepRank
	}
	_, c, h := oklch.LCH(hexOf(theme.Resolve(f, mode != "light", false).P.Primary))
	if c < 0.04 {
		return keepRank
	}
	for n := 2; n <= 6; n++ { // the hues that are not red: green, yellow, blue, magenta, cyan
		_, _, hs := oklch.LCH(hexOf(ansi.BasicColor(n)))
		if hueDiff(h, hs) <= farPinMax {
			return primaryFirst
		}
	}
	return keepRank
}

// FarPin is a pin that moves a role far from its own hue while a slot nearer it is
// free: the role reads as another state there (FarPins).
type FarPin struct {
	Mode, Role string
	Col        string  // the role as the theme writes it, truecolour hex
	Hue        float64 // its OKLCH hue
	Pin        int     // the pin it has
	Moved      float64 // degrees of hue between the two
	Nearer     int     // the pin the tool would give it (Pins16, with this pin and the later roles' lifted)
	Yielded    string  // the role whose pin held Nearer, "" when none
}

// farPinMax is how far a pin may move a role's hue before FarPins reports it: ANSI's
// six hues are 60° apart, so a move past that lands nearer another hue than the role's.
const farPinMax = 60

// FarPins are the theme's ansi16 pins that move a role with a hue (chroma 0.04 or
// more) more than farPinMax degrees of OKLCH hue from its truecolour value, when the
// tool itself would pin it nearer: Pins16 with that pin lifted, and with the pins of
// the roles after it in PinOrder lifted too, so a brand whose yellow is held by its
// warning's pin is told the yellow is its (derive gives it that way). A pin that is the
// nearest free hue (an orange brand on magenta, with red error's and yellow warning's)
// is not reported, however far: nothing nearer is free. Greys and neutrals carry no hue.
func FarPins(f theme.File) []FarPin {
	var out []FarPin
	if len(f.ANSI16) == 0 {
		return nil
	}
	unpinned := f
	unpinned.ANSI16 = nil
	evicted := slices.ContainsFunc(Findings16(unpinned), func(x Finding16) bool { return x.kind == evict16 })
	for _, mode := range []string{"dark", "light"} {
		pins := f.ANSI16[mode]
		if len(pins) == 0 || f.Roles[mode] == (theme.Roles{}) {
			continue
		}
		order := PinOrder(f, mode, evicted)
		tc := theme.Resolve(f, mode != "light", false).P
		for _, role := range sortedRoles16(pins) {
			n := pins[role]
			if _, isState := order[role]; !isState || theme.Grey16(n) {
				continue
			}
			col := paletteHex(tc, role)
			_, c, h := oklch.LCH(col)
			if c < 0.04 {
				continue
			}
			_, _, hs := oklch.LCH(hexOf(ansi.BasicColor(n)))
			moved := hueDiff(h, hs)
			if moved <= farPinMax {
				continue
			}
			// the tool's own pin for the role, with this pin and the later roles' lifted
			lifted := f
			lifted.ANSI16 = map[string]map[string]int{}
			for m, mp := range f.ANSI16 {
				lifted.ANSI16[m] = map[string]int{}
				for r, v := range mp {
					if m == mode && (r == role || order[r] > order[role]) {
						continue
					}
					lifted.ANSI16[m][r] = v
				}
			}
			nearer, ok := Pins16(lifted)[mode][role]
			if !ok {
				nearer = theme.ANSI16Index(lifted, mode, role)
			}
			if nearer == n {
				continue
			}
			yielded := ""
			for _, r := range sortedRoles16(pins) {
				if r != role && pins[r] == nearer {
					yielded = r
				}
			}
			out = append(out, FarPin{Mode: mode, Role: role, Col: col, Hue: h, Pin: n, Moved: moved, Nearer: nearer, Yielded: yielded})
		}
	}
	return out
}

// hueDiff is the smaller angle between two hues, in degrees.
func hueDiff(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// slotName is a 16-colour slot's hue as a terminal names it.
func slotName(n int) string {
	names := []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}
	if n >= 8 {
		return "bright " + names[n-8]
	}
	return names[n]
}

// farPinText is the warning for a far pin (FarPins).
func farPinText(x FarPin) string {
	_, _, hn := oklch.LCH(hexOf(ansi.BasicColor(x.Nearer)))
	hue := slotName(x.Pin)
	hue = strings.TrimPrefix(hue, "bright ")
	s := fmt.Sprintf("%s: %s %s is pinned to ANSI %d at 16 colours (-profile ansi), %s, %.0f° of hue from its own %.0f°, so %s reads as %s there; %s is nearer its hue (%s, %.0f° off)",
		x.Mode, x.Role, x.Col, x.Pin, slotName(x.Pin), x.Moved, x.Hue, roleText(x.Role), hue, Finding16{Mode: x.Mode, Role: x.Role, Pin: x.Nearer}.PinJSON(), strings.TrimPrefix(slotName(x.Nearer), "bright "), hueDiff(x.Hue, hn))
	if x.Yielded != "" {
		s += fmt.Sprintf(", with %s moved off it", x.Yielded)
	}
	return s
}

// roleText is what a state role is on screen, for a finding's sentence.
func roleText(role string) string {
	if role == "primary" {
		return "the brand"
	}
	return "a " + role
}

// paletteHex is a state role of a resolved palette as hex.
func paletteHex(p theme.Palette, role string) string {
	switch role {
	case "primary":
		return hexOf(p.Primary)
	case "error":
		return hexOf(p.Error)
	case "warning":
		return hexOf(p.Warning)
	case "success":
		return hexOf(p.Success)
	case "info":
		return hexOf(p.Info)
	}
	return ""
}

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
	pins := Pins16(themeOf(modes))
	js := JSON(name, modes, pins)
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
		if d, near := NearPrimary(m.Roles); near {
			fmt.Fprintf(stderr, "warning: %s\n", nearPrimaryText(m.Name, m.Roles, d))
		}
	}
	if len(pins) > 0 {
		var parts []string
		for _, mode := range []string{"dark", "light"} {
			for _, role := range sortedRoles16(pins[mode]) {
				parts = append(parts, fmt.Sprintf("%s %s %d", mode, role, pins[mode][role]))
			}
		}
		fmt.Fprintf(stderr, "pinned at 16 colours (\"ansi16\", kept in the theme): %s\n", strings.Join(parts, ", "))
		tf := themeOf(modes)
		tf.ANSI16 = pins
		for _, x := range FarPins(tf) {
			fmt.Fprintf(stderr, "warning: %s\n", farPinText(x))
		}
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

// themeOf is the derive document as the engine reads it, for the 16-colour pass.
func themeOf(modes []Mode) theme.File {
	f := theme.File{Roles: map[string]theme.Roles{}}
	for _, m := range modes {
		b, _ := json.Marshal(m.Roles)
		var r theme.Roles
		_ = json.Unmarshal(b, &r)
		f.Roles[m.Name] = r
	}
	return f
}

func sortedRoles16(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// nearPrimaryText is the warning for a warning role beside primary (NearPrimary).
func nearPrimaryText(mode string, r map[string]string, d float64) string {
	return fmt.Sprintf("%s: warning %s sits beside primary %s in hue and chroma (%.3f apart in OKLab a/b, under %.2f), so a warning row reads as a focused one; move warning's hue, or pin a 16-colour slot and accept it in the read",
		mode, r["warning"], r["primary"], d, nearPrimaryMax)
}
