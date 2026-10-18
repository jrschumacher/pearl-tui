package diversity

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tuitools/internal/pyfmt"
)

// SkelAxis is one composition choice of an app's main screen: what a theme file can't
// say, so the app declares it on the `skeleton:` line of its TUI.md.
type SkelAxis struct {
	Key    string
	Values []string // the vocabulary; another word is counted as written, with a note
	Kit    string   // the kit showcase's own value
	// NoneIsDead: "none" here is the absence "-" names, not a choice, so one render
	// gives one count (no chart, or no turns, shared by two apps is not a likeness).
	NoneIsDead bool
}

// SkeletonAxes are the counted composition choices, in table order. Each is one that two
// cold builds of one spec came out sharing while their paint differed (#45): the header
// row, where the session facts sit, which region leads, what the last row holds, where a
// sentence naming the page's state sits, the kit's chart used as it ships, and what ends
// a chat turn.
//
// The kit column is the showcase's services frame at 80×24: a one-row header with the
// context right, the list leading, the failing service's sentence ("4.8% of checkout
// requests failing since 14:02") below the list, keys with "synced 4s ago" at the right,
// one-row sparklines whose values are blocks (the ⣀ of an idle row is a floor, and a
// floor names no family), and the chat screen's turn-ending rule.
//
// Some axes move together (chart and chartGlyphs when an app keeps the kit's chart; head,
// status and foot when it keeps ui.Header and ui.Footer), so a count over them is a
// coarse measure: --max-shared-skeleton 3 is the one integer that separated the pair
// evals, by a margin of 1 on each side, not a measured constant.
var SkeletonAxes = []SkelAxis{
	{"head", []string{"bar", "sentence", "none", "folio"}, "bar", false},
	{"status", []string{"header", "top-row", "bottom-row", "footer", "pane", "none"}, "header", false},
	{"order", []string{"main-first", "summary-first", "input-first", "sidebar-left", "sidebar-right", "single"}, "main-first", false},
	{"foot", []string{"keys", "keys-status", "status", "none"}, "keys-status", false},
	{"verdict", []string{"none", "top", "bottom", "header", "pane"}, "bottom", false},
	{"chart", []string{"sparkline", "trace", "bars", "gauge", "grid", "art", "other"}, "sparkline", true},
	{"chartGlyphs", []string{"blocks", "braille", "shade", "line", "dots", "ascii"}, "blocks", true},
	{"turnEnd", []string{"rule", "marker", "space"}, "rule", true},
}

// Unknown marks an axis the app's TUI.md doesn't declare. Like Dead, it never counts.
const Unknown = "?"

// KitName is the kit's column in the skeleton table: the showcase's layout, which every
// kit theme shares, so it is one column, not seven.
const KitName = "kit"

// skeletonTemplate is the line a note asks for, every axis with its vocabulary.
func skeletonTemplate() string {
	parts := make([]string, len(SkeletonAxes))
	for i, a := range SkeletonAxes {
		parts[i] = a.Key + " <" + strings.Join(a.Values, "|") + pick(a.NoneIsDead, "|-", "") + ">"
	}
	return "skeleton: " + strings.Join(parts, " · ")
}

var (
	skelLine    = regexp.MustCompile(`^\s*(?:[-*]\s+)?skeleton(?:\s*:\s*|\s+)(.*)$`)
	themeHead   = regexp.MustCompile(`(?i)^#{1,6}\s+(?:\d+\.\s*)?theme\b`)
	brandHead   = regexp.MustCompile(`(?i)^#{1,6}\s+(?:\d+\.\s*)?brand\b`)
	anyHead     = regexp.MustCompile(`^(#{1,6})\s`)
	jsonToken   = regexp.MustCompile("[^\\s`'\"(]+\\.json")
	itemSplit   = regexp.MustCompile(`\s*[·;,]\s*`)
	keyValSplit = regexp.MustCompile(`\s*[:=]\s*|\s+`)
)

