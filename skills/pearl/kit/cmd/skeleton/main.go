// Command skeleton is the smallest app on the kit's shape: a list, an item's page, a
// confirm and the keys overlay. Copy it as cmd/<app> and grow it into yours; this doc
// comment is the first thing to rewrite. The flags are the harness every app keeps:
//
//	go run ./cmd/<app>                                interactive, the embedded theme
//	go run ./cmd/<app> -theme themes/x.json           another theme
//	go run ./cmd/<app> -golden -w 80 -h 24            every state as text; fails on a row ≠ w cells
//	go run ./cmd/<app> -state list -w 120 -h 40       one state as ANSI
//	go run ./cmd/<app> -out frames -sizes 80x24,120x40 [-light] [-ascii] [-profile ansi]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"

	// your module's themes/ package (kit/README.md rewrites this path): its embedded
	// JSON is the brand the binary draws with no -theme flag
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/themes"
)

// states are every state your app can show, states[0] the main screen. A state you
// can't render here is one nobody reviewed: add each new screen, step and dialog.
var states = []string{"list", "detail", "confirm", "help", "toosmall"}

// overlays are the states that open a dialog ("help" stays the keys overlay's name);
// scrolls those whose body may scroll at the floor; overlayPages the pages that draw a
// focus, each of which the tests open every overlay over. unjudgedPages is for a Huh
// form's page alone (kit/README.md): leave it empty.
var (
	overlays      = []string{"confirm", "help"}
	scrolls       = []string{"help"}
	overlayPages  = []string{"list", "detail"}
	unjudgedPages = map[string]string{}
)

// openOverlay opens overlay s over whatever page m shows, as its key would.
func (m *app) openOverlay(s string) {
	switch s {
	case "confirm":
		m.ov, m.btn = ovConfirm, btnCancel
	case "help":
		m.helpFrom, m.ov = ovNone, ovHelp
	}
}

// setState rebuilds the model for a named state with the keys a user would press, so
// a state can't drift from the behaviour that reaches it.
func (m *app) setState(s string) {
	profile := m.profile
	*m = *newApp(m.file, m.t.Dark, m.ascii)
	m.profile = profile
	m.resolve(m.t.Dark)
	switch s {
	case "detail":
		m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	case "confirm", "help":
		m.openOverlay(s)
	case "toosmall":
		m.w, m.h = frameSize(s, m.w, m.h)
	}
}

func main() {
	themePath := flag.String("theme", "", "theme JSON (default: the embedded theme, themes.Default)")
	light := flag.Bool("light", false, "use the light role set")
	ascii := flag.Bool("ascii", false, "ASCII glyphs (TERM=linux)")
	state := flag.String("state", "", "render one state as ANSI and exit")
	golden := flag.Bool("golden", false, "render every state as plain text and verify sizes")
	out := flag.String("out", "", "write every state at every size to this directory")
	sizes := flag.String("sizes", "80x24,120x40", "sizes for -out")
	w := flag.Int("w", 80, "width")
	h := flag.Int("h", 24, "height")
	profile := flag.String("profile", "truecolor", "truecolor | ansi256 | ansi | ascii (NO_COLOR=1: ascii)")
	flag.Parse()

	f, err := embeddedTheme()
	if *themePath != "" {
		f, err = theme.Load(*themePath)
	}
	exitIf(err, 1)
	if *state == "" && !*golden && *out == "" {
		_, err := tea.NewProgram(newApp(f, !*light, *ascii || os.Getenv("TERM") == "linux")).Run()
		exitIf(err, 1)
		return
	}
	explicit := false
	flag.Visit(func(fl *flag.Flag) { explicit = explicit || fl.Name == "profile" })
	prof, err := ui.RenderProfile(*profile, explicit, os.Getenv("NO_COLOR"))
	exitIf(err, 2)
	m := newApp(f, !*light, *ascii)
	m.profile = prof
	frame := func(s string, w, h int) string {
		m.setState(s)
		c := m.render(frameSize(s, w, h))
		c.PaintBackground()
		return ui.Downsample(c.String(), prof)
	}
	if *state != "" {
		fmt.Println(frame(*state, *w, *h))
		return
	}
	if *golden { // every state once, at -w×-h, as text on stdout
		*sizes = fmt.Sprintf("%dx%d", *w, *h)
	} else {
		exitIf(os.MkdirAll(*out, 0o755), 1)
	}
	bad := 0
	for _, sz := range strings.Split(*sizes, ",") {
		var sw, sh int
		fmt.Sscanf(strings.TrimSpace(sz), "%dx%d", &sw, &sh)
		for _, s := range states {
			txt := frame(s, sw, sh)
			fw, fh := frameSize(s, sw, sh)
			if err := ui.CheckSize(ansi.Strip(txt), s, fw, fh); err != nil {
				fmt.Fprintln(os.Stderr, "FAIL", err)
				bad++
			}
			if *golden {
				fmt.Printf("=== %s %dx%d\n%s\n", s, fw, fh, ansi.Strip(txt))
			} else {
				exitIf(os.WriteFile(filepath.Join(*out, frameName(s, sw, sh)), []byte(txt+"\n"), 0o644), 1)
			}
		}
	}
	if bad > 0 {
		os.Exit(1)
	}
}

// embeddedTheme is the theme the binary carries (themes/embed.go), so the brand draws
// wherever the binary is run from.
func embeddedTheme() (theme.File, error) {
	b, err := themes.Default()
	if err != nil {
		return theme.File{}, err
	}
	return theme.Parse(b)
}

func exitIf(err error, code int) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}

// frameSize is the size a state renders at for a requested w×h: the too-small screen
// is drawn at 50×14, below the app's minimum, whatever was asked.
func frameSize(s string, w, h int) (int, int) {
	if s == "toosmall" {
		return 50, 14
	}
	return w, h
}

// frameName names an -out file by the size the state renders at, which tuilint frames
// reads back as the frame's size.
func frameName(s string, w, h int) string {
	w, h = frameSize(s, w, h)
	return fmt.Sprintf("%s-%dx%d.ansi", s, w, h)
}
