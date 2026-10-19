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
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	os.Exit(0)
	return nil
}
