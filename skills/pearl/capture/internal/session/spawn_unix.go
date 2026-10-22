//go:build unix

package session

import (
	"errors"
	"os/exec"
	"syscall"

	"github.com/charmbracelet/x/xpty"
)

// spawn starts cmd on the pty in a session of its own, the pty its controlling
// terminal, so a resize reaches it as SIGWINCH and the line discipline treats it as a
// terminal's foreground job. The parent's copy of the pty's slave side is closed, so the
// master reads EOF once the program and its children are gone.
func spawn(p xpty.Pty, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := p.Start(cmd); err != nil {
		return err
	}
	if up, ok := p.(*xpty.UnixPty); ok {
		_ = up.Slave().Close()
	}
	return nil
}

// hangup sends SIGHUP to the program's process group, as a closing terminal does.
func hangup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGHUP)
	}
}

// kill sends SIGKILL to the program's process group.
func kill(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func exitOf(cmd *exec.Cmd, err error) Exit {
	e := Exit{Exited: true}
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		e.Code = -1
		return e
	}
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		e.Code, e.Signal = -1, ws.Signal().String()
		return e
	}
	e.Code = cmd.ProcessState.ExitCode()
	return e
}