// LoadApp reads an app: its TUI.md's § Theme names the theme file (relative to the
// TUI.md), which gives the expression axes, and its `skeleton:` line gives the skeleton
// axes. Notes are [P2] lines: a missing line, axis or known word never fails the run,
// and the axis then doesn't count. A TUI.md whose § Theme names no readable theme is an
// input error.
func LoadApp(path string) (Theme, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, nil, err
	}
	sc := scanTUI(data)
	themePath, skel, found := sc.themePath, sc.skel, sc.found
	if themePath == "" {
		return Theme{}, nil, fmt.Errorf("%s: § Theme names no theme file (want a line such as themes/<brand>.json under ### Theme)", path)
	}
	if !filepath.IsAbs(themePath) {
		themePath = filepath.Join(filepath.Dir(path), themePath)
	}
	t, err := Load(themePath)
	if err != nil {
		return Theme{}, nil, fmt.Errorf("%s: § Theme: %w", path, err)
	}
	t.App, t.Source, t.themeFile = true, path, absPath(themePath)
	t.Skeleton = map[string]string{}
	for _, a := range SkeletonAxes {
		t.Skeleton[a.Key] = Unknown
	}
	var notes []string
	note := func(format string, a ...any) {
		notes = append(notes, fmt.Sprintf("[P2] %s: ", path)+fmt.Sprintf(format, a...))
	}
	if !found {
		switch {
		case sc.stray > 0:
			note("the skeleton line at line %d is outside § Brand read, so it isn't read; move it into § Brand read", sc.stray)
		case !sc.brand:
			note("no § Brand read heading, so no skeleton line is read; add the section with the line:\n       %s", skeletonTemplate())
		default:
			note("no skeleton line in § Brand read, so its composition isn't counted; add one, the main screen as planned at brand and corrected once the default frame renders:\n       %s", skeletonTemplate())
		}
		return t, notes, nil
	}
	known, valueless := map[string]SkelAxis{}, map[string]bool{}
	for _, a := range SkeletonAxes {
		known[strings.ToLower(a.Key)] = a
	}
	for _, item := range itemSplit.Split(strings.TrimSpace(skel), -1) {
		if item == "" {
			continue
		}
		f := keyValSplit.Split(item, 3)
		a, ok := known[strings.ToLower(f[0])]
		if !ok {
			note("skeleton names %q, which is no axis (%s)", f[0], axisKeys())
			continue
		}
		if len(f) < 2 || f[1] == "" {
			note("skeleton %s has no value (%s), so it doesn't count", a.Key, strings.Join(a.Values, " | "))
			valueless[a.Key] = true
			continue
		}
		v := strings.ToLower(f[1])
		if v == "n/a" || (v == "none" && a.NoneIsDead) {
			v = Dead
		}
		if t.Skeleton[a.Key] != Unknown {
			note("skeleton %s is declared twice; the first (%s) counts", a.Key, t.Skeleton[a.Key])
			continue
		}
		if v != Dead && !contains(a.Values, v) {
			note("skeleton %s %q is not in its vocabulary (%s, or - where the screen has none); it counts as written, so it matches only itself", a.Key, v, strings.Join(a.Values, " | "))
		}
		t.Skeleton[a.Key] = v
	}
	deadGlyphs(t.Skeleton) // no chart: its glyphs need no declaration
	var missing []string
	for _, a := range SkeletonAxes {
		if t.Skeleton[a.Key] == Unknown && !valueless[a.Key] {
			missing = append(missing, a.Key)
		}
	}
	if len(missing) > 0 {
		note("skeleton declares no %s, so %s (write - where the screen has none)", strings.Join(missing, ", "), pick(len(missing) == 1, "it doesn't count", "they don't count"))
	}
	return t, notes, nil
}

// deadGlyphs: with no chart there is no chart glyph family to share.
func deadGlyphs(s map[string]string) {
	if s["chart"] == Dead {
		s["chartGlyphs"] = Dead
	}
}

