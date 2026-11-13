package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"tuicap/internal/redact"
	"tuicap/internal/session"
	"tuicap/internal/tape"
)

// flow.json

type flowFile struct {
	Tool    string    `json:"tool"`
	Level   string    `json:"level"`
	Command []string  `json:"command"`
	Tape    string    `json:"tape"`
	Profile string    `json:"profile"`
	Light   bool      `json:"light"`
	Runs    []flowRun `json:"runs"`
	Error   string    `json:"error,omitempty"`
}

type flowRun struct {
	Size      string       `json:"size"`
	Snapshots []flowSnap   `json:"snapshots"`
	After     []flowInput  `json:"input_after_last_snapshot,omitempty"`
	Exit      session.Exit `json:"exit"`
	Error     string       `json:"error,omitempty"`
}

type flowSnap struct {
	State     string      `json:"state"`
	Frame     string      `json:"frame"`
	Size      string      `json:"size"`
	Profile   string      `json:"profile"`
	Input     []flowInput `json:"input"`
	ElapsedMS int64       `json:"elapsed_ms"`
	Stable    bool        `json:"stable"`
}

type flowInput struct {
	Key    string `json:"key,omitempty"`
	Type   string `json:"type,omitempty"`
	Resize string `json:"resize,omitempty"`
}

// sequences.json

type seqFile struct {
	Tool  string   `json:"tool"`
	Level string   `json:"level"`
	Runs  []seqRun `json:"runs"`
}

type seqRun struct {
	Size      string                    `json:"size"`
	Snapshots []seqSnap                 `json:"snapshots"`
	After     map[string]int            `json:"after_last_snapshot"`
	Total     map[string]*session.Total `json:"total"`
	Title     *seqTitle                 `json:"title,omitempty"`
	Unknown   []session.Unknown         `json:"unknown"`
	Cursor    session.Cursor            `json:"final_cursor"`
	End       seqEnd                    `json:"end"`
}

type seqSnap struct {
	State  string         `json:"state"`
	Events map[string]int `json:"events"`
	Cursor session.Cursor `json:"cursor"`
}

type seqTitle struct {
	Count  int    `json:"count"`
	Length int    `json:"length"`
	Text   string `json:"text,omitempty"`
}

type seqEnd struct {
	Exited bool     `json:"exited"`
	Clean  *bool    `json:"clean"`
	Left   []string `json:"left"`
	Note   string   `json:"note,omitempty"`
}

// capture runs the tape once per size and writes the bundle.
func capture(o *options, stderr io.Writer) int {
	level := "full"
	if o.privacy {
		level = "privacy"
	}
	flow := flowFile{Tool: "tuicap", Level: level, Command: o.argv, Tape: filepath.Base(o.tapeName), Profile: o.profile, Light: o.light}
	if o.privacy {
		flow.Command = make([]string, len(o.argv))
		for i, a := range o.argv {
			flow.Command[i] = o.red.String(a)
		}
	}
	seqs := seqFile{Tool: "tuicap", Level: level}
	failed := false
	written := map[string][]string{} // size → frame paths
	for _, sz := range o.sizes {
		fr, sr, err := runOnce(o, sz, written, stderr)
		if err != nil {
			fr.Error = err.Error()
			fmt.Fprintf(stderr, "tuicap: %s: %v\n", sz, err)
			failed = true
		}
		flow.Runs = append(flow.Runs, fr)
		seqs.Runs = append(seqs.Runs, sr)
	}
	if failed {
		flow.Error = "a run failed: see runs[].error"
	}
	if err := writeJSON(filepath.Join(o.out, "flow.json"), flow); err != nil {
		fmt.Fprintf(stderr, "tuicap: %v\n", err)
		return 1
	}
	if err := writeJSON(filepath.Join(o.out, "sequences.json"), seqs); err != nil {
		fmt.Fprintf(stderr, "tuicap: %v\n", err)
		return 1
	}
	sheets := writeSheets(o, written, stderr)
	if err := os.WriteFile(filepath.Join(o.out, "README.txt"), []byte(readme(o, level, written, sheets)), 0o644); err != nil {
		fmt.Fprintf(stderr, "tuicap: %v\n", err)
		return 1
	}
	n := 0
	for _, f := range written {
		n += len(f)
	}
	fmt.Fprintf(stderr, "tuicap: %d frames (%s level) in %s\n", n, level, o.out)
	if failed {
		return 1
	}
	return 0
}

