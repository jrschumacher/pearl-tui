// quayside is the command-line client for Quayside Cloud, a fictional
// data-protection service. This fixture is only its first-run wizard,
// `quayside setup`; the API and the profile store are fakes, and nothing here
// touches the network.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
)

const usage = `quayside: back up and restore your data with Quayside Cloud

Usage:
  quayside setup                 set up a profile (interactive)
  quayside setup --token <t>     set up a profile without the wizard

Flags for setup --token:
  --profile <name>   profile to create or update (default: the current default)
  --endpoint <url>   API endpoint (default: ` + cloudEndpoint + `)
  --output <fmt>     table, json or yaml (default: table)
  --color <when>     auto, always or never (default: auto)
  --org <name>       default organisation
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, newFakeService(), newFakeStore()))
}

func run(args []string, stdout, stderr io.Writer, svc Service, store Store) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if args[0] != "setup" {
		fmt.Fprintf(stderr, "quayside: unknown command %q\n\n%s", args[0], usage)
		return 2
	}

	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	token := fs.String("token", "", "API token; runs without the wizard")
	profile := fs.String("profile", store.Default(), "profile to create or update")
	endpoint := fs.String("endpoint", cloudEndpoint, "API endpoint")
	output := fs.String("output", "table", "table, json or yaml")
	color := fs.String("color", "auto", "auto, always or never")
	org := fs.String("org", "", "default organisation")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	if *token != "" {
		p := Profile{Name: *profile, Endpoint: *endpoint, Auth: "token", Token: *token, Output: *output, Color: *color, Org: *org}
		if err := headless(p, stdout, svc, store); err != nil {
			fmt.Fprintln(stderr, "quayside: setup failed:", err)
			return 1
		}
		return 0
	}

	m := newModel(svc, store)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(stderr, "quayside:", err)
		return 1
	}
	if m.cancelled || m.w.step != stepDone {
		fmt.Fprintln(stdout, "Setup cancelled. Nothing was saved.")
		return 1
	}
	fmt.Fprintf(stdout, "Profile %s saved to %s.\n", m.w.profileName(), m.w.saved)
	return 0
}

// headless is `quayside setup --token`: no screen, one line per stage.
func headless(p Profile, out io.Writer, svc Service, store Store) error {
	switch p.Output {
	case "table", "json", "yaml":
	default:
		return fmt.Errorf("--output must be table, json or yaml, not %q", p.Output)
	}
	switch p.Color {
	case "auto", "always", "never":
	default:
		return fmt.Errorf("--color must be auto, always or never, not %q", p.Color)
	}
	if p.Name == "" || !nameRe.MatchString(p.Name) {
		return errors.New("--profile must be lowercase letters, numbers and dashes")
	}

	fmt.Fprintf(out, "Checking %s ... ", p.Endpoint)
	if err := svc.Check(p.Endpoint); err != nil {
		fmt.Fprintln(out, "failed")
		return err
	}
	fmt.Fprintln(out, "ok")

	fmt.Fprintf(out, "Connecting as %s ... ", maskToken(p.Token))
	if err := svc.Connect(p); err != nil {
		fmt.Fprintln(out, "failed")
		return err
	}
	fmt.Fprintln(out, "ok")

	path, err := store.Save(p)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Profile %s saved to %s.\n", p.Name, path)
	return nil
}
