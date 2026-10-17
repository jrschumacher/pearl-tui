// Palette derives a terminal role palette from brand colours, and checks one.
//
// Brand colours are measured from the brand (logo, site CSS, style guide), never invented.
// This turns 1–3 of them into the full role set a TUI needs, in OKLCH so lightness steps
// are perceptual, with neutrals tinted toward the brand hue (the single biggest reason a
// palette reads "designed" instead of "default grey"), status colours pulled into the
// brand's lightness/chroma band so they sit with it, and every text role checked for
// contrast against the background it will actually sit on.
//
//	palette derive "#6B50FF" "#FF60FF" [--accent "#E8FE96"] [--bg "#201F26"] [--light-bg "#FBFAF7"]
//	               [--tint 0.012] [--name crush] [--out theme-roles.json] [--swatch swatch.ansi]
//	palette check theme.json        contrast report for an existing theme's roles (exit 1 on fail)
//	                                a theme may accept a deviation in writing:
//	                                "contrastAccept": {"dark.primary": "the brand's own hex, used as a fill"}
//
// `derive` prints the roles as JSON (`{"dark": {...}, "light": {...}}`, the theme contract's
// shape), a contrast table, and the nearest xterm-256 index for each role (for the
// ANSI256 fallback). `--swatch` writes an .ansi file you can render with cellshot.
// Options take their value as the next argument or after "=".
//
// `check` fills the on-colours a theme leaves out exactly as the kit's engine does
// (tuikit/theme), so it checks the inks the app will draw. It reads role colours by the
// engine's rule too: "#RGB" or "#RRGGBB", nothing else. Lip Gloss reads "6B50FF" (no #)
// as an ANSI index or no colour, and "#6B50FFAA" as no colour, so those exit 2.
// It exits 2 on the theme's glyphs too (theme.CheckGlyphs): a glyph with no ASCII twin
// (in glyphs.ascii or the kit's map), a twin that isn't ASCII, or two marks of one
// meaning (done/current/todo/pending/partial/warn/fail, radio on/off, check on/off,
// thumb/track, focus/blur bar) that share a twin under -ascii. And on its expressions
// (theme.CheckExpressions): a key the kit doesn't read or a value it doesn't know, which
// theme.Load draws as the default silently ("dialog": "rule" for "rules"), named with
// the values the kit knows.
//
// It reports, on stderr, state roles that land on one of the 16 colours (-profile ansi)
// after the theme's "ansi16" pins, each with the pin that separates them:
//
//	"ansi16": {"dark": {"primary": 11}, "light": {"primary": 3}}
//
// Under "focus": "fg" primary is a state, and a clash with it exits 1: the brand's one
// promise (amber = new) breaks where amber downsamples to error's red. A clash among
// error, warning, success and info is a warning: their glyphs and words still differ.
// A pin that names no role, no mode or no colour 0-15 exits 2, as theme.Load fails it.
//
// Thresholds (WCAG 2 contrast ratio against the role's background):
//
//	fg ≥ 7      fgMuted ≥ 4.5     fgSubtle ≥ 2.6 (hints only, never the only carrier of meaning)
//	primary, success, error, warning ≥ 3.5    secondary, info ≥ 3.0    tertiary, accent ≥ 4.5
//	border 1.14–1.9 and bgSubtle 1.1–2.2 (barely there, but there)
//	fg ≥ 4.5 and fgMuted ≥ 3.0 on bgSubtle too
//	onPrimary, onWarning ≥ 4.5 on their fill; onSecondary, onError, onSuccess, fgSelected ≥ 3.5
//	plus the pairs a theme's expressions put on screen: "footer": "bar" draws the key
//	hints (fgSubtle) on bgSubtle, so that pair must clear tuilint's 2.3 floor; and minus
//	the ones they take off: under "focus": "fg" no selection is a primary fill, so
//	fgSelected on primary isn't checked (primary on bg and onPrimary still are)
//
// derive aims higher than these floors (fgMuted 4.6, fg 11–12, accent and status 4.5).
// On a mid-tone fill where neither ink clears its floor, derive deepens the ink toward
// black or white along its hue (the engine does the same), so the brand keeps its hex.
//
// Exit status: 0 OK, 1 contrast problems or a state clash under focus "fg", 2 usage or
// input errors.
package main

import (
	_ "embed"
	"os"

	"tuitools/internal/cmddoc"
	"tuitools/internal/palette"
)

//go:embed main.go
var src string

func main() {
	os.Exit(palette.Main(os.Args[1:], cmddoc.Text(src), os.Stdout, os.Stderr))
}
