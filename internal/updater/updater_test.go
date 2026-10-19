package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func updateFixture(t *testing.T, content []byte, checksum string) (*Manager, string, <-chan struct{}) {
	t.Helper()
	target := filepath.Join(t.TempDir(), "localert")
	if err := os.WriteFile(target, []byte("original executable"), 0755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/new-binary" {
			w.Write(content)
			return
		}
		json.NewEncoder(w).Encode(Manifest{Version: "0.4.0", Assets: map[string]Asset{runtime.GOOS + "-" + runtime.GOARCH: {URL: "new-binary", SHA256: checksum, Size: int64(len(content))}}})
	}))
	t.Cleanup(server.Close)
	restarted := make(chan struct{}, 1)
	m := New(Options{Version: "0.3.0", Executable: target, Feed: func() (string, error) { return server.URL + "/manifest.json", nil }, Probe: func(context.Context, string, string) error { return nil }, Restart: func() error { restarted <- struct{}{}; return nil }})
	t.Cleanup(m.Close)
	return m, target, restarted
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func TestUpdateChecksDownloadsVerifiesAndKeepsPreviousExecutable(t *testing.T) {
	content := []byte("a verified new executable")
	m, target, restarted := updateFixture(t, content, digest(content))
	status, err := m.Check(context.Background())
	if err != nil || status.Phase != "available" || status.LatestVersion != "0.4.0" {
		t.Fatalf("check: %+v %v", status, err)
	}
	if err := m.Install(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-restarted:
	case <-time.After(5 * time.Second):
		t.Fatalf("update did not request restart: %+v", m.Status())
	}
	data, _ := os.ReadFile(target)
	if string(data) != string(content) {
		t.Fatal("downloaded executable was not installed")
	}
	old, _ := os.ReadFile(target + ".previous")
	if string(old) != "original executable" {
		t.Fatal("previous executable was not retained")
	}
	if m.Status().Phase != "restarting" {
		t.Fatal("restart status missing")
	}
}

func TestBadChecksumAndFailedProbeLeaveOriginalUntouched(t *testing.T) {
	for _, test := range []struct {
		name       string
		checksum   string
		probeError bool
	}{{"checksum", strings.Repeat("0", 64), false}, {"executable probe", digest([]byte("download")), true}} {
		t.Run(test.name, func(t *testing.T) {
			m, target, restarted := updateFixture(t, []byte("download"), test.checksum)
			if test.probeError {
				m.opts.Probe = func(context.Context, string, string) error { return errors.New("wrong executable version") }
			}
			if _, err := m.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := m.Install(); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for m.Status().Phase != "error" && time.Now().Before(deadline) {
				time.Sleep(5 * time.Millisecond)
			}
			if m.Status().Phase != "error" {
				t.Fatalf("bad release was not rejected: %+v", m.Status())
			}
			data, _ := os.ReadFile(target)
			if string(data) != "original executable" {
				t.Fatal("invalid release replaced the installed program")
			}
			select {
			case <-restarted:
				t.Fatal("invalid release requested a restart")
			default:
			}
		})
	}
}

func TestFeedURLValidationAndExplicitReleaseCheck(t *testing.T) {
	for _, value := range []string{"https://releases.example/feed.json", "http://127.0.0.1:9000/feed.json", "http://[::1]:9000/feed.json", ""} {
		if err := ValidateURL(value); err != nil {
			t.Errorf("valid source %s: %v", value, err)
		}
	}
	for _, value := range []string{"http://example.com/feed.json", "file:///tmp/a.json", "https://user:password@example.com/feed", "https://example.com/feed#fragment", "javascript:alert(1)"} {
		if err := ValidateURL(value); err == nil {
			t.Errorf("accepted invalid update URL: %s", value)
		}
	}
	m, _, _ := updateFixture(t, []byte("content"), digest([]byte("content")))
	if err := m.Install(); !errors.Is(err, ErrNoUpdate) {
		t.Fatal("update was installed without a release check")
	}
	m.opts.Feed = func() (string, error) { return "", nil }
	if status, err := m.Check(context.Background()); err == nil || status.Phase != "error" {
		t.Fatal("unconfigured source claimed to be up to date")
	}
}
