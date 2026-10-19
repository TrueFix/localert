// Package updater checks a publisher's release manifest and installs verified,
// platform-specific executables. Updates are always initiated by the dashboard.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"
)

const maxBinarySize = 128 << 20

var ErrBusy = errors.New("an update or restart is already in progress")
var ErrNoUpdate = errors.New("check for a newer release before upgrading")

type Asset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size,omitempty"`
}
type Manifest struct {
	Version     string           `json:"version"`
	Notes       string           `json:"notes,omitempty"`
	PublishedAt string           `json:"published_at,omitempty"`
	Assets      map[string]Asset `json:"assets"`
}

type Status struct {
	CurrentVersion string     `json:"current_version"`
	LatestVersion  string     `json:"latest_version,omitempty"`
	Phase          string     `json:"phase"`
	Notes          string     `json:"notes,omitempty"`
	CheckedAt      *time.Time `json:"checked_at,omitempty"`
	Error          string     `json:"error,omitempty"`
	Downloaded     int64      `json:"downloaded"`
	Total          int64      `json:"total"`
	Platform       string     `json:"platform"`
	Container      bool       `json:"container"`
}

type Options struct {
	Version    string
	Executable string
	Feed       func() (string, error)
	Restart    func() error
	Client     *http.Client
	Container  bool
	// Probe is replaceable in unit tests; production executes the verified binary's --version.
	Probe func(context.Context, string, string) error
}

type Manager struct {
	mu            sync.Mutex
	opts          Options
	status        Status
	asset         Asset
	checkedSource string
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func New(opts Options) *Manager {
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many release redirects")
			}
			return ValidateURL(r.URL.String())
		}}
	}
	if opts.Probe == nil {
		opts.Probe = probe
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{opts: opts, status: Status{CurrentVersion: opts.Version, Phase: "idle", Platform: runtime.GOOS + "-" + runtime.GOARCH, Container: opts.Container}, ctx: ctx, cancel: cancel}
}

func ValidateURL(value string) error {
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || len(value) > 2048 || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return errors.New("enter a valid HTTPS release-manifest URL")
	}
	if u.Scheme == "https" {
		return nil
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme == "http" && (strings.EqualFold(u.Hostname(), "localhost") || (ip != nil && ip.IsLoopback())) {
		return nil
	}
	return errors.New("release downloads require HTTPS (loopback HTTP is supported for local release testing)")
}

func (m *Manager) Status() Status { m.mu.Lock(); defer m.mu.Unlock(); return m.status }
func (m *Manager) Busy() bool     { m.mu.Lock(); defer m.mu.Unlock(); return m.busy() }
func (m *Manager) busy() bool {
	return m.status.Phase == "checking" || m.status.Phase == "downloading" || m.status.Phase == "installing" || m.status.Phase == "restarting"
}
func (m *Manager) Close() { m.mu.Lock(); m.cancel(); m.mu.Unlock(); m.wg.Wait() }

func (m *Manager) Invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy() {
		return
	}
	m.status = Status{CurrentVersion: m.opts.Version, Phase: "idle", Platform: m.status.Platform, Container: m.opts.Container}
	m.asset = Asset{}
	m.checkedSource = ""
}

func (m *Manager) Check(ctx context.Context) (Status, error) {
	m.mu.Lock()
	if m.busy() {
		result := m.status
		m.mu.Unlock()
		return result, ErrBusy
	}
	m.status.Phase, m.status.Error = "checking", ""
	m.asset = Asset{}
	m.mu.Unlock()
	err := m.check(ctx)
	if err != nil {
		m.fail(err)
	}
	return m.Status(), err
}