// keyGap separates keys of one Key line: written together, ESC and the next key read
// as alt+key.
const keyGap = 20 * time.Millisecond

func runOnce(o *options, sz size, written map[string][]string, stderr io.Writer) (fr flowRun, sr seqRun, err error) {
	fr = flowRun{Size: sz.String(), Snapshots: []flowSnap{}}
	sr = seqRun{Size: sz.String(), Snapshots: []seqSnap{}, Unknown: []session.Unknown{}}
	cfg := session.Config{Argv: o.argv, Env: o.env, W: sz.w, H: sz.h, Fg: o.fg, Bg: o.bg, Full: !o.privacy}
	if o.raw {
		f, err := os.Create(filepath.Join(o.out, "raw-"+sz.String()+".log"))
		if err != nil {
			return fr, sr, err
		}
		defer f.Close()
		cfg.Raw = f
	}
	sess, err := session.Start(cfg)
	if err != nil {
		return fr, sr, err
	}
	var inputs []flowInput
	finish := func() {
		fr.Exit = sess.Close()
		sr.After = sess.Cut("(end)")
		if len(inputs) > 0 {
			fr.After = inputs
		}
		tr := sess.Tracker()
		sr.Total = tr.Totals()
		sr.Unknown = append(sr.Unknown, tr.Unknowns()...)
		if c := sr.Total["title"]; c != nil {
			sr.Title = &seqTitle{Count: c.Count, Length: tr.TitleLen, Text: tr.Title}
		}
		sr.Cursor = sess.FinalCursor()
		sr.End = seqEnd{Exited: !fr.Exit.Killed, Left: tr.Left()}
		if sr.End.Left == nil {
			sr.End.Left = []string{}
		}
		if fr.Exit.Killed {
			sr.End.Note = "tuicap ended the program after the tape: end the tape with its quit key and WaitExit to judge a clean exit"
		} else {
			clean := len(sr.End.Left) == 0
			sr.End.Clean = &clean
		}
	}
	defer finish()

	deadline := time.Now().Add(o.timeout)
	waitFor := func(c tape.Cmd) time.Duration {
		d := o.wait
		if c.Dur > 0 {
			d = c.Dur
		}
		return min(d, time.Until(deadline))
	}
	exitedEarly := func(c tape.Cmd) error {
		x := sess.ExitStatus()
		how := fmt.Sprintf("with status %d", x.Code)
		if x.Signal != "" {
			how = "on " + x.Signal
		}
		return fmt.Errorf("line %d (%s): the program exited %s before the tape ended (end it with WaitExit if it should)", c.Line, c.Src, how)
	}
	// input waits for the program's first output, as a person waits for a prompt
	ready := func(c tape.Cmd) error {
		if err := sess.WaitOutput(min(o.wait, time.Until(deadline))); err != nil {
			if session.IsExited(err) {
				return exitedEarly(c)
			}
			return fmt.Errorf("line %d (%s): no output from the program after %s", c.Line, c.Src, o.wait)
		}
		return nil
	}
	exitedOK := false
	for _, c := range o.cmds {
		if time.Now().After(deadline) {
			return fr, sr, fmt.Errorf("line %d (%s): the run passed -timeout %s", c.Line, c.Src, o.timeout)
		}
		if !exitedOK {
			select {
			case <-sess.Done():
				return fr, sr, exitedEarly(c)
			default:
			}
		}
		switch c.Op {
		case tape.OpKey:
			if err := ready(c); err != nil {
				return fr, sr, err
			}
			for i, k := range c.Keys {
				if i > 0 {
					time.Sleep(keyGap)
				}
				if err := sess.Send(k.Bytes); err != nil {
					return fr, sr, exitedEarly(c)
				}
				name := k.Name
				if o.privacy && utf8.RuneCountInString(name) == 1 {
					name = o.red.String(name)
				}
				inputs = append(inputs, flowInput{Key: name})
			}
		case tape.OpType:
			if err := ready(c); err != nil {
				return fr, sr, err
			}
			if err := sess.Type(c.Text); err != nil {
				return fr, sr, exitedEarly(c)
			}
			text := c.Text
			if o.privacy {
				text = o.red.String(text)
			}
			inputs = append(inputs, flowInput{Type: text})
		case tape.OpWait:
			time.Sleep(min(c.Dur, time.Until(deadline)))
		case tape.OpWaitStable:
			d := waitFor(c)
			if !sess.WaitStable(d) {
				select {
				case <-sess.Done():
					return fr, sr, exitedEarly(c)
				default:
				}
				fmt.Fprintf(stderr, "tuicap: %s line %d: the screen was still changing after %s (an animation?); going on\n", sz, c.Line, d)
			}
		case tape.OpWaitFor:
			d := waitFor(c)
			if err := sess.WaitFor(c.Text, d); err != nil {
				if session.IsExited(err) {
					return fr, sr, exitedEarly(c)
				}
				return fr, sr, fmt.Errorf("line %d: WaitFor: %q was not on screen after %s", c.Line, c.Text, d)
			}
		case tape.OpResize:
			if err := sess.Resize(c.W, c.H); err != nil {
				return fr, sr, exitedEarly(c)
			}
			inputs = append(inputs, flowInput{Resize: fmt.Sprintf("%dx%d", c.W, c.H)})
		case tape.OpWaitExit:
			d := waitFor(c)
			if err := sess.WaitExit(d); err != nil {
				return fr, sr, fmt.Errorf("line %d: WaitExit: %v", c.Line, err)
			}
			exitedOK = true
		case tape.OpSnapshot:
			if !exitedOK {
				if err := ready(c); err != nil {
					return fr, sr, err
				}
			}
			f := sess.Snapshot()
			fsz := size{f.W, f.H}.String()
			rel := filepath.Join("frames", fmt.Sprintf("%s-%s.ansi", c.Text, fsz))
			for _, p := range written[fsz] {
				if p == rel {
					return fr, sr, fmt.Errorf("line %d: %s was already captured in this bundle", c.Line, rel)
				}
			}
			if o.privacy {
				o.red.Lines(f.Lines)
			}
			frame := session.Encode(f.Lines)
			if err := session.CheckSize(frame, f.W, f.H); err != nil {
				return fr, sr, fmt.Errorf("line %d: %s: %v", c.Line, rel, err)
			}
			if err := os.WriteFile(filepath.Join(o.out, rel), []byte(frame), 0o644); err != nil {
				return fr, sr, err
			}
			written[fsz] = append(written[fsz], rel)
			if inputs == nil {
				inputs = []flowInput{}
			}
			fr.Snapshots = append(fr.Snapshots, flowSnap{State: c.Text, Frame: rel, Size: fsz, Profile: o.profile, Input: inputs,
				ElapsedMS: sess.Elapsed().Milliseconds(), Stable: f.Stable})
			sr.Snapshots = append(sr.Snapshots, seqSnap{State: c.Text, Events: sess.Cut(c.Text), Cursor: f.Cursor})
			inputs = nil
			note := ""
			if !f.Stable {
				note = " (the screen was still changing)"
			}
			fmt.Fprintf(stderr, "tuicap: %s%s\n", rel, note)
		}
	}
	return fr, sr, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// owned are the files tuicap writes in a bundle, and so may replace.
func owned(name string) bool {
	switch {
	case name == "flow.json", name == "sequences.json", name == "README.txt", name == "frames":
		return true
	case strings.HasPrefix(name, "sheet-") && strings.HasSuffix(name, ".png"),
		strings.HasPrefix(name, "raw-") && strings.HasSuffix(name, ".log"):
		return true
	}
	return false
}

// prepareOut makes dir an empty bundle: it must be empty or missing, or a bundle tuicap
// wrote (it has flow.json and nothing tuicap doesn't own), whose files are removed so a
// privacy bundle never keeps a full-level file from an earlier run.
func prepareOut(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(filepath.Join(dir, "frames"), 0o755)
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		isBundle := false
		for _, e := range entries {
			if !owned(e.Name()) {
				return fmt.Errorf("-out %s holds %s, which tuicap didn't write: use an empty directory or an earlier bundle", dir, e.Name())
			}
			isBundle = isBundle || e.Name() == "flow.json"
		}
		if !isBundle {
			return fmt.Errorf("-out %s is not empty and has no flow.json: use an empty directory or an earlier bundle", dir)
		}
		frames, _ := os.ReadDir(filepath.Join(dir, "frames"))
		for _, f := range frames {
			if f.IsDir() || filepath.Ext(f.Name()) != ".ansi" {
				return fmt.Errorf("-out %s/frames holds %s, which tuicap didn't write", dir, f.Name())
			}
		}
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return os.MkdirAll(filepath.Join(dir, "frames"), 0o755)
}

// writeSheets draws a contact sheet per size with cellshot, when it is on PATH.
func writeSheets(o *options, written map[string][]string, stderr io.Writer) []string {
	cellshot, err := exec.LookPath("cellshot")
	if err != nil || len(written) == 0 {
		return nil
	}
	var sheets []string
	for _, sz := range sortedKeys(written) {
		name := "sheet-" + sz + ".png"
		args := append([]string{}, written[sz]...)
		args = append(args, "--sheet", name)
		if o.light {
			args = append(args, "--light")
		}
		cmd := exec.Command(cellshot, args...)
		cmd.Dir = o.out
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(stderr, "tuicap: cellshot %s: %v\n%s", name, err, out)
			continue
		}
		sheets = append(sheets, name)
	}
	return sheets
}

