package hook

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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
	var stderr bytes.Buffer
	code := Main(strings.NewReader(input), &stderr)
	return code, stderr.String()
}

func TestHook(t *testing.T) {
	tui := fixture(t, "tui")
	plain := fixture(t, "plain")
	if err := os.WriteFile(filepath.Join(tui, "form", "label.go"), []byte("package form\n\n// Label names the field.\nconst Label = \"Name\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, input string
		code        int
		want        []string // substrings of stderr; none means stderr must be empty
	}{
		{"unthemed huh form", post("Edit", tui, filepath.Join(tui, "form", "form.go")), 2,
			[]string{"1 blocking problem in form after this edit", "C-HUHTHEME form/form.go:9  huh form without WithTheme", "tuilint code -h"}},
		{"clean file in the same package", post("Write", tui, filepath.Join(tui, "form", "label.go")), 2,
			[]string{"C-HUHTHEME form/form.go:9"}},
		{"relative path, MultiEdit", post("MultiEdit", tui, "form/form.go"), 2, []string{"C-HUHTHEME form/form.go:9"}},
		{"cwd outside the module", post("Edit", t.TempDir(), filepath.Join(tui, "form", "form.go")), 2,
			[]string{filepath.Join(tui, "form", "form.go") + ":9"}},
		{"clean package, findings elsewhere in the module", post("Write", tui, filepath.Join(tui, "clean", "clean.go")), 0, nil},
		{"not Go", post("Write", tui, filepath.Join(tui, "notes.md")), 0, nil},
		{"Go module that isn't a TUI", post("Edit", plain, filepath.Join(plain, "main.go")), 0, nil},
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

	// The non-TUI module is silent because of the gate, not because it is clean.
	fs, err := codelint.Lint(plain, "", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(fs, "C-EMOJI") {
		t.Errorf("plain fixture should fail tuilint code on its own (C-EMOJI); got %v", fs)
	}
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
