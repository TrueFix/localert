// Package reload asks a running Localert process to restart gracefully.
package reload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"localert/internal/store"
)

var (
	ErrNotRunning = errors.New("Localert is not running (no PID file or process)")
	ErrReload     = errors.New("could not ask Localert to restart")
)

// DefaultPIDFile matches the per-user installer layout.
func DefaultPIDFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		root := os.Getenv("LocalAppData")
		if root == "" {
			root = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(root, "Localert", "localert.pid"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "Localert", "localert.pid"), nil
	default:
		state := os.Getenv("XDG_STATE_HOME")
		if state == "" {
			state = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(state, "localert", "localert.pid"), nil
	}
}

func readPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func dashboardHost(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "http://127.0.0.1:8787"
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func httpReload(ctx context.Context, base string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/api/local/reload", nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	switch response.StatusCode {
	case http.StatusAccepted:
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("%w: open the dashboard on this computer", ErrReload)
	default:
		return fmt.Errorf("%w: HTTP %d", ErrReload, response.StatusCode)
	}
}

func waitForRestart(ctx context.Context, base, previous string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(20 * time.Second)
	}
	seenDown := false
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(base, "/")+"/api/health", nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err != nil {
			seenDown = true
		} else {
			id := ""
			body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
			response.Body.Close()
			if response.StatusCode == 200 {
				id = parseInstanceID(body)
			}
			if id != "" && previous != "" && id != previous {
				return nil
			}
			if seenDown && id != "" {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	if !seenDown && previous != "" {
		return fmt.Errorf("%w: running process did not restart", ErrReload)
	}
	return fmt.Errorf("%w: timed out waiting for Localert to come back", ErrReload)
}

func parseInstanceID(body []byte) string {
	text := string(body)
	if i := strings.Index(text, `"instance_id":"`); i >= 0 {
		rest := text[i+len(`"instance_id":"`):]
		if j := strings.IndexByte(rest, '"'); j > 0 {
			return rest[:j]
		}
	}
	return ""
}

// Request asks the running instance to restart. pidFile may be empty to use DefaultPIDFile.
func Request(ctx context.Context, dataDir, pidFile string) error {
	if pidFile == "" {
		var err error
		pidFile, err = DefaultPIDFile()
		if err != nil {
			return err
		}
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}
	state, err := store.Open(absolute)
	if err != nil {
		return fmt.Errorf("open local state: %w", err)
	}
	settings, err := state.Settings()
	state.Close()
	if err != nil {
		return err
	}
	base := dashboardHost(settings.ListenAddress)
	previous := instanceID(base)

	if pid, ok := readPID(pidFile); ok {
		if err := signalReload(pid); err == nil {
			return waitForRestart(ctx, base, previous)
		}
	}

	if err := httpReload(ctx, base); err != nil {
		if errors.Is(err, ErrReload) || isUnavailable(err) {
			return fmt.Errorf("%w (%v)", ErrNotRunning, err)
		}
		return err
	}
	return waitForRestart(ctx, base, previous)
}

func instanceID(base string) string {
	response, err := http.Get(strings.TrimSuffix(base, "/") + "/api/health")
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if i := strings.Index(string(body), `"instance_id":"`); i >= 0 {
		rest := string(body)[i+len(`"instance_id":"`):]
		if j := strings.IndexByte(rest, '"'); j > 0 {
			return rest[:j]
		}
	}
	return ""
}

func isUnavailable(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && (op.Op == "dial" || op.Op == "read")
}
