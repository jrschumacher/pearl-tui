// Package tuicap holds tuicap, the skill's capture tool: it runs any terminal program in
// a pseudo-terminal, drives it with a tape, and writes its screens as the .ansi frames
// cellshot and tuilint frames read, for an app with no in-process harness. It is a module
// of its own, beside tools/, because it needs a pty and a VT emulator (Charm's x/xpty and
// x/vt), which the tools' dependency rule leaves out.
//
// Build it once, then run the binary from anywhere:
//
//	go -C <skill>/capture build -o "$HOME/.cache/tuitools/" ./cmd/tuicap
package tuicap
