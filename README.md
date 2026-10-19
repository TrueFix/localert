# Localert

Turn an HTTP request into a native desktop notification. Localert runs a small local server with a web dashboard for managing a separate API key for every app.

**Distribute a single executable.** The dashboard’s HTML, CSS, JavaScript, built-in icons, and SQLite engine are included in the binary. Node.js, Python, an SQLite installation, a separate web server, and an assets folder are not needed to run it.

## Run

```sh
./bin/localert --open
```

On the first run, Localert creates its SQLite database and automatically opens **http://127.0.0.1:8787**. Choose a dashboard password (8–128 characters), confirm it, and create an app key. Save the key when it is shown: its secret is displayed only once.

After setup, Localert starts quietly. Open the dashboard and sign in with your password whenever you want to manage apps. Notifications keep working while the browser is closed and do not require a dashboard login on every restart. Use `--open` to open the browser explicitly or `--no-open` to suppress automatic first-run browser opening.

On Windows, run the Windows executable instead. Keep Localert running in your signed-in desktop session.

## Install and start automatically at sign-in

The per-user installers copy the executable into a permanent location, configure desktop-login autostart, and start Localert immediately. They do not require administrator privileges. The browser opens if password setup is still needed; subsequent sign-ins start the service in the background.

`make build` and `make release` copy the installer helpers into `bin/`. From this project folder:

### Linux or macOS

```sh
sh bin/install.sh
```

To select a downloaded binary explicitly:

```sh
sh scripts/install.sh --binary /path/to/localert-linux-amd64
# On macOS, select localert-macos-arm64 or localert-macos-amd64 instead.
```

The installer detects the OS and CPU and can also find the matching release binary beside itself.

### Windows

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\bin\install.ps1
```

Or select the downloaded executable:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install.ps1 -BinaryPath C:\Downloads\localert-windows-amd64.exe
```

The execution-policy option applies to this installer process. The installer adds a hidden background launcher at sign-in and an **Open Localert** link in the user’s Start menu (named Localert).

| OS | Installed executable | Autostart mechanism |
| --- | --- | --- |
| Linux | `~/.local/bin/localert` | `$XDG_CONFIG_HOME/autostart/localert.desktop` (normally `~/.config/autostart/`) |
| macOS | `~/.local/bin/localert` | `~/Library/LaunchAgents/dev.localert.agent.plist`, in the graphical Aqua session |
| Windows | `%LocalAppData%\Programs\Localert\localert.exe` | `Localert.lnk` in the current user’s Startup folder |

Linux uses XDG-compatible desktop autostart. macOS uses a LaunchAgent with automatic restart. Windows uses built-in Windows PowerShell to launch the executable with its console hidden. Autostart runs **when the user signs in**, so Localert has access to that user’s desktop notifications.

Logs: `$XDG_STATE_HOME/localert/localert.log` (normally `~/.local/state/localert/localert.log`) on Linux; `~/Library/Logs/Localert/localert.log` on macOS; `%LocalAppData%\Localert\logs\` on Windows.

### Installer options and uninstall

```sh
sh bin/install.sh --no-start                       # Install for the next sign-in
sh bin/install.sh --data-dir "/path/to/my data"    # Custom database/icon directory
sh bin/install.sh --uninstall                      # Remove executable and autostart
```

```powershell
.\bin\install.ps1 -NoStart
.\bin\install.ps1 -DataDir "D:\LocalertData"
.\bin\install.ps1 -Uninstall
```

Uninstall preserves the database, icons, and logs. Reinstalling with the same data directory preserves the password, app keys, counters, and history. Unix users can set `LOCALERT_INSTALL_DIR`; Windows users can pass `-InstallDir` to choose another executable directory. Use the same executable-directory override when uninstalling.

The installers are optional helpers. The application itself is still one self-contained executable.

## Send a notification

```sh
curl -X POST http://127.0.0.1:8787/api/notify \
  -H "Authorization: Bearer YOUR_APP_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "icon": "success",
    "title": "Build complete",
    "message": "Your project is ready to ship."
  }'
