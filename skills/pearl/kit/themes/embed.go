// Package themes holds the theme JSON the binary carries. go:embed can't reach outside
// its own package, so the directory that holds themes/<brand>.json is a package: the
// kit's showcase and a project's cmd/<app> read Default and draw the brand with no
// -theme flag, wherever the binary is run from.
package themes

import (
	"embed"
	"fmt"
	"slices"
)

// FS holds every theme in this directory.
//
//go:embed *.json
var FS embed.FS

// Default is the theme a binary built from this module draws with no -theme flag: the
// one JSON in this directory (a project's themes/<brand>.json), or ink.json among several
// (the kit's gallery). A project that keeps several themes names its own here.
func Default() ([]byte, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	switch {
	case len(names) == 1:
		return FS.ReadFile(names[0])
	case slices.Contains(names, "ink.json"):
		return FS.ReadFile("ink.json")
	}
	return nil, fmt.Errorf("themes: %d themes and no ink.json: name the default in themes.Default", len(names))
}