// KitSkeleton is the kit showcase's value on every skeleton axis.
func KitSkeleton() map[string]string {
	s := map[string]string{}
	for _, a := range SkeletonAxes {
		s[a.Key] = a.Kit
	}
	deadGlyphs(s)
	return s
}

type tuiScan struct {
	themePath, skel string
	found           bool // a skeleton line in § Brand read
	brand           bool // a § Brand read heading
	stray           int  // the line of the first skeleton line outside § Brand read
}

// scanTUI returns the first .json path under the § Theme heading and the text after the
// first `skeleton:` line in § Brand read (to the next heading as high as its own). In §
// Brand read a fenced block is read too, since brand.md § 1 shows the read in a fence;
// elsewhere a skeleton line is never read, only remembered, so the note can say where it is.
func scanTUI(data []byte) tuiScan {
	var r tuiScan
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	fence, inTheme, brandLevel, n := false, false, 0, 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fence = !fence
			continue
		}
		if m := anyHead.FindStringSubmatch(line); m != nil && !fence {
			inTheme = themeHead.MatchString(line)
			if brandHead.MatchString(line) {
				brandLevel, r.brand = len(m[1]), true
			} else if len(m[1]) <= brandLevel {
				brandLevel = 0
			}
			continue
		}
		if inTheme && !fence && r.themePath == "" {
			if m := jsonToken.FindString(line); m != "" {
				r.themePath = m
			}
		}
		m := skelLine.FindStringSubmatch(line)
		switch {
		case m == nil || r.found:
		case brandLevel > 0:
			r.skel, r.found = m[1], true
		case !fence && r.stray == 0:
			r.stray = n
		}
	}
	return r
}

