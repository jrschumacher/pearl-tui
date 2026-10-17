// Package diversity is the engine behind the diversity command: tabulate each theme's
// visible choices and fail a set of themes that are one design in several colours.
// It is a port of scripts/diversity.py and prints the same bytes (see the testdata
// goldens). It differs on purpose in four places:
//
//   - below --max-popular 0.5 one axis can fail on several values; they are listed in
//     first-seen order, where diversity.py iterates a set (hash order, so its output
//     changed from run to run);
//   - an unreadable or malformed theme exits 2 with one line on stderr instead of a
//     traceback (exit 1); usage errors exit 2 as argparse's do, with similar wording;
//   - a --baseline name that matches no theme exits 2 with one line on stderr, where
//     diversity.py skipped the min-distance rule silently (the default baseline, when
//     missing, only adds a note on stderr);
//   - -h prints the command's Go doc comment.
package diversity

import (
	"fmt"
	"os"
	"strings"

	"tuitools/internal/pyfmt"
)

// Axis is one counted choice: an expression ("x") or a signature glyph ("g").
type Axis struct {
	Group, Key string
	Default    any
}

// Axes are the 21 counted choices, in table order.
var Axes = []Axis{
	{"x", "header", "rule"}, {"x", "wordmark", "gradient"}, {"x", "sectionTitle", "accent"},
	{"x", "titleCase", "sentence"}, {"x", "selection", "bar"}, {"x", "button", "pill"}, {"x", "tag", "filled"},
	{"x", "dialog", "rounded"}, {"x", "dialogTitle", "rule"}, {"x", "tabs", "underline"}, {"x", "footer", "keys"},
	{"x", "keys", "inline"}, {"x", "density", "airy"}, {"x", "backdrop", "none"}, {"x", "scrim", "dim"},
	{"x", "paintBg", false},
	{"g", "headerRule", ""}, {"g", "rule", "─"}, {"g", "focusBar", "▌"}, {"g", "cursor", "›"}, {"g", "done", "✓"},
}

// twoValued axes are exempt from the popularity rule: one of two values always has a
// majority.
var twoValued = map[string]bool{"titleCase": true, "density": true, "backdrop": true, "scrim": true, "paintBg": true, "keys": true}

// Dead marks an axis nothing renders under the theme's other choices.
const Dead = "-"

// Theme is one theme's name and its value on every axis, as Python's str() prints it.
type Theme struct {
	Name    string
	Choices map[string]string
}

// Load reads a theme file's choices. Dead axes are "-": headerRule under a
// bar/stamp/minimal header, wordmark under a banner/stamp header, dialogTitle under a
// rules dialog.
func Load(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	v, err := pyfmt.Decode(data)
	if err != nil {
		return Theme{}, fmt.Errorf("%s: %w", path, err)
	}
	t, ok := v.(map[string]any)
	if !ok {
		return Theme{}, fmt.Errorf("%s: not a JSON object", path)
	}
	group := func(k string) (map[string]any, error) {
		g, ok := t[k]
		if !ok {
			return map[string]any{}, nil
		}
		m, ok := g.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: %s is not an object", path, k)
		}
		return m, nil
	}
	x, err := group("expressions")
	if err != nil {
		return Theme{}, err
	}
	g, err := group("glyphs")
	if err != nil {
		return Theme{}, err
	}
	out := map[string]string{}
	for _, a := range Axes {
		src := x
		if a.Group == "g" {
			src = g
		}
		val, ok := src[a.Key]
		if !ok {
			val = a.Default
		}
		out[a.Key] = pyfmt.Str(val)
	}
	switch out["header"] {
	case "bar", "stamp", "minimal":
		out["headerRule"] = Dead
	}
	switch out["header"] {
	case "banner", "stamp":
		out["wordmark"] = Dead
	}
	if out["dialog"] == "rules" { // the title sits in the top rule (not in tuilint.py)
		out["dialogTitle"] = Dead
	}
	name := path
	if n, ok := t["name"]; ok {
		name = pyfmt.Str(n)
	}
	return Theme{name, out}, nil
}

// Options are the thresholds.
type Options struct {
	MaxShared   int             // two themes may agree on at most this many axes
	MaxPopular  float64         // one value may be used by at most this share of brand themes
	Baseline    map[string]bool // theme names that are the defaults to move away from
	MinDistance int             // a brand theme differs from each baseline on at least this many axes
}

