<div align="center">

<img src="web/public/favicon.svg" width="76" height="76" alt="Localert's smiling bell" />

# Localert

**Your apps have news. Give them a little ping. 🔔**

A tiny desktop notification server with a cozy dashboard.<br />
One binary. Your computer. No extra runtime.

[![Latest release](https://img.shields.io/github/v/release/TrueFix/localert?color=8061c3&style=flat-square)](https://github.com/TrueFix/localert/releases/latest)
[![Release builds](https://github.com/TrueFix/localert/actions/workflows/release.yml/badge.svg)](https://github.com/TrueFix/localert/actions/workflows/release.yml)
[![Platforms](https://img.shields.io/badge/Linux%20%C2%B7%20Windows%20%C2%B7%20macOS-desktop%20ready-5a9e83?style=flat-square)](https://github.com/TrueFix/localert/releases/latest)

[**Download Localert**](https://github.com/TrueFix/localert/releases/latest) · [Send a notification](#-send-a-little-hello) · [Build it yourself](#-for-the-builders)

</div>

## ✨ Small app, lovely things

- **Real desktop notifications** from a simple HTTP request.
- **A key for every app**, managed in a password-protected dashboard.
- **History your way** — keep a count, a number of days, both, or everything.
- **A fresh start in one click** — check for updates, upgrade, and restart from Settings.
- **A cozy local home** — pastel colors, readable type, dark mode, and SQLite storage.
- **Cloud Connect** *(coming soon)* — register with TrueFix cloud for a public HTTPS URL to your notification API, without opening ports on your router.

The website, font, icons, and SQLite engine are all inside the executable. Your keys and history live on your computer. 💜

## 🏡 Make yourself at home

Install straight from GitHub — the script detects your system, downloads the latest release, and verifies its checksum before installing.

**Linux & macOS**

```sh
curl -fsSL https://github.com/TrueFix/localert/releases/latest/download/install.sh | bash
```

**Windows · PowerShell**

```powershell
& ([scriptblock]::Create((Invoke-RestMethod https://github.com/TrueFix/localert/releases/latest/download/install.ps1)))
```

Or download the binary manually from the [latest release](https://github.com/TrueFix/localert/releases/latest):

| Your computer | Binary |
| --- | --- |
| Linux · Intel / AMD | `localert-linux-amd64` |
| Linux · ARM64 | `localert-linux-arm64` |
| Windows 10 / 11 | `localert-windows-amd64.exe` |
| macOS · Intel | `localert-macos-amd64` |
| macOS · Apple Silicon | `localert-macos-arm64` |

The installer sets up autostart for your user. On first launch, Localert opens **http://127.0.0.1:8787** — choose a password, create an app key, and you're home. After that, it starts quietly when you sign in.

Prefer a portable setup? Run the binary directly with `--open`. On Linux/macOS, make it executable first with `chmod +x <filename>`.

## 💌 Send a little hello

```sh
curl -X POST http://127.0.0.1:8787/api/notify \
  -H "Authorization: Bearer YOUR_APP_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "icon": "success",
    "title": "Build complete!",
    "message": "Your project is ready. Time for a little break."
  }'
```

Use `bell`, `info`, `success`, `warning`, or `error` for the icon, or send an HTTPS image URL or base64 PNG/JPEG.

Try the **Playground** for a test notification. The dashboard's **Guide** has Python and JavaScript examples and API details.

## 🌷 A few things worth knowing

- **Updates:** Open **Settings → Software updates** to check for updates, upgrade, and restart. Your password, app keys, and history stay with you. Localert never updates itself in the background.
- **Your network:** open **Settings → Network** and choose **My whole network** so phones, servers, and other PCs can send notifications with their app key. Use trusted networks; traffic is plain HTTP and first-time setup stays local-only. If a saved address can't be used, Localert falls back to `127.0.0.1:8787`. Starting with `--addr` overrides this setting.
- **Cloud Connect:** open **Settings → Cloud Connect** to preview what’s coming—a TrueFix-managed public URL for your `/api/notify` endpoint. Registration and tunnel setup are not available yet; your data still stays local until you opt in.
- **Password recovery:** run the executable with `--reset-password`, then open the dashboard locally to choose a new one. App keys and history are kept.
- **Notifications need your desktop session.** Localert must run as your signed-in user. Do Not Disturb and OS notification settings still apply.
- **Limits:** each app can send 60 notifications per minute, and all apps together 120 per minute. Extra requests get HTTP `429` with `Retry-After`.

### 🍃 Light enough to run forever

Localert is built to sit quietly in the background for years:

- **Idle cost:** 0% CPU and about **3 MB of private memory**. The rest of what your task manager shows (~14 MB) is shared, read-only program code that the OS can reclaim.
- **No background polling:** it only wakes up for requests and a few milliseconds of housekeeping every 5 minutes. The dashboard refreshes only while it's open and visible.
- **Gives memory back:** after a burst of notifications, memory returns to the OS. A built-in watchdog restarts Localert if memory ever grows past 256 MB.
- **Disk stays tiny:**
  - Logs rotate at 1 MB, with one backup kept.
  - History follows your retention setting (latest 100 by default).
  - The custom-icon cache keeps at most 64 images.
  - The database journal is capped at about 1 MB.
- **Hard to overload:** request bodies, icon sizes, concurrent deliveries (4), and open connections (64) are capped, so a misbehaving app can't take your computer down.
- **Crash-safe:** background tasks recover from unexpected errors, and SQLite keeps your database consistent even after a crash or power cut.

### 📁 Where things live

| | Linux | macOS | Windows |
| --- | --- | --- | --- |
| **Program** | `~/.local/bin/localert` | `~/.local/bin/localert` | `%LocalAppData%\Programs\Localert\localert.exe` |
| **Data** (database, icons) | `~/.config/localert` | `~/Library/Application Support/localert` | `%AppData%\localert` |
| **Logs** | `~/.local/state/localert` | `~/Library/Logs/Localert` | `%LocalAppData%\Localert\logs` |
| **Autostart** | `~/.config/autostart/localert.desktop` | `~/Library/LaunchAgents/dev.localert.agent.plist` | `Localert.lnk` in your Startup folder |

## 🧹 Uninstall

The same installer removes Localert. By default, your database and logs are **kept**, so reinstalling picks up where you left off. Add **purge** to delete them too.

**Linux & macOS**

```sh
# Remove the program and autostart (keeps your data)
curl -fsSL https://github.com/TrueFix/localert/releases/latest/download/install.sh | bash -s -- --uninstall

# Remove everything, including app keys, history, and logs
curl -fsSL https://github.com/TrueFix/localert/releases/latest/download/install.sh | bash -s -- --uninstall --purge
```

**Windows · PowerShell**

```powershell
# Remove the program and autostart (keeps your data)
& ([scriptblock]::Create((Invoke-RestMethod https://github.com/TrueFix/localert/releases/latest/download/install.ps1))) -Uninstall

# Remove everything, including app keys, history, and logs
& ([scriptblock]::Create((Invoke-RestMethod https://github.com/TrueFix/localert/releases/latest/download/install.ps1))) -Uninstall -Purge
```

Purge only deletes files Localert created, never other files in those folders. If you used a custom `--data-dir`, pass the same path when uninstalling.

**Ran the binary yourself, without the installer?** Stop it (Ctrl+C), delete the executable, and optionally delete the data folder from the table above.

**Docker:** `docker compose down`, then delete the `localert-data` folder if you no longer need it.

## 🛠️ For the builders

You'll need **Go 1.24+**, **Node.js 22+**, npm, and Make.

```sh
git clone https://github.com/TrueFix/localert.git
cd localert
make build
./bin/localert --open
```

### 📦 Release a new version

Update `VERSION`, the version in `web/package.json`, and the fallback in `cmd/localert/main.go`. Then push a version tag:

```sh
git tag v0.4.1
git push origin v0.4.1
```

[GitHub Actions](https://github.com/TrueFix/localert/actions/workflows/release.yml) builds all binaries, runs tests, and creates a GitHub release with checksums and the update manifest.

---

<div align="center">Small app. Thoughtful notifications. 💜</div>
