// Command cellshot renders ANSI terminal frames to PNG/GIF, honestly, so you can SEE a TUI.
//
// You cannot look at a terminal. This is your eyes. It draws every cell the way a modern
// terminal (Ghostty, Kitty, WezTerm, iTerm2 with built-in box drawing) does: box drawing,
// block elements, shades, quadrants, sextants, braille and powerline separators are drawn
// geometrically (so they join and fill like a real terminal, never as font tofu), text uses
// an embedded DejaVu Sans Mono (the same on every machine), and wide characters take two
// cells, by the same width rules as lipgloss.
//
//	cellshot frame.ansi out.png                    one frame
//	cellshot a.ansi b.ansi c.ansi --sheet out.png  contact sheet, labelled by filename
//	cellshot f*.ansi --gif out.gif --ms 80         animation from frames
//	some-cmd | cellshot - out.png                  from stdin
//
// Options
//
//	--cell WxH        pixel size of one cell (default 12x24; use 16x32 to inspect detail)
//	--bg #rrggbb      terminal default background (default #16161a)   --light  = #fafafa/#222
//	--fg #rrggbb      terminal default foreground (default #d8d8d8)
//	--cols N --rows N expected size; pads the image and reports any mismatch (exit 1)
//	--grid            faint cell grid (for cell-art review)
//	--scale N         integer upscale after rendering (crisp)
//	--label TEXT      caption bar above the image (sheets use file names)
//
// Prints one line per input: size in cells, problems found (rows of unequal width, glyphs no
// font could draw, emoji, non-SGR escape sequences). Exit status 1 if --cols/--rows mismatch.
// Italic is the regular face slanted 11°; emoji have no colour font and draw from the
// embedded faces when they can, as tofu otherwise. Only dependency: Go.
package main
