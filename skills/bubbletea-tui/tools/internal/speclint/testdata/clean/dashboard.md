# Tidewatch TUI

Read before any UI change. Update it when a decision changes. Tidewatch and its services
are fictional.

## Design

### Product
Tidewatch watches a fleet's job queues: queued → running → done, with the failures
pulled to the top. Operators leave it open on a second monitor all shift.

### Users and terminals
Operators on desktops and over SSH, inside tmux; 120×40 common, 80×24 floor; dark first.

### Brand read
palette: deep teal neutrals; coral means "failing" and nothing else
focus: the ▌ gutter in fg on a bgSubtle row
tone: quiet, factual, tidal
geometry: rules not boxes; ─ section rules, │ between panels
motif: a job moves left to right through its lanes, queued → running → done; a retry ⇒ back to queued
remember: coral rows are the only colour on screen

### Theme
themes/tidewatch.json   (palette derive from #0E4D52 #FF6F59)

### Screens
board — every queue, its lanes and its failing jobs
job — one job's attempts and log tail

## Architecture

### Stack
Go 1.26.7; charm.land/bubbletea/v2 v2.1.0, lipgloss/v2 v2.0.6, bubbles/v2 v2.2.1.

### States and keys
One action table per state; footer and dispatcher both read it.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| board                | board            | ↑↓/jk move · enter open · tab/shift+tab panel · r refresh · ? help · q quit ‖ 1-4 jump to panel |
| board-empty          | board            | r refresh · ? help · q quit |
| board-error          | board            | r retry · ? help · q quit |
| board-partial        | board            | ↑↓/jk move · enter open · r retry failed · ? help · q quit |
| job                  | job              | ↑↓/jk scroll · g/G top/bottom · R rerun · esc back |
| help                 | overlay          | ? close · esc close |

Global: ctrl+c quit

### Async and the clock
refresh → fetchCmd → boardMsg; stale results: gen. The lanes animate only while a job
moves: queued → running is one 200ms slide, then the tick stops.

## Decisions
- 2026-10-16 coral only for failing because focus in coral hid failures
- Global: g is reserved for go-to-top in every list, so no state binds g to anything else

## Assumed
- 80×24 floor
