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
// (tuikit/theme), so it checks the inks the app will draw.
//
// Thresholds (WCAG 2 contrast ratio against the role's background):
//
//	fg ≥ 7      fgMuted ≥ 4.5     fgSubtle ≥ 2.6 (hints only, never the only carrier of meaning)
//	primary, success, error, warning ≥ 3.5    secondary, info ≥ 3.0    tertiary, accent ≥ 4.5
//	border 1.14–1.9 and bgSubtle 1.1–2.2 (barely there, but there)
//	fg ≥ 4.5 and fgMuted ≥ 3.0 on bgSubtle too
//	onPrimary, onWarning ≥ 4.5 on their fill; onSecondary, onError, onSuccess, fgSelected ≥ 3.5
//
// derive aims higher than these floors (fgMuted 4.6, fg 11–12, accent and status 4.5).
//
// Exit status: 0 OK, 1 contrast problems, 2 usage or input errors.
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
