# AGENTS.md

Guidance for coding agents working **on** this repository: improving the skill, its
kit and its tools. If you are an agent *using* the skill to build a TUI, read
`skills/bubbletea-tui/SKILL.md` instead.

## What this repo is

One agent skill, `skills/bubbletea-tui/`, published through skills.sh
(`npx skills add jrschumacher/skill-bubbletea-tui`). It teaches models to build Bubble
Tea v2 TUIs at Crush quality with a per-brand identity.

```
skills/bubbletea-tui/
  SKILL.md        router: stance, laws, domain DO/DON'T, slop test, command table (keep < 500 lines)
  references/     domain references + cmd-<command>.md, one level deep, loaded on demand
  tools/          Go module `tuitools`: cellshot, tuilint, tuivet, palette, diversity (the build loop)
  scripts/        wordmark, logo2cells (Python 3 + Pillow/numpy; brand-asset conversion only)
  kit/            Go module `tuikit`: theme/, ui/, themes/*.json, cmd/showcase (reference app + harness)
  evals/          evals.json (skill-creator format) + fixtures (each Go fixture its own module)
.claude-plugin/   plugin.json + marketplace.json: the repo root is a Claude Code plugin that
                  bundles skills/bubbletea-tui (default skills/ scan) and hooks/
hooks/            hooks.json (PostToolUse on Edit|Write|MultiEdit) → tuilint-edit.sh, which
                  builds the tools once and runs `tuilint hook`; test.sh tests it end to end
tools/gallery.sh  regenerates docs/images from the kit
ROADMAP.md        what's next, and the open problems
```

Keep `.claude-plugin/plugin.json` `version` equal to SKILL.md `metadata.version` (CI
checks); a set version pins users to it, so bump both with every release. Don't set a
version in `marketplace.json`: `plugin.json` wins anyway.

## Commands

```sh
cd skills/bubbletea-tui/kit && go vet ./... && go test ./...      # footprint, contrast, behaviour
cd skills/bubbletea-tui/kit && go run ./cmd/showcase               # the reference app, live
cd skills/bubbletea-tui/tools && go vet ./... && go test ./...    # the tools, held to Python-era goldens
go -C skills/bubbletea-tui/tools build -o "$HOME/.cache/tuitools/" ./cmd/... && export PATH="$HOME/.cache/tuitools:$PATH"
tuilint code skills/bubbletea-tui/kit
diversity skills/bubbletea-tui/kit/themes/*.json
for t in skills/bubbletea-tui/kit/themes/*.json; do palette check "$t"; done
skills-ref validate skills/bubbletea-tui                           # pip install "git+https://github.com/agentskills/agentskills#subdirectory=skills-ref"
tools/gallery.sh                                                   # after any change to what the kit renders
hooks/test.sh                                                      # the plugin's lint hook, end to end
claude plugin validate .                                           # plugin, marketplace and hooks manifests
```

CI (`.github/workflows/ci.yml`) runs all of these plus a render-and-lint pass over every
theme in four variants: dark, `-light`, `-ascii` and `-profile ansi256`. Light, ASCII and
256 colours are where footers, rules and downsampled roles break first, so lint the
variant you touched before pushing, not only dark. `.github/workflows/conventional.yml`
checks the PR title and commit subjects (§ Commits).

## How the skill improves (the loop)

The skill gets better from observed misses, not from adding prose.

1. **A miss** comes from dogfooding (someone built an app with the skill and something
   was wrong) or from a cold eval run. File it with the skill-feedback issue template.
2. **Write the miss as an assertion** in `evals/evals.json` before changing anything.
3. **Classify it**:
   - (a) the skill never says it → add one line to the right reference
   - (b) the skill says it and the agent ignored it → make it a tool, a lint rule, a
     test in the kit, or a structural pattern the kit forces; then shorten the prose
   - (c) prose can't express it → a tool only
   If most misses are (b), the skill is too long: cut before adding.
4. **Make the kit pass it** (the kit is what agents copy, so the fix must live there).
5. **Edit the skill**, one change per cause.
6. **Verify**: tests, lint, renders, and for visual or behavioural changes the critic
   loop (`references/verify.md`): render, look at the PNGs, run an adversarial critic
   subagent, fix, resume the same critic. The kit went 12 → 18 → 20/24 this way.

