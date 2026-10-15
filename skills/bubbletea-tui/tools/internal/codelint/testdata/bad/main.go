// Command bad trips every tuilint code rule. A want comment is what the Go analyzers
// report on that line; codelint_test.go lists each place tuilint.py differs, and why.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	title  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")) // want "C-HEX"
	accent = lipgloss.Color("212")                                     // want `C-HEX: colour literal lipgloss.Color\("212"\)`
	pink   = "#FF5FAF"                                                 // want "C-HEX"
	links  = []string{"https://charm.land", "#00FF00"}                 // want "C-HEX"
	v1path = "github.com/charmbracelet/bubbletea"
)

const frame = 10 * time.Millisecond

type tickMsg struct{}

type cache map[string][]byte

// ReadFile reads from memory: no I/O.
func (c cache) ReadFile(name string) ([]byte, error) { return c[name], nil }

type model struct {
	width  int
	name   string
	status string
	row    []string
	files  cache
}

func (m model) Init() tea.Cmd {
	fmt.Println("starting") // want "C-PRINT"
	return tea.Batch(
		tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} }), // want "C-FASTTICK: 16ms"
		tea.Every(time.Second/60, func(time.Time) tea.Msg { return tickMsg{} }),     // want `C-FASTTICK: 16.7ms`
		tea.Tick(frame, func(time.Time) tea.Msg { return tickMsg{} }),               // want "C-FASTTICK: 10ms"
		tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} }),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c": // want "C-KEYS: 6 keys"
			return m, tea.Quit
		case "enter":
			time.Sleep(100 * time.Millisecond) // want "C-SLEEP: blocking call time.Sleep in Update:"
		case "r":
			resp, err := http.Get("https://example.com") // want "C-SLEEP: blocking call http.Get in Update:"
			if err == nil {
				resp.Body.Close()
			}
		case "l":
			m.load() // the read happens in load
		case "w":
			return m, func() tea.Msg {
				time.Sleep(time.Second) // a Cmd runs off the update goroutine: fine
				return tickMsg{}
			}
		}
		logf("key %s", msg) // a helper that prints
	}
	return m, nil
}

func (m *model) load() {
	b, _ := os.ReadFile("state.json") // want `C-SLEEP: blocking call os.ReadFile in Update \(via load\)`
	m.name = string(b)
}

func logf(format string, args ...any) {
	fmt.Printf(format, args...) // want `C-PRINT: fmt.Printf in Update \(via logf\)`
}

func (m model) View() tea.View {
	b, _ := os.ReadFile("banner.txt") // want "C-VIEWIO: blocking call os.ReadFile in View"
	cmd := exec.Command("git", "status")
	out, _ := cmd.Output()              // want "C-VIEWIO: blocking call cmd.Output in View"
	logo, _ := m.files.ReadFile("logo") // in memory
	fmt.Fprintln(os.Stderr, "drawing")  // want `C-PRINT: fmt.Fprintln\(os.Stderr\) in View`
	var s strings.Builder
	s.WriteString(title.Render("✨ Done") + string(b) + string(out) + string(logo)) // want "C-EMOJI: emoji '✨'"
	s.WriteString(strings.Repeat(" ", m.width-len(m.name)))                        // want `C-LEN: len\(m.name\) in width maths`
	s.WriteString(strings.Repeat(" ", m.width-len(m.status)))                      // want `C-LEN: len\(m.status\)`
	gap := m.width - len([]rune(m.name))                                           // want `C-LEN: len\(\[\]rune\(m.name\)\).*counts runes`
	w := len(m.name)                                                               // want "C-LEN"
	s.WriteString(strings.Repeat("─", m.width-utf8.RuneCountInString(m.name)))     // want `C-LEN: utf8.RuneCountInString\(m.name\)`
	if len(m.row) > 0 && lipgloss.Width(m.row[0]) < m.width {
		s.WriteString(m.row[0])
	}
	if lipgloss.Width(m.name) > m.width && len(m.name) > 3 {
		s.WriteString("…")
	}
	n := len(m.name)
	_ = gap + w
	s.WriteRune('🚀')           // want "C-EMOJI: emoji '🚀'"
	s.WriteString(`🎉 shipped`) // want "C-EMOJI: emoji '🎉'"
	s.WriteString("⏳ waiting") // want "C-EMOJI: emoji '⏳'"
	s.WriteString("❤️ thanks") // want "C-EMOJI: emoji '❤️'"
	s.WriteString("✓ ok ⚠ careful" + fmt.Sprint(n, accent, pink, links, v1path))
	return tea.NewView(s.String())
}

// splash has unnamed receivers.
type splash struct{}

func (splash) Init() tea.Cmd                       { return nil }
func (splash) Update(tea.Msg) (tea.Model, tea.Cmd) { return splash{}, nil }
func (splash) View() tea.View {
	b, _ := os.ReadFile("splash.txt") // want "C-VIEWIO"
	return tea.NewView(string(b))
}

type config struct{ verbose bool }

func (c config) usage() {
	fmt.Println("usage: bad [-v]") // before the program starts: fine
}

func main() {
	config{}.usage()
	if _, err := tea.NewProgram(model{}).Run(); err != nil { // want "C-SIZE" "C-NOCOLOR"
		fmt.Println(err)
		os.Exit(1)
	}
	_ = splash{}
}
