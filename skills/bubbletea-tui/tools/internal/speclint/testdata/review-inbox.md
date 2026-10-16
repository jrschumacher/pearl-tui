# tern TUI

Synthetic: the shape of a TUI.md that init wrote from a spec, with the misses a reviewer
caught by hand. tern and its services are fictional.

## Design

### Product
A review inbox: review requests and mentions from Forge, pages from Beacon. Sessions run
all day in a tmux pane, locally and over SSH.

### Brand read
palette: deep slate neutrals, one accent, signal amber
tone: calm, nocturnal, precise
motif: inbox state dots in the first column, ● unread, ◐ seen, ○ done
remember: amber marks whatever needs you: unread rows, the focused row, the active tour stop

### Theme
themes/tern.json   (palette derive from #F2A33A, measured from the spec)

## Architecture

### States and keys
One action table per state; footer and dispatcher both read it.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| inbox                | inbox            | ↑↓/jk move · enter open · e done · s snooze · x select · E done selected · / filter · ctrl+s save search · tab next filter · r refresh · ? help · q quit |
| inbox-empty          | inbox            | r refresh · ? help · q quit |
| thread               | thread           | ↑↓ scroll · r reply · n note · esc back |
| reply                | overlay          | enter send · shift+enter newline · esc cancel |
| note                 | overlay          | enter save · shift+enter newline · esc cancel |
| ask                  | overlay          | enter ask · ctrl+enter newline · esc cancel · esc discard |

Global: ctrl+k switch · ctrl+c quit
