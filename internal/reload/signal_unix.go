//go:build !windows

package reload

import (
	"os"
	"syscall"
)

func signalReload(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.SIGUSR1)
}