```

Successful response (`200 OK`):

```json
{
  "id": "evt_…",
  "status": "sent",
  "created_at": "2026-01-01T12:00:00Z"
}
```

`sent` means the operating system accepted the notification. Your OS controls its appearance and whether a banner is visible, including Do Not Disturb settings.

### Python

```python
import os
import requests

response = requests.post(
    "http://127.0.0.1:8787/api/notify",
    headers={"Authorization": f"Bearer {os.environ['LOCALERT_API_KEY']}"},
    json={"icon": "info", "title": "Backup finished", "message": "All files are up to date."},
    timeout=15,
)
response.raise_for_status()
```

### JavaScript / Node.js

```js
const response = await fetch("http://127.0.0.1:8787/api/notify", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${process.env.LOCALERT_API_KEY}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    icon: "warning",
    title: "Job needs attention",
    message: "The import completed with 3 skipped records.",
  }),
});
if (!response.ok) throw new Error(await response.text());
```

### Request fields

| Field | Required | Description |
| --- | --- | --- |
| `title` | Yes | 1–128 characters, single line. |
| `message` | Yes | 1–4,096 characters of plain text; line breaks supported. |
| `icon` | No | `bell` (default), `info`, `success`, `warning`, `error`, an HTTPS image URL, or a `data:image/png;base64,…` / `data:image/jpeg;base64,…` URL. |

Custom icons must be PNG or JPEG, at most **1 MiB** and **1024 × 1024** pixels. HTTPS icons are downloaded with a five-second timeout and cached locally as PNGs.

Each app may send **60 notifications per minute**. Up to four notifications are processed concurrently. Requests use JSON; the body limit is 2 MiB. Errors return `{ "error": "…", "code": "…" }`.

| HTTP status | Meaning |
| --- | --- |
| `400` | Invalid JSON, fields, or icon. |
| `401` | Missing, invalid, or revoked app key. |
| `403` | Foreign browser origin or an invalid Host header. |
| `413` / `415` | Request too large / incorrect Content-Type. |
| `429` | Per-app rate limit or service busy; inspect `Retry-After`. |
| `502` | Desktop delivery failed. `detail` contains the OS error; the attempt is saved in activity. |

## Dashboard

- **Overview:** a compact, centered workspace with live counts and recent notifications.
- **Apps:** create named keys, search them, check usage, and revoke access.
- **Playground:** preview and send a real notification; optionally test an app key.
- **Activity:** paginated delivery history, including failure details; search and filter by status.
- **Guide:** copyable cURL, Python, and JavaScript examples.
- **Settings:** retention preferences, software updates/restart, dashboard password, and instance details.

The dashboard uses a centered layout with breathing room on either side, pastel lavender/mint/peach cards, a little bell mascot, and a bundled rounded font. Typography defaults to an 18px base, 17px body/form text, and 14–15px secondary text, with higher-contrast labels and larger controls. Text uses relative sizing so browser font preferences and zoom are respected. Light and dark modes work offline; the font and all other web assets are embedded in the executable.

### Notification log retention

Open **Settings → Preferences** to choose:

| Mode | What is kept |
| --- | --- |
| By count | The latest N notifications (1–100,000). |
| By days | Notifications from the last N days (1–3,650). |
| A bit of both | Notifications within the age limit, capped at the chosen count. |
| Keep everything | All notification history until you clear it. |

The default remains **100 notifications**, preserving existing behavior. Preferences are saved in SQLite and survive restarts and upgrades. Limits apply immediately when saved and whenever a notification is recorded; age-based cleanup also runs every minute and on startup. Pruning and clearing history leave all-time counters and app keys intact.

Activity uses server-side pagination, search, and status filtering, so retaining more than 100 items also makes those older items browsable. These preferences control notification history in SQLite; the installer's process-output files are separate diagnostic logs.

### Software updates and dashboard restart

Open **Settings → Software updates** to see the running version and platform, **Check for updates**, and **Upgrade & restart** when a newer release is available. **Restart Localert** gives the current version a fresh start without downloading anything.

Your publisher can preconfigure the release feed in the binary. For a custom build, open **Update source**, enter the publisher's HTTPS `update-manifest.json` URL, and save it once. A missing feed or a failed check is reported explicitly; it is never presented as “up to date”. Loopback HTTP feeds are also supported for local release testing.

An upgrade downloads the matching OS/CPU executable, checks its SHA-256 checksum and size, runs its `--version` preflight, retains the old executable as `<executable>.previous`, replaces the installed file, and restarts. Invalid downloads leave the running executable unchanged. The dashboard shows progress, reconnects, and reloads the new embedded HTML/CSS/JavaScript. Passwords, app keys, preferences, and history remain in the same database. Dashboard sessions expire on restart, so sign in again when it returns.

Updates run with the current user's permissions; the per-user install locations are writable by that user. Unix restarts re-execute the process, preserving the identity tracked by a LaunchAgent or desktop launcher. Windows starts a new hidden process after closing the listener and database. Installed launchers pass a PID-file path so process tracking stays correct.

The dashboard uses a password and an HTTP-only session cookie. Passwords are salted and hashed with PBKDF2-SHA256 (600,000 iterations); plaintext passwords are never stored. Sessions expire after 24 hours, on process restart, or when credentials change. App keys can send notifications but cannot administer the service. Key secrets are stored as SHA-256 hashes, and revocation persists across restarts.

First-time setup is accepted only over a loopback connection on the computer running Localert. You can change the dashboard password in **Settings**; this invalidates other dashboard sessions while leaving app keys working.

### Upgrading from the JSON-storage version

If `state.json` exists in the data directory, Localert imports its app keys, usage counters, and recent history into SQLite exactly once. The original JSON file remains untouched as a backup. Existing app keys continue working.

The upgrade screen asks for your old `admin_token` from `state.json` and a new dashboard password. Once configured, sign in using the password. Subsequent startups use SQLite and never re-import the JSON file, so revoked keys cannot reappear.

### Forgotten password

Run the executable locally with:

```sh
./bin/localert --reset-password
```

Then open the dashboard on the same computer to choose a new password. If the service is not running, start it again. Existing app keys and notification history are kept. If you use a custom data directory, supply the same `--data-dir` when resetting.

## Options

```text
--addr       127.0.0.1:8787    HTTP listen address
--data-dir   <user config>/localert
--open                        Open the dashboard in your browser
--no-open                     Suppress automatic first-run browser opening
--reset-password              Reset dashboard credentials; keep app keys and history
--pid-file                    Optional PID file used by desktop launchers
--version                     Print version and exit
```

For notifications from another computer on your network:

```sh
./bin/localert --addr 0.0.0.0:8787
```

Use `http://THIS_PC_LAN_IP:8787/api/notify` from the other computer with its app key. The service uses HTTP; use a trusted network or a TLS-terminating proxy for remote deployment. The Host header must use a literal IP address, `localhost`, or the configured listen host. Browser calls must be same-origin; desktop/server-side software can call the API directly.

