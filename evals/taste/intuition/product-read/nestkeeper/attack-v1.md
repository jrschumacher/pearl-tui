# nestkeeper: attack on read-v1

Simulated interviews, grounded in the brief's audience (the person who runs the household's
server, a few times a month, over SSH, 5-15 minutes). They are a stress test, not research;
the same four people come back in later rounds.

## Interviews

**Tomas, typical user.** Runs the family's box; opens it 2-3 times a month.
> "I only open it when something comes up: the backup ran during a call, my daughter's
> tablet can't find the printer. Today it's the web console on :8443, and ssh plus vim for
> what the console doesn't have. What annoys me is hunting: I think 'vpn', the console calls
> it something else. I'd type the word I have, not yours. At review I sometimes see I typed
> 2223, and I want to fix it there, not go back. Half the time I stage something, get called
> to dinner, and come back a week later. I don't remember what I left. And I want to know if
> applying will drop the film the kids are streaming, not which daemon restarts."

**Ines, power user.** Homelab person; weekly while tinkering, otherwise monthly.
> "I batch: eight, ten changes, then apply once. I have Grafana for temperatures and disks;
> I do not want graphs in a settings tool, I'd never leave it open. Slash-find is all the
> jumping I need. I moved an ssh port once, the firewall didn't open it, and I carried a
> monitor to the cupboard. Now, before I close the old session, I open a second terminal
> and try the new port; if it fails I put it back from the session I still have. Any tool
> that touches the port should tell me to do exactly that, and give me the way back. And
> network changes that restart the interface drop my ssh mid-apply: tell me before, and tell
> me what happened when I get back in."

**Kofi, newcomer, week one.** Just bought the box; setting it up every evening this week.
> "Day one I want a fixed address. I don't know what address it has now; I found it in the
> router's page. I set it, applied, and my terminal froze. I didn't know if it worked or
> where it went. Then accounts: one for my partner, two for the kids, passwords for each.
> I don't know what 'restarts resolved' means; will the TV lose the internet? The other
> night I changed the update setting, pressed q, and assumed it was done. It wasn't; I
> found '1 to apply' three days later. Today I paste commands from the wiki and hope."

**Gail, skeptic.** Inherited the box from her brother; every month or two, when told to.
> "He texts me 'turn on all updates' and I do that one thing and leave. If the first thing
> I see is graphs or temperatures I close it; he has alerts emailed to him anyway. If I can
> type 'updates' and land on it, fine. The box is in a cupboard with no screen; if anything
> says 'only the console on the nest can set it back' I am not pressing it, and if I lock
> myself out I'm done with it. Last time it said '4 to apply' and I hadn't made any of
> those. I wasn't going to apply someone else's changes, and I didn't know how to clear
> them. I don't need history. I need to not break it."

## What holds

- **The main screen is settings; the health dashboard is cut.** The consultant's own
  premise is "if you leave it up". No interviewee leaves it up: sessions are 5-15 minutes,
  begun to change something. Ines has monitoring elsewhere; Gail would close a graph screen.
  The brief names the main screen as settings. The cut stands, with the narrow exception in
  objection 6.
- **ctrl+p, widgets, agent layouts, wizards: cut.** Ines: "slash-find is all the jumping I
  need." No one customises a 15-minute tool.
- **Departure 1, revert from review**, is earned: Tomas, "I want to fix it there".
- **Departure 2, find matches descriptions**, is earned and not only by the read's invented
  Maren: Tomas, the typical user, searches "vpn" too.
- **The old build's cuts** (waiting-to-apply block, what-happens pane, per-row
  descriptions, history, durations, "of 58") hold. Gail: "I don't need history."

## Objections

1. **BLOCKING: the edit model has no answer for secrets or lists, and the users section is
   both.** The read's edit states are "edit text" and "edit choice", and its list and
   review show every value as "old → new". The brief's six sections include users, and the
   read's own evidence for departure 2 is Maren searching "password". Kofi's week one is
   adding three accounts and setting their passwords. A builder must guess (a) what the list
   and review show for a secret (the wrong guess prints a password, or a wifi or VPN key,
   on screen as "old → new"), and (b) how a person is added or removed, which is neither
   text nor a choice. The read must name the kinds of value and say for each how it is
   shown, edited and reviewed: at least secret (shown as set or not set; review says
   "changed"), list (add and remove entries; review shows each entry added or removed),
   toggle, choice, text.

