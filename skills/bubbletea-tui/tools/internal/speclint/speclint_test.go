package speclint

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// table wraps rows (and an optional Global line) in the template's table.
func table(rows string, global string) string {
	s := "# x TUI\n\n### States and keys\n\n| state (harness name) | screen / overlay | keys |\n|---|---|---|\n" + rows + "\n"
	if global != "" {
		s += "\nGlobal: " + global + "\n"
	}
	return s
}

func rules(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Rule]++
	}
	return m
}

func TestFooterProseIsNotHidden(t *testing.T) {
	// A text-entry row whose only unbolded item is prose hides no key, so it needs no
	// help key in its footer (and ? would type a character there anyway).
	for _, doc := range []string{
		table("| compose | conversation | type to write · **enter preview** · **shift+enter/ctrl+j newline** · **esc cancel** |", ""),
		table("| palette | overlay | ↑↓ move · enter run · esc close ‖ type to filter |", ""),
	} {
		if fs := Lint("t.md", doc); len(fs) != 0 {
			t.Errorf("want no findings:\n%s\n%s", doc, dump(fs))
		}
	}
}

func TestRules(t *testing.T) {
	cases := []struct {
		name, doc string
		want      map[string]int // rule → count; rules not named must not fire
	}{
		// S-TABLE
		{"no table", "# x TUI\n\nkeys: enter open\n", map[string]int{"S-TABLE": 1}},
		{"table", table("| list | list | enter open · q quit |", ""), nil},
		{"a Global line under Decisions is prose", table("| list | list | g top |", "ctrl+c quit") + "\n## Decisions\n- Global: shift+enter is reserved for send because the list is long\n", nil},
		{"a Global line in a subsection of the keys section counts", table("| list | list | g top |", "") + "\n#### Everywhere\nGlobal: shift+enter send\n", map[string]int{"S-KEYAMBIG": 1}},
		{"a Global line before the table is prose", "# x TUI\n\nGlobal: shift+enter is reserved\n\n" + table("| list | list | g top |", ""), nil},
		{"table with bold global and backticks", table("| list | list | `enter` open |", "") + "\n**Global:** `ctrl+c` quit\n", nil},

		// S-KEYAMBIG fires
		{"shift+enter", table("| reply | overlay | enter send · shift+enter newline |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+enter", table("| reply | overlay | ctrl+enter send |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+m", table("| list | list | ctrl+m mark |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+i", table("| list | list | ctrl+i info |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+h warns", table("| list | list | ctrl+h history |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+backspace", table("| list | list | ctrl+backspace delete word |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+backspace with a fallback", table("| list | list | ctrl+backspace/ctrl+w delete word |", ""), nil},
		{"ctrl+[", table("| list | list | ctrl+[ back |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"shift+space", table("| list | list | shift+space page up |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+tab", table("| list | list | ctrl+tab next pane |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"shift+backspace", table("| list | list | shift+backspace clear |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+shift+letter", table("| list | list | ctrl+shift+k kill |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+/", table("| list | list | ctrl+/ comment |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+digit", table("| list | list | ctrl+1 first tab |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+@", table("| list | list | ctrl+@ mark |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"super", table("| list | list | cmd+k clear |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"in Global", table("| list | list | enter open |", "shift+enter newline"), map[string]int{"S-KEYAMBIG": 1}},
		{"fallback for another action", table("| reply | overlay | shift+enter newline · ctrl+j send |", ""), map[string]int{"S-KEYAMBIG": 1}},

		// S-KEYAMBIG silent
		{"slash fallback", table("| reply | overlay | enter send · shift+enter/ctrl+j newline |", ""), nil},
		{"fallback item in the same cell", table("| reply | overlay | enter send · shift+enter newline · ctrl+j newline |", ""), nil},
		{"keys every terminal sends", table("| list | list | shift+tab back · ctrl+space mark · alt+enter run · ctrl+j down · ? help ‖ ctrl+_ undo · ctrl+\\ split · ctrl+k up · alt+backspace kill |", ""), nil},
		{"</> and +/- are their own keys, not the slash", table("| list | list | / filter · </> sort · +/- interval · = reset |", ""), nil},
		{"the slash key and ctrl+", table("| list | list | / filter · ? help · ^R refresh · Ctrl+K up |", ""), nil},

		// S-KEYDUP
		{"dup in a state", table("| list | list | esc back · esc clear |", ""), map[string]int{"S-KEYDUP": 1}},
		{"dup across alternatives", table("| list | list | ↑↓/jk move · k kill |", ""), map[string]int{"S-KEYDUP": 1}},
		{"a modified arrow pair is two keys", table("| chat | chat | **↑↓ line** · **shift+↑↓ message** |", ""), nil},
		{"the modifier holds across alternatives", table("| chat | chat | ↑/↓ line · shift+↑/↓ by message |", ""), nil},
		{"a modified pair still duplicates", table("| chat | chat | shift+↑↓ message · shift+↓ jump |", ""), map[string]int{"S-KEYDUP": 1}},
		{"a modifier stays off a character", table("| list | list | **ctrl+c/q quit** · **q queue** |", ""), map[string]int{"S-KEYDUP": 1}},
		{"a state overrides a Global key", table("| list | list | ctrl+k kill line |", "ctrl+k switch"), nil},
		{"overrides of esc, ? and ctrl+c", table("| confirm | overlay | enter confirm · esc cancel |\n| help | overlay | ? close · esc close |\n| creating | wizard | ctrl+c cancel |", "esc back · ? help · ctrl+c quit"), nil},
		{"one cell, two actions", table("| status | status | c commit · c comment |", ""), map[string]int{"S-KEYDUP": 1}},
		{"one cell, near-alike actions, across a footer cut", table("| s | s | p preview · p previous · u unstage · u unstash · ? help ‖ o open · o open in browser · pgdn page down · pgdn page up |", ""), map[string]int{"S-KEYDUP": 4}},
		{"one cell, the same action reworded by punctuation", table("| s | s | [ ] back/fwd · [ back fwd |", ""), nil},
		{"Global, two actions", table("| a | a | x go |", "c commit · c comment"), map[string]int{"S-KEYDUP": 1}},
		{"a state restating a Global pair in fewer words", table("| pr | pr | t tour · [ ] back/fwd |", "ctrl+c quit, [ ] back/forward between items."), nil},
		{"a pair written with a space", table("| list | list | j k scroll · k kill |", ""), map[string]int{"S-KEYDUP": 1}},
		{"dup in Global, once", table("| a | a | x go |\n| b | b | y go |", "ctrl+k switch · ctrl+k palette"), map[string]int{"S-KEYDUP": 1}},
		{"shift+g is G", table("| list | list | G bottom · shift+g end |", ""), map[string]int{"S-KEYDUP": 1}},
		{"same action twice", table("| list | list | ? help · q quit |", "? help · ctrl+c quit"), nil},
		{"same key in two states", table("| a | a | r refresh |\n| b | b | r reply |", ""), nil},
		{"g and G differ", table("| list | list | g top · G bottom |", ""), nil},
		{"a spaced slash joins two keys", table("| list | list | q / ctrl+c quit · / search · ctrl+d / ctrl+u half page · d delete · u undo |", ""), nil},
		{"a spaced slash still finds a dup", table("| list | list | q / ctrl+c quit · ctrl+c copy |", ""), map[string]int{"S-KEYDUP": 1}},
		{"comma and or between alternatives", table("| list | list | j/k, ↑/↓ move · h/l or ←/→ switch · enter: open |", ""), nil},
		{"comma and or still find a dup", table("| list | list | j/k, ↑/↓ move · h/l or ←/→ switch · ← back |", ""), map[string]int{"S-KEYDUP": 1}},

		// S-TABLE warns on what it can't read
		{"an unreadable key", table("| list | list | enter open · foo bar |", ""), map[string]int{"S-TABLE": 1}},
		{"prose and empty cells are not keys", table("| list-filter | list | type to filter · esc clear |\n| splash | splash | — |\n| about | overlay | any key close |", ""), nil},

		// S-KEYSHADOW
		{"ctrl+k over a textarea", table("| reply | overlay | enter send · shift+enter/ctrl+j newline · esc cancel |", "ctrl+k switch · ctrl+c quit"), map[string]int{"S-KEYSHADOW": 1}},
		{"printable over a text input", table("| search | search (text input) | enter go · esc cancel |", "q quit"), map[string]int{"S-KEYSHADOW": 1}},
		{"Global separated by commas", table("| ask | overlay | enter send · shift+enter/ctrl+j newline · esc cancel |", "ctrl+c quit, ctrl+k switcher."), map[string]int{"S-KEYSHADOW": 1}},
		{"type to", table("| list-filter | list | type to filter · esc clear |", "/ filter"), map[string]int{"S-KEYSHADOW": 1}},
		{"no text entry", table("| list | list | enter open · esc back |", "ctrl+k switch · q quit"), nil},
		{"filter alone is not text entry", table("| list | list | / filter · esc back |", "q quit"), nil},
		{"the state binds it", table("| reply | overlay | enter send · ctrl+j newline · ctrl+k switch |", "ctrl+k switch"), nil},
		{"keys a text box leaves alone", table("| reply | editor | enter send · ctrl+j newline |", "esc back · ctrl+c quit · ctrl+o switch"), nil},

		// S-GLYPHWIDTH
		{"motif with no ascii", "### Brand read\nmotif: ◐ seen ★ starred\n" + table("| list | list | enter open |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"glyph in the key table", table("| list | list ★ | enter open ◐ |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"reported once per glyph", "### Brand read\nmotif: ★\nsignature: the ★ row\n" + table("| list | list | enter open |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"ascii twin on the line", "### Brand read\nmotif: ★ starred ◐ seen (ascii * ~)\nsignature: the ★ row\n" + table("| list | list ★ | enter open |", ""), nil},
		{"ascii twin elsewhere", "### Brand read\nmotif: ★ starred\n\n### Glyphs\nASCII: ★ → *\n" + table("| list | list | enter open |", ""), nil},
		{"glyphs the kit maps to ascii", "### Brand read\nfocus: the ▌ gutter, ─ rules, │ rails, ● ○ ◆ ▲ ■ state marks\n" + table("| list | list ● | enter open ◆ |", ""), nil},
		{"arrows in prose", "### Brand read\nmotion: queued → running → done, retries ⇒ backoff ↻\n" + table("| list | list | enter open |", ""), nil},
		{"an arrow in the key table", table("| list | list | enter open ⇒ detail |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"arrow key names", table("| list | list | ↑↓/jk move · ←→ choose |", ""), nil},
		{"outside the brand read", "### Screens\nlist — ★ rows\n\n### Brand read\ntone: calm — precise…\n" + table("| list | list | enter open |", ""), nil},
		{"narrow symbols", "### Brand read\nmotif: ✓ done ❯ cursor ▸ open\n" + table("| list | list | enter open |", ""), nil},
		// S-FOOTER: 76 columns inside the kit's gutters at 80
		{"a footer that overflows", table("| inbox | inbox | ↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · ctrl+s save search · r refresh · ? help · q quit |", ""), map[string]int{"S-FOOTER": 1}},
		{"an overlay's footer overflows too", table("| confirm | overlay | ←→ choose · enter confirm deletion of every selected item · esc cancel and keep the items · ? help |", ""), map[string]int{"S-FOOTER": 1}},
		{"a cut that fits", table("| inbox | inbox | ↑↓/jk move · enter open · e done · s snooze · x select · ? help ‖ E done selected · / filter · ctrl+s save search · r refresh · q quit |", ""), nil},
		{"a cut that still overflows", table("| inbox | inbox | ↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · ctrl+s save search · ? help ‖ r refresh · q quit |", ""), map[string]int{"S-FOOTER": 1}},
		{"a cut with no help key in the footer", table("| inbox | inbox | ↑↓/jk move · enter open ‖ r refresh |", ""), map[string]int{"S-FOOTER": 1}},
		{"a cut with help in Global", table("| inbox | inbox | ↑↓/jk move · enter open ‖ r refresh |", "? help · ctrl+c quit"), nil},
		{"a cut that hides nothing", table("| inbox | inbox | ↑↓/jk move · enter open ‖ |", ""), nil},
		{"help named in a phrase", table("| inbox | inbox | ↑↓/jk move · enter open · f1 show help ‖ r refresh |", ""), nil},
		{"f1 help overflowing", table("| inbox | inbox | ↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · r refresh · f1 help |", ""), map[string]int{"S-FOOTER": 1}},
		{"?/f1 help overflowing", table("| inbox | inbox | ↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · r refresh · ?/f1 help |", ""), map[string]int{"S-FOOTER": 1}},
		{"a bold footer that fits", table("| inbox | inbox | **↑↓/jk move** · **enter open** · **e done** · **? help** · s snooze · x select · E done selected · / filter · ctrl+s save search · r refresh · q quit |", ""), nil},
		{"a bold key alone marks its item", table("| inbox | inbox | **↑↓/jk** move · **enter** open · **?** help · s snooze · x select · E done selected · / filter · ctrl+s save search · r refresh · q quit |", ""), nil},
		{"a bold footer too wide", table("| inbox | inbox | **↑↓/jk move** · **enter open** · **e done** · **s snooze** · **x select** · **E done selected** · **/ filter** · **ctrl+s save search** · **? help** · r refresh · q quit |", ""), map[string]int{"S-FOOTER": 1}},
		{"a bold footer with no help key", table("| inbox | inbox | **↑↓/jk move** · **enter open** · r refresh |", ""), map[string]int{"S-FOOTER": 1}},
		{"bold and ‖ in one cell", table("| inbox | inbox | **↑↓/jk move** · **enter open** · ? help ‖ r refresh |", ""), map[string]int{"S-TABLE": 1}},
		{"bold footers written by a cold agent", table("| tour | pr / tour | **↑↓/jk scroll** · **n next stop** · **d show diff** · **a ask** · **v finish** · **? help** · p previous stop · t conversation · c cards · tab focus inbox · [ back · ] forward · ctrl+p palette · q quit |\n| cards | pr / cards | **1 fine** · **2 question** · **3 blocking** · **e edit note** · **esc back to tour** · **? help** · ↑↓/jk move · v finish · q quit |", ""), nil},
		{"a writing state overflowing", table("| tour-writing | tour | ↑↓/jk scroll · pgup/pgdn page · g/G top/bottom · tab next section · shift+tab previous section · r write the rest · esc stop writing · ? help |", ""), map[string]int{"S-FOOTER": 1}},
		{"ctrl+g opens help while typing", table("| reply | overlay | enter send · ctrl+j newline · esc cancel · ctrl+g help ‖ ctrl+o attach |", ""), nil},
		{"76 columns exactly fit", table("| a | a | a "+strings.Repeat("x", 28)+" · b "+strings.Repeat("y", 42)+" |", ""), nil},
		// ui.Help draws `shift+enter/ctrl+j` as one alternative (ctrl+j, or shift+enter where
		// the terminal disambiguates keys), so S-FOOTER measures the widest one, not all 18
		{"a newline fallback is one alternative wide", table("| compose | conversation | **enter send** · **shift+enter/ctrl+j newline** · **ctrl+o attach** · **esc cancel** · **ctrl+g help** · ctrl+r history |", ""), nil},
		// a footer badge (ui.Footer's left: a mode, PLAN) takes its width and a space from
		// the 76 columns; a row names it as its first item
		{"a badge's row fits without it", table("| chat | chat | **enter send** · **ctrl+s sessions** · **ctrl+t toggle tools** · **esc cancel** · **ctrl+g help** · ctrl+r history |", ""), nil},
		{"a badge takes its share of the footer", table("| chat-plan | chat | [plan] · **enter send** · **ctrl+s sessions** · **ctrl+t toggle tools** · **esc cancel** · **ctrl+g help** · ctrl+r history |", ""), map[string]int{"S-FOOTER": 1}},
		{"a badge is no key", table("| chat-plan | chat | `[plan]` · **enter send** · **esc cancel** · **ctrl+g help** · ctrl+r history |", ""), nil},
		{"a key's condition takes no footer width", table("| a | a | **a "+strings.Repeat("x", 28)+"** · **b "+strings.Repeat("y", 42)+" (if the list is long)** |", ""), nil},
		{"nor does a later key's marker", table("| a | a | a "+strings.Repeat("x", 28)+" · b "+strings.Repeat("y", 42)+" (later: the y screen) |", ""), nil},
		{"77 columns don't", table("| a | a | a "+strings.Repeat("x", 28)+" · b "+strings.Repeat("y", 43)+" |", ""), map[string]int{"S-FOOTER": 1}},
		{"Global keys are not in the footer", table("| list | list | ↑↓ move · enter open · ? help |", "ctrl+c quit · ctrl+p palette · ctrl+o switch · ctrl+s save · ctrl+r reload · ctrl+t theme · f1 manual"), nil},
		{"an empty cell has no footer", table("| splash | splash | — |", ""), nil},
		// the help overlay is full help: it needs no key that opens it; a state drawn with
		// no footer row (toosmall through ui.TooSmallText) shows no keys at all
		{"the help overlay hides keys behind nothing", table("| help | overlay | **esc close** · ↑↓ scroll · ?/h/f1/ctrl+g close |", ""), nil},
		{"help over a tour is help too", table("| help-tour | overlay | esc close ‖ ↑↓ scroll |", ""), nil},
		{"a state with no footer row", table("| toosmall | any | ‖ q quit |", ""), nil},
		// a help item's aliases are for the dispatcher: the footer draws its first key
		// (`? help`), which ui.Help pins
		{"?/h/f1 help counts as ? help", table("| a | a | a "+strings.Repeat("x", 28)+" · b "+strings.Repeat("y", 34)+" · ?/h/f1 help |", ""), nil},
		{"? help that doesn't fit", table("| a | a | a "+strings.Repeat("x", 28)+" · b "+strings.Repeat("y", 35)+" · ?/h/f1 help |", ""), map[string]int{"S-FOOTER": 1}},

		// S-SLOTS: a read from before move and signature replaced remember
		{"remember alone", "### Brand read\ntone: calm\nremember: the orange row\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"remember beside a move", "### Brand read\nremember: the orange row\nmove: the detail leads\n" + table("| list | list | enter open |", ""), nil},
		{"remember beside a signature", "### Brand read\n- remember: the orange row\n- **signature:** a tide strip\n" + table("| list | list | enter open |", ""), nil},
		{"remember in a fence", "### Brand read\n```\nremember: the orange row\n```\n" + table("| list | list | enter open |", ""), nil},
		{"remember in bold", "### Brand read\n**Remember:** the orange flag\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"remember in backticks", "### Brand read\n`remember:` lines are gone; see the move\n" + table("| list | list | enter open |", ""), nil},
		{"remember outside the brand read", "### Brand read\ntone: calm\n\n## Decisions\n- Remember: q quits from every screen\n" + table("| list | list | enter open |", ""), nil},
		{"a move outside the brand read", "### Brand read\nremember: the orange row\n\n### Keys\n- move: up and down\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},

		// S-SLOTS: every § Screens block after the main screen's says what the brand
		// changes there, so two brands on one spec don't share the screens behind it
		{"a secondary screen with no brand line", "### Screens\n#### list — every job\n80×24: list 0,1 80×22\n#### detail — one job\ncomponents: ui.KV\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"two secondary blocks with none", "### Screens\n#### list — every job\n#### detail — one job\n#### confirm — stop it\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 2}},
		{"secondary blocks with brand lines", "### Screens\n#### list — every job\n#### detail — one job\nbrand: terse: labels without verbs, one footer verb per key\n#### confirm — stop it\n- **brand:** the question is the verb and its object\n" + table("| list | list | enter open |", ""), nil},
		{"nothing, because counts", "### Screens\n#### list — every job\n#### help — every key\nbrand: nothing, because help lists the action tables as written\n" + table("| list | list | enter open |", ""), nil},
		{"the main screen needs none", "### Screens\n#### list — every job\n80×24: list 0,1 80×22\n" + table("| list | list | enter open |", ""), nil},
		{"a screen list of one-liners has no blocks", "### Screens\nlist — every job\ndetail — one job\n" + table("| list | list | enter open |", ""), nil},
		{"a brand line in the sketch is not one", "### Screens\n#### list — every job\n#### detail — one job\n```\nbrand: terse\n# 4120 indexer\n```\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"brand in backticks is prose", "### Screens\n#### list — every job\n#### detail — one job\n`brand:` comes later\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"an empty brand line", "### Screens\n#### list — every job\n#### detail — one job\nbrand:\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"a brand line under a sub-heading of its block", "### Screens\n#### list — every job\n#### detail — one job\n##### regions\nbrand: terse\n" + table("| list | list | enter open |", ""), nil},
		{"a bare nothing is no reason", "### Screens\n#### list — every job\n#### detail — one job\nbrand: nothing\n#### help — every key\nbrand: -\n#### confirm — stop it\nbrand: N/A.\n#### toosmall — too small\nbrand: none\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 4}},
		{"nothing with a reason passes", "### Screens\n#### list — every job\n#### help — every key\nbrand: nothing: help lists the action tables as written\n#### toosmall — too small\nbrand: none, because the kit's one sentence is the spec's\n" + table("| list | list | enter open |", ""), nil},
		{"a placeholder is no reason", "### Screens\n#### list — every job\n#### detail — one job\nbrand: TBD\n#### help — every key\nbrand: todo\n#### confirm — stop it\nbrand: ?\n#### toosmall — too small\nbrand: nothing yet\n#### picker — pick one\nbrand: <what the three words change here>\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 5}},
		{"a reason that starts like a placeholder is one", "### Screens\n#### list — every job\n#### detail — one job\nbrand: nothing yet differs from the list, because the row is the detail\n" + table("| list | list | enter open |", ""), nil},
		{"a brand line in a ~~~ fence is not one", "### Screens\n#### list — every job\n#### detail — one job\n~~~\nbrand: terse\n```\n~~~\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"group headings hold blocks", "## Screens\n### Main\n#### list — every job\n### Overlays\n#### confirm — stop it\n#### help — every key\nbrand: nothing, because help lists the tables\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"a sibling screens section continues it", "### Screens\n#### list — every job\n### Secondary screens\n#### detail — one job\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"a numbered screens heading", "### 4. Screens\n#### list — every job\n#### detail — one job\n" + table("| list | list | enter open |", ""), map[string]int{"S-SLOTS": 1}},
		{"dialogs under Architecture are not § Screens", "### Screens\n#### list — every job\n#### detail — one job\nbrand: terse\n## Architecture\n### Dialogs\n#### confirm — ui.Dialog\n" + table("| list | list | enter open |", ""), nil},
		{"screens as models is not § Screens", "## Architecture\n### Screens as models\n#### list.go\n#### detail.go\n" + table("| list | list | enter open |", ""), nil},
		{"blocks after § Screens are not screens", "### Screens\n#### list — every job\n### Motion\n#### hero\n#### rows\n" + table("| list | list | enter open |", ""), nil},

		{"brand section ends at the next heading", "## Brand\n### Brand read\ntone: calm\n## Architecture\nregions ★ ★\n" + table("| list | list | enter open |", ""), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := Lint("TUI.md", c.doc)
			got := rules(fs)
			for _, f := range fs {
				if f.Rule == "S-FOOTER" && strings.Contains(f.Msg, "e.g. `") {
					if again := relint(t, f, ""); len(again) > 0 {
						t.Errorf("the suggested cut still fires:\n%s", dump(again))
					}
				}
			}
			want := c.want
			if want == nil {
				want = map[string]int{}
			}
			if len(got) != len(want) {
				t.Fatalf("got %v, want %v\n%s", got, want, dump(fs))
			}
			for r, n := range want {
				if got[r] != n {
					t.Fatalf("got %v, want %v\n%s", got, want, dump(fs))
				}
			}
		})
	}
}

// Some findings are P2 under a rule that is P1 by default: they must not fail init.
func TestWarningsDontBlock(t *testing.T) {
	for _, doc := range []string{
		table("| list | list | ctrl+h history |", ""),
		table("| list | list | enter open · foo bar |", ""),
	} {
		fs := Lint("TUI.md", doc)
		if len(fs) != 1 || fs[0].Sev != "P2" {
			t.Errorf("want one P2, got\n%s", dump(fs))
		}
	}
}

func dump(fs []Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.Rule + " " + f.Msg + "\n")
	}
	return b.String()
}

func TestLegacy(t *testing.T) {
	cases := map[string]string{ // key → what arrives ("ok" if itself, "" if nothing)
		"shift+enter": "enter", "ctrl+enter": "enter", "ctrl+shift+enter": "enter", "alt+shift+enter": "alt+enter",
		"ctrl+m": "enter", "ctrl+i": "tab", "ctrl+[": "esc", "ctrl+h": "ok", "ctrl+@": "ctrl+space",
		"ctrl+2": "ctrl+space", "ctrl+/": "ctrl+_", "ctrl+shift+k": "ctrl+k", "ctrl+K": "ok",
		"ctrl+tab": "tab", "ctrl+shift+tab": "shift+tab", "shift+space": "space", "ctrl+shift+space": "ctrl+space",
		"shift+backspace": "backspace", "ctrl+backspace": "ctrl+h", "alt+ctrl+backspace": "ctrl+alt+h", "shift+esc": "esc",
		"super+k": "", "meta+x": "", "hyper+a": "", "ctrl+1": "", "ctrl+,": "", "ctrl+.": "",
		"enter": "ok", "alt+enter": "ok", "shift+tab": "ok", "ctrl+space": "ok", "ctrl+j": "ok", "ctrl+k": "ok",
		"ctrl+_": "ok", "ctrl+\\": "ok", "ctrl+]": "ok", "ctrl+^": "ok", "shift+up": "ok", "ctrl+right": "ok",
		"alt+backspace": "ok", "esc": "ok", "q": "ok", "G": "ok", "f5": "ok", "ctrl+f5": "ok",
	}
	for in, want := range cases {
		as, why, ok := legacy(parseKey(in))
		got := as
		if ok {
			got = "ok"
		}
		if got != want {
			t.Errorf("legacy(%s) = %q (%s), want %q", in, got, why, want)
		}
		if !ok && why == "" {
			t.Errorf("legacy(%s) gives no reason", in)
		}
	}
}

func TestReadItem(t *testing.T) {
	cases := map[string]string{ // item → keys | action
		"q / ctrl+c quit":            "q ctrl+c | quit",
		"ctrl+d / ctrl+u half page":  "ctrl+d ctrl+u | half page",
		"j/k, ↑/↓ move":              "j k up down | move",
		"h/l or ←/→ switch":          "h l left right | switch",
		"enter: open":                "enter | open",
		"[ ] back/fwd":               "[ ] | back/fwd",
		"j k scroll":                 "j k | scroll",
		"/ search":                   "/ | search",
		"esc, q close":               "esc q | close",
		"q / go back":                "q | / go back",
		"esc, to close":              "esc | to close",
		"1-3 jump to tab":            "1 2 3 | jump to tab",
		"hjkl move":                  "h j k l | move",
		"<enter> open":               "enter | open",
		"shift+enter/ctrl+j newline": "shift+enter ctrl+j | newline",
		// < and > are keys, not a <name> wrapper round /; + is a key, not a modifier on /
		"</> sort":     "< > | sort",
		"+/- interval": "+ - | interval",
	}
	for in, want := range cases {
		b := readItem(strings.Fields(in))
		var ks []string
		for _, k := range b.keys {
			ks = append(ks, k.String())
		}
		if got := strings.Join(ks, " ") + " | " + b.action; got != want || b.bad != "" {
			t.Errorf("readItem(%q) = %q (bad %q), want %q", in, got, b.bad, want)
		}
	}
	if b := readItem(strings.Fields("foo/x bar")); b.bad != "foo" || len(b.keys) != 1 {
		t.Errorf("readItem(foo/x bar): bad %q, keys %v", b.bad, b.keys)
	}
}

func TestParseToken(t *testing.T) {
	cases := map[string]string{
		"↑↓/jk": "up down j k", "shift+enter/ctrl+j": "shift+enter ctrl+j", "/": "/", "ctrl+/": "ctrl+/",
		"?": "?", "g/G": "g G", "pgup/pgdn": "pgup pgdown", "⏎": "enter", "Ctrl+Shift+K": "ctrl+shift+k",
		"^C": "ctrl+c", "shift+a": "A", "Esc": "esc", "opt+enter": "alt+enter", "←→": "left right", "tab/shift+tab": "tab shift+tab",
		// a modifier on an arrow pair holds for each arrow, and carries to a later bare arrow
		"shift+↑↓": "shift+up shift+down", "shift+↑/↓": "shift+up shift+down", "ctrl+←→": "ctrl+left ctrl+right",
		"alt+↑↓/←→": "alt+up alt+down alt+left alt+right",
		// never onto a character or a named key: "ctrl+c/q" is ctrl+c or q
		"ctrl+c/q": "ctrl+c q", "ctrl+d/f/t": "ctrl+d f t", "shift+a/b": "A b",
		"ctrl+c/esc": "ctrl+c esc", "ctrl+n/ctrl+p": "ctrl+n ctrl+p", "ctrl+j/↑": "ctrl+j up",
		// "</>" is < or >, not the / key in a <name> wrapper; "+/-" is + or -, the + no modifier
		"</>": "< >", "+/-": "+ -", "-/+": "- +", "<enter>": "enter", "<ctrl+c>": "ctrl+c",
	}
	for in, want := range cases {
		var got []string
		for _, k := range parseToken(in) {
			got = append(got, k.String())
		}
		if strings.Join(got, " ") != want {
			t.Errorf("parseToken(%q) = %q, want %q", in, strings.Join(got, " "), want)
		}
	}
}

// The welp-like fixture fires every key and glyph rule; the same file after review is clean.
func TestReviewInbox(t *testing.T) {
	var out, errb bytes.Buffer
	code := Main([]string{"testdata/review-inbox.md"}, &out, &errb)
	if code != 1 {
		t.Errorf("exit %d, want 1\n%s%s", code, out.String(), errb.String())
	}
	golden := filepath.Join("testdata", "review-inbox.golden.txt")
	if *update {
		if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want) {
		t.Errorf("output differs from %s (go test -update to accept):\n%s", golden, out.String())
	}
	for _, r := range []string{"S-KEYAMBIG", "S-KEYDUP", "S-KEYSHADOW", "S-GLYPHWIDTH", "S-FOOTER", "S-SLOTS"} {
		if !strings.Contains(out.String(), r) {
			t.Errorf("review-inbox.md does not fire %s", r)
		}
	}

	out.Reset()
	if code := Main([]string{"testdata/review-inbox-fixed.md"}, &out, &errb); code != 0 || out.String() != "OK  0 blocking, 0 warnings\n" {
		t.Errorf("review-inbox-fixed.md: exit %d\n%s", code, out.String())
	}
}

// cmd-init.md's template, and the template filled in for a product, both lint clean, so
// an agent that follows it starts from a passing TUI.md.
func TestTemplatePasses(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "references", "cmd-init.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(src), "````markdown\n")
	tmpl, _, ok2 := strings.Cut(rest, "````")
	if !ok || !ok2 {
		t.Fatal("cmd-init.md has no ````markdown template block")
	}
	if fs := Lint("cmd-init.md template", tmpl); len(fs) != 0 {
		t.Errorf("the template in cmd-init.md fails tuilint spec:\n%s", dump(fs))
	}
	ex, err := os.ReadFile(filepath.Join("testdata", "template-example.md"))
	if err != nil {
		t.Fatal(err)
	}
	if fs := Lint("template-example.md", string(ex)); len(fs) != 0 {
		t.Errorf("template-example.md fails tuilint spec:\n%s", dump(fs))
	}
	// its secondary blocks pass because they say what the brand changes, not because
	// the rule can't see them
	bare := regexp.MustCompile(`(?m)^brand:.*$`).ReplaceAllString(string(ex), "")
	fs := Lint("template-example.md", bare)
	if got := rules(fs)["S-SLOTS"]; got != 2 {
		t.Errorf("template-example.md without its brand: lines: %d S-SLOTS, want 2 (deploy, confirm)", got)
	}
	// the rule reads the first block as the main screen, and says so when it flags
	for _, f := range fs {
		if !strings.Contains(f.Msg, "put the main screen's block first") {
			t.Errorf("S-SLOTS message doesn't say the main screen's block comes first: %s", f.Msg)
		}
	}
}

// Ordinary TUI.md files lint clean; only real problems fire. Each of these drew false
// findings before: a ▌ gutter and arrows in prose (dashboard), states overriding Global
// esc, ? and ctrl+c (wizard), `q / ctrl+c quit` and `j/k, ↑/↓ move` (git), and a
// "Global:" note under § Decisions (dashboard). Their widest states overflowed the
// 80-column footer (S-FOOTER), so dashboard marks its footer with ‖ and git with bold.
func TestOrdinaryFilesLintClean(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "clean", "*.md"))
	if err != nil || len(files) < 3 {
		t.Fatalf("testdata/clean: %v %v", files, err)
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if fs := Lint(f, string(src)); len(fs) != 0 {
			t.Errorf("%s:\n%s", f, dump(fs))
		}
	}
	// and a real duplicate in one of them still fires
	if fs := Lint("TUI.md", table("| status | status | c commit · c comment |", "ctrl+c quit")); len(fs) != 1 || fs[0].Rule != "S-KEYDUP" || fs[0].Sev != "P1" {
		t.Errorf("c commit · c comment: want one P1 S-KEYDUP, got\n%s", dump(fs))
	}
}

func TestMainExitCodes(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Main([]string{"-h"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "usage: tuilint spec") {
		t.Errorf("-h: exit %d\n%s", code, out.String())
	}
	if code := Main([]string{"--nope"}, &out, &errb); code != 2 {
		t.Errorf("unknown flag: exit %d, want 2", code)
	}
	if code := Main([]string{"testdata/missing.md"}, &out, &errb); code != 2 {
		t.Errorf("missing file: exit %d, want 2", code)
	}
	j := filepath.Join(t.TempDir(), "out.json")
	out.Reset()
	if code := Main([]string{"testdata/review-inbox.md", "--json", j}, &out, &errb); code != 1 {
		t.Errorf("--json: exit %d, want 1", code)
	}
	b, err := os.ReadFile(j)
	if err != nil {
		t.Fatal(err)
	}
	var fs []Finding
	if err := json.Unmarshal(b, &fs); err != nil || len(fs) == 0 || fs[0].Sev != "P1" || fs[0].Where == "" {
		t.Errorf("--json wrote %s (%v)", b, err)
	}
}

// The -h text is the package doc comment, whole and without source.
func TestUsageIsTheDocComment(t *testing.T) {
	u := Usage()
	for _, want := range []string{"usage: tuilint spec [TUI.md...]", "S-KEYAMBIG", "S-KEYDUP", "S-KEYSHADOW", "S-GLYPHWIDTH", "S-FOOTER", "S-TABLE", "‖",
		"  | state (harness name) | screen / overlay | keys"} {
		if !strings.Contains(u, want) {
			t.Errorf("usage lacks %q:\n%s", want, u)
		}
	}
	if strings.Contains(u, "package speclint") || strings.Contains(u, "//") {
		t.Errorf("usage leaks source:\n%s", u)
	}
	for id := range Severity {
		if !strings.Contains(u, id+" ") {
			t.Errorf("usage does not document %s", id)
		}
	}
}

// packFooter lays hints out as kit/ui Help does: two spaces apart, 3 columns kept for
// "…" while hints remain, and the first ? or ctrl+g hint pinned last, wherever it is.
func TestPackFooter(t *testing.T) {
	cases := []struct {
		items   string
		w       int
		dropped string
	}{
		{"a one · b two", 12, ""},
		{"a one · b two", 11, "b two"},
		{"a one · b two · c three", 16, "c three"},         // a one + b two = 12, + reserve 3 = 15 ≤ 16, then c three needs 9
		{"a one · b two · c three", 14, "b two · c three"}, // 12 + reserve 3 > 14
		{"a one · b two · ? help", 13, "b two"},            // ? help pinned: 13 - 8 = 5 for the rest
		{"a one · ? help · b two", 13, "b two"},            // ? help pinned from the middle too
		{"a very long first hint", 10, "a very long first hint"},
		{"shift+enter/ctrl+j newline · b two", 26, ""}, // measured as shift+enter newline: 19 + 2 + 5
		{"shift+enter/ctrl+j newline · b two", 25, "b two"},
		{"↑↓/jk move · b two", 13, "b two"}, // alternatives every terminal sends are all drawn
	}
	for _, c := range cases {
		_, dropped := packFooter(strings.Split(c.items, " · "), c.w)
		if got := strings.Join(dropped, " · "); got != c.dropped {
			t.Errorf("packFooter(%q, %d) drops %q, want %q", c.items, c.w, got, c.dropped)
		}
	}
}

// kit/ui/testdata/footer-cases.json is what the kit's ui.Footer drops at 80 columns
// (kit/ui TestFooterCases renders it), on the default theme or, for a case marked
// ascii, on it resolved for ASCII; packFooter must drop the same hints.
func TestFooterCases(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "kit", "ui", "testdata", "footer-cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name           string
		ASCII          bool
		Hints, Dropped []string
	}
	if err := json.Unmarshal(b, &cases); err != nil || len(cases) == 0 {
		t.Fatalf("footer-cases.json: %v", err)
	}
	for _, c := range cases {
		reserve := moreReserve
		if c.ASCII {
			reserve = asciiReserve
		}
		_, dropped := packFooterReserving(c.Hints, footerBudget, reserve)
		if strings.Join(dropped, "|") != strings.Join(c.Dropped, "|") {
			t.Errorf("%s: packFooter drops %q, the kit drops %q", c.Name, dropped, c.Dropped)
		}
	}
}

var suggestionRe = regexp.MustCompile("e\\.g\\. `(.*)`$")

// relint lints the cell an S-FOOTER finding suggests, in place of the row, and returns
// any S-FOOTER it draws: a suggestion must fix what it was given for.
func relint(t *testing.T, f Finding, global string) []Finding {
	t.Helper()
	m := suggestionRe.FindStringSubmatch(f.Msg)
	if m == nil {
		t.Fatalf("S-FOOTER suggests no cell: %s", f.Msg)
	}
	var out []Finding
	for _, g := range Lint("TUI.md", table("| s | s | "+m[1]+" |", global)) {
		if g.Rule == "S-FOOTER" {
			out = append(out, g)
		}
	}
	return out
}

func TestSuggestCut(t *testing.T) {
	cases := []struct{ name, cell, front, back string }{
		{"f1 help is not pinned, so it is packed last",
			"↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · r refresh · f1 help",
			"↑↓/jk move · enter open · e done · s snooze · x select · E done selected · f1 help", "/ filter · r refresh"},
		{"?/f1 help is drawn as ? help, pinned last",
			"↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · r refresh · ?/f1 help",
			"↑↓/jk move · enter open · e done · s snooze · x select · E done selected · ?/f1 help", "/ filter · r refresh"}, // as narrow as f1 help
		{"the way out and the state's purpose stay in front",
			"↑↓/jk scroll · pgup/pgdn page · g/G top/bottom · tab next section · shift+tab previous section · r write the rest · esc stop writing · ? help",
			"↑↓/jk scroll · pgup/pgdn page · r write the rest · esc stop writing · ? help", "g/G top/bottom · tab next section · shift+tab previous section"},
		{"quit stays, secondary actions go from the end",
			"↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · ctrl+s save search · tab next filter · r refresh · ? help · q quit",
			"↑↓/jk move · enter open · e done · s snooze · x select · q quit · ? help", "E done selected · / filter · ctrl+s save search · tab next filter · r refresh"},
		{"roll back is an action, not a way out",
			"↑↓/jk move · enter open · d deploy · r roll back · ctrl+p commands · l logs · c copy id · s sort by status · q quit",
			"↑↓/jk move · enter open · d deploy · r roll back · q quit · ? help", "ctrl+p commands · l logs · c copy id · s sort by status"},
	}
	for _, c := range cases {
		front, back := suggestCut(hints(c.cell), footerBudget)
		if got, want := strings.Join(front, " · ")+" ‖ "+strings.Join(back, " · "), c.front+" ‖ "+c.back; got != want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, want)
		}
	}
}

// S-ACCENT: an accents line that gives the hue one meaning, then lines that put "the
// accent" or "primary" on something else (the welp brief: amber = new, then the wordmark,
// the dots and a refresh ramp in the accent). A line that uses the hue for its meaning
// passes; the template's placeholder accents line says nothing.
func TestAccentDrift(t *testing.T) {
	brand := func(lines ...string) string {
		return "# x TUI\n\n### Brand read\n" + strings.Join(lines, "\n") + "\n\n" +
			table("| inbox | inbox | enter open · q quit |", "")
	}
	drift := brand("accents: amber = new, nothing else; focus = weight", "remember: the wordmark in the accent",
		"", "### Motion", "hero: refresh → dots in the primary hue, 20 cells")
	fs := Lint("TUI.md", drift)
	var hits []Finding
	for _, f := range fs {
		if f.Rule == "S-ACCENT" {
			hits = append(hits, f)
		}
	}
	if len(hits) != 2 || hits[0].Sev != "P2" || hits[0].Where != "5" || hits[1].Where != "8" ||
		!strings.Contains(hits[0].Msg, "`remember: the wordmark in the accent`") || !strings.Contains(hits[0].Msg, "line 4 says `accents: amber = new, nothing else; focus = weight`") {
		t.Errorf("drift: %s", dump(fs))
	}
	for name, doc := range map[string]string{
		"the hue's own meaning": brand("accents: amber = new, nothing else", "remember: new rows take the accent; the wordmark is fg, bold", "motion: the refresh dots are fgMuted; the accent only on new"),
		"the hue named":         brand("accents: amber = new", "remember: amber on the unread count, the primary accent of the screen is amber"),
		"no accents line":       brand("palette: slate and amber", "remember: the wordmark in the accent"),
		"placeholder":           brand("accents: <hue> = <its one meaning>, one line per hue", "remember: the wordmark in the accent"),
		// two hues on one line, ;-separated: the second pair's meaning counts too
		"two hues, one line": brand("accents: amber = new, nothing else; teal = running", "motion: the dots spin in the accent while a fetch is running"),
		// a second accents line adds its pair
		"two accents lines": brand("accents: amber = new", "accents: teal = running", "motion: running rows pulse in the accent"),
		// primary as a word for something that isn't the hue
		"primary pane":   brand("accents: amber = new", "focus: the primary pane keeps the cursor; the secondary pane is dim"),
		"primary key":    brand("accents: amber = new", "the primary key of a row is its id, shown in fgMuted"),
		"primary button": brand("accents: amber = new", "buttons: the primary button is bold fg, the rest fgMuted"),
	} {
		if n := rules(Lint("TUI.md", doc))["S-ACCENT"]; n != 0 {
			t.Errorf("%s: %d S-ACCENT, want 0:\n%s", name, n, dump(Lint("TUI.md", doc)))
		}
	}
}

// S-GOLDEN: with --frames, each state's footer in <state>-80x24.ansi against its row.
// The recreate run's empty table advertised its row keys (`↑↓/jk move · enter edit`)
// while TUI.md's empty row said `a add · esc back`, and TestHelpIsHandled passed.
// A footer hint is a key drawn in another style than its action, as ui.Help draws it,
// so status text on the row is not one; ASCII twins (^v, <>) read as their arrows; a
// row of only a toast compares nothing; a … means Help dropped the rest; a badge drawn
// one blank before the first hint ("[PROD] enter next field") is not part of it.
func TestGoldenFooters(t *testing.T) {
	key, desc, off := "\x1b[1;38;2;230;230;230m", "\x1b[22;38;2;140;140;140m", "\x1b[0m"
	footer := func(hints ...string) string { // each "key action", drawn as ui.Help does
		var b strings.Builder
		b.WriteString("  ")
		for i, h := range hints {
			if i > 0 {
				b.WriteString("  ")
			}
			k, d, _ := strings.Cut(h, " ")
			b.WriteString(key + k + off + " " + desc + d + off)
		}
		return b.String()
	}
	doc := table(strings.Join([]string{
		"| list | list | **↑↓/jk move** · **enter open** · **a add** · **? help** · d delete |",
		"| list-empty | list | **a add** · **esc back** · **? help** |",
		"| list-busy | list | as list; **x cancel** |",
		"| list-toast | list | **a add** · **? help** |",
		"| confirm | overlay | ←→ choose · enter confirm · esc cancel |",
		"| form | form | enter next field · esc back ‖ tab/shift+tab field |",
		"| list-selected | list | **↑↓/jk move** · **enter open** · **? help** · d delete |",
		"| list-picked | list | **↑↓/jk move** · **enter open** |",
		"| grid | overlay | **↑↓←→ pick** · **enter send** · **esc back** |",
		"| list-top | list | **↓ select** · **/ filter** · **? help** |",
		"| list-filter | list | **enter keep** · **esc drop** · **ctrl+g help** · backspace erase |",
		"| inbox | inbox | **↑↓ move** · **esc clear filter (if a filter is applied)** · **ctrl+p palette** (later) · **? help** |",
		"| reply | overlay | **enter send** · **↑↓/jk scroll (while the body overflows)** · **esc cancel** · **a ask (later: ask)** |",
		"| inbox-stale | inbox | **↑↓ move** · **d done (if read)** · **? help** |",
	}, "\n"), "ctrl+c quit")
	cases := []struct {
		name, frame string
		want        []string // substrings of the finding for this state; none when empty
	}{
		{"list", footer("↑↓/jk move", "enter open", "a add", "? help") + "  " + desc + "5 rows" + off, nil},
		{"list-empty", footer("↑↓/jk move", "enter open", "a add", "esc back", "? help"), []string{"shows `↑↓/jk move`, `enter open`, which the row doesn't bind"}},
		{"list-busy", footer("↑↓/jk move", "enter open", "x cancel", "…", "? help"), nil},
		{"list-toast", "  " + desc + "copied 6 lines" + off, nil},
		{"confirm", footer("<> choose", "enter confirm"), []string{"lacks `esc cancel`, which the row puts in the footer"}},
		// a badge one blank before the first hint, in its own style, is not part of it
		{"form", "  \x1b[1;38;2;207;162;78m[PROD]" + off + " " + strings.TrimPrefix(footer("enter next field", "esc back"), "  "), nil},
		// ui.Footer's right status after the pinned help hint ("4 of 101", its count in
		// the key's ink) is status, not a hint; so is an "N of M" count with no help hint
		{"list-selected", footer("↑↓/jk move", "enter open", "? help") + "  " + key + "4" + off + desc + " of 101" + off, nil},
		{"list-picked", footer("↑↓/jk move", "enter open") + "  " + key + "1" + off + desc + " of 9" + off, nil},
		// ASCII draws the four arrows as ^v<>; NO_COLOR draws a prompt one blank before
		// the first hint with no style to tell them apart
		{"grid", footer("^v<> pick", "enter send", "esc back"), nil},
		{"list-top", footer("v select", "/ filter", "? help"), nil},
		{"list-filter", "  /yes| enter keep  esc drop        ctrl+g help  9 rows", nil},
		// a key the footer shows only under a condition, `(if …)`, `(when …)` or `(while
		// …)`, or whose screen comes later in the slice, `(later)`, may be missing from a
		// golden; a key the row doesn't bind at all is still one the footer offers falsely
		{"inbox", footer("↑↓ move", "? help"), nil},
		{"reply", footer("enter send", "esc cancel"), nil},
		{"inbox-stale", footer("↑↓ move", "x close", "? help"), []string{"shows `x close`, which the row doesn't bind"}},
	}
	dir := t.TempDir()
	for _, c := range cases {
		frame := strings.Repeat("\n", 23) + c.frame + "\n"
		if err := os.WriteFile(filepath.Join(dir, c.name+"-80x24.ansi"), []byte(frame), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := map[string]string{}
	for _, f := range LintFrames("TUI.md", doc, dir) {
		if f.Rule == "S-GOLDEN" {
			if f.Sev != "P2" {
				t.Errorf("S-GOLDEN is %s, want P2", f.Sev)
			}
			state, _, _ := strings.Cut(f.Msg, ":")
			got[state] = f.Msg
		}
	}
	for _, c := range cases {
		msg, ok := got[c.name]
		if len(c.want) == 0 && ok {
			t.Errorf("%s: %s, want no finding", c.name, msg)
		}
		if len(c.want) > 0 && !ok {
			t.Errorf("%s: no finding, want %q", c.name, c.want)
		}
		for _, w := range c.want {
			if ok && !strings.Contains(msg, w) {
				t.Errorf("%s: %s, want %q", c.name, msg, w)
			}
		}
	}
	// without --frames nothing is compared, and --frames must name a directory
	if fs := Lint("TUI.md", doc); rules(fs)["S-GOLDEN"] != 0 {
		t.Errorf("Lint without frames: %v", rules(fs))
	}
	var out, errb bytes.Buffer
	tui := filepath.Join(dir, "TUI.md")
	if err := os.WriteFile(tui, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Main([]string{tui, "--frames", filepath.Join(dir, "list-80x24.ansi")}, &out, &errb); code != 2 {
		t.Errorf("--frames on a file: exit %d, want 2", code)
	}
	out.Reset()
	if code := Main([]string{tui, "--frames=" + dir}, &out, &errb); code != 0 || !strings.Contains(out.String(), "S-GOLDEN") {
		t.Errorf("--frames=dir: exit %d\n%s", code, out.String())
	}
}

// S-REGION: with --frames, a region of a screen's budget line whose whole rectangle is
// blank in <screen>-<W>x<H>.ansi: the layout moved and the budget went stale (a recreate
// run's response pane drawn 8 rows higher than TUI.md said). A gap is blank by design; a
// region with any text is not reported, even when the text is another region's (a status
// line under a transcript touches it legitimately).
func TestRegionBudgets(t *testing.T) {
	doc := "# x TUI\n\n### Screens\n#### list — every job\n" +
		"80×24:  header 0,0 80×1 · gap 0,1 80×1 · list 0,2 80×10 · status 0,20 80×2 · footer 0,23 80×1\n" +
		"120×40: header 0,0 120×1 · list 0,2 120×30 · footer 0,39 120×1\n" +
		"#### detail — one item\nbrand: nothing, because it is the list's row whole\n" +
		"80×24:  pane 40,0 40×23\n" +
		table("| list | list | enter open |", "")
	dir := t.TempDir()
	rows := make([]string, 24)
	for i := range rows {
		rows[i] = strings.Repeat(" ", 80)
	}
	rows[0] = fmt.Sprintf("%-80s", "  app · list")
	for i := 2; i < 12; i++ {
		rows[i] = fmt.Sprintf("%-80s", fmt.Sprintf("  item %d", i))
	}
	rows[12] = fmt.Sprintf("%-80s", "  status: 10 items") // the status line drawn right under the list
	rows[23] = fmt.Sprintf("%-80s", "  enter open")
	if err := os.WriteFile(filepath.Join(dir, "list-80x24.ansi"), []byte(strings.Join(rows, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got []Finding
	for _, f := range LintFrames("TUI.md", doc, dir) {
		if f.Rule == "S-REGION" {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Sev != "P2" || got[0].Where != "5" || !strings.Contains(got[0].Msg, "`status 0,20 80×2`") || !strings.Contains(got[0].Msg, "list-80x24.ansi") {
		t.Errorf("a blank status region: %+v, want one P2 S-REGION on line 5 naming `status 0,20 80×2` and the frame", got)
	}
	if fs := Lint("TUI.md", doc); rules(fs)["S-REGION"] != 0 {
		t.Errorf("without --frames: %v", rules(fs))
	}
}
