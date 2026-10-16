package speclint

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// footerCut in a keys cell ends what the footer shows: the items before it are the
// footer, the items after it are help-only (`show: false` in the action table, listed
// by the ? overlay).
const footerCut = '‖'

// How the kit draws a footer (kit/ui/chrome.go Help and Footer, kit/theme/derive.go),
// at the 80-column design width. TestFooterMirrorsKit holds these to the kit's source.
const (
	footerWidth  = 80
	kitGutter    = 2 // theme default side margin, on both sides of the footer row
	hintSep      = 2 // Help joins hints with two spaces
	moreReserve  = 3 // room Help keeps for " …" while hints remain
	footerBudget = footerWidth - 2*kitGutter
)

// footerItems splits a keys cell into the items the footer shows (every item, or those
// before a ‖) and the help-only items after the ‖, each as the kit renders a hint:
// `key desc`, single-spaced.
func footerItems(cell string) (footer, helpOnly []string, cut bool) {
	before, after, cut := strings.Cut(clean(cell), string(footerCut))
	return hints(before), hints(after), cut
}

func hints(part string) []string {
	if emptyCell.MatchString(strings.TrimSpace(part)) {
		return nil
	}
	var out []string
	for _, it := range splitItems(part) {
		if f := strings.Fields(it); len(f) > 0 {
			out = append(out, strings.Join(f, " "))
		}
	}
	return out
}

var helpAction = regexp.MustCompile(`(?i)^(?:full )?(?:help|keys)$`)

// opensHelp reports an item that opens full help: Help pins `?` and ctrl+g last.
func opensHelp(item string) bool {
	b := readItem(strings.Fields(item))
	for _, k := range b.keys {
		if k.String() == "?" || k.String() == "ctrl+g" {
			return true
		}
	}
	return helpAction.MatchString(b.action)
}

// packFooter lays items out as ui.Help does in w columns and returns the ones it
// drops behind "…". A first item wider than w is cut, so it counts as dropped too.
func packFooter(items []string, w int) (shown, dropped []string) {
	var pinned string
	if n := len(items); n > 1 && pinsLast(items[n-1]) {
		pinned = items[n-1]
		w -= ansi.StringWidth(pinned) + hintSep
		items = items[:n-1]
	}
	used := 0
	for i, it := range items {
		need := ansi.StringWidth(it)
		if len(shown) > 0 {
			need += hintSep
		}
		reserve := 0
		if i < len(items)-1 {
			reserve = moreReserve
		}
		if used+need+reserve > w && len(shown) > 0 {
			dropped = items[i:]
			break
		}
		shown = append(shown, it)
		used += need
	}
	if used > w && len(dropped) == 0 && len(shown) > 0 {
		dropped = shown // the only hint, cut by Fit
		shown = nil
	}
	if pinned != "" {
		shown = append(shown, pinned)
	}
	return shown, dropped
}

// pinsLast is Help's own test: it pins a last hint whose key is exactly ? or ctrl+g.
func pinsLast(item string) bool {
	f := strings.Fields(item)
	return len(f) > 0 && (f[0] == "?" || f[0] == "ctrl+g")
}

func hintsWidth(items []string) int {
	n := 0
	for i, it := range items {
		if i > 0 {
			n += hintSep
		}
		n += ansi.StringWidth(it)
	}
	return n
}

// footerFit checks each state's footer against the 80-column footer the kit draws,
// and that a state which hides keys behind ‖ still shows how to reach them.
func (s *spec) footerFit(add func(string, int, string, ...any)) {
	globalHelp := false
	for _, g := range s.globals {
		for _, b := range g.bindings {
			globalHelp = globalHelp || opensHelp(b.token+" "+b.action)
		}
	}
	for _, r := range s.rows {
		_, dropped := packFooter(r.footer, footerBudget)
		if len(dropped) > 0 {
			what, fix := "its footer keys are", "Mark the cut with ‖"
			if r.cut {
				what, fix = "its footer keys before ‖ are still", "Move ‖ left"
			}
			front, back := suggestCut(r.footer)
			add("S-FOOTER", r.line, "%s: %s %d columns, and an %d-column footer has %d inside the kit's %d-cell gutters, so ui.Help drops `%s` behind …. %s: keys after it are help-only (show: false in the action table; ? help lists them): `%s ‖ %s`",
				r.state, what, hintsWidth(r.footer), footerWidth, footerBudget, kitGutter,
				strings.Join(dropped, " · "), fix,
				strings.Join(front, " · "), strings.Join(append(back, r.helpOnly...), " · "))
			continue
		}
		if !r.cut || globalHelp {
			continue
		}
		help := false
		for _, it := range r.footer {
			help = help || opensHelp(it)
		}
		if !help {
			add("S-FOOTER", r.line, "%s: the keys after ‖ are help-only, but its footer shows no key that opens help, so nobody finds them. End the footer with `? help` (ctrl+g while a text box has focus), before ‖",
				r.state)
		}
	}
}

// suggestCut splits an overflowing footer where ui.Help would, with the help key moved
// last (where Help pins it) or `? help` added, so the help-only keys stay findable.
func suggestCut(items []string) (front, back []string) {
	var rest []string
	help := ""
	for _, it := range items {
		if help == "" && opensHelp(it) {
			help = it
			continue
		}
		rest = append(rest, it)
	}
	if help == "" {
		help = "? help"
	}
	return packFooter(append(rest, help), footerBudget)
}
