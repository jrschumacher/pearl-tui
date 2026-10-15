// Package framelint is `tuilint frames`: deterministic slop checks on rendered
// truecolor ANSI frames, a port of tuilint.py's frame rules with the same IDs,
// severities, thresholds, output and JSON.
//
//	F-WIDTH     [P0] a row is not exactly W cells / frame is not H rows
//	F-EMOJI     [P1] emoji glyphs
//	F-DEFAULT   [P1] colours from library examples and defaults (#7D56F4, 205, 62/63/170/212, ...)
//	F-OFFPAL    [P1] (with --theme) a colour not in, or blended from, the theme's roles.
//	                 A frame with only 256-colour SGR (-profile ansi256) is judged against the
//	                 xterm colours the roles downsample to; there a role that lands on a
//	                 default index (204) is the role, and its hex is judged in truecolour
//	F-CONTRAST  [P1] text whose fg/bg contrast is < 2.3:1 (P2 note on a borderless overlay).
//	                 Box drawing, blocks, braille and ASCII rule runs (----, ====) are decoration
//	F-BOXES     [P1] more than one bordered box (nested or side by side noted)
//	F-RAINBOW   [P2] more than 7 distinct foreground hues
//	F-LEVELS    [P2] fewer than 3 neutral text levels on a dense frame
//	F-BOLD      [P2] more than 30% of text cells bold
//	F-EDGE      [P2] content touching column 0 or the last column on most rows
//	F-EMPTY     [P2] frame is more than 98.5% blank
//	F-TOFU      [P2] private-use (Nerd Font) or replacement glyphs
package framelint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const usage = `usage: tuilint frames <dir|files...> [--cols W --rows H] [--theme theme.json] [--bg #rrggbb] [--json out.json]

Lints rendered truecolor ANSI frames (the output of your -state X -w W -h H flag).
A directory means its *.ansi files. A finding prints as [SEV] ID  file:where  message.
P0/P1 findings fail the run (exit 1); P2 are warnings to fix or justify in one line.
--bg is the terminal default background for cells that set none (default #16161a).
`

var sevOrder = map[string]int{"P0": 0, "P1": 1, "P2": 2}

// Main runs the mode with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	opts := map[string]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-h" || a == "--help" {
			fmt.Fprint(stdout, usage)
			return 0
		}
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		k := a[2:]
		switch k {
		case "cols", "rows", "theme", "json", "bg":
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
		findings = append(findings, lintText(p, string(data), cols, rows, pal, bg)...)
	}
	fmt.Fprintf(stdout, "linted %d frame(s)\n", len(paths))
	return Report(findings, opts["json"], stdout, stderr)
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
// verdict, writes the JSON when asked, and returns the exit status.
func Report(findings []Finding, jsonPath string, stdout, stderr io.Writer) int {
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
	verdict := "OK"
	if fails > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(stdout, "%s  %d blocking, %d warnings\n", verdict, fails, len(findings)-fails)
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
	if fails > 0 {
		return 1
	}
	return 0
}
