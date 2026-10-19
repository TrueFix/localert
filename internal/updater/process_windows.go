package updater

import (
	"os"
	"os/exec"
	"syscall"
)

func hideConsole(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }

func Restart(executable string, arguments []string) error {
	cmd := exec.Command(executable, arguments...)
	hideConsole(cmd)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