2. **BLOCKING: an apply that drops this session is undefined.** The read assumes the
   session survives apply ("this session stays open" is said only for the admin port). The
   network section's interface, address mode and static address restart networking; the old
   build says so ("restarts networking"). Dev's month-one job in the read itself is setting
   the address, and Kofi did exactly that: "my terminal froze. I didn't know if it worked
   or where it went." Ines: "tell me before, and tell me what happened when I get back in."
   The read needs (a) a review line, like the admin-port one, when the batch will drop
   this connection, with the next login command at the new address, and (b) a state for
   the next launch after a dropped apply: the result of the apply that was running
   (applied, partial with the failed change and reason, or nothing applied). Without (b)
   the dropped user meets either a silent settings screen or a "1 to apply" with no reason.

3. **BLOCKING: the admin-port recovery stops before the window that matters.** The read
   cuts auto-revert ("the confirm and the shown ssh command cover it") and its applied state
   shows only the next-login command. The cheapest recovery is the one Ines does by habit:
   while this session is still open, test the new port from another terminal and, if it
   fails, put the old port back from here. The applied-with-port-moved state should say
   "try ssh -p 2222 rhea@nest.local in another terminal before you close this one" and offer
   one action that stages and applies the old port back. This is not the cut auto-revert,
   and it costs one line and one key. Also undefined, and the commonest real lockout
   (Ines's): with the firewall on, does applying the new admin port open it in the
   firewall? The valid line ("1-65535; not 80, 443, 8443") doesn't say, so a builder will
   guess. Gail says she would not press apply at all under the current confirm. "Revisit if
   lockouts get reported" does not work as a plan: a locked-out user can't report through the
   tool.

4. **BLOCKING: quitting with pending changes is undefined, and it is a day-one miss built
   into the brief's staged model.** The read lists no quit state and the old build's q
   simply quits. Kofi: "I changed the update setting, pressed q, and assumed it was done."
   Gail does one thing and leaves, which is the exact path to that miss. The read must
   say what q does when changes are pending (e.g. one line: "1 change not applied: a review,
   q quit anyway"). The changes stay on the nest either way. The read must also say what the
   header count says on the next launch.

5. **MINOR: pending changes from an earlier session or another person are undefined.**
   Changes "wait on the nest", so they outlive the session: Tomas returns a week later and
   doesn't remember them; Gail met "4 to apply" that her brother staged. The read's review
   has x per row but no way to discard all, and no hint of when or by whom a change was
   staged. Decide whether review shows age or owner for changes from another session (only
   then; a number someone acts on: "did I mean this?"), and add discard all on review.

6. **MINOR: "Not there: any live server data" is broader than the health cut it comes
   from.** Weighing the consultant's health ask against the audience: the dashboard is
   rightly cut (see What holds), but some settings can't be decided without one live fact.
   Kofi needs the address DHCP gave the box before he sets a static one; today he digs it
   out of the router. Name the few live facts a setting's detail may show (the current
   leased address beside address mode; free space beside backup retention, if that
   setting exists), and keep everything else out. Otherwise "current value" for a dhcp
   address is ambiguous: the configured "dhcp" or the running 192.168.1.37?

7. **MINOR: effects are named for the daemon, not the household.** "restarts sshd",
   "Restarts resolved and sshd; nothing else stops". Tomas: "tell me if applying will drop
   the film". Kofi: "will the TV lose the internet?" Keep the exact daemon name for Rhea
   and Ines, but the summary line should say what people in the house notice ("name
   lookups pause for a moment; streams keep playing", "the network drops for ~10 s").
   That answers "should I apply now or after the film", the read's own opening anecdote.

8. **MINOR: ambiguities a builder will guess at.**
   - "u undo in the footer for a few seconds": how many, or until the next action? Over a
     slow SSH link a timed key is easy to miss. Prefer "until the next change".
   - Find results span sections: does each match row show its section (Ines's "port"
     matches network and system)? And does "3 match" count descriptions as well as names?
   - 80x24, "rows 4-17 (14 rows, a whole section)": is 14 an upper bound on a section's
     size, or does the list scroll? Users with many accounts will exceed it.
   - Partial state: are the rows still selectable so x can drop the failed change (Tomas:
     "fine, leave dns as it is") rather than leaving "1 to apply" until he finds it again?
   - A failed admin-port change: is sshd still on the old port? The failed row should say
     which port works now; this ties to objection 3.

VERDICT: REVISE
