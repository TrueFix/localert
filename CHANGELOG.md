# Changelog

All notable changes are recorded here. Versions follow [Semantic Versioning](https://semver.org/).

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
