// Command showcase is the kit's reference app: one product (an ops console
// with an assistant) drawn entirely from the theme, so the same screens can
// be rendered in any brand. It doubles as the verification harness.
//
//	go run ./cmd/showcase                              interactive (alt screen), the embedded theme
//	go run ./cmd/showcase -theme themes/x.json         another brand
//	go run ./cmd/showcase -first-run                   start with the setup wizard (ACCESSIBLE=1: plain prompts)
//	go run ./cmd/showcase -golden -w 80 -h 24          every state as plain text; fails on any row ≠ W cells
//	go run ./cmd/showcase -state services -w 120 -h 40 one state as truecolor ANSI (for cellshot)
//	go run ./cmd/showcase -out frames -sizes 80x24,120x40 [-profile ansi256] [-light] [-ascii]
//	                                                   every state at every size into frames/*.ansi
//	NO_COLOR=1 go run ./cmd/showcase -out frames       as -profile ascii: no colour, as the terminal would
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
	"tuikit/themes"
	"tuikit/ui"
)

// states are every screen the app can show. A state you can't render here is
// a state nobody reviewed.
var states = []string{"services", "services-toast", "deploys", "palette", "confirm", "help", "deploy",
	"incidents", "assistant", "assistant-busy", "splash", "toosmall",
	"setup-target", "setup-namespace", "setup-invalid", "setup-auth", "setup-alerts", "setup-review", "setup-signin", "setup-failed", "setup-done"}

// setState rebuilds the model from scratch for a named state, using the same
// actions a user would take, so fixtures can't drift from behaviour.
func (m *app) setState(s string) {
	frame, profile := m.frame, m.profile
	*m = *newApp(m.file, m.t.Dark, m.ascii)
	m.frame, m.profile = frame, profile
	m.resolve(m.t.Dark)
	m.screen = scrServices
	switch s {
	case "services-toast":
		m.rollback()
	case "deploys":
		m.screen = scrDeploys
		m.sel = m.svcIndex(m.events[m.depSel].Svc)
	case "palette":
		m.ov, m.query = ovPalette, "ro"
	case "confirm":
		m.ov, m.btn = ovConfirm, 1
	case "help":
		m.ov = ovHelp
	case "deploy":
		m.openDeploy()
	case "incidents":
		m.screen = scrIncidents
	case "assistant":
		m.screen, m.input = scrAssistant, "ship it as v2.4.1, and add a test for the unit parsing"
	case "assistant-busy":
		m.screen = scrAssistant
		m.sent = []string{"ship it as v2.4.1, and add a test for the unit parsing"}
		m.busy, m.busySince = true, frame-240
	case "splash":
		m.screen = scrSplash
	}
	if strings.HasPrefix(s, "setup-") {
		m.setupState(s, frame)
	}
}

