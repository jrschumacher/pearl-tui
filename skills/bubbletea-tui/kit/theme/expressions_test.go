package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A misspelled expression value ("rule" for dialog "rules") rendered a default silently,
// hiding the typo or a theme written for a newer kit. CheckExpressions (palette check,
// the kit's tests) names it and the values the kit knows; Load still loads the theme, so
// no app refuses to start over one, and Resolve draws the documented default.
func TestUnknownExpressionValues(t *testing.T) {
	for _, c := range []struct {
		json, want, field, fallback string
	}{
		{`{"expressions": {"dialog": "rule"}}`, `expressions.dialog "rule" is not a value the kit knows: rounded | square | double | heavy | none | rules`, "Dialog", "rounded"},
		{`{"expressions": {"wordmark": "plane"}}`, `expressions.wordmark "plane" is not a value the kit knows: gradient | solid | inverse | tracked | bracket | upper | plain`, "Wordmark", "gradient"},
		{`{"expressions": {"focus": "foreground"}}`, `expressions.focus "foreground" is not a value the kit knows: primary | fg`, "Focus", "primary"},
		{`{"expressions": {"selection": "Bar"}}`, `expressions.selection "Bar"`, "Selection", "bar"},
		{`{"expressions": {"button": "pills"}}`, `expressions.button "pills"`, "Button", "pill"},
		// these drew something other than their default before: an unknown tab style had
		// no underline row, an unknown scrim no dim, an unknown tag no fill
		{`{"expressions": {"tabs": "underlined"}}`, `expressions.tabs "underlined"`, "Tabs", "underline"},
		{`{"expressions": {"scrim": "dimmed"}}`, `expressions.scrim "dimmed"`, "Scrim", "dim"},
		{`{"expressions": {"tag": "fill"}}`, `expressions.tag "fill"`, "Tag", "filled"},
		// labelCase has no word for its default: unset keeps each component's own case
		{`{"expressions": {"labelCase": "small"}}`, `expressions.labelCase "small" is not a value the kit knows: upper | lower | as-is (it would draw as if unset)`, "LabelCase", ""},
	} {
		f, err := Load(writeTheme(t, c.json))
		if err != nil {
			t.Errorf("Load(%s) = %v: an expression must never stop an app from starting", c.json, err)
			continue
		}
		if err := CheckExpressions(f.Expressions); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("CheckExpressions(%s) = %v, want %q", c.json, err, c.want)
		}
		x := reflect.ValueOf(Resolve(f, true, false).X).FieldByName(c.field).String()
		if x != c.fallback {
			t.Errorf("%s resolves to %q, want the default %q", c.json, x, c.fallback)
		}
	}
	if err := CheckExpressions(Expressions{}); err != nil {
		t.Errorf("an empty expressions block: %v (every key is optional)", err)
	}
	if err := CheckExpressions(Default().Expressions); err != nil {
		t.Errorf("Default: %v", err)
	}
}

// Every theme beside the package and every test theme names only expression keys and
// values the kit knows: what palette check asks.
func TestThemesUseKnownExpressions(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	if len(files) == 0 {
		t.Fatal("no theme JSON in ../themes or ../../themes")
	}
	for _, path := range append(files, test...) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckExpressions(f.Expressions); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		b, _ := os.ReadFile(path)
		var raw struct {
			Expressions map[string]json.RawMessage `json:"expressions"`
		}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}
		for k := range raw.Expressions {
			if !ExpressionKey(k) {
				t.Errorf("%s: expressions.%s is not a key the kit reads", path, k)
			}
		}
	}
}
