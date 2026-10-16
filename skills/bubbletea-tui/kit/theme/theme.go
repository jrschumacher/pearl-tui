// Package theme turns a brand theme file (theme.json) into the styles every
// screen uses. Screens never hold a hex value or a glyph literal: they ask the
// theme. A brand changes the JSON; the screens stay the same.
//
// Three layers, all in the JSON:
//
//	roles        the palette by role name, a dark and a light set
//	glyphs       the brand's glyph vocabulary (rule, focus bar, cursor, status…)
//	expressions  how each idiom element is drawn (header, selection, buttons…)
//
// Missing roles are derived (see derive.go), missing glyphs and expressions
// fall back to defaults, so a partial theme still renders.
package theme

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"reflect"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

// Roles is the palette by role. Hex strings in the JSON.
type Roles struct {
	Primary   string `json:"primary"`   // the brand hue: focus, current, selected rows, header rule
	Secondary string `json:"secondary"` // second hue: primary buttons, gradient end, subtitles
	Tertiary  string `json:"tertiary"`  // prompts, cursors, input carets
	Accent    string `json:"accent"`    // section titles, headings; reads as "heading", never "alert"

	Bg        string `json:"bg"`        // the background the brand expects
	BgPanel   string `json:"bgPanel"`   // one step up: recessed/raised panels (tool output, cards)
	BgSubtle  string `json:"bgSubtle"`  // two steps: unselected buttons, inactive chips, dim bars
	BgOverlay string `json:"bgOverlay"` // three steps: popovers, completions, hover

	Fg         string `json:"fg"`         // body text
	FgSoft     string `json:"fgSoft"`     // long-form prose (slightly softer than fg)
	FgMuted    string `json:"fgMuted"`    // secondary text: keys, done steps, metadata
	FgSubtle   string `json:"fgSubtle"`   // tertiary text: hints, placeholders (never the only carrier of meaning)
	FgSelected string `json:"fgSelected"` // text on a primary-filled selected row

	Border      string `json:"border"`      // rules: barely lighter than bg
	BorderFocus string `json:"borderFocus"` // the one dialog's border

	Success string `json:"success"`
	Error   string `json:"error"`
	Warning string `json:"warning"`
	Info    string `json:"info"`

	OnPrimary   string `json:"onPrimary"`   // text on primary fill
	OnSecondary string `json:"onSecondary"` // text on secondary fill
	OnError     string `json:"onError"`
	OnWarning   string `json:"onWarning"`
	OnSuccess   string `json:"onSuccess"`

	White string `json:"white,omitempty"` // legacy themes: used as OnPrimary/OnSecondary when those are missing
}

