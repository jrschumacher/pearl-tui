# teach — capture the design context once, so every later change is consistent

Run this first in a project with no `TUI.md`, or when the user says the product or brand
has changed. It writes `TUI.md` at the project root. Every other command reads it before
doing anything.

## 1. Explore Before Asking

Read what exists before you ask the user anything:

- the README, the CLI's `--help`, existing commands and screens (`grep -r "lipgloss\|bubbletea"`)
- brand assets in the repo: logos (`*.svg *.png`), CSS variables, a style guide, fonts
- the product's own copy: docs, site text, error messages. This is the voice.
- the current terminal output, rendered (`cellshot.py`), if a TUI already exists

## 2. Ask Only What You Couldn't Infer

At most three questions per round, each with your best guess as the default:

1. **Who uses it, where, and how long for?** (field engineers on rugged laptops for hours;
   developers in a split pane for seconds)
2. **The brand in three words, and one thing it must never feel like.**
3. **Brand assets**: primary colours (hex), the logo file, the typeface, any product
   that should be the reference ("feel like Linear's CLI", "like Crush but calmer").

If the user has no brand: propose one. Write two short brand reads that differ in kind
(not in colour), and let them pick. Never fall back to the default theme silently.

## 3. Write TUI.md

```markdown
# <Product> TUI

## Product
<what it does, for whom, the jobs the TUI is for, how long a session lasts>

## Users and terminals
<who; terminal sizes (assume 80×24 floor); light/dark; SSH/tmux; NO_COLOR users>

## Brand read
palette / tone / geometry / density / type / motif / remember   (references/brand.md §1)

## Theme
themes/<brand>.json   (derived with palette.py from: <measured hexes, source>)

## Decisions
- <date> <decision> because <reason>          ← append as the project goes

## Screens
<name> — <one line>; states: <list>            ← kept in sync with the harness states

## Never
<anti-references; things this product must not look or sound like>
```

## 4. Then

Hand off to `brand` (no theme yet) or `shape` (theme exists, screens next). Tell the user
in two lines what you captured and what you assumed.
