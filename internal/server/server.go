package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"localert/internal/notify"
	"localert/internal/store"
	"localert/internal/updater"
)

const cookieName = "localert_session"

type Options struct {
	Store   *store.Store
	Sender  notify.Sender
	Icons   *notify.Icons
	Assets  fs.FS
	Version string
	Address string
	Logger  *slog.Logger
	Updater *updater.Manager
	// AddressLocked is true when --addr was passed, which overrides the saved setting.
	AddressLocked bool
}

type session struct {
	expires    time.Time
	credential string
}
type window struct {
	start time.Time
	count int
}

type Server struct {
	opts       Options
	started    time.Time
	mu         sync.Mutex
	sessions   map[[32]byte]session
	rates      map[string]window
	deliveries chan struct{}
	handler    http.Handler
}

func New(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	s := &Server{opts: opts, started: time.Now().UTC(), sessions: make(map[[32]byte]session), rates: make(map[string]window), deliveries: make(chan struct{}, 4)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/notify", s.notification)
	mux.HandleFunc("GET /api/admin/setup", s.setupStatus)
	mux.HandleFunc("POST /api/admin/setup", s.setup)
	mux.HandleFunc("POST /api/admin/login", s.login)
	mux.Handle("GET /api/admin/session", s.admin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]bool{"authenticated": true})
	})))
	mux.Handle("POST /api/admin/logout", s.admin(http.HandlerFunc(s.logout)))
	mux.Handle("POST /api/admin/password", s.admin(http.HandlerFunc(s.changePassword)))
	mux.Handle("GET /api/admin/overview", s.admin(http.HandlerFunc(s.overview)))
	mux.Handle("GET /api/admin/keys", s.admin(http.HandlerFunc(s.listKeys)))
	mux.Handle("POST /api/admin/keys", s.admin(http.HandlerFunc(s.createKey)))
	mux.Handle("DELETE /api/admin/keys/{id}", s.admin(http.HandlerFunc(s.revokeKey)))
	mux.Handle("GET /api/admin/history", s.admin(http.HandlerFunc(s.history)))
	mux.Handle("DELETE /api/admin/history", s.admin(http.HandlerFunc(s.clearHistory)))
	mux.Handle("POST /api/admin/test", s.admin(http.HandlerFunc(s.testNotification)))
	mux.Handle("PUT /api/admin/settings/retention", s.admin(http.HandlerFunc(s.saveRetention)))
	mux.Handle("PUT /api/admin/settings/updates", s.admin(http.HandlerFunc(s.saveUpdateFeed)))
	mux.Handle("PUT /api/admin/settings/network", s.admin(http.HandlerFunc(s.saveNetwork)))
	mux.Handle("GET /api/admin/updates", s.admin(http.HandlerFunc(s.updateStatus)))
	mux.Handle("POST /api/admin/updates/check", s.admin(http.HandlerFunc(s.checkUpdates)))
	mux.Handle("POST /api/admin/updates/install", s.admin(http.HandlerFunc(s.installUpdate)))
	mux.Handle("POST /api/admin/restart", s.admin(http.HandlerFunc(s.restart)))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "not_found", "API endpoint not found") })
	mux.HandleFunc("/", s.assets)
	s.handler = s.headers(mux)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

