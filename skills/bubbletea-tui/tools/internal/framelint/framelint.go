// Package framelint is `tuilint frames`: deterministic slop checks on rendered
// truecolor ANSI frames, a port of tuilint.py's frame rules with the same IDs,
// severities, thresholds, output and JSON.
//
//	F-WIDTH     [P0] a row is not exactly W cells / frame is not H rows
//	F-EMOJI     [P1] emoji glyphs
//	F-DEFAULT   [P1] colours from library examples and defaults (#7D56F4, 205, 62/63/170/212, ...,
//	                 and Huh's #F780E2 fuchsia and #7571F9 / #5A56E0 indigo)
//	F-OFFPAL    [P1] (with --theme) a colour not in, or blended from, the theme's roles.
//	                 A frame with only 256-colour SGR (-profile ansi256) is judged against the
//	                 xterm colours the roles downsample to; there a role that lands on a
//	                 default index (204) is the role, and its hex is judged in truecolour.
//	                 A frame with only 16-colour SGR (-profile ansi) is not judged: its 16
//	                 slots are the terminal's palette, and every role lands on one
//	F-CONTRAST  [P1] text whose fg/bg contrast is < 2.3:1. Box drawing and blocks (U+2500–259F),
//	                 braille, legacy blocks and rule runs (────, ----, ====, ····, and the
//	                 "--" a titled rule leads with) are decoration; status
//	                 marks (● ◆ ▶ ○ ◐ •) are text. On a frame with a dialog, only the dialog
//	                 and the footer are judged: a bordered box, an ASCII +--+ box, or a
//	                 "dialog": "rules" panel (two rule rows of one span around a filled panel),
//	                 found by its shape whatever the frame's name (a confirm's sending state
//	                 too; on a frame not named as an overlay, a ruled or ASCII one counts only
//	                 with a scrim: a blank row above and below it, and the page's other rows
//	                 dimmed under 2.3:1 or hidden, so a sectioned page is judged whole). On an
//	                 overlay frame (named dialog, palette, confirm, modal, overlay,
//	                 popup, help or keys) with none found, a P2 note instead ("dialog": "none"
//	                 has no edge to find).
//	                 In a 16-colour frame: text in the same slot as its background
//	F-BOXES     [P1] more than one bordered box (nested or side by side noted)
//	F-RAINBOW   [P2] more than 7 distinct foreground hues (not judged on a 256-colour frame, where
//	                 downsampling scatters a gradient's steps over hues the design never chose)
//	F-LEVELS    [P2] fewer than 3 neutral text levels on a dense frame (not judged on a
//	                 frame with no colour, NO_COLOR: its levels are weight and glyphs)
//	F-BOLD      [P2] more than 30% of text cells bold (not on a splash*, intro* or wordmark* frame,
//	                 nor under 200 text cells: a few bold labels there are a ratio, not a wall)
//	F-EDGE      [P2] content touching column 0 or the last column on most rows
//	F-EMPTY     [P2] frame is more than 98.5% blank (not on a splash*, intro* or wordmark* frame:
//	                 a wordmark on an empty page is what a splash is)
//	F-TOFU      [P2] private-use (Nerd Font) or replacement glyphs
//	F-TAB       [P1] a raw tab in a frame: lipgloss measures it as 0 cells, a terminal as the
//	                 run to the next 8-column stop, so the row tears; expand tabs before fitting.
//	                 The kit's canvas expands tabs (ui.ExpandTabs), so this fires only where a
//	                 view bypasses it: file or log content written straight into View(), the
//	                 place a tab arrives from data; it stays P1 because the row it tears is real
//	F-TRUNC     [P2] more than 3 rows ending in an ellipsis (…, the theme's glyphs.ellipsis, or
//	                 ... with --ascii) at one column: a column budgeted narrower than its content
//	                 (not a table's last column headed path, file, command or url, its columns
//	                 two blanks or one apart, nor an ellipsis an overlay's cut put beside its
//	                 dialog, nor page rows on an overlay frame whose column header the dialog
//	                 hides: the page's own frame shows it and is judged there). With --spec
//	                 TUI.md, a column that cuts by decision isn't either: a `cuts:` line in
//	                 § Decisions names its header (`- cuts: command, title — no longest`) or a
//	                 region of a budget line at the frame's size (`80×24: … list 1,10 78×13`)
//	                 that holds every cut row (`cuts: list`)
//	F-ASCII     [P1] (with --ascii) a cell outside ASCII in a frame rendered for ASCII; with
//	                 --unicode, also a "?" where the unicode frame of the same state draws
//	                 something else (a rune the sweep had no twin for)
package framelint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const usage = `usage: tuilint frames <dir|files...> [--cols W --rows H] [--theme theme.json] [--bg #rrggbb] [--ascii [--unicode dir|file]] [--spec TUI.md] [--max-warnings N] [--json out.json]

Lints rendered truecolor ANSI frames (the output of your -state X -w W -h H flag).
A directory means its *.ansi files. A name ending -<W>x<H>.ansi (what -out writes) sets
that frame's size, and --cols/--rows each override their own dimension; with neither,
W is the widest row and H is free.
A finding prints as [SEV] ID  file:where  message.
P0/P1 findings fail the run (exit 1); P2 are warnings to fix or justify in one line.
--max-warnings N also fails the run (exit 1) when more than N warnings remain, so a
CI baseline catches a new warning class. Exit 2 on usage errors or an unreadable input.
--bg is the terminal default background for cells that set none (default #16161a).
--ascii says the frames were rendered for ASCII (-ascii: TERM=linux, --ascii), where
every cell must be ASCII: a glyph that leaks through is tofu on a console font.
--unicode names the unicode render of the same states (a directory, matched by file
name, or one frame): a "?" in the ASCII frame where the unicode frame draws something
else is a rune the ASCII sweep had no twin for, which reads as a question.
--spec names the TUI.md whose § Decisions "cuts:" lines say which columns cut on purpose
(F-TRUNC).

  F-WIDTH     [P0] a row is not exactly W cells / frame is not H rows
  F-EMOJI     [P1] emoji glyphs
  F-DEFAULT   [P1] library example and default colours: #7D56F4, 205, 62/63/170/212, ...,
                   and Huh's #F780E2 fuchsia (Charm theme, spinner), #7571F9 / #5A56E0 indigo
  F-OFFPAL    [P1] (with --theme) a colour not in, or blended from, the theme's roles
                   (a 16-colour frame, -profile ansi, is not judged: the terminal owns those slots)
  F-CONTRAST  [P1] text whose fg/bg contrast is < 2.3:1; box drawing, blocks, braille and
                   rule runs (their ASCII twins too: ----, ====, and a titled rule's "--" lead)
                   are decoration, status marks (● ◆ ▶ ○ •) are text. Where a frame has a
                   dialog, found by its shape whatever the frame's name (a bordered box, an
                   ASCII +--+ box or a "dialog": "rules" panel: two rule rows of one span
                   around a filled panel; off an overlay-named frame, only over a scrim that
                   dims or hides the rest of the page), only the dialog is judged; an overlay frame (named
                   dialog|palette|confirm|modal|overlay|popup|help|keys) with none found is a
                   P2 note. In a 16-colour frame, text in the same slot as its background
  F-BOXES     [P1] more than one bordered box
  F-RAINBOW   [P2] more than 7 distinct foreground hues (not on a 256-colour frame)
  F-LEVELS    [P2] fewer than 3 neutral text levels on a dense frame (not on a no-colour frame)
  F-BOLD      [P2] more than 30% of text cells bold (not on a splash*/intro*/wordmark* frame,
                   nor under 200 text cells)
  F-EDGE      [P2] content touching column 0 or the last column on most rows
  F-EMPTY     [P2] frame is more than 98.5% blank (not on a splash*/intro*/wordmark* frame)
  F-TOFU      [P2] private-use (Nerd Font) or replacement glyphs
  F-TAB       [P1] a raw tab: 0 cells to lipgloss, up to 8 to a terminal; expand it before fitting
                   (the kit's canvas does; this catches a view that bypasses it with file or
                   log content)
  F-TRUNC     [P2] more than 3 rows ending in an ellipsis at one column: widen it from the
                   longest fixture, or, for paths, URLs, code, name its header or budget region
                   on a "cuts:" line in TUI.md § Decisions and pass --spec TUI.md
  F-ASCII     [P1] (with --ascii) a cell outside ASCII: give the glyph an ASCII twin;
                   with --unicode, a "?" where the unicode frame of the state has none
`

