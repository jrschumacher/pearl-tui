# AGENTS.md

Guidance for coding agents working **on** this repository: improving the skill, its
kit and its tools. If you are an agent *using* the skill to build a TUI, read
`skills/pearl/SKILL.md` instead.

## What this repo is

One agent skill, `skills/pearl/`, published through skills.sh
(`npx skills add jrschumacher/pearl-tui`). It teaches models to design and build terminal
UIs at Crush quality with a per-brand identity. The design laws, the theme contract
(`kit/themes/*.json`) and the frame tools are framework-neutral; Bubble Tea v2 is the one
framework the kit (`kit/theme/bubbletea`, `kit/ui/bubbletea`) and `tuilint code` support
today. Another framework would get its own `kit/theme/<framework>` and
`kit/ui/<framework>` and a harness that writes the same `.ansi` frames (ROADMAP.md).

```
skills/pearl/
  SKILL.md        router: stance, laws, domain DO/DON'T, slop test, command table (keep < 500 lines)
  references/     domain references, cmd-<command>.md and <framework>-<topic>.md, one
                  level deep, loaded on demand
  tools/          Go module `tuitools`: cellshot, tuilint, tuivet, palette, diversity (the build loop),
                  feedback (files an issue here from a user's agent)
  capture/        Go module `tuicap`: runs any app in a PTY from a tape and writes its screens as
                  .ansi frames, a flow and a sequence log, in full or privacy mode (eyes on an
                  app with no harness)
  scripts/        wordmark, logo2cells (Python 3 + Pillow/numpy; brand-asset conversion only)
  kit/            Go module `tuikit`: themes/*.json (the framework-neutral contract),
                  theme/bubbletea/ and ui/bubbletea/ (one framework's engine and components),
                  cmd/showcase (the Bubble Tea reference app + harness)
.claude-plugin/   plugin.json + marketplace.json: the repo root is a Claude Code plugin that
                  bundles skills/pearl (default skills/ scan) and hooks/
hooks/            hooks.json (PostToolUse on Edit|Write|MultiEdit) → tuilint-edit.sh, which
                  builds the tools once and runs `tuilint hook`; test.sh tests it end to end
evals/            evals.json (skill-creator format) + fixtures (each Go fixture its own module)
tools/gallery.sh  regenerates docs/images from the kit
ROADMAP.md        what's next, and the open problems
```

Everything under `skills/pearl/` ships: `npx skills add` copies that directory
whole, with no ignore mechanism, and the plugin bundles it. That is why `evals/` (the
prompts and graded assertions, which a cold agent under evaluation must not be able to
read) lives at the repo root; skill-creator takes the path.

`version` in `.claude-plugin/plugin.json` and `marketplace.json` is the release
(`1.0.0-rc.1` today) and equals SKILL.md's `metadata.version`, which the `feedback` tool
reports. Claude Code pins plugin installs to it: a commit on `main` reaches plugin users
only when a release bumps all three, together, with a `CHANGELOG.md` entry; CI's
plugin job fails unless the three and the top `## <version>` heading agree. Tags are
`v<version>`, made after the release PR merges.

## Commands

```sh
cd skills/pearl/kit && go vet ./... && go test ./...              # footprint, contrast, behaviour
cd skills/pearl/kit && go run ./cmd/showcase                       # the reference app, live
cd skills/pearl/tools && go vet ./... && go test ./...            # the tools, held to Python-era goldens
cd skills/pearl/capture && go vet ./... && go test ./...          # tuicap (needs a pty)
go -C skills/pearl/tools build -o "$PWD/.tuitools/" ./cmd/... && go -C skills/pearl/capture build -o "$PWD/.tuitools/" ./cmd/tuicap && export PATH="$PWD/.tuitools:$PATH"   # per checkout, gitignored; all seven tools, feedback and tuicap included
tuilint code skills/pearl/kit
tuilint spec skills/pearl/tools/internal/speclint/testdata/template-example.md   # init's template, filled in (CI: covered by the tools tests on the template)
diversity skills/pearl/kit/themes/*.json
for t in skills/pearl/kit/themes/*.json; do palette check "$t"; done
skills-ref validate skills/pearl                                   # pip install "git+https://github.com/agentskills/agentskills#subdirectory=skills-ref"
tools/gallery.sh                                                   # after any change to what the kit renders
hooks/test.sh                                                      # the plugin's lint hook, end to end
claude plugin validate .                                           # plugin, marketplace and hooks manifests
jq -e '.skill_name=="pearl"' evals/evals.json && jq -r '.evals[].files[]' evals/evals.json | xargs -n1 test -f   # the evals parse and their fixtures exist
```

