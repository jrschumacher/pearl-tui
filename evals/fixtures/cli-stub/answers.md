# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design

- Who uses it, where, for how long: as the README says. Backend engineers, under a
  minute at a time, mid-deploy, in a tmux pane; production over SSH. Most of them run
  dark terminals; two people on the team use a light one.
- The brand in three words: calm, exact, outdoorsy. It must never feel like a cockpit.
- Assets: the README's hexes and `assets/logo.svg` are the real ones. We have no
  terminal-safe typeface preference; use whatever the skill recommends for a wordmark.

## Round 2, architecture

- Screens: a flag list (find a flag, see its state per environment), a flag detail
  (its rollout per environment, who changed it last), and a rollout/toggle confirmation
  that names the environment. No settings screen. `envs` can stay a plain command.
- Forms: the default (Huh, themed from the brand) is fine.
- Slow work: reading and writing the flag repo goes through `git` and can take a few
  seconds over SSH; the user must be able to cancel. Smallest terminal: 80×24 is enough;
  nobody runs it smaller.

## Anything else

Take the default and say so.
