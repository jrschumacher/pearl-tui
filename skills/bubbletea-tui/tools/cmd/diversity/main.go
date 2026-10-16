// Diversity defines "different", or every brand picks the same different.
//
//	diversity themes/*.json kit/themes/ink.json kit/themes/charm.json [--max-shared 9] [--max-popular 0.5]
//	          [--baseline ink,charm --min-distance 8]
//
// A baseline is matched by theme name, so its file must be among the themes: an explicit
// --baseline name that matches none exits 2 (the default ink,charm only notes it on
// stderr; --baseline "" turns the rule off).
//
// Tabulates every theme's visible choices (expressions + signature glyphs) and fails when:
//
//   - two themes agree on more than --max-shared of the counted axes (one design, two colours)
//   - one value is used by more than --max-popular of the themes (a house style, not a trait);
//     two-valued axes are exempt (one of two values always has a majority)
//   - a theme sits closer than --min-distance axes to a baseline (the default ink theme, the
//     Crush study): it is the default with a new palette, which is exactly the slop it replaces
//
// Dead axes (choices nothing renders) are shown as "-" and never count: headerRule under a
// bar/stamp/minimal header, wordmark under a banner/stamp header, dialogTitle under a rules
// dialog.
//
// It is a floor, not a target. A plain brand stays plain and spends its differences on the
// two or three moves it owns; never move an axis just to pass. Learned over three rounds of
// nine brands: parallel agents given a bigger vocabulary converge on the NEWEST options, so
// after a first pass the orchestrator writes a seating plan (each contested value goes to
// the brand whose product actually has it) and lets each brand finish in its lane.
//
// Exit status: 0 OK, 1 problems, 2 usage or input errors.
package main

import (
	_ "embed"
	"os"

	"tuitools/internal/cmddoc"
	"tuitools/internal/diversity"
)

//go:embed main.go
var src string

func main() {
	os.Exit(diversity.Main(os.Args[1:], cmddoc.Text(src), os.Stdout, os.Stderr))
}
