package ui

import (
	"testing"

	"github.com/charmbracelet/colorprofile"
)

// -profile wins; without it NO_COLOR means no colour when strconv.ParseBool says true,
// exactly as colorprofile reads it for the live program.
func TestRenderProfile(t *testing.T) {
	for _, c := range []struct {
		name     string
		explicit bool
		noColor  string
		want     colorprofile.Profile
	}{
		{"truecolor", false, "", colorprofile.TrueColor},
		{"truecolor", false, "1", colorprofile.Ascii},
		{"truecolor", false, "true", colorprofile.Ascii},
		{"truecolor", false, "0", colorprofile.TrueColor},
		{"truecolor", false, "false", colorprofile.TrueColor},
		{"truecolor", false, "yes", colorprofile.TrueColor},
		{"truecolor", false, "x", colorprofile.TrueColor},
		{"ansi", true, "1", colorprofile.ANSI},
		{"ansi256", true, "", colorprofile.ANSI256},
	} {
		if got, err := RenderProfile(c.name, c.explicit, c.noColor); err != nil || got != c.want {
			t.Errorf("RenderProfile(%q, %v, %q) = %v %v, want %v", c.name, c.explicit, c.noColor, got, err, c.want)
		}
	}
	if _, err := RenderProfile("sixteen", true, ""); err == nil {
		t.Error("RenderProfile(sixteen) accepted")
	}
}