## Rules

- **The kit must compile, pass its tests and lint clean at every commit.** A cold agent
  copies it verbatim.
- **Every advertised key must be handled.** Keys live in one action table per state
  (`kit/cmd/showcase/app.go`); `TestHelpIsHandled` enforces it. Don't add a footer
  binding any other way.
- **No literals in screens.** Colours and glyphs come from the theme. `tuilint code`
  fails on hex outside `theme/`.
- **Libraries go through the kit's theme.** Huh forms are built with `ui.NewForm` and
  spinners with `ui.HuhSpinner` / `ui.BubblesSpinner`; `tuilint code` fails a bare
  `huh.NewForm` or an unstyled spinner [C-HUHTHEME]. Huh v2.0.3's quirks and the kit's
  workarounds are in `references/architecture.md`; each has a test, so don't undo one.
- **Pinned versions.** `charm.land/bubbletea/v2 v2.1.0`, `lipgloss/v2 v2.0.6`,
  `bubbles/v2 v2.2.1`, `huh/v2 v2.0.3`, Ultraviolet `v0.0.0-20261008173134-6b8d4baf91b4`.
  To bump: update `kit/go.mod`, run everything above, and update
  `references/architecture.md`, whose API claims are verified against these versions, and
  the stack line in `references/cmd-init.md`'s TUI.md template.
- **Fictional brands only** in `kit/themes/`. No real companies' marks or palettes, apart
  from the clearly labelled Charmtone study (`charm.json`).
- **SKILL.md frontmatter** stays within the Agent Skills spec: `name` (must equal the
  directory name), `description` (≤ 1024 chars), `license`, `compatibility`, `metadata`.
- **References stay one level deep** and are named by domain or `cmd-<command>.md`. A new
  command needs a row in SKILL.md's command table.
- **Tools** (`tools/`) are Go and depend only on the kit, `charmbracelet/x/ansi` and
  `golang.org/x/{image,text,tools}`. Each command's package doc is its `-h` text. Their tests
  hold them to goldens taken from the Python originals: change a golden only on purpose,
  and say why in the commit. Anything the agent runs on every iteration belongs here.
- **Scripts** (`scripts/`) are Python with Pillow and numpy, kept only for converting
  brand assets (wordmark, logo), which run once per brand. Each has a `--help`
  docstring that is its documentation.
- **Writing style** for skill text: verdict headings ("Space And Rules, Not Boxes"), a
  reason with every rule, DO/DON'T pairs, concrete numbers and glyphs, no filler. Say
  what to do, not what to avoid feeling.

## Commits

[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), because the
history is how users of a skill learn what changed in it.

```
type(scope)!: description        e.g.  fix(kit): keep the focus bar on resize
```

- **Types**: `feat` (the skill or kit can do something new), `fix` (a miss fixed), `docs`,
  `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `style`, `revert`.
- **Scopes** (optional; comma-separate several):

  | Scope | Covers |
  |---|---|
  | `skill` | SKILL.md and `references/` |
  | `kit` | the Go kit: `theme/`, `ui/`, `cmd/showcase` |
  | `themes` | `kit/themes/*.json` |
  | `tools` | the Go tools in `tools/` |
  | `scripts` | the Python brand-asset converters |
  | `evals` | `evals/` and its fixtures |
  | `site` | the landing page |

  Work that spans the repo takes no scope: `ci: …`, `docs: …`, `chore: …`. Dependency
  bumps are `build(<module>): …`, e.g. `build(kit): bump lipgloss to v2.0.7`.
- **`!`** for a change that breaks something agents or users rely on: a theme contract
  key, a kit API, a tool's flags or output, a command's name.
- Lowercase description, imperative, no full stop, header ≤ 100 chars.

CI (`.github/workflows/conventional.yml`) checks the PR title, since squash-merge makes it
the commit on `main`, and every non-merge commit on a PR and pushed to `main`. `fixup!`
and `squash!` commits fail: squash them before merge. The allowed types and scopes are
the workflow's `env`; a new one goes there and in the table above, in the same PR.

## Pull requests

Squash-merge. The PR title becomes the commit on `main`, so it follows the same format.

Small and single-purpose. Say which miss or roadmap item it addresses, show before/after
contact sheets for anything visual, and include the test or lint rule that keeps it
fixed.
