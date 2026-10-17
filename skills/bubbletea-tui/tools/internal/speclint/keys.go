package speclint

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Modifiers, in the order Ultraviolet's Key.Keystroke prints them.
const (
	modCtrl = 1 << iota
	modAlt
	modShift
	modMeta
	modHyper
	modSuper
)

var modNames = []struct {
	bit  int
	name string
}{{modCtrl, "ctrl"}, {modAlt, "alt"}, {modShift, "shift"}, {modMeta, "meta"}, {modHyper, "hyper"}, {modSuper, "super"}}

// modAlias maps how people write a modifier onto its Bubble Tea name.
var modAlias = map[string]int{
	"ctrl": modCtrl, "control": modCtrl, "ctl": modCtrl,
	"alt": modAlt, "opt": modAlt, "option": modAlt,
	"shift": modShift,
	"meta":  modMeta,
	"hyper": modHyper,
	"super": modSuper, "cmd": modSuper, "command": modSuper, "win": modSuper,
}

// named keys and their spellings, onto Bubble Tea's String() names
var keyAlias = map[string]string{
	"enter": "enter", "return": "enter", "ret": "enter", "⏎": "enter", "↵": "enter", "↩": "enter",
	"tab": "tab", "⇥": "tab",
	"backspace": "backspace", "bksp": "backspace", "bs": "backspace", "⌫": "backspace",
	"esc": "esc", "escape": "esc",
	"space": "space", "spc": "space", "spacebar": "space", "␣": "space",
	"up": "up", "↑": "up", "down": "down", "↓": "down", "left": "left", "←": "left", "right": "right", "→": "right",
	"home": "home", "end": "end", "insert": "insert", "ins": "insert", "delete": "delete", "del": "delete",
	"pgup": "pgup", "pageup": "pgup", "pgdown": "pgdown", "pgdn": "pgdown", "pagedown": "pgdown",
}

// key is one normalised keystroke.
type key struct {
	mods int
	base string // a Bubble Tea key name ("enter", "f5") or one character ("k", "K", "/")
}

func (k key) String() string {
	var b strings.Builder
	for _, m := range modNames {
		if k.mods&m.bit != 0 {
			b.WriteString(m.name + "+")
		}
	}
	b.WriteString(k.base)
	return b.String()
}

