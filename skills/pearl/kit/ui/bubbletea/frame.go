package ui

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// What an app's harness needs to write a frame as a terminal would show it: the
// profile its flags and NO_COLOR ask for, the frame downsampled to it, and its size
// checked.

var profileNames = map[string]colorprofile.Profile{"truecolor": colorprofile.TrueColor, "ansi256": colorprofile.ANSI256,
	"ansi": colorprofile.ANSI, "ascii": colorprofile.Ascii}

// RenderProfile is the profile renders are drawn for: name (a -profile flag: truecolor,
// ansi256, ansi or ascii) when explicit, else no colour when NO_COLOR is true as
// strconv.ParseBool reads it (1, t, true…; "yes" or "x" is not), which is exactly how
// colorprofile v0.4.3, and so Bubble Tea's detection for the live program, reads it.
func RenderProfile(name string, explicit bool, noColor string) (colorprofile.Profile, error) {
	if off, _ := strconv.ParseBool(noColor); !explicit && off {
		return colorprofile.Ascii, nil
	}
	p, ok := profileNames[name]
	if !ok {
		return 0, fmt.Errorf("-profile %q: want truecolor, ansi256, ansi or ascii", name)
	}
	return p, nil
}

// Downsample writes a truecolour frame through colorprofile, as Bubble Tea's renderer
// does for a terminal at prof.
func Downsample(s string, prof colorprofile.Profile) string {
	if prof == colorprofile.TrueColor {
		return s
	}
	var b bytes.Buffer
	wr := &colorprofile.Writer{Forward: &b, Profile: prof}
	_, _ = wr.WriteString(s)
	return b.String()
}

// CheckSize reports whether txt, a frame with its escapes stripped, is exactly w×h
// cells; name is the state, for the error.
func CheckSize(txt, name string, w, h int) error {
	rows := strings.Split(txt, "\n")
	if len(rows) != h {
		return fmt.Errorf("%s: %d rows, want %d", name, len(rows), h)
	}
	for i, r := range rows {
		if n := lipgloss.Width(r); n != w {
			return fmt.Errorf("%s: row %d is %d cells, want %d", name, i, n, w)
		}
	}
	return nil
}
