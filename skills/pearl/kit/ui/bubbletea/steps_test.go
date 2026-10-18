package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The step row never exceeds its width: it sheds connectors, then names.
func TestStepsRowFits(t *testing.T) {
	th := testTheme(t, false)
	s := Steps{Names: []string{"Cluster", "Sign-in", "Alerts", "Review", "Connect", "Finish"}, Cur: 2}
	for w := 10; w <= 90; w++ {
		if got := lipgloss.Width(s.Row(th, w)); got > w {
			t.Fatalf("row is %d cells at width %d", got, w)
		}
	}
	if !strings.Contains(ansi.Strip(s.Row(th, 30)), "Alerts") {
		t.Fatal("a narrow row must keep the current step's name")
	}
	if s.Count() != "3/6 Alerts" {
		t.Fatalf("count %q", s.Count())
	}
}

// Summary values are cut by cells with an ellipsis, never wrapped.
func TestSummaryTruncates(t *testing.T) {
	th := testTheme(t, false)
	rows := []SummaryRow{{Label: "Feedback channel", Value: "notifications-setup-feedback-eu-west", Set: true}, {Label: "Namespace", Value: "default"}}
	out := Summary(th, "Summary", Tag(th, "muted", "DRAFT"), rows, 24)
	for _, r := range out {
		if lipgloss.Width(r) > 24 {
			t.Fatalf("row %q is wider than 24", ansi.Strip(r))
		}
	}
	if !strings.Contains(ansi.Strip(strings.Join(out, "\n")), "notifications-setup-fee…") {
		t.Fatalf("value not cut with an ellipsis:\n%s", ansi.Strip(strings.Join(out, "\n")))
	}
	if !strings.Contains(ansi.Strip(out[5]), "default") {
		t.Fatalf("an unset row must say default: %q", ansi.Strip(out[5]))
	}
	if got := lipgloss.Width(SummaryStrip(th, rows, 30)); got > 30 {
		t.Fatalf("strip is %d cells", got)
	}
}