var sevOrder = map[string]int{"P0": 0, "P1": 1, "P2": 2}

// Main runs the mode with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	opts := map[string]string{}
	var pos []string
	ascii := false
	maxWarnings := -1
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-h" || a == "--help" {
			fmt.Fprint(stdout, usage)
			return 0
		}
		if a == "--ascii" {
			ascii = true
			continue
		}
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		k := a[2:]
		switch k {
		case "cols", "rows", "theme", "json", "bg", "max-warnings", "unicode", "spec":
		default:
			fmt.Fprintf(stderr, "tuilint frames: unknown flag %s\n%s", a, usage)
			return 2
		}
		if i+1 >= len(args) {
			fmt.Fprintf(stderr, "tuilint frames: %s needs a value\n", a)
			return 2
		}
		opts[k] = args[i+1]
		i++
	}
	num := func(k string) (int, bool) {
		v, ok := opts[k]
		if !ok {
			return 0, true
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			fmt.Fprintf(stderr, "tuilint frames: --%s %q is not a number\n", k, v)
			return 0, false
		}
		return n, true
	}
	cols, ok1 := num("cols")
	rows, ok2 := num("rows")
	if !ok1 || !ok2 {
		return 2
	}
	if v, ok := opts["max-warnings"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			fmt.Fprintf(stderr, "tuilint frames: --max-warnings %q is not a count\n", v)
			return 2
		}
		maxWarnings = n
	}
	bg := RGB{0x16, 0x16, 0x1a}
	if v, ok := opts["bg"]; ok {
		if !hexRe.MatchString("#" + strings.TrimLeft(v, "#")) {
			fmt.Fprintf(stderr, "tuilint frames: --bg %q is not #rrggbb\n", v)
			return 2
		}
		bg = hexc(v)
	}
	var pal *Palette
	if p, ok := opts["theme"]; ok {
		var err error
		if pal, err = LoadPalette(p); err != nil {
			fmt.Fprintf(stderr, "tuilint frames: theme: %v\n", err)
			return 2
		}
	}
	var spec *Spec
	if p, ok := opts["spec"]; ok {
		var err error
		if spec, err = LoadSpec(p); err != nil {
			fmt.Fprintf(stderr, "tuilint frames: --spec: %v\n", err)
			return 2
		}
	}
	uni, hasUni := opts["unicode"]
	if hasUni && !ascii {
		fmt.Fprintf(stderr, "tuilint frames: --unicode compares an ASCII render; it needs --ascii\n")
		return 2
	}
	uniDir := false
	if hasUni {
		st, err := os.Stat(uni)
		if err != nil {
			fmt.Fprintf(stderr, "tuilint frames: --unicode: %v\n", err)
			return 2
		}
		uniDir = st.IsDir()
	}
	paths, err := expand(pos)
	if err != nil {
		fmt.Fprintf(stderr, "tuilint frames: %v\n", err)
		return 2
	}
	findings := []Finding{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "tuilint frames: %v\n", err)
			return 2
		}
		w, h := cols, rows
		if fw, fh, ok := sizeFromName(p); ok {
			if w == 0 {
				w = fw
			}
			if h == 0 {
				h = fh
			}
		}
		fs := lintText(p, string(data), w, h, pal, bg, ascii, spec)
		if hasUni {
			up := uni
			if uniDir {
				up = filepath.Join(uni, filepath.Base(p))
			}
			// a state the unicode render lacks has nothing to compare against
			if ud, err := os.ReadFile(up); err == nil {
				fs = append(fs, LintASCIIQuestion(filepath.Base(p), parseText(string(data)), parseText(string(ud)))...)
			}
		}
		findings = append(findings, fs...)
	}
	fmt.Fprintf(stdout, "linted %d frame(s)\n", len(paths))
	return Report(findings, opts["json"], maxWarnings, stdout, stderr)
}

