package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"localert/internal/notify"
	"localert/internal/store"
)

type fakeSender struct {
	mu   sync.Mutex
	sent []notify.Notification
	err  error
}

func (sender *fakeSender) Send(ctx context.Context, n notify.Notification, path string) error {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.sent = append(sender.sent, n)
	return sender.err
}

type fixture struct {
	server *httptest.Server
	store  *store.Store
	sender *fakeSender
	client *http.Client
}

const testPassword = "desktop test password"

func setup(t *testing.T) fixture {
	return newFixture(t, true)
}

func newFixture(t *testing.T, configured bool) fixture {
	t.Helper()
	dir := t.TempDir()
	state, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { state.Close() })
	if configured {
		if err := state.Setup(testPassword, ""); err != nil {
			t.Fatal(err)
		}
	}
	icons, err := notify.NewIcons(dir)
	if err != nil {
		t.Fatal(err)
	}
	sender := &fakeSender{}
	ts := httptest.NewUnstartedServer(nil)
	ts.Config.Handler = New(Options{Store: state, Sender: sender, Icons: icons, Address: ts.Listener.Addr().String(), Version: "test", Assets: fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><h1>Localert</h1>")}}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	ts.Start()
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	client := ts.Client()
	client.Jar = jar
	return fixture{ts, state, sender, client}
}

func (f fixture) request(t *testing.T, method, path, body, bearer string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, data
}

func (f fixture) login(t *testing.T) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"password": testPassword})
	resp, data := f.request(t, "POST", "/api/admin/login", string(body), "")
	if resp.StatusCode != 200 {
		t.Fatalf("login: %d %s", resp.StatusCode, data)
	}
	cookies := resp.Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("session cookie missing protection")
	}
}

func TestFullAppKeyLifecycleAndSessionIsolation(t *testing.T) {
	f := setup(t)
	if resp, _ := f.request(t, "GET", "/api/admin/overview", "", ""); resp.StatusCode != 401 {
		t.Fatal("anonymous admin access")
	}
	if resp, _ := f.request(t, "POST", "/api/admin/login", `{"password":"wrong"}`, ""); resp.StatusCode != 401 {
		t.Fatal("invalid password accepted")
	}
	f.login(t)
	resp, data := f.request(t, "POST", "/api/admin/keys", `{"name":"Build app"}`, "")
	if resp.StatusCode != 201 {
		t.Fatalf("create key: %d %s", resp.StatusCode, data)
	}
	var created struct {
		Key    store.Key `json:"key"`
		Secret string    `json:"secret"`
	}
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatal(err)
	}
	if created.Secret == "" || created.Key.ID == "" {
		t.Fatal("missing generated key")
	}
	payload := `{"title":"Build complete","message":"Ready <now> & safe","icon":"success"}`
	if resp, data := f.request(t, "POST", "/api/notify", payload, created.Secret); resp.StatusCode != 200 {
		t.Fatalf("notify: %d %s", resp.StatusCode, data)
	}
	if len(f.sender.sent) != 1 || f.sender.sent[0].Message != "Ready <now> & safe" {
		t.Fatal("notification content changed")
	}
	resp, data = f.request(t, "GET", "/api/admin/overview", "", "")
	if resp.StatusCode != 200 || bytes.Contains(data, []byte(created.Secret)) || bytes.Contains(data, []byte("hash")) {
		t.Fatalf("overview exposes credentials: %s", data)
	}
	snapshot, err := f.store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Stats.Sent != 1 || snapshot.Keys[0].Sent != 1 || snapshot.Events[0].AppName != "Build app" {
		t.Fatal("delivery history or usage missing")
	}
	if resp, _ := f.request(t, "POST", "/api/notify", payload, testPassword); resp.StatusCode != 401 {
		t.Fatal("dashboard password used as app key")
	}
	if resp, _ := f.request(t, "DELETE", "/api/admin/keys/"+created.Key.ID, "", ""); resp.StatusCode != 204 {
		t.Fatal("could not revoke app key")
	}
	if resp, _ := f.request(t, "POST", "/api/notify", payload, created.Secret); resp.StatusCode != 401 {
		t.Fatal("revoked key accepted")
	}
	if resp, _ := f.request(t, "POST", "/api/admin/logout", "", ""); resp.StatusCode != 200 {
		t.Fatal("logout failed")
	}
	if resp, _ := f.request(t, "GET", "/api/admin/overview", "", created.Secret); resp.StatusCode != 401 {
		t.Fatal("app key allowed admin access")
	}
}

