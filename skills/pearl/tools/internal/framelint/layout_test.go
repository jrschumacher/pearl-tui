package framelint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// layoutFrame lays rows out on a w×h page: each row padded to w, blank rows to h.
// A row may carry SGR; its visible width is what is padded.
func layoutFrame(t *testing.T, w, h int, rows map[int]string) Frame {
	t.Helper()
	var b strings.Builder
	for y := range h {
		r := rows[y]
		vis := len([]rune(stripSGR(r)))
		if vis > w {
			t.Fatalf("row %d is %d cells, wider than %d: %q", y, vis, w, r)
		}
		b.WriteString(r + strings.Repeat(" ", w-vis))
		if y < h-1 {
			b.WriteString("\n")
		}
	}
	f := parseText(b.String())
	for y, row := range f.Rows {
		if len(row) != w {
			t.Fatalf("row %d parsed to %d cells, want %d (wide glyph in a fixture?)", y, len(row), w)
		}
	}
	return f
}

func stripSGR(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func rules(fs []Finding, rule string) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

// F-FOOTWRAP: a key footer whose hints spill onto a second row at the bottom of the
// frame blocks; a status row over the keys, a help listing's key column and a help
// grid of keys in columns do not, nor two key rows of which the upper had room.
func TestFootWrap(t *testing.T) {
	const content = "  Profile ─────────────────────────────────────────"
	cases := []struct {
		name string
		w, h int
		rows map[int]string
		want int
	}{
		{"wrapped at 80", 80, 24, map[int]string{0: " quayside", 2: content,
			22: " enter next  arrows choose  shift+tab back  ctrl+b Back screen  esc cancel",
			23: " ctrl+c quit"}, 1},
		{"wrapped, a blank row under it", 80, 24, map[int]string{0: " quayside", 2: content,
			21: " enter next  arrows choose  shift+tab back  ctrl+b Back screen  esc cancel",
			22: " ctrl+c quit"}, 1},
		{"wrapped, key and action two spaces apart", 60, 16, map[int]string{0: "  app", 2: content[:40],
			14: "  enter  next   ↑↓  choose   shift+tab  back   esc  cancel",
			15: "  ctrl+c  quit   ?  keys"}, 1},
		{"wrapped at a narrow width", 50, 14, map[int]string{0: " quayside", 2: content[:40],
			12: " enter next  arrows choose  shift+tab back",
			13: " ctrl+b Back screen  esc cancel  ctrl+c quit"}, 1},
		{"one row", 80, 24, map[int]string{0: " quayside", 2: content,
			23: " enter next  arrows choose  shift+tab back  esc cancel  ctrl+c quit"}, 0},
		{"the kit's: a status row over the keys", 80, 24, map[int]string{0: "  app", 2: content,
			22: "  cluster staging-eu-west-1 / namespace payments / sign-in Browser / alerts o…",
			23: "   SETUP  enter next  ↑↓ choose  esc back                ? keys  not saved yet"}, 0},
		{"the kit's: a status row with · over the keys", 80, 24, map[int]string{0: "  app", 2: content,
			22: "  cluster staging-eu-west-1 · namespace payments · sign-in Browser · alerts o…",
			23: "  ↑↓ select  d deploy  r roll back  ctrl+p commands  …  ? keys  syncs every 5s"}, 0},
		{"the kit's: a rule over the keys", 80, 24, map[int]string{0: "  app", 2: content,
			22: "  " + strings.Repeat("─", 76),
			23: "  ↑↓ select  d deploy  r roll back  ctrl+p commands  …  ? keys  syncs every 5s"}, 0},
		{"a help listing's key column over the footer", 80, 24, map[int]string{0: "  app", 2: content,
			19: "               ↑↓          select",
			20: "               d           deploy",
			21: "               shift+tab   previous view",
			22: "               ctrl+p      commands",
			23: "  esc close  ↑↓ scroll"}, 0},
		{"a help grid of keys in columns", 80, 24, map[int]string{0: "  app", 2: content,
			20: "  ↑/k up             / filter               q quit",
			21: "  ↓/j down           esc clear filter       ? close help",
			22: "  g/home go to start  enter open            ctrl+p commands  tab next view",
			23: "  G/end go to end    space select           ctrl+c force quit  shift+tab back"}, 0},
		{"two key rows, the upper with room", 80, 24, map[int]string{0: "  app", 2: content,
			22: "  enter next  esc back",
			23: "  ctrl+c quit"}, 0},
		{"key rows mid-frame", 80, 24, map[int]string{0: "  app", 2: content,
			10: " enter next  arrows choose  shift+tab back  ctrl+b Back screen  esc cancel",
			11: " ctrl+c quit",
			23: "  ? help"}, 0},
		{"prose at the bottom", 80, 24, map[int]string{0: "  app", 2: content,
			22: "  Profile staging-eu saved to ~/.config/quayside/profiles.toml; it is now the",
			23: "  default. Run quayside status to check it."}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := LintFootWrap("setup-80x24.ansi", layoutFrame(t, c.w, c.h, c.rows))
			if len(got) != c.want {
				t.Fatalf("%d findings, want %d: %+v", len(got), c.want, got)
			}
			if c.want > 0 && (got[0].Sev != "P1" || got[0].Rule != "F-FOOTWRAP") {
				t.Errorf("finding %+v, want a P1 F-FOOTWRAP", got[0])
			}
		})
	}
}

