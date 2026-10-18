# Dunlin TUI

Read before any UI change. Update it when a decision changes.

## Design

### Product
A live monitor for one build host: what is it doing, and which process is responsible.

### Brand read
palette   one slate blue that points; amber is high, red is pinned
tone      terse: the thing, the number, then stop
geometry  square and light
density   compact
type      lowercase labels
motif     ◂ the probe
move: the alarm row (watchful): an added row under the header that names what is wrong now, before any number
signature: a 12-cell load trace beside the name in the header
skeleton: head bar · status header · order summary-first · foot keys-status · verdict top · chart trace · chartGlyphs blocks · turnEnd -

### Theme
themes/dunlin.json   (palette derive "#5B7FC4", chosen)

### Screens

#### overview — what is the host doing; the eye lands on the alarm row
80×24:  header 0,0 80×1 · alarm 0,1 80×1 · resources 0,2 80×7 · list 0,9 80×14 · footer 0,23 80×1