For portable runtime data:

```sh
./bin/localert --data-dir ./localert-data
```

Runtime state is automatically created separately from the executable:

```text
<data-dir>/
  localert.db        SQLite: password hash, app key hashes, counters, history
  localert.db-wal    SQLite support file, when active
  localert.db-shm    SQLite support file, when active
  icons/            cached notification images
```

Default directories: `$XDG_CONFIG_HOME/localert` or `~/.config/localert` on Linux; `%AppData%\localert` on Windows; `~/Library/Application Support/localert` on macOS. SQLite transactions keep history, usage counters, and key updates consistent. On Unix, the database is owner-readable/writable only. The dashboard’s Settings page shows the actual database path.

## Native desktop support

| OS | Backend | Notes |
| --- | --- | --- |
| Linux | Freedesktop Notifications over D-Bus | Pure Go; no `notify-send` dependency. Requires a graphical desktop’s session bus and notification service. |
| Windows 10/11 | Windows Toast API through built-in Windows PowerShell | Registers Localert’s notification identity for the current user. No third-party runtime required. |
| macOS | Built-in `osascript` Notification Center API | Uses `terminal-notifier` if installed to support custom icons. The built-in fallback uses the system icon. |

Linux, Windows, and macOS use separate binaries for their OS/CPU. Desktop delivery requires an interactive user session, including when configuring autostart. A headless container or system service generally cannot display a notification on another user’s desktop.

## Build from source

