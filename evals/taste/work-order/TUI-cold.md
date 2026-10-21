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
colour: truecolor → 256 → 16 → none; under -profile ascii, focus stays visible by the `|` gutter, bold and the `->` cursor, the current step by `>`, a failure by `x` and its tag word (UNREACHABLE, REJECTED), the undecided part of the route by `.`
dialogs and overlays: the only overlay is help; where the background step is gone (16 colours, NO_COLOR, ascii), the edge is a ─ rule above and below (dialog: rules), so it holds in every profile

### Brand read
palette: lure blue #6EC1E4 does the pointing (focus, the current step, the light on the route); dusk violet #9C8FD6 is the far end of that light (the lure's trail while connecting, the far end of the route's lit run once connected) and never stands alone; abyss #0B1420 is the page and every neutral leans toward it; light terminals keep both hues on #F3F6F9 (source: README § Brand; no asset file exists)
tone: calm, precise, quietly warm: short sentences that say what happens and what is kept ("Nothing is sent until you start", "your answers are kept"); exact numbers with units (10%, 30 days, 14:02:41); no exclamation marks, no "Successfully", no "Loading..."
geometry: soft and light: thin ─ rules, ╌ for what is not decided yet, no boxes on the page; the one overlay sits between two rules
density: airy: one blank row between groups at 80×24, because a once-per-machine task is read, not scanned
type: the wordmark is lowercase `lanternfish`, plain bold, never capitalised or tracked; sentence-case titles and labels
motif: the lure, one lit dot ● (ascii *): beside the name in the header, and the light that runs the route; it marks nothing else
move: the journey is a 32-cell column left of the form at every width (40 at 120 columns): every step this user will see, each done one carrying its answer, in place of the kit's step row above the form and its summary strip under it, because precise: every answer stays pinned beside the step that set it, and the form never moves
signature: the route, one row under the header that draws where this machine's data goes (`ingest-07 ── logs · traces ──── lanternfish cloud · eu-west`): ╌ and "destination not set" while undecided, filled in the moment each answer is chosen, a light running it while connecting, lit whole once the first heartbeat arrives, because luminous
skeleton: head bar · status top-row · order sidebar-left · foot keys · verdict none · chart other · chartGlyphs line · turnEnd -
accents: primary (lure blue) = lit: the focus gutter, the current step, the route's light; secondary (dusk violet) = the light's far end, on the route only (the wordmark is plain, § type); success (green) = collecting, on the done tag; error (coral) = failed, on the fail mark and its tag; warning = a field that needs fixing, under that field only
focus: the ▌ gutter in primary and a bold label; "focus": "primary" (primary is the brand's light, not a state)
glyphs: ✓ done (ascii +) · ▸ current step (ascii >) · ○ step to come (ascii o) · ✕ failed (ascii x) · ● the lure (ascii *) · ╌ route not decided (ascii .) · ─ route decided (ascii -) · ━ route lit (ascii =) · ▌ focus gutter (ascii |) · › cursor (ascii ->) · ■ □ ticked, not ticked (ascii [x] [ ])
width-safe: every mark sits in a 2-cell column (mark and a space), since ● ○ ▸ ■ □ are East Asian ambiguous and draw two cells in CJK locales
expressions: header minimal (the name left, the machine right, because calm) · wordmark plain · selection caret (› in the focus ink) · button underline: the act in the lure ink, bold and underlined, the rest as muted text, because soft and light (no filled blocks on the page; a pill is one) · dialog rules · footer keys (no badge: there is only one mode) · density airy · scrim none: under help the journey and form hide in every profile while the header, route and footer stay, because calm (one layer of text at a time; help lists this step's keys, so nothing behind it must be read) and the edge holds the same at truecolour, 16 colours and no colour · working spinner (arc ◜◠◝◞◡◟ for the sign-in wait; the connecting light is the app's own) · paintBg false (a fresh terminal over SSH keeps its own background)

### Theme
themes/lanternfish.json   (written by brand, 2026-10-21: palette derive "#6EC1E4" "#9C8FD6" --bg "#0B1420" --light-bg "#F3F6F9" (source: README § Brand), then tuned: tertiary a lightness step of lure blue (#A3D6EE / #1F6C8B) in place of a coral complement that sat beside error, info parked on a neutral (#9FACBD / #4F5B61) because the app draws no info and blue means the lure alone; prompt » (ascii >>) so it never shares › and ▸'s >; route marks lure ● routeOpen ╌ routeSet ─ routeLit ━ in glyphs.marks; palette check OK; diversity 8 from ink)

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
80×24:  header 0,0 80×1 · route 2,2 76×1 · journey 2,4 32×18 (mark 2 | name 18 | answer 12) · form 36,4 42×18 · footer 0,23 80×1
120×40: header 0,0 120×1 · route 2,2 116×1 · journey 2,4 40×14 (mark 2 | name 20 | answer 18) · this machine 2,20 40×4 · form 46,4 72×34 (the measure) · footer 0,39 120×1
collapse: < 110 cols: the journey goes to 32 cells with answers cut at 12 (the review step shows each whole), "this machine" folds into the header's `setup on ingest-07`, and done's link takes a full-width row under the journey; < 30 rows: "this machine" goes; < 80×24: toosmall
components: ui.Header (minimal), ui.Section (the form's title with `step n of N` as its info), ui.NewForm (Huh select), ui.Steps for the journey's marks, ui.KV (this machine), ui.Help, ui.TooSmall; new: Journey (ui.Steps.List draws marks and names but no answer column, and the column takes focus in review-edit) and Route (the signature: a one-row path no kit component draws)
fixtures: host ingest-07; the longest host `ingest-worker-eu-central-12.tidewater.internal` (the header and route cut it with …, the route keeps the destination whole); org tidewater.example; identity providers Quaylock, Brightgate ID, Moorhen Directory, another SAML 2.0 provider; regions us-east, us-west, eu-west, eu-central, ap-southeast; collector https://collector.tidewater.internal:4318 with CA /etc/ssl/tidewater/collector-ca.pem; alerts platform-alerts@tidewater.example and https://hooks.tidewater.example/lanternfish; first heartbeat 14:02:41; the unreachable endpoint ingest.eu-west.lanternfish.example:443; the harness user picked SSO (Quaylock), cloud eu-west, logs and traces, sampling 10%, retention 30 days, email alerts
step count: the steps this user will see, known so far: a step every path has counts from the start (whichever of Region or Collector follows step 2, shown as "Region or collector" until chosen), and a branch step joins the journey and the count the moment its answer is chosen (SSO, API token, Trace sampling, Log retention), so signin reads 1 of 8 and the harness user ends at 11 of 11
flow: the spec's seven steps in the spec's order, with each branch step only when an earlier answer calls for it, because one person configures one machine they are standing on, the answers are only drafts until review's Start collecting, and a fleet already has its expert path outside the TUI (`--token` in CI); calm asks one question per step and nothing more, and support docs follow this order
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ destination not set  
                                                                                
  ▸ Sign in                         Sign in ───────────────────── step 1 of 8   
  ○ Where data goes                                                             
  ○ Region or collector             ▌ How do you sign in?                       
  ○ What to collect                 ▌ each one works over SSH                   
  ○ Alerts                          ▌ › Browser    open a link on any device    
  ○ Review                          ▌   SSO        via your identity provider   
  ○ Connecting                      ▌   API token  paste one from lanternfish   
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ↑↓ choose · enter continue · ? help                                           
```

#### sso — the SSO details step, half filled: domain typed, the identity provider being chosen
brand: precise: each field's hint says exactly what to type ("the part after @ in your work email"), and the second field's hint names the domain just typed, so the user checks one against the other
80×24 / 120×40: the setup frame; form: domain input, then the provider select
keys: tab moves between the two fields; enter on the domain moves to the provider, on the provider continues
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌ destination not set  
                                                                                
  ✓ Sign in           SSO           SSO details ───────────────── step 2 of 9   
  ▸ SSO details                                                                 
  ○ Where data goes                   Organisation domain                       
  ○ Region or collector               the part after @ in your work email       
  ○ What to collect                   > tidewater.example                       
  ○ Alerts                                                                      
  ○ Review                          ▌ Identity provider                         
  ○ Connecting                      ▌ the one tidewater.example signs in with   
  ○ Done                            ▌ › Quaylock                                
                                    ▌   Brightgate ID                           
                                    ▌   Moorhen Directory                       
                                    ▌   another SAML 2.0 provider               
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  ↑↓ choose · enter continue · tab field · esc back · ctrl+g help               
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
brand: precise: the error under the field names the path and the fault (`/etc/ssl/tidewater/collector-ca.pem: no such file`), in the warning ink with the warn mark, and focus returns to that field

#### collect — what to collect, with logs and traces ticked
brand: precise: each option names what it gathers on this machine, and a quiet line under the field says ticking traces and logs adds their steps, which appear in the journey as they are ticked
flow: ticking a box adds its step to the journey and the count at once; unticking drops that step and only its answer
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           What to collect ──────────── step 5 of 11   
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         ▌ What should ingest-07 send?               
  ✓ Region            eu-west       ▌ at least one                              
  ▸ What to collect                 ▌   ■ Logs     files and the journal        
  ○ Trace sampling                  ▌   □ Metrics  cpu, memory, disk, network   
  ○ Log retention                   ▌ › ■ Traces   spans from your services     
  ○ Alerts                                                                      
  ○ Review                            traces adds a sampling step and logs a    
  ○ Connecting                        retention step; the list shows both       
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  space tick · ↑↓ move · enter continue · esc back · ? help                     
```

#### collect-invalid — enter with nothing ticked
brand: calm: `Pick at least one to collect` under the field, nothing else moves

#### sampling — the trace sampling rate, 10% by default
brand: precise: the rate is a number with its % drawn after it, and a muted line restates it as a fact that updates as you type (`10% sends about one trace in ten`)
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Trace sampling ───────────── step 6 of 11   
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         ▌ Sampling rate                             
  ✓ Region            eu-west       ▌ share of traces sent, 1 to 100            
  ✓ What to collect   logs, traces  ▌ > 10▏%                                    
  ▸ Trace sampling                                                              
  ○ Log retention                     10% sends about one trace in ten          
  ○ Alerts                                                                      
  ○ Review                                                                      
  ○ Connecting                                                                  
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  enter continue · esc back · ctrl+g help                                       
```

#### sampling-invalid — a rate outside 1 to 100
brand: calm: `Use a whole number from 1 to 100` under the field; the previous valid rate stays in the journey

#### retention — log retention: 7, 30 or 90 days, 30 by default
brand: precise: the three options as numbers with their unit, 30 marked as the default in subtle

#### alerts — optional alerts: email, webhook, or skip for now
brand: calm: skipping is a first-class option named `Skip for now`, not a failure to answer; choosing email or webhook opens its one input on the same step

#### review — everything chosen, editable, then start collecting
brand: quietly warm: the answers as a labelled list read top to bottom, one sentence that nothing is sent until you start, and two buttons named for what they do: `Start collecting` (focused) and `Edit an answer`
flow: the review step is the confirmation of starting, with focus on the act because it is what the flow exists for (interaction.md § Confirm By Consequence: the act a flow exists for starts on the act); `Edit an answer` moves focus to the journey (review-edit); an edit returns here, walking any step the edit adds; no dialog follows Start collecting
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Review ───────────────────── step 9 of 11   
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         Sign in        SSO · Quaylock               
  ✓ Region            eu-west       Organisation   tidewater.example            
  ✓ What to collect   logs, traces  Data goes to   lanternfish cloud · eu-west  
  ✓ Trace sampling    10%           Collect        logs, traces                 
  ✓ Log retention     30 days       Trace sampling 10%                          
  ✓ Alerts            email         Log retention  30 days                      
  ▸ Review                          Alerts         email platform-alerts@t…     
  ○ Connecting                                                                  
  ○ Done                            Nothing is sent until you start.            
                                                                                
                                    ▌  Start collecting    Edit an answer       
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  enter start collecting · ←→ choose · esc back · ? help                        
```

#### review-edit — choosing which answer to change
brand: precise: the journey takes the focus gutter on a done step, the form shows that step's answer in full, and `enter edit` opens it

#### connecting — the agent connects and waits for the first heartbeat (2 to 5 seconds)
brand: luminous, never blinking lights: the one thing that moves is the light running the route; the form says what it waits for, the seconds after 3, and the usual time
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ────────━━━●───────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Connecting ──────────────── step 10 of 11   
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud         Connecting to lanternfish cloud · eu-west   
  ✓ Region            eu-west       waiting for the first heartbeat · 3s        
  ✓ What to collect   logs, traces                                              
  ✓ Trace sampling    10%           usually 2 to 5 seconds                      
  ✓ Log retention     30 days                                                   
  ✓ Alerts            email                                                     
  ✓ Review                                                                      
  ▸ Connecting                                                                  
  ○ Done                                                                        
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  esc cancel · ? help                                                           
```

#### connecting-slow — no heartbeat after 10 seconds
brand: calm: `Still waiting for ingest-07's first heartbeat · 14s`, what usually causes it (outbound HTTPS blocked), and that esc cancels with every answer kept

#### failed — the destination is unreachable
brand: calm and exact: a tag word (UNREACHABLE), the endpoint and the timeout, the one likely cause and the next step, that nothing started and every answer is kept; the route breaks at ✕ before the destination; `Retry` focused, `Back to review` beside it
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ── logs · traces ───────────────────── lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Connecting ──────────────── step 10 of 11   
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud          UNREACHABLE  lanternfish cloud · eu-west   
  ✓ Region            eu-west                                                   
  ✓ What to collect   logs, traces  ingest-07 can't reach                       
  ✓ Trace sampling    10%           ingest.eu-west.lanternfish.example:443      
  ✓ Log retention     30 days       (timed out after 10s). Check that this      
  ✓ Alerts            email         machine allows outbound HTTPS, then retry.  
  ✓ Review                          Nothing started; your answers are kept.     
  ✕ Connecting        unreachable                                               
  ○ Done                            ▌  Retry    Back to review                  
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
                                                                                
  enter retry · ←→ choose · esc back to review · ? help                         
```

#### failed-auth — the token or sign-in was rejected at connect
brand: calm and exact: tag REJECTED, `lanternfish cloud eu-west rejected this machine's token`, and `Back to sign-in` focused, which returns here through review with every other answer kept

#### done — collecting; what and where to look
brand: quietly warm: one green tag word (COLLECTING), the first heartbeat's time as proof, the link to this host, and `lanternfish status` named for later; the route is lit whole with the lure at the destination
sketch (exactly 80 wide, real content):
```
  ● lanternfish                                             setup on ingest-07  
                                                                                
  ingest-07 ━━ logs · traces ━━━━━━━━━━━━━━━━━━━━● lanternfish cloud · eu-west  
                                                                                
  ✓ Sign in           SSO           Done ────────────────────── step 11 of 11   
  ✓ SSO details       tidewater.e…                                              
  ✓ Where data goes   cloud          COLLECTING  logs and traces                
  ✓ Region            eu-west                                                   
  ✓ What to collect   logs, traces  First heartbeat from ingest-07 at           
  ✓ Trace sampling    10%           14:02:41, in lanternfish cloud · eu-west.   
  ✓ Log retention     30 days                                                   
  ✓ Alerts            email         lanternfish status shows what this          
  ✓ Review                          agent is collecting, any time.              
  ✓ Connecting        14:02:41                                                  
  ▸ Done                                                                        
                                                                                
                                                                                
  Where to look ─────────────────────────────────────────────────────────────   
                                                                                
  https://eu-west.lanternfish.example/tidewater/hosts/ingest-07                 
                                                                                
                                                                                
                                                                                
  enter finish · ? help                                                         
```

#### quit-armed — ctrl+c pressed once after an answer was given
brand: calm: the footer row says `ctrl+c again to quit · nothing is saved` for 4 s, then the keys return; before any answer, ctrl+c quits at once

#### help — every key of the current step, grouped
brand: nothing, because help lists the action tables as written; its edge is the rules dialog (dialog: rules)

#### toosmall — below 80×24
brand: calm: `lanternfish needs 80×24 · this terminal is 72×20`, and q quit

### Motion
hero: connecting → the lure ● with a 3-cell trail (lure blue to dusk violet) runs the route's open run toward the destination, one cell per frame (50 ms), and starts over; at most the 30 cells before the destination; it loops, it never fills, because the wait has no fraction (motion.md: no fake progress)
rows: the sign-in wait's arc spinner ◜◠◝◞◡◟ beside its subject and seconds; idle: nothing moves, and the done frame's lit route is still

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
| signin               | setup (select)   | **↑↓ choose** · **enter continue** · **? help** · j/k choose |
| signin-wait          | setup            | **c copy link** · **esc back** · **? help** |
| signin-failed        | setup            | **enter retry** · **←→ choose** · **esc back** · **? help** |
| sso                  | setup (text box) | **↑↓ choose** · **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** |
| token                | setup (text box) | **enter continue** · **ctrl+u clear** · **esc back** · **ctrl+g help** |
| destination          | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** · j/k choose |
| region               | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** · j/k choose |
| collector            | setup (text box) | **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** |
| collector-invalid    | setup (text box) | **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** |
| collect              | setup (multi-select) | **space tick** · **↑↓ move** · **enter continue** · **esc back** · **? help** · x tick |
| collect-invalid      | setup (multi-select) | **space tick** · **↑↓ move** · **enter continue** · **esc back** · **? help** · x tick |
| sampling             | setup (text box) | **enter continue** · **esc back** · **ctrl+g help** |
| sampling-invalid     | setup (text box) | **enter continue** · **esc back** · **ctrl+g help** |
| retention            | setup (select)   | **↑↓ choose** · **enter continue** · **esc back** · **? help** · j/k choose |
| alerts               | setup (select, text box once email or webhook is chosen) | **↑↓ choose** · **enter continue** · **tab/shift+tab field** · **esc back** · **ctrl+g help** |
| review               | setup            | **enter start collecting** · **←→ choose** · **esc back** · **? help** · e edit an answer |
| review-edit          | setup (journey focused) | **↑↓ choose** · **enter edit** · **esc back to review** · **? help** |
| connecting           | setup            | **esc cancel** · **? help** |
| connecting-slow      | setup            | **esc cancel** · **? help** |
| failed               | setup            | **enter retry** · **←→ choose** · **esc back to review** · **? help** |
| failed-auth          | setup            | **enter back to sign-in** · **←→ choose** · **esc back to review** · **? help** |
| done                 | setup            | **enter finish** · **? help** · q quit |
| quit-armed           | setup (toast over the footer) | **ctrl+c quit** · **esc keep going** |
| help                 | overlay          | **esc close** · **? close** |
| toosmall             | any              | q quit |

Global: ctrl+c quit · ctrl+z suspend

### Harness and fixtures
cmd/lanternfish/fixtures.go feeds setState for the harness (-state, -out) and every test,
with the harness user of § Screens (SSO, cloud eu-west, logs and traces) for the eight
named review states (signin, sso, collect, sampling, review, connecting, failed, done)
and the token and self-hosted paths for the rest.
fakes: sign-in (browser link and code, SSO, token check) → fakeAuth; the ingest endpoint and collector (a heartbeat after 3 s, unreachable after 10 s, a rejected token) → fakeIngest; this machine's facts (hostname, os, agent version) → a fixed fakeHost. No render or test touches the network or reads the real hostname.

### Async and the clock
sign-in approval → pollApprovalCmd every 5 s while signin-wait shows → approvalMsg | approvalFailedMsg; connect → connectCmd (context, 10 s per attempt) → heartbeatMsg | connectFailedMsg; the CA path check → checkCAPathCmd → caCheckedMsg; cancel: esc cancels the context and returns to the step before with every answer kept; stale results: a gen on each, so a result after esc is dropped.
Frame clock: one gen-stamped tick, armed only while the route's light runs (connecting) or the sign-in spinner shows; a stale tick dies; elapsed seconds come from the wall clock at each frame.
Product clock: the approval poll every 5 s while signin-wait shows, its own msg and gen; leaving the state stops it, entering it returns the arm cmd, never a gen bump alone.
largest: 11 steps, 5 regions, one 4-option provider list: nothing to virtualise.

### Size floor
Below 80×24: ui.TooSmall, worded by the spec ("lanternfish needs 80×24"). Below 110 columns: the journey at 32 cells with answers cut at 12; "this machine" folds into the header; done's link takes a full-width row. Below 30 rows: "this machine" goes.

### Accessible and non-TTY
ACCESSIBLE=1 or --accessible: no alt screen, Huh WithAccessible(true), one prompt per line in the same order, the token read with echo off and never printed, connecting as new lines ("connecting to lanternfish cloud eu-west", then "first heartbeat 14:02:41"). `lanternfish init --token <t>` runs with no screen at all, TTY or not, and prints plain lines. stdout not a TTY and no --token: one plain line saying the setup needs a terminal or --token, exit 2, never escape codes.

## First slice
The whole setup flow, every state in § States and keys, rendered from fixtures through
the harness at 80×24 and 120×40, before fakeAuth and fakeIngest are swapped for the real
services. The user chose this: the flow is the product's one screen, so the slice is the
app's UI.
Then: wire the real sign-in, ingest and collector checks; the headless `--token` path's plain output; the accessible mode's golden transcript.

## Work orders
One standalone prompt per next step, rewritten at every hand-off (references/collaborate.md).
Done: 1. brand (themes/lanternfish.json, sheet-brand.png, sheet-brand-light.png) and 2. craft
(cmd/lanternfish, every state of § States and keys on fixtures; sheet-80.png, sheet-120.png,
sheet-light.png, sheet-ascii.png, connecting.gif), 2026-10-21.

### 3. audit: the mechanics of the setup flow
keep: "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
keep: "A pasted token is never shown again after it's pasted, not even masked to its last four characters."
keep: "The steps stay in the order above. Support docs are written against it."
reference: none
method: references/cmd-audit.md on cmd/lanternfish: footprint, profiles, every key per state, the one frame clock and the approval poll, exit paths (ctrl+c twice, ctrl+z, esc everywhere), the -ascii and NO_COLOR frames; then the loop (verify.md) with a critic
deliver: the audit's findings fixed, frames/app re-rendered, sheets re-made
decide: nothing; build it

### 4. craft: wire the real services behind the fakes (TUI.md § First slice, Then:)
keep: "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
keep: "A pasted token is never shown again after it's pasted, not even masked to its last four characters."
keep: "The steps stay in the order above. Support docs are written against it."
reference: none
method: replace fakeAuth, fakeIngest and fakeHost (cmd/lanternfish/fixtures.go) with the real sign-in, ingest, collector and CA checks behind the same messages; the headless --token path's plain output against the real service (what destination and collection a token run uses is the person's to answer: README does not say); the accessible mode's golden transcript; keep the fakes for the harness and tests
deliver: the real services wired, the harness unchanged, an accessible transcript golden, go test ./... green
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
- 2026-10-21 craft: › (cursor) takes the ascii twin -> and ▸ (current step) keeps >, replacing the decision that they share >: the kit's TestOneFocusMarkerColumn fails two > columns in one -ascii frame (the journey's and a select's), whatever the reason; cutRight moved to ▹ (ascii >) because a cut-edge mark takes one cell
- 2026-10-21 brand: button underline in place of pill, because § geometry says no blocks on the page and the read's own button line asked for "the rest as text", which a pill row draws as two filled chips; diversity's 6 axes from ink was what surfaced the contradiction
- 2026-10-21 brand: scrim none in place of dim, because help is the only overlay and lists this step's keys, so nothing behind it must stay readable, and at 16 colours and NO_COLOR nothing dims anyway: one look for help in every profile; header, route and footer stay
- 2026-10-21 brand: the wordmark stays plain (§ type, § expressions); the palette line's "wordmark gradient's end" was dropped, since two lines said plain and one said gradient
- 2026-10-21 brand: the input prompt is » (ascii >>), because › (cursor) and ▸ (current) already share >, and palette check fails a prompt that draws as the cursor's twin
- 2026-10-21 craft: the token and the sampling rate are hand-written fields (the token is never drawn; Huh's input can't draw the % after its caret); every other input is Huh through ui.NewForm
- 2026-10-21 craft: field faults (collector-invalid, collect-invalid, sampling-invalid) are checked by the app on enter and drawn by it under the form, the warn mark and the words in the warning ink, wrapped and never cut; Huh's own validation, in the error ink with ✕, is not used
- 2026-10-21 craft: selects take a draft answer committed on enter, so moving a cursor never adds or drops a branch step; the collect multi-select is bound live, since ticking is choosing
- 2026-10-21 craft: alerts opens its address input on the same step after enter on Email or Webhook; its help key is ? until the input is open and ctrl+g after (the harness alerts frame shows the select focused with the input open, the row of § States and keys)
- 2026-10-21 craft: help's footer shows both esc close and ? close, as § States and keys says; help is the only overlay, scrim none hides the journey and form under it, so no page needs an unjudgedPages entry
- 2026-10-21 craft: review-edit draws the focus bar in the chosen done step's mark column (no cell at the screen edge)
- 2026-10-21 craft: the failed route breaks at ✕ before the destination, as the failed block's brand line says (its sketch drew an unbroken route)
- 2026-10-21 craft: option descriptions fit 40 cells in every profile (destination: "stored in a region" / "your own collector"; alerts "one email address"; metrics "cpu, memory, disk, net"), because the ASCII cursor -> and [x] are wider and Huh wraps an option that doesn't fit
- 2026-10-21 craft: quit-armed's footer reads `ctrl+c quit  esc keep going` with `nothing is saved` at its right: the § States and keys row is the contract (tuilint spec S-GOLDEN); the brand line's "ctrl+c again to quit" wording is not used
- 2026-10-21 craft: lanternfish init --token runs on fakeIngest with no screen and prints plain lines; with no token and no terminal, one line and exit 2; the destination and collection a token run uses are the account's (README does not say; flagged for the person)
- 2026-10-21 craft: the skill is kept in fresh clones as a committed copy in .claude/skills/pearl (§ Assumed)
- 2026-10-21 craft verify: critic was the documented fallback (this session's own fresh pass; no subagent tool). Round 1 FIX 19/24 (P1 -ascii select options wrapped mid-description; P2 the route's runs in the border role read as absent; P2 review-edit's focus bar at column 0; P2 prose broken at a hyphen, "eu-/west"); all fixed. Round 2 SHIP 22/24; left: P3 a measure-capped form leaves 200×60's right half empty, P3 help at 200×60 is sparse (tuilint F-EMPTY, P2 note), P3 quit-armed's footer words differ from its brand line (above)
- 2026-10-21 craft: the kit's ui.List and ui.Table marker column (internal/ui/list.go) widens to the cursor's ASCII twin and a space under selection caret, since -> is two cells; the kit's TestItemMarkAndLead failed on it (feedback for the skill: a multi-cell cursor twin breaks the kit's 2-cell marker)
- 2026-10-21 ctrl+c quits at once before any answer and asks for a second press after one (quit-armed), because three to ten minutes of answers are lost with no undo, and a second press costs less than a dialog

## Assumed
- move and signature: the journey column left of the form, and the route under the header (picks/pick-move.png, option A over B, the accordion), because the form stays in one place on every step while the accordion moves it down a row per step, and every answer stays beside its step (precise)
- connecting: the light runs the route (picks/pick-connecting.png option A, picks/connecting-route-light.gif; over B, a spinner line, and C, the fish with its lure pulsing), because the one moving thing is the signature on the data's own path, 21 cells at 80 columns, and nothing else moves (never a wall of blinking lights)
- the step count counts the step after "where data goes" from the start, as "Region or collector", because every path has one; a branch step joins the count when its answer is chosen
- browser and SSO sign-in show a link and a short code to open on any device (device flow), because most runs are over SSH
- the CA path is checked on this machine when the collector step continues (exists, readable, a PEM certificate); nothing is checked over the network before connecting
- sampling is a whole-number input from 1 to 100 with % drawn after it, not a slider
- alerts: choosing email or webhook opens its one input on the same step, not a new step (the spec brackets no step for it)
- quit: ctrl+c twice once an answer exists (quit-armed); esc never quits
- review edits through the journey (review-edit), and an edit returns to review
- 120×40 adds "this machine" (hostname, os, agent version) under the journey; below 110 columns it folds into the header
- connect gives up after 10 s per attempt (failed), and says "still waiting" from 10 s while a heartbeat may still come (connecting-slow)
- non-TTY without --token prints one line and exits 2
- ACCESSIBLE=1 / --accessible mode as § Accessible and non-TTY says
- paintBg false, the help overlay as dialog: rules, footer with no badge
- keeping the skill in fresh clones (references to the skill's SKILL.md Setup): not set up; at craft, commit the `.claude/skills/pearl` copy, because a lock file pins no commit
- fixtures: every name (tidewater.example, Quaylock, Brightgate ID, Moorhen Directory, ingest-07) is fictional