func (m *Manager) check(ctx context.Context) error {
	feed, err := m.opts.Feed()
	if err != nil {
		return err
	}
	if feed == "" {
		return errors.New("add your publisher's release-manifest URL in Settings first")
	}
	if err := ValidateURL(feed); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	response, err := m.get(ctx, feed)
	if err != nil {
		return fmt.Errorf("check releases: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil {
		return err
	}
	if len(data) > 256<<10 {
		return errors.New("release manifest is too large")
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("invalid release manifest: %w", err)
	}
	version := "v" + strings.TrimPrefix(manifest.Version, "v")
	if !semver.IsValid(version) {
		return errors.New("release manifest needs a valid semantic version")
	}
	asset, ok := manifest.Assets[runtime.GOOS+"-"+runtime.GOARCH]
	if !ok {
		return errors.New("this release has no binary for " + runtime.GOOS + "/" + runtime.GOARCH)
	}
	sum, err := hex.DecodeString(asset.SHA256)
	if err != nil || len(sum) != sha256.Size {
		return errors.New("release asset needs a SHA-256 checksum")
	}
	if asset.Size < 0 || asset.Size > maxBinarySize {
		return errors.New("release asset exceeds the 128 MiB size limit")
	}
	reference, err := url.Parse(asset.URL)
	if err != nil || asset.URL == "" {
		return errors.New("release asset needs a download URL")
	}
	asset.URL = response.Request.URL.ResolveReference(reference).String()
	if err := ValidateURL(asset.URL); err != nil {
		return err
	}
	now := time.Now().UTC()
	phase := "up_to_date"
	current := "v" + strings.TrimPrefix(m.opts.Version, "v")
	if semver.Compare(version, current) > 0 {
		phase = "available"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asset, m.checkedSource = asset, feed
	m.status = Status{CurrentVersion: m.opts.Version, LatestVersion: strings.TrimPrefix(version, "v"), Phase: phase, Notes: manifest.Notes, CheckedAt: &now, Platform: m.status.Platform, Container: m.opts.Container}
	return nil
}

func (m *Manager) Install() error {
	if m.opts.Container {
		return errors.New("update the container with docker compose pull && docker compose up -d")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy() {
		return ErrBusy
	}
	if m.status.Phase != "available" || m.asset.URL == "" {
		return ErrNoUpdate
	}
	feed, err := m.opts.Feed()
	if err != nil {
		return err
	}
	if feed != m.checkedSource {
		return errors.New("release source changed; check for updates again")
	}
	if m.ctx.Err() != nil {
		return errors.New("Localert is shutting down")
	}
	asset, version := m.asset, m.status.LatestVersion
	m.status.Phase, m.status.Error, m.status.Downloaded, m.status.Total = "downloading", "", 0, asset.Size
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.install(asset, version); err != nil {
			m.fail(err)
		}
	}()
	return nil
}

func (m *Manager) RequestRestart() error {
	m.mu.Lock()
	if m.busy() {
		m.mu.Unlock()
		return ErrBusy
	}
	m.status.Phase, m.status.Error = "restarting", ""
	m.mu.Unlock()
	if err := m.opts.Restart(); err != nil {
		m.fail(err)
		return err
	}
	return nil
}

func (m *Manager) install(asset Asset, version string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 3*time.Minute)
	defer cancel()
	response, err := m.get(ctx, asset.URL)
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	defer response.Body.Close()
	if response.ContentLength > maxBinarySize {
		return errors.New("download exceeds the 128 MiB size limit")
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	file, err := os.CreateTemp(filepath.Dir(m.opts.Executable), ".localert-download-*"+suffix)
	if err != nil {
		return fmt.Errorf("executable directory is not writable: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	hash := sha256.New()
	reader := &progressReader{reader: io.LimitReader(response.Body, maxBinarySize+1), progress: func(n int64) {
		m.mu.Lock()
		m.status.Downloaded = n
		if m.status.Total == 0 && response.ContentLength > 0 {
			m.status.Total = response.ContentLength
		}
		m.mu.Unlock()
	}}
	size, err := io.Copy(io.MultiWriter(file, hash), reader)
	if err != nil {
		return err
	}
	if size == 0 || size > maxBinarySize || (asset.Size > 0 && size != asset.Size) {
		return errors.New("downloaded binary has an unexpected size")
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), asset.SHA256) {
		return errors.New("checksum mismatch; the installed executable was left unchanged")
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Chmod(file.Name(), 0755); err != nil {
		return err
	}
	if err := m.opts.Probe(ctx, file.Name(), version); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	m.status.Phase = "installing"
	m.mu.Unlock()
	sum, _ := hex.DecodeString(asset.SHA256)
	input, err := os.Open(file.Name())
	if err != nil {
		return err
	}
	err = selfupdate.Apply(input, selfupdate.Options{TargetPath: m.opts.Executable, Checksum: sum, OldSavePath: m.opts.Executable + ".previous"})
	input.Close()
	if err != nil {
		return fmt.Errorf("install release: %w", err)
	}
	m.mu.Lock()
	m.status.Phase = "restarting"
	m.mu.Unlock()
	if err := m.opts.Restart(); err != nil {
		return fmt.Errorf("release installed; restart was not queued: %w", err)
	}
	return nil
}

func (m *Manager) get(ctx context.Context, address string) (*http.Response, error) {
	if err := ValidateURL(address); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Localert/"+m.opts.Version)
	response, err := m.opts.Client.Do(request)
	if err != nil {
		return nil, err
	}
	if err := ValidateURL(response.Request.URL.String()); err != nil {
		response.Body.Close()
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("release server returned HTTP %d", response.StatusCode)
	}
	return response, nil
}

func (m *Manager) fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Phase, m.status.Error = "error", err.Error()
}

func probe(ctx context.Context, path, version string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	hideConsole(cmd)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("downloaded binary cannot run on this machine: %w", err)
	}
	if strings.TrimSpace(string(output)) != "Localert "+version && strings.TrimSpace(string(output)) != "Localert v"+version {
		return errors.New("downloaded executable version does not match the release manifest")
	}
	return nil
}

type progressReader struct {
	reader   io.Reader
	read     int64
	progress func(int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	r.progress(r.read)
	return n, err
}