func axisKeys() string {
	k := make([]string, len(SkeletonAxes))
	for i, a := range SkeletonAxes {
		k[i] = a.Key
	}
	return strings.Join(k, ", ")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func pick(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

func apps(themes []Theme) []Theme {
	var out []Theme
	for _, t := range themes {
		if t.App {
			out = append(out, t)
		}
	}
	return out
}

func live(v string) bool { return v != Dead && v != Unknown }

// SkeletonTable renders the skeleton axes for the kit and each app, as Table does.
func SkeletonTable(themes []Theme) string {
	as := apps(themes)
	if len(as) == 0 {
		return ""
	}
	kit := KitSkeleton()
	var b strings.Builder
	b.WriteString(pyfmt.Ljust("skeleton", 13))
	b.WriteString(pyfmt.Ljust(KitName, 12))
	for _, t := range as {
		b.WriteString(pyfmt.Ljust(pyfmt.Head(t.Name, 10), 12))
	}
	b.WriteByte('\n')
	for _, a := range SkeletonAxes {
		b.WriteString(pyfmt.Ljust(a.Key, 13))
		b.WriteString(pyfmt.Ljust(pyfmt.Head(kit[a.Key], 10), 12))
		for _, t := range as {
			b.WriteString(pyfmt.Ljust(pyfmt.Head(t.Skeleton[a.Key], 10), 12))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// sharedSkeleton lists the skeleton axes two skeletons both declare, alike, and the
// number both declare live.
func sharedSkeleton(s1, s2 map[string]string) (shared []string, both int) {
	for _, a := range SkeletonAxes {
		v1, v2 := s1[a.Key], s2[a.Key]
		if !live(v1) || !live(v2) {
			continue
		}
		both++
		if v1 == v2 {
			shared = append(shared, a.Key)
		}
	}
	return shared, both
}

// FromTheRead follows the skeleton failures and newcomer flags, once: they name the axis,
// never a reason to move it, because in #37 an agent moved an axis only to pass, against
// its own read; and editing a word is cheaper than moving a region.
const FromTheRead = "skeleton failures and flags name an axis, not a reason to move it. Declare the frame as built: " +
	"a line the frame contradicts is the critic's P1.\n" +
	"Move an axis only where the brand read's move: or signature: gives the reason; a value two brands both have " +
	"a reason for: the orchestrator seats it, a lone build keeps it. An axis moved only to pass moves back."

// SkeletonFailures is the rule that needs two apps: two skeletons alike on more than
// --max-shared-skeleton axes are one composition in two paints. Three is the line the
// pair evals drew, with a margin of 1 each side: the pairs a blind critic called one
// product (5 shared) or siblings from one shop (4) fail, the pair it called two products
// (2) passes. A family that shares its chrome on purpose raises it.
func SkeletonFailures(themes []Theme, o Options) []string {
	as := apps(themes)
	var fails []string
	for i, t1 := range as {
		for _, t2 := range as[i+1:] {
			shared, both := sharedSkeleton(t1.Skeleton, t2.Skeleton)
			if len(shared) > o.MaxSharedSkeleton {
				fails = append(fails, fmt.Sprintf("%s and %s share %d of %d skeleton axes (> %d): %s: one composition in two paints",
					t1.Name, t2.Name, len(shared), both, o.MaxSharedSkeleton, strings.Join(shared, ", ")))
			}
		}
	}
	return fails
}

// NewcomerFlags are [P2] lines for a value more than --max-popular of the apps hold (with
// two apps, both) where the kit doesn't: newcomers leaving the default the same way.
// Counted over the apps alone, so the kit's themes can't dilute two builds that converged
// (two chat builds both chose the ╌ rule; seven kit themes hid it). A flag, not a failure:
// one shared departure is weak evidence and the product class may ask for it (every
// monitor has a history chart); the skeleton count is the failing rule.
func NewcomerFlags(themes []Theme, o Options) []string {
	as := apps(themes)
	if len(as) < 2 {
		return nil
	}
	var flags []string
	// A value counts as a departure when no baseline (and not the theme default, for an
	// expression) or the kit's skeleton holds it: two builds keeping the default are the
	// min-distance rule's business, two builds leaving it the same way are this one's.
	kitHolds := map[string]map[string]bool{}
	for _, a := range Axes {
		kitHolds[a.Key] = map[string]bool{pyfmt.Str(a.Default): true}
		for _, t := range themes {
			if o.Baseline[t.Name] && !t.App {
				kitHolds[a.Key][t.Choices[a.Key]] = true
			}
		}
	}
	kit := KitSkeleton()
	type axis struct {
		label, kitWord string
		val            func(Theme) string
		kit            map[string]bool
	}
	var axes []axis
	for _, a := range Axes {
		if twoValued[a.Key] {
			continue
		}
		k := a.Key
		axes = append(axes, axis{k, "not a baseline's or the default", func(t Theme) string { return t.Choices[k] }, kitHolds[k]})
	}
	for _, a := range SkeletonAxes {
		k := a.Key
		axes = append(axes, axis{"skeleton " + k, "not the kit's " + kit[k], func(t Theme) string { return t.Skeleton[k] }, map[string]bool{kit[k]: true}})
	}
	// a value the house-style rule already fails is not flagged again
	failed := map[string]bool{}
	var others []Theme
	for _, t := range themes {
		if !o.Baseline[t.Name] {
			others = append(others, t)
		}
	}
	for _, h := range houseStyles(others, o) {
		failed[h.key+"="+h.val] = true
	}
	for _, a := range axes {
		var vals []string
		holders := map[string][]string{}
		for _, t := range as {
			v := a.val(t)
			if !live(v) || a.kit[v] {
				continue
			}
			if holders[v] == nil {
				vals = append(vals, v)
			}
			holders[v] = append(holders[v], t.Name)
		}
		for _, v := range vals {
			h := holders[v]
			if failed[a.label+"="+v] || len(h) < 2 || float64(len(h))/float64(len(as)) <= o.MaxPopular {
				continue
			}
			who := "every app"
			if len(h) < len(as) {
				who = fmt.Sprintf("%d/%d apps", len(h), len(as))
			}
			flags = append(flags, fmt.Sprintf("[P2] %s=%s held by %s (%s; > %d%%), %s: newcomers converging on one departure",
				a.label, v, who, strings.Join(h, ", "), int(o.MaxPopular*100), a.kitWord))
		}
	}
	return flags
}

// KitAdvice is one line per app that names the skeleton axes it keeps from the kit. It is
// advice, never a failure: a plain brand keeps most of them, and the brand read's move is
// the one that should show as a difference.
func KitAdvice(themes []Theme) []string {
	kit := KitSkeleton()
	var out []string
	for _, t := range apps(themes) {
		shared, both := sharedSkeleton(t.Skeleton, kit)
		switch {
		case both == 0 || len(shared) == 0:
		case len(shared) == both:
			out = append(out, fmt.Sprintf("skeleton of %s: the kit's on all %d declared axes, so its brand read's move doesn't show here: if the move is built, declare it as built; if not, build it (brand.md § Move A Region)", t.Name, both))
		default:
			out = append(out, fmt.Sprintf("skeleton of %s: the kit's on %s (%d/%d); its brand read's move should be among the others (advice, not a failure)", t.Name, strings.Join(shared, ", "), len(shared), both))
		}
	}
	return out
}

// Verbose is the whole seating plan: for every axis, each value and who holds it (the
// kit's skeleton column included), then every pair's shared axes. The default report
// names shared axes only for a failing pair; this lets an agent see where every theme and
// app sits before choosing what, if anything, its brand read moves.
func Verbose(themes []Theme, o Options) string {
	var b strings.Builder
	b.WriteString("seating: every value of every axis, and who holds it\n")
	seat := func(key string, val func(Theme) string, ts []Theme, kit string) {
		var vals []string
		holders := map[string][]string{}
		add := func(v, name string) {
			if holders[v] == nil {
				vals = append(vals, v)
			}
			holders[v] = append(holders[v], name)
		}
		if kit != "" {
			add(kit, KitName)
		}
		for _, t := range ts {
			add(val(t), t.Name)
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			label := v
			if v == "" {
				label = `""`
			}
			parts[i] = label + ": " + strings.Join(holders[v], ", ")
		}
		b.WriteString(pyfmt.Ljust(key, 13) + strings.Join(parts, " · ") + "\n")
	}
	for _, a := range Axes {
		k := a.Key
		seat(k, func(t Theme) string { return t.Choices[k] }, themes, "")
	}
	if as := apps(themes); len(as) > 0 {
		kit := KitSkeleton()
		for _, a := range SkeletonAxes {
			k := a.Key
			seat(k, func(t Theme) string { return t.Skeleton[k] }, as, kit[k])
		}
	}
	b.WriteString("\npairs: the axes each pair shares\n")
	for i, t1 := range themes {
		for _, t2 := range themes[i+1:] {
			var shared []string
			both := 0
			for _, a := range Axes {
				v1, v2 := t1.Choices[a.Key], t2.Choices[a.Key]
				if v1 == Dead || v2 == Dead {
					continue
				}
				both++
				if v1 == v2 {
					shared = append(shared, a.Key)
				}
			}
			fmt.Fprintf(&b, "%s · %s: %d/%d axes", t1.Name, t2.Name, len(shared), both)
			if len(shared) > 0 {
				b.WriteString(" (" + strings.Join(shared, ", ") + ")")
			}
			if t1.App && t2.App {
				ss, sb := sharedSkeleton(t1.Skeleton, t2.Skeleton)
				fmt.Fprintf(&b, "; skeleton %d/%d", len(ss), sb)
				if len(ss) > 0 {
					b.WriteString(" (" + strings.Join(ss, ", ") + ")")
				}
			}
			b.WriteByte('\n')
		}
	}
	for _, t := range apps(themes) {
		ss, sb := sharedSkeleton(t.Skeleton, KitSkeleton())
		fmt.Fprintf(&b, "%s · %s: skeleton %d/%d", t.Name, KitName, len(ss), sb)
		if len(ss) > 0 {
			b.WriteString(" (" + strings.Join(ss, ", ") + ")")
		}
		b.WriteByte('\n')
	}
	return b.String()
}
