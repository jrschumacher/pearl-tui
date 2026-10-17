# tern TUI

Synthetic: review-inbox.md after the review. tern and its services are fictional.

## Design

### Product
A review inbox: review requests and mentions from Forge, pages from Beacon. Sessions run
all day in a tmux pane, locally and over SSH.

### Brand read
palette: deep slate neutrals; amber means unread and nothing else
focus: the ▌ gutter (ascii |) in fg on a bgSubtle row, so the cursor never reads as unread
tone: calm, nocturnal, precise
motif: inbox state dots in a 2-cell column, ● unread, ◑ seen, ○ done (ascii * ~ o)
remember: amber is "this needs you", and only that

### Theme
themes/tern.json   (palette derive from #F2A33A, measured from the spec)

## Architecture

### States and keys
One action table per state; footer and dispatcher both read it.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| inbox                | inbox            | ↑↓/jk move · enter open · e done · s snooze · x select · q quit · ? help ‖ E done selected · / filter · ctrl+s save search · tab next filter · r refresh |
| inbox-empty          | inbox            | r refresh · ? help · q quit |
| thread               | thread           | ↑↓ scroll · r reply · n note · esc back |
| reply                | overlay          | enter send · shift+enter/ctrl+j newline · esc cancel |
| note                 | overlay          | enter save · shift+enter/ctrl+j newline · esc cancel |
| ask                  | overlay          | enter ask · ctrl+enter/ctrl+j newline · esc cancel |

Global: ctrl+o switch · ctrl+c quit
