# Roadmap

Where the skill is, what comes next, and the open problems that are bigger than this
repository. Items move from here into issues when someone picks them up.

## Where it is (v0.1)

- The skill: a stance, laws, domain references, 11 commands, and about 40 named anti-patterns.
- A compiling Bubble Tea v2 kit with seven themes. Its tests cover footprint, contrast,
  unstyled text, ASCII, honest help and behaviour.
- Tools: an ANSI renderer (`cellshot`), a slop linter (`tuilint`), a palette deriver,
  a theme-distance lint, and wordmark and logo converters.
- Verified by an adversarial critic over three rounds (12 → 18 → 20/24) on the kit.
- **Not yet verified:** a cold run, where a fresh session builds a real app with only
  the skill. Everything above is the authors' own loop. The next phase is the real test.

## Phase 1: Dogfood (now)

Build real apps with it and turn every miss into a fix.

1. **Build one app you actually need** in a fresh session with only the skill installed.
   Start with `/bubbletea-tui init`, then `brand`, `shape` and `craft`. A good first
   target is a CLI with a brand you already own (the tructl wizard and splash are ideal:
   a known-good reference exists to compare against). Keep that brand's theme in the
   app's repo, not in this one.
2. **Don't help the agent.** If it asks a question, answer only as a real user would. A
   question is itself a finding: the skill was unclear.
3. **Record each miss** with the *Skill feedback* issue template: what you asked, the
   contact sheet, what was wrong (with cells cited), and the miss class (a/b/c, see
   `AGENTS.md`).
4. **Measure**:
   - Did it run the loop unprompted (renders, lint, critic)?
   - How many critic rounds did it take to reach SHIP?
   - Which anti-patterns appeared anyway?
   - Did `TestHelpIsHandled` exist in its app?
5. **Run the evals** in `skills/bubbletea-tui/evals/evals.json` with skill-creator: with
   the skill, without it, and with the previous version, three runs per case. Grade by
   hand first, then check that `tuilint` agrees with your grades. The linter earns trust
   only when it matches a human on apps it has never seen.

Exit criterion: two consecutive cold runs that you'd ship, with the automated checks
agreeing with your grades line for line.

## Phase 2: Tooling we control

- **Accessible mode in the kit.** `--accessible` / `ACCESSIBLE=1`: no alt screen,
  line-by-line prompts (Huh's accessible forms), status as new lines, no repainting. Add
  a golden test of its plain transcript. The guidance is in `interaction.md`; the kit
  doesn't implement it yet.
- **Drive the real binary, not just the model.** The harness renders model states
  in-process, which misses bugs in program options, the real renderer, terminal queries
  and resize timing. Add a Go `tuidrive` to `tools/`: spawn the app in a pty at W×H, send keys, settle,
  capture the screen through a VT emulator (Charm's `x/vt`), and write `.ansi` frames into
  the same `cellshot`/`tuilint` pipeline. Evaluate adopting Microsoft's
  [tui-test](https://github.com/microsoft/tui-test) (Playwright-style API, xterm.js
  emulation, Node) as the heavier option, and Charm's `teatest` for Go-side golden tests.
- **Temporal lint for animation** (`cell-art.md` rule 14) as a script over frame
  sequences.
- **Hooks**: a PostToolUse hook that runs `tuilint code` on edited `.go` files, as
  Impeccable does with its detectors. Agents ignore prose; they don't ignore a hook.
- **Distribution**: a Claude plugin marketplace manifest, and builds for other agents
  (Cursor, Codex, Gemini CLI) once the formats are verified.
- **Theme gallery**: a browsable page of every kit theme, with the same states side by
  side, to pick a starting point for `brand`.
- **Other renderers**: the theme JSON is language-neutral by design. A ratatui (Rust) or
  Textual (Python) reader would test that claim and widen the audience.
- **Design tokens interop**: export roles in the W3C Design Tokens (DTCG) format so a
  brand's web tokens and terminal theme can come from one source.

## Phase 3: Industry gaps (bigger than this repo)

These are problems no one has solved for terminals. The skill can only work around them
today. Each would be worth a write-up, a proposal, or a contribution upstream.

### 1. There is no accessibility standard for terminal UIs

WCAG is written for the web. Parts of it map cleanly to TUIs: contrast (1.4.3), use of
colour (1.4.1), keyboard operability (2.1.1) and visible focus (2.4.7). The skill already
enforces those. The hard parts don't map at all:

- A screen reader sees a grid of characters and a cursor. It gets no roles ("this is a
  list, item 3 of 8 is selected"), no relationships, and no live-region semantics for
  status messages (WCAG 4.1.3).
- Canvas frameworks (Bubble Tea included) repaint regions and move the cursor to update
  spinners, which readers announce as noise. Users report readers crashing as history
  grows ("The text mode lie", xogium.me, 2026).

**What would help:** a written mapping of WCAG 2.2 onto terminals (what applies, what
can't, what's missing), published and offered to Charm and other framework authors.
Upstream, Bubble Tea could gain a first-class accessible renderer mode: linear output, a
cursor policy, no repaint for status. There is precedent for structured signals to the
terminal: OSC 133 semantic prompts, and Bubble Tea v2.1's program-status reporting
(OSC 7501). An "ARIA for terminals" could follow the same route: escape sequences that
annotate a region's role and state, which terminals expose to their platform's
accessibility API.

### 2. There is no Playwright for TUIs, because there are no semantics to locate

Pieces exist. Microsoft's tui-test drives a real terminal with a Playwright-like API.
Charm's teatest gives Go golden tests, VHS scripts recordings, and pexpect is decades
old. What's missing is what makes Playwright good: **locators by meaning**
(`getByRole('button', {name: 'Save'})`). In a terminal you can only find text at
coordinates. The same semantic annotations that would serve screen readers (gap 1)
would give test tools stable locators. Accessibility and testability are one problem
here, as ARIA proved on the web.

### 3. Models can't see terminals

Every model building a TUI designs blind unless someone builds it eyes. This skill's
answer (a harness, an honest renderer and a critic) works, but it is bespoke. A small
MCP server that starts any TUI in a pty and returns a PNG plus a plain-text grid would
give every agent the same ability. Keystrokes in, screen out, at a fixed size. It
pairs naturally with `tuidrive` above.

### 4. No shared ground truth for what terminals render

Glyph widths (emoji, East Asian ambiguous characters, variation selectors),
box-drawing joins, colour profiles and OSC support differ across terminals, fonts and
multiplexers. Every TUI author rediscovers this. The web has caniuse. A small,
test-backed capability matrix for terminals would turn a lot of folklore into data, and
let `tuilint` warn about a glyph by terminal.

### 5. Design tokens stop at the browser

Brands keep tokens for web and mobile. Terminal UIs get a hand-picked palette, which is
how pink-purple defaults spread. A token profile for terminals (roles, the four colour
profiles, glyph vocabulary, an ASCII fallback) would let a brand ship its terminal
identity the way it ships its web identity. The theme contract here is a working draft
of that profile.

## Decisions pending

- **The skill's name** (`bubbletea-tui` today). A memorable name helps on skills.sh.
  Renaming means changing the folder and the frontmatter `name`, together.
- **The license holder's name** in `LICENSE`.
