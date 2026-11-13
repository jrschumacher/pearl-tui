//go:build linux

package ui

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// pty opens a pseudo-terminal pair: the side a program reads (pts) and the side the
// user types into (ptm). Skips where the machine has none.
func pty(t *testing.T) (ptm, pts *os.File) {
	t.Helper()
	ptm, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { ptm.Close() })
	var unlock int32
	var n uint32
	var errno syscall.Errno
	rc, err := ptm.SyscallConn()
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	rc.Control(func(fd uintptr) {
		if _, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); errno != 0 {
			return
		}
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	})
	if errno != 0 {
		t.Skipf("no pseudo-terminal: %v", errno)
	}
	pts, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { pts.Close() })
	return ptm, pts
}

// echoing reports whether the terminal echoes what is typed into it.
func echoing(t *testing.T, f *os.File) bool {
	t.Helper()
	var tio syscall.Termios
	var errno syscall.Errno
	rc, err := f.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	rc.Control(func(fd uintptr) {
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&tio)))
	})
	if errno != 0 {
		t.Fatalf("reading the terminal's mode: %v", errno)
	}
	return tio.Lflag&syscall.ECHO != 0
}

// askSecret starts a secret Ask on pts and waits until it has turned the echo off.
func askSecret(t *testing.T, pts *os.File, out *bytes.Buffer) (*Input, chan error) {
	t.Helper()
	in := NewInput("Token", "", "", "")
	in.Secret = true
	done := make(chan error, 1)
	go func() { done <- in.Ask(NewPrompter(pts, out)) }()
	for end := time.Now().Add(5 * time.Second); echoing(t, pts); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatal("Ask never turned the echo off")
		}
	}
	return &in, done
}

func wait(t *testing.T, done chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Ask did not return")
	}
	return nil
}

// On a terminal a secret is read with the echo off, so the typed characters never reach
// the screen, and the echo is back on afterwards. (A failed read restores it through
// the same deferred restore in x/term's ReadPassword; a pty can't be made to fail a
// read and still be asked for its mode, so that path is not driven here.) Failure
// messages name lengths, never the value.
func TestSecretAskOnATerminal(t *testing.T) {
	const secret = "hunter2-tok"
	ptm, pts := pty(t)
	if !echoing(t, pts) {
		t.Skip("a fresh pseudo-terminal does not echo here")
	}
	var out bytes.Buffer
	in, done := askSecret(t, pts, &out)
	if _, err := ptm.Write([]byte(secret + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := wait(t, done); err != nil {
		t.Fatal(err)
	}
	if in.Value() != secret {
		t.Fatalf("read %d characters, typed %d", len(in.Value()), len(secret))
	}
	if !echoing(t, pts) {
		t.Error("the echo is still off after the answer")
	}
	// what the terminal showed the user: nothing of the secret
	if err := ptm.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, _ := ptm.Read(buf)
	if bytes.Contains(buf[:n], []byte("hunter")) || strings.Contains(out.String(), "hunter") {
		t.Error("the secret was echoed")
	}
}
