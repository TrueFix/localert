//go:build !windows

package updater

import (
	"os"
	"os/exec"
	"syscall"
)

func hideConsole(*exec.Cmd) {}

// Start a fresh process and exit. This matches Windows behavior and works reliably
// after installs that wrap Localert in shell launchers or redirect stdio.
func Restart(executable string, arguments []string) error {
	cmd := exec.Command(executable, arguments...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	os.Exit(0)
	return nil
}