func TestIsKey(t *testing.T) {
	for _, k := range []string{"enter", "esc", "tab", "space", "backspace", "pgup", "pgdn", "home", "end",
		"arrows", "↑↓", "←→", "↑", "q", "?", "/", "G", "1", "ctrl+c", "alt+enter", "shift+tab", "C-x",
		"^c", "f5", "j/k", "↑/k", "g/home", "[enter]", "<esc>"} {
		if !isKey(k) {
			t.Errorf("isKey(%q) = false, want true", k)
		}
	}
	for _, k := range []string{"cluster", "SETUP", "·", "│", "|", "…", "-", "–", "next", "Back", "keys",
		"staging-eu", "5s", "ctrl+", "//", "ʃ", "[", "]"} {
		if isKey(k) {
			t.Errorf("isKey(%q) = true, want false", k)
		}
	}
}

// F-SPREAD: a step column spread evenly over the height, and an action row stranded
// at the bottom far under its form, warn; packed steps, a summary of label, value and
// blank with tags beside the labels, an action row under its form, the kit's status
// row over the footer, a splash and a backdrop field do not.
func TestSpread(t *testing.T) {
	steps := func(rows map[int]string, at ...int) map[int]string {
		names := []string{"✓ Profile", "2 Connection", "3 Access", "4 Preferences", "5 Review", "6 Done"}
		for i, y := range at {
			rows[y] = " " + names[i]
		}
		return rows
	}
	button := "\x1b[48;2;108;142;191m  \x1b[38;2;255;255;255mCheck connection & continue\x1b[39m  \x1b[m"
	footer := " enter next  esc cancel  ctrl+c quit"
	cases := []struct {
		name  string
		frame string
		w, h  int
		rows  map[int]string
		want  []string // where each finding is
	}{
		{"steps spread over 40 rows", "connection-120x40.ansi", 120, 40,
			steps(map[int]string{0: " quayside", 39: footer}, 2, 9, 16, 23, 30), []string{"r2c1"}},
		{"steps spread over 24 rows", "connection-80x24.ansi", 80, 24,
			steps(map[int]string{0: " quayside", 23: footer}, 2, 6, 10, 14, 18), []string{"r2c1"}},
		{"steps spread, a form beside them", "connection-120x40.ansi", 120, 40,
			steps(map[int]string{0: " quayside", 39: footer, 3: "   staging-eu",
				4: strings.Repeat(" ", 34) + "┃ Destination", 5: strings.Repeat(" ", 34) + "┃ > Quayside Cloud"},
				2, 9, 16, 23, 30), []string{"r2c1"}},
		{"a button stranded at the bottom", "connection-80x24.ansi", 80, 24, map[int]string{0: " quayside", 23: footer,
			5: "    Endpoint", 7: "    https://api.quayside.example/v2", 21: "    " + button}, []string{"r21c4"}},
		{"a [ bracketed ] button stranded at the bottom", "connection-80x24.ansi", 80, 24, map[int]string{0: " quayside", 23: footer,
			5: "    Endpoint", 7: "    https://api.quayside.example/v2", 20: "    ›[ Check connection ]   [ Back ]"}, []string{"r20c4"}},
		{"both", "connection-120x40.ansi", 120, 40,
			steps(map[int]string{0: " quayside", 39: footer, 4: strings.Repeat(" ", 36) + "Endpoint",
				6: strings.Repeat(" ", 36) + "https://api.quayside.example/v2", 37: strings.Repeat(" ", 36) + button},
				2, 9, 16, 23, 30), []string{"r2c1", "r37c36"}},
		{"steps packed from the top", "collect-120x40.ansi", 120, 40,
			steps(map[int]string{0: " lanternfish", 39: footer}, 2, 3, 4, 5, 6, 7), nil},
		{"steps in groups, a blank row apart", "collect-80x24.ansi", 80, 24,
			steps(map[int]string{0: " lanternfish", 23: footer}, 2, 4, 6, 8, 10, 12), nil},
		{"uneven gaps", "collect-80x24.ansi", 80, 24,
			steps(map[int]string{0: " lanternfish", 23: footer}, 2, 6, 14, 18), nil},
		{"three spread steps", "collect-80x24.ansi", 80, 24,
			steps(map[int]string{0: " lanternfish", 23: footer}, 2, 8, 14), nil},
		{"a summary: label, value, blank, a tag beside each label", "setup-target-120x40.ansi", 120, 40, map[int]string{
			0: "  app", 39: footer,
			4:  strings.Repeat(" ", 75) + "Summary ───────────────",
			6:  strings.Repeat(" ", 75) + "Cluster" + strings.Repeat(" ", 29) + "default",
			7:  strings.Repeat(" ", 75) + "staging-eu-west-1",
			8:  strings.Repeat(" ", 75) + "a second line",
			9:  strings.Repeat(" ", 75) + "a third line",
			10: strings.Repeat(" ", 75) + "Namespace" + strings.Repeat(" ", 27) + "default",
			11: strings.Repeat(" ", 75) + "default",
			12: strings.Repeat(" ", 75) + "a second line",
			13: strings.Repeat(" ", 75) + "a third line",
			14: strings.Repeat(" ", 75) + "Sign-in" + strings.Repeat(" ", 29) + "default",
			15: strings.Repeat(" ", 75) + "Browser",
			16: strings.Repeat(" ", 75) + "a second line",
			17: strings.Repeat(" ", 75) + "a third line",
			18: strings.Repeat(" ", 75) + "Alerts" + strings.Repeat(" ", 30) + "default",
			19: strings.Repeat(" ", 75) + "on for failed deploys"}, nil},
		{"a button under its form", "deploy-80x24.ansi", 80, 24, map[int]string{0: "  app", 23: footer,
			5: "    Endpoint", 7: "    https://api.quayside.example/v2", 9: "    " + button}, nil},
		{"the kit's status row over the footer", "setup-auth-80x24.ansi", 80, 24, map[int]string{0: "  app", 23: footer,
			6: "  ▌ Sign in with", 7: "  ▌ › Browser", 21: "  cluster staging-eu-west-1 · namespace payments · sign-in Browser"}, nil},
		{"the kit's toast over the footer", "services-toast-80x24.ansi", 80, 24, map[int]string{0: "  app", 23: footer,
			6: "    SERVICE   VERSION", 7: "    checkout-api   v2.3.9",
			22: "  \x1b[48;2;40;120;60m DONE \x1b[m Rolled back checkout-api from v2.4.0 to v2.3.9"}, nil},
		{"a splash", "splash-120x40.ansi", 120, 40,
			steps(map[int]string{0: " quayside", 39: footer}, 2, 9, 16, 23, 30), nil},
		{"a backdrop field", "field-120x40.ansi", 120, 40, map[int]string{0: "  app", 39: footer,
			2:  "ʃ    ʃ    ʃ    ʃ    ʃ    ʃ",
			10: "ʃ    ʃ    ʃ    ʃ    ʃ    ʃ",
			18: "ʃ    ʃ    ʃ    ʃ    ʃ    ʃ",
			26: "ʃ    ʃ    ʃ    ʃ    ʃ    ʃ",
			34: "ʃ    ʃ    ʃ    ʃ    ʃ    ʃ"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := LintSpread(c.frame, layoutFrame(t, c.w, c.h, c.rows))
			var where []string
			for _, f := range got {
				if f.Sev != "P2" || f.Rule != "F-SPREAD" {
					t.Errorf("finding %+v, want a P2 F-SPREAD", f)
				}
				where = append(where, f.Where)
			}
			if strings.Join(where, " ") != strings.Join(c.want, " ") {
				t.Errorf("findings at %v, want %v: %+v", where, c.want, got)
			}
		})
	}
}