// Table renders the axis-by-theme table, names and values cut to 10 code points.
func Table(themes []Theme) string {
	var b strings.Builder
	b.WriteString(pyfmt.Ljust("axis", 13))
	for _, t := range themes {
		b.WriteString(pyfmt.Ljust(pyfmt.Head(t.Name, 10), 12))
	}
	b.WriteByte('\n')
	for _, a := range Axes {
		b.WriteString(pyfmt.Ljust(a.Key, 13))
		for _, t := range themes {
			b.WriteString(pyfmt.Ljust(pyfmt.Head(t.Choices[a.Key], 10), 12))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Failures applies the three rules and returns one line per failure.
func Failures(themes []Theme, o Options) []string {
	var fails []string
	for i, t1 := range themes {
		for _, t2 := range themes[i+1:] {
			if o.Baseline[t1.Name] || o.Baseline[t2.Name] { // baselines are judged by MinDistance
				continue
			}
			var shared []string
			for _, a := range Axes {
				if v := t1.Choices[a.Key]; v == t2.Choices[a.Key] && v != Dead {
					shared = append(shared, a.Key)
				}
			}
			if len(shared) > o.MaxShared {
				fails = append(fails, fmt.Sprintf("%s and %s share %d axes (> %d): %s",
					t1.Name, t2.Name, len(shared), o.MaxShared, strings.Join(shared, ", ")))
			}
		}
	}
	var others []Theme
	for _, t := range themes {
		if !o.Baseline[t.Name] {
			others = append(others, t)
		}
	}
	for _, t := range others {
		for _, b := range themes {
			if !o.Baseline[b.Name] {
				continue
			}
			dist := 0
			var shared []string // what to move, each traced to the brand read, not to this rule
			for _, a := range Axes {
				v, bv := t.Choices[a.Key], b.Choices[a.Key]
				if v == Dead || bv == Dead {
					continue
				}
				if v != bv {
					dist++
				} else {
					shared = append(shared, a.Key)
				}
			}
			if dist < o.MinDistance {
				fails = append(fails, fmt.Sprintf("%s differs from baseline %s on only %d axes (< %d): a default with a new palette; shares %s",
					t.Name, b.Name, dist, o.MinDistance, strings.Join(shared, ", ")))
			}
		}
	}
	if len(others) >= 3 {
		for _, a := range Axes {
			if twoValued[a.Key] {
				continue
			}
			var vals []string
			count := map[string]int{}
			for _, t := range others {
				if v := t.Choices[a.Key]; v != Dead {
					if count[v] == 0 {
						vals = append(vals, v)
					}
					count[v]++
				}
			}
			n := 0
			for _, c := range count {
				n += c
			}
			for _, v := range vals {
				if float64(count[v])/float64(len(others)) > o.MaxPopular && n >= 3 {
					fails = append(fails, fmt.Sprintf("%s=%s used by %d/%d brand themes (> %d%%): a house style",
						a.Key, v, count[v], len(others), int(o.MaxPopular*100)))
				}
			}
		}
	}
	return fails
}

// Nearest names, for each theme that isn't a baseline, the other non-baseline theme it
// shares most axes with (first in input order on a tie), as "shared/live" where live
// counts the axes neither theme has dead. It is advice, never a failure: a brand that
// passes the baselines can still be the kit's moss with a new palette, and nothing
// else says so.
func Nearest(themes []Theme, o Options) []string {
	var others []Theme
	for _, t := range themes {
		if !o.Baseline[t.Name] {
			others = append(others, t)
		}
	}
	if len(others) < 2 {
		return nil
	}
	var out []string
	for _, t := range others {
		best, bestShared, bestLive := "", -1, 0
		for _, u := range others {
			if u.Name == t.Name {
				continue
			}
			shared, live := 0, 0
			for _, a := range Axes {
				v, uv := t.Choices[a.Key], u.Choices[a.Key]
				if v == Dead || uv == Dead {
					continue
				}
				live++
				if v == uv {
					shared++
				}
			}
			if shared > bestShared {
				best, bestShared, bestLive = u.Name, shared, live
			}
		}
		out = append(out, fmt.Sprintf("nearest to %s: %s (shares %d/%d axes)", t.Name, best, bestShared, bestLive))
	}
	return out
}

// Report is the full output: the table, a blank line, the nearest-theme notes, the
// failures, and the verdict.
func Report(themes []Theme, o Options) (string, int) {
	fails := Failures(themes, o)
	var b strings.Builder
	b.WriteString(Table(themes))
	b.WriteByte('\n')
	for _, n := range Nearest(themes, o) {
		b.WriteString(n + "\n")
	}
	for _, f := range fails {
		b.WriteString("FAIL " + f + "\n")
	}
	if len(fails) == 0 {
		b.WriteString("OK\n")
		return b.String(), 0
	}
	fmt.Fprintf(&b, "%d problem(s)\n", len(fails))
	return b.String(), 1
}
