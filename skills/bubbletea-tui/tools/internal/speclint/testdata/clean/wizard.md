# Keelson TUI

Read before any UI change. Keelson and its templates are fictional.

## Design

### Product
Keelson scaffolds a new service from a template: pick a template, set its options,
confirm, and watch it create the repository.

### Brand read
palette: oak and brass on dark; brass means "the step you're on"
focus: the ▌ gutter (ascii |)
tone: steady, workmanlike
motif: a step rail, ● done, ◆ current, ○ to do

### Theme
themes/keelson.json   (palette derive from #3B2A1E #C8962E)

## Architecture

### Forms
Huh v2 for the template and options steps, themed with the kit's `ui.NewForm`.

### States and keys
One action table per state; footer and dispatcher both read it. A state that binds a
Global key to its own action overrides it.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| template             | wizard (select)  | ↑↓/jk choose · enter next · esc back |
| options              | wizard (multi-select) | ↑↓/jk move · space toggle · enter next · esc back |
| confirm              | overlay          | ←→ choose · enter create · esc cancel |
| creating             | wizard           | ctrl+c cancel |
| done                 | wizard           | enter open · q quit |
| help                 | overlay          | ? close · esc close |

Global: esc back · ? help · ctrl+c quit

## Decisions
- 2026-10-16 ctrl+c cancels the create and keeps the wizard open because a half-made repo
  needs cleaning up before quitting