// testdata/layout/before holds a real restyle's frames (a capture of the restyle-setup
// fixture as an agent shipped it): at 80x24 its key footer wraps to a second row, at
// 120x40 its five steps sit 5–6 blank rows apart down the left, and its connection
// form's button is stranded at the bottom at both sizes. testdata/layout/reference is
// that fixture's reference (lanternfish): a packed step column and a one-row footer.
func TestLayoutRulesOnARealRestyle(t *testing.T) {
	want := map[string]string{
		"access-80x24.ansi":      "F-FOOTWRAP r23c1",
		"connection-80x24.ansi":  "F-FOOTWRAP r23c1, F-SPREAD r20c2",
		"profile-80x24.ansi":     "F-FOOTWRAP r23c1",
		"access-120x40.ansi":     "F-SPREAD r2c1",
		"connection-120x40.ansi": "F-SPREAD r2c1, F-SPREAD r37c34",
		"profile-120x40.ansi":    "F-SPREAD r2c1",
	}
	got := map[string][]string{}
	fs, _, code := run(t, "testdata/layout/before")
	for _, f := range fs {
		if f.Rule == "F-FOOTWRAP" || f.Rule == "F-SPREAD" {
			got[f.File] = append(got[f.File], f.Rule+" "+f.Where)
		}
	}
	for file, w := range want {
		if g := strings.Join(got[file], ", "); g != w {
			t.Errorf("%s: %q, want %q", file, g, w)
		}
	}
	if code != 1 {
		t.Errorf("exit %d, want 1 (F-FOOTWRAP blocks)", code)
	}
	refs, _ := filepath.Glob("testdata/layout/reference/*.ansi")
	if len(refs) == 0 {
		t.Fatal("no reference frames")
	}
	fs, _, _ = run(t, "testdata/layout/reference")
	for _, f := range fs {
		if f.Rule == "F-FOOTWRAP" || f.Rule == "F-SPREAD" {
			t.Errorf("reference %s: %+v", f.File, f)
		}
	}
}

// The synthetic and kit frames the Python-era goldens hold trip neither rule, so the
// goldens stand as they are.
func TestLayoutRulesLeaveTheGoldensAlone(t *testing.T) {
	for _, dir := range []string{"testdata/synthetic", "testdata/kit", "testdata/pages", "testdata/overlays"} {
		err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(p) != ".ansi" {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			f := parseText(string(data))
			for _, x := range append(LintFootWrap(filepath.Base(p), f), LintSpread(filepath.Base(p), f)...) {
				t.Errorf("%s: %+v", p, x)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
