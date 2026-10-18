# Changelog

What changed in the skill, its kit and its tools, by release. The commit history on
`main` (Conventional Commits) has every change; this file has what a user of the skill
needs to know when updating. Versions follow SKILL.md's `metadata.version`, which the
plugin manifests repeat.

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
  `npx skills add jrschumacher/pearl-tui` or `/plugin marketplace add jrschumacher/pearl-tui`
  and `/plugin install pearl@jrschumacher`; commands are `/pearl <command>`; the skill
  directory is `skills/pearl` (`.claude/skills/pearl` once installed).
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