var sizeSuffix = regexp.MustCompile(`-(\d+)x(\d+)\.ansi$`)

// sizeFromName reads W×H from a "-<W>x<H>.ansi" suffix, the name a harness's -out
// writes, so a directory of mixed sizes lints each frame at its own size without
// flags. Deliberate deviation: tuilint.py took W from the widest row and never
// checked H unless --cols/--rows were given; those flags still win.
func sizeFromName(path string) (int, int, bool) {
	m := sizeSuffix.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return 0, 0, false
	}
	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	return w, h, w > 0 && h > 0
}

// expand turns directories into their *.ansi files (sorted, dotfiles skipped,
// as Python's glob does) and keeps other paths as given.
func expand(pos []string) ([]string, error) {
	var paths []string
	for _, p := range pos {
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			paths = append(paths, p)
			continue
		}
		m, err := filepath.Glob(filepath.Join(escapeGlob(p), "*.ansi"))
		if err != nil {
			return nil, err
		}
		for _, f := range m {
			if !strings.HasPrefix(filepath.Base(f), ".") {
				paths = append(paths, f)
			}
		}
	}
	return paths, nil
}

func escapeGlob(p string) string {
	r := strings.NewReplacer(`*`, `\*`, `?`, `\?`, `[`, `\[`, `\`, `\\`)
	return r.Replace(p)
}

// Report sorts findings by severity, rule and file, prints them and the
// verdict, writes the JSON when asked, and returns the exit status: 1 on a P0 or
// P1, or on more than maxWarnings warnings (-1 for no limit).
func Report(findings []Finding, jsonPath string, maxWarnings int, stdout, stderr io.Writer) int {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if sevOrder[a.Sev] != sevOrder[b.Sev] {
			return sevOrder[a.Sev] < sevOrder[b.Sev]
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.File < b.File
	})
	fails := 0
	for _, f := range findings {
		fmt.Fprintf(stdout, "[%s] %-11s %s:%s  %s\n", f.Sev, f.Rule, f.File, f.Where, f.Msg)
		if f.Sev == "P0" || f.Sev == "P1" {
			fails++
		}
	}
	warnings := len(findings) - fails
	verdict, over := "OK", ""
	if fails > 0 {
		verdict = "FAIL"
	}
	if maxWarnings >= 0 && warnings > maxWarnings {
		verdict, over = "FAIL", fmt.Sprintf(" (more than --max-warnings %d)", maxWarnings)
	}
	fmt.Fprintf(stdout, "%s  %d blocking, %d warnings%s\n", verdict, fails, warnings, over)
	if jsonPath != "" {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", " ")
		if err := enc.Encode(findings); err != nil {
			fmt.Fprintf(stderr, "tuilint frames: %v\n", err)
			return 2
		}
		if err := os.WriteFile(jsonPath, b.Bytes(), 0o644); err != nil {
			fmt.Fprintf(stderr, "tuilint frames: %v\n", err)
			return 2
		}
	}
	if fails > 0 || over != "" {
		return 1
	}
	return 0
}
