# Changelog

All notable changes to this project will be documented in this file.

Release versions use the `YY.MM.DD` format.

## [26.04.08] - 2026-04-08

### Changed
- Project baseline raised to Go 1.26 with preferred toolchain `go1.26.2`
- CI now validates the project with Go 1.26.2
- Public and internal documentation were synchronized with the current client options, rate-limiter behavior, dependency graph, and query-ID refresh behavior

### Fixed
- Query ID refresh now preserves previously cached runtime IDs when a scrape returns nothing
- Global request throttling now reserves request slots safely under concurrency and honors context cancellation
- Inline pagination no longer incurs an extra delay after `MaxPages` is already reached
- Removed dead config fields from the public config surface and aligned the docs with the supported settings
- Applied low-risk Go 1.24+ modernizations across the codebase and normalized formatter drift

## [26.03.24] - 2026-03-24

### Added
- Homebrew tap installation (`brew install mudrii/tap/gobird`)

### Fixed
- Rate limiter no longer sleeps while holding the mutex, unblocking concurrent callers
- `Retry-After` header values are capped at 60 seconds to prevent server-controlled hangs
- `NewWithTokens` public API now validates credential format, preventing header injection
- Chrome cookie extraction rejects profile hints that point outside Chrome/Chromium directories
- CI actions pinned to full commit SHAs instead of mutable version tags
- Local `--json` and `--limit` flags on subcommands no longer shadow the global persistent flags, fixing `--json --plain` mutual-exclusion bypass and negative limit validation
- `paginateCursor` returns consistent `(nil, error)` on first-page failure instead of mixed `(result, error)`
- `mediaAppend` now routes through the rate limiter instead of calling `httpClient.Do` directly
- `FilterAuthorChain` walks to the chain root before BFS, fixing dropped branches in newest-first tweet ordering
- Search pagination allows per-page query ID refresh instead of a single global refresh
- Empty query ID list now returns a descriptive error instead of `{success: false, err: nil}`
- `bestVideoVariant` uses strict greater-than for bitrate comparison, making GIF/duplicate-bitrate selection deterministic
- Scraped query IDs are validated against an alphanumeric format before caching

### Security
- `pkg/bird.NewWithTokens` validates token format to prevent HTTP header injection via malformed `ct0` values
- Chrome `profileHint` path traversal blocked — absolute paths must resolve under known Chrome directories
- GitHub Actions pinned to immutable commit SHAs to mitigate supply chain risk

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
