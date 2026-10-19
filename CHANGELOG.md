# Changelog

All notable changes are recorded here. Versions follow [Semantic Versioning](https://semver.org/).

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

[0.4.0]: https://github.com/TrueFix/localert/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/TrueFix/localert/releases/tag/v0.3.1
[0.3.0]: https://github.com/TrueFix/localert/releases/tag/v0.3.0
[0.2.0]: https://github.com/TrueFix/localert/releases/tag/v0.2.0
[0.1.0]: https://github.com/TrueFix/localert/releases/tag/v0.1.0
