// Package redact turns a frame's words into placeholders while keeping its layout:
// every letter becomes x or X (keeping case), every digit 0, and widths, styles, glyphs,
// box drawing, punctuation and whitespace stay. URL-, email- and path-shaped runs are
// redacted whole, punctuation included. A short allowlist of key names and generic UI
// words survives, so a redacted frame can still be critiqued.
package redact

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/rivo/uniseg"
)

// Allow is the words kept as they are (matched without case): key names a footer
// shows, and generic UI words. Short on purpose: each is a word a product can't be
// recognised by. ↑ ↓ ← → and other glyphs are kept anyway; so is a single letter
// after ctrl+, alt+ or shift+ (ctrl+c).
var Allow = []string{
	"enter", "esc", "tab", "space", "shift", "ctrl", "alt", "pgup", "pgdn", "home", "end", "backspace", "q",
	"help", "quit", "back", "next", "cancel", "keys", "yes", "no", "ok",
}

// Redactor redacts with Allow plus extra words.
type Redactor struct{ allow map[string]bool }

// New returns a redactor that also keeps extra.
func New(extra ...string) *Redactor {
	r := &Redactor{allow: map[string]bool{}}
	for _, w := range append(slices.Clone(Allow), extra...) {
		if w = strings.ToLower(strings.TrimSpace(w)); w != "" {
			r.allow[w] = true
		}
	}
	return r
}

// Allowed reports whether the word is kept.
func (r *Redactor) Allowed(word string) bool { return r.allow[strings.ToLower(word)] }

// unit is one grapheme: a cell, or one cluster of a string.
type unit struct {
	s     string
	whole bool // in a URL, email or path: redact every non-space unit
	keep  bool // in an allowed word
}

// Lines redacts a frame's cells in place. A wide letter becomes two narrow
// placeholders, so every row keeps its width and every cell its style.
func (r *Redactor) Lines(lines []uv.Line) {
	for _, line := range lines {
		var idx []int // cell index of each unit
		var us []unit
		for x, c := range line {
			if c.Content == "" && c.Width == 0 {
				continue // the second half of a wide character
			}
			idx = append(idx, x)
			us = append(us, unit{s: c.Content})
		}
		r.mark(us)
		for i, u := range us {
			out, ok := r.replace(u)
			if !ok {
				continue
			}
			x := idx[i]
			c := line[x]
			w := max(1, c.Width)
			for k := range w {
				if x+k >= len(line) {
					break
				}
				nc := c
				nc.Content, nc.Width = out, 1
				line[x+k] = nc
			}
		}
	}
}

// String redacts text, such as what a tape typed.
func (r *Redactor) String(s string) string {
	var us []unit
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		us = append(us, unit{s: g.Str()})
	}
	r.mark(us)
	var b strings.Builder
	for _, u := range us {
		if out, ok := r.replace(u); ok {
			b.WriteString(out)
		} else {
			b.WriteString(u.s)
		}
	}
	return b.String()
}

var (
	urlRe   = regexp.MustCompile(`^(?i)([a-z][a-z0-9+.-]*://|www\.)\S`)
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	extRe   = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9]{0,7}$`)
)

// pathLike is a run that names a file: it starts at a root (/, ~/, ./, ../, C:\),
// has two or more separators, or one and a file extension (kit/README.md). Key hints
// such as j/k, y/n, tab/shift+tab and 1/3 are not paths.
func pathLike(s string) bool {
	if !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
		return false
	}
	switch {
	case strings.HasPrefix(s, "/") && len(s) > 1, strings.HasPrefix(s, "~/"), strings.HasPrefix(s, "./"), strings.HasPrefix(s, "../"):
		return true
	case len(s) > 3 && unicode.IsLetter(rune(s[0])) && s[1] == ':' && s[2] == '\\':
		return true
	}
	seps := strings.Count(s, "/") + strings.Count(s, "\\")
	return seps >= 2 || (seps == 1 && extRe.MatchString(s))
}

const edgePunct = `()[]{}<>"'.,;:!?` + "`"

// mark finds the runs redacted whole and the words kept.
func (r *Redactor) mark(us []unit) {
	// whitespace-separated tokens: URL, email and path shapes
	for i := 0; i < len(us); {
		if blank(us[i].s) {
			i++
			continue
		}
		j := i
		for j < len(us) && !blank(us[j].s) {
			j++
		}
		a, b := i, j // trim the brackets and punctuation around the token
		for a < b && strings.Contains(edgePunct, us[a].s) {
			a++
		}
		for b > a && strings.Contains(edgePunct, us[b-1].s) {
			b--
		}
		tok := join(us[a:b])
		if urlRe.MatchString(tok) || emailRe.MatchString(tok) || pathLike(tok) {
			for k := a; k < b; k++ {
				us[k].whole = true
			}
		}
		i = j
	}
	// words: runs of letters and digits
	for i := 0; i < len(us); {
		if !wordy(us[i].s) {
			i++
			continue
		}
		j := i
		for j < len(us) && wordy(us[j].s) {
			j++
		}
		word := join(us[i:j])
		keep := r.Allowed(word)
		// a single letter after ctrl+, alt+ or shift+ is a key
		if !keep && j-i == 1 && i >= 2 && us[i-1].s == "+" {
			k := i - 1
			for k > 0 && wordy(us[k-1].s) {
				k--
			}
			switch strings.ToLower(join(us[k : i-1])) {
			case "ctrl", "alt", "shift":
				keep = true
			}
		}
		if keep {
			for k := i; k < j; k++ {
				us[k].keep = true
			}
		}
		i = j
	}
}

// replace is a unit's placeholder, and whether it changes.
func (r *Redactor) replace(u unit) (string, bool) {
	if blank(u.s) || (u.keep && !u.whole) {
		return "", false
	}
	first := []rune(u.s)[0]
	switch {
	case unicode.IsUpper(first) || unicode.IsTitle(first):
		return "X", true
	case unicode.IsLetter(first):
		return "x", true
	case unicode.IsNumber(first):
		return "0", true
	case u.whole:
		return "x", true
	}
	return "", false
}

func wordy(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

func blank(s string) bool { return strings.TrimSpace(s) == "" }

func join(us []unit) string {
	var b strings.Builder
	for _, u := range us {
		b.WriteString(u.s)
	}
	return b.String()
}