// Glyphs is the brand's glyph vocabulary. Every glyph has an ASCII fallback
// (see ASCII) used when the theme is resolved with ascii=true.
type Glyphs struct {
	HeaderRule string   `json:"headerRule"` // fills the header after the wordmark: ╱ ─ ═ · ━ ╌ ~ (or "")
	Rule       string   `json:"rule"`       // section rules: ─ ━ ╌ ┄ ═ ·
	FocusBar   string   `json:"focusBar"`   // left marker of the focused block: ▌ ┃ │ ▎ ❯
	BlurBar    string   `json:"blurBar"`    // the same position, unfocused ("" = padding only)
	Cursor     string   `json:"cursor"`     // selection cursor in lists/selects: › ❯ > ▸ →
	Prompt     string   `json:"prompt"`     // input prompt: > $ ❯ ›
	Done       string   `json:"done"`       // ✓ ✔ ■ ◆ x
	Current    string   `json:"current"`    // ● ◆ ▶ >
	Todo       string   `json:"todo"`       // ○ ◇ · □ -
	Fail       string   `json:"fail"`       // × ✗ ✕ !
	Warn       string   `json:"warn"`       // ▲ ! ◬ (status: degraded) — must differ in shape from Current and Fail
	Pending    string   `json:"pending"`    // ● ◌ ·
	Bullet     string   `json:"bullet"`     // • · ▪ -
	Sep        string   `json:"sep"`        // separator in detail lines: · • │ /
	Arrow      string   `json:"arrow"`      // → › ▸
	Ellipsis   string   `json:"ellipsis"`   // …
	RadioOn    string   `json:"radioOn"`    // ◉ ● (•)
	RadioOff   string   `json:"radioOff"`   // ○ ◯ ( )
	CheckOn    string   `json:"checkOn"`    // ☑ [x] ■
	CheckOff   string   `json:"checkOff"`   // ☐ [ ] □
	TreeMid    string   `json:"treeMid"`    // ├──
	TreeEnd    string   `json:"treeEnd"`    // ╰── └──
	TreeRail   string   `json:"treeRail"`   // │
	Thumb      string   `json:"thumb"`      // scrollbar thumb ┃ █
	Track      string   `json:"track"`      // scrollbar track │ ░ (or " ")
	Spinner    []string `json:"spinner"`    // frames
	FieldA     string   `json:"fieldA"`     // backdrop field bright glyph ◆ ▲ ╱
	FieldB     string   `json:"fieldB"`     // backdrop field subtle glyph ◇ △
}

// Expressions are the brand's choices for how each element is drawn. These are
// the axes that make two themes read as two products, not one app recoloured.
type Expressions struct {
	Header       string `json:"header"`       // rule | bar | minimal | stamp | banner
	Wordmark     string `json:"wordmark"`     // gradient | solid | inverse | tracked | bracket | upper
	SectionTitle string `json:"sectionTitle"` // accent | primary | muted-upper | fg
	TitleCase    string `json:"titleCase"`    // sentence | upper | lower
	Selection    string `json:"selection"`    // bar | fill | caret | inverse  (how a selected/focused row is marked)
	Button       string `json:"button"`       // pill | bracket | underline | inverse
	Tag          string `json:"tag"`          // filled | bracket | bold
	Dialog       string `json:"dialog"`       // rounded | square | double | heavy | none (none = filled panel)
	DialogTitle  string `json:"dialogTitle"`  // rule | plain | bar   (rule = title + headerRule glyph run in a gradient)
	Tabs         string `json:"tabs"`         // underline | pill | bracket | slash
	Footer       string `json:"footer"`       // keys | bar | rule
	Keys         string `json:"keys"`         // inline | boxed
	Density      string `json:"density"`      // airy | compact
	Backdrop     string `json:"backdrop"`     // none | field
	Working      string `json:"working"`      // spinner | scramble   (the hero "working" indicator; scramble is Crush's)
	Scrim        string `json:"scrim"`        // dim | none   (dim the screen under a dialog)
	PaintBg      bool   `json:"paintBg"`      // own the terminal background (tea.View.BackgroundColor)
	Gutter       int    `json:"gutter"`       // side margin in cells (default 2)
	Measure      int    `json:"measure"`      // max prose width (default 100)
}

// Wordmark is an optional multi-row wordmark (scripts/wordmark.py output).
type Wordmark struct {
	Rows  []string `json:"rows"`
	Style string   `json:"style"` // half | shade | ascii (shade gets three tones)
}

// File is the theme JSON.
type File struct {
	Name        string           `json:"name"`
	Brand       string           `json:"brand"` // the CLI's own name; flows into headers, paths, env vars
	Read        string           `json:"read"`  // one paragraph: why these choices (see references/brand.md)
	Roles       map[string]Roles `json:"roles"` // "dark", "light"
	Glyphs      Glyphs           `json:"glyphs"`
	Expressions Expressions      `json:"expressions"`
	Wordmark    Wordmark         `json:"wordmark"`
}