func (s *Server) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if !s.validHost(r.Host) {
			apiError(w, 403, "invalid_host", "use the configured address or localhost to connect")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				if err != nil || u.Scheme != scheme || !strings.EqualFold(u.Host, r.Host) || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
					apiError(w, 403, "invalid_origin", "cross-origin requests are not allowed")
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				apiError(w, 403, "invalid_origin", "cross-site requests are not allowed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Literal hosts allow LAN use with --addr 0.0.0.0:8787 while rejecting arbitrary
// Host headers (including DNS rebinding against the default loopback listener).
func (s *Server) validHost(authority string) bool {
	host, port, err := net.SplitHostPort(authority)
	if err != nil {
		return false
	}
	configured, configuredPort, err := net.SplitHostPort(s.opts.Address)
	if err != nil || port != configuredPort {
		return false
	}
	if strings.EqualFold(host, configured) || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	listenIP := net.ParseIP(configured)
	if configured == "" || (listenIP != nil && listenIP.IsUnspecified()) {
		return true
	}
	return ip.IsLoopback() && listenIP != nil && listenIP.IsLoopback()
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok", "version": s.opts.Version, "instance_id": s.started.Format(time.RFC3339Nano)})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.allow("login", 30) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "rate_limited", "too many login attempts; try again in a minute")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body, 4096) {
		return
	}
	status, err := s.opts.Store.SetupStatus()
	if err != nil {
		s.internalError(w, err)
		return
	}
	if status.SetupRequired {
		apiError(w, 409, "setup_required", "create your dashboard password before signing in")
		return
	}
	version, valid, err := s.opts.Store.VerifyPassword(body.Password)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if !valid {
		apiError(w, 401, "unauthorized", "incorrect password")
		return
	}
	s.startSession(w, r, version)
}

func loopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// localRequest is true when the TCP client is this computer (loopback or a local interface IP).
// Needed after --reset-password when the dashboard is opened via a LAN URL on the same machine.
func localRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	for _, address := range lanAddresses() {
		if ip.String() == address {
			return true
		}
	}
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, address := range interfaces {
		network, ok := address.(*net.IPNet)
		if !ok {
			continue
		}
		local := network.IP.To4()
		if local != nil && local.Equal(ip) {
			return true
		}
	}
	// Host browser on localhost while RemoteAddr is a container bridge gateway (Docker, etc.).
	if hostLooksLocalhost(r) && dockerBridgeGateway(ip) {
		return true
	}
	return false
}

func hostLooksLocalhost(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func dockerBridgeGateway(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 && ip[3] == 1
}

func (s *Server) setupStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.opts.Store.SetupStatus()
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, 200, struct {
		store.SetupStatus
		LocalSetupAllowed bool `json:"local_setup_allowed"`
	}{status, localRequest(r)})
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if !localRequest(r) {
		apiError(w, 403, "local_setup_only", "open Localert on this computer (http://127.0.0.1) to finish setup after a password reset")
		return
	}
	if !s.allow("setup", 10) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "rate_limited", "too many setup attempts; try again in a minute")
		return
	}
	var body struct {
		Password    string `json:"password"`
		LegacyToken string `json:"legacy_token"`
	}
	if !decodeJSON(w, r, &body, 4096) {
		return
	}
	err := s.opts.Store.Setup(body.Password, body.LegacyToken)
	switch {
	case errors.Is(err, store.ErrAlreadyConfigured):
		apiError(w, 409, "already_configured", err.Error())
		return
	case errors.Is(err, store.ErrInvalidPassword):
		apiError(w, 400, "invalid_password", err.Error())
		return
	case errors.Is(err, store.ErrLegacyToken):
		apiError(w, 401, "legacy_token_required", err.Error())
		return
	case err != nil:
		s.internalError(w, err)
		return
	}
	version, err := s.opts.Store.CredentialVersion()
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.startSession(w, r, version)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	if !s.allow("password", 10) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "rate_limited", "too many password attempts; try again in a minute")
		return
	}
	var body struct {
		Current     string `json:"current_password"`
		Replacement string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body, 4096) {
		return
	}
	err := s.opts.Store.ChangePassword(body.Current, body.Replacement)
	if errors.Is(err, store.ErrInvalidPassword) {
		apiError(w, 400, "invalid_password", err.Error())
		return
	}
	if errors.Is(err, store.ErrIncorrectPassword) {
		apiError(w, 403, "incorrect_password", err.Error())
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	version, err := s.opts.Store.CredentialVersion()
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.mu.Lock()
	clear(s.sessions)
	s.mu.Unlock()
	s.startSession(w, r, version)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, version string) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		s.internalError(w, err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	now := time.Now()
	s.mu.Lock()
	for digest, value := range s.sessions {
		if now.After(value.expires) {
			delete(s.sessions, digest)
		}
	}
	if len(s.sessions) >= 64 {
		var oldest [32]byte
		first := true
		for digest, value := range s.sessions {
			if first || value.expires.Before(s.sessions[oldest].expires) {
				oldest = digest
				first = false
			}
		}
		delete(s.sessions, oldest)
	}
	s.sessions[sha256.Sum256([]byte(token))] = session{expires: now.Add(24 * time.Hour), credential: version}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/api/admin", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: 86400})
	writeJSON(w, 200, map[string]bool{"authenticated": true})
}

