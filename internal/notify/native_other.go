//go:build !linux && !darwin && !windows

package notify

import (
	"context"
	"errors"
)

func (Native) Send(context.Context, Notification, string) error {
	return errors.New("native notifications are supported on Linux, Windows, and macOS")
}
