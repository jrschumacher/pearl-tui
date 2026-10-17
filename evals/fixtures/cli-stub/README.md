# cairn

Feature flags for small teams, from the terminal. Synthetic fixture for the
`init-interview` eval: every product, service and brand in it is fictional.

cairn keeps one flag file per environment (`dev`, `staging`, `prod`) in a Git repo and
lets you list, toggle and roll out flags without opening a web console. It is used by
backend engineers during a deploy, usually for under a minute at a time, in a split tmux
pane next to the deploy logs, and over SSH into a bastion host for production.

## Usage

```
cairn flags list [--env prod]          every flag with its state per environment
cairn flags toggle <name> --env <env>  flip a flag
cairn flags rollout <name> --percent N  ramp a flag to a share of users
cairn envs                             the environments and their last change
cairn --help
```

## Brand

- Name: lowercase `cairn`, always.
- Colours: slate blue `#3B5B8C` (primary) and lichen `#9CB37A` (secondary) on near-black
  `#101418`. Light terminals use the same hexes on `#F6F5F1`.
- Logo: `assets/logo.svg`, a stack of three flat stones.
- Typeface on the website: Inter. The brand is calm, exact and a little outdoorsy; it
  must never feel like a dashboard full of gauges.
