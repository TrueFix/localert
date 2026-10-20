# Changelog

All notable changes are recorded here. Versions follow [Semantic Versioning](https://semver.org/).

## [0.5.9] - 2026-10-20

### Fixed
- Added the missing **Play notification sound** toggle to the Playground. The toggle updates the live JSON request body and works for both app-key notifications and dashboard test notifications.
- Browser coverage verifies sound enabled and disabled in outgoing Playground requests.

## [0.5.8] - 2026-10-20

### Added
- Optional **`sound`** boolean in the notification API: `true` requests a native notification sound on Linux, macOS, and Windows; omitted or `false` requests silent delivery. Desktop sound support, volume, and Do Not Disturb settings apply.
- Sound field documentation in the README and dashboard API guide.

## [0.5.7] - 2026-10-20

### Fixed
- **Release workflow** runs when a tag is force-updated (not only on first tag create).

### Added
- **`localert --reload`** — gracefully restart the running instance (SIGUSR1 on Linux/macOS, local HTTP fallback on Windows). Uses the installer PID file unless you pass `--pid-file`.

### Fixed
- Brief pause before spawning a restarted process so upgrades and dashboard restarts bind the listen port reliably after shutdown.

## [0.5.6] - 2026-10-20

### Added
- **`localert --change-password`** — generate a new dashboard password, print it once to the terminal, and sign out other dashboard sessions (same `--data-dir` as the running instance).

## [0.5.5] - 2026-10-20

### Fixed
- Dashboard shows a clear **password recovery** screen after `localert --reset-password`, and rechecks setup when you focus the tab or stay signed in.
- Setup after recovery works when you open **http://127.0.0.1** from the host while Localert runs in Docker (bridge gateway).

## [0.5.4] - 2026-10-20

### Fixed
- **`localert --reset-password`** runs immediately (no longer blocked by update-feed initialization) and prints the data directory path.
- After a CLI password reset, **setup works** when you open the dashboard via this computer’s LAN address (not only `127.0.0.1`), and the sign-in page **detects recovery** without a full reload.

## [0.5.3] - 2026-10-20

### Fixed
- Dashboard **Restart**, **Upgrade & restart**, and **Save & restart** (network) work reliably when Localert was installed via the launcher script (stdio redirection no longer breaks restart).
- Linux/macOS launcher passes **`--no-open`** on autostart so restarts stay quiet.

## [0.5.2] - 2026-10-20

### Changed
- **Mobile navigation** — floating rounded menu with lavender header, icons, and backdrop blur (replaces the full-height sheet that ignored dashboard styling).
- **Desktop header** — removed the redundant smile-icon user menu; theme toggle and sign out stay in the top bar.

## [0.5.1] - 2026-10-20

### Added
- **Settings → Cloud Connect** — preview of TrueFix-managed public URLs for `/api/notify` (registration and tunnel setup coming soon).
- README notes on Cloud Connect and how it will relate to local and LAN modes.

## [0.5.0] - 2026-10-19

Built to run quietly for years. Measured with 1,200 notifications plus repeated dashboard and asset requests:
private memory goes from 9.5 MB stuck after a burst to 5.6 MB and released, threads from 20 to 9, and data on disk from 4.2 MB to 1.2 MB.
Idle remains at 0% CPU and about 3 MB of private memory.

### Changed
- Memory from bursts is returned to the OS. A small Go memory limit and two scheduler threads keep the idle footprint low.
- Logs rotate at 1 MB with one backup. Installed launchers no longer append to a log forever, and crash output is capped.
- The custom-icon cache keeps at most 64 images (least recently used are removed). Built-in icons are drawn once and remote icons cached for an hour.
- The SQLite journal is capped at about 1 MB and checkpointed more often, with a smaller page cache.
- Dashboard files stream from the executable instead of being copied for every request.
- Housekeeping runs every 5 minutes instead of every minute.

### Added
- A shared limit of 120 notifications per minute across all apps, so many apps together can't flood the desktop or CPU.
- At most 64 open connections; extra clients wait instead of exhausting resources.
- A memory watchdog that restarts Localert if memory grows past 256 MB.
- Background tasks recover from unexpected errors instead of crashing.
- `--log-file` flag, plus `--purge` / `-Purge` for a complete uninstall.
- README sections on resource use, file locations, and uninstalling.
- GitHub releases list every commit since the previous tag, with the full CHANGELOG section as the release body.

## [0.4.2] - 2026-10-19

### Added
- **Settings → Network**: choose "Just this computer" (127.0.0.1) or "My whole network" (0.0.0.0) and a port, then save and restart from the dashboard.
- Copyable LAN addresses for sending notifications from other devices.
- If a saved address can't be used at startup, Localert falls back to 127.0.0.1:8787 so you're never locked out. `--addr` still overrides the setting.

### Fixed
- Playground no longer gets cut off on phones.
- Stat card menus line up with their titles, and card footers have proper padding.
- App names use the available table width instead of truncating early.
- Consistent "View all" links and no wrapping in the `--addr` notice.

## [0.4.1] - 2026-10-19

### Changed
- Improved notification icons: bell, checkmark, exclamation, X mark, and info symbols.
- Icons are now 128×128 with rounded colored backgrounds.

## [0.4.0] - 2026-10-19

### Fixed
- Newly created app keys wrap within the dialog on small screens, with a dedicated copy button.
- Linux icons are passed as image data over D-Bus, including from containers.

### Added
- Download-and-install scripts with OS/architecture detection and SHA-256 verification.
- Multi-architecture Docker images on GitHub Container Registry and a Linux desktop Compose setup.
- Container JSON diagnostics, bounded Docker logs, and image-managed upgrades.
- A tracked VERSION file and changelog entries attached to tagged releases.

### Changed
- GitHub's latest release manifest is the default update source for all builds.

## [0.3.1] - 2026-10-19

### Changed
- Larger typography, higher-contrast helper text, and more comfortable controls.
- Shorter README and automated tagged GitHub releases.

## [0.3.0] - 2026-10-19

### Added
- Configurable notification history retention by count, age, both, or forever.
- Verified binary updates and dashboard-triggered restarts.
- Centered pastel dashboard with mobile layouts and dark mode.

## [0.2.0] - 2026-10-19

### Added
- SQLite persistence, first-run password setup, and migration from JSON storage.
- Per-user installers and desktop-login autostart on Linux, Windows, and macOS.

## [0.1.0] - 2026-10-19

### Added
- Native notification HTTP API, per-app keys, and embedded web dashboard.

[0.5.9]: https://github.com/TrueFix/localert/compare/v0.5.8...v0.5.9
[0.5.8]: https://github.com/TrueFix/localert/compare/v0.5.7...v0.5.8
[0.5.7]: https://github.com/TrueFix/localert/compare/v0.5.6...v0.5.7
[0.5.6]: https://github.com/TrueFix/localert/compare/v0.5.5...v0.5.6
[0.5.5]: https://github.com/TrueFix/localert/compare/v0.5.4...v0.5.5
[0.5.4]: https://github.com/TrueFix/localert/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/TrueFix/localert/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/TrueFix/localert/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/TrueFix/localert/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/TrueFix/localert/compare/v0.4.2...v0.5.0
[0.4.2]: https://github.com/TrueFix/localert/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/TrueFix/localert/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/TrueFix/localert/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/TrueFix/localert/releases/tag/v0.3.1
[0.3.0]: https://github.com/TrueFix/localert/releases/tag/v0.3.0
[0.2.0]: https://github.com/TrueFix/localert/releases/tag/v0.2.0
[0.1.0]: https://github.com/TrueFix/localert/releases/tag/v0.1.0
