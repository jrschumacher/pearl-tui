# TUI slop: the fingerprints of a model-made terminal app

If you showed the screen to someone who lives in a terminal and said "a model made this",
would they believe you at a glance? If yes, that is the problem. Each item below is a
default that models reach for because it is the easiest thing to type in Lip Gloss or the
first thing in a README. Each has an ID. The ones a machine can see are checked by
`tuilint` (rule IDs in brackets).

A default is not banned because it is ugly. It is banned because it is *everyone's*, so
choosing it says nothing about this product. The brief can earn any of these back. Say
so in one line when it does.

## Structure

**S1 Box soup** [F-BOXES]. Every region in a `RoundedBorder()`, boxes beside boxes, boxes
inside boxes. Two adjacent borders spend two columns per seam and make the screen read as
a widget demo. *Instead:* separate regions with whitespace, a faint rule
(`Section`), or one background step (`bgPanel`). Keep the border for the one thing that
floats above everything else: a dialog.

**S2 The README layout.** Lip Gloss's README demo (tabs with connected borders, a "dialog"
in a double border in the middle of the page, a colour grid, a status bar of pink and
purple segments) reproduced as an app. It is a demo of the library's features, not a
layout for any product.

**S3 Centre everything.** A title centred, a list centred, a footer centred. Terminals
read top-left to bottom-right; centring is for one thing (a splash, an empty state, a
dialog). A centred list is impossible to scan.

**S4 Help as a paragraph.** `Press q to quit, j/k to move, enter to select…` as one
muted sentence at the bottom. *Instead:* `key desc` pairs from real `key.Binding`s,
packed from the left by importance, truncated with `…`, and changed per context.

**S5 The fixed 80.** Hard-coded widths, no `tea.WindowSizeMsg`, layout breaks at 120 and
tears at 79 [C-SIZE] [F-WIDTH]. *Instead:* budget columns for 80×24 and 120×40, collapse
on purpose, and show "terminal too small" below the minimum.

**S6 Full-bleed text.** Prose running edge to edge at 200 columns. *Instead:* a measure
of 72–100 cells for prose; tables and diffs may go wide.

**S7 Edge-to-edge content** [F-EDGE]. Text in column 0 and in the last column. *Instead:*
a 1–2 cell gutter, the same on every screen.

## Colour

**C1 Pink and purple by default** [F-DEFAULT]. `#7D56F4`, `#FF5FAF`/205, `#EE6FF8`,
62/63/170/212: the colours of the Bubble Tea and Lip Gloss examples and Bubbles' default
list delegate. Huh adds its own: `#7571F9`/`#5A56E0` titles and the `#F780E2` spinner. They are the strongest "made with the tutorial" signal there is. Brand
roles come from the brand (`palette`), never from an example.

**C2 Hex in the view** [C-HEX] [F-OFFPAL]. `lipgloss.Color("#ff0000")` in a screen
function. Ten screens later there are eleven reds. *Instead:* roles (`t.S.Err`,
`t.P.Primary`) from one theme file.

**C3 Rainbow** [F-RAINBOW]. Every column a different colour, every status a different
hue, gradients on body text. Colour carries meaning only when most things don't have it.
Lightness does hierarchy (fg / fgSoft / fgMuted / fgSubtle), and hue is for state and
brand.

**C4 Pure grey on a tinted world.** `#888888` text on a navy background reads dirty.
Neutrals lean toward the background's hue (`palette` does this).

**C5 One text level** [F-LEVELS]. Everything in the default foreground, everything bold
[F-BOLD]. Without three to four text levels there is no hierarchy, only a wall.

**C6 Colour as the only signal.** A selected row marked only by background, an error
only in red. Under NO_COLOR, at ANSI16, and for the 8% of men with colour-vision
deficiency, it is gone. Add a glyph or a word.

**C7 Painted holes.** The theme paints a background, then an inner style reset lets the
terminal's default bg through in the middle of a bar. *Instead:* draw on a canvas and
`Backfill`, or `OnBg` for a string.

**C8 Gradient everything.** A rainbow title, a gradient border on every panel, gradient
body text. A gradient is a highlight for a handful of cells: the wordmark, one rule, a
spinner.

**C9 Invisible faintness** [F-CONTRAST]. `fgSubtle` at 1.8:1 for anything a user must
read. The faintest level is ≥ 2.6:1 and only for hints.

**C10 A library left on its defaults** [C-HUHTHEME]. A Huh form, field or spinner, or a
Bubbles spinner, with no theme: it renders the library's palette in the middle of yours,
most often Huh's pink spinner. *Instead:* theme it from the brand theme with
`ui.NewForm` (`components.md` § Forms And Wizards).

## Glyphs

**G1 Emoji as icons** [F-EMOJI] [C-EMOJI]. 🚀 ✅ ❌ 📁 ⚡. Widths disagree across terminals
(the row tears), the colours ignore your palette, and it is the most recognisable
model-default there is. *Instead:* the theme's glyph vocabulary (`✓ × ● ○ ◆ › ▸ ╱`).

