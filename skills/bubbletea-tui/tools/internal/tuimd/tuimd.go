// Package tuimd reads the parts of a TUI.md (references/cmd-init.md's template) that
// more than one lint holds frames to: the region budgets of § Screens and the `cuts:`
// lines of § Decisions.
package tuimd

import (
	"regexp"
	"strconv"
	"strings"
)

// Region is one `name x,y w×h` item of a budget line: a rectangle of the frame, in
// cells from the top left, 0-based.
type Region struct {
	Name       string
	X, Y, W, H int
}

// Contains reports whether the cell at column x, row y is in r.
func (r Region) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Budget is one `80×24: …` line: the regions of a screen at one frame size.
type Budget struct {
	Screen  string // the block's heading up to its job: "overview" of "overview — what is …"
	W, H    int
	Line    int // 1-based
	Regions []Region
}

// Cut is one `cuts:` line of § Decisions: the column headers or region names whose
// content is cut on purpose (paths, URLs, commands: no longest to budget from), on one
// screen (`cuts: detail: name`) or, with no screen named, on every screen.
type Cut struct {
	Screen string   // lowercased; "" for every screen
	Names  []string // lowercased, as written
	Line   int
}

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	// a block's heading names its job after a dash or a colon: "detail — one job"
	jobRe    = regexp.MustCompile(`\s[—–-]\s|:`)
	budgetRe = regexp.MustCompile(`^(\d+)\s*[×x]\s*(\d+)\s*:\s*(.*)$`)
	regionRe = regexp.MustCompile(`^(.*?\S)\s+(\d+)\s*,\s*(\d+)\s+(\d+)\s*[×x]\s*(\d+)\b`)
	listMark = regexp.MustCompile(`^(?:[-*+]|\d+\.)\s+`)
	// a decision's lead: a date as `2026-10-18`, `2026-10-18:`, `2026-10-18 —` or
	// `[2026-10-18]`, then at most one `word:` tag (`shape:`)
	cutsRe    = regexp.MustCompile(`(?i)^(?:\[?\d{4}-\d{2}-\d{2}\]?\s*[:—–-]?\s*)?(?:\w+:\s*)?cuts\s*:\s*(.*)$`)
	scopeRe   = regexp.MustCompile(`^\s*([\w-]+)\s*:\s*(.*)$`)
	reasonCut = regexp.MustCompile(`\s[—–-]\s|;|\s+because\s|\(`)
	decisions = regexp.MustCompile(`(?i)^(?:\d+[.)]?\s+)?decisions$`)
)

func clean(s string) string {
	return strings.NewReplacer("**", "", "__", "", "`", "").Replace(strings.TrimSpace(s))
}

// lines yields the lines outside ``` and ~~~ fences (a sketch is not a budget), with
// their 1-based numbers.
func lines(src string, each func(n int, t string)) {
	fence := ""
	for i, l := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(l)
		for _, m := range []string{"```", "~~~"} {
			if strings.HasPrefix(t, m) {
				switch fence {
				case "":
					fence = m
				case m:
					fence = ""
				}
				t = ""
			}
		}
		if fence != "" || t == "" {
			continue
		}
		each(i+1, t)
	}
}

// Budgets reads every `<W>×<H>:` line (or `<W>x<H>:`) of a TUI.md, each under the
// screen block heading it follows. An item without coordinates ("header, watch line as
// overview") is skipped; so is the template's `<region> <x,y w×h>`.
func Budgets(src string) []Budget {
	var out []Budget
	screen := ""
	lines(src, func(n int, t string) {
		if m := headingRe.FindStringSubmatch(t); m != nil {
			screen = strings.ToLower(strings.TrimSpace(jobRe.Split(clean(m[2]), 2)[0]))
			return
		}
		m := budgetRe.FindStringSubmatch(clean(t))
		if m == nil {
			return
		}
		b := Budget{Screen: screen, Line: n}
		b.W, _ = strconv.Atoi(m[1])
		b.H, _ = strconv.Atoi(m[2])
		for _, item := range strings.Split(m[3], "·") {
			r := regionRe.FindStringSubmatch(strings.TrimSpace(item))
			if r == nil {
				continue
			}
			g := Region{Name: strings.ToLower(r[1])}
			g.X, _ = strconv.Atoi(r[2])
			g.Y, _ = strconv.Atoi(r[3])
			g.W, _ = strconv.Atoi(r[4])
			g.H, _ = strconv.Atoi(r[5])
			b.Regions = append(b.Regions, g)
		}
		if len(b.Regions) > 0 {
			out = append(out, b)
		}
	})
	return out
}

// Cuts reads the `cuts:` lines of § Decisions (a heading `Decisions`, to the next heading
// at its level or above): `- 2026-10-18 cuts: command, program — a command line has no
// longest; the detail shows it whole`, or for one screen `- cuts: detail: name — …` (a
// screen is a heading with a job, `#### detail — …`; after any other word the colon
// starts the reason). The names end at the reason (a dash, a ";", "because" or a "("),
// and are split on commas.
func Cuts(src string) []Cut {
	screens := map[string]bool{} // the names of headings with a job: the screen blocks
	lines(src, func(_ int, t string) {
		if m := headingRe.FindStringSubmatch(t); m != nil && jobRe.MatchString(clean(m[2])) {
			screens[strings.ToLower(strings.TrimSpace(jobRe.Split(clean(m[2]), 2)[0]))] = true
		}
	})
	var out []Cut
	level := 0
	lines(src, func(n int, t string) {
		if m := headingRe.FindStringSubmatch(t); m != nil {
			switch {
			case decisions.MatchString(clean(m[2])):
				level = len(m[1])
			case level > 0 && len(m[1]) <= level:
				level = 0
			}
			return
		}
		if level == 0 {
			return
		}
		m := cutsRe.FindStringSubmatch(listMark.ReplaceAllString(clean(t), ""))
		if m == nil {
			return
		}
		names := reasonCut.Split(m[1], 2)[0]
		c := Cut{Line: n}
		if sc := scopeRe.FindStringSubmatch(names); sc != nil {
			if screens[strings.ToLower(sc[1])] {
				c.Screen, names = strings.ToLower(sc[1]), sc[2]
			} else {
				names = sc[1] // `cuts: path: a path has no longest`: the colon starts the reason
			}
		}
		for _, nm := range strings.Split(names, ",") {
			if nm = strings.ToLower(strings.TrimSpace(nm)); nm != "" {
				c.Names = append(c.Names, nm)
			}
		}
		if len(c.Names) > 0 {
			out = append(out, c)
		}
	})
	return out
}