func (s *Server) admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			apiError(w, 401, "unauthorized", "sign in with your dashboard password")
			return
		}
		digest := sha256.Sum256([]byte(cookie.Value))
		s.mu.Lock()
		value, ok := s.sessions[digest]
		if ok && time.Now().After(value.expires) {
			delete(s.sessions, digest)
			ok = false
		}
		s.mu.Unlock()
		if !ok {
			apiError(w, 401, "unauthorized", "your session expired; sign in again")
			return
		}
		version, err := s.opts.Store.CredentialVersion()
		if err != nil {
			s.internalError(w, err)
			return
		}
		if version == "" || version != value.credential {
			s.mu.Lock()
			delete(s.sessions, digest)
			s.mu.Unlock()
			apiError(w, 401, "unauthorized", "your password changed; sign in again")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(cookieName)
	s.mu.Lock()
	delete(s.sessions, sha256.Sum256([]byte(cookie.Value)))
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/api/admin", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	writeJSON(w, 200, map[string]bool{"authenticated": false})
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.opts.Store.Snapshot()
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, 200, struct {
		store.Snapshot
		Version       string    `json:"version"`
		Platform      string    `json:"platform"`
		StartedAt     time.Time `json:"started_at"`
		Address       string    `json:"address"`
		DatabasePath  string    `json:"database_path"`
		AddressLocked bool      `json:"address_locked"`
		LANAddresses  []string  `json:"lan_addresses"`
	}{snapshot, s.opts.Version, runtime.GOOS, s.started, s.opts.Address, s.opts.Store.Path(), s.opts.AddressLocked, lanAddresses()})
}

// lanAddresses lists this computer's private IPv4 addresses for display.
func lanAddresses() []string {
	result := []string{}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return result
	}
	for _, address := range addresses {
		network, ok := address.(*net.IPNet)
		if !ok {
			continue
		}
		ip := network.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		result = append(result, ip.String())
	}
	return result
}

func (s *Server) saveNetwork(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListenAddress string `json:"listen_address"`
	}
	if !decodeJSON(w, r, &body, 4096) {
		return
	}
	if s.opts.AddressLocked {
		apiError(w, 409, "address_locked", "Localert was started with --addr; remove that flag to manage the address here")
		return
	}
	if err := store.ValidateListenAddress(body.ListenAddress); err != nil {
		apiError(w, 400, "invalid_address", err.Error())
		return
	}
	if !s.hasUpdater(w) {
		return
	}
	if s.opts.Updater.Busy() {
		apiError(w, 409, "restart_busy", updater.ErrBusy.Error())
		return
	}
	if err := s.opts.Store.SetListenAddress(body.ListenAddress); err != nil {
		s.internalError(w, err)
		return
	}
	if err := s.opts.Updater.RequestRestart(); err != nil {
		apiError(w, 409, "restart_busy", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"listen_address": body.ListenAddress, "status": "restarting"})
}

func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.opts.Store.Snapshot()
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"keys": snapshot.Keys})
}

func (s *Server) createKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &body, 4096) {
		return
	}
	key, secret, err := s.opts.Store.CreateKey(body.Name)
	if errors.Is(err, store.ErrDuplicateName) {
		apiError(w, 409, "duplicate_name", err.Error())
		return
	}
	if errors.Is(err, store.ErrInvalidName) || errors.Is(err, store.ErrKeyLimit) {
		apiError(w, 400, "invalid_key", err.Error())
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"key": key, "secret": secret})
}

func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request) {
	err := s.opts.Store.RevokeKey(r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, 404, "not_found", err.Error())
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	limit, offset := 25, 0
	var err error
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil {
			apiError(w, 400, "invalid_pagination", "invalid page size")
			return
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		offset, err = strconv.Atoi(v)
		if err != nil {
			apiError(w, 400, "invalid_pagination", "invalid offset")
			return
		}
	}
	status, search := r.URL.Query().Get("status"), r.URL.Query().Get("search")
	if limit < 1 || limit > 100 || offset < 0 || len(search) > 256 || (status != "" && status != "sent" && status != "failed") {
		apiError(w, 400, "invalid_pagination", "use limit 1–100, a non-negative offset, and sent or failed status")
		return
	}
	page, err := s.opts.Store.History(limit, offset, status, search)
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, 200, page)
}