func isNamed(s string) bool {
	if _, ok := keyAlias[strings.ToLower(s)]; ok {
		return true
	}
	ls := strings.ToLower(s)
	if len(ls) >= 2 && ls[0] == 'f' {
		for _, r := range ls[1:] {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// parseKey normalises one written key: "Ctrl+K", "^C", "shift+a", "⏎", "pgdn".
func parseKey(s string) key {
	var k key
	if len(s) == 2 && s[0] == '^' && s[1] != '^' {
		k.mods, s = modCtrl, s[1:]
	}
	for {
		i := strings.Index(s, "+")
		if i <= 0 || i == len(s)-1 {
			break
		}
		bit, ok := modAlias[strings.ToLower(s[:i])]
		if !ok {
			break
		}
		k.mods |= bit
		s = s[i+1:]
	}
	if n, ok := keyAlias[strings.ToLower(s)]; ok {
		k.base = n
	} else if isNamed(s) {
		k.base = strings.ToLower(s)
	} else {
		k.base = s
	}
	if utf8.RuneCountInString(k.base) == 1 {
		r, _ := utf8.DecodeRuneInString(k.base)
		switch {
		case k.mods&modShift != 0 && unicode.IsLower(r) && k.mods&^modShift == 0:
			// shift+a arrives as "A"
			k.mods, k.base = 0, string(unicode.ToUpper(r))
		case k.mods&modCtrl != 0 && unicode.IsUpper(r):
			// "Ctrl+K" is how menus write ctrl+k; ctrl+shift+k says shift itself
			k.base = string(unicode.ToLower(r))
		}
	}
	return k
}

// splitAlts splits a key token into its "/" alternatives. A "/" that is the whole
// token, or follows a "+", is the slash key.
func splitAlts(tok string) []string {
	var out []string
	start := 0
	for i := 0; i < len(tok); i++ {
		if tok[i] != '/' || i == 0 || i == len(tok)-1 || tok[i-1] == '+' || tok[i-1] == '/' {
			continue
		}
		out = append(out, tok[start:i])
		start = i + 1
	}
	return append(out, tok[start:])
}

var arrowGlyph = map[rune]bool{'↑': true, '↓': true, '←': true, '→': true}

// modPrefix splits the modifiers off a written key as written: "shift+↑↓" is "shift+"
// and "↑↓"; "/" and "ctrl+" alone keep theirs as the key.
func modPrefix(a string) (prefix, rest string) {
	rest = a
	for {
		i := strings.Index(rest, "+")
		if i <= 0 || i == len(rest)-1 {
			return a[:len(a)-len(rest)], rest
		}
		if _, ok := modAlias[strings.ToLower(rest[:i])]; !ok {
			return a[:len(a)-len(rest)], rest
		}
		rest = rest[i+1:]
	}
}

// arrows reports whether s is one or more arrow glyphs ("↑", "↑↓", "←→").
func arrows(s string) bool {
	for _, r := range s {
		if !arrowGlyph[r] {
			return false
		}
	}
	return s != ""
}

// expandAlt turns one alternative into keys: "↑↓" is up and down, "jk" is j and k,
// "shift+↑↓" is shift+up and shift+down.
func expandAlt(a string) []key {
	runes := []rune(a)
	if prefix, rest := modPrefix(a); utf8.RuneCountInString(rest) > 1 && arrows(rest) {
		var ks []key
		for _, r := range rest {
			ks = append(ks, parseKey(prefix+string(r)))
		}
		return ks
	}
	if len(runes) == 2 && !strings.Contains(a, "+") && !isNamed(a) && unicode.IsLetter(runes[0]) && unicode.IsLetter(runes[1]) {
		return []key{{base: string(runes[0])}, {base: string(runes[1])}}
	}
	if len(runes) > 2 && len(runes) <= 4 && strings.Trim(a, "hjkl") == "" {
		ks := make([]key, len(runes)) // "hjkl", "hjk"
		for i, r := range runes {
			ks[i] = key{base: string(r)}
		}
		return ks
	}
	if len(a) == 3 && a[1] == '-' && a[0] >= '0' && a[0] < a[2] && a[2] <= '9' {
		var ks []key // "1-9"
		for d := a[0]; d <= a[2]; d++ {
			ks = append(ks, key{base: string(d)})
		}
		return ks
	}
	return []key{parseKey(a)}
}

// parseToken expands a whole key token ("shift+enter/ctrl+j", "↑↓/jk").
func parseToken(tok string) []key {
	ks, _ := readToken(tok)
	return ks
}

// readToken expands a key token into its keys, leaving out any alternative that names
// no key; bad is the first one left out ("" when every one is a key).
func readToken(tok string) (ks []key, bad string) {
	if n := len(tok); n > 2 && tok[0] == '<' && tok[n-1] == '>' {
		tok = tok[1 : n-1] // "<enter>"
	}
	carry, kind := "", "" // the last modifiers written, on an arrow or on a character
	for _, a := range splitAlts(tok) {
		if a == "" {
			continue
		}
		// "shift+↑/↓" and "ctrl+d/f/t": a bare alternative of the same kind as the last
		// modified one takes its modifiers; a named key ("ctrl+c/esc") keeps none
		prefix, rest := modPrefix(a)
		k := altKind(rest)
		switch {
		case prefix != "":
			carry, kind = prefix, k
		case carry != "" && k != "" && k == kind:
			a = carry + a
		}
		for _, k := range expandAlt(a) {
			if utf8.RuneCountInString(k.base) == 1 || isNamed(k.base) {
				ks = append(ks, k)
			} else if bad == "" {
				bad = a
			}
		}
	}
	return ks, bad
}

// altKind is what a carried modifier may land on: "arrow" for arrow glyphs, "char"
// for one other character, "" for anything else (a named key, a range).
func altKind(s string) string {
	switch {
	case arrows(s):
		return "arrow"
	case utf8.RuneCountInString(s) == 1:
		return "char"
	}
	return ""
}

// looksKey reports whether a field reads as a key token in full ("ctrl+c", "↑/↓,").
func looksKey(tok string) bool {
	tok = strings.TrimRight(tok, ",:")
	if len(tok) > 1 && strings.Trim(tok, "abcdefghijklmnopqrstuvwxyz") == "" && strings.Trim(tok, "hjkl") != "" && !isNamed(tok) {
		return false // a word ("to", "go"), not a pair of keys
	}
	ks, bad := readToken(tok)
	return len(ks) > 0 && bad == ""
}

// legacy says what a terminal without key disambiguation delivers for k: ok is false
// when that differs from k. as is the key that arrives ("" when nothing does) and why
// says how. Verified against Ultraviolet v0.0.0-20261008173134-6b8d4baf91b4's legacy
// decoder (decoder.go parseControl, key_table.go buildKeysTable), which Bubble Tea
// v2.1.0 uses when the terminal ignores its request for the kitty protocol and
// modifyOtherKeys.
func legacy(k key) (as string, why string, ok bool) {
	if k.mods&(modMeta|modHyper|modSuper) != 0 {
		return "", "only the kitty keyboard protocol sends meta, hyper and super", false
	}
	switch k.base {
	case "enter", "tab", "backspace", "esc", "space":
		lost := k.mods & (modCtrl | modShift)
		switch {
		case k.base == "tab" && k.mods&modCtrl == 0:
			return "", "", true // tab, shift+tab (CSI Z), alt+tab
		case k.base == "space" && k.mods&modShift == 0:
			return "", "", true // space, ctrl+space (NUL), alt+space
		case lost == 0:
			return "", "", true
		}
		rest := key{mods: k.mods &^ modCtrl, base: k.base}
		if k.base != "tab" {
			rest.mods &^= modShift
		}
		if k.base == "space" && k.mods&modCtrl != 0 {
			rest = key{mods: modCtrl | (k.mods & modAlt), base: "space"}
		}
		if k.base == "backspace" && k.mods&modCtrl != 0 {
			return key{mods: modCtrl | (k.mods & modAlt), base: "h"}.String(),
				"legacy encoding has no code for ctrl+backspace: xterm-like terminals send BS (0x08), which decodes as ctrl+h, and the rest send DEL (0x7f), plain backspace", false
		}
		return rest.String(), k.base + " is one byte, so its modifiers are lost", false
	}
	if k.mods&modCtrl == 0 || utf8.RuneCountInString(k.base) != 1 {
		return "", "", true
	}
	r, _ := utf8.DecodeRuneInString(k.base)
	alt := k.mods & modAlt
	switch {
	case r == 'm' && k.mods&modShift == 0:
		return key{mods: alt, base: "enter"}.String(), "ctrl+m and enter are both CR (0x0d)", false
	case r == 'i' && k.mods&modShift == 0:
		return key{mods: alt, base: "tab"}.String(), "ctrl+i and tab are both HT (0x09)", false
	case r == '[':
		return key{mods: alt, base: "esc"}.String(), "ctrl+[ and esc are both ESC (0x1b)", false
	case r == '@' || r == '2':
		return key{mods: modCtrl | alt, base: "space"}.String(), "ctrl+@, ctrl+2 and ctrl+space are all NUL (0x00)", false
	case r == '/':
		return key{mods: modCtrl | alt, base: "_"}.String(), "terminals send ctrl+/ as US (0x1f), which decodes as ctrl+_", false
	case unicode.IsLetter(r) && r < unicode.MaxASCII:
		if k.mods&modShift != 0 || unicode.IsUpper(r) {
			return key{mods: modCtrl | alt, base: string(unicode.ToLower(r))}.String(), "ctrl+shift+<letter> is the same control byte as ctrl+<letter>", false
		}
		return "", "", true
	case strings.ContainsRune(`\]^_`, r):
		return "", "", true // FS, GS, RS, US
	case r == ' ':
		return "", "", true
	}
	return "", "ctrl+" + k.base + " has no legacy code of its own; terminals send the bare key, another key's control byte, or nothing", false
}

// bsClash reports a key that arrives as itself but shares its byte with backspace:
// Ultraviolet decodes BS (0x08) as ctrl+h, and xterm-like terminals send BS for
// ctrl+backspace (a few for backspace), so those arrive as ctrl+h too.
func bsClash(k key) bool {
	return k.base == "h" && k.mods&^modAlt == modCtrl
}

// textKeys are the keys a focused Bubbles v2.2.1 textarea or textinput handles in its
// DefaultKeyMap (textarea/textarea.go, textinput/textinput.go), read from the module
// source. textinput's tab (AcceptSuggestion) is left out: it only acts while a
// suggestion shows.
var textKeys = func() map[string]string {
	m := map[string]string{}
	add := func(what string, ks ...string) {
		for _, k := range ks {
			s := parseKey(k).String()
			if _, ok := m[s]; !ok {
				m[s] = what
			}
		}
	}
	add("character forward", "right", "ctrl+f")
	add("character backward", "left", "ctrl+b")
	add("word forward", "alt+right", "ctrl+right", "alt+f")
	add("word backward", "alt+left", "ctrl+left", "alt+b")
	add("next line", "down", "ctrl+n")
	add("previous line", "up", "ctrl+p")
	add("delete word backward", "alt+backspace", "ctrl+w", "ctrl+backspace")
	add("delete word forward", "alt+delete", "alt+d", "ctrl+delete")
	add("delete after cursor", "ctrl+k")
	add("delete before cursor", "ctrl+u")
	add("insert newline", "enter", "ctrl+m")
	add("delete character backward", "backspace", "ctrl+h")
	add("delete character forward", "delete", "ctrl+d")
	add("line start", "home", "ctrl+a")
	add("line end", "end", "ctrl+e")
	add("page up", "pgup")
	add("page down", "pgdown")
	add("paste", "ctrl+v")
	add("input begin", "alt+<", "ctrl+home")
	add("input end", "alt+>", "ctrl+end")
	add("capitalize word", "alt+c")
	add("lowercase word", "alt+l")
	add("uppercase word", "alt+u")
	add("transpose characters", "ctrl+t")
	add("select character forward", "shift+right")
	add("select character backward", "shift+left")
	add("select word forward", "ctrl+shift+right", "alt+shift+right", "alt+shift+f")
	add("select word backward", "ctrl+shift+left", "alt+shift+left", "alt+shift+b")
	add("select line up", "shift+up")
	add("select line down", "shift+down")
	add("select all", "ctrl+g")
	add("copy selection", "ctrl+shift+c")
	return m
}()

// consumed reports what a focused text component does with k, or "".
func consumed(k key) string {
	if what, ok := textKeys[k.String()]; ok {
		return what
	}
	if k.mods == 0 && (k.base == "space" || utf8.RuneCountInString(k.base) == 1) {
		r, _ := utf8.DecodeRuneInString(k.base)
		if k.base == "space" || unicode.IsPrint(r) {
			return "types the character"
		}
	}
	return ""
}
