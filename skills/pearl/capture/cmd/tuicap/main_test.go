package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"

	"tuicap/internal/redact"
	"tuicap/internal/session"
)

// The -h text is the package doc comment; make sure it survives the extraction whole.
func TestUsageIsTheDocComment(t *testing.T) {
	u := usage()
	for _, want := range []string{
		"Command tuicap captures any terminal program's screens",
		"  tuicap -tape setup.tape -out capture -- ./bin/app init",
		"  Snapshot setup-target   write frames/setup-target-<W>x<H>.ansi",
		"  -level L           full (default) | privacy",
		"Exit 0 when every Snapshot was written",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("usage lacks %q", want)
		}
	}
}

var fixtureBin string

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		os.Exit(0) // tuicap is Unix only
	}
	dir, err := os.MkdirTemp("", "tuicap-fixture")
	if err != nil {
		panic(err)
	}
	fixtureBin = filepath.Join(dir, "fixture")
	gobin := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(gobin); err != nil {
		gobin = "go"
	}
	cmd := exec.Command(gobin, "build", "-o", fixtureBin, "./testdata/fixture")
	cmd.Dir = "../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

const fixtureTape = `# the fixture: its screen, a move, a filter typed, a resize, a clean quit
WaitFor Larkspur Ledger
WaitStable
Snapshot main
Key down
WaitFor ▌ Quarry
WaitStable
Snapshot moved
Key /
Type s3cret Plans
WaitFor s3cret Plans
Key enter enter
WaitFor Opened
WaitStable
Snapshot filtered
Resize 100x30
WaitStable
Snapshot wide
Key q
WaitExit
`

// run runs tuicap with a tape and returns its exit status and stderr.
func run(t *testing.T, tapeText string, args ...string) (int, string) {
	t.Helper()
	tp := filepath.Join(t.TempDir(), "t.tape")
	if err := os.WriteFile(tp, []byte(tapeText), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "") // no cellshot: the bundle must not need it
	var stderr bytes.Buffer
	code := Main(append([]string{"-tape", tp}, args...), nil, &stderr)
	return code, stderr.String()
}

// grid reads a frame back into cells with the same emulator.
func grid(t *testing.T, path string, w, h int) [][]uv.Cell {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.CheckSize(string(b), w, h); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	e := vt.NewEmulator(w, h)
	go func() {
		buf := make([]byte, 256)
		for {
			if _, err := e.Read(buf); err != nil {
				return
			}
		}
	}()
	defer e.InputPipe().(io.Closer).Close() // not e.Close: see session.closeReplies
	_, _ = e.WriteString(strings.ReplaceAll(strings.TrimSuffix(string(b), "\n"), "\n", "\r\n"))
	g := make([][]uv.Cell, h)
	for y := range h {
		for x := range w {
			g[y] = append(g[y], *e.CellAt(x, y))
		}
	}
	return g
}