func (s *Server) saveRetention(w http.ResponseWriter, r *http.Request) {
	var value store.Retention
	if !decodeJSON(w, r, &value, 4096) {
		return
	}
	if err := s.opts.Store.SetRetention(value); err != nil {
		if errors.Is(err, store.ErrInvalidRetention) {
			apiError(w, 400, "invalid_retention", err.Error())
			return
		}
		s.internalError(w, err)
		return
	}
	writeJSON(w, 200, value)
}

func (s *Server) saveUpdateFeed(w http.ResponseWriter, r *http.Request) {
	var value struct {
		URL string `json:"url"`
	}
	if !decodeJSON(w, r, &value, 4096) {
		return
	}
	value.URL = strings.TrimSpace(value.URL)
	if err := updater.ValidateURL(value.URL); err != nil {
		apiError(w, 400, "invalid_update_source", err.Error())
		return
	}
	if s.opts.Updater != nil && s.opts.Updater.Busy() {
		apiError(w, 409, "update_busy", updater.ErrBusy.Error())
		return
	}
	if err := s.opts.Store.SetUpdateFeed(value.URL); err != nil {
		s.internalError(w, err)
		return
	}
	if s.opts.Updater != nil {
		s.opts.Updater.Invalidate()
	}
	writeJSON(w, 200, map[string]string{"url": value.URL})
}

func (s *Server) hasUpdater(w http.ResponseWriter) bool {
	if s.opts.Updater == nil {
		apiError(w, 503, "unavailable", "restart and update controls are unavailable in this runtime")
		return false
	}
	return true
}
func (s *Server) updateStatus(w http.ResponseWriter, r *http.Request) {
	if s.hasUpdater(w) {
		writeJSON(w, 200, s.opts.Updater.Status())
	}
}
func (s *Server) checkUpdates(w http.ResponseWriter, r *http.Request) {
	if !s.hasUpdater(w) {
		return
	}
	if !s.allow("update-check", 20) {
		apiError(w, 429, "rate_limited", "try checking again in a minute")
		return
	}
	status, err := s.opts.Updater.Check(r.Context())
	if errors.Is(err, updater.ErrBusy) {
		apiError(w, 409, "update_busy", err.Error())
		return
	}
	// Failed checks are a visible status, never a misleading up-to-date response.
	writeJSON(w, 200, status)
}
func (s *Server) installUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.hasUpdater(w) {
		return
	}
	if err := s.opts.Updater.Install(); err != nil {
		apiError(w, 409, "update_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, s.opts.Updater.Status())
}
func (s *Server) restart(w http.ResponseWriter, r *http.Request) {
	if !s.hasUpdater(w) {
		return
	}
	if err := s.opts.Updater.RequestRestart(); err != nil {
		apiError(w, 409, "restart_busy", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "restarting"})
}

func (s *Server) clearHistory(w http.ResponseWriter, r *http.Request) {
	if err := s.opts.Store.ClearHistory(); err != nil {
		s.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) notification(w http.ResponseWriter, r *http.Request) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		w.Header().Set("WWW-Authenticate", "Bearer")
		apiError(w, 401, "unauthorized", "provide your app key as Authorization: Bearer la_...")
		return
	}
	key, ok, err := s.opts.Store.Authenticate(parts[1])
	if err != nil {
		s.internalError(w, err)
		return
	}
	if !ok {
		apiError(w, 401, "unauthorized", "invalid or revoked app key")
		return
	}
	s.deliver(w, r, key.ID, key.Name)
}

func (s *Server) testNotification(w http.ResponseWriter, r *http.Request) {
	s.deliver(w, r, "dashboard", "Dashboard test")
}

