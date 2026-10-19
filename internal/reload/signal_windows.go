//go:build windows

package reload

func signalReload(pid int) error {
	return ErrReload
}
