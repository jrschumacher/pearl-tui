package hook

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"tuitools/internal/codelint"
)

// fixture copies testdata/<name> into a temporary directory, since the hook skips files
// under testdata, and returns the copy's path.
func fixture(t *testing.T, name string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), name)
	src := filepath.Join("testdata", name)
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// post is the PostToolUse input Claude Code sends after an edit of file.
func post(tool, cwd, file string) string {
	in := map[string]any{
		"session_id":      "test",
		"hook_event_name": "PostToolUse",
		"cwd":             cwd,
		"tool_name":       tool,
		"tool_input":      map[string]any{"file_path": file, "old_string": "a", "new_string": "b"},
		"tool_response":   map[string]any{"filePath": file},
		"tool_use_id":     "toolu_test",
	}
	b, _ := json.Marshal(in)
	return string(b)
}

func runHook(t *testing.T, input string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Main(strings.NewReader(input), &stdout, &stderr)
	if stdout.Len() > 0 {
		t.Errorf("stdout is for the hook's own failures only; got %q", stdout.String())
	}
	return code, stderr.String()
}

func TestHook(t *testing.T) {
	tui := fixture(t, "tui")
	plain := fixture(t, "plain")
	bare := fixture(t, "bare")
	exe, _ := os.Executable()
	if err := os.WriteFile(filepath.Join(tui, "form", "label.go"), []byte("package form\n\n// Label names the field.\nconst Label = \"Name\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, input string
		code        int
		want        []string // substrings of stderr; none means stderr must be empty
	}{
		{"unthemed huh form", post("Edit", tui, filepath.Join(tui, "form", "form.go")), 2,
			[]string{"tuilint code: 1 blocking problem in form/form.go after this edit", "C-HUHTHEME form/form.go:9  huh form without WithTheme", "`" + exe + " code -h`"}},
		// form.go's finding was there before this edit, and isn't its doing
		{"clean file in the same package", post("Write", tui, filepath.Join(tui, "form", "label.go")), 0, nil},
		{"relative path, MultiEdit", post("MultiEdit", tui, "form/form.go"), 2, []string{"C-HUHTHEME form/form.go:9"}},
		{"cwd outside the module", post("Edit", t.TempDir(), filepath.Join(tui, "form", "form.go")), 2,
			[]string{filepath.Join(tui, "form", "form.go") + ":9"}},
		{"clean package, findings elsewhere in the module", post("Write", tui, filepath.Join(tui, "clean", "clean.go")), 0, nil},
		{"not Go", post("Write", tui, filepath.Join(tui, "notes.md")), 0, nil},
		{"Go module that isn't a TUI", post("Edit", plain, filepath.Join(plain, "main.go")), 0, nil},
		{"Bubble Tea app that never opted in to the skill", post("Edit", bare, filepath.Join(bare, "main.go")), 0, nil},
		{"no file path", `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`, 0, nil},
		{"unreadable input", `{"tool_input":`, 0, []string{"tuilint hook: unreadable hook input"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, stderr := runHook(t, c.input)
			if code != c.code {
				t.Errorf("exit %d, want %d; stderr:\n%s", code, c.code, stderr)
			}
			if c.want == nil && stderr != "" {
				t.Errorf("want silence, got:\n%s", stderr)
			}
			for _, w := range c.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr lacks %q:\n%s", w, stderr)
				}
			}
		})
	}

	// The non-TUI module and the app without TUI.md are silent because of the gates, not
	// because they are clean.
	fs, err := codelint.Lint(plain, "", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(fs, "C-EMOJI") {
		t.Errorf("plain fixture should fail tuilint code on its own (C-EMOJI); got %v", fs)
	}
	if err := os.WriteFile(filepath.Join(bare, Marker), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr := runHook(t, post("Edit", bare, filepath.Join(bare, "main.go")))
	for _, w := range []string{"C-HEX main.go:", "C-EMOJI main.go:", "C-SIZE main.go:"} {
		if code != 2 || !strings.Contains(stderr, w) {
			t.Errorf("bare app with a TUI.md: exit %d, want 2 with %q; stderr:\n%s", code, w, stderr)
		}
	}
}

// The kit itself opts in through its theme package at theme/bubbletea: no TUI.md, the
// framework's layout, and a hex in ui/bubbletea is the edit's finding.
func TestHookKitLayout(t *testing.T) {
	kit := filepath.Join(t.TempDir(), "kit")
	src := filepath.Join("..", "..", "..", "kit")
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(kit, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(kit, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(kit, Marker)); err == nil {
		t.Fatal("the kit holds a TUI.md: this test is about the theme-package opt-in")
	}
	leak := filepath.Join(kit, "ui", "bubbletea", "leak.go")
	if err := os.WriteFile(leak, []byte("package ui\n\nimport \"charm.land/lipgloss/v2\"\n\nvar hexLeak = lipgloss.Color(\"#ff0000\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr := runHook(t, post("Write", kit, leak))
	if code != 2 || !strings.Contains(stderr, "C-HEX ui/bubbletea/leak.go:5") {
		t.Errorf("hex in the kit's ui/bubbletea: exit %d, want 2 with C-HEX; stderr:\n%s", code, stderr)
	}
}

// A panic in the hook is the hook's failure, not the edit's: a note for the user on
// stdout, exit 0, nothing for the model.
func TestHookPanic(t *testing.T) {
	tui := fixture(t, "tui")
	lint = func(string, string, io.Writer, codelint.Load) ([]codelint.Finding, error) { panic("boom\nstack") }
	t.Cleanup(func() { lint = codelint.LintWith })
	var stdout, stderr bytes.Buffer
	code := Main(strings.NewReader(post("Edit", tui, filepath.Join(tui, "form", "form.go"))), &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	var out struct{ SystemMessage string }
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil || !strings.Contains(out.SystemMessage, "crashed (boom)") {
		t.Errorf("stdout = %q (%v), want a systemMessage naming the panic", stdout.String(), err)
	}
	if strings.HasPrefix(stderr.String(), Header) {
		t.Errorf("a crash must not look like findings: %q", stderr.String())
	}
}

// GOFLAGS=-mod=mod would let the go command write go.sum from the module cache, even
// with GOPROXY=off. The hook loads read-only, so the user's module is left as it was.
func TestHookReadOnly(t *testing.T) {
	tui := fixture(t, "tui")
	if err := os.Remove(filepath.Join(tui, "go.sum")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	before := snapshot(t, tui)
	code, stderr := runHook(t, post("Edit", tui, filepath.Join(tui, "form", "form.go")))
	if code != 2 || !strings.Contains(stderr, "C-HUHTHEME form/form.go:9") {
		t.Errorf("exit %d, want 2 with C-HUHTHEME (syntax-only); stderr:\n%s", code, stderr)
	}
	if after := snapshot(t, tui); !maps.Equal(before, after) {
		t.Errorf("the hook changed the module:\nbefore %v\nafter  %v", keys(before), keys(after))
	}
}

// snapshot maps every file under dir to its contents.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		m[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func keys(m map[string]string) []string {
	return slices.Sorted(maps.Keys(m))
}

func hasRule(fs []codelint.Finding, rule string) bool {
	for _, f := range fs {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestTarget(t *testing.T) {
	tui := fixture(t, "tui")
	plain := fixture(t, "plain")
	write := func(p, src string) string {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// a file importing Huh in a module whose go.mod doesn't require it yet
	write(filepath.Join(plain, Marker), "")
	early := write(filepath.Join(plain, "wizard", "wizard.go"), "package wizard\n\nimport _ \"charm.land/huh/v2/spinner\"\n")
	cases := []struct {
		file string
		ok   bool
	}{
		{filepath.Join(tui, "form", "form.go"), true},
		{early, true},
		{filepath.Join(plain, "main.go"), false},
		{write(filepath.Join(tui, "form", "form_test.go"), "package form\n"), false},
		{write(filepath.Join(tui, "testdata", "x.go"), "package x\n"), false},
		{write(filepath.Join(tui, "vendor", "x", "x.go"), "package x\n"), false},
		{filepath.Join(tui, "form", "deleted.go"), false},
		{filepath.Join(t.TempDir(), "nomod.go"), false},
	}
	for _, c := range cases {
		if _, ok := Target(c.file); ok != c.ok {
			t.Errorf("Target(%s) = %v, want %v", c.file, ok, c.ok)
		}
	}
	if dir, _ := Target(filepath.Join(tui, "form", "form.go")); dir != tui {
		t.Errorf("module dir = %s, want %s", dir, tui)
	}
}

func TestRequiresTUI(t *testing.T) {
	for gomod, want := range map[string]bool{
		"module x\n\nrequire charm.land/bubbletea/v2 v2.1.0\n":             true,
		"module x\n\nrequire (\n\tcharm.land/huh/v2 v2.0.3 // indirect\n)": true,
		"module x\n\nrequire charm.land/lipgloss/v2 v2.0.6\n":              false,
		"module x\n\n// charm.land/bubbletea/v2 someday\n":                 false,
		"module charm.land/bubbletea/v3\n":                                 false,
	} {
		if got := requiresTUI(gomod); got != want {
			t.Errorf("requiresTUI(%q) = %v, want %v", gomod, got, want)
		}
	}
}

func TestOptedIn(t *testing.T) {
	mk := func(files map[string]string) string {
		root := t.TempDir()
		for p, src := range files {
			p = filepath.Join(root, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}
	kit := "package theme\n\ntype Roles struct{ Primary string }\n\nfunc Load(path string) (Roles, error) { return Roles{}, nil }\n"
	cases := []struct {
		name  string
		files map[string]string
		mod   string // the module directory, relative to the root
		want  bool
	}{
		{"TUI.md at the module root", map[string]string{"TUI.md": ""}, ".", true},
		{"TUI.md at the repository root", map[string]string{".git/HEAD": "", "TUI.md": "", "app/go.mod": ""}, "app", true},
		{"TUI.md in a worktree root (.git file)", map[string]string{".git": "gitdir: x", "TUI.md": "", "app/go.mod": ""}, "app", true},
		{"TUI.md above the repository root", map[string]string{"TUI.md": "", "repo/.git/HEAD": "", "repo/app/go.mod": ""}, "repo/app", false},
		{"TUI.md above a module outside any repository", map[string]string{"TUI.md": "", "app/go.mod": ""}, "app", false},
		{"TUI.md in a package, not the module root", map[string]string{"ui/TUI.md": ""}, ".", false},
		{"kit theme at internal/theme", map[string]string{"internal/theme/theme.go": kit}, ".", true},
		{"kit theme at theme", map[string]string{"theme/theme.go": kit}, ".", true},
		{"kit theme at theme/bubbletea (the kit's own layout)", map[string]string{"theme/bubbletea/theme.go": kit}, ".", true},
		{"kit theme at internal/theme/bubbletea", map[string]string{"internal/theme/bubbletea/theme.go": kit}, ".", true},
		{"kit theme two levels down", map[string]string{"theme/a/b/theme.go": kit}, ".", false},
		{"a theme package that isn't the kit's", map[string]string{"internal/theme/theme.go": "package theme\n\nfunc Load() {}\n"}, ".", false},
		{"nothing", map[string]string{"main.go": "package main\n"}, ".", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := OptedIn(filepath.Join(mk(c.files), c.mod)); got != c.want {
				t.Errorf("OptedIn = %v, want %v", got, c.want)
			}
		})
	}
}