func TestDeliveryFailureIsReportedAndRecorded(t *testing.T) {
	f := setup(t)
	f.sender.err = errors.New("desktop is unavailable")
	_, secret, err := f.store.CreateKey("Monitor")
	if err != nil {
		t.Fatal(err)
	}
	resp, data := f.request(t, "POST", "/api/notify", `{"title":"Alert","message":"Disk full"}`, secret)
	if resp.StatusCode != 502 || !bytes.Contains(data, []byte("desktop is unavailable")) {
		t.Fatalf("delivery error: %d %s", resp.StatusCode, data)
	}
	snapshot, err := f.store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Stats.Failed != 1 || snapshot.Stats.Sent != 0 || snapshot.Events[0].Status != "failed" {
		t.Fatal("failed delivery was not recorded correctly")
	}
}

func TestValidationOriginAndHostProtection(t *testing.T) {
	f := setup(t)
	_, key, err := f.store.CreateKey("Client")
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{`{`, `{"title":"","message":"x"}`, `{"title":"x","message":""}`, `{"title":"x","message":"x","unexpected":true}`, `{"title":"x","message":"x"} {}`, `{"title":"x","message":"x","icon":"/tmp/icon.png"}`} {
		if resp, data := f.request(t, "POST", "/api/notify", payload, key); resp.StatusCode != 400 {
			t.Errorf("bad payload: %d %s", resp.StatusCode, data)
		}
	}
	for _, test := range []struct{ origin, host string }{{"https://other.example", ""}, {"null", ""}, {"", "rebound.example:8787"}} {
		req, _ := http.NewRequest("POST", f.server.URL+"/api/admin/login", strings.NewReader(`{"password":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		if test.origin != "" {
			req.Header.Set("Origin", test.origin)
		}
		if test.host != "" {
			req.Host = test.host
		}
		resp, err := f.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Errorf("accepted foreign origin/host: %+v (%d)", test, resp.StatusCode)
		}
	}
	if len(f.sender.sent) != 0 {
		t.Fatal("invalid input reached desktop notifier")
	}
	req, _ := http.NewRequest("POST", f.server.URL+"/api/notify", strings.NewReader(`{"title":"x","message":"x"}`))
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := f.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 415 {
		t.Fatal("accepted non-JSON content type")
	}
}

func TestPerAppRateLimitAndEmbeddedAssets(t *testing.T) {
	f := setup(t)
	_, first, _ := f.store.CreateKey("First")
	_, second, _ := f.store.CreateKey("Second")
	for i := 0; i < 60; i++ {
		if resp, data := f.request(t, "POST", "/api/notify", `{"title":"Rate test","message":"Test"}`, first); resp.StatusCode != 200 {
			t.Fatalf("request %d: %d %s", i, resp.StatusCode, data)
		}
	}
	if resp, _ := f.request(t, "POST", "/api/notify", `{"title":"Rate test","message":"Test"}`, first); resp.StatusCode != 429 || resp.Header.Get("Retry-After") == "" {
		t.Fatal("rate limit missing")
	}
	if resp, _ := f.request(t, "POST", "/api/notify", `{"title":"Other app","message":"Test"}`, second); resp.StatusCode != 200 {
		t.Fatal("rate limit affected another app")
	}
	if resp, data := f.request(t, "GET", "/", "", ""); resp.StatusCode != 200 || !bytes.Contains(data, []byte("Localert")) || resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("embedded dashboard or security headers missing")
	}
	if resp, _ := f.request(t, "GET", "/missing.js", "", ""); resp.StatusCode != 404 {
		t.Fatal("missing asset not reported")
	}
}

func TestFirstRunSetupIsLocalAndOneTime(t *testing.T) {
	f := newFixture(t, false)
	resp, data := f.request(t, "GET", "/api/admin/setup", "", "")
	if resp.StatusCode != 200 || !bytes.Contains(data, []byte(`"setup_required":true`)) {
		t.Fatalf("fresh setup status: %d %s", resp.StatusCode, data)
	}
	if resp, _ := f.request(t, "POST", "/api/admin/login", `{"password":"not configured"}`, ""); resp.StatusCode != 409 {
		t.Fatal("login did not request first-run setup")
	}
	req := httptest.NewRequest("POST", f.server.URL+"/api/admin/setup", strings.NewReader(`{"password":"remote takeover"}`))
	req.RemoteAddr = "203.0.113.1:12345"
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.server.Config.Handler.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("remote client was allowed to claim the installation")
	}
	if resp, _ := f.request(t, "POST", "/api/admin/setup", `{"password":"short"}`, ""); resp.StatusCode != 400 {
		t.Fatal("short setup password was accepted")
	}
	body, _ := json.Marshal(map[string]string{"password": testPassword})
	if resp, data := f.request(t, "POST", "/api/admin/setup", string(body), ""); resp.StatusCode != 200 {
		t.Fatalf("setup: %d %s", resp.StatusCode, data)
	}
	if resp, _ := f.request(t, "GET", "/api/admin/overview", "", ""); resp.StatusCode != 200 {
		t.Fatal("setup did not establish a dashboard session")
	}
	if resp, _ := f.request(t, "POST", "/api/admin/setup", `{"password":"replacement password"}`, ""); resp.StatusCode != 409 {
		t.Fatal("setup replaced existing credentials")
	}
	if _, valid, err := f.store.VerifyPassword(testPassword); err != nil || !valid {
		t.Fatal("setup password changed unexpectedly")
	}
}

func TestPasswordResetSetupAllowsSameMachineLANClient(t *testing.T) {
	f := setup(t)
	f.login(t)
	if err := f.store.ResetPassword(); err != nil {
		t.Fatal(err)
	}
	status, err := f.store.SetupStatus()
	if err != nil || !status.PasswordRecovery {
		t.Fatalf("reset did not mark password recovery: %+v %v", status, err)
	}
	localIP := ""
	for _, address := range lanAddresses() {
		localIP = address
		break
	}
	if localIP == "" {
		t.Skip("no private IPv4 address on this machine")
	}
	body, _ := json.Marshal(map[string]string{"password": testPassword})
	req := httptest.NewRequest("POST", f.server.URL+"/api/admin/setup", strings.NewReader(string(body)))
	req.RemoteAddr = net.JoinHostPort(localIP, "4242")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.server.Config.Handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("local LAN client setup after reset: %d %s", w.Code, w.Body.Bytes())
	}
	req = httptest.NewRequest("GET", f.server.URL+"/api/admin/setup", nil)
	req.RemoteAddr = net.JoinHostPort("203.0.113.7", "4242")
	w = httptest.NewRecorder()
	f.server.Config.Handler.ServeHTTP(w, req)
	if !bytes.Contains(w.Body.Bytes(), []byte(`"local_setup_allowed":false`)) {
		t.Fatalf("remote client should not be treated as local: %s", w.Body.Bytes())
	}
}

func TestPasswordChangeInvalidatesOtherSessionsAndPreservesKeys(t *testing.T) {
	f := setup(t)
	f.login(t)
	other := f
	otherClient := *f.client
	otherClient.Jar, _ = cookiejar.New(nil)
	other.client = &otherClient
	other.login(t)
	_, key, err := f.store.CreateKey("Autostart app")
	if err != nil {
		t.Fatal(err)
	}
	if resp, _ := f.request(t, "POST", "/api/admin/password", `{"current_password":"wrong","new_password":"replacement password"}`, ""); resp.StatusCode != 403 {
		t.Fatal("password changed without current password")
	}
	body, _ := json.Marshal(map[string]string{"current_password": testPassword, "new_password": "replacement password"})
	if resp, data := f.request(t, "POST", "/api/admin/password", string(body), ""); resp.StatusCode != 200 {
		t.Fatalf("change password: %d %s", resp.StatusCode, data)
	}
	if resp, _ := other.request(t, "GET", "/api/admin/overview", "", ""); resp.StatusCode != 401 {
		t.Fatal("old session survived password change")
	}
	if resp, _ := f.request(t, "GET", "/api/admin/overview", "", ""); resp.StatusCode != 200 {
		t.Fatal("password change did not renew the current session")
	}
	if _, valid, _ := f.store.Authenticate(key); !valid {
		t.Fatal("dashboard password change revoked an app key")
	}
	if err := f.store.ResetPassword(); err != nil {
		t.Fatal(err)
	}
	if resp, _ := f.request(t, "GET", "/api/admin/overview", "", ""); resp.StatusCode != 401 {
		t.Fatal("local password recovery did not invalidate sessions")
	}
}

func TestSettingsAndHistoryRequireAdminAndPersistValidatedPreferences(t *testing.T) {
	f := setup(t)
	_, key, err := f.store.CreateKey("Sender")
	if err != nil {
		t.Fatal(err)
	}
	if response, _ := f.request(t, "PUT", "/api/admin/settings/retention", `{"mode":"forever","count":100,"days":30}`, key); response.StatusCode != 401 {
		t.Fatal("app key changed dashboard settings")
	}
	f.login(t)
	if response, _ := f.request(t, "PUT", "/api/admin/settings/retention", `{"mode":"count","count":0,"days":30}`, ""); response.StatusCode != 400 {
		t.Fatal("invalid retention was accepted")
	}
	for i := 0; i < 5; i++ {
		if err := f.store.Record(store.Event{Title: "Build", Status: "sent", CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if response, data := f.request(t, "PUT", "/api/admin/settings/retention", `{"mode":"count","count":3,"days":30}`, ""); response.StatusCode != 200 {
		t.Fatalf("save retention: %d %s", response.StatusCode, data)
	}
	response, data := f.request(t, "GET", "/api/admin/history?limit=1&offset=1&search=build", "", "")
	var page store.HistoryPage
	if err := json.Unmarshal(data, &page); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || page.Total != 3 || len(page.Events) != 1 {
		t.Fatalf("paginated history: %d %s", response.StatusCode, data)
	}
	if response, _ := f.request(t, "PUT", "/api/admin/settings/updates", `{"url":"http://external.example/feed.json"}`, ""); response.StatusCode != 400 {
		t.Fatal("insecure external release source accepted")
	}
	if response, _ := f.request(t, "PUT", "/api/admin/settings/updates", `{"url":"https://releases.example/feed.json"}`, ""); response.StatusCode != 200 {
		t.Fatal("valid update source rejected")
	}
	snapshot, err := f.store.Snapshot()
	if err != nil || snapshot.HistoryTotal != 3 || snapshot.Stats.Sent != 5 || snapshot.Settings.UpdateFeed == "" {
		t.Fatal("preferences pruned counters or failed to persist")
	}
}

func TestNetworkSettingRequiresAdminValidAddressAndUnlockedFlag(t *testing.T) {
	f := setup(t)
	if response, _ := f.request(t, "PUT", "/api/admin/settings/network", `{"listen_address":"0.0.0.0:8787"}`, ""); response.StatusCode != 401 {
		t.Fatal("anonymous network change accepted")
	}
	f.login(t)
	if response, _ := f.request(t, "PUT", "/api/admin/settings/network", `{"listen_address":"10.0.0.1:8787"}`, ""); response.StatusCode != 400 {
		t.Fatal("arbitrary interface accepted")
	}
	// The test fixture has no updater, so a valid address cannot restart.
	if response, _ := f.request(t, "PUT", "/api/admin/settings/network", `{"listen_address":"0.0.0.0:8787"}`, ""); response.StatusCode != 503 {
		t.Fatal("network change accepted without restart support")
	}
	if settings, _ := f.store.Settings(); settings.ListenAddress != store.DefaultListenAddress {
		t.Fatal("address saved even though restart was unavailable")
	}
	response, data := f.request(t, "GET", "/api/admin/overview", "", "")
	if response.StatusCode != 200 || !bytes.Contains(data, []byte(`"lan_addresses":`)) || !bytes.Contains(data, []byte(`"address_locked":false`)) {
		t.Fatalf("overview missing network details: %s", data)
	}
}
