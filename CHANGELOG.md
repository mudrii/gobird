# Changelog

All notable changes to this project will be documented in this file.

Release versions use the `YY.MM.DD` format.

## [26.03.15] - 2026-03-15

Initial public open source release.

### Added
- `gobird` CLI for reading, searching, posting, following, bookmarking, and timeline browsing on X/Twitter
- Go client library under `pkg/bird`
- Browser-backed authentication from Safari, Chrome/Chromium, and Firefox
- JSON and human-readable output modes
- JSON5 configuration support
- Query ID inspection and runtime refresh behavior
- Acceptance tests and browser extraction regression coverage

### Changed
- Safari extraction now supports modern `Cookies.binarycookies` with legacy SQLite fallback
- Chrome extraction supports `CHROME_SAFE_STORAGE_PASSWORD` for macOS Keychain subprocess denial cases
- Browser-derived credentials are validated before use
- CLI error classification and acceptance coverage were hardened for release use

### Notes
- This project uses X/Twitter's unofficial private web APIs and may break when upstream behavior changes.
