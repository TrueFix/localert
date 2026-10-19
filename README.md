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

- **Updates:** Open **Settings → Software updates** to check for updates, upgrade, and restart. Your password, app keys, and history stay with you.
- **Storage:** `localert.db` lives in `~/.config/localert` on Linux, `%AppData%\localert` on Windows, or `~/Library/Application Support/localert` on macOS.
- **Your network:** open **Settings → Network** and choose **My whole network** so phones, servers, and other PCs can send notifications with their app key. Use trusted networks; first-time setup stays local-only.
- **Password recovery:** run the executable with `--reset-password`, then open the dashboard locally to choose a new one.
- **Uninstall:** run `sh install.sh --uninstall`, or `install.ps1 -Uninstall` on Windows. Your saved data is kept.

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
