// usage: tuilint spec [TUI.md...] [--json out.json]
//
// Lints the plan before the code: the States-and-keys table and the brand read of a
// TUI.md written by init (references/cmd-init.md). With no file it reads ./TUI.md.
// A finding prints as [SEV] ID  file:line  message. P0/P1 findings fail the run
// (exit 1); P2 are warnings to fix or justify in one line. Exit 2 on usage errors.
//
// It reads the table in the template's format, keys as `key action · key action`, and
// a `Global:` line (or a row whose state is "global") whose keys apply in every state:
//
//	| state (harness name) | screen / overlay | keys                                      |
//	| reply                | overlay          | enter send · shift+enter/ctrl+j newline · esc cancel |
//	Global: ctrl+c quit · ctrl+p switch
//
// A key token may list alternatives with "/" (`shift+enter/ctrl+j`, `↑↓/jk`, `1-9`), and
// an item may list more tokens before its action: `j/k, ↑/↓ move`, `h/l or ←/→ switch`,
// `q / ctrl+c quit`, `[ ] back`. A trailing ":" ends the keys (`enter: open`).
//
//	S-TABLE      [P1] no States-and-keys table in the template's format: nothing else can be checked
//	             [P2] a key token it can't read as a key: the key rules skip it
//	S-KEYAMBIG   [P1] a key that terminals without key disambiguation (tmux by default, many
//	                  terminals, most SSH sessions) deliver as another key, with no fallback for
//	                  the same action in the same cell. Bubble Tea v2.1.0 asks for disambiguation;
//	                  where the terminal ignores that, Ultraviolet decodes legacy bytes, so:
//	                    shift+ or ctrl+ on enter, tab, backspace, esc, space → the bare key
//	                    (shift+tab and ctrl+space have their own codes and are fine)
//	                    ctrl+m → enter    ctrl+i → tab    ctrl+[ → esc    ctrl+@ → ctrl+space
//	                    ctrl+h → the byte backspace sends on many terminals
//	                    ctrl+shift+<letter> → ctrl+<letter>    ctrl+/ → ctrl+_ (0x1f)
//	                    ctrl+<digit> or other ctrl+<punctuation> → no legacy code at all
//	                    meta+, hyper+, super+ → only the kitty protocol sends them
//	                  Fix: `shift+enter/ctrl+j newline` (ctrl+j is LF, which every terminal sends).
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
//	S-GLYPHWIDTH [P2] a symbol of East Asian Width "ambiguous" (● ○ ◐ ◆ ▲ ─ │ ▌ █ ★ …: one
//	                  cell in most terminals, two in CJK locales) in a brand, motif, glyph or
//	                  identity section or in the key table, with no line anywhere that names its
//	                  ASCII twin (a line holding the glyph and the word "ascii"). Arrow keys in
//	                  a key token (↑↓←→) are key names, not glyphs, and are skipped.
package speclint
