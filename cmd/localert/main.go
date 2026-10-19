package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"localert/internal/logfile"
	"localert/internal/notify"
	"localert/internal/server"
	"localert/internal/store"
	"localert/internal/updater"
	"localert/web"
)

var version = "0.5.5"
var defaultUpdateFeed = "https://github.com/TrueFix/localert/releases/latest/download/update-manifest.json"

func main() {
	if err := run(); err != nil {
		slog.Error("Localert stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", store.DefaultListenAddress, "HTTP listen address; overrides the Network setting in the dashboard")
	dataDir := flag.String("data-dir", "", "state directory (default: user config directory / localert)")
	open := flag.Bool("open", false, "open the dashboard in your default browser")
	noOpen := flag.Bool("no-open", false, "disable automatic browser opening during first-run setup")
	resetPassword := flag.Bool("reset-password", false, "reset the dashboard password while keeping app keys and history")
	showVersion := flag.Bool("version", false, "print version and exit")
	pidFile := flag.String("pid-file", "", "optional PID file used by the desktop launcher")
	logFile := flag.String("log-file", "", "write logs to this file, capped at 1 MiB plus one backup")
	containerMode := flag.Bool("container", false, "use container-managed upgrades and JSON diagnostic logs")
	flag.Parse()
	explicitAddr := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			explicitAddr = true
		}
	})
	if *showVersion {
		fmt.Println("Localert", version)
		return nil
	}
	tuneRuntime()
	var out io.Writer = os.Stdout
	var logOut io.Writer = os.Stderr
	if *logFile != "" {
		var err error
		*logFile, err = filepath.Abs(*logFile)
		if err != nil {
			return err
		}
		writer, err := logfile.Open(*logFile)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer writer.Close()
		out, logOut = writer, writer
	}
	if *containerMode {
		slog.SetDefault(slog.New(slog.NewJSONHandler(logOut, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(logOut, nil)))
	}
	log.SetOutput(logOut)
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments; run localert --help for usage")
	}
	if *open && *noOpen {
		return errors.New("use either --open or --no-open")
	}
	if *dataDir == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("find config directory: %w (or set --data-dir)", err)
		}
		*dataDir = filepath.Join(dir, "localert")
	}
	absolute, err := filepath.Abs(*dataDir)
	if err != nil {
		return err
	}
	state, err := store.Open(absolute)
	if err != nil {
		return fmt.Errorf("open local state: %w", err)
	}
	defer state.Close()
	if *resetPassword {
		if err := state.ResetPassword(); err != nil {
			return fmt.Errorf("reset dashboard password: %w", err)
		}
		fmt.Fprintf(out, "Dashboard password reset for %s.\nApp keys and history were kept.\nOpen the dashboard on this computer to choose a new password (http://127.0.0.1:8787). If Localert is already running, refresh the page or sign in again.\n", absolute)
		return nil
	}
	if err := updater.ValidateURL(defaultUpdateFeed); err != nil {
		return fmt.Errorf("invalid publisher update feed: %w", err)
	}
	if err := state.InitializeUpdateFeed(defaultUpdateFeed); err != nil {
		return err
	}
	status, err := state.SetupStatus()
	if err != nil {
		return err
	}
	listenAddr := *addr
	if !explicitAddr {
		if settings, err := state.Settings(); err == nil && settings.ListenAddress != "" {
			listenAddr = settings.ListenAddress
		}
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil && !explicitAddr && listenAddr != store.DefaultListenAddress {
		// Never lock users out: a saved address that cannot bind falls back to local-only.
		slog.Warn("saved listen address unavailable; using local-only address", "address", listenAddr, "error", err)
		listenAddr = store.DefaultListenAddress
		listener, err = net.Listen("tcp", listenAddr)
	}
	if err != nil {
		return fmt.Errorf("listen on %s: %w", listenAddr, err)
	}
	defer listener.Close()
	if *pidFile != "" {
		*pidFile, err = filepath.Abs(*pidFile)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(*pidFile), 0700); err != nil {
			return err
		}
		pid := strconv.Itoa(os.Getpid())
		if err := os.WriteFile(*pidFile, []byte(pid+"\n"), 0600); err != nil {
			return err
		}
		defer func() {
			value, _ := os.ReadFile(*pidFile)
			if string(value) == pid+"\n" {
				_ = os.Remove(*pidFile)
			}
		}()
	}
	icons, err := notify.NewIcons(absolute)
	if err != nil {
		return fmt.Errorf("prepare icon cache: %w", err)
	}
	actualAddr := listener.Addr().String()
	if requestedHost, _, _ := net.SplitHostPort(listenAddr); requestedHost == "0.0.0.0" {
		// Go listens dual-stack ([::]); report the address the user chose.
		_, port, _ := net.SplitHostPort(actualAddr)
		actualAddr = net.JoinHostPort("0.0.0.0", port)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	restartRequests := make(chan struct{}, 1)
	updates := updater.New(updater.Options{Version: version, Executable: executable, Container: *containerMode,
		Feed: func() (string, error) { settings, err := state.Settings(); return settings.UpdateFeed, err },
		Restart: func() error {
			select {
			case restartRequests <- struct{}{}:
				return nil
			default:
				return updater.ErrBusy
			}
		},
	})
	defer updates.Close()
	handler := server.New(server.Options{Store: state, Sender: notify.Native{}, Icons: icons, Assets: web.Assets(), Version: version, Address: actualAddr, Updater: updates, AddressLocked: explicitAddr})
	httpServer := &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10,
		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn),
	}
	host, port, _ := net.SplitHostPort(actualAddr)
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	dashboardURL := "http://" + net.JoinHostPort(host, port)
	fmt.Fprintf(out, "\n  Localert %s · native desktop notifications\n\n  Dashboard    %s\n  Notify API   %s/api/notify\n  Listening    %s\n  Database     %s\n", version, dashboardURL, dashboardURL, actualAddr, state.Path())
	if status.SetupRequired {
		fmt.Fprintln(out, "\n  First-run setup: open the dashboard on this computer to create your password.")
	} else {
		fmt.Fprintln(out, "\n  Sign in to the dashboard with your password to manage app keys.")
	}
	if status.LegacyTokenRequired {
		fmt.Fprintln(out, "  Existing data was imported. Use your previous admin token from state.json to finish upgrading.")
	}
	fmt.Fprintln(out, "  Press Ctrl+C to stop.")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go maintain(ctx, state, icons, func() {
		select {
		case restartRequests <- struct{}{}:
		default:
		}
	})
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- httpServer.Serve(limitListener(listener, maxConnections)) }()
	if *open || (status.SetupRequired && !*noOpen) {
		if err := openBrowser(dashboardURL); err != nil {
			slog.Warn("open the dashboard URL manually", "error", err)
		}
	}
	select {
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdown)
	case <-restartRequests:
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdown); err != nil {
			return err
		}
		stop()
		updates.Close()
		if err := state.Close(); err != nil {
			return err
		}
		// Without an explicit --addr, the restarted process reads the saved Network setting.
		args := []string{"--data-dir", absolute, "--no-open"}
		if explicitAddr {
			args = append(args, "--addr", actualAddr)
		}
		if *containerMode {
			args = append(args, "--container")
		}
		if *pidFile != "" {
			args = append(args, "--pid-file", *pidFile)
		}
		if *logFile != "" {
			args = append(args, "--log-file", *logFile)
		}
		if err := updater.Restart(executable, args); err != nil {
			return fmt.Errorf("restart Localert: %w", err)
		}
		return nil
	}
	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

