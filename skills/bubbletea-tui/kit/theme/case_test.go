package theme

import "testing"

// titleCase lower lowercased whole titles, so both cold monitors in #61 moved SIGTERM
// out of their confirm's title rather than show "send sigterm?". A word in capitals
// keeps them, and a span marked Verbatim keeps its case under lower and upper.
func TestCaseKeepsAcronymsAndVerbatim(t *testing.T) {
	th := func(tc, lc string) *Theme {
		f := Default()
		f.Expressions.TitleCase, f.Expressions.LabelCase = tc, lc
		return Resolve(f, true, false)
	}
	lower, upper, sentence := th("lower", "lower"), th("upper", "upper"), th("sentence", "")
	for _, c := range []struct{ in, want string }{
		{"Send SIGTERM to 13912?", "send SIGTERM to 13912?"},
		{"CPU and Memory", "CPU and memory"},
		{"Merge 3 PRs", "merge 3 PRs"},
		{"SIGTERM's grace period", "SIGTERM's grace period"},
		{"Upgrade to HTTP/2", "upgrade to HTTP/2"},
		{"A New Session", "a new session"},
		{"Open In " + Verbatim("iTerm2"), "open in iTerm2"},
		{"Roll back checkout-api?", "roll back checkout-api?"},
		{"Terminal too small", "terminal too small"},
	} {
		if got := lower.Case(c.in); got != c.want {
			t.Errorf("lower: Case(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := upper.Case("open in " + Verbatim("kubectl")); got != "OPEN IN kubectl" {
		t.Errorf("upper: a Verbatim span is cased: %q", got)
	}
	if got := sentence.Case("Open in " + Verbatim("iTerm2")); got != "Open in iTerm2" {
		t.Errorf("sentence: the Verbatim marks reach the screen: %q", got)
	}
	// labels lower capitals (a header "CPU" is "cpu" under labelCase lower) but keep a Verbatim span
	if got := lower.Label("CPU "+Verbatim("iOS"), "upper"); got != "cpu iOS" {
		t.Errorf("lower: Label = %q, want %q", got, "cpu iOS")
	}
	if got := upper.Label(Verbatim("v2.3.1"), "as-is"); got != "v2.3.1" {
		t.Errorf("upper: Label of a Verbatim span = %q", got)
	}
}
