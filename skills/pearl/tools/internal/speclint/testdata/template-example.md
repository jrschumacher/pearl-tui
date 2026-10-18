# Halyard TUI

Read before any UI change. Update it when a decision changes.

## Design

### Product
Halyard deploys and rolls back services. The TUI is for on-call engineers checking
service health, shipping a version and rolling back at 3am. Sessions last minutes.

### Users and terminals
SREs on laptops and over SSH, often inside tmux; 80×24 design floor, 120×40 common;
dark and light; NO_COLOR honoured; ACCESSIBLE=1 for screen readers.

### Brand read
palette: harbour navy neutrals; signal orange means "needs you" (failing, unacked) and nothing else
focus: the ▌ gutter (ascii |) in fg on a bgSubtle row, never in the signal colour
tone: terse, nautical, calm under pressure
geometry: square, rules not boxes; ─ section rules (ascii -)
density: compact
type: tracked uppercase section titles
motif: status in a 2-cell column, ● healthy, ▲ degraded, × down (ascii * ^ x)
move: the failing service's detail leads, above the list, not under it (calm under pressure: what needs you comes first)
signature: a 12-cell strip of the last hour's deploys and roll-backs beside the name in the header, on the services frame as it opens

### Theme
themes/halyard.json   (palette derive from #12355B #FF7A1A, measured from the logo SVG)

### Screens
#### services — health of every service, pick one to act on; the eye lands on the failing service
the move and the signature: § Brand read

#### deploy — ship a version to an environment; its review step is the confirmation (rung 3 via the form: it says what ships where, the button is the act as danger, focus arrives on cancel), so no dialog follows
brand: terse: the fields are nouns (`service`, `version`, `to`) with no helper sentences, and the last step's button says `ship v2.4.1 to prod`, not `submit`

#### confirm-rollback — the one dialog: roll back, with the version named; rung 3: it ships to prod, so it confirms even when the target is healthy
brand: calm under pressure: the question names both versions and what changes (`roll checkout-api back to v2.3.9? v2.4.0 stops taking traffic`); the buttons are `roll back` and `keep v2.4.0`, the safe one focused

### Never
pink-purple, emoji, rounded boxes around regions, "Loading..."

## Architecture

### Stack
Go 1.26.7; charm.land/bubbletea/v2 v2.1.0, lipgloss/v2 v2.0.6, bubbles/v2 v2.2.1,
huh/v2 v2.0.3; ultraviolet v0.0.0-20261008173134-6b8d4baf91b4.
theme/ and ui/ copied from the kit into internal/.

### Root model and regions
One *model in internal/app; regions are structs drawn into rectangles (x,y w×h at 80×24):

    header   0,0    80×1
    list     0,2    32×20
    detail   34,2   46×20
    footer   0,23   80×1

focus: list | detail        overlays: palette, confirm, help (front one gets keys)

### Forms
Huh v2 for the deploy form, themed with the kit's `ui.NewForm` from *theme.Theme.
Hand-written: none.

### Components
Bubbles: textinput (palette filter), viewport (logs). Kit ui: List, Table, Dialog, Footer.

### States and keys
One action table per state; footer and dispatcher both read it.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| services             | services         | ↑↓/jk move · enter open · d deploy · r roll back · q quit · ? help ‖ ctrl+p commands |
| services-empty       | services         | r refresh · ? help · q quit |
| services-error       | services         | r retry · ? help · q quit |
| deploy               | deploy           | enter next field · esc back · ctrl+g help ‖ tab/shift+tab field |
| palette              | overlay (text input) | enter run · ↑↓ choose · esc close |
| confirm-rollback     | overlay          | ←→ choose · enter confirm · esc cancel |

Global: ctrl+c quit

### Harness and fixtures
internal/app/fixtures.go feeds `setState` for the harness (-state, -out) and every test;
fakes: the deploy API is an in-memory `fakeDeployer`, kubectl a recorded transcript.

### Async and the clock
deploy → deployCmd → deployDoneMsg; cancel: the context in m.cancel; stale results: gen.
Frame clock: one gen-stamped tick, armed only while a deploy animates; a stale tick dies.
Product clock: a poll of service health every 5s, its own msg and gen; the handler that changes it (interval, pause) returns the new arm cmd, never a gen bump alone.

### Size floor
Below 60×18: ui.TooSmall. Below 110 columns: the detail pane collapses to a summary line.
Below 30 rows: the sparkline row goes.

### Accessible and non-TTY
ACCESSIBLE=1 or --accessible: no alt screen, Huh WithAccessible(true), status as new
lines. stdout not a TTY: --json, never escape codes.

## First slice
services, deploy and confirm rendered from fixtures through the harness, every state at
80×24 and 120×40, before the deploy API or kubectl are wired.

## Decisions
- 2026-10-16 signal orange only for "needs you" because focus in orange hid failing rows

## Assumed
- 60×18 floor
