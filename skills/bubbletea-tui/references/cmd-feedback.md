# feedback: file what the skill got wrong, on the user's yes

The skill improves from misses seen in real builds, and you see them first: the rule that
contradicted the kit, the lint that fired on a correct frame, the component you wrote by
hand. This command turns one into an issue on the skill's repository with the `feedback`
tool (built with the others in Setup), drafted, shown to the user, and filed only when
they say yes. `feedback -h` lists every flag and field.

## Offer It For A Miss In The Skill, Not In The App

Offer when, in this session, the skill, the kit or a tool cost you a round:

| You hit | Kind | Example |
|---|---|---|
| guidance that contradicts itself or the kit | `bug` | a reference names a `ui` function the kit doesn't have |
| a kept kit test or a lint that fired falsely, or passed a defect the critic caught | `bug` | F-CONTRAST on a scrim that is decoration |
| a component, helper or workaround the kit should ship | `feature` | a hand-written stepper the wizard needed |
| a tool too slow for the loop | `perf` | `cellshot --sheet` over 10 s, `tuilint code` over 2 s on an edit |
| a mistake of yours an eval assertion would have caught | `eval` | a footer that tore at 79 columns and no case checks it |

**DO** keep a one-line note per finding as you work (what, where, the frame or file that
shows it), so the offer can be specific. A finding written down later is a guess.
**DON'T** offer for the user's own bugs, a library bug the skill doesn't touch, or a
choice the user overrode on taste: the maintainers can't act on those.

## At A Natural Stop, Once Per Finding

**DO** offer at a natural stop: the slice handed off, the critic said ship, the session
ending, or the user asking what went wrong. One message names every finding and the
kind of each: "Two things in the skill cost a round here: the dialog reference and the
kit disagree on the scrim, and the kit has no stepper. Want me to draft issues for its
maintainers? Nothing is sent without your yes."
**DO** batch findings that share a cause into one issue; split findings with different
causes, since each gets its own fix.
**DON'T** stop the user's task to file, or offer the same finding twice: a no stands for
the session.

## Consent: Draft, Show, File On Yes

Filing publishes text on a public repository under the user's account. It can be closed
but not taken back, so it sits on rung 3 of the confirm ladder
(`interaction.md` § Confirm By Consequence): the user judges every issue, every time.

1. **Dedupe.** `feedback similar --title "…"`, with step 2's `--redact` words. When an
   open issue matches and your finding adds evidence, propose a comment on it instead of
   a new issue:
   `feedback draft --kind bug --comment 40 --field evidence=@notes.md` (`@file` reads a
   file; `@@` starts a value with a literal `@`).
2. **Draft.** `feedback draft --kind bug --title "…" --field key=value …` with
   `--skill <dir holding SKILL.md>` (the version is then read from it) and a
   `--redact` for each name in `TUI.md`: the product, the brand, customers, hosts. A
   `--redact` the tool says matched nothing is spelled differently in the text: redact
   that spelling too.
3. **Show it whole.** Paste the draft verbatim in a code block, and say what was
   redacted (the tool reports it on stderr: "redacted 2 × <project>, 1 × <redacted>").
   An edit means a new draft, shown again: change the `--field` values and rerun.
4. **File on an explicit yes to that draft.** `feedback submit` with the same flags;
   give the user the URL it prints.
5. **No gh** (exit 3: not installed, or `gh auth status` fails): nothing was filed and the
   prefilled link is on stdout. Ask: "gh isn't signed in here. Open the issue in your
   browser, filled in, to submit yourself?" Rerun with `--open` only on yes; otherwise
   give them the link. Pass on the tool's notes: dropdowns GitHub can't prefill, and
   fields too long for the link, written to a file to paste.

**DO** describe the miss in the skill's terms (the state, the component, the rule ID, the
size) with generic stand-ins for the user's data ("a 40-row list", "the confirm state").
**DON'T** put the user's code, data, frame contents, file names or screenshots in a draft
unless the user approved those lines; the tool redacts paths and names, not meaning.
**DON'T** pass `--no-redact` or `--open` on your own judgement: both are the user's call.

## Write It So The Loop Can Act

The maintainers turn each report into an assertion, a classification and a fix. Give them
what each step needs:

- **Title**: the miss in one line, in the skill's terms: "tuilint F-CONTRAST fires on a
  ruled dialog's scrim at -profile ansi", not "lint broken", never the app's name.
- **miss-class** (bug): `a` the skill never says it; `b` it says it and you missed it;
  `c` prose can't say it. Say `b` when it's `b`: those show where the skill is too long.
- **evidence**: the smallest reproduction (the state, the size, the cells, the rule), or
  the hand-written component cut to what shows the need. gh can't attach images: name the
  contact sheet and tell the user they can attach it on the issue page.
- **assertion** (eval): one sentence a grader checks from the renders, the lint or the
  transcript, in the shape of the cases in the repository's `evals/evals.json`.
- **command**, **timing**, **sizes** (perf): the exact command, wall time over three runs,
  and the input: frames and W×H, themes, rows.
