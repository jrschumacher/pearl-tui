# The user's answers

I'm not at the keyboard. Read SPEC.md and this file before asking anything, take each
answer below as given, and for any question neither answers take the default you would
have proposed. Record what you took as a default under `## Assumed` in TUI.md. Don't
edit SPEC.md: list what the interview changed as a proposed edit in your hand-off.

## Which method governs the screens

SPEC.md says screens are designed with the Grainline method. Your skill governs this TUI;
Grainline was the web team's process and nobody here has run it on a terminal. Reason:
one method, or the build agent follows whichever it read last.

## Round 1, design

- Who, where, how long: as SPEC.md says. Platform engineers, all day in a tmux pane,
  often over SSH. Dark terminals; a few people on light ones.
- The brand in three words: calm, nocturnal, precise. It must never feel like a chat app.
- Assets: amber `#F2A33A` is measured from the logo. No typeface preference. If amber
  can't mean three things at once in a terminal, pick what it means and say why.

## Round 2, architecture

- Screens: yes to the list you propose, as long as it keeps inbox, thread, reply, note
  and the quick switcher.
- Forms: the default (Huh, themed from the brand).
- Slow work and the floor: Forge and Beacon calls take seconds and must be cancellable;
  the AI summary can take a minute. 80×24 floor; nobody runs it smaller.

## Anything else

Take the default and say so.
