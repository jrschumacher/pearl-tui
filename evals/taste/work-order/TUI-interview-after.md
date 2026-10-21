# lanternfish TUI

Read before any UI change. Update it when a decision changes.

## Design

### Product
`lanternfish init` is the first-run setup of the lanternfish observability agent: once per
machine, on the machine itself, often over SSH. It connects the agent to an account, says
where the data goes and what to collect, and starts collecting. A session lasts three to
ten minutes and is read carefully: a wrong answer here is missing data later. Product
behaviour: README.md (the only project doc; nothing here restates it).
customers: the platform engineer putting the agent on new hosts (a few machines a week, over SSH; today a config file copied from the team wiki); the backend engineer connecting the one machine their service runs on (once, in a fresh terminal; today they ask the platform team)
jobs: connect this machine and start collecting (every run) · recover from a failed connect without losing an answer (some runs) · know what is being collected and where to look (the end of every run)
slice: the brief's one screen, the setup flow, in the brief's seven steps and their branch steps; nothing added. Ideas left out are `cut:` lines in § Decisions.

### Users and terminals
Platform and backend engineers, once per machine, over SSH or in a fresh terminal (TERM
may be anything, tmux common); 80×24 is the floor and the design size, 120×40 common;
dark mostly, light supported (the same hues on #F3F6F9); NO_COLOR honoured; ACCESSIBLE=1
for screen readers; `lanternfish init --token <t>` in CI has no terminal at all.
colour: truecolor → 256 → 16 → none; under -profile ascii, focus stays visible by the `|` gutter and bold, the current step by `^`, a cursor by `>`, a failure by `x` and its tag word (UNREACHABLE, REJECTED, EXPIRED), the undecided part of the route by `.`, the decided part by `-`, the lit part by `=`
dialogs and overlays: the only overlay is help; where the background step is gone (16 colours, NO_COLOR, ascii), the edge is a ─ rule above and below (dialog: rules), so it holds in every profile

### Brand read
palette: lure blue #6EC1E4 does the pointing (focus, the current step, the light on the route); dusk violet #9C8FD6 is the far end of that light (the route's lit run once connected, the light's trail) and the act's button fill (the kit's secondary: Start collecting, Retry); abyss #0B1420 is the page and every neutral leans toward it; light terminals keep both hues on #F3F6F9 (source: README § Brand; no asset file exists)
tone: calm, precise, quietly warm: short sentences that say what happens and what is kept ("Nothing is sent until you start", "your answers are kept"); exact numbers with units (10%, 30 days, 14:02:41); no exclamation marks, no "Successfully", no "Loading..."
geometry: soft and light: thin ─ rules and a thin ▎ focus bar, ╌ for what is not decided yet, no boxes on the page; the one overlay sits between two rules, and a hairline rule sets the keys apart at the foot
density: airy: one blank row between groups at 80×24, because a once-per-machine task is read, not scanned
type: the wordmark is lowercase `lanternfish`, plain bold, never capitalised or tracked; sentence-case titles and labels; section titles in the pale lure tint (accent), the glow of the light; state words (COLLECTING, UNREACHABLE) as bold words, never filled chips, because calm
motif: the lure, one lit dot ● (ascii *): beside the name in the header, and the light that runs the route; it marks nothing else
move: the journey is a 32-cell column left of the form at every width (40 at 120 columns): every step this user will see, each done one carrying its answer, in place of the kit's step row above the form and its summary strip under it, because precise: every answer stays pinned beside the step that set it, and the form never moves
signature: the route, one row under the header that draws where this machine's data goes (`ingest-07 ── logs · traces ──── lanternfish cloud · eu-west`): ╌ and "destination not set" while undecided, filled in the moment each answer is chosen, a light running it while connecting, lit whole once the first heartbeat arrives, because luminous
skeleton: head bar · status top-row · order sidebar-left · foot keys · verdict none · chart other · chartGlyphs line · turnEnd -
accents: primary (lure blue) = lit: the focus gutter, the current step, the route's light, the lure; secondary (dusk violet) = the light's far end: the lit route, the light's trail, the act's button fill; accent (pale lure) = section titles, and a link underlined (the device link, where to look: tertiary drew them in error's red at 16 colours); tertiary is unused; success (green) = collecting, on the done word only; error (coral red) = failed, on the fail mark and its word; warning = a field to fix (the CA path), with ! beside it; info is unused, because it sits 6° from primary
focus: the ▎ gutter in primary and a bold label; "focus": "primary" (primary is the brand's light, not a state)
glyphs: ✓ done (ascii +) · ▸ current step (ascii ^: the cursor › is >, and the two stand in one frame; @ is the ASCII spark ramp's top) · ○ step to come (ascii o) · ✕ failed (ascii x) · ! a field to fix (ascii !) · ● the lure (ascii *) · ╌ route not decided (ascii .) · ─ route decided (ascii -) · ━ route lit (ascii =) · ▎ focus gutter (ascii |) · › cursor (ascii >) · » prompt (ascii >>) · ■ □ ticked, not ticked (ascii [x] [ ]) · ◌ pending, unused (ascii _: its kit twin . is the open route's)
width-safe: every mark sits in a 2-cell column (mark and a space), since ● ○ ▸ ■ □ are East Asian ambiguous and draw two cells in CJK locales
expressions: header minimal, drawn by the app (the lure and the name left, the machine right, because calm) · wordmark plain · sectionTitle accent (luminous: the titles glow in the lure's pale tint) · selection caret (› in the focus ink) · button pill for the act, the rest quieter · tag bold (calm: a state is a bold word, not a lit chip) · dialog rules · footer rule (a hairline over the keys: the page's one rule style closes it) · density airy · scrim dim · working spinner (arc ◜◠◝◞◡◟ for the sign-in wait; the connecting light is the app's own) · paintBg false (a fresh terminal over SSH keeps its own background) · measure 72

### Theme
themes/lanternfish.json   (palette derive "#6EC1E4" "#9C8FD6" --bg "#0B1420" --light-bg "#F3F6F9", from README § Brand, roles kept as derived; palette check OK; diversity 8 axes from ink, OK against ink and charm; the skeleton shares only head with the kit)

### Reference
none: there is no screen to match and no app to imitate; the brand read is the target
exact: none; inspiration: none
approved: none

### Screens
One screen, the setup flow, drawn as one frame whose form changes per step. The first
block is that frame on its default state; each later block is a step or overlay drawn in
the same frame (header, route, journey, form, footer) unless it says otherwise.

#### signin — the setup frame on step 1; the eye lands on the question, then the journey beside it
job: configure (flows.md § Configure): sign in, the first of the seven steps
data: header: where you are (the product, this machine) · route: the product's purpose, where this machine's data goes, filled as answers land · journey: every step this user will see, done ones with their answer · form: the one question now · footer: keys
carry: the route and the journey stay on every step, so what is decided and what is left are never a screen away
80×24:  header 0,0 80×1 · route 2,2 76×1 · journey 2,4 32×17 (mark 2 | name 18 | answer 12) · form 36,4 42×17 · footer rule 2,22 76×1 · keys 0,23 80×1
120×40: header 0,0 120×1 · route 2,2 116×1 · journey 2,4 40×11 (mark 2 | name 20 | answer 18; its rows are the steps this user will see) · this machine 2,13 40×5 (one row under the journey, so it moves down as steps join) · form 46,4 72×33 (the measure) · footer rule 2,38 116×1 · keys 0,39 120×1
collapse: < 110 cols: the journey goes to 32 cells with answers cut at 12 (the review step shows each whole), "this machine" folds into the header's `setup on ingest-07`, and done's link takes a full-width row under the journey; < 30 rows: "this machine" goes; < 80×24: toosmall
components: the header row drawn by the app (ui.Wordmark beside the lure, t.ProductMark), ui.Section (the form's title with `step n of N` as its info), ui.NewForm (Huh select, multi-select, ui.NewInput), ui.KV (this machine, review), ui.Actions, ui.Tag, ui.Spinner, ui.Footer, ui.DialogScrolled (help), ui.TooSmallText; new: ui.Journey (internal/ui/journey.go: ui.Steps.List draws marks and names but no answer column, and the column takes focus in review-edit) and ui.Route (internal/ui/route.go: the signature, a one-row path no kit component draws)
fixtures: host ingest-07; the longest host `ingest-worker-eu-central-12.tidewater.internal` (the header and route cut it with …, the route keeps the destination whole); org tidewater.example; identity providers Quaylock, Brightgate ID, Moorhen Directory, another SAML 2.0 provider; regions us-east, us-west, eu-west, eu-central, ap-southeast; collector https://collector.tidewater.internal:4318 with CA /etc/ssl/tidewater/collector-ca.pem; alerts platform-alerts@tidewater.example and https://hooks.tidewater.example/lanternfish; first heartbeat 14:02:41; the unreachable endpoint ingest.eu-west.lanternfish.example:443; the harness user picked SSO (Quaylock), cloud eu-west, logs and traces, sampling 10%, retention 30 days, email alerts
step count: the steps this user will see, known so far: a step every path has counts from the start (whichever of Region or Collector follows step 2, shown as "Region or collector" until chosen), and a branch step joins the journey and the count the moment its answer is chosen (SSO, API token, Trace sampling, Log retention), so signin reads 1 of 8 and the harness user ends at 11 of 11
flow: the spec's seven steps in the spec's order, with each branch step only when an earlier answer calls for it, because one person configures one machine they are standing on, the answers are only drafts until review's Start collecting, and a fleet already has its expert path outside the TUI (`--token` in CI); calm asks one question per step and nothing more, and support docs follow this order
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ destination not set  
                                                                                
  ▸ Sign in                         Sign in ────────────────────── step 1 of 8  
  ○ Where data goes                                                             
  ○ Region or collector             ▎ How do you sign in?                       
  ○ What to collect                 ▎ each one works over SSH                   
  ○ Alerts                          ▎ › Browser    a link, on any device        
  ○ Review                          ▎   SSO        your identity provider       
  ○ Connecting                      ▎   API token  paste a token                
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  ↑↓ choose  enter continue                                             ? help  
```

#### sso — the SSO details step, half filled: domain typed, the identity provider being chosen
brand: precise: each field's hint says exactly what to type ("the part after @ in your work email", "the one your organisation signs in with")
80×24 / 120×40: the setup frame; form: domain input, then the provider select
keys: tab moves between the two fields; enter on the domain moves to the provider, on the provider continues
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ destination not set  
                                                                                
  ✓ Sign in           SSO           SSO details ────────────────── step 2 of 9  
  ▸ SSO details                                                                 
  ○ Where data goes                   Organisation domain                       
  ○ Region or collector               the part after @ in your work email       
  ○ What to collect                     tidewater.example                       
  ○ Alerts                                                                      
  ○ Review                          ▎ Identity provider                         
  ○ Connecting                      ▎ the one your organisation signs in with   
  ○ Done                            ▎ › Quaylock                                
                                    ▎   Brightgate ID                           
                                    ▎   Moorhen Directory                       
                                    ▎   another SAML 2.0 provider               
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  ↑↓ choose  enter continue  tab/shift+tab field  esc back         ctrl+g help  
```

#### signin-wait — waiting for approval in a browser (Browser, and SSO after its details)
brand: calm over SSH: the link and a short code on their own rows, "open this on any device", the arc spinner with what it waits for and the seconds, and what to do if nothing happens (`esc back` keeps the choice; the code expires in 5:00)
80×24 / 120×40: the setup frame; form: the link (OSC 8, its whole URL), the code, the wait line
flow: a timeout or a denial becomes signin-failed with Retry, never a silent return

#### signin-failed — the approval timed out or was denied
brand: calm: says which (`The code expired after 5:00` or `Quaylock declined the sign-in for tidewater.example`), that nothing was saved, and offers `Retry` (focused) and `Back` to choose another way

#### token — paste an API token
brand: quietly warm about the secret: the hint says where tokens come from and that this one won't be shown; once pasted the field reads `pasted · hidden` in subtle, never a character of the token, not a mask, not a length
keys: ctrl+u clears it to paste another

#### destination — where data goes: lanternfish cloud or self-hosted
brand: precise: each option's description says where the data ends up (`lanternfish cloud: stored in a region you choose next` · `self-hosted: sent to your own collector`); choosing one writes its name onto the route at once

#### region — one of five cloud regions
brand: precise: codes as the product prints them (eu-west), a muted place name beside each, and a hint that it is where this organisation's data is stored

#### collector — the self-hosted collector's URL and an optional CA certificate path
brand: precise: the URL field's hint shows the expected shape (`https://host:port`); the CA field says it is optional and is checked on this machine when you continue, before anything connects

#### collector-invalid — the CA path doesn't exist or isn't a certificate
brand: precise: the error under the field names the fault (`! No such file on this machine`) under the path it checked, in the warning ink with the warn mark, and focus returns to that field

#### collect — what to collect, with logs and traces ticked
brand: precise: each option names what it gathers on this machine, and a quiet line under the field says ticking traces and logs adds their steps, which appear in the journey as they are ticked
flow: ticking a box adds its step to the journey and the count at once; unticking drops that step and only its answer
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           What to collect ───────────── step 5 of 11  
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         ▎ What should ingest-07 send?               
  ✓ Region            eu-west       ▎ at least one                              
  ▸ What to collect                 ▎   ■ Logs     files, journal               
  ○ Trace sampling                  ▎   □ Metrics  cpu, memory, disk            
  ○ Log retention                   ▎ › ■ Traces   your services' spans         
  ○ Alerts                                                                      
  ○ Review                            traces adds a sampling step and logs a    
  ○ Connecting                        retention step; the list shows both       
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  space tick  ↑↓ move  enter continue  esc back                         ? help  
```

#### collect-invalid — enter with nothing ticked
brand: calm: `Pick at least one to collect` under the field, nothing else moves

#### sampling — the trace sampling rate, 10% by default
brand: precise: the hint says the rate is a percent from 1 to 100, and a muted line under it restates it as a fact that updates as you type (`10% sends about one trace in ten`)
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Trace sampling ────────────── step 6 of 11  
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         ▎ Sampling rate                             
  ✓ Region            eu-west       ▎ percent of traces sent, 1 to 100          
  ✓ What to collect   logs, traces  ▎ » 10                                      
  ▸ Trace sampling                                                              
  ○ Log retention                     10% sends about one trace in ten          
  ○ Alerts                                                                      
  ○ Review                                                                      
  ○ Connecting                                                                  
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  enter continue  esc back                                         ctrl+g help  
```

#### sampling-invalid — a rate outside 1 to 100
brand: calm: `Use a whole number from 1 to 100` under the field; the previous valid rate stays in the journey

#### retention — log retention: 7, 30 or 90 days, 30 by default
brand: precise: the three options as numbers with their unit, 30 marked as the default in subtle

#### alerts-choose — optional alerts: email, webhook, or skip for now
brand: calm: skipping is a first-class option named `Skip for now`, not a failure to answer; each option says where alerts go in two or three words

#### alerts — email chosen: its address on the same step
brand: calm: the choice stays named above the field (`Alerts Email`), the hint says where alerts from ingest-07 go, and esc returns to the choice, the address kept

#### review — everything chosen, editable, then start collecting
brand: quietly warm: the answers as a labelled list read top to bottom, one sentence that nothing is sent until you start, and two buttons named for what they do: `Start collecting` (focused) and `Edit an answer`
flow: the review step is the confirmation of starting, with focus on the act because it is what the flow exists for (interaction.md § Confirm By Consequence: the act a flow exists for starts on the act); `Edit an answer` moves focus to the journey (review-edit); an edit returns here, walking any step the edit adds; no dialog follows Start collecting
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Review ────────────────────── step 9 of 11  
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         Sign in       SSO · Quaylock                
  ✓ Region            eu-west       Organisation  tidewater.example             
  ✓ What to collect   logs, traces  Data goes to  lanternfish cloud · eu-west   
  ✓ Trace sampling    10%           Collect       logs, traces                  
  ✓ Log retention     30 days       Sampling      10% of traces                 
  ✓ Alerts            email         Retention     logs 30 days                  
  ▸ Review                          Alerts        email platform-alerts@tidew…  
  ○ Connecting                                                                  
  ○ Done                            Nothing is sent until you start.            
                                                                                
                                    ▎   Start collecting     Edit an answer     
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  enter start collecting  ←→ choose  esc back                           ? help  
```

#### review-edit — choosing which answer to change
brand: precise: the journey takes the focus gutter on a done step, the review list stays beside it with every answer whole, the buttons give way to `choose an answer in the list to change it`, and `enter edit` opens the step

#### connecting — the agent connects and waits for the first heartbeat (2 to 5 seconds)
brand: luminous, never blinking lights: the one thing that moves is the light running the route; the form says what it waits for, the seconds after 3, and the usual time
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ●──────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Connecting ───────────────── step 10 of 11  
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         Connecting to lanternfish cloud · eu-west   
  ✓ Region            eu-west       waiting for the first heartbeat · 4s        
  ✓ What to collect   logs, traces                                              
  ✓ Trace sampling    10%           usually 2 to 5 seconds                      
  ✓ Log retention     30 days                                                   
  ✓ Alerts            email                                                     
  ✓ Review                                                                      
  ▸ Connecting                                                                  
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  esc cancel                                                            ? help  
```

#### connecting-slow — no heartbeat after 10 seconds
brand: calm: `Still waiting for the first heartbeat 14s` beside the arc, what usually causes it (outbound HTTPS blocked), and that esc cancels with every answer kept

#### failed — the destination is unreachable
brand: calm and exact: a bold word (UNREACHABLE), the endpoint on its own row and the timeout (30 s), the one likely cause and the next step, that nothing started and every answer is kept; the route breaks at ✕ before the destination; `Retry` focused, `Back to review` beside it
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ──────────────✕╌╌╌╌╌╌ lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Connecting ───────────────── step 10 of 11  
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         UNREACHABLE lanternfish cloud · eu-west     
  ✓ Region            eu-west                                                   
  ✓ What to collect   logs, traces  ingest-07 can't reach                       
  ✓ Trace sampling    10%           ingest.eu-west.lanternfish.example:443      
  ✓ Log retention     30 days       (timed out after 30s). Check that this      
  ✓ Alerts            email         machine allows outbound HTTPS, then retry.  
  ✓ Review                          Nothing started; your answers are kept.     
  ✕ Connecting        unreachable                                               
  ○ Done                            ▎   Retry     Back to review                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  enter retry  ←→ choose  esc back to review                            ? help  
```

#### failed-auth — the token or sign-in was rejected at connect
brand: calm and exact: tag REJECTED, `lanternfish cloud eu-west rejected this machine's token`, and `Back to sign-in` focused, which returns here through review with every other answer kept

#### done — collecting; what and where to look
brand: quietly warm: one green bold word (COLLECTING), the first heartbeat's time as proof, the link to this host, and `lanternfish status` named for later; the route is lit whole with the lure at the destination
sketch (exactly 80 wide, real content; the frame as built, from the -golden harness):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ━━ logs · traces ━━━━━━━━━━━━━━━━━━━━● lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Done ─────────────────────── step 11 of 11  
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         COLLECTING logs and traces                  
  ✓ Region            eu-west                                                   
  ✓ What to collect   logs, traces  First heartbeat from ingest-07 at           
  ✓ Trace sampling    10%           14:02:41, in lanternfish cloud · eu-west.   
  ✓ Log retention     30 days                                                   
  ✓ Alerts            email         lanternfish status shows what this agent    
  ✓ Review                          is collecting, any time.                    
  ✓ Connecting        14:02:41                                                  
  ▸ Done                                                                        
                                                                                
  Where to look ──────────────────────────────────────────────────────────────  
                                                                                
  https://eu-west.lanternfish.example/tidewater/hosts/ingest-07                 
                                                                                
                                                                                
                                                                                
  ────────────────────────────────────────────────────────────────────────────  
  enter finish                                                          ? help  
```

#### quit-armed — ctrl+c pressed once after an answer was given
brand: calm: for 4 s the footer offers `ctrl+c quit` and `esc keep going`, with `your answers are not saved` at its right in the warning ink; then the step's keys return; before any answer, ctrl+c quits at once

#### help — every key of the current step, grouped
brand: nothing, because help lists the action tables as written; its edge is the rules dialog (dialog: rules)

#### toosmall — below 80×24
brand: calm: `lanternfish needs 80x24 · this terminal is 72x20`, and q quit

### Motion
hero: connecting → the lure ● with a 3-cell trail (lure blue to dusk violet) runs the route's open run toward the destination, one cell per frame (50 ms), and starts over; at most the 30 cells before the destination; it loops, it never fills, because the wait has no fraction (motion.md: no fake progress)
rows: the sign-in wait's arc spinner ◜◠◝◞◡◟ beside its subject and seconds; idle: nothing moves, and the done frame's lit route is still; built: ui.Route's RouteRunning draws the light from the frame (internal/ui/route.go), frames/app/connecting.gif shows it

### Never
a wall of blinking lights (two things moving at once, colour on every row); pink-purple; emoji; boxes around regions; "Loading...", "Successfully", exclamation marks; a token drawn in any form, masked or cut; a step the user won't see counted in `step n of N`

## Architecture

### Stack
Go 1.26.7; charm.land/bubbletea/v2 v2.1.0, lipgloss/v2 v2.0.6, bubbles/v2 v2.2.1,
huh/v2 v2.0.3; ultraviolet v0.0.0-20261008173134-6b8d4baf91b4.
The kit's theme/bubbletea and ui/bubbletea copied into internal/theme and internal/ui;
the showcase into cmd/lanternfish, its screens replaced by the setup flow.

### Root model and regions
One *model in cmd/lanternfish (where the kit's copy block puts the harness and its tests);
regions are structs drawn into rectangles (x,y w×h at 80×24):

    header   0,0    80×1
    route    2,2    76×1
    journey  2,4    32×18
    form     36,4   42×18
    footer   0,23   80×1

focus: form | journey (review-edit only)        overlays: help (front one gets keys)
flow state: one setupFlow holding every answer (Huh binds pointers to it), the steps this user will see computed from it on every change, the current step and the furthest reached; going back never clears an answer; changing an answer that removes a later step drops only that step's answer.

### Forms
Huh v2 for every step's input (select, multi-select, input, confirm), themed with the kit's
`ui.NewForm` from *theme.Theme. Hand-written: the token field (Huh's password echo draws a
mask, and the token is never drawn: it takes the paste into memory and the field reads
`pasted · hidden`), the review list and buttons, the wait, failed and done bodies (no input:
kit screens, as in the showcase's setup).

### Components
Bubbles: none beyond Huh's. Kit ui: Header, Section, Steps, KV, Help, Footer, Toast (quit-armed), Tag, Buttons/Actions, Dialog (help, rules), TooSmall, Anim (the arc spinner). New: Journey, Route.

### States and keys
One action table per state; footer and dispatcher both read it. The flow loads nothing
into a list, so its empty, loading, error and partial rows are the wait, invalid and
failed states below.
Footer keys are **bold**: main actions first, then the way out and ? help (ctrl+g help where a text box types ?). The rest are help-only.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| signin               | setup (select)   | **↑↓ choose** · **enter continue** · **? help** |
| signin-wait          | setup            | **c copy link** · **esc back** · **? help** |
| signin-failed        | setup            | **enter retry** · **←→ choose** · **esc back** · **? help** · r retry |
| sso                  | setup (text box) | **↑↓ choose** · **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** |
| token                | setup (text box) | **enter continue** · **ctrl+u clear** · **esc back** · **ctrl+g help** · backspace clear |
| destination          | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** |
| region               | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** |
| collector            | setup (text box) | **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** · backspace delete · ←→ move |
| collector-invalid    | setup (text box) | **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** · backspace delete · ←→ move |
| collect              | setup (multi-select) | **space tick** · **↑↓ move** · **enter continue** · **esc back** · **? help** |
| collect-invalid      | setup (multi-select) | **space tick** · **↑↓ move** · **enter continue** · **esc back** · **? help** |
| sampling             | setup (text box) | **enter continue** · **esc back** · **ctrl+g help** · backspace delete · ←→ move |
| sampling-invalid     | setup (text box) | **enter continue** · **esc back** · **ctrl+g help** · backspace delete · ←→ move |
| retention            | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** |
| alerts-choose        | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** |
| alerts               | setup (text box) | **enter continue** · **esc back** · **ctrl+g help** · backspace delete · ←→ move |
| review               | setup            | **enter start collecting** · **←→ choose** · **esc back** · **? help** · e edit an answer |
| review-edit          | setup (journey focused) | **↑↓ choose** · **enter edit** · **esc back to review** · **? help** |
| connecting           | setup            | **esc cancel** · **? help** |
| connecting-slow      | setup            | **esc cancel** · **? help** |
| failed               | setup            | **enter retry** · **←→ choose** · **esc back to review** · **? help** |
| failed-auth          | setup            | **enter back to sign-in** · **←→ choose** · **esc back to review** · **? help** |
| done                 | setup            | **enter finish** · **? help** · q quit |
| quit-armed           | setup            | **ctrl+c quit** · **esc keep going** |
| help                 | overlay          | **esc close** · ↑↓ scroll (while the body overflows) · ? close |
| toosmall             | any              | q quit |

Global: ctrl+c quit · ctrl+z suspend

### Harness and fixtures
cmd/lanternfish/fixtures.go feeds setState for the harness (-state, -out) and every test,
with the harness user of § Screens (SSO, cloud eu-west, logs and traces) for the eight
named review states (signin, sso, collect, sampling, review, connecting, failed, done)
and the token and self-hosted paths for the rest.
fakes (cmd/lanternfish/fixtures.go): sign-in (a device link and code, approved on the first poll; the code lives 5:00) → poll and approvalExpires; the ingest endpoint and collector (a heartbeat after 3 s, unreachable after 30 s, a rejected token; `-fake unreachable|rejected` walks them live) → connect and fakeIngest; the CA check → checkCA over a fixed set of paths; this machine's facts (hostname, os, agent version) → a fixed fakeHost. No render or test touches the network or reads the real hostname.

### Async and the clock
sign-in approval → pollApprovalCmd every 5 s while signin-wait shows → approvalMsg | approvalFailedMsg; connect → connect(gen) (30 s per attempt; "still waiting" from 10 s) → heartbeatMsg | connectFailedMsg; the CA path check → checkCAPathCmd → caCheckedMsg; cancel: esc cancels the context and returns to the step before with every answer kept; stale results: a gen on each, so a result after esc is dropped.
Frame clock: one gen-stamped tick, armed only while the route's light runs (connecting) or the sign-in spinner shows; a stale tick dies; elapsed seconds come from the wall clock at each frame.
Product clock: the approval poll every 5 s while signin-wait shows, its own msg and gen; leaving the state stops it, entering it returns the arm cmd, never a gen bump alone.
largest: 11 steps, 5 regions, one 4-option provider list: nothing to virtualise.

### Size floor
Below 80×24: ui.TooSmall, worded by the spec ("lanternfish needs 80×24"). Below 110 columns: the journey at 32 cells with answers cut at 12; "this machine" folds into the header; done's link takes a full-width row. Below 30 rows: "this machine" goes.

### Accessible and non-TTY
ACCESSIBLE=1 or --accessible: no alt screen, Huh WithAccessible(true), one prompt per line in the same order, the token read with echo off and never printed, connecting as new lines ("connecting to lanternfish cloud eu-west", then "first heartbeat 14:02:41"). `lanternfish init --token <t>` runs with no screen at all, TTY or not, and prints plain lines. stdout not a TTY and no --token: one plain line saying the setup needs a terminal or --token, exit 2, never escape codes.

## First slice
Built 2026-10-21: the whole setup flow, all 26 states of § States and keys, rendered from
fixtures through the harness at 80×24 and 120×40 (frames/app/, sheet-*.png), on fakes for
sign-in, ingest and this machine (cmd/lanternfish/fixtures.go). The person chose it: the
flow is the product's one screen, so the slice is the app's UI.
Then: wire the real sign-in, ingest and collector checks in place of the fakes; the
accessible mode's golden transcript; `lanternfish status`.

## Work orders
One standalone prompt per next step, rewritten at every hand-off (references/collaborate.md):

### 1. audit: the built setup flow's mechanics, before any integration
keep: "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
keep: "A pasted token is never shown again after it's pasted, not even masked to its last four characters."
keep: "The steps stay in the order above. Support docs are written against it."
reference: none
method: references/cmd-audit.md on cmd/lanternfish: footprint, profiles, keys (tuilint spec TUI.md --frames frames/app/dark), the one clock, exit paths, the accessible mode run end to end with a screen reader's eyes (ACCESSIBLE=1 lanternfish init), the three F-LEVELS warnings on collector and collector-invalid; the loop (verify.md)
deliver: frames/app/ re-rendered, sheet-80.png, sheet-120.png, sheet-light.png, sheet-ascii.png, the audit's findings in TUI.md § Decisions
decide: nothing; build it (the person's open picks are in picks/ and § Assumed: overruling one comes first)

### 2. craft: wire the real services in place of the fakes
keep: "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
keep: "A pasted token is never shown again after it's pasted, not even masked to its last four characters."
keep: "The steps stay in the order above. Support docs are written against it."
reference: none
method: references/cmd-craft.md §5: replace poll, approvalExpires, connect, checkCA and thisHost in cmd/lanternfish/fixtures.go with the real sign-in, ingest and host checks behind the same messages; keep the fakes for the harness and the tests (no render or test touches the network); the headless --token path on the real ingest; the loop (verify.md)
deliver: the same frames and sheets, unchanged except where real data differs; a test per message kept
decide: nothing; build it

## Decisions
- 2026-10-21 the brand read comes from README § Brand and the person's three words (calm, precise, luminous; never a wall of blinking lights), so there was no brand-read pick point
- 2026-10-21 the steps keep the spec's order and names, because support docs are written against it (the person's words, quoted in § Work orders)
- 2026-10-21 forms are Huh v2 through ui.NewForm, because the person took the default; the token field is hand-written because Huh's password field draws a mask
- 2026-10-21 the floor is 80×24, because the person set it; below it, toosmall
- 2026-10-21 first slice: the whole flow, every state, because the person chose it and the flow is the product's only screen
- 2026-10-21 review is the confirmation of starting, focused on Start collecting, with no confirm-* state, because starting is what the flow exists for and nothing in a first run destroys or replaces what is running; if init ever runs on a configured machine, review names what it replaces and focus moves to the safe button (rung 3)
- 2026-10-21 the token is never drawn, in any state or mode, from the moment it is pasted: the field reads `pasted · hidden`, review reads `API token · pasted, hidden`, accessible mode reads it with echo off, because the person said so
- 2026-10-21 customers' attack (four simulated interviews, fallback critic: this session's own fresh pass, no subagent available): blocking, browser sign-in over SSH (answered: every sign-in shows a link and code to open on any device, signin-wait); blocking, a CA path wrong only at connect (answered: checked on this machine when the collector step continues, collector-invalid); blocking, "setup tools say connected and nothing arrives" (answered: connecting waits for the first heartbeat, done shows its time); blocking, a rejected token after nine steps (answered: failed-auth goes back to sign-in with every other answer kept); minor, a count that grows from 8 to 11 (answered: the collect hint says why, and the journey shows the new steps the moment they are ticked); round 2 found nothing blocking
- 2026-10-21 cut: remembering answers from another machine — a first run on a new machine has nothing to remember, and fleets use `--token` in CI
- 2026-10-21 cut: printing an equivalent command line on review — only `--token` is a flag the product has; inventing the rest changes the product, which is README's to decide
- 2026-10-21 cut: a splash with the fish — the task is three to ten minutes of reading, the lure beside the name carries the mark, and a splash would outlive no work
- 2026-10-21 cuts: journey: answer — no longest (a domain, an email); the review step and the 120-column journey show each answer whole
- 2026-10-21 cuts: done: link — no longest; at 80 columns it takes a full-width row, cut with … only past 76 cells, and its OSC 8 target holds the whole URL
- 2026-10-21 cuts: review: alerts — no longest (an email or a webhook URL); review-edit shows it whole in the form
- 2026-10-21 ▸ (current step) takes the ascii twin ^, not >, because the kept TestOneFocusMarkerColumn found ▸ and the cursor › both drawing > in two columns of one frame: two focus marks under ASCII (shape had accepted the collision; the test overruled it); @ failed TestTwinsStayApartAcrossMeanings, being the ASCII spark ramp's top
- 2026-10-21 ctrl+c quits at once before any answer and asks for a second press after one (quit-armed), because three to ten minutes of answers are lost with no undo, and a second press costs less than a dialog
- 2026-10-21 brand: four axes moved from the kit's ink after diversity counted 4, each for a word of the read: tag bold (calm: a state is a bold word, not a lit chip), focusBar ▎ (geometry: light), sectionTitle accent (luminous: titles in the lure's pale tint), footer rule (geometry: the page's one rule style closes it); now 8 from ink and OK
- 2026-10-21 brand: the prompt is » (ascii >>), not ›, because palette check refused the cursor's glyph as the prompt under selection caret; pending ◌ takes the ascii twin _ because its kit twin . is the open route's
- 2026-10-21 brand: secondary fills the act's button (the kit's pill), so the accents line names it beside the route; info is left unused because it sits 6° from primary, and the cancel toast is muted, not info
- 2026-10-21 craft: the pages whose focused field is a Huh text box or multi-select (sso, collector, collector-invalid, collect, collect-invalid, sampling, sampling-invalid, alerts) are unjudged under help: Huh can't draw them blurred short of Field.Blur, which validates them; the select pages are drawn blurred under help from a copy of the form, so the main screen is judged
- 2026-10-21 craft: the help key is ctrl+g on every step that holds a text box, whichever field has focus, so the key doesn't change as tab moves between fields
- 2026-10-21 cuts: collector: Collector URL — no longest (a URL); review shows it whole
- 2026-10-21 craft: links take accent, underlined, not tertiary, because tertiary lands on error's red at 16 colours and done's link read as a failure (critic round 1)
- 2026-10-21 craft: option descriptions are short enough for one row in ASCII, where the cursor and [x] are wider (TestOptionsFitTheFormColumn), because the ASCII sign-in wrapped "via your identity / provider" (critic round 1)
- 2026-10-21 craft: 120×40 adds the whole answers and "this machine", and nothing else: a calm, one-question-at-a-time setup read once gains noise, not use, from a third pane; the critic marked the blank lower half (P3) and it stays
- 2026-10-21 craft: paragraphs wrap at spaces only, because lipgloss.Wrap broke eu-west and ingest.eu-west at the hyphen

## Assumed
- move and signature: the journey column left of the form, and the route under the header (picks/pick-move.png, option A over B, the accordion), because the form stays in one place on every step while the accordion moves it down a row per step, and every answer stays beside its step (precise)
- connecting: the light runs the route (picks/pick-connecting.png option A, picks/connecting-route-light.gif; over B, a spinner line, and C, the fish with its lure pulsing), because the one moving thing is the signature on the data's own path, 21 cells at 80 columns, and nothing else moves (never a wall of blinking lights)
- the step count counts the step after "where data goes" from the start, as "Region or collector", because every path has one; a branch step joins the count when its answer is chosen
- browser and SSO sign-in show a link and a short code to open on any device (device flow), because most runs are over SSH
- the CA path is checked on this machine when the collector step continues (exists, readable, a PEM certificate); nothing is checked over the network before connecting
- sampling is a whole-number input from 1 to 100 (the hint says percent), not a slider
- alerts: choosing email or webhook opens its one input on the same step, not a new step (the spec brackets no step for it)
- quit: ctrl+c twice once an answer exists (quit-armed); esc never quits
- review edits through the journey (review-edit), and an edit returns to review
- 120×40 adds "this machine" (hostname, os, agent version) under the journey; below 110 columns it folds into the header
- connect gives up after 30 s per attempt (failed), and says "still waiting" from 10 s while a heartbeat may still come (connecting-slow)
- non-TTY without --token prints one line and exits 2
- ACCESSIBLE=1 / --accessible mode as § Accessible and non-TTY says
- paintBg false, the help overlay as dialog: rules, footer with no badge
- keeping the skill in fresh clones (references to the skill's SKILL.md Setup): not set up; at craft, commit the `.claude/skills/pearl` copy, because a lock file pins no commit
- fixtures: every name (tidewater.example, Quaylock, Brightgate ID, Moorhen Directory, ingest-07) is fictional
