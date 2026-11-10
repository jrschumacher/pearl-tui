// Command brandapp is CI's stand-in for an app built on the kit (ci.yml, kit import): two
// pages (a list and an item) and one overlay of its own (a confirm) beside help, built on
// the kit's shape with none of the showcase's screens. The kit's harness runs against it
// from the kept harness_test.go, as kit/README.md says; this main.go is the kit's with the
// README's edits made (setup wiring gone, states, overlays, scrolls, overlayPages and
// openOverlay its own).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"example.com/brandapp/themes"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// states are every screen the app can show.
var states = []string{"list", "detail", "confirm", "help", "toosmall"}

// overlays, scrolls, overlayPages and unjudgedPages: the harness tests open each
// overlay over each page.
var overlays = []string{"confirm", "help"}
var scrolls = []string{"help"}
var overlayPages = []string{"list", "detail"}
var unjudgedPages = map[string]string{}

func (m *app) openOverlay(s string) {
	switch s {
	case "confirm":
		m.ov, m.btn = ovConfirm, 1
	case "help":
		m.helpFrom, m.ov = ovNone, ovHelp
	}
}

func (m *app) setState(s string) {
	profile := m.profile
	*m = *newApp(m.file, m.t.Dark, m.ascii)
	m.profile = profile
	m.resolve(m.t.Dark)
	switch s {
	case "detail":
		m.screen = scrDetail
	case "confirm", "help":
		m.openOverlay(s)
	case "toosmall":
		m.w, m.h = frameSize(s, m.w, m.h)
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
	profile := flag.String("profile", "truecolor", "truecolor | ansi256 | ansi | ascii (NO_COLOR=1: ascii)")
	flag.Parse()

	f, err := embeddedTheme()
	if *themePath != "" {
		f, err = theme.Load(*themePath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *state == "" && !*golden && *out == "" {
		m := newApp(f, !*light, *ascii || os.Getenv("TERM") == "linux")
		if _, err := tea.NewProgram(m).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	explicit := false
	flag.Visit(func(fl *flag.Flag) { explicit = explicit || fl.Name == "profile" })
	prof, err := ui.RenderProfile(*profile, explicit, os.Getenv("NO_COLOR"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	m := newApp(f, !*light, *ascii)
	m.profile = prof
	render := func(s string, w, h int) string {
		m.setState(s)
		w, h = frameSize(s, w, h)
		c := m.render(w, h)
		c.PaintBackground()
		return ui.Downsample(c.String(), prof)
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
			if err := ui.CheckSize(txt, s, fw, fh); err != nil {
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
				fw, fh := frameSize(s, sw, sh)
				if err := ui.CheckSize(ansi.Strip(txt), s, fw, fh); err != nil {
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
		fmt.Fprintf(os.Stderr, "wrote %d frames (%d states x %s) to %s (bg %s)\n", len(written), len(states), *sizes, *out, hex(m.t))
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
