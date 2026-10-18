// usage: tuilint spec [TUI.md...] [--frames DIR] [--json out.json]
//
// Lints the plan before the code: the States-and-keys table and the brand read of a
// TUI.md written by init (references/cmd-init.md). With no file it reads ./TUI.md.
// A finding prints as [SEV] ID  file:line  message (--json writes it with "where", the
// line, as frames and code do). P0/P1 findings fail the run (exit 1); P2 are warnings to
// fix or justify in one line. Exit 2 on usage errors, an unreadable file, or a --frames
// that is not a directory. --frames names the golden frames (<state>-80x24.ansi, by
// harness name) to hold the table to (S-GOLDEN) and the screens' region budgets
// (S-REGION).
//
// It reads the table in the template's format, keys as `key action · key action`, and
// a `Global:` line (or a row whose state is "global") whose keys apply in every state.
// A Global line counts only after the table and before the next heading at the table
// section's level or above; elsewhere it is prose:
//
//	| state (harness name) | screen / overlay | keys                                      |
//	| services             | services         | **↑↓ move** · **enter open** · **d deploy** · **q quit** · **? help** · l logs · c copy id |
//	| services-empty       | services         | r refresh · ? help · q quit |
//	| deploy               | overlay          | ←→ choose · enter confirm · esc cancel ‖ y copy id · o open dashboard |
//	| reply                | overlay          | enter send · shift+enter/ctrl+j newline · esc cancel |
//	Global: ctrl+c quit
//
// The footer shows a state's keys in table order. A cell marks which ones in one of two
// ways: its **bold** items are the footer, or the items before a ‖ are. The other keys
// are help-only (show: false in the action table, listed by ? help). A cell with
// neither marker is all footer, so older tables read as before. A state whose footer
// carries a badge (ui.Footer's left: a mode, PLAN) names it as the cell's first item,
// `[plan] · **enter send** · …`; it is no key, and it takes its width and a space from
// the footer.
//
// A key token may list alternatives with "/" (`shift+enter/ctrl+j`, `↑↓/jk`, `1-9`), and
// an item may list more tokens before its action: `j/k, ↑/↓ move`, `h/l or ←/→ switch`,
// `q / ctrl+c quit`, `[ ] back`. A trailing ":" ends the keys (`enter: open`).
//
// An item the footer shows only under a condition ends with it in parentheses, after
// `if`, `when` or `while`: `**esc clear filter (if a filter is applied)**`, `**↑↓ scroll
// (while the body overflows)**`. An item whose screen comes later in the slice ends with
// `(later)` or `(later: <screen>)`. The rules read the item without the mark (it takes no
// footer width), and S-GOLDEN doesn't ask a golden to show it.
//
//	S-TABLE      [P1] no States-and-keys table in the template's format: nothing else can be checked
//	             [P2] a key token it can't read as a key: the key rules skip it
//	             [P2] a keys cell with both footer markers, bold and ‖: pick one (the ‖ is read)
//	S-KEYAMBIG   [P1] a key that terminals without key disambiguation (tmux by default, many
//	                  terminals, most SSH sessions) deliver as another key, with no fallback for
//	                  the same action in the same cell. Bubble Tea v2.1.0 asks for disambiguation;
//	                  where the terminal ignores that, Ultraviolet decodes legacy bytes, so:
//	                    shift+ or ctrl+ on enter, tab, backspace, esc, space → the bare key
//	                    (shift+tab and ctrl+space have their own codes and are fine)
//	                    ctrl+m → enter    ctrl+i → tab    ctrl+[ → esc    ctrl+@ → ctrl+space
//	                    ctrl+backspace → ctrl+h (BS) on xterm-like terminals, backspace (DEL) on the rest
//	                    ctrl+shift+<letter> → ctrl+<letter>    ctrl+/ → ctrl+_ (0x1f)
//	                    ctrl+<digit> or other ctrl+<punctuation> → no legacy code at all
//	                    meta+, hyper+, super+ → only the kitty protocol sends them
//	                  Fix: `shift+enter/ctrl+j newline` (ctrl+j is LF, which every terminal sends).
//	             [P2] ctrl+h: it arrives as itself, but BS (0x08) also decodes as ctrl+h, and
//	                  xterm-like terminals send BS for ctrl+backspace (a few for backspace)
//	S-KEYDUP     [P1] one key bound to two different actions in one state's cell, or in Global:
//	                  the first match wins and the other hint lies. A state that binds a
//	                  Global key to its own action overrides it (dispatch runs overlay, then
//	                  focused region, then globals), so that is not a finding.
//	S-KEYSHADOW  [P2] a Global key that a Bubbles v2.2.1 textarea or textinput consumes by
//	                  default (ctrl+a b d e f g h k m n p t u v w, ctrl+shift+c, alt+b c d f l u < >,
//	                  enter, backspace, delete, home, end, pgup, pgdown, the arrows and their shift,
//	                  ctrl and alt forms, and every printable key, which it types),
//	                  in a state with text entry that doesn't bind the key itself. A state counts
//	                  as text entry only when its row has a newline action, says "type to" or
//	                  "typing", or names textarea, textinput, text input/box/field, input,
//	                  editor or compose(r). A row that only says filter or search doesn't count.
//	S-GLYPHWIDTH [P2] a symbol of East Asian Width "ambiguous" (◑ ★ ⇒ …: one cell in most
//	                  terminals, two in CJK locales) in a brand, motif, glyph or identity
//	                  section or in the key table, with no line anywhere that names its ASCII
//	                  twin (a line holding the glyph and the word "ascii"). Skipped: glyphs
//	                  the kit's ASCII map already draws a twin for (─ │ ▌ ● ◐ ○ ◆ ▲ ■ → …,
//	                  theme.ASCIITwin), arrows (U+2190–U+21FF) outside the key table, which
//	                  are prose, and arrow keys in a key token (↑↓←→), which are key names.
//	S-FOOTER     [P2] a state's footer keys (all of them, the bold ones, or those before ‖) don't
//	                  fit the footer the kit draws at 80 columns: ui.Help packs `key action` hints
//	                  two spaces apart into the 76 columns inside the kit's default 2-cell gutters
//	                  (it assumes them: a theme with a 1-cell gutter has 2 more columns), keeps 3
//	                  for "…" while hints remain, pins the first `?` or `ctrl+g` hint last wherever
//	                  it sits, and drops the rest. A token pairing a key some terminals can't tell
//	                  apart with its fallback (`shift+enter/ctrl+j`) counts as its widest single
//	                  alternative: ui.HelpKey draws only the one that works. A help item that
//	                  lists aliases for the dispatcher (`?/h/f1 help`) counts as its first key,
//	                  `? help`, the hint the footer draws and pins. A badge the cell
//	                  names first (`[plan] · …`) takes its width and a space from the 76; status
//	                  text on the row takes more. The message names what it drops and suggests a
//	                  footer that fits, with the state's way out (esc, quit, back, close, cancel)
//	                  and its one main action kept and the other keys moved out from the end.
//	                  Global keys are not in a state's footer, and a state whose cell marks no
//	                  footer keys (`‖ q quit`: toosmall, drawn by ui.TooSmallText) has none.
//	             [P2] a state with a marked footer and help-only keys whose footer shows no key
//	                  that opens help (`?`, ctrl+g, an action that says help), when Global doesn't
//	                  bind one: nobody finds the help-only keys. The help overlay itself (a state
//	                  named help, keys or shortcuts, or help-<over what>) is full help.
//	S-ACCENT     [P2] the `accents:` lines give each hue one meaning (`accents: amber = new,
//	                  nothing else; teal = running`) and another line puts "the accent" or "the
//	                  primary hue/colour" on something else (the wordmark, the spinner's dots, a
//	                  refresh ramp), quoting both. A line that names a hue or a word of a meaning
//	                  is the same meaning; "primary pane", "primary key" are not the hue. Table
//	                  rows and fenced sketches are not read. A warning: the critic stays the judge.
//	S-SLOTS      [P2] in a brand section, a `remember:` line with no `move:` or `signature:`
//	                  line (in backticks or a fence it is prose, not a slot): a brand read
//	                  written before those two replaced it. Name one region elsewhere in the
//	                  screen's order than the kit's (or gone, or added) and one element the
//	                  kit doesn't draw, as references/brand.md § Move A Region says; brand,
//	                  craft and the critic look for those two.
//	             [P2] in § Screens (`Screens`, `4. Screens`, and a sibling right after it such
//	                  as `Secondary screens` or `Overlays`), a block (a heading under it) after
//	                  the first, which is read as the main screen (so its block comes first),
//	                  with no `brand:` line: a secondary screen or overlay that doesn't say what
//	                  the brand's three words change on it (copy voice, dialog form, footer
//	                  phrasing). A heading with no job (no dash or colon) whose next heading is
//	                  deeper groups blocks (`Main`, `Overlays`) and is not one. `brand: nothing,
//	                  because …` counts; a bare `nothing`, `none`, `n/a` or `-`, an empty value,
//	                  a line in a ``` or ~~~ fence (the sketch) or in backticks doesn't. Two
//	                  brands on one spec otherwise share every screen behind the main one word
//	                  for word (references/cmd-shape.md § Behind The Main Screen).
//	S-GOLDEN     [P2] (with --frames) a state's footer in <state>-80x24.ansi and its row disagree:
//	                  the footer shows a hint whose keys neither the row nor Global binds (row
//	                  keys over an empty table), or lacks a key the row puts in the footer while
//	                  nothing is dropped behind … (not one marked `(if …)` or `(later)`). A hint is a key drawn in another style than its
//	                  action, as ui.Help draws it, so status text on the row is not one; with no
//	                  style (NO_COLOR), a prompt or badge one blank before the first hint ("/yes▏
//	                  enter keep") is not part of it; ASCII twins (^v, <>, ^v<>) read as arrows,
//	                  and a lone v as v or ↓; a row "as list; x cancel" is list's keys and
//	                  these; a badge one blank before the first hint ("[PROD] enter next") is not
//	                  part of it, nor is ui.Footer's right status (what follows the help hint
//	                  ui.Help pins last, and a count such as `4 of 101` or `1/9` anywhere); a
//	                  frame whose last row holds no hint (a toast) is not compared.
//	S-REGION     [P2] (with --frames) a region of a screen block's budget line (`80×24: header
//	                  0,0 80×1 · list 1,2 78×20 · …`) whose whole rectangle is blank in the
//	                  screen's golden at that size, <screen>-80x24.ansi or -120x40.ansi (the
//	                  block's heading up to its job): the region is drawn elsewhere or not at
//	                  all, so the budget went stale. A gap (gap, gutter, margin, spacer) is
//	                  blank by design; a region with any text in it is not reported, even when
//	                  the text is another region's that moved into it.
package speclint
