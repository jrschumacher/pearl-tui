//go:build !unix

package session

import (
	"errors"
	"os/exec"

	"github.com/charmbracelet/x/xpty"
)

// spawn refuses: tuicap needs a Unix pty, a session and a process group to signal.
func spawn(xpty.Pty, *exec.Cmd) error {
	return errors.New("tuicap runs on Unix only (Linux, macOS, the BSDs, WSL)")
}

func hangup(*exec.Cmd) {}
func kill(*exec.Cmd)   {}

func exitOf(cmd *exec.Cmd, err error) Exit {
	if cmd.ProcessState == nil {
		return Exit{Exited: true, Code: -1}
	}
	return Exit{Exited: true, Code: cmd.ProcessState.ExitCode()}
}
