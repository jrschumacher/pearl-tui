package main

import (
	"fmt"
	"strings"
)

type step int

const (
	stepProfile step = iota
	stepConnection
	stepAccess
	stepPreferences
	stepReview
	stepDone
)

var stepNames = []string{"Profile", "Connection", "Access", "Preferences", "Review"}

const newProfile = "+new"

// answers are bound straight into the Huh fields, so they survive going back
// and rebuilding a step's form.
type answers struct {
	Profile  string // an existing profile's name, or newProfile
	NewName  string
	Dest     string // "cloud" or "custom"
	Endpoint string // custom endpoint
	Auth     string // "browser" or "token"
	Token    string
	Approved bool // browser sign-in confirmed
	Output   string
	Color    string
	Org      string
	Connect  bool // review: true connects, false goes back
}

// wizard is the flow, without any of the screen.
type wizard struct {
	step   step
	a      answers
	done   [5]bool
	store  Store
	err    error  // last check/connect failure, shown on the current step
	busy   string // non-empty while a fake check or connect runs
	saved  string // where the profile was written
	loaded string // the profile whose values were last copied into a
}

func newWizard(store Store) *wizard {
	w := &wizard{store: store}
	w.a = answers{
		Profile: store.Default(),
		Dest:    "cloud",
		Auth:    "browser",
		Output:  "table",
		Color:   "auto",
		Connect: true,
	}
	w.load()
	return w
}

func (w *wizard) isNew() bool { return w.a.Profile == newProfile }

// profileName is the name the profile will be saved under.
func (w *wizard) profileName() string {
	if w.isNew() {
		return strings.TrimSpace(w.a.NewName)
	}
	return w.a.Profile
}

// endpoint is the URL the connection step will check.
func (w *wizard) endpoint() string {
	if w.a.Dest == "custom" {
		return strings.TrimSpace(w.a.Endpoint)
	}
	return cloudEndpoint
}

// load copies an existing profile's settings into the answers when the
// profile being updated changes.
func (w *wizard) load() {
	if w.isNew() || w.loaded == w.a.Profile {
		return
	}
	for _, p := range w.store.Profiles() {
		if p.Name != w.a.Profile {
			continue
		}
		w.loaded = p.Name
		if p.Endpoint == cloudEndpoint {
			w.a.Dest, w.a.Endpoint = "cloud", ""
		} else {
			w.a.Dest, w.a.Endpoint = "custom", p.Endpoint
		}
		w.a.Auth, w.a.Token = p.Auth, p.Token
		w.a.Output, w.a.Color, w.a.Org = p.Output, p.Color, p.Org
		return
	}
}

// complete is called when the current step's form is submitted. It returns
// the fake operation to run, if any ("check" or "connect"); otherwise the
// wizard has already moved on.
func (w *wizard) complete() string {
	w.err = nil
	switch w.step {
	case stepProfile:
		w.load()
		w.advance()
	case stepConnection:
		w.busy = "check"
		return "check"
	case stepReview:
		if !w.a.Connect {
			w.a.Connect = true
			w.back()
			return ""
		}
		w.busy = "connect"
		return "connect"
	default:
		w.advance()
	}
	return ""
}

// finish records the result of a fake check or connect.
func (w *wizard) finish(op string, err error) {
	w.busy = ""
	if err != nil {
		w.err = err
		return
	}
	if op == "connect" {
		path, serr := w.store.Save(w.profile())
		if serr != nil {
			w.err = serr
			return
		}
		w.saved = path
	}
	w.advance()
}

func (w *wizard) advance() {
	if w.step < stepDone {
		w.done[w.step] = true
		w.step++
	}
}

// back goes to the previous screen. It does nothing on the first step, on
// the done screen, or while a check is running.
func (w *wizard) back() bool {
	if w.busy != "" || w.step == stepProfile || w.step == stepDone {
		return false
	}
	w.err = nil
	w.step--
	return true
}

func (w *wizard) profile() Profile {
	p := Profile{
		Name:     w.profileName(),
		Endpoint: w.endpoint(),
		Auth:     w.a.Auth,
		Output:   w.a.Output,
		Color:    w.a.Color,
		Org:      strings.TrimSpace(w.a.Org),
	}
	if w.a.Auth == "token" {
		p.Token = w.a.Token
	}
	return p
}

// summary is a done step's answer, shown under it in the step column.
func (w *wizard) summary(s step) string {
	switch s {
	case stepProfile:
		if w.isNew() {
			return "new: " + w.profileName()
		}
		return w.a.Profile
	case stepConnection:
		if w.a.Dest == "custom" {
			return w.endpoint()
		}
		return "Quayside Cloud"
	case stepAccess:
		if w.a.Auth == "token" {
			return "API token"
		}
		return "browser sign-in"
	case stepPreferences:
		return fmt.Sprintf("%s · %s · %s", w.a.Output, w.a.Color, w.a.Org)
	case stepReview:
		return "connected"
	}
	return ""
}

// title is the header's right-hand side: "Update staging-eu · Set up · 2 of 5".
func (w *wizard) title() string {
	var what string
	if w.isNew() {
		what = "New profile"
		if n := w.profileName(); n != "" {
			what = "Create " + n
		}
	} else {
		what = "Update " + w.a.Profile
	}
	if w.step == stepDone {
		return what + " · Set up · done"
	}
	return fmt.Sprintf("%s · Set up · %d of %d", what, int(w.step)+1, len(stepNames))
}

func maskToken(t string) string {
	if len(t) <= 4 {
		return strings.Repeat("•", len(t))
	}
	return strings.Repeat("•", 8) + t[len(t)-4:]
}

// review lists every answer for the last step.
func (w *wizard) review() []string {
	access := "Sign in with the browser"
	if w.a.Auth == "token" {
		access = "API token " + maskToken(w.a.Token)
	}
	org := strings.TrimSpace(w.a.Org)
	if org == "" {
		org = "(none)"
	}
	return []string{
		"Profile:      " + w.profileName(),
		"Endpoint:     " + w.endpoint(),
		"Access:       " + access,
		"Output:       " + w.a.Output,
		"Color:        " + w.a.Color,
		"Organisation: " + org,
	}
}