**G2 Nerd Font dependency** [F-TOFU]. Private-use icons with no fallback, so users
without the font see tofu boxes. Behind a flag at most, with a tier-0–4 fallback.

**G3 Figlet "standard".** The default figlet banner for a title. *Instead:* a
`wordmark.py` raster in the brand's own typeface, tracked caps, or the plain name in a
gradient.

**G4 Braille everywhere.** `⣾⣽⣻` as the spinner for everything, chosen because it was
first in the list. Choose the spinner for the brand's voice (see `motion.md`).

**G5 ASCII-art borders as decoration.** `====` and `****` rows framing sections.

**G6 Mixed vocabularies.** `✓` for done in one place, `[x]` in another, `✔` in a third.
One glyph per meaning, from the theme.

**G7 A glyph typed in a screen** [C-GLYPH]. `"▏"` as the caret, `" · "` between details,
`"↑ 3 more"`: the brand can't change it, ASCII mode has no twin for it, and the next
screen types a different one. *Instead:* `t.G.Caret`, `t.G.Sep`, `t.G.More`; a glyph
the theme lacks becomes a `Glyphs` field with a twin.

## Motion

**M1 The 60 fps idle** [C-FASTTICK]. A tick running while nothing moves: CPU and SSH
bytes for nothing. *Instead:* one shared clock that ticks only while something visible
animates.

**M2 "Loading..."** A spinner with no subject. *Instead:* say what is happening and how
far along it is ("indexing 1,204 files", "deploying 2/3 pods").

**M3 Fake progress.** A progress bar that advances on a timer. If you don't know the
fraction, show an indeterminate spinner and the elapsed time.

**M4 Animation as decoration.** A shimmer on a static header, a pulsing border. Motion
means "something is happening". Use it only when something is.

**M5 Splash that outlives the work.** A 2-second intro before every launch. Show it only
while something actually loads, or ≤ 600 ms on first run.

## Interaction

**I1 Blocking Update** [C-SLEEP] [C-VIEWIO]. `time.Sleep`, HTTP or file I/O in `Update`
or `View`. The UI freezes and keypresses queue. Everything slow is a `tea.Cmd`.

**I2 Two focuses / no focus.** A focused input *and* a highlighted list row, or nothing
marked at all. Exactly one visible focus, always.

**I3 Focus that shifts layout.** Adding a border or a `> ` prefix to the focused item
pushes the text over. The focus marker occupies the same cells focused or not (`Gutter`).

**I4 Lying help** [C-KEYS]. The help line says `e edit`; `e` does nothing on this screen.
Help and dispatch read one per-state action table, and a test presses every key.

**I9 The wrong target.** Select the fifth row, press `r`, and the dialog offers to roll
back the first. Fixtures hide it because the first row is always the interesting one.

**I10 The multiplying clock.** Every key press returns a new `tea.Tick`, so after ten
keys there are ten loops and the spinner runs at 10× speed. One clock, with a generation
number, armed only if not already running.

**I5 `q` in a text field.** Quit bound to `q` while the user is typing a name. Global
keys yield to the focused input; `ctrl+c` always quits (or confirms quitting).

**I6 Esc does nothing.** Every overlay and sub-screen closes on `esc`, and `esc` never
discards typed content without asking.

**I7 Destructive default.** A "Delete" dialog with Delete focused. The safe option is
the default; the destructive one is coloured and needs a deliberate move.

**I8 fmt.Print in a program** [C-PRINT]. Corrupts the frame. Use `tea.Println` (above the
program), a log file (`tea.LogToFile`), or a status line.

**I11 Canvas-only.** The only way to do the task is the full-screen UI, with spinners
repainting in place and no accessible or plain mode. Screen-reader users are locked
out (`interaction.md` § Screen Readers).

## Copy

**W1 Robot voice.** "An error occurred." "Operation completed successfully." Say what
happened, to what, and what to do next: "Can't reach prod (timeout after 10s). Check the
VPN, or press r to retry."

**W2 Title Case Everything.** Sentence case for content and labels; the theme decides if
titles go UPPER as a brand move.

**W3 Exclamation marks and cheer** where the brand is calm. Voice comes from the brand
read, not from a model's default enthusiasm.

**W4 Placeholder content.** "Item 1, Item 2", "Lorem ipsum", "foo/bar". Design with the
real data, including the long name, the failing row and the empty list.

## The TUI slop test (run it on every screen before showing it)

1. Squint at the PNG. Can you tell which product this is, or is it "a Bubble Tea app"?
2. Count the borders. More than one box on screen? Justify each, or remove it.
3. Find the focus. Exactly one thing is marked, and you found it in under a second.
4. Strip the colour (`-profile ascii`). Is selection, error and focus still visible?
5. Resize to 80×24 and 200×60. Same design, more or less room, nothing torn.
6. Read every word aloud. Would a person who works here write it?
7. Look for anything pink-purple, any emoji, any `Loading...`. Remove it.
8. Name the one thing a user will remember about this screen. If you can't, it has no
   identity yet. Go back to the brand read.
