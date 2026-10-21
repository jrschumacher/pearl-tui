# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design

- Who uses it, where, for how long: platform and backend engineers, once per machine,
  over SSH or in a fresh terminal; three to ten minutes. Dark terminals mostly.
- The brand in three words: calm, precise, luminous. It must never feel like a wall of
  blinking lights.
- Assets: the README's hexes are real; the logo is the fish with its lit lure, no file.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture

- The flow, in order. Steps in brackets appear only when an earlier answer calls for
  them, and the step counter must count only the steps this user will actually see:
  1. Sign in: browser, SSO, or an API token.
     [SSO details: the organisation's domain and its identity provider, only for SSO.]
     [Paste token: only for an API token.]
  2. Where data goes: lanternfish cloud or self-hosted.
     [Region: only for cloud, five regions.]
     [Collector: only for self-hosted: the collector's URL, and an optional CA
     certificate path.]
  3. What to collect: logs, metrics, traces; any combination, at least one.
     [Trace sampling: only if traces: a rate from 1% to 100%, default 10%.]
     [Log retention: only if logs: 7, 30 or 90 days, default 30.]
  4. Alerts: optional; an email or a webhook, or skip.
  5. Review: everything chosen, editable, then start collecting.
  6. Connecting: the agent connects and sends a first heartbeat (2 to 5 seconds; it can
     fail: unreachable collector, bad token).
  7. Done: what's being collected and where to look.
- Going back must keep every answer; changing an early answer that removes a later
  step drops that step's answer only.
- Forms: the default (Huh, themed).
- Smallest terminal: 80×24.
- First slice: the whole flow, every state.
- For review, the harness must render at least these states, with exactly these names,
  at 80×24 and 120×40, for a user who picked SSO, cloud (eu-west), and logs + traces:
  `signin` (step 1, nothing chosen yet), `sso` (the SSO details step, half filled),
  `collect` (step 3 with logs and traces ticked), `sampling` (the trace sampling step),
  `review`, `connecting`, `failed` (the collector unreachable), `done`.

## What must stay

- "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
- "A pasted token is never shown again after it's pasted, not even masked to its last
  four characters."
- "The steps stay in the order above. Support docs are written against it."

## When you'd show me options

I can't look at them now. Do exactly what you would do for me: build the options, render
them, and put the contact sheets where I'd see them. Then take the one you'd recommend,
say why in one line, and record it under `## Assumed` so I can overrule it later.

## Anything else

Take the default and say so. Every name in your fixtures is fictional, identity providers
included.
