//go:build !windows

package updater

import (
	"os"
	"os/exec"
	"syscall"
)

func hideConsole(*exec.Cmd) {}

// Re-exec preserves the process identity tracked by desktop launchers/LaunchAgents.
func Restart(executable string, arguments []string) error {
	return syscall.Exec(executable, append([]string{executable}, arguments...), os.Environ())
}