func text(g [][]uv.Cell) string {
	var b strings.Builder
	for _, row := range g {
		for _, c := range row {
			b.WriteString(c.Content)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

var frames = []struct {
	name string
	w, h int
}{{"main", 80, 24}, {"moved", 80, 24}, {"filtered", 80, 24}, {"wide", 100, 30}}

func TestCaptureFullAndPrivacy(t *testing.T) {
	full, priv := filepath.Join(t.TempDir(), "full"), filepath.Join(t.TempDir(), "priv")
	if code, errs := run(t, fixtureTape, "-raw", "-out", full, "--", fixtureBin); code != 0 {
		t.Fatalf("full: exit %d\n%s", code, errs)
	}
	if code, errs := run(t, fixtureTape, "-level", "privacy", "-out", priv, "--", fixtureBin); code != 0 {
		t.Fatalf("privacy: exit %d\n%s", code, errs)
	}

	// the full frames are the fixture's screens, at the size of each snapshot
	want := map[string][]string{
		"main":     {"Larkspur Ledger · accounts 12", "▌ Larkspur Ledger — 42 entries", "↑/↓ move · enter open"},
		"moved":    {"▌ Quarry Notes — 42 entries"},
		"filtered": {"filter: s3cret Plans", "Opened Quarry Notes"},
		"wide":     {"▌ Quarry Notes — 42 entries"},
	}
	for _, f := range frames {
		g := grid(t, filepath.Join(full, "frames", f.name+"-"+size{f.w, f.h}.String()+".ansi"), f.w, f.h)
		for _, s := range want[f.name] {
			if !strings.Contains(text(g), s) {
				t.Errorf("%s: frame lacks %q:\n%s", f.name, s, text(g))
			}
		}
		if bg := g[0][f.w-1].Style.Bg; bg == nil {
			t.Errorf("%s: the title bar's background is lost at its last cell", f.name)
		}
	}

	allowed := redact.New()
	for _, f := range frames {
		name := filepath.Join("frames", f.name+"-"+size{f.w, f.h}.String()+".ansi")
		a, b := grid(t, filepath.Join(full, name), f.w, f.h), grid(t, filepath.Join(priv, name), f.w, f.h)
		for y := range f.h {
			for x := range f.w {
				ca, cb := a[y][x], b[y][x]
				if ca.Width != cb.Width || !ca.Style.Equal(&cb.Style) {
					t.Fatalf("%s %d,%d: layout differs: %q %+v vs %q %+v", f.name, x, y, ca.Content, ca.Style, cb.Content, cb.Style)
				}
				r := []rune(ca.Content + " ")[0]
				switch {
				case unicode.IsLetter(r) || unicode.IsDigit(r):
					if cb.Content != ca.Content && !strings.Contains("xX0", cb.Content) {
						t.Errorf("%s %d,%d: %q became %q", f.name, x, y, ca.Content, cb.Content)
					}
				case cb.Content != ca.Content && cb.Content != "x": // punctuation changes only inside a URL, email or path
					t.Errorf("%s %d,%d: glyph %q became %q", f.name, x, y, ca.Content, cb.Content)
				}
			}
		}
		pt := text(b)
		// no word survives unless allowed (or the letter after ctrl+)
		for _, w := range strings.FieldsFunc(pt, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			if strings.Trim(w, "xX0") == "" || allowed.Allowed(w) || w == "c" && strings.Contains(pt, "ctrl+c") {
				continue
			}
			t.Errorf("%s: %q survived redaction", f.name, w)
		}
		for _, secret := range []string{"Larkspur", "Quarry", "s3cret", "larkspur.example", "/var/log", "ops@", "42", "12"} {
			if strings.Contains(pt, secret) {
				t.Errorf("%s: %q survived redaction", f.name, secret)
			}
		}
		if f.name == "main" {
			for _, kept := range []string{"enter", "quit", "help", "ctrl+c", "q quit", "↑/↓", "?"} {
				if !strings.Contains(pt, kept) {
					t.Errorf("main: allowlisted %q was redacted:\n%s", kept, pt)
				}
			}
		}
	}

	// flow.json: the run as data
	var fl flowFile
	readJSON(t, filepath.Join(full, "flow.json"), &fl)
	if len(fl.Runs) != 1 || len(fl.Runs[0].Snapshots) != 4 {
		t.Fatalf("flow.json: want 1 run of 4 snapshots, got %+v", fl.Runs)
	}
	snaps := fl.Runs[0].Snapshots
	if got := snaps[2].Input; len(got) != 4 || got[1].Type != "s3cret Plans" || got[0].Key != "/" {
		t.Errorf("flow.json: filtered's input = %+v", got)
	}
	if snaps[3].Size != "100x30" || snaps[3].Input[0].Resize != "100x30" || !snaps[3].Stable {
		t.Errorf("flow.json: wide = %+v", snaps[3])
	}
	if x := fl.Runs[0].Exit; !x.Exited || x.Code != 0 || x.Killed {
		t.Errorf("flow.json: exit = %+v, want exited 0", x)
	}
	var fp flowFile
	readJSON(t, filepath.Join(priv, "flow.json"), &fp)
	if got := fp.Runs[0].Snapshots[2].Input[1].Type; got != "x0xxxx Xxxxx" {
		t.Errorf("privacy flow.json: typed text = %q, want x0xxxx Xxxxx", got)
	}

	// sequences.json: the fixture's alt screen and cursor, its title, link and unknown sequence
	for _, dir := range []string{full, priv} {
		var sf seqFile
		readJSON(t, filepath.Join(dir, "sequences.json"), &sf)
		r := sf.Runs[0]
		first := r.Snapshots[0].Events
		if first["alt_screen_on"] != 1 || first["cursor_hide"] != 1 || first["title"] != 1 || first["hyperlink"] != 1 || first["unknown"] != 1 {
			t.Errorf("%s: first snapshot's events = %v", dir, first)
		}
		if r.After["alt_screen_off"] != 1 || r.After["cursor_show"] != 1 {
			t.Errorf("%s: after the last snapshot = %v, want the alt screen left and the cursor shown", dir, r.After)
		}
		if tot := r.Total["alt_screen_on"]; tot == nil || tot.First != "main" || tot.Count != 1 {
			t.Errorf("%s: total alt_screen_on = %+v", dir, tot)
		}
		if !r.Snapshots[1].Cursor.Hidden || r.Cursor.Hidden {
			t.Errorf("%s: cursor hidden while running and shown at the end: %+v, %+v", dir, r.Snapshots[1].Cursor, r.Cursor)
		}
		if r.End.Clean == nil || !*r.End.Clean || len(r.End.Left) != 0 {
			t.Errorf("%s: end = %+v, want clean", dir, r.End)
		}
		if r.Title == nil || r.Title.Length != len("Larkspur ledger") {
			t.Errorf("%s: title = %+v", dir, r.Title)
		}
		if len(r.Unknown) != 1 || r.Unknown[0].Kind != "CSI ?x" {
			t.Errorf("%s: unknown = %+v", dir, r.Unknown)
		}
	}

	// privacy writes no text it can't redact: titles, unknown bytes, the raw log
	b, _ := os.ReadFile(filepath.Join(priv, "sequences.json"))
	for _, leak := range []string{"Larkspur", "\\u001b[?7"} {
		if bytes.Contains(b, []byte(leak)) {
			t.Errorf("privacy sequences.json holds %q", leak)
		}
	}
	b, _ = os.ReadFile(filepath.Join(full, "sequences.json"))
	if !bytes.Contains(b, []byte("Larkspur ledger")) || !bytes.Contains(b, []byte(`\u001b[?7;1x`)) {
		t.Error("full sequences.json lacks the title's text or the unknown sequence's bytes")
	}
	if _, err := os.Stat(filepath.Join(full, "raw-80x24.log")); err != nil {
		t.Error("full -raw wrote no raw log")
	}
	ents, _ := os.ReadDir(priv)
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "raw-") {
			t.Error("privacy bundle holds a raw log")
		}
	}
	if b, _ := os.ReadFile(filepath.Join(priv, "README.txt")); !bytes.Contains(b, []byte("privacy level")) {
		t.Error("README.txt doesn't say the level")
	}
}

func TestSizesRunTheTapeOncePerSize(t *testing.T) {
	out := filepath.Join(t.TempDir(), "b")
	tp := "WaitFor Larkspur\nWaitStable\nSnapshot main\nKey q\nWaitExit\n"
	if code, errs := run(t, tp, "-sizes", "60x12,90x20", "-profile", "ansi", "-out", out, "--", fixtureBin); code != 0 {
		t.Fatalf("exit %d\n%s", code, errs)
	}
	for _, s := range []size{{60, 12}, {90, 20}} {
		grid(t, filepath.Join(out, "frames", "main-"+s.String()+".ansi"), s.w, s.h)
	}
	// the same bundle again: tuicap replaces its own files
	if code, errs := run(t, tp, "-out", out, "--", fixtureBin); code != 0 {
		t.Fatalf("rerun: exit %d\n%s", code, errs)
	}
	if _, err := os.Stat(filepath.Join(out, "frames", "main-60x12.ansi")); err == nil {
		t.Error("a rerun kept a frame from the earlier bundle")
	}
}

func TestFailures(t *testing.T) {
	for _, tc := range []struct {
		name, tape string
		args       []string
		code       int
		msg        string
	}{
		{"waitfor times out", "WaitFor nowhere to be seen\n", []string{"-wait", "300ms"}, 1, `"nowhere to be seen" was not on screen after 300ms`},
		{"early exit", "WaitFor Larkspur\nKey q\nWait 300ms\nKey down\n", nil, 1, "the program exited with status 0 before the tape ended"},
		{"waitexit times out", "WaitFor Larkspur\nWaitExit 200ms\n", nil, 1, "still running after 200ms"},
		{"bad tape", "Snapshot\n", nil, 2, "line 1: Snapshot needs a state name"},
		{"bad key", "Key ctrl+1\n", nil, 2, "ctrl+1 has no legacy encoding"},
		{"no program", "Wait 1ms\n", []string{"-out", "x"}, 2, "no program"},
		{"raw at privacy", "Wait 1ms\n", []string{"-level", "privacy", "-raw"}, 2, "not at -level privacy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args
			if tc.name != "no program" {
				args = append(append([]string{"-out", filepath.Join(t.TempDir(), "o")}, args...), "--", fixtureBin)
			}
			code, errs := run(t, tc.tape, args...)
			if code != tc.code || !strings.Contains(errs, tc.msg) {
				t.Errorf("exit %d, stderr:\n%s\nwant exit %d and %q", code, errs, tc.code, tc.msg)
			}
		})
	}
}

func TestOutMustBeEmptyOrABundle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, errs := run(t, "Wait 1ms\n", "-out", dir, "--", fixtureBin)
	if code != 2 || !strings.Contains(errs, "notes.txt, which tuicap didn't write") {
		t.Errorf("exit %d: %s", code, errs)
	}
}

func TestChildEnv(t *testing.T) {
	env := childEnv([]string{"HOME=/h", "TERM=screen", "NO_COLOR=1", "TMUX=x"}, profileEnv["ansi"], []string{"ACCESSIBLE=1", "TERM=linux"})
	got := strings.Join(env, " ")
	if got != "HOME=/h TERM=linux ACCESSIBLE=1" {
		t.Errorf("childEnv = %q", got)
	}
}
