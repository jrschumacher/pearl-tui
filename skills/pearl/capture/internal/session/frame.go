package session

import (
	"fmt"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Encode writes lines as a frame in the kit harness's format: one row per line, SGR
// only (no cursor moves), every cell written, each row closed with a reset when it
// leaves a style open, rows joined by "\n", and a final "\n".
func Encode(lines []uv.Line) string {
	rows := make([]string, len(lines))
	for i, l := range lines {
		rows[i] = l.Render()
	}
	return strings.Join(rows, "\n") + "\n"
}

// CheckSize fails a frame that isn't exactly w×h cells.
func CheckSize(frame string, w, h int) error {
	rows := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	if len(rows) != h {
		return fmt.Errorf("frame has %d rows, want %d", len(rows), h)
	}
	for i, r := range rows {
		if n := ansi.StringWidth(r); n != w {
			return fmt.Errorf("frame row %d is %d cells, want %d", i, n, w)
		}
		if strings.ContainsAny(ansi.Strip(r), "\x1b\r\t") {
			return fmt.Errorf("frame row %d holds a control character", i)
		}
	}
	return nil
}
