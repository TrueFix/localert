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

Or download the binary and installer manually from the [latest release](https://github.com/TrueFix/localert/releases/latest):

| Your computer | Binary | Installer |
| --- | --- | --- |
| Linux · Intel / AMD | `localert-linux-amd64` | `install.sh` |
| Linux · ARM64 | `localert-linux-arm64` | `install.sh` |
| Windows 10 / 11 | `localert-windows-amd64.exe` | `install.ps1` |
| macOS · Intel | `localert-macos-amd64` | `install.sh` |
| macOS · Apple Silicon | `localert-macos-arm64` | `install.sh` |

**Linux & macOS**

```sh
sh install.sh
```

**Windows**

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\install.ps1
```

The installer sets up autostart for your user. On first launch, Localert opens **http://127.0.0.1:8787** — choose a password, create an app key, and you're home. After that, it starts quietly when you sign in.

Prefer a portable setup? Run the matching executable directly with `--open`. On Linux/macOS, make it executable first with `chmod +x <filename>`. Keep Localert in your signed-in desktop session so notifications can appear.

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

Use `bell`, `info`, `success`, `warning`, or `error` for the icon, or send an HTTPS image URL or base64 PNG/JPEG. Custom icons on macOS use `terminal-notifier` when installed.

Try the **Playground** for a test notification. The dashboard's **Guide** has Python and JavaScript examples, request limits, and API details.

## 🌷 A few things worth knowing

- **Updates:** official release builds already know where to check. Open **Settings → Software updates** to upgrade; your password, app keys, and history stay with you.
- **Storage:** `localert.db` lives in `~/.config/localert` on Linux, `%AppData%\localert` on Windows, or `~/Library/Application Support/localert` on macOS. Linux respects `XDG_CONFIG_HOME`; `--data-dir` chooses a custom folder.
- **Your network:** `--addr 0.0.0.0:8787` lets apps on your LAN send notifications using this computer's IP and their app key.
- **Password recovery:** run the executable with `--reset-password`, then open the dashboard locally to choose a new one.
- **Uninstall:** run `sh install.sh --uninstall`, or `install.ps1 -Uninstall` on Windows. Your saved data is kept.

## 🛠️ For the builders

You'll need **Go 1.26+**, **Node.js 22.12+**, npm, and Make.

```sh
git clone https://github.com/TrueFix/localert.git
cd localert
make build
./bin/localert --open
```

`make check` runs backend and installer checks. For the browser checks:

```sh
npm --prefix web exec -- playwright install chromium
npm --prefix web test
```

### 📦 Ship a fresh little release

Update `VERSION`, the version in `web/package.json`/lockfile, and the fallback in `cmd/localert/main.go`. Add the matching entry to [CHANGELOG.md](CHANGELOG.md), then push a version tag from the current `main`:

```sh
git tag v0.4.0
git push origin v0.4.0
```

[GitHub Actions](https://github.com/TrueFix/localert/actions/workflows/release.yml) validates the version, builds all five binaries, runs the checks, publishes multi-architecture images to GHCR, and creates a GitHub release with installers, Compose, checksums, and the update manifest. Release notes come from the checked-in changelog. Tags such as `v0.5.0-rc.1` become prereleases and do not update the `latest` image tag.

The official update manifest is **https://github.com/TrueFix/localert/releases/latest/download/update-manifest.json**. It is built into Localert by default. Older installs with a blank/custom source can choose **Use official GitHub feed** in Settings and save it.

## 🐳 A container for your Linux desktop

Images are published as `ghcr.io/truefix/localert:latest` and version tags such as `:v0.4.0` for amd64 and arm64.

Desktop notifications need the **Linux desktop session bus**, not a headless server or Docker Desktop's VM. The included Compose file uses your user ID, shares only the session-bus socket, and binds the HTTP server to host loopback so first-run setup remains local.

Download [compose.yml](compose.yml) into a folder, then run from your signed-in Linux desktop:

```sh
export LOCALERT_UID="$(id -u)" LOCALERT_GID="$(id -g)"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
mkdir -p localert-data
docker compose up -d
```

Open **http://127.0.0.1:8787** to create your password. Stop any standalone Localert instance first if it uses the same port. Docker Desktop on Windows/macOS does not expose the native desktop bus; use the normal binary there.

```sh
docker compose logs -f                    # Diagnostics
docker compose pull && docker compose up -d  # Upgrade; database is kept
docker compose down                      # Stop; database is kept
```

Set `LOCALERT_VERSION=v0.4.0` to pin a container version. Containers use image upgrades rather than replacing their own binary. Compose retains up to three 10 MB diagnostic log files. Notification history retention is configured separately in dashboard Settings. Never mount the Docker socket.

For a local image build: `LOCALERT_VERSION=0.4.0 docker compose build`. Container errors use structured JSON on stderr; startup information goes to stdout. Native installations retain their per-user launcher logs.

---

<div align="center">Small app. Thoughtful notifications. 💜</div>
