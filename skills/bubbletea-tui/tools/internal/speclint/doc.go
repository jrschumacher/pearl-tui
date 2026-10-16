// usage: tuilint spec [TUI.md...] [--json out.json]
//
// Lints the plan before the code: the States-and-keys table and the brand read of a
// TUI.md written by init (references/cmd-init.md). With no file it reads ./TUI.md.
// A finding prints as [SEV] ID  file:line  message. P0/P1 findings fail the run
// (exit 1); P2 are warnings to fix or justify in one line. Exit 2 on usage errors.
//
// It reads the table in the template's format, keys as `key action · key action`, and
// a `Global:` line (or a row whose state is "global") whose keys apply in every state.
// A Global line counts only after the table and before the next heading at the table
// section's level or above; elsewhere it is prose:
//
//	| state (harness name) | screen / overlay | keys                                      |
//	| inbox                | inbox            | **↑↓/jk move** · **enter open** · **q quit** · **? help** · r refresh |
//	| thread               | thread           | ↑↓ scroll · r reply · esc back · ? help ‖ y copy link · o open in browser |
//	| reply                | overlay          | enter send · shift+enter/ctrl+j newline · esc cancel |
//	Global: ctrl+c quit · ctrl+p switch
//
// The footer shows a state's keys in table order. A cell marks which ones in one of two
// ways: its **bold** items are the footer, or the items before a ‖ are. The other keys
// are help-only (show: false in the action table, listed by ? help). A cell with
// neither marker is all footer, so older tables read as before.
//
// A key token may list alternatives with "/" (`shift+enter/ctrl+j`, `↑↓/jk`, `1-9`), and
// an item may list more tokens before its action: `j/k, ↑/↓ move`, `h/l or ←/→ switch`,
// `q / ctrl+c quit`, `[ ] back`. A trailing ":" ends the keys (`enter: open`).
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
//	S-GLYPHWIDTH [P2] a symbol of East Asian Width "ambiguous" (◐ ★ █ ⇒ …: one cell in most
//	                  terminals, two in CJK locales) in a brand, motif, glyph or identity
//	                  section or in the key table, with no line anywhere that names its ASCII
//	                  twin (a line holding the glyph and the word "ascii"). Skipped: glyphs
//	                  the kit's ASCII map already draws a twin for (─ │ ▌ ● ○ ◆ ▲ ■ → …,
//	                  theme.ASCIITwin), arrows (U+2190–U+21FF) outside the key table, which
//	                  are prose, and arrow keys in a key token (↑↓←→), which are key names.
//	S-FOOTER     [P2] a state's footer keys (all of them, the bold ones, or those before ‖) don't
//	                  fit the footer the kit draws at 80 columns: ui.Help packs `key action` hints
//	                  two spaces apart into the 76 columns inside the kit's default 2-cell gutters
//	                  (it assumes them: a theme with a 1-cell gutter has 2 more columns), keeps 3
//	                  for "…" while hints remain, pins a last `?` or `ctrl+g` hint, and drops the
//	                  rest. The message names what it drops and suggests a footer that fits, with
//	                  the state's way out (esc, quit, back, close, cancel) and its one main action
//	                  kept and the other keys moved out from the end. Global keys are not in a
//	                  state's footer; a mode badge or status text on the row takes more.
//	             [P2] a state with a marked footer and help-only keys whose footer shows no key
//	                  that opens help (`?`, ctrl+g, an action that says help), when Global doesn't
//	                  bind one: nobody finds the help-only keys.
package speclint
