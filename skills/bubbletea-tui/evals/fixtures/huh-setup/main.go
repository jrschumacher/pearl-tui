// A deliberately untouched Huh v2 first-run wizard: the "before" for the
// restyle-a-huh-wizard eval. It works, and it looks like every Huh demo:
// Huh's default theme and spinner, a hand-rolled summary box, a 2-option
// select used as a confirm, a review that defaults to "Edit settings", and a
// counter that says "Review 5/5" until the end. Don't fix it here.
//
// kestrel is a fictional hosting CLI.
package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
)

type config struct {
	region, project, auth, review string
	telemetry, makeDefault        string
	notify                        bool
}

type model struct {
	cfg   *config
	form  *huh.Form
	step  int
	width int
}

var steps = []string{"Region", "Project", "Sign-in", "Notifications", "Review"}

var summaryBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(40)

func (m *model) build() *huh.Form {
	c := m.cfg
	switch m.step {
	case 0:
		return huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("Region").Options(
				huh.NewOption("us-east (Virginia)", "us-east"),
				huh.NewOption("eu-west (Dublin)", "eu-west"),
				huh.NewOption("ap-south (Mumbai)", "ap-south")).Value(&c.region),
			huh.NewInput().Title("Default project").Placeholder("setup-feedback").Value(&c.project),
		))
	case 1:
		return huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title("How do you want to sign in?").Options(
			huh.NewOption("Browser", "browser"),
			huh.NewOption("API token", "token")).Value(&c.auth)))
	case 2:
		return huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title("Share anonymous usage data?").Options(
			huh.NewOption("No", "no"),
			huh.NewOption("Yes", "yes")).Value(&c.telemetry)))
	case 3:
		return huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Email me when a deploy fails?").Value(&c.notify)))
	default:
		return huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title("Everything look right?").Options(
			huh.NewOption("Edit settings", "edit"),
			huh.NewOption("Continue", "continue"),
			huh.NewOption("Cancel", "cancel")).Value(&c.review)))
	}
}

func (m *model) Init() tea.Cmd {
	m.form = m.build()
	return m.form.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			return m, tea.Quit
		}
	}
	f, cmd := m.form.Update(msg)
	m.form = f.(*huh.Form)
	if m.form.State == huh.StateCompleted {
		if m.step == len(steps)-1 {
			return m, tea.Quit
		}
		m.step++
		m.form = m.build()
		return m, m.form.Init()
	}
	return m, cmd
}

func (m *model) summary() string {
	c := m.cfg
	s := fmt.Sprintf("Region: %s\nProject: %s\nSign-in: %s\nTelemetry: %s\nNotify: %v",
		c.region, c.project, c.auth, c.telemetry, c.notify)
	return summaryBox.Render("Summary\n\n" + s)
}

func (m *model) View() tea.View {
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("kestrel setup · %s %d/%d", steps[m.step], m.step+1, len(steps)))
	body := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(75).Render(m.form.View()), m.summary())
	return tea.NewView(title + "\n\n" + body + "\n")
}

func main() {
	c := &config{region: "us-east", project: "setup-feedback", auth: "browser", telemetry: "no", review: "edit", makeDefault: "yes"}
	if _, err := tea.NewProgram(&model{cfg: c}).Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	if c.review != "continue" {
		fmt.Println("Setup cancelled.")
		return
	}

	// after the review the counter is stuck on the last step
	last := fmt.Sprintf("kestrel setup · Review %d/%d", len(steps), len(steps))
	fmt.Println(last)
	err := spinner.New().Title("Signing in...").Action(func() { time.Sleep(3 * time.Second) }).Run()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println(last)
	var makeDefault bool
	if err := huh.NewConfirm().Title("Make this your default project?").Value(&makeDefault).Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Println("Successfully configured kestrel!")
}
