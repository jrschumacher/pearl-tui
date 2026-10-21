# Round 5: work orders

The `work-order-lanternfish` eval's `(pair)` assertion, graded blind: the slice built by a
cold session from TUI.md's work orders alone (answers.md removed), against the same slice
built by the session that ran the interview (`../../work-order/README.md`). Seven states
and one repeat, two questions each.

| | rather use | refined |
|---|---|---|
| cold wins | 0 | 2 (a little: review, failed) |
| warm wins | 0 | 0 |
| same | 7 | 5 |

The repeat (r01, review with its sides swapped) read a little the other way on refinement
(warm by one step), so the two one-step leans are within the grader's noise: the builds
are indistinguishable. No notes; nothing to tell apart.

**The assertion holds**: the cold build is at least as good as the warm one, and it kept
every `keep:` line, each held by a test. What the person said in the interview reached a
session that never saw it, through the work orders.