// setupState walks the wizard to a named step the way a user would get
// there: earlier steps entered, their values chosen.
func (m *app) setupState(s string, frame int) {
	m.openSetup()
	f := m.setup
	if s == "setup-target" {
		return
	}
	if s == "setup-namespace" || s == "setup-invalid" { // the input focused, typed into: one cursor, one prompt
		f.form.Update(huh.NextField())
		typed := "payments"
		if s == "setup-invalid" {
			typed = "Payments EU" // Huh validates on enter and keeps focus here
		}
		for _, r := range typed {
			f.form.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
		if s == "setup-invalid" {
			f.form.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		}
		return
	}
	f.namespace = "payments"
	steps := map[string]setupStep{"setup-auth": stAuth, "setup-alerts": stAlerts, "setup-review": stReview,
		"setup-signin": stSignin, "setup-failed": stSignin, "setup-done": stDone}
	for st := stAuth; st <= steps[s]; st++ {
		m.goStep(st)
	}
	f.since = frame - 240 // 12 s into the wait
	if s == "setup-failed" {
		f.failed, f.acts = true, ui.NewActions(0, "Retry", "Back")
	}
}

func main() {
	themePath := flag.String("theme", "", "theme JSON (default: the embedded theme, themes.Default)")
	light := flag.Bool("light", false, "use the light role set")
	ascii := flag.Bool("ascii", false, "ASCII glyphs (TERM=linux, --ascii)")
	state := flag.String("state", "", "render one state as ANSI and exit")
	golden := flag.Bool("golden", false, "render every state as plain text and verify sizes")
	out := flag.String("out", "", "write every state at every size to this directory")
	sizes := flag.String("sizes", "80x24,120x40", "sizes for -out")
	w := flag.Int("w", 80, "width")
	h := flag.Int("h", 24, "height")
	frame := flag.Int("frame", 24, "animation frame for renders")
	profile := flag.String("profile", "truecolor", "truecolor | ansi256 | ansi | ascii (NO_COLOR=1: ascii)")
	firstRun := flag.Bool("first-run", false, "start with the setup wizard")
	flag.Parse()

	f, err := embeddedTheme()
	if *themePath != "" {
		f, err = theme.Load(*themePath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *state == "" && !*golden && *out == "" && *firstRun && ui.Accessible() {
		if err := runAccessibleSetup(theme.Resolve(f, !*light, true), os.Stdin, os.Stdout, 2*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *state == "" && !*golden && *out == "" {
		m := newApp(f, !*light, *ascii || os.Getenv("TERM") == "linux")
		m.firstRun = *firstRun
		if _, err := tea.NewProgram(m).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	explicit := false
	flag.Visit(func(fl *flag.Flag) { explicit = explicit || fl.Name == "profile" })
	prof, err := renderProfile(*profile, explicit, os.Getenv("NO_COLOR"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	m := newApp(f, !*light, *ascii)
	m.frame, m.profile = *frame, prof
	render := func(s string, w, h int) string {
		m.setState(s)
		w, h = frameSize(s, w, h)
		c := m.render(w, h)
		c.PaintBackground()
		return downsample(c.String(), prof)
	}
	switch {
	case *state != "":
		fmt.Println(render(*state, *w, *h))
	case *golden:
		bad := 0
		for _, s := range states {
			txt := ansi.Strip(render(s, *w, *h))
			fw, fh := frameSize(s, *w, *h)
			fmt.Printf("=== %s %dx%d\n%s\n", s, fw, fh, txt)
			if err := checkSize(txt, s, *w, *h); err != nil {
				fmt.Fprintln(os.Stderr, "FAIL", err)
				bad++
			}
		}
		if bad > 0 {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "OK %d states at %dx%d\n", len(states), *w, *h)
	default:
		if err := os.MkdirAll(*out, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		bad := 0
		written := map[string]bool{}
		for _, sz := range strings.Split(*sizes, ",") {
			sw, sh := parseSize(sz)
			for _, s := range states {
				name := filepath.Join(*out, frameName(s, sw, sh))
				if written[name] {
					continue
				}
				written[name] = true
				txt := render(s, sw, sh)
				if err := checkSize(ansi.Strip(txt), s, sw, sh); err != nil {
					fmt.Fprintln(os.Stderr, "FAIL", err)
					bad++
				}
				if err := os.WriteFile(name, []byte(txt+"\n"), 0o644); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
		}
		if bad > 0 {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %d frames (%d states × %s) to %s (bg %s)\n", len(written), len(states), *sizes, *out, hex(m.t))
	}
}

// embeddedTheme is the theme the binary carries (themes/embed.go): the brand draws with
// no flag, wherever the binary is run from, instead of a path that breaks when it moves.
func embeddedTheme() (theme.File, error) {
	b, err := themes.Default()
	if err != nil {
		return theme.File{}, err
	}
	return theme.Parse(b)
}

// frameSize is the size a state renders at for a requested w×h: the too-small
// screen is drawn at 50×14, below the app's minimum, whatever was asked.
func frameSize(s string, w, h int) (int, int) {
	if s == "toosmall" {
		return 50, 14
	}
	return w, h
}

// frameName names an -out file by the size the state renders at, which tuilint
// frames reads back as the frame's size; the too-small frame is written once.
func frameName(s string, w, h int) string {
	w, h = frameSize(s, w, h)
	return fmt.Sprintf("%s-%dx%d.ansi", s, w, h)
}

func checkSize(txt, s string, w, h int) error {
	w, h = frameSize(s, w, h)
	rows := strings.Split(txt, "\n")
	if len(rows) != h {
		return fmt.Errorf("%s: %d rows, want %d", s, len(rows), h)
	}
	for i, r := range rows {
		if n := lipgloss.Width(r); n != w {
			return fmt.Errorf("%s: row %d is %d cells, want %d", s, i, n, w)
		}
	}
	return nil
}

var profileNames = map[string]colorprofile.Profile{"truecolor": colorprofile.TrueColor, "ansi256": colorprofile.ANSI256,
	"ansi": colorprofile.ANSI, "ascii": colorprofile.Ascii}

// renderProfile is the profile renders are drawn for: -profile when given, else no
// colour when NO_COLOR is true as strconv.ParseBool reads it (1, t, true…; "yes" or
// "x" is not), which is exactly how colorprofile v0.4.3, and so Bubble Tea's
// detection for the live program, reads it.
func renderProfile(name string, explicit bool, noColor string) (colorprofile.Profile, error) {
	if off, _ := strconv.ParseBool(noColor); !explicit && off {
		return colorprofile.Ascii, nil
	}
	p, ok := profileNames[name]
	if !ok {
		return 0, fmt.Errorf("-profile %q: want truecolor, ansi256, ansi or ascii", name)
	}
	return p, nil
}

// downsample writes a truecolour frame through colorprofile, as Bubble Tea's renderer
// does for a terminal at prof.
func downsample(s string, prof colorprofile.Profile) string {
	if prof == colorprofile.TrueColor {
		return s
	}
	var b bytes.Buffer
	wr := &colorprofile.Writer{Forward: &b, Profile: prof}
	_, _ = wr.WriteString(s)
	return b.String()
}

func parseSize(s string) (int, int) {
	a, b, _ := strings.Cut(strings.TrimSpace(s), "x")
	w, _ := strconv.Atoi(a)
	h, _ := strconv.Atoi(b)
	return w, h
}

func hex(t *theme.Theme) string {
	r, g, b, _ := t.P.Bg.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func itoa(n int) string { return strconv.Itoa(n) }
func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}
