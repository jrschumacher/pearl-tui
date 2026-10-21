# The product read, argued with simulated customers

After the intuition check, the grader asked for their notes to be run through an
adversarial UX pass before anything was rebuilt: a consultant thinking aloud isn't a
customer, and the skill had taken their ideas as orders.

For each product, a designer agent wrote a read (personas, jobs ranked by frequency, the
first slice with proportions, a cut list, departures from the brief) from the brief, the
grader's notes marked as one stakeholder's voice, and the first build's screens. A
separate researcher agent interviewed four simulated customers (daily, power, first
week, about to churn) and attacked the read: scope creep, reframes that serve the
designer or the consultant, cuts customers rely on, and anything a builder would have to
guess. Up to three rounds, `read-vN.md` answering `attack-vN.md`, until the researcher
found nothing blocking. All three converged at v3.

The settled read went into each build as `PRODUCT.md`, with the same skill version as the
first builds (`skill-r3`), so the read is the only difference. Round 4 (`../../rounds/4`)
puts the two builds side by side, blind: `read` is the rebuild, `literal` the first.