CI (`.github/workflows/ci.yml`) runs all of these plus a render-and-lint pass over every
theme (and the test themes in `kit/theme/bubbletea/testdata`) in the variants its loop
lists, seven today: dark, `-light`, `-ascii` (linted with `--ascii`), `-profile ansi256`,
`-profile ansi`, `-light -profile ansi` and `NO_COLOR=1`. Light, ASCII, 256 and 16
colours and no colour are where footers, rules, background steps and downsampled roles
break first, so lint the variant you touched before pushing, not only dark. Each render
is held to `.github/tuilint-baseline.json` (`"<theme>-<variant>": warnings`; a missing
entry means 0) through `tuilint frames --max-warnings`, so a new warning class fails CI;
to raise an entry on purpose, justify the warning in the PR and edit the number (today only
moss's `dialog: none` overlays, which have no edge for the linter to find). It also runs
`kit/README.md`'s copy block, as written, in a fresh module, because a kit test that
names a kit path passes here and fails in every project that copied the kit, and then
deletes the showcase's screens as the README says and runs the kept harness tests
against `.github/kit-stub` (two pages, one overlay), because a kit test that names a
showcase state fails the same way; checks that
every tool invocation in SKILL.md and `references/` names a real mode and flag (the
`cmddoc` test in `tools/`), that every `references/cmd-*.md` has a row in SKILL.md's
command table, and that `evals/evals.json` parses and its fixtures exist; and runs each
Python script once for real on the quarry fixture. `.github/workflows/conventional.yml`
checks the PR title and commit subjects (§ Commits).

## How the skill improves (the loop)

The skill gets better from observed misses, not from adding prose.

1. **A miss** comes from dogfooding (someone built an app with the skill and something
   was wrong) or from a cold eval run. File it with an issue form (`bug`, `performance`,
   `eval-suggestion`, `feature` in `.github/ISSUE_TEMPLATE/`); a user's agent files the
   same forms with the `feedback` tool (`references/cmd-feedback.md`), so its reports
   arrive labelled `feedback` with the version and the miss class filled in. Keep
   `tools/internal/feedback`'s `Kinds` and the forms in step: a test holds them together.
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
  fails on hex [C-HEX] and on a glyph (`·`, `▏`, `→`) [C-GLYPH] outside `theme/bubbletea/`
  and `ui/bubbletea/canvas.go`'s twin table; a new glyph is a `theme.Glyphs` field with
  an ASCII twin.
- **Libraries go through the kit's theme.** Huh forms are built with `ui.NewForm` and
  spinners with `ui.HuhSpinner` / `ui.BubblesSpinner`; `tuilint code` fails a bare
  `huh.NewForm` or an unstyled spinner [C-HUHTHEME]. Huh v2.0.3's quirks and the kit's
  workarounds are in `references/bubbletea-architecture.md`; each has a test, so don't
  undo one.
- **Pinned versions.** `charm.land/bubbletea/v2 v2.1.0`, `lipgloss/v2 v2.0.6`,
  `bubbles/v2 v2.2.1`, `huh/v2 v2.0.3`, Ultraviolet `v0.0.0-20261008173134-6b8d4baf91b4`.
  To bump: update `kit/go.mod`, run everything above, and update
  `references/bubbletea-architecture.md`, whose API claims are verified against these
  versions, the `go get` line in `kit/README.md`'s copy block, and the stack line in
  `references/cmd-init.md`'s TUI.md template. `capture/go.mod` pins Ultraviolet to the
  kit's commit too, so its emulator's cells are the kit canvas's: bump them together.
- **Fictional brands only** in `kit/themes/`. No real companies' marks or palettes, apart
  from the clearly labelled Charmtone study (`charm.json`).
- **SKILL.md frontmatter** stays within the Agent Skills spec: `name` (must equal the
  directory name), `description` (≤ 1024 chars), `license`, `compatibility`, `metadata`.
- **References stay one level deep** and are named by domain, `cmd-<command>.md`, or
  `<framework>-<topic>.md` when the whole file is one framework's
  (`bubbletea-architecture.md`); a mixed reference opens with a *Bubble Tea:* line naming
  its framework sections. A new command needs a row in SKILL.md's command table.
- **Tools** (`tools/`) are Go and depend only on the kit, `charmbracelet/x/ansi` and
  `golang.org/x/{image,text,tools}`. Each command's package doc is its `-h` text. Their tests
  hold them to goldens taken from the Python originals: change a golden only on purpose,
  and say why in the commit. Anything the agent runs on every iteration belongs here.
- **Capture** (`capture/`, `tuicap`) is its own module because it needs a PTY and a
  terminal emulator: it depends only on Charm's `x/vt`, `x/xpty` and `x/ansi`,
  Ultraviolet at the kit's commit, and `uniseg`. Its frames must stay cell-identical
  with the harness's for the same state (confirm, help, setup); a change that breaks
  that says why. Its privacy level never writes a letter outside the allowlist.
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
  | `kit` | the Go kit: `theme/<framework>/`, `ui/<framework>/`, `cmd/showcase` |
  | `themes` | `kit/themes/*.json` |
  | `tools` | the Go tools in `tools/` |
  | `scripts` | the Python brand-asset converters |
  | `evals` | `evals/` and its fixtures |
  | `plugin` | the Claude Code plugin: `hooks/` and `.claude-plugin/` |

  Work that spans the repo takes no scope: `ci: …`, `docs: …`, `chore: …`. Dependency
  bumps are `build(<module>): …`, e.g. `build(kit): bump lipgloss to v2.0.7`.
- **`!`** for a change that breaks something agents or users rely on: a theme contract
  key, a kit API, a tool's flags or output, a command's name.
- Lowercase description, imperative, no full stop, header ≤ 100 chars (GitHub's squash
  suffix ` (#N)` doesn't count).

CI (`.github/workflows/conventional.yml`) checks the PR title, since squash-merge makes it
the commit on `main`, and every non-merge commit on a PR and pushed to `main`. `fixup!`
and `squash!` commits fail: squash them before merge. The allowed types and scopes are
the workflow's `env`; a new one goes there and in the table above, in the same PR.

## Pull requests

Squash-merge. The PR title becomes the commit on `main`, so it follows the same format.

Small and single-purpose. Say which miss or roadmap item it addresses, show before/after
contact sheets for anything visual, and include the test or lint rule that keeps it
fixed.
