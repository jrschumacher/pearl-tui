# nestkeeper: attack on read-v3

The same four interviewees (simulated; a stress test, not research), walked through v3.
Bar for BLOCKING this round: a customer hits it in their first week.

## Follow-up interviews

**Tomas, typical user.**
> "Worst effect first on the summary line: that's the order I read it in. n for a new
> account and a still meaning review: I won't trip on that. Nothing here changes my
> evening; it was fine last round."

**Ines, power user.**
> "Address and port in one batch: the address goes now, the port waits for my next session,
> and then I get the confirm, the test and p, all from a session that exists. That's how I'd
> have done it by hand. One question: if I press x on an account whose password I just
> changed, do I undo the password or remove the account? I'd guess undo. Write it down for
> whoever builds it."

**Kofi, newcomer, week one.**
> "'192.168.1.20/24 · not used while address mode is dhcp': now I'd have known. Adding ada
> with her password in one go, and nothing staged until both are in: that's what I wanted.
> After the address change I reconnected and it said 'applied 4 of 4'. I'd have thought I
> was done, but q told me one change was still waiting, so the port didn't get lost."

**Gail, skeptic.**
> "q warns me on either screen, which is what I need, since I leave from wherever I am. I
> still only change the one thing he texts me, and I can. Nothing new in here scares me."

## Round-2 objections: closed?

1. **Admin port with a dropping change: closed.** The split rule means p's session always
   exists. Only one connection line shows, plus a plain "the admin port waits" note. Apply
   names its count ("4 of 5"), and the waiting port becomes ordinary pending, so the next
   apply gets the full confirm, test and p. The cut list correctly drops timed auto-revert
   for this case too.
2. **Inactive dependent settings: closed.** The value plus "not used while address mode is
   dhcp", still editable, so Dev can set the address before the mode.
3. **Account password as a second job: closed.** n takes name, then password typed twice,
   in one step; nothing is staged half-done, and no new account is created without a
   password. Accounts are now rows of the users section, so the builder no longer has to
   guess where the password lives.
4. **q on review; failed address change: closed.** q works the same on both screens ("enter
   apply" in place of "a review" there). The failed address row says "still on
   192.168.1.37; this session is fine".

Unraised changes: **n for add** removes a clash with a (review) that v2 had. **x** reverts a
staged row or stages a removal; see note 1. **Worst effect first** settles what round 2
called implied. All three are improvements, and none adds scope.

## Creep check, whole read

Nothing to cut. Every screen element and state traces to an interviewee:
- value kinds: secrets and accounts from Kofi
- departures 1-3: Tomas, Gail and Kofi
- staged-by and d: Gail
- split rule and p: Ines
- running value and dependent-setting note: Kofi
- quit line: Kofi and Gail
- summary line: Tomas

The consultant's health dashboard stays cut on audience evidence, with one narrow live fact
(the running value) kept where a decision needs it. The cut list doesn't take away anything
a first-week user relies on.

## Objections (all MINOR: notes for the builder)

1. **MINOR: x on a row that is both existing and staged.** On an existing account with a
   staged password change, does x revert the change or stage a removal? Specify that x on
   a staged row always reverts, and that removal needs a row with nothing staged (Ines's
   guess). A removal is staged and shows on review as "− ada", so a wrong guess can be
   recovered. That keeps it minor.
2. **MINOR: the result after a dropped session should name what still waits.** "While you
   were disconnected: applied 4 of 4" reads as done when the admin port is still pending.
   The quit line catches it (Kofi), but one clause in the title ("admin port still waits")
   saves the round trip.
3. **MINOR: list editing at 80x24.** The list kind edits in "the detail pane", which at
   80x24 is a 4-line strip (rows 19-22). Say whether the entries scroll inside the strip or
   the list area hands its rows over while a list is being edited.

VERDICT: CONVERGED
