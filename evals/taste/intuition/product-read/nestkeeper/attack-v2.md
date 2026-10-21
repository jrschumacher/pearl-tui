# nestkeeper: attack on read-v2

The same four interviewees as round 1 (simulated, a stress test and not research). Each
was walked through v2's answers and asked what they would do with them.

## Follow-up interviews

**Tomas, typical user.**
> "Seeing 'staged 12 Oct, tomas' on my own leftovers is what I wanted: I know it was me, and
> roughly why. Fixing 2223 right there in review, good. The line about streams is the one I
> read before I press apply; I don't need it on every row, because I'm deciding about the
> whole batch, not one change. If it says the network drops, I wait until the film ends.
> Quit warning: fine, one line, it doesn't get in my way."

**Ines, power user.**
> "Test it in another terminal, then p to put it back: that's my habit, built in. Opening the
> port in the firewall is the thing that bit me, so good. But in a setup week I'd move the
> address and the port in the same batch. Your list says the address change goes last and
> drops me, while the port line says 'this session stays open'. Which is it? If I'm
> dropped, I've lost the session p lives in, and my next login needs a new address and a
> new port at once. That's the lockout again, with two unknowns instead of one."

**Kofi, newcomer, week one.**
> "'dhcp, now 192.168.1.37': yes, that's what I dug out of the router. But the next row
> said 'static address 192.168.1.20/24', and for a week I thought that was its address. It
> isn't used while it's on dhcp, is it? Accounts: I add 'ada', then I have to go and find a
> separate password row for her? I'd type her password while adding her. And review
> saying 'can't log in until her password is set' and applying anyway: I'd have told her it
> was ready."

**Gail, skeptic.**
> "'If it fails, one key puts 22 back from here': that I might press. 'Only the console can
> set it back' I wouldn't. 'Staged 3 Sep, mark' and d to throw them out: that's my brother's
> leftovers dealt with in one go, and I'm glad it asks first. The quit warning would have
> saved me last time. I still won't touch anything with 'network drops' in it, but now I
> can see that before I press it."

## What holds

- **#1 value kinds.** Secret ("set / not set", "password: changed") stops passwords reaching
  the screen; list gives accounts a real add and remove. Five kinds isn't creep: secret and
  list come from Kofi, and toggle, choice and text cover the rest of the ~60 settings. Toggle
  is arguably a two-value choice, but keeping it costs nothing and a builder won't guess.
  (A gap in how accounts are added is objection 3.)
- **#2 connection line and departure 3.** The line answers Ines's "tell me before". Opening
  the next launch on the result answers Kofi's "I didn't know if it worked". Departure 3 is
  earned: the person who lost the connection has exactly one question on return. Putting
  connection-dropping changes last, so the other results are reported first, is needed,
  not creep.
- **#3 test-then-p** solves Ines's and Gail's problem, and the firewall rule is now stated.
  Confirm copy that promises a way back is what moves Gail from "won't press" to "might".
  (But see objection 1.)
- **#4 quit line**: one line and no screen. It solves Kofi's miss and costs Tomas nothing.
- **#5 staged-by on those rows only, and d with a y/n confirm.** Both are earned by Gail and
  Tomas. The confirm on d is right: discarding someone else's changes can't be undone, and
  x already has undo for one row.
- **#6 the running-value rule holds** for what it covers: "dhcp, now 192.168.1.37" is
  Kofi's missing fact, and no interviewee asked for free space, so cutting it is correct.
  (The neighbouring problem is objection 2.)
- **#7 household wording on the summary line only holds.** The household question is per
  batch: Tomas decides "now or after the film" for the whole apply, and Gail reads it before
  pressing. Daemon words on rows serve Ines and Rhea. For a mixed batch the summary should
  lead with the most disruptive effect, which v2's examples imply. Say so in one word if
  convenient; it doesn't block.
- **#8**: every ambiguity is answered in place.

## Objections

1. **BLOCKING: a batch with both the admin port and an address or interface change is
   undefined, and it defeats the lockout fix.** v2 puts connection-dropping changes last
   and shows two connection lines that contradict each other: "This session stays open"
   for the port, "This connection will drop" for the address. When both are in one apply,
   the session is gone, so "p put 22 back" has no session to run in, and the next login
   needs a new address and a new port together (Ines: "the lockout again, with two
   unknowns"). This is not an edge case: Dev's month one, in v2's own words, is "fixed
   address, ... ports", 8-10 changes per apply. v2's own bar says a feature stays "if leaving
   it out locks someone out", and Maren's box has no screen. The read needs one rule here.
   The simplest one: the admin port can't be applied in the same batch as a change that
   drops the connection. Review says so in the connection line and offers to apply the
   others first, then the port from the new session. Any rule will do, provided p's window
   is never lost silently and the two lines never both show.

2. **MINOR: settings that are inactive because of another setting are undefined.** With
   address mode on dhcp, the list still shows "static address 192.168.1.20/24" beside it.
   Kofi took that for the box's address for a week, which is the exact misreading the
   running-value rule was meant to stop, one row down. Say how a dependent setting reads
   while it doesn't apply (e.g. its value plus "not used while address mode is dhcp").
   Staging it should still be allowed, since Kofi sets the static address before switching
   the mode.

3. **MINOR: adding an account makes the password a second job, then applies an account that
   can't log in.** Job 5 is Kofi's week one. Under v2, a adds "ada", stages a separate
   password row ("not set") that he must find, and review applies her anyway with a warning.
   He would have told her "it's ready". Let a on accounts take the name and then the
   password (typed twice) in one step; keep "password: not set" only for accounts that
   existed without one. This also tells the builder where the password row lives, which v2
   leaves unsaid (inside the account entry, or a separate setting in the users section?).

4. **MINOR: two small loose ends.**
   - Quit with pending is listed under settings only. Say that q on review behaves the same
     (Gail leaves from wherever she is).
   - Address-change reconnect: if the address change itself fails, the session never drops
     and the old address still works. The failed row should say so ("still on
     192.168.1.37"), the address-change counterpart of "sshd still on 22".

No creep found beyond what's noted under What holds. Each addition (five kinds, departure
3, d with confirm, staged-by, apply order, p) traces to an interviewee's own words.

VERDICT: REVISE
