# gobird Configuration Reference

This document covers all aspects of gobird's configuration system: file locations, field definitions, JSON5 syntax, browser cookie extraction internals, feature flag overrides, and credential validation rules.

## Table of Contents

- [Config File Locations and Search Order](#config-file-locations-and-search-order)
- [JSON5 Syntax Guide](#json5-syntax-guide)
- [All Config Fields](#all-config-fields)
- [Environment Variable Overrides](#environment-variable-overrides)
- [Credential Validation Rules](#credential-validation-rules)
- [Browser Cookie Extraction](#browser-cookie-extraction)
- [Feature Flag Override System](#feature-flag-override-system)
- [Timeout Configuration](#timeout-configuration)
- [Per-Command Config Overrides via Flags](#per-command-config-overrides-via-flags)
- [Example Configurations](#example-configurations)

---

## Config File Locations and Search Order

gobird loads configuration from files in the following order. Later files override earlier ones. Fields not present in a later file are not cleared — they retain values from earlier files.

### Default search order

1. `~/.config/bird/config.json5` — global user configuration
2. `./.birdrc.json5` — project-local config in the current working directory

Both files are loaded if they exist. The project-local file takes precedence on any fields it defines.

### Explicit config file

An explicit path bypasses the default search order entirely. Only that single file is loaded.

**Via flag:**
```sh
bird --config /path/to/myconfig.json5 home
```

**Via environment variable:**
```sh
BIRD_CONFIG=/path/to/myconfig.json5 bird home
```

When `--config` or `BIRD_CONFIG` is set, `~/.config/bird/config.json5` and `./.birdrc.json5` are both ignored.

### Loading algorithm (pseudo-code)

```
if --config flag or BIRD_CONFIG is set:
    load that single file
else:
    load ~/.config/bird/config.json5   (if it exists)
    merge ./.birdrc.json5              (if it exists, overrides global)

apply default values for any zero-value fields
apply environment variable overrides
```

---

## JSON5 Syntax Guide

gobird config files use JSON5 format via the [hujson](https://github.com/tailscale/hujson) parser, which is a strict superset of JSON. It is normalized to standard JSON before parsing, so all standard JSON parsers can read the result.

### Supported JSON5 features

**Single-line comments (`//`):**
```json5
{
  // This is a comment
  "defaultBrowser": "safari", // inline comment
}
```

**Block comments (`/* ... */`):**
```json5
{
  /*
   * Multi-line
   * comment block
   */
  "timeoutMs": 30000,
}
```

**Trailing commas:**
```json5
{
  "authToken": "abc",
  "ct0": "def", // trailing comma on last item is OK
}
```

**What is NOT supported:**
- Unquoted keys (standard JSON requires quoted keys)
- Single-quoted strings
- Hexadecimal numbers
- Multiline strings

Since hujson is a strict superset of JSON, valid JSON is always valid hujson.

---

## All Config Fields

Complete field reference with types, defaults, and corresponding environment variables.

| Field | JSON key | Go type | Default | Env var override | Description |
|-------|----------|---------|---------|-----------------|-------------|
| AuthToken | `authToken` | `string` | `""` | `AUTH_TOKEN`, `TWITTER_AUTH_TOKEN` | Twitter `auth_token` cookie value |
| Ct0 | `ct0` | `string` | `""` | `CT0`, `TWITTER_CT0` | Twitter `ct0` cookie value |
| DefaultBrowser | `defaultBrowser` | `string` | `""` | — | Browser for cookie extraction: `safari`, `chrome`, or `firefox` |
| ChromeProfile | `chromeProfile` | `string` | `""` | — | Chrome profile name (e.g., `"Default"`, `"Profile 1"`) |
| ChromeProfileDir | `chromeProfileDir` | `string` | `""` | — | Absolute path to Chrome profile directory or Cookies DB |
| FirefoxProfile | `firefoxProfile` | `string` | `""` | — | Firefox profile name or substring for matching |
| CookieSource | `cookieSource` | `string` or `[]string` | `[]` | — | Ordered list of browsers to try for cookie extraction |
| CookieTimeoutMs | `cookieTimeoutMs` | `int` | `0` | `BIRD_COOKIE_TIMEOUT_MS` | Cookie extraction timeout in milliseconds (0 = unlimited) |
| TimeoutMs | `timeoutMs` | `int` | `0` (→ 30000) | `BIRD_TIMEOUT_MS` | HTTP request timeout in milliseconds (0 = 30000 default) |
| QuoteDepth | `quoteDepth` | `int` | `1` | `BIRD_QUOTE_DEPTH` | Quoted tweet expansion depth |
| QueryIDCachePath | `queryIdCachePath` | `string` | `""` | — | Override path for the on-disk query ID cache |
| FeatureOverridesPath | `featureOverridesPath` | `string` | `""` | — | Path to a feature flag overrides JSON file |

### Field details

#### `authToken`

The `auth_token` cookie from an active Twitter/X browser session. Must be exactly 40 lowercase hexadecimal characters.

```json5
{
  "authToken": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
}
```

#### `ct0`

The `ct0` CSRF token cookie. Must be 32–160 alphanumeric characters.

```json5
{
  "ct0": "abc123def456abc123def456abc123def456abc123abc123"
}
```

#### `defaultBrowser`

Selects which browser to extract cookies from when no explicit tokens are provided.

```json5
{
  "defaultBrowser": "safari"
}
```

Valid values: `"safari"`, `"chrome"`, `"firefox"`. Case-insensitive.

#### `chromeProfile`

Chrome profile name. Used when `defaultBrowser` or `--browser` is `"chrome"`.

```json5
{
  "chromeProfile": "Profile 1"
}
```

The actual cookie DB path resolved will be:
`~/Library/Application Support/Google/Chrome/Profile 1/Cookies`

#### `chromeProfileDir`

A more specific override for Chrome. Accepts:
- Absolute path to a profile directory → appends `/Cookies`
- Absolute path ending in `.sqlite` or `Cookies` → used directly

Takes precedence over `chromeProfile` when both are set.

#### `firefoxProfile`

Firefox profile name or substring. Used to filter which Firefox profile to read cookies from. If multiple profiles match the substring, all are tried and their cookies merged.

```json5
{
  "firefoxProfile": "default-release"
}
```

#### `cookieSource`

Specifies the order in which browsers are tried for cookie extraction. Accepts a single string or an array.

```json5
// Single browser:
{
  "cookieSource": "chrome"
}

// Multiple browsers in priority order:
{
  "cookieSource": ["safari", "chrome", "firefox"]
}
```

When `cookieSource` is set, `defaultBrowser` is still used as a fallback if `cookieSource` is empty.

Priority resolution when multiple settings exist:
1. `--cookie-source` flag(s) on the command line
2. `cookieSource` in config file
3. `--browser` flag
4. `defaultBrowser` in config file
5. Default order: `safari` → `chrome` → `firefox`

#### `cookieTimeoutMs`

Timeout for browser cookie extraction. Use this to prevent hanging when Chrome's Keychain access dialog appears.

```json5
{
  "cookieTimeoutMs": 3000
}
```

#### `timeoutMs`

HTTP request timeout. The internal default is 30,000 ms (30 seconds). Set a higher value on slow connections or a lower value to fail fast.

```json5
{
  "timeoutMs": 60000
}
```

#### `quoteDepth`

How many levels deep to recursively fetch quoted tweets. The default after applying `applyDefaults` is `1`.

| Value | Effect |
|-------|--------|
| `0` | No quoted tweet expansion |
| `1` | Expand one level of quoted tweets (default) |
| `2+` | Expand recursively to that depth |

```json5
{
  "quoteDepth": 0
}
```

#### `queryIdCachePath`

The internal client caches runtime-scraped GraphQL query IDs to a file. This field overrides where that file is stored. Rarely needed.

#### `featureOverridesPath`

Path to a JSON file containing feature flag overrides. See the [Feature Flag Override System](#feature-flag-override-system) section.

---

## Environment Variable Overrides

Environment variables are applied after config files are loaded and always win over file values.

| Variable | Type | Applies to | Notes |
|----------|------|-----------|-------|
| `AUTH_TOKEN` | string | `authToken` | Takes precedence over `TWITTER_AUTH_TOKEN` |
| `TWITTER_AUTH_TOKEN` | string | `authToken` | Alias; only used if `AUTH_TOKEN` is not set |
| `CT0` | string | `ct0` | Takes precedence over `TWITTER_CT0` |
| `TWITTER_CT0` | string | `ct0` | Alias; only used if `CT0` is not set |
| `BIRD_CONFIG` | string | (config path) | Selects a specific config file, bypassing default search |
| `BIRD_TIMEOUT_MS` | int string | `timeoutMs` | Parsed with `strconv.Atoi`; invalid values are ignored |
| `BIRD_COOKIE_TIMEOUT_MS` | int string | `cookieTimeoutMs` | Parsed with `strconv.Atoi` |
| `BIRD_QUOTE_DEPTH` | int string | `quoteDepth` | Parsed with `strconv.Atoi` |
| `BIRD_FEATURES_JSON` | JSON string | (feature overrides) | Inline feature override JSON (see Feature Flags section) |
| `BIRD_FEATURES_PATH` | string | (feature overrides) | Path to feature overrides JSON file |

---

## Credential Validation Rules

Both `auth_token` and `ct0` are validated before being sent to the API. Invalid values are rejected with an error.

### auth_token format

- Exactly 40 characters
- Only lowercase hexadecimal characters: `[0-9a-f]`
- Regex: `^[0-9a-f]{40}$`

**Valid example:** `a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2`

**Invalid examples:**
- `A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2` — uppercase letters
- `a1b2c3d4e5f6` — too short (12 chars)
- `a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2xx` — too long

### ct0 format

- Between 32 and 160 characters
- Only alphanumeric characters: `[0-9a-zA-Z]`
- Regex: `^[0-9a-zA-Z]{32,160}$`

**Valid example:** `abc123ABC456abc123ABC456abc123AB`

**Invalid examples:**
- `short` — fewer than 32 characters
- `abc-123` — contains a hyphen (not alphanumeric)

### Where validation runs

Validation runs in `auth.ResolveCredentials` for all three credential tiers:
1. When tokens come from CLI flags (`--auth-token` + `--ct0`)
2. When tokens come from environment variables (`AUTH_TOKEN` + `CT0`)
3. After browser cookie extraction

The error message includes the tier that failed:
```
invalid credentials from flags: auth_token has invalid format (expected 40 hex characters)
invalid credentials from environment: ct0 has invalid format (expected 32–160 alphanumeric characters)
credential resolution failed: no valid credentials found (browser: ...)
```

---

## Browser Cookie Extraction

gobird reads cookies directly from the browser's on-disk SQLite cookie store. No browser process needs to be running. The cookies queried are `auth_token` and `ct0` for domains matching `%.x.com` or `%.twitter.com`.

### How credentials are prioritized across domains

When multiple domain entries exist (e.g., `.x.com` and `.twitter.com`), the `x.com` domain value is preferred.

### Safari

**Platform:** macOS only

**Cookie store paths (tried in order):**
1. `~/Library/Containers/com.apple.Safari/Data/Library/Cookies/Cookies.db` (sandboxed Safari)
2. `~/Library/Cookies/Cookies.db` (non-sandboxed fallback)

**Implementation:** Opens the SQLite DB in read-only immutable mode. Queries the `cookies` table for `name IN ('auth_token','ct0')`.

**Notes:** No decryption required — Safari stores cookie values as plaintext in the DB.

```sh
# Explicitly select Safari:
bird home --browser safari

# Or in config:
# { "defaultBrowser": "safari" }
```

### Chrome / Chromium

**Platform:** macOS only

**Cookie store paths (tried in order):**
1. If `--chrome-profile-dir` is an absolute path ending in `.sqlite` or `Cookies` → used directly
2. If `--chrome-profile-dir` is an absolute directory → appends `/Cookies`
3. If `--chrome-profile` is a profile name → `~/Library/Application Support/Google/Chrome/<profile>/Cookies`
4. Same path under Chromium
5. `~/Library/Application Support/Google/Chrome/Default/Cookies` (fallback)
6. `~/Library/Application Support/Google/Chrome/Profile 1/Cookies` (fallback)
7. `~/Library/Application Support/Chromium/Default/Cookies` (fallback)

**Encryption:** Chrome encrypts cookie values with AES-128-CBC. The key is derived using PBKDF2-SHA1 (1003 iterations, salt `saltysalt`) from a password retrieved from the macOS Keychain under the service `Chrome Safe Storage`, account `Chrome`. Cookie values are prefixed with `v10` or `v11`.

**Keychain access:** gobird calls `security find-generic-password -w -a Chrome -s "Chrome Safe Storage"`. This may trigger a macOS authorization dialog on first use. Use `--cookie-timeout` to set a deadline if Keychain access hangs.

```sh
bird home --browser chrome
bird home --browser chrome --chrome-profile "Work"
bird home --chrome-profile-dir "/path/to/profile"
```

### Firefox

**Platform:** macOS (reads from `~/Library/Application Support/Firefox/Profiles/`)

**Cookie store:** `<profile-dir>/cookies.sqlite`

**Implementation:** Lists all profile directories, optionally filtering by `--firefox-profile` as a substring match. Opens each matching `cookies.sqlite` in read-only immutable mode. Queries the `moz_cookies` table for `name IN ('auth_token','ct0')`.

**No decryption required:** Firefox stores cookie values as plaintext.

```sh
bird home --browser firefox
bird home --browser firefox --firefox-profile "default-release"
```

### Extraction timeout

Use `--cookie-timeout` (milliseconds) or `cookieTimeoutMs` in config to abort cookie extraction if it takes too long. This is especially useful when Chrome's Keychain authorization dialog may appear.

```sh
bird home --browser chrome --cookie-timeout 5000
```

```json5
{
  "defaultBrowser": "chrome",
  "cookieTimeoutMs": 5000
}
```

---

## Feature Flag Override System

gobird sends GraphQL feature flags with every API request. These flags control which API features are enabled. The values are hardcoded per operation in the binary, but can be overridden at runtime for debugging or compatibility adjustments.

### How it works

Feature overrides are loaded once per process (via `sync.Once`) from one of two sources:

1. `BIRD_FEATURES_JSON` environment variable — an inline JSON string
2. `BIRD_FEATURES_PATH` environment variable — path to a JSON file

If both are set, `BIRD_FEATURES_JSON` takes precedence.

### Override file format

```json
{
  "global": {
    "some_feature_flag": true,
    "another_flag": false
  },
  "sets": {
    "search": {
      "rweb_video_timestamps_enabled": false
    },
    "homeTimeline": {
      "vibe_api_enabled": false
    }
  }
}
```

**`global`:** Overrides applied to every feature set used by every operation.

**`sets`:** Per-operation overrides. Applied after global overrides. Valid set names:

| Set name | Operations affected |
|----------|-------------------|
| `article` | Base feature set (most operations derive from this) |
| `tweetDetail` | `GetTweet`, `GetReplies`, `GetThread` |
| `search` | `Search`, `GetAllSearchResults`, `mentions` |
| `tweetCreate` | `Tweet`, `Reply`, `TweetWithMedia`, `ReplyWithMedia` |
| `timeline` | Base for bookmark/home/like timelines |
| `bookmarks` | `GetBookmarks`, `GetBookmarkFolderTimeline` |
| `likes` | `GetLikes` |
| `homeTimeline` | `GetHomeTimeline`, `GetHomeLatestTimeline` |
| `lists` | `GetOwnedLists`, `GetListMemberships`, `GetListTimeline` |
| `userTweets` | `GetUserTweets` |
| `following` | `GetFollowing`, `GetFollowers` |
| `explore` | `GetNews` (GenericTimelineById) |

### Merge order

For a given operation, features are assembled in this order (later entries win):

1. Hardcoded base map for the operation
2. Global overrides from `global` block
3. Set-specific overrides from `sets[setName]`

### Examples

**Disable a feature globally:**
```sh
BIRD_FEATURES_JSON='{"global":{"vibe_api_enabled":false}}' bird home
```

**Override a feature for search only:**
```sh
BIRD_FEATURES_JSON='{"sets":{"search":{"rweb_video_timestamps_enabled":false}}}' bird search golang
```

**Using a file:**
```sh
BIRD_FEATURES_PATH=/path/to/overrides.json bird home
```

```json5
// overrides.json
{
  "global": {
    "premium_content_api_read_enabled": false
  },
  "sets": {
    "homeTimeline": {
      "responsive_web_text_conversations_enabled": true
    }
  }
}
```

---

## Timeout Configuration

gobird has two separate timeouts:

### HTTP request timeout

Controls how long a single HTTP request can take before being aborted.

- **Default:** 30,000 ms (30 seconds)
- **Config field:** `timeoutMs`
- **Env var:** `BIRD_TIMEOUT_MS`
- **Flag:** `--timeout <ms>`

Flag takes precedence over config, which takes precedence over env var.

### Cookie extraction timeout

Controls how long browser cookie extraction can run before being aborted. Particularly useful for Chrome when Keychain access prompts appear.

- **Default:** 0 (no timeout — blocks indefinitely)
- **Config field:** `cookieTimeoutMs`
- **Env var:** `BIRD_COOKIE_TIMEOUT_MS`
- **Flag:** `--cookie-timeout <ms>`

When the timeout fires, an error is returned:
```
cookie extraction timed out after 5000ms
```

---

## Per-Command Config Overrides via Flags

Every config value can be overridden at invocation time using flags. Flags always win over config file values and environment variables.

### Override precedence (highest to lowest)

```
CLI flag > environment variable > config file value > built-in default
```

### Common per-command overrides

```sh
# Use a different config file for one command:
bird --config ~/.config/bird/work.json5 home

# Override credentials for one command:
bird tweet "hello" --auth-token <token> --ct0 <ct0>

# Override browser for one command:
bird home --browser chrome --chrome-profile Work

# Override timeout for one command:
bird bookmarks --timeout 120000

# Override cookie timeout for one command:
bird check --browser chrome --cookie-timeout 3000

# Override quote depth for one command:
bird read 1234567890 --quote-depth 3

# Disable quoted tweet expansion for one command:
bird thread 1234567890 --quote-depth 0

# Override the number of pages:
bird following --max-pages 5
```

### Flag-to-config field mapping

| Flag | Config field | Default |
|------|-------------|---------|
| `--auth-token` | `authToken` | `""` |
| `--ct0` | `ct0` | `""` |
| `--browser` | `defaultBrowser` | `""` |
| `--config` | (config path) | search order |
| `--cookie-source` | `cookieSource` | `[]` |
| `--chrome-profile` | `chromeProfile` | `""` |
| `--chrome-profile-dir` | `chromeProfileDir` | `""` |
| `--firefox-profile` | `firefoxProfile` | `""` |
| `--cookie-timeout` | `cookieTimeoutMs` | `0` |
| `--timeout` | `timeoutMs` | `0` |
| `--quote-depth` | `quoteDepth` | `-1` (use config/default) |
| `--count` / `-n` | (fetch limit) | `0` |
| `--max-pages` | (page cap) | `0` |

**Note on `--quote-depth`:** The flag default is `-1`, which signals "use the value from config or the built-in default of 1." Setting `--quote-depth 0` explicitly disables quoted tweet expansion for that invocation.

---

## Example Configurations

### Minimal Safari config

```json5
// ~/.config/bird/config.json5
{
  "defaultBrowser": "safari",
}
```

### Explicit tokens with conservative timeouts

```json5
{
  "authToken": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "ct0": "abc123def456abc123def456abc123def456abc123abc123",
  "timeoutMs": 45000,
  "quoteDepth": 0,
}
```

### Chrome with profile and timeout

```json5
{
  "defaultBrowser": "chrome",
  "chromeProfile": "Work",
  "cookieTimeoutMs": 5000,
  "timeoutMs": 30000,
  "quoteDepth": 1,
}
```

### Multi-browser fallback order

```json5
{
  // Try Safari first, then Chrome, then Firefox
  "cookieSource": ["safari", "chrome", "firefox"],
  "chromeProfile": "Default",
  "cookieTimeoutMs": 3000,
}
```

### Project-local override (`.birdrc.json5`)

Place this in a project directory to use different credentials than your global config:

```json5
// ./.birdrc.json5
{
  // Use a specific Chrome profile for this project's Twitter account
  "defaultBrowser": "chrome",
  "chromeProfile": "ProjectAccount",
  "quoteDepth": 0,
}
```

### CI / automation config

```json5
// Credentials come from environment: AUTH_TOKEN + CT0
// This file just sets timeouts and disables interactive features.
{
  "timeoutMs": 120000,
  "cookieTimeoutMs": 1,  // fail fast if someone accidentally sets a browser
  "quoteDepth": 0,
}
```

In CI, set credentials via environment variables:

```sh
export AUTH_TOKEN=...
export CT0=...
bird check
bird search "golang" --json > results.json
```
