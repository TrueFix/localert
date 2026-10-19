package notify

import (
	"context"
	"fmt"
	"os/exec"
)

func (Native) Send(ctx context.Context, n Notification, iconPath string) error {
	if executable, err := exec.LookPath("terminal-notifier"); err == nil {
		if err := exec.CommandContext(ctx, executable, "-title", n.Title, "-message", n.Message,
			"-appIcon", iconPath, "-group", "localert").Run(); err == nil {
			return nil
		}
	}
	// Arguments are passed as argv rather than interpolated into AppleScript.
	script := "on run argv\ndisplay notification (item 2 of argv) with title (item 1 of argv)\nend run"
	if out, err := exec.CommandContext(ctx, "osascript", "-e", script, "--", n.Title, n.Message).CombinedOutput(); err != nil {
		return fmt.Errorf("macOS notification: %w: %s", err, out)
	}
	return nil
}
