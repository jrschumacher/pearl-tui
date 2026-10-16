package speclint

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
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

func TestRules(t *testing.T) {
	cases := []struct {
		name, doc string
		want      map[string]int // rule → count; rules not named must not fire
	}{
		// S-TABLE
		{"no table", "# x TUI\n\nkeys: enter open\n", map[string]int{"S-TABLE": 1}},
		{"table", table("| list | list | enter open · q quit |", ""), nil},
		{"table with bold global and backticks", table("| list | list | `enter` open |", "") + "\n**Global:** `ctrl+c` quit\n", nil},

		// S-KEYAMBIG fires
		{"shift+enter", table("| reply | overlay | enter send · shift+enter newline |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+enter", table("| reply | overlay | ctrl+enter send |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+m", table("| list | list | ctrl+m mark |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+i", table("| list | list | ctrl+i info |", ""), map[string]int{"S-KEYAMBIG": 1}},
		{"ctrl+h", table("| list | list | ctrl+h history |", ""), map[string]int{"S-KEYAMBIG": 1}},
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
		{"keys every terminal sends", table("| list | list | shift+tab back · ctrl+space mark · alt+enter run · ctrl+j down · ctrl+_ undo · ctrl+\\ split · ctrl+k up · alt+backspace kill |", ""), nil},
		{"the slash key and ctrl+", table("| list | list | / filter · ? help · ^R refresh · Ctrl+K up |", ""), nil},

		// S-KEYDUP
		{"dup in a state", table("| list | list | esc back · esc clear |", ""), map[string]int{"S-KEYDUP": 1}},
		{"dup across alternatives", table("| list | list | ↑↓/jk move · k kill |", ""), map[string]int{"S-KEYDUP": 1}},
		{"a state overrides a Global key", table("| list | list | ctrl+k kill line |", "ctrl+k switch"), nil},
		{"overrides of esc, ? and ctrl+c", table("| confirm | overlay | enter confirm · esc cancel |\n| help | overlay | ? close · esc close |\n| creating | wizard | ctrl+c cancel |", "esc back · ? help · ctrl+c quit"), nil},
		{"one cell, two actions", table("| status | status | c commit · c comment |", ""), map[string]int{"S-KEYDUP": 1}},
		{"one cell, near-alike actions", table("| s | s | p preview · p previous · u unstage · u unstash · o open · o open in browser · pgdn page down · pgdn page up |", ""), map[string]int{"S-KEYDUP": 4}},
		{"one cell, the same action reworded by punctuation", table("| s | s | [ ] back/fwd · [ back fwd |", ""), nil},
		{"Global, two actions", table("| a | a | x go |", "c commit · c comment"), map[string]int{"S-KEYDUP": 1}},
		{"a state restating a Global pair in fewer words", table("| pr | pr | t tour · [ ] back/fwd |", "ctrl+c quit, [ ] back/forward between items."), nil},
		{"a pair written with a space", table("| list | list | j k scroll · k kill |", ""), map[string]int{"S-KEYDUP": 1}},
		{"dup in Global, once", table("| a | a | x go |\n| b | b | y go |", "ctrl+k switch · ctrl+k palette"), map[string]int{"S-KEYDUP": 1}},
		{"shift+g is G", table("| list | list | G bottom · shift+g end |", ""), map[string]int{"S-KEYDUP": 1}},
		{"same action twice", table("| list | list | ? help · q quit |", "? help · ctrl+c quit"), nil},
		{"same key in two states", table("| a | a | r refresh |\n| b | b | r reply |", ""), nil},
		{"g and G differ", table("| list | list | g top · G bottom |", ""), nil},

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
		{"motif with no ascii", "### Brand read\nmotif: ● unread ○ done\n" + table("| list | list | enter open |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"glyph in the key table", table("| list | list ● | enter open ◆ |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"reported once per glyph", "### Brand read\nmotif: ●\nremember: the ● row\n" + table("| list | list | enter open |", ""), map[string]int{"S-GLYPHWIDTH": 1}},
		{"ascii twin on the line", "### Brand read\nmotif: ● unread ○ done (ascii * o)\nremember: the ● row\n" + table("| list | list ● | enter open |", ""), nil},
		{"ascii twin elsewhere", "### Brand read\nmotif: ● unread\n\n### Glyphs\nASCII: ● → *\n" + table("| list | list | enter open |", ""), nil},
		{"arrow key names", table("| list | list | ↑↓/jk move · ←→ choose |", ""), nil},
		{"outside the brand read", "### Screens\nlist — ● rows\n\n### Brand read\ntone: calm — precise…\n" + table("| list | list | enter open |", ""), nil},
		{"narrow symbols", "### Brand read\nmotif: ✓ done ❯ cursor ▸ open\n" + table("| list | list | enter open |", ""), nil},
		{"brand section ends at the next heading", "## Brand\n### Brand read\ntone: calm\n## Architecture\nregions ─ ─\n" + table("| list | list | enter open |", ""), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := Lint("TUI.md", c.doc)
			got := rules(fs)
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
		"ctrl+m": "enter", "ctrl+i": "tab", "ctrl+[": "esc", "ctrl+h": "backspace", "ctrl+@": "ctrl+space",
		"ctrl+2": "ctrl+space", "ctrl+/": "ctrl+_", "ctrl+shift+k": "ctrl+k", "ctrl+K": "ok",
		"ctrl+tab": "tab", "ctrl+shift+tab": "shift+tab", "shift+space": "space", "ctrl+shift+space": "ctrl+space",
		"shift+backspace": "backspace", "ctrl+backspace": "backspace", "shift+esc": "esc",
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

func TestParseToken(t *testing.T) {
	cases := map[string]string{
		"↑↓/jk": "up down j k", "shift+enter/ctrl+j": "shift+enter ctrl+j", "/": "/", "ctrl+/": "ctrl+/",
		"?": "?", "g/G": "g G", "pgup/pgdn": "pgup pgdown", "⏎": "enter", "Ctrl+Shift+K": "ctrl+shift+k",
		"^C": "ctrl+c", "shift+a": "A", "Esc": "esc", "opt+enter": "alt+enter", "←→": "left right", "tab/shift+tab": "tab shift+tab",
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
	for _, r := range []string{"S-KEYAMBIG", "S-KEYDUP", "S-KEYSHADOW", "S-GLYPHWIDTH"} {
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
	if err := json.Unmarshal(b, &fs); err != nil || len(fs) == 0 || fs[0].Sev != "P1" || fs[0].Line == 0 {
		t.Errorf("--json wrote %s (%v)", b, err)
	}
}

// The -h text is the package doc comment, whole and without source.
func TestUsageIsTheDocComment(t *testing.T) {
	u := Usage()
	for _, want := range []string{"usage: tuilint spec [TUI.md...]", "S-KEYAMBIG", "S-KEYDUP", "S-KEYSHADOW", "S-GLYPHWIDTH", "S-TABLE",
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
