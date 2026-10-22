package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"charm.land/huh/v2"
)

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// buildForm makes the Huh form for the current step. Values are bound to the
// wizard's answers.
func (m *model) buildForm() *huh.Form {
	w := m.w
	a := &w.a
	var f *huh.Form

	switch w.step {
	case stepProfile:
		opts := []huh.Option[string]{huh.NewOption("Create a new profile", newProfile)}
		for _, p := range w.store.Profiles() {
			label := p.Name
			if p.Name == w.store.Default() {
				label += " (current default)"
			}
			opts = append(opts, huh.NewOption(label, p.Name))
		}
		f = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Profile").
					Description("Create a new profile or update an existing one.").
					Options(opts...).
					Value(&a.Profile),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Profile name").
					Placeholder("e.g. prod-us").
					Value(&a.NewName).
					Validate(func(s string) error {
						s = strings.TrimSpace(s)
						if s == "" {
							return errors.New("enter a name for the profile")
						}
						if !nameRe.MatchString(s) {
							return errors.New("use lowercase letters, numbers and dashes")
						}
						for _, p := range w.store.Profiles() {
							if p.Name == s {
								return fmt.Errorf("%s already exists, pick it from the list to update it", s)
							}
						}
						return nil
					}),
			).WithHideFunc(func() bool { return !w.isNew() }),
		)

	case stepConnection:
		f = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Destination").
					Options(
						huh.NewOption("Quayside Cloud", "cloud"),
						huh.NewOption("Custom endpoint", "custom"),
					).
					Value(&a.Dest),
				huh.NewNote().
					Title("Endpoint").
					DescriptionFunc(func() string {
						if a.Dest == "custom" {
							return "You'll enter the URL on the next screen."
						}
						return cloudEndpoint
					}, &a.Dest),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Endpoint URL").
					Placeholder("https://quayside.internal/v2").
					Value(&a.Endpoint).
					Validate(func(s string) error {
						u, err := url.Parse(strings.TrimSpace(s))
						if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
							return errors.New("enter a full URL, like https://quayside.internal/v2")
						}
						return nil
					}),
			).WithHideFunc(func() bool { return a.Dest != "custom" }),
		)

	case stepAccess:
		code, verify := m.svc.DeviceCode()
		f = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("How do you want to sign in?").
					Options(
						huh.NewOption("Sign in with the browser", "browser"),
						huh.NewOption("Paste an API token", "token"),
					).
					Value(&a.Auth),
			),
			huh.NewGroup(
				huh.NewNote().
					Title("Sign in with the browser").
					Description(fmt.Sprintf("Open %s\nand enter the code %s", verify, code)),
				huh.NewConfirm().
					Title("Did you approve the code in the browser?").
					Affirmative("Yes, continue").
					Negative("Not yet").
					Value(&a.Approved).
					Validate(func(ok bool) error {
						if !ok {
							return errors.New("approve the code in your browser first")
						}
						return nil
					}),
			).WithHideFunc(func() bool { return a.Auth != "browser" }),
			huh.NewGroup(
				huh.NewInput().
					Title("API token").
					Description("Create one under Settings > API tokens.").
					EchoMode(huh.EchoModePassword).
					Value(&a.Token).
					Validate(func(s string) error {
						if !strings.HasPrefix(strings.TrimSpace(s), "qs_") {
							return errors.New("tokens start with qs_")
						}
						return nil
					}),
			).WithHideFunc(func() bool { return a.Auth != "token" }),
		)

	case stepPreferences:
		f = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Output format").
					Options(huh.NewOptions("table", "json", "yaml")...).
					Value(&a.Output),
				huh.NewSelect[string]().
					Title("Color").
					Options(huh.NewOptions("auto", "always", "never")...).
					Value(&a.Color),
				huh.NewInput().
					Title("Default organisation").
					Placeholder("my-org").
					Value(&a.Org),
			),
		)

	case stepReview:
		f = huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Review").
					Description(strings.Join(w.review(), "\n")),
				huh.NewConfirm().
					Title("Connect with these settings?").
					Affirmative("Connect").
					Negative("Back").
					Value(&a.Connect),
			),
		)

	default:
		return nil
	}

	// give the form the whole height of the screen
	return f.WithShowHelp(false).WithWidth(m.formWidth()).WithHeight(m.formHeight())
}
