# Rigging TUI

Read before any UI change. Rigging is fictional.

## Design

### Product
Rigging stages, commits and pushes in one screen, for people who live in the terminal.

### Brand read
palette: tar black neutrals; rope gold for added lines, rust for removed
focus: the ▌ gutter (ascii |)
tone: plain, quick
motif: file state in a 2-cell column, + added, ~ modified, - removed

### Theme
themes/rigging.json   (palette derive from #1A1814 #D9A441)

## Architecture

### Components
Bubbles: textarea (commit message), textinput (search), viewport (diff).

### States and keys
One action table per state; footer and dispatcher both read it.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| status               | status           | j/k, ↑/↓ move · enter: open · s stage · u unstage · c commit · p push · ? help ‖ / search · q / ctrl+c quit |
| status-clean         | status           | p push · / search · ? help · q / ctrl+c quit |
| diff                 | diff             | j/k, ↑/↓ scroll · h/l or ←/→ hunk · s stage hunk · esc back · ? help ‖ ctrl+d / ctrl+u half page |
| commit               | overlay (textarea) | ctrl+s commit · enter newline · esc cancel |
| search               | status (text input) | enter go · esc cancel |
| branches             | overlay          | j/k, ↑/↓ move · enter: checkout · n new · d delete · esc back |

Global: ctrl+c quit

## Decisions
- 2026-10-16 ctrl+s commits because enter is a newline in the message
