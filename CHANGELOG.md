# Changelog

What changed in the skill, its kit and its tools, by release. The commit history on
`main` (Conventional Commits) has every change; this file has what a user of the skill
needs to know when updating. Versions follow SKILL.md's `metadata.version`, which the
plugin manifests repeat.

## 1.0.0-rc.2

The skill now plans each screen from the product's job and data, argues the plan with
simulated customers, stops for your taste choices with rendered options, and hands off
work orders a fresh session can run. Every change came from blind, human-graded rounds
(`evals/taste/`).

- **Hold a screen to your reference.** A reference you supply has a slot in `TUI.md`
  (§ Reference: what is exact, what is inspiration, the deviations you approved) and
  travels into the critic's packet; restyle and critique compare real renders at
  80×24 against it, and fidelity is its own MATCH / DEVIATES verdict, never a SHIP
  score (#89).
- **A journey pass per screen.** Each screen block starts with `job:`, `data:` and
  `carry:` lines, and the new `references/flows.md`, loaded per screen, says where each
  kind of data goes: live beside its list, background status as a glyph with a help key,
  money as a receipt, the product's purpose kept in view as a strip. Sheets for
  monitoring, checkout, media, browse and configure; a wizard keeps its whole journey in
  view (#90).
- **Confirm by intent.** Focus starts on the act a flow exists for (a purchase) and on
  the safe button where an act destroys or replaces what runs; no button is a bare
  `Cancel` (#90).
- **Panes that scroll.** `ui.Pane` scrolls both ways with header rows that stay put;
  the wheel scrolls the pane under the pointer and never moves the selection; `ui.Hit`
  and `ui.WheelGate` route and coalesce mouse events (#90).
- **Read the product before the screens.** `shape` writes `customers:` and `jobs:`
  (ranked by how often) into § Product and the cut list into § Decisions, then has
  simulated customers attack the read for up to three rounds: a stakeholder's idea is
  one voice, an unclear value gets a label, never a cut (#90).
- **Pick points and work orders.** The skill stops only for your taste (the brand read,
  the main screen's move and signature, a splash style, a layout two arrangements pass)
  with two to four rendered options, and records your reason as a rule. `init` and
  `shape` end with `## Work orders` in `TUI.md`: one prompt per next step, quoting what
  you said must stay, that a fresh session or another model can run (#91).
- **Kit fixes** (#92): buttons are marked by a fill where colour shows (brackets only
  without colour); toasts draw every kind in its own role; `muted-upper` titles
  uppercase; gradients stay off error's red at 16 colours; `ui.Settle` delivers slow
  commands under load; a zero-width middle table column stretches; the popover's
  scrollbar reads at 16 colours on light; lists and tables measure a wide cursor;
  `tuilint spec` reads no spinner frame as a key.

### Updating

Nothing breaks. A project that copied the kit before this release keeps working. To
take the kit fixes, copy the changed `.go` files from `kit/ui/bubbletea/` and
`kit/theme/bubbletea/` into your `internal/ui` and `internal/theme` file by file
(`cp -r` into an existing directory nests it), and keep your own theme file. Existing `TUI.md` files lint as before; `## Work orders`, `customers:` and
`jobs:` are new sections the next `init` or `shape` fills in.

## 1.0.0-rc.1

The first release candidate. Pearl is an agent skill for designing and building
terminal UIs that look as considered as Charm's Crush and unmistakably like the
product's own brand. Its laws, brand system and frame tools are framework-neutral;
its first and only framework today is Bubble Tea v2.

- **The skill**: a stance, ten laws, domain references (medium, idiom, colour, layout,
  components, interaction, motion, brand, cell art, verify, anti-patterns) and twelve
  commands: `init` interviews the user and writes the design and the architecture into
  `TUI.md`; `brand`, `shape`, `craft`, `critique`, `audit`, `polish`, `harden`,
  `animate`, `art` and `restyle` build on it; `feedback` drafts an issue on the skill's
  own repository, filed only on the user's yes.
- **A brand system**: one theme file per product (roles, glyphs with ASCII twins,
  expression axes, an optional status ladder), derived from the brand's real colours,
  type and logo, held to contrast in dark and light at truecolour, 256 and 16 colours,
  and to distance from the default.
- **The Bubble Tea kit** (`kit/`, module `tuikit`): a compiling theme engine and
  components (canvas, header, tabs, list, table, filter, dialogs with a scrim, toasts
  with undo and grace windows, Huh forms themed from the brand, a wizard stepper, chat
  and tool-call rows, spinners on one frame clock), eight themes, and a reference app
  with a harness whose tests a copied app keeps: footprint at five sizes, contrast,
  one focus, honest help, and every advertised key handled.
- **Eyes**: Go tools built in one step. `cellshot` renders ANSI to PNG, GIF and contact
  sheets; `tuilint` lints frames, Bubble Tea code and `TUI.md`'s keys; `tuivet` runs
  the code rules under `go vet`; `palette` derives and checks a theme; `diversity`
  measures themes against each other and the default; `feedback` files a miss.
- **A Claude Code plugin**: the same skill plus a hook that runs `tuilint code` after
  every Go edit in a project built with it and hands blocking findings back to Claude.

### Breaking

- The skill is renamed from `bubbletea-tui` to `pearl`, and the repository from
  `jrschumacher/skill-bubbletea-tui` to `jrschumacher/pearl-tui`: install with
  `npx skills add jrschumacher/pearl-tui` or
  `/plugin marketplace add jrschumacher/pearl-tui` and
  `/plugin install pearl@jrschumacher`; commands are `/pearl <command>`; the skill
  directory is `skills/pearl` (`.claude/skills/pearl` once installed). Remove the old
  install first, or the skill loads twice and the old copy stops updating:
  - skills.sh: delete `.claude/skills/bubbletea-tui` (and `.agents/skills/bubbletea-tui`
    when it is a link there, or `~/.claude/skills/bubbletea-tui`) and the
    `bubbletea-tui` entry in `skills-lock.json`, then
    `npx skills add jrschumacher/pearl-tui`;
  - plugin: `/plugin uninstall bubbletea-tui@jrschumacher`, then
    `/plugin marketplace remove jrschumacher`,
    `/plugin marketplace add jrschumacher/pearl-tui` and
    `/plugin install pearl@jrschumacher`.
- The kit's packages move under their framework: `kit/theme` is `kit/theme/bubbletea`
  and `kit/ui` is `kit/ui/bubbletea`, imported as `tuikit/theme/bubbletea` and
  `tuikit/ui/bubbletea`. The package names stay `theme` and `ui`. `kit/themes/*.json`,
  the theme contract, stays where it is. Projects that already copied the kit need no
  change: their imports stay `internal/theme` and `internal/ui`. A new copy uses the
  updated block in `kit/README.md`, which copies the packages flat to those same paths
  and drops `/bubbletea` from the imports. Don't rerun it over an existing copy: `cp -r`
  into existing directories nests them, and the donor line overwrites the brand theme.
- `references/architecture.md` is `references/bubbletea-architecture.md`.
- The plugin manifests carry a `version`, so plugin installs update on a release, not on
  every commit to `main`.