// Load reads a theme file. Every role colour must be "#RGB" or "#RRGGBB" (see
// CheckHex), or empty to be derived: Lip Gloss draws anything else as a different
// colour or none, so a theme that would render wrong fails here instead.
func Load(path string) (File, error) {
	var f File
	b, err := os.ReadFile(path)
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("%s: %w", path, err)
	}
	modes := make([]string, 0, len(f.Roles))
	for m := range f.Roles {
		modes = append(modes, m)
	}
	sort.Strings(modes)
	for _, m := range modes {
		if err := f.Roles[m].check(); err != nil {
			return f, fmt.Errorf("%s: roles.%s.%w", path, m, err)
		}
	}
	return f, nil
}

// CheckHex reports whether s is a colour Lip Gloss draws as written: "#RGB" or
// "#RRGGBB", either case. Lip Gloss reads a string without "#" as an ANSI index
// ("12"), a packed integer ("123456") or no colour at all ("6B50FF"), and any
// other length after "#" as no colour, so all of those are errors here.
func CheckHex(s string) error {
	if s == "" {
		return fmt.Errorf("empty colour: write #RGB or #RRGGBB")
	}
	body, hasHash := strings.CutPrefix(s, "#")
	digits := (len(body) == 3 || len(body) == 6) && strings.Trim(body, "0123456789abcdefABCDEF") == ""
	switch {
	case hasHash && digits:
		return nil
	case !strings.Contains(s, "#"):
		fix := "#RGB or #RRGGBB"
		if digits {
			fix = fmt.Sprintf("%q", "#"+s)
		}
		return fmt.Errorf("%q has no #: Lip Gloss reads it as an ANSI index or no colour; write %s", s, fix)
	}
	return fmt.Errorf("%q is not #RGB or #RRGGBB", s)
}

// check validates every non-empty role, naming the role by its JSON key.
func (r Roles) check() error {
	v, t := reflect.ValueOf(r), reflect.TypeOf(r)
	for i := range t.NumField() {
		s := v.Field(i).String()
		if s == "" {
			continue
		}
		if err := CheckHex(s); err != nil {
			key, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	return nil
}

// Palette is a resolved role set as colours.
type Palette struct {
	Primary, Secondary, Tertiary, Accent                  color.Color
	Bg, BgPanel, BgSubtle, BgOverlay                      color.Color
	Fg, FgSoft, FgMuted, FgSubtle, FgSelected             color.Color
	Border, BorderFocus                                   color.Color
	Success, Error, Warning, Info                         color.Color
	OnPrimary, OnSecondary, OnError, OnWarning, OnSuccess color.Color
}

// Theme is a resolved theme: one mode, one glyph set, ready-made styles.
type Theme struct {
	File
	Dark  bool
	ASCII bool
	P     Palette
	G     Glyphs
	X     Expressions
	S     Styles
}

// Resolve picks the dark or light roles, fills gaps, and builds every style.
// Call it on startup and again when tea.BackgroundColorMsg arrives.
func Resolve(f File, dark, ascii bool) *Theme {
	r, ok := f.Roles["dark"]
	if !dark {
		if l, ok2 := f.Roles["light"]; ok2 {
			r, ok = l, true
		}
	}
	if !ok {
		r = Default().Roles["dark"]
	}
	t := &Theme{File: f, Dark: dark, ASCII: ascii}
	t.P = derive(r, dark)
	t.G = glyphDefaults(f.Glyphs)
	if ascii {
		t.G = ASCII(t.G)
	}
	t.X = exprDefaults(f.Expressions)
	t.S = build(t)
	return t
}

// AppName is the brand name the theme declares.
func (t *Theme) AppName() string {
	if t.Brand != "" {
		return t.Brand
	}
	return "app"
}

// Hex parses "#rrggbb" or "#rgb" into a colour (nil for ""). It does not
// validate: see CheckHex.
func Hex(s string) color.Color {
	if s == "" {
		return nil
	}
	return lipgloss.Color(s)
}
