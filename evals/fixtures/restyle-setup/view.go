package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	accent = lipgloss.Color("#6C8EBF")
	muted  = lipgloss.Color("#8A8F98")

	appName     = lipgloss.NewStyle().Bold(true).Foreground(accent)
	headerRight = lipgloss.NewStyle().Foreground(muted)
	stepCurrent = lipgloss.NewStyle().Underline(true).Bold(true)
	stepDoneSt  = lipgloss.NewStyle().Foreground(accent)
	stepTodo    = lipgloss.NewStyle().Foreground(muted)
	answerSt    = lipgloss.NewStyle().Foreground(muted)
	errorSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("#D9534F"))
	buttonSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(accent).Padding(0, 2)
	footerSt    = lipgloss.NewStyle().Foreground(muted).Padding(0, 1)
)

const (
	sidebarWidth = 30
	narrowWidth  = 100
	// rows under the form for an error and the button
	statusRows = 3
)

const footerKeys = "enter next  arrows choose  shift+tab back  ctrl+b Back screen  esc cancel  ctrl+c quit"

func (m *model) narrow() bool { return m.width < narrowWidth }

func (m *model) footer() string {
	return footerSt.Width(m.width).Render(footerKeys)
}

// bodyHeight is everything between the header and the footer.
func (m *model) bodyHeight() int {
	h := m.height - 2 - 1 - lipgloss.Height(m.footer())
	return max(h, 5)
}

func (m *model) formWidth() int {
	if m.narrow() {
		return max(m.width-4, 20)
	}
	return max(m.width-sidebarWidth-6, 20)
}

func (m *model) formHeight() int {
	h := m.bodyHeight() - statusRows
	if m.narrow() {
		h -= 3 // breadcrumb, answers, blank
	}
	return max(h, 3)
}

func (m *model) header() string {
	left := appName.Render("quayside")
	right := headerRight.Render(m.w.title())
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	return " " + left + strings.Repeat(" ", gap) + right
}

func (m *model) stepLabel(i int) string {
	s := step(i)
	switch {
	case s == m.w.step:
		return stepCurrent.Render(fmt.Sprintf("%d %s", i+1, stepNames[i]))
	case m.w.done[i] || m.w.step == stepDone:
		return stepDoneSt.Render("✓ " + stepNames[i])
	default:
		return stepTodo.Render(fmt.Sprintf("%d %s", i+1, stepNames[i]))
	}
}

// sidebar spreads the five steps evenly over the body height.
func (m *model) sidebar(h int) string {
	rows := make([]string, h)
	for i := range stepNames {
		r := i * h / len(stepNames)
		rows[r] = m.stepLabel(i)
		if m.w.done[i] && r+1 < h {
			rows[r+1] = "  " + answerSt.Render(m.w.summary(step(i)))
		}
	}
	return lipgloss.NewStyle().Width(sidebarWidth).Render(strings.Join(rows, "\n"))
}

// breadcrumb replaces the sidebar on narrow terminals.
func (m *model) breadcrumb() string {
	var parts, answers []string
	for i := range stepNames {
		parts = append(parts, m.stepLabel(i))
		if m.w.done[i] {
			answers = append(answers, m.w.summary(step(i)))
		}
	}
	return strings.Join(parts, " / ") + "\n" + answerSt.Render(strings.Join(answers, "  "))
}

func (m *model) main() string {
	w := m.w
	switch {
	case w.step == stepDone:
		return strings.Join([]string{
			stepDoneSt.Render("✓ Connected"),
			"",
			fmt.Sprintf("Profile %s saved to %s.", w.profileName(), w.saved),
			"Try it: quayside status",
			"",
			answerSt.Render("Press enter to exit."),
		}, "\n")
	case w.busy == "check":
		return "Checking connection to " + w.endpoint() + " ..."
	case w.busy == "connect":
		return "Connecting to " + w.endpoint() + " ..."
	}

	var b strings.Builder
	b.WriteString(m.form.View())
	b.WriteString("\n")
	if w.err != nil {
		b.WriteString(errorSt.Render("Error: " + w.err.Error()))
	}
	b.WriteString("\n\n")
	if w.step == stepConnection {
		b.WriteString(buttonSt.Render("Check connection & continue"))
	}
	return b.String()
}

func (m *model) View() tea.View {
	if m.width == 0 || m.form == nil && m.w.step != stepDone {
		return tea.NewView("")
	}
	var body string
	if m.narrow() {
		body = lipgloss.NewStyle().Padding(0, 2).Render(m.breadcrumb() + "\n\n" + m.main())
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().PaddingLeft(1).Render(m.sidebar(m.bodyHeight())),
			"   ",
			m.main(),
		)
	}
	body = lipgloss.NewStyle().Height(m.bodyHeight()).MaxHeight(m.bodyHeight()).Render(body)

	v := tea.NewView(m.header() + "\n\n" + body + "\n\n" + m.footer())
	v.AltScreen = true
	return v
}
