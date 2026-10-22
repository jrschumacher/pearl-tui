package tape

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cmds, err := Parse(strings.NewReader(`# comment

WaitFor Connect a cluster
Key down*2 enter ctrl+c
Type   two  spaces kept
Wait 200ms
WaitStable
WaitStable 2s
Snapshot setup-target
Resize 120x40
WaitExit 1s
Snapshot after-exit
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 10 {
		t.Fatalf("got %d commands", len(cmds))
	}
	if c := cmds[0]; c.Op != OpWaitFor || c.Text != "Connect a cluster" || c.Line != 3 {
		t.Errorf("WaitFor = %+v", c)
	}
	var names []string
	for _, k := range cmds[1].Keys {
		names = append(names, k.Name)
	}
	if got := strings.Join(names, " "); got != "down down enter ctrl+c" {
		t.Errorf("keys = %q", got)
	}
	if c := cmds[2]; c.Text != "  two  spaces kept" {
		t.Errorf("Type text = %q", c.Text)
	}
	if cmds[3].Dur != 200*time.Millisecond || cmds[4].Dur != 0 || cmds[5].Dur != 2*time.Second {
		t.Errorf("durations: %v %v %v", cmds[3].Dur, cmds[4].Dur, cmds[5].Dur)
	}
	if c := cmds[7]; c.W != 120 || c.H != 40 {
		t.Errorf("Resize = %+v", c)
	}
}

func TestParseErrors(t *testing.T) {
	for tape, want := range map[string]string{
		"Press enter":     "line 1: unknown command",
		"Key":             "Key needs at least one key name",
		"Key hyper+x":     "unknown modifier",
		"Key down*0":      "repeat count",
		"Key shift+enter": "modified enter has no legacy encoding",
		"Wait soon":       "Wait needs a duration",
		"Snapshot a/b":    "state name",
		"Resize 80":       "want WxH",
		"WaitExit\nKey q": "line 2: only Snapshot may follow WaitExit",
		"\n\nWaitFor":     "line 3: WaitFor needs text",
	} {
		if _, err := Parse(strings.NewReader(tape)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want %q", tape, err, want)
		}
	}
}

func TestKeyBytes(t *testing.T) {
	for name, want := range map[string]string{
		"enter": "\r", "esc": "\x1b", "tab": "\t", "shift+tab": "\x1b[Z", "space": " ", "backspace": "\x7f",
		"up": "\x1b[A", "ctrl+up": "\x1b[1;5A", "shift+right": "\x1b[1;2C", "home": "\x1b[H", "end": "\x1b[F",
		"pgup": "\x1b[5~", "pgdn": "\x1b[6~", "delete": "\x1b[3~", "alt+delete": "\x1b[3;3~", "f1": "\x1bOP", "f5": "\x1b[15~",
		"q": "q", "?": "?", "/": "/", "shift+a": "A", "ctrl+c": "\x03", "ctrl+C": "\x03", "ctrl+[": "\x1b", "ctrl+space": "\x00",
		"alt+x": "\x1bx", "alt+enter": "\x1b\r", "ctrl+backspace": "\x08", "+": "+", "ctrl+?": "\x7f",
	} {
		k, err := ParseKey(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := k.Bytes(false); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	k, _ := ParseKey("down")
	if got := k.Bytes(true); got != "\x1bOB" {
		t.Errorf("down in application cursor mode = %q, want ESC O B", got)
	}
}
