package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"
	"time"

	"tuicap/internal/redact"
	"tuicap/internal/tape"
)

// The package doc comment is the -h text: embed it rather than keep a second copy.
//
//go:embed doc.go
var docSource string

// usage turns doc.go's comment block into plain text.
func usage() string {
	var b strings.Builder
	for _, line := range strings.Split(docSource, "\n") {
		if !strings.HasPrefix(line, "//") {
			break
		}
		line = strings.TrimPrefix(strings.TrimPrefix(line, "//"), " ")
		if strings.HasPrefix(line, "\t") {
			line = "  " + line[1:]
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func main() {
	os.Exit(Main(os.Args[1:], os.Stdin, os.Stderr))
}

type size struct{ w, h int }

func (s size) String() string { return fmt.Sprintf("%dx%d", s.w, s.h) }

type options struct {
	sizes    []size
	tapeName string
	cmds     []tape.Cmd
	out      string
	profile  string
	env      []string // the program's environment
	light    bool
	fg, bg   color.Color
	wait     time.Duration
	timeout  time.Duration
	privacy  bool
	red      *redact.Redactor
	allow    []string // -allow
	raw      bool
	argv     []string
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

// Main runs tuicap and returns its exit status.
func Main(args []string, stdin io.Reader, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--h")) {
		fmt.Fprint(stderr, usage()) // -h is the height, so a bare -h asks for help
		if len(args) == 0 {
			return 2
		}
		return 0
	}
	fs := flag.NewFlagSet("tuicap", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	w := fs.Int("w", 80, "")
	h := fs.Int("h", 24, "")
	sizes := fs.String("sizes", "", "")
	tapePath := fs.String("tape", "", "")
	out := fs.String("out", "", "")
	profile := fs.String("profile", "truecolor", "")
	var envs multi
	fs.Var(&envs, "env", "")
	light := fs.Bool("light", false, "")
	wait := fs.Duration("wait", 5*time.Second, "")
	timeout := fs.Duration("timeout", 2*time.Minute, "")
	level := fs.String("level", "full", "")
	allow := fs.String("allow", "", "")
	raw := fs.Bool("raw", false, "")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stderr, usage())
			return 0
		}
		fmt.Fprintf(stderr, "tuicap: %v (tuicap -h for usage)\n", err)
		return 2
	}
	o, err := configure(fs, stdin, *w, *h, *sizes, *tapePath, *out, *profile, envs, *light, *wait, *timeout, *level, *allow, *raw)
	if err != nil {
		fmt.Fprintf(stderr, "tuicap: %v (tuicap -h for usage)\n", err)
		return 2
	}
	if err := prepareOut(o.out); err != nil {
		fmt.Fprintf(stderr, "tuicap: %v\n", err)
		return 2
	}
	return capture(o, stderr)
}

func configure(fs *flag.FlagSet, stdin io.Reader, w, h int, sizes, tapePath, out, profile string, envs []string,
	light bool, wait, timeout time.Duration, level, allow string, raw bool) (*options, error) {
	o := &options{out: out, profile: profile, light: light, wait: wait, timeout: timeout, raw: raw, argv: fs.Args()}
	if len(o.argv) == 0 {
		return nil, errors.New("no program: put it after --, as in tuicap -tape t.tape -out dir -- ./app")
	}
	if out == "" {
		return nil, errors.New("-out DIR is required")
	}
	if tapePath == "" {
		return nil, errors.New("-tape FILE is required")
	}
	var r io.Reader = stdin
	o.tapeName = "stdin"
	if tapePath != "-" {
		f, err := os.Open(tapePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r, o.tapeName = f, tapePath
	}
	cmds, err := tape.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.tapeName, err)
	}
	if len(cmds) == 0 {
		return nil, fmt.Errorf("%s: the tape has no commands", o.tapeName)
	}
	o.cmds = cmds
	if sizes != "" {
		for _, s := range strings.Split(sizes, ",") {
			sw, sh, err := tape.ParseSize(strings.TrimSpace(s))
			if err != nil {
				return nil, fmt.Errorf("-sizes: %w", err)
			}
			o.sizes = append(o.sizes, size{sw, sh})
		}
	} else {
		if _, _, err := tape.ParseSize(fmt.Sprintf("%dx%d", w, h)); err != nil {
			return nil, fmt.Errorf("-w/-h: %w", err)
		}
		o.sizes = []size{{w, h}}
	}
	switch level {
	case "full":
	case "privacy":
		o.privacy = true
		if raw {
			return nil, errors.New("-raw writes the program's output verbatim: not at -level privacy")
		}
	default:
		return nil, fmt.Errorf("-level %q: want full or privacy", level)
	}
	var extra []string
	if allow != "" {
		extra = strings.Split(allow, ",")
	}
	o.red, o.allow = redact.New(extra...), extra
	colour, ok := profileEnv[profile]
	if !ok {
		return nil, fmt.Errorf("-profile %q: want truecolor, ansi256, ansi or ascii", profile)
	}
	for _, e := range envs {
		if !strings.Contains(e, "=") || strings.HasPrefix(e, "=") {
			return nil, fmt.Errorf("-env %q: want KEY=VALUE", e)
		}
	}
	o.env = childEnv(os.Environ(), colour, envs)
	o.fg, o.bg = color.RGBA{0xd8, 0xd8, 0xd8, 0xff}, color.RGBA{0x16, 0x16, 0x1a, 0xff}
	if light {
		o.fg, o.bg = color.RGBA{0x22, 0x22, 0x22, 0xff}, color.RGBA{0xfa, 0xfa, 0xfa, 0xff}
	}
	return o, nil
}

// profileEnv is what a terminal of each colour depth tells a program.
var profileEnv = map[string][]string{
	"truecolor": {"TERM=xterm-256color", "COLORTERM=truecolor"},
	"ansi256":   {"TERM=xterm-256color"},
	"ansi":      {"TERM=xterm"},
	"ascii":     {"TERM=xterm-256color", "NO_COLOR=1"},
}

// colourVars are dropped from tuicap's own environment: each can change the colour
// depth a program detects (colorprofile reads all of them).
var colourVars = []string{"TERM", "COLORTERM", "NO_COLOR", "CLICOLOR", "CLICOLOR_FORCE", "TTY_FORCE", "TMUX", "STY",
	"WT_SESSION", "TERM_PROGRAM", "TERM_PROGRAM_VERSION", "GOOGLE_CLOUD_SHELL", "COLUMNS", "LINES"}

func childEnv(parent, colour, extra []string) []string {
	m := map[string]string{}
	var order []string
	set := func(kv string) {
		k, v, _ := strings.Cut(kv, "=")
		if _, ok := m[k]; !ok {
			order = append(order, k)
		}
		m[k] = v
	}
	for _, kv := range parent {
		k, _, _ := strings.Cut(kv, "=")
		drop := false
		for _, c := range colourVars {
			drop = drop || k == c
		}
		if !drop {
			set(kv)
		}
	}
	for _, kv := range append(append([]string{}, colour...), extra...) {
		set(kv)
	}
	env := make([]string, 0, len(order))
	for _, k := range order {
		env = append(env, k+"="+m[k])
	}
	return env
}