Build requirements: **Go 1.26+**, **Node.js 22.12+**, and npm. SQLite is built using a pure-Go driver, so cross-compilation does not require CGO or system SQLite libraries.

```sh
make build
./bin/localert --open
```

Without Make:

```sh
npm --prefix web ci
npm --prefix web run build
go build -trimpath -ldflags "-s -w" -o bin/localert ./cmd/localert
```

Build release binaries for Linux (amd64/arm64), Windows (amd64), and macOS (amd64/arm64):

```sh
make release
```

Release builds also create **`bin/update-manifest.json`** and **`bin/SHA256SUMS`**. Publish the manifest beside the five platform binaries on an HTTPS host. Relative asset URLs work when those files share a static directory.

For GitHub releases, generate **absolute asset URLs** and optionally preconfigure the latest-feed URL for end users:

```sh
make release VERSION=0.3.1 \
  RELEASE_BASE_URL="https://github.com/YOUR_ORG/localert/releases/download/v0.3.1/" \
  UPDATE_FEED_URL="https://github.com/YOUR_ORG/localert/releases/latest/download/update-manifest.json"
```

Upload the platform binaries, `update-manifest.json`, and `SHA256SUMS` as release assets. The feed points to the latest release, while asset URLs point to the specific release version. A saved update-source choice—including clearing it—takes precedence over publisher defaults on future starts. No release repository is hardcoded into the project.

To regenerate a manifest for already-built binaries:

```sh
go run ./cmd/localert-release --version 0.3.1 --dir bin \
  --base-url "https://your-releases.example/localert/0.3.1/" \
  --notes "A few thoughtful improvements."
```

Manifest format:

```json
{
  "version": "0.3.1",
  "notes": "Release notes shown in the dashboard.",
  "assets": {
    "linux-amd64": {
      "url": "localert-linux-amd64",
      "sha256": "<64-character SHA-256 hex digest>",
      "size": 12345678
    }
  }
}
```

Include entries for each supported platform: `linux-amd64`, `linux-arm64`, `windows-amd64`, `darwin-amd64`, and `darwin-arm64`. Only strictly newer semantic versions are offered; downloaded executables are limited to 128 MiB.

Go embeds `web/dist` at compile time, so build the frontend before compiling the executable. Every frontend change requires rebuilding the binary for distribution.

### Development

```sh
# Terminal 1 (after an initial frontend build)
go run ./cmd/localert

# Terminal 2
npm --prefix web run dev
```

The Vite development server proxies `/api` to `127.0.0.1:8787`. The production executable serves its embedded assets directly.

### Verification

```sh
go test -race ./internal/...
go vet ./...
sh -n scripts/install.sh
python -m unittest discover -s scripts -p 'test_*.py'
```

The backend tests exercise first-run setup, password hashing/change/recovery, session invalidation, database migration, key persistence/revocation, transactional rollback, retention modes, paginated history, authenticated preferences, download verification, notification errors, icon validation, and rate limiting. Unix installer checks use isolated user directories and a mocked macOS launchctl. Linux integration checks exercise the real executable, including a dashboard restart and an upgrade to a separately compiled test release while preserving user data.

Browser smoke test (build `bin/localert` first):

```sh
npm --prefix web exec -- playwright install chromium
npm --prefix web test
```

This runs a private copy of the binary with temporary SQLite data. It checks setup, app keys, delivery, process restart, password changes, retention, release checking, a real one-click binary upgrade and reconnect, and desktop/mobile layouts. A test release is compiled with Go and served on a loopback feed; the project executable is never replaced. Headless machines may report a delivery failure accurately; set `LOCALERT_E2E_NATIVE=1` to require native notification delivery.

## Project layout

```text
cmd/localert/       CLI and server lifecycle
cmd/localert-release/ Release manifest and checksum generator
internal/server/   HTTP API, admin sessions, authentication
internal/store/    Embedded SQLite, password hashing, hashed keys, JSON migration
internal/notify/   OS notification backends and icon handling
internal/updater/  Release checks, verified updates, and platform restart
web/src/           React + shadcn/ui management dashboard
web/embed.go       Embedded production assets
scripts/           Per-user Windows/Linux/macOS installers
```