func (s *Server) deliver(w http.ResponseWriter, r *http.Request, appID, appName string) {
	var n notify.Notification
	if !decodeJSON(w, r, &n, 2<<20) {
		return
	}
	n.Title, n.Message, n.Icon = strings.TrimSpace(n.Title), strings.TrimSpace(n.Message), strings.TrimSpace(n.Icon)
	if n.Title == "" || utf8.RuneCountInString(n.Title) > 128 || strings.ContainsAny(n.Title, "\x00\r\n") {
		apiError(w, 400, "invalid_title", "title must be 1–128 characters on a single line")
		return
	}
	if n.Message == "" || utf8.RuneCountInString(n.Message) > 4096 || strings.ContainsRune(n.Message, '\x00') {
		apiError(w, 400, "invalid_message", "message must be 1–4096 characters")
		return
	}
	if !s.allow(appID, 60) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "rate_limited", "maximum of 60 notifications per minute per app")
		return
	}
	// A shared budget keeps many apps together from flooding the desktop or CPU.
	if !s.allow("all-apps", 120) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "rate_limited", "maximum of 120 notifications per minute across all apps")
		return
	}
	select {
	case s.deliveries <- struct{}{}:
		defer func() { <-s.deliveries }()
	default:
		w.Header().Set("Retry-After", "1")
		apiError(w, 429, "busy", "notification service is busy; try again shortly")
		return
	}
	id, err := store.NewEventID()
	if err != nil {
		s.internalError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	path, err := s.opts.Icons.Resolve(ctx, n.Icon)
	if err != nil {
		apiError(w, 400, "invalid_icon", err.Error())
		return
	}
	if n.Icon == "" {
		n.Icon = "bell"
	}
	err = s.opts.Sender.Send(ctx, n, path)
	event := store.Event{ID: id, AppID: appID, AppName: appName, Title: n.Title, Message: n.Message, Icon: n.Icon, Status: "sent", CreatedAt: time.Now().UTC()}
	// Retain a compact icon label in history instead of a potentially large data URL.
	if strings.HasPrefix(n.Icon, "data:") {
		event.Icon = "custom"
	}
	if err != nil {
		event.Status = "failed"
		event.Error = err.Error()
		if len(event.Error) > 1024 {
			event.Error = event.Error[:1024]
		}
	}
	recordErr := s.opts.Store.Record(event)
	if recordErr != nil {
		s.opts.Logger.Error("persist notification history", "error", recordErr)
	}
	if err != nil {
		s.opts.Logger.Warn("notification delivery failed", "app", appName, "error", err)
		writeJSON(w, 502, map[string]any{"error": "desktop notification could not be delivered; run Localert in a signed-in desktop session", "code": "delivery_failed", "detail": event.Error, "id": id})
		return
	}
	response := map[string]any{"id": id, "status": "sent", "created_at": event.CreatedAt}
	if recordErr != nil {
		response["warning"] = "notification delivered, but history could not be saved"
	}
	writeJSON(w, 200, response)
}

func (s *Server) allow(id string, limit int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if len(s.rates) > 128 {
		for key, value := range s.rates {
			if now.Sub(value.start) >= time.Minute {
				delete(s.rates, key)
			}
		}
	}
	value := s.rates[id]
	if now.Sub(value.start) >= time.Minute {
		value = window{start: now}
	}
	if value.count >= limit {
		return false
	}
	value.count++
	s.rates[id] = value
	return true
}

func (s *Server) assets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		apiError(w, 405, "method_not_allowed", "method not allowed")
		return
	}
	assetPath := strings.TrimPrefix(r.URL.Path, "/")
	if assetPath == "" {
		assetPath = "index.html"
	}
	if !fs.ValidPath(assetPath) {
		http.NotFound(w, r)
		return
	}
	// Stream straight from the embedded file: no per-request copy of large bundles.
	file, err := s.opts.Assets.Open(assetPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	content, seekable := file.(io.ReadSeeker)
	if err != nil || info.IsDir() || !seekable {
		http.NotFound(w, r)
		return
	}
	if assetPath == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", `"`+s.opts.Version+`"`)
	} else if strings.HasPrefix(assetPath, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	if kind := mime.TypeByExtension(path.Ext(assetPath)); kind != "" {
		w.Header().Set("Content-Type", kind)
	}
	http.ServeContent(w, r, assetPath, time.Time{}, content)
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.opts.Logger.Error("request failed", "error", err)
	apiError(w, 500, "internal_error", "could not save local data; check the Localert terminal")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any, limit int64) bool {
	typeName, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typeName != "application/json" {
		apiError(w, 415, "invalid_content_type", "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			apiError(w, 413, "body_too_large", "request body is too large")
		} else {
			apiError(w, 400, "invalid_json", "invalid JSON request: "+err.Error())
		}
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		apiError(w, 400, "invalid_json", "request must contain a single JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func apiError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": message, "code": code})
}
