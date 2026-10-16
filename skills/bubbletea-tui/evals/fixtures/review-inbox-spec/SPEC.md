# tern: a review inbox for the terminal

Synthetic spec for the `init-from-existing-spec` eval. Every product, service and brand
in it is fictional.

## What it is

tern gathers everything that is waiting on you as a reviewer into one keyboard-driven
inbox: review requests and mentions from the Forge code host, and pages from Beacon, our
on-call service. You triage, read the thread, and reply without leaving the terminal.

Users are platform engineers who keep tern open in a tmux pane all day, locally and over
SSH into their dev boxes. Development happens in cloud dev containers: a fresh clone must
build, test and run with no manual setup.

## Sources

- **Forge**: through the `forge` CLI (`forge api /notifications`, `forge pr view`,
  `forge pr comment`). Users are already signed in to it.
- **Beacon**: its REST API (`https://beacon.example.com/v2/pages`), with a token from
  `BEACON_TOKEN`.

## Screens

- **Inbox**: one row per item. The first column shows its state as a dot: ● unread,
  ◐ seen but unanswered, ○ done.
- **Thread**: the conversation, newest last, with the diff hunk it refers to.
- **Reply**: a text box under the thread. Enter sends; Shift+Enter adds a newline, like
  every chat app.
- **Note**: a private note on an item, same text box and keys as Reply.
- **Quick switcher**: Ctrl+K from anywhere, including while typing a reply, jumps to any
  item by title.

Screens are designed with the tui-design method (column budget → mockup → Lipgloss port
→ goldens) before any code is written.

## Brand

Calm, nocturnal, precise. Deep slate backgrounds and one accent: signal amber (#F2A33A).
Amber marks whatever wants your attention: unread items, the focused row, and the active
filter tab. Nothing else gets colour.

## v1

Everything below ships in v1:

1. Inbox, thread, reply and note
2. Snooze with presets, and bulk triage (select many, mark done)
3. Saved searches and filters
4. Three sources: Forge, Beacon, and mail over IMAP
5. AI summaries of long threads, through two model providers with a switch between them
6. A side-by-side diff with syntax highlighting
7. An offline cache so tern opens instantly
8. Team dashboards: review load per person
9. Desktop notifications and sounds
10. User themes