func sortedKeys(m map[string][]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func readme(o *options, level string, written map[string][]string, sheets []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "A tuicap capture bundle, %s level.\n\n", level)
	if o.privacy {
		b.WriteString(`Privacy level: the frames keep their layout cell for cell (widths, colours and
styles, glyphs, box drawing, punctuation, whitespace), but every letter is x or X
and every digit 0; URL-, email- and path-shaped runs are redacted whole. Key names
and generic UI words are kept (` + strings.Join(allowList(o), " ") + `).
Typed text and the command line are redacted the same way. No titles' text, no
unknown sequences' bytes, no raw log. Words can't be reviewed here; layout,
colour, hierarchy and glyphs can.
`)
	} else {
		b.WriteString(`Full level: frames as captured, typed text in flow.json, titles' text and unknown
sequences' bytes in sequences.json. It shows the product as it is: share a
-level privacy bundle instead when the product is private.
`)
	}
	b.WriteString("\nContents\n\n")
	for _, sz := range sortedKeys(written) {
		fmt.Fprintf(&b, "  frames/*-%s.ansi   %d frames at %s: the screen at each Snapshot, SGR only\n", sz, len(written[sz]), sz)
	}
	b.WriteString(`  flow.json        the run: per snapshot its state, size, colour profile, the keys,
                   text and resizes sent before it, elapsed ms, whether the screen was
                   stable; and how the program ended
  sequences.json   what the program did to the terminal beyond colour, per snapshot
                   and in total: alt screen, synchronized output, cursor, mouse,
                   bracketed paste, focus, titles, links, clipboard, clears, queries,
                   unknown sequences; and whether it left the terminal clean
`)
	for _, s := range sheets {
		fmt.Fprintf(&b, "  %-16s a contact sheet of those frames (cellshot)\n", s)
	}
	if o.raw {
		b.WriteString("  raw-WxH.log      the program's output bytes, verbatim\n")
	}
	fmt.Fprintf(&b, "\nCaptured by tuicap (profile %s, %s terminal). See them with:\n\n", o.profile, map[bool]string{true: "light", false: "dark"}[o.light])
	b.WriteString("  cellshot frames/<state>-80x24.ansi out.png\n  tuilint frames frames\n")
	return b.String()
}

func allowList(o *options) []string {
	return append(append(append([]string{}, redact.Allow...), o.allow...), "and arrows and other glyphs")
}
