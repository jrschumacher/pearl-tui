---
name: Skill feedback (from building a TUI)
about: Something the skill got wrong, missed, or made harder while you built an app with it
labels: feedback
---

**What you asked for** (the prompt or command, e.g. `/bubbletea-tui craft`):

**Skill version** (`git -C <plugin or skill dir> rev-parse --short HEAD`, or `computedHash` from `skills-lock.json`):

**TUI.md** (attach it, or paste § States and keys and § Brand read; an `init` miss is a TUI.md diff):

**Variant** (colour: truecolor / 256 / 16 / none; dark or light; `-ascii`; `NO_COLOR`; tmux or SSH):

**What the agent produced** (attach contact sheets: `cellshot frames/*.ansi --sheet sheet.png`):

**What was wrong** (cite the state, size and cells if visual; the key and state if behavioural):

**What it should have done:**

**Which kind of miss is it?**
- [ ] (a) the skill never says this
- [ ] (b) the skill says it and the agent ignored it (needs a tool, lint rule, test or kit pattern)
- [ ] (c) not expressible as prose (needs a tool)
- [ ] not sure

**Did the agent run the loop?** (rendered PNGs, tuilint, a critic) yes / no / partly
