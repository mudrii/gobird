# gobird CLI Reference

`bird` is the command-line interface for gobird, a Twitter/X client written in Go. It supports reading tweets, posting, searching, managing bookmarks, following/unfollowing users, and more — all authenticated via browser cookies or explicit token values.

## Table of Contents

- [Installation](#installation)
- [Global Flags](#global-flags)
- [Authentication](#authentication)
- [Config File](#config-file)
- [Output Formats](#output-formats)
- [Commands](#commands)
  - [version](#version)
  - [read](#read)
  - [replies](#replies)
  - [thread](#thread)
  - [tweet](#tweet)
  - [reply](#reply)
  - [search](#search)
  - [mentions](#mentions)
  - [home](#home)
  - [bookmarks](#bookmarks)
  - [unbookmark](#unbookmark)
  - [following](#following)
  - [followers](#followers)
  - [likes](#likes)
  - [whoami](#whoami)
  - [about](#about)
  - [follow](#follow)
  - [unfollow](#unfollow)
  - [user-tweets](#user-tweets)
  - [lists](#lists)
  - [list-timeline](#list-timeline)
  - [news](#news)
  - [trending](#trending)
  - [check](#check)
  - [query-ids](#query-ids)
- [Environment Variables](#environment-variables)
- [Exit Codes](#exit-codes)

---

## Installation

### go install

```sh
go install github.com/mudrii/gobird/cmd/gobird@latest
```

The installed binary is named `gobird`, but the CLI command is `bird` when invoked directly from a built binary.

### Build from source

```sh
git clone https://github.com/mudrii/gobird
cd gobird
make build
# binary is placed at bin/gobird
```

The Makefile injects version and git SHA at link time:

```sh
make build   # produces bin/gobird with version + SHA
make test    # run all tests
make ci      # vet + test + test-race + build
```

---

## Global Flags

These flags are accepted by every subcommand. They are declared as persistent flags on the root command.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--auth-token` | | string | `""` | Twitter `auth_token` cookie value |
| `--ct0` | | string | `""` | Twitter `ct0` cookie value |
| `--browser` | | string | `""` | Browser to extract cookies from: `safari`, `chrome`, `firefox` |
| `--config` | | string | `""` | Explicit config file path |
| `--json` | | bool | `false` | Output as JSON |
| `--json-full` | | bool | `false` | Output as JSON including raw API response fields |
| `--plain` | | bool | `false` | Plain text output (no ANSI colors, no emoji) |
| `--no-color` | | bool | `false` | Disable ANSI color escape codes |
| `--no-emoji` | | bool | `false` | Disable emoji characters in output |
| `--count` / `--limit` | `-n` | int | `0` | Maximum total items to fetch (0 = no limit) |
| `--max-pages` | | int | `0` | Maximum number of API pages to fetch (0 = no limit) |
| `--cookie-source` | | string array | `nil` | Browser cookie source(s) in priority order |
| `--chrome-profile` | | string | `""` | Chrome profile name |
| `--chrome-profile-dir` | | string | `""` | Chrome/Chromium profile directory or cookie DB path |
| `--firefox-profile` | | string | `""` | Firefox profile name |
| `--cookie-timeout` | | int | `0` | Cookie extraction timeout in milliseconds |
| `--timeout` | | int | `0` | HTTP request timeout in milliseconds (default: 30000) |
| `--quote-depth` | | int | `-1` | Quoted tweet expansion depth (-1 uses config/default of 1) |
| `--media` | | string array | `nil` | Media file path(s) to attach (tweet/reply only) |
| `--alt` | | string array | `nil` | Alt text for each corresponding `--media` file |
| `--version` | | bool | `false` | Print version and git SHA |

**Mutual exclusivity:** `--json`, `--json-full`, and `--plain` cannot be combined. Passing more than one returns exit code 2.

**`--count` vs `--limit`:** Both flags bind to the same variable. `--count` (with shorthand `-n`) is the canonical public form; `--limit` is hidden but accepted for backward compatibility.

---

## Authentication

The CLI resolves credentials in the following priority order:

### 1. Explicit flags

Both `--auth-token` and `--ct0` must be provided together. If only one is given, this tier is skipped.

```sh
bird read 1234567890 \
  --auth-token a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 \
  --ct0 abc123def456abc123def456abc123def456abc123
```

### 2. Environment variables

Set both `AUTH_TOKEN` and `CT0` (preferred names), or their aliases `TWITTER_AUTH_TOKEN` and `TWITTER_CT0`.

```sh
export AUTH_TOKEN=a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
export CT0=abc123def456abc123def456abc123def456abc123
bird home
```

Alias resolution order:
- `AUTH_TOKEN` is checked first, then `TWITTER_AUTH_TOKEN`
- `CT0` is checked first, then `TWITTER_CT0`

### 3. Browser cookie extraction

When no explicit tokens are provided, cookies are read directly from an installed browser's SQLite cookie store.

```sh
# Use Safari (macOS only)
bird home --browser safari

# Use Chrome with a specific profile
bird home --browser chrome --chrome-profile "Profile 1"

# Use Firefox with a profile name hint
bird home --browser firefox --firefox-profile myprofile

# Try browsers in a custom order
bird home --cookie-source safari --cookie-source chrome

# Set a timeout to avoid hanging on slow keychain access
bird home --browser chrome --cookie-timeout 5000
```

**Safari:** Reads from `~/Library/Containers/com.apple.Safari/Data/Library/Cookies/Cookies.db`, falling back to `~/Library/Cookies/Cookies.db`.

**Chrome:** Reads from `~/Library/Application Support/Google/Chrome/<profile>/Cookies`. Cookie values are AES-128-CBC decrypted using a key retrieved from the macOS Keychain (`Chrome Safe Storage`). Also searches Chromium paths.

**Firefox:** Reads from `~/Library/Application Support/Firefox/Profiles/<profile>/cookies.sqlite`. No decryption required; Firefox stores cookies as plaintext in SQLite.

When `--browser` or `--cookie-source` are not set, the default extraction order is: safari → chrome → firefox.

---

## Config File

The config file uses [JSON5](https://json5.org/) syntax (comments and trailing commas are allowed). It is parsed by [hujson](https://github.com/tailscale/hujson).

### Search order

Config files are loaded and merged in this order (later values override earlier ones):

1. `~/.config/bird/config.json5` — global user config
2. `./.birdrc.json5` — project-local config (current working directory)

A single explicit path bypasses the search order entirely:

```sh
bird --config /path/to/myconfig.json5 home
```

The environment variable `BIRD_CONFIG` also selects an explicit path:

```sh
BIRD_CONFIG=/path/to/myconfig.json5 bird home
```

### All config fields

```json5
{
  // Explicit credentials — use these or browser extraction, not both.
  "authToken": "",          // string: 40 hex characters
  "ct0": "",                // string: 32-160 alphanumeric characters

  // Browser selection for cookie extraction.
  "defaultBrowser": "",     // string: "safari" | "chrome" | "firefox"

  // Chrome-specific profile selection.
  "chromeProfile": "",      // string: Chrome profile name (e.g. "Default", "Profile 1")
  "chromeProfileDir": "",   // string: absolute path to profile dir or Cookies DB file

  // Firefox-specific profile selection.
  "firefoxProfile": "",     // string: Firefox profile name or substring hint

  // Cookie source order for browser extraction.
  // Accepts a single string or an array of strings.
  "cookieSource": [],       // string | string[]: ["safari", "chrome", "firefox"]

  // Timeouts.
  "cookieTimeoutMs": 0,     // int: browser cookie extraction timeout in ms (0 = unlimited)
  "timeoutMs": 0,           // int: HTTP request timeout in ms (0 = 30000 default)

  // Tweet display.
  "quoteDepth": 1,          // int: quoted tweet expansion depth (default: 1)

  // Advanced: override where query IDs are cached on disk.
  "queryIdCachePath": "",   // string: path to query ID cache file

  // Advanced: override feature flag values (see Feature Flag Overrides section).
  "featureOverridesPath": "", // string: path to feature overrides JSON file
}
```

### Example config

```json5
{
  // Use Safari cookies by default on macOS.
  "defaultBrowser": "safari",

  // Increase timeout for slow networks.
  "timeoutMs": 60000,

  // Don't expand quoted tweets.
  "quoteDepth": 0,

  // Cookie extraction should give up quickly.
  "cookieTimeoutMs": 3000,
}
```

### env var overrides

Environment variables override config file values when set. They are applied after the file is loaded.

| Environment Variable | Config Field | Notes |
|---------------------|--------------|-------|
| `AUTH_TOKEN` | `authToken` | Takes precedence over `TWITTER_AUTH_TOKEN` |
| `TWITTER_AUTH_TOKEN` | `authToken` | Alias for `AUTH_TOKEN` |
| `CT0` | `ct0` | Takes precedence over `TWITTER_CT0` |
| `TWITTER_CT0` | `ct0` | Alias for `CT0` |
| `BIRD_TIMEOUT_MS` | `timeoutMs` | Integer string |
| `BIRD_COOKIE_TIMEOUT_MS` | `cookieTimeoutMs` | Integer string |
| `BIRD_QUOTE_DEPTH` | `quoteDepth` | Integer string |
| `BIRD_CONFIG` | (file path) | Selects explicit config file |
| `BIRD_FEATURES_JSON` | (feature overrides) | Inline JSON string for feature flag overrides |
| `BIRD_FEATURES_PATH` | (feature overrides) | Path to feature overrides JSON file |

---

## Output Formats

By default, output is formatted text with ANSI color and emoji where appropriate. The format is controlled by mutually exclusive flags.

| Flag | Behavior |
|------|----------|
| (default) | Formatted text with ANSI color and emoji |
| `--plain` | Formatted text, no color, no emoji |
| `--no-color` | Color disabled, emoji still active |
| `--no-emoji` | Emoji disabled, color still active |
| `--json` | JSON array of normalized objects, 2-space indented |
| `--json-full` | JSON array including `_raw` field with original API response |

**Tweet text format (default):**
```
🐦 @handle: tweet text (replies: N, likes: N, rts: N)
```

**User format (default):**
```
👤 @handle (Name) - followers: 12.3K, following: 456 ✓
```

**List format (default):**
```
📋 List Name [list-id] (members: 42, owner: @handle)
```

**News/trending format (default):**
```
📰 Headline (Category) [https://...] 🤖
```

---

## Commands

### version

Print the build version and git SHA.

**Syntax:**
```
bird version
bird --version
```

**Examples:**
```sh
bird version
# v1.2.3 (abc1234)

bird --version
# dev (unknown)
```

---

### read

Read a single tweet by ID or URL.

**Syntax:**
```
bird read <tweet-id-or-url>
bird <tweet-id-or-url>
```

The root command also accepts a single bare argument and delegates to `read`.

**Description:** Fetches and displays a tweet. Respects `--quote-depth` for expanding quoted tweets.

**Examples:**
```sh
bird read 1234567890123456789
bird read https://x.com/user/status/1234567890123456789
bird 1234567890123456789 --json
bird read 1234567890123456789 --json-full --quote-depth 2
```

**Exit codes:** 0 success, 1 API/auth error, 2 invalid argument.

---

### replies

Fetch replies to a tweet.

**Syntax:**
```
bird replies <tweet-id-or-url>
```

**Description:** Returns all reply tweets to the specified tweet. Paginated; respects `--count` and `--max-pages`.

**Flags:** All global flags apply. Output is a list of tweets separated by `---`.

**Examples:**
```sh
bird replies 1234567890123456789
bird replies https://x.com/user/status/1234567890 --count 20 --json
```

---

### thread

Fetch a tweet thread.

**Syntax:**
```
bird thread <tweet-id-or-url> [--filter author|full]
```

**Description:** Fetches all tweets in a thread starting from the given tweet. Two filter modes control which replies to include:

| `--filter` value | Behavior |
|-----------------|----------|
| `author` (default) | Only tweets by the original author (author chain) |
| `full` | All tweets in the conversation chain |

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--filter` | string | `""` | Thread filter mode: `author` or `full` |

**Examples:**
```sh
bird thread 1234567890123456789
bird thread 1234567890123456789 --filter full
bird thread https://x.com/user/status/1234567890 --json --count 50
```

---

### tweet

Post a new tweet.

**Syntax:**
```
bird tweet <text>
```

**Description:** Posts a new tweet with the given text. Returns the new tweet ID on success (as plain text, or `{"id":"..."}` with `--json`).

Media attachments: use `--media` to attach up to 4 files. Each `--media` may have a corresponding `--alt` for alt text. Supported MIME types: `image/*`, `video/*`, `audio/*`. Maximum file size: 512 MiB.

**Examples:**
```sh
bird tweet "Hello, world!"
bird tweet "Check this out" --media /path/to/image.png --alt "A screenshot"
bird tweet "Photo dump" \
  --media /path/to/a.jpg --alt "First photo" \
  --media /path/to/b.jpg --alt "Second photo"
bird tweet "Silent post" --json
```

**Output:**
- Default: `1234567890123456789`
- `--json`: `{"id": "1234567890123456789"}`

---

### reply

Reply to a tweet.

**Syntax:**
```
bird reply <tweet-id-or-url> <text>
```

**Description:** Posts a reply to the specified tweet. Accepts the same `--media` and `--alt` flags as `tweet`. Returns the new tweet ID.

**Examples:**
```sh
bird reply 1234567890123456789 "Great point!"
bird reply https://x.com/user/status/1234567890 "Agreed" --media /path/to/img.png
bird reply 1234567890 "This" --json
```

---

### search

Search tweets.

**Syntax:**
```
bird search <query>
```

**Description:** Searches for tweets matching the query. Supports Twitter's full search syntax (operators like `from:`, `to:`, `#hashtag`, `"exact phrase"`, etc.). Paginated.

**Examples:**
```sh
bird search "golang"
bird search "from:user filter:links" --count 50
bird search "#gobird" --json
bird search "site:example.com" --max-pages 3 --plain
```

---

### mentions

Fetch mentions of a user.

**Syntax:**
```
bird mentions [--user <handle>]
```

**Description:** Returns tweets that mention the specified handle. Defaults to the currently authenticated user when `--user` is not given.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--user` | string | `""` | Twitter handle to fetch mentions for (without `@`) |

**Examples:**
```sh
# Mentions of the authenticated user
bird mentions

# Mentions of a specific user
bird mentions --user someuser
bird mentions --user someuser --count 25 --json
```

---

### home

Fetch the home timeline.

**Syntax:**
```
bird home [--latest]
```

**Description:** Returns tweets from the authenticated user's home timeline. By default uses the algorithmic feed. With `--latest`, uses the chronological "Following" feed.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `0` | Maximum number of tweets to fetch |
| `--json` | bool | `false` | Output as JSON |
| `--latest` | bool | `false` | Use latest (chronological) timeline instead of algorithmic feed |

Note: the command-local `--json` and `--limit` flags shadow the global flags with the same names on this command.

**Examples:**
```sh
bird home
bird home --latest
bird home --limit 30 --json
bird home --latest --plain --no-emoji
```

---

### bookmarks

Fetch bookmarks.

**Syntax:**
```
bird bookmarks [--folder <folder-id>]
```

**Description:** Returns bookmarked tweets for the authenticated user. Optionally fetch from a specific bookmark folder.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `0` | Maximum number of tweets to fetch |
| `--json` | bool | `false` | Output as JSON |
| `--folder` | string | `""` | Bookmark folder ID |

**Examples:**
```sh
bird bookmarks
bird bookmarks --limit 20
bird bookmarks --folder VjlhbGxhYnVzAAA= --json
bird bookmarks --plain --count 50
```

---

### unbookmark

Remove a tweet from bookmarks.

**Syntax:**
```
bird unbookmark <tweet-id-or-url>
```

**Description:** Removes the specified tweet from the authenticated user's bookmarks. Prints `unbookmarked <id>` on success.

**Examples:**
```sh
bird unbookmark 1234567890123456789
bird unbookmark https://x.com/user/status/1234567890123456789
```

---

### following

List users the account follows.

**Syntax:**
```
bird following [--user <numeric-id>]
```

**Description:** Returns the list of users that the given account follows. Defaults to the authenticated user. The `--user` flag requires a numeric user ID, not a handle.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--user` | string | `""` | Numeric Twitter user ID (not a handle) |
| `--limit` | int | `0` | Maximum number of users to fetch |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird following
bird following --user 123456789
bird following --limit 100 --json
```

---

### followers

List followers of the account.

**Syntax:**
```
bird followers [--user <numeric-id>]
```

**Description:** Returns the list of users who follow the given account. Defaults to the authenticated user. `--user` requires a numeric ID.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--user` | string | `""` | Numeric Twitter user ID (not a handle) |
| `--limit` | int | `0` | Maximum number of users to fetch |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird followers
bird followers --user 123456789 --limit 200
bird followers --json
```

---

### likes

Fetch tweets liked by the current user.

**Syntax:**
```
bird likes
```

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `0` | Maximum number of tweets to fetch |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird likes
bird likes --limit 50 --json
```

---

### whoami

Print the currently authenticated user.

**Syntax:**
```
bird whoami
```

**Description:** Resolves the credentials and prints the authenticated user's numeric ID, handle, and display name.

**Output:**
```
ID: 123456789
Username: @handle
Name: Display Name
```

**Examples:**
```sh
bird whoami
bird whoami --auth-token <token> --ct0 <ct0>
```

---

### about

Show account info for a user.

**Syntax:**
```
bird about <@handle>
```

**Description:** Fetches and displays public profile information for the given handle using the `AboutAccountQuery` endpoint.

**Output:**
```
ID: 123456789
Username: @handle
Name: Display Name
Followers: 12345
Following: 678
Created: Mon Jan 02 15:04:05 +0000 2006
```

**Examples:**
```sh
bird about @golang
bird about golang
bird about @someuser
```

---

### follow

Follow a user.

**Syntax:**
```
bird follow <@handle-or-numeric-id>
```

**Description:** Follows the given user. Accepts either `@handle` (resolved to a numeric ID via API lookup) or a bare numeric ID. Prints `followed <handle>` on success.

**Examples:**
```sh
bird follow @golang
bird follow 123456789
```

---

### unfollow

Unfollow a user.

**Syntax:**
```
bird unfollow <@handle-or-numeric-id>
```

**Description:** Unfollows the given user. Prints `unfollowed <handle>` on success.

**Examples:**
```sh
bird unfollow @spamaccount
bird unfollow 123456789
```

---

### user-tweets

Fetch tweets from a user's timeline.

**Syntax:**
```
bird user-tweets <@handle>
```

**Description:** Returns tweets posted by the given user. The handle argument may include or omit the `@` prefix.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `0` | Maximum number of tweets to fetch |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird user-tweets @golang
bird user-tweets golang --limit 50
bird user-tweets @user --json
```

---

### lists

List owned lists or memberships.

**Syntax:**
```
bird lists [--memberships]
```

**Description:** Prints the authenticated user's owned Twitter lists. With `--memberships`, shows lists the user is a member of instead.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--memberships` | bool | `false` | Show lists you are a member of |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird lists
bird lists --memberships
bird lists --json
```

---

### list-timeline

Fetch timeline for a list.

**Syntax:**
```
bird list-timeline <list-id-or-url>
```

**Description:** Returns tweets from the specified Twitter list's timeline. Accepts a numeric list ID or a URL of the form `https://x.com/i/lists/<id>`.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `0` | Maximum number of tweets to fetch |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird list-timeline 1234567890123456789
bird list-timeline https://x.com/i/lists/1234567890123456789
bird list-timeline 1234567890 --limit 30 --json
```

---

### news

Fetch news from explore tabs.

**Syntax:**
```
bird news [--tabs <tab1,tab2,...>]
```

**Description:** Fetches news items from Twitter's Explore tabs. Default tabs (when `--tabs` is not specified): `forYou`, `news`, `sports`, `entertainment`. Deduplicates items across tabs.

Available tab names: `forYou`, `news`, `sports`, `entertainment`, `trending`.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--tabs` | string | `""` | Comma-separated tab names |
| `--limit` | int | `0` | Maximum number of items per tab (default: 20) |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird news
bird news --tabs news,sports
bird news --tabs forYou --limit 10 --json
bird news --plain
```

---

### trending

Fetch trending topics.

**Syntax:**
```
bird trending
```

**Description:** Fetches trending topics from the `trending` Explore tab. This is a convenience alias equivalent to `bird news --tabs trending`.

**Command-specific flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `0` | Maximum number of items (default: 20) |
| `--json` | bool | `false` | Output as JSON |

**Examples:**
```sh
bird trending
bird trending --json
bird trending --limit 10 --plain
```

---

### check

Validate credentials and print current user.

**Syntax:**
```
bird check
```

**Description:** Resolves credentials, makes an authenticated API call, and prints `OK: @handle` on success. Prefixes errors with `FAIL:`. Useful for verifying that authentication is working correctly.

**Exit codes:** 0 on success (`OK: @handle`), 1 on any failure (`FAIL: ...`).

**Examples:**
```sh
bird check
# OK: @myhandle

bird check --browser safari
# OK: @myhandle

AUTH_TOKEN=... CT0=... bird check
# OK: @myhandle
```

---

### query-ids

Print the current fallback query ID cache.

**Syntax:**
```
bird query-ids
```

**Description:** Prints the hardcoded fallback GraphQL query IDs as JSON. Useful for debugging API query routing or verifying which IDs the binary was compiled with.

**Examples:**
```sh
bird query-ids
bird query-ids | jq '.TweetDetail'
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AUTH_TOKEN` | Twitter `auth_token` cookie (preferred over `TWITTER_AUTH_TOKEN`) |
| `TWITTER_AUTH_TOKEN` | Twitter `auth_token` cookie (alias) |
| `CT0` | Twitter `ct0` cookie (preferred over `TWITTER_CT0`) |
| `TWITTER_CT0` | Twitter `ct0` cookie (alias) |
| `BIRD_CONFIG` | Path to config file (bypasses default search order) |
| `BIRD_TIMEOUT_MS` | HTTP request timeout in milliseconds |
| `BIRD_COOKIE_TIMEOUT_MS` | Browser cookie extraction timeout in milliseconds |
| `BIRD_QUOTE_DEPTH` | Quoted tweet expansion depth |
| `BIRD_FEATURES_JSON` | Inline JSON string to override GraphQL feature flags |
| `BIRD_FEATURES_PATH` | Path to a JSON file containing feature flag overrides |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Runtime error (API failure, auth error, network error) |
| `2` | Usage error (unknown command, unknown flag, invalid argument) |

Exit code 2 is triggered by error messages containing: `unknown command`, `unknown flag`, `accepts`, `requires`, `invalid`, or `missing`.