const (
	maxConnections    = 64               // open HTTP connections; extra clients wait
	memoryLimit       = 96 << 20         // soft Go heap limit; normal use stays far below
	memoryEmergency   = 256 << 20        // a leak this large restarts the process
	maintenancePeriod = 5 * time.Minute  // a few milliseconds of work, then sleep
	restartGrace      = 10 * time.Minute // never restart-loop a freshly started process
)

// tuneRuntime keeps an always-on background service small. Environment
// variables (GOMAXPROCS, GOMEMLIMIT) still take precedence.
func tuneRuntime() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(min(2, runtime.NumCPU()))
	}
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(memoryLimit)
	}
}

// maintain runs rare housekeeping. A panic here is logged, never fatal.
func maintain(ctx context.Context, state *store.Store, icons *notify.Icons, restart func()) {
	started := time.Now()
	period := maintenancePeriod
	if value, err := time.ParseDuration(os.Getenv("LOCALERT_MAINTENANCE_INTERVAL")); err == nil && value >= time.Second {
		period = value // test hook for soak tests
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("maintenance recovered from a panic", "panic", recovered, "stack", string(debug.Stack()))
				}
			}()
			if err := state.PruneHistory(); err != nil {
				slog.Error("prune notification history", "error", err)
			}
			if err := icons.Prune(); err != nil {
				slog.Warn("prune icon cache", "error", err)
			}
			// Give memory used by bursts of notifications back to the OS.
			debug.FreeOSMemory()
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			if memory.HeapAlloc > memoryEmergency && time.Since(started) > restartGrace {
				slog.Error("memory use is unexpectedly high; restarting Localert", "heap_mb", memory.HeapAlloc>>20)
				restart()
			}
		}()
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

// limitListener caps concurrent connections so a misbehaving client on the
// network cannot exhaust file descriptors or memory.
func limitListener(listener net.Listener, limit int) net.Listener {
	return &limitedListener{Listener: listener, slots: make(chan struct{}, limit)}
}

type limitedListener struct {
	net.Listener
	slots chan struct{}
}

func (l *limitedListener) Accept() (net.Conn, error) {
	l.slots <- struct{}{}
	conn, err := l.Listener.Accept()
	if err != nil {
		<-l.slots
		return nil, err
	}
	return &limitedConn{Conn: conn, release: func() { <-l.slots }}, nil
}

type limitedConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *limitedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}
