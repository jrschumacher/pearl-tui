# quayside

The command-line client for Quayside Cloud, a data-protection service: scheduled
backups of databases and file stores, kept off-site, restored from the terminal. Every
product, service and brand in this directory is fictional, and nothing here touches the
network: the API and the profile store are fakes (`service.go`).

This module is only the client's first-run wizard, `quayside setup`. People run it once
per machine or per environment, usually right after installing the CLI, often over SSH
on a server they are about to protect, and then rarely see it again. It writes a
profile: which endpoint to talk to, how to authenticate, and a few output preferences.

## Usage

```
quayside setup                         the wizard (needs a terminal)
quayside setup --token <t> [flags]     the same, headless: no screen, one line per stage
quayside --help
```

Headless flags: `--profile <name>` (default: the current default profile),
`--endpoint <url>`, `--output table|json|yaml`, `--color auto|always|never`,
`--org <name>`.

## The wizard

Five steps, then a done screen:

1. **Profile**: create a new profile (and name it) or update an existing one. The fake
   store has `local`, `local-dev` and `staging-eu`, the current default. Updating a
   profile pre-fills the later steps from it.
2. **Connection**: Quayside Cloud (`https://api.quayside.example/v2`) or a custom
   endpoint URL, then a connection check (about 1 s).
3. **Access**: sign in with the browser (a device code and a URL to open), or paste an
   API token (masked; tokens start with `qs_`).
4. **Preferences**: output format, colour, default organisation.
5. **Review & connect**: every answer, then connect (about 2 s).

Keys: `enter` next, arrows to choose, `shift+tab` previous field, `ctrl+b` previous
screen, `esc` cancel, `ctrl+c` quit.

## Running it

Go 1.26.7; the dependencies are pinned in `go.mod` (Bubble Tea v2.1.0, Lip Gloss
v2.0.6, Bubbles v2.2.1, Huh v2.0.3).

```sh
go run . setup                          # the wizard
go run . setup --token qs_live_9f3a7c21 # headless
go vet ./... && go test ./...           # the flow, not the look
```

To see the failure paths:

- **Check fails (unreachable)**: choose *Custom endpoint* and enter any URL whose host
  contains `unreachable`, e.g. `https://unreachable.quayside.example/v2`; or run with
  `QUAYSIDE_FAKE_FAIL=check`.
- **Connect fails (refused)**: run with `QUAYSIDE_FAKE_FAIL=connect`.

Both work headless too.

## Brand

- **Name**: `quayside`, always lowercase, in the product and in prose. The service is
  "Quayside Cloud".
- **Colours**: harbor blue `#4F8FD9` (primary) and sand `#D9C49A` (secondary) on
  `#1E2229`.
- **Voice**: steady, plain, exact. Say what happened and what to do next, with the real
  endpoint, profile or code in the sentence. No exclamation marks, no cheerleading, no
  jargon the user didn't type.
- **Feel**: a calm, trustworthy tool someone runs once while setting up a server. It
  must never feel like a corporate admin panel.
- **Mark**: none yet.

## Files

```
main.go        the CLI: usage, flags, the headless --token path
wizard.go      the flow: steps, answers, back/forward, check and connect results
forms.go       one Huh form per step
model.go       the Bubble Tea model
view.go        the layout and styles
service.go     the fake API and profile store
setup_test.go  tests of the flow and the headless path
```
