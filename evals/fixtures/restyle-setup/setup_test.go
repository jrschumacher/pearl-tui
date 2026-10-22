package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// instant is a Service with no delays and switchable failures.
type instant struct {
	failCheck, failConnect bool
	checked                []string
}

func (s *instant) Check(ep string) error {
	s.checked = append(s.checked, ep)
	if s.failCheck {
		return errUnreachable
	}
	return nil
}
func (s *instant) DeviceCode() (string, string) {
	return "TEST-CODE", "https://quayside.example/device"
}
func (s *instant) Connect(Profile) error {
	if s.failConnect {
		return errRefused
	}
	return nil
}

func TestDefaultsLoadCurrentProfile(t *testing.T) {
	w := newWizard(newFakeStore())
	if w.a.Profile != "staging-eu" || w.a.Org != "tidewater-eu" || w.a.Dest != "cloud" {
		t.Fatalf("defaults not loaded from staging-eu: %+v", w.a)
	}
	if got, want := w.title(), "Update staging-eu · Set up · 1 of 5"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
}

func TestSwitchingProfileLoadsItsSettings(t *testing.T) {
	w := newWizard(newFakeStore())
	w.a.Profile = "local-dev"
	w.complete()
	if w.step != stepConnection {
		t.Fatalf("step = %d, want connection", w.step)
	}
	if w.a.Dest != "custom" || w.endpoint() != "http://localhost:8640/v2" || w.a.Output != "json" {
		t.Fatalf("local-dev not loaded: %+v", w.a)
	}
}

func TestWalkThroughEveryStep(t *testing.T) {
	store := newFakeStore()
	w := newWizard(store)
	w.a.Profile = newProfile
	w.a.NewName = "prod-us"

	if op := w.complete(); op != "" || w.step != stepConnection {
		t.Fatalf("profile: op=%q step=%d", op, w.step)
	}
	if op := w.complete(); op != "check" || w.busy != "check" {
		t.Fatalf("connection should start a check, got op=%q busy=%q", op, w.busy)
	}
	if w.back() {
		t.Fatal("back must do nothing while a check runs")
	}
	w.finish("check", nil)
	if w.step != stepAccess || w.busy != "" {
		t.Fatalf("after check: step=%d busy=%q", w.step, w.busy)
	}
	w.a.Auth, w.a.Token = "token", "qs_live_1234abcd"
	w.complete()
	w.a.Output, w.a.Color, w.a.Org = "yaml", "never", "harbor"
	w.complete()
	if w.step != stepReview {
		t.Fatalf("step = %d, want review", w.step)
	}
	if op := w.complete(); op != "connect" {
		t.Fatalf("review should connect, got %q", op)
	}
	w.finish("connect", nil)
	if w.step != stepDone || w.saved == "" {
		t.Fatalf("not done: step=%d saved=%q", w.step, w.saved)
	}
	got := store.saved[len(store.saved)-1]
	want := Profile{Name: "prod-us", Endpoint: cloudEndpoint, Auth: "token", Token: "qs_live_1234abcd", Output: "yaml", Color: "never", Org: "harbor"}
	if got != want {
		t.Fatalf("saved %+v, want %+v", got, want)
	}
	for i := range stepNames {
		if !w.done[i] {
			t.Fatalf("step %d not marked done", i)
		}
	}
	if w.back() {
		t.Fatal("back must do nothing on the done screen")
	}
}

func TestCheckFailureStaysOnConnection(t *testing.T) {
	w := newWizard(newFakeStore())
	w.complete()
	w.complete()
	w.finish("check", errUnreachable)
	if w.step != stepConnection || !errors.Is(w.err, errUnreachable) {
		t.Fatalf("step=%d err=%v", w.step, w.err)
	}
	if op := w.complete(); op != "check" || w.err != nil {
		t.Fatalf("retry: op=%q err=%v", op, w.err)
	}
}

func TestConnectFailureStaysOnReview(t *testing.T) {
	store := newFakeStore()
	w := newWizard(store)
	w.step = stepReview
	w.complete()
	w.finish("connect", errRefused)
	if w.step != stepReview || !errors.Is(w.err, errRefused) || len(store.saved) != 0 {
		t.Fatalf("step=%d err=%v saved=%d", w.step, w.err, len(store.saved))
	}
}

func TestReviewBackGoesToPreferences(t *testing.T) {
	w := newWizard(newFakeStore())
	w.step = stepReview
	w.a.Connect = false
	if op := w.complete(); op != "" || w.step != stepPreferences || !w.a.Connect {
		t.Fatalf("op=%q step=%d connect=%v", op, w.step, w.a.Connect)
	}
}

func TestBackKeepsAnswers(t *testing.T) {
	w := newWizard(newFakeStore())
	w.complete()
	w.a.Dest, w.a.Endpoint = "custom", "https://q.internal/v2"
	if !w.back() || w.step != stepProfile {
		t.Fatal("back from connection should reach profile")
	}
	if w.back() {
		t.Fatal("back on the first step must do nothing")
	}
	w.complete()
	if w.endpoint() != "https://q.internal/v2" {
		t.Fatalf("endpoint lost going back: %q", w.endpoint())
	}
}

func TestModelRunsCheckAndAdvances(t *testing.T) {
	svc := &instant{}
	m := newModel(svc, newFakeStore())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	m.w.complete() // profile
	m.rebuild()
	m.w.complete() // connection: starts the check
	msg := m.run("check")()
	m.Update(msg)
	if m.w.step != stepAccess || m.form == nil {
		t.Fatalf("step=%d form=%v", m.w.step, m.form)
	}
	if len(svc.checked) != 1 || svc.checked[0] != cloudEndpoint {
		t.Fatalf("checked %v", svc.checked)
	}

	svc.failCheck = true
	m.w.back()
	m.w.complete()
	m.Update(m.run("check")())
	if m.w.step != stepConnection || m.w.err == nil {
		t.Fatalf("failed check: step=%d err=%v", m.w.step, m.w.err)
	}
}

func TestModelKeys(t *testing.T) {
	m := newModel(&instant{}, newFakeStore())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.w.complete()
	m.rebuild()
	m.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	if m.w.step != stepProfile {
		t.Fatalf("ctrl+b: step=%d", m.w.step)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.cancelled || cmd == nil {
		t.Fatal("esc should cancel and quit")
	}
}

func TestHeadlessToken(t *testing.T) {
	store := newFakeStore()
	var out, errb bytes.Buffer
	code := run([]string{"setup", "--token", "qs_live_9f3a7c21", "--org", "harbor"}, &out, &errb, &instant{}, store)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "Profile staging-eu saved") || strings.Contains(out.String(), "9f3a7c21") {
		t.Fatalf("output:\n%s", out.String())
	}
	if p := store.saved[0]; p.Token != "qs_live_9f3a7c21" || p.Org != "harbor" || p.Auth != "token" {
		t.Fatalf("saved %+v", p)
	}

	out.Reset()
	errb.Reset()
	code = run([]string{"setup", "--token", "qs_x"}, &out, &errb, &instant{failConnect: true}, newFakeStore())
	if code != 1 || !strings.Contains(errb.String(), "refused") {
		t.Fatalf("refused: exit %d, stderr %q", code, errb.String())
	}

	code = run([]string{"setup", "--token", "qs_x", "--output", "xml"}, &out, &errb, &instant{}, newFakeStore())
	if code != 1 {
		t.Fatalf("bad --output accepted")
	}
}
