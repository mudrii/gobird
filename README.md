# gobird

[![Go Version](https://img.shields.io/badge/go-1.24%2B-blue)](https://go.dev/)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)](#development)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

gobird is a Twitter/X CLI tool and Go client library.

This project uses X/Twitter's unofficial private web APIs. It is intended for personal, research, and automation use, and upstream changes can break behavior without notice.

---

## Features

- Post tweets and replies with optional media attachments (images, video, GIFs)
- Read single tweets by ID or URL
- Fetch tweet threads and replies
- Search tweets with full-text queries
- Browse the home timeline (algorithmic and chronological)
- Fetch mentions, bookmarks (including bookmark folders), and liked tweets
- List and browse user timelines, followers, and following
- Follow and unfollow accounts
- Fetch owned lists, list memberships, and list timelines
- Browse explore news tabs and trending topics
- Three output modes: colourised human-readable, `--json`, and `--json-full` (with raw API data)
- Authentication from CLI flags, environment variables, or browser cookie extraction (Safari, Chrome, Firefox)
- JSON5 config file with environment variable overrides
- Paginated fetching with configurable limits and page caps
- Quoted tweet expansion with configurable depth
- Inspectable query IDs with runtime refresh from the X.com bundle

---

## Installation

### Release binaries

Prebuilt binaries are published on the GitHub Releases page for supported platforms.

1. Download the archive for your platform from `https://github.com/mudrii/gobird/releases`
2. Extract it
3. Move `gobird` into a directory on your `PATH`, for example:

```sh
tar -xzf gobird_26.03.15_darwin_arm64.tar.gz
install gobird /usr/local/bin/gobird
gobird --version
```

### go install

```sh
go install github.com/mudrii/gobird/cmd/gobird@latest
```

The binary is installed as `gobird`.

### Build from source

```sh
git clone https://github.com/mudrii/gobird.git
cd gobird
make build
# binary at bin/gobird
```

### Post-install configuration

`gobird` works without a config file if you pass credentials with flags, environment variables, or browser extraction. For a persistent setup, create a JSON5 config file at one of these locations:

- `~/.config/gobird/config.json5`
- `~/.gobirdrc.json5`

Minimal example:

```json5
{
  authToken: "your-auth-token",
  ct0: "your-ct0-token",
  browser: "safari",
  output: "human"
}
```

First-run checks:

```sh
gobird --version
gobird check --browser safari
gobird whoami
```

For all config keys and browser-specific options, see [docs/configuration.md](docs/configuration.md).

### Updating

If you installed with release binaries, download the new archive for the next release, replace the existing `gobird` binary, and run:

```sh
gobird --version
```

If you installed with `go install`, update with:

```sh
go install github.com/mudrii/gobird/cmd/gobird@latest
gobird --version
```

If you built from source, update by pulling the latest changes and rebuilding:

```sh
git pull --ff-only
make build
./bin/gobird --version
```

---

## Authentication

gobird needs two Twitter/X session cookies: `auth_token` (40 hex characters) and `ct0` (32–160 alphanumeric characters). Credentials are resolved in this priority order:

### Method 1: CLI flags

```sh
gobird --auth-token <token> --ct0 <ct0> whoami
```

### Method 2: Environment variables

```sh
export AUTH_TOKEN=abc123...  # or TWITTER_AUTH_TOKEN
export CT0=xyz789...         # or TWITTER_CT0
gobird whoami
```

### Method 3: Browser cookie extraction (automatic)

gobird can extract cookies directly from a logged-in browser with no manual copy-paste:

```sh
# Use the default order: Safari → Chrome → Firefox
gobird whoami

# Pin to a specific browser
gobird --browser chrome whoami
gobird --browser firefox --firefox-profile default-release whoami
gobird --browser safari whoami

# Specify multiple sources with explicit order
gobird --cookie-source chrome --cookie-source safari whoami
```

Supported browsers: **Safari** (macOS), **Chrome / Chromium**, **Firefox**.

To find your cookies manually: open x.com, open DevTools → Application → Cookies → `x.com`. Copy the values for `auth_token` and `ct0`.

---

## CLI Quick Start

### Read a tweet

```sh
$ gobird read 1867654321098765432
@golang (The Go Programming Language) [Mon Dec 16 14:22:01 +0000 2024]
Go 1.24 is out! Download it at https://go.dev/dl
replies:142 retweets:893 likes:4201
```

### Post a tweet

```sh
$ gobird tweet "Hello from gobird!"
1867700000000000001
```

### Reply to a tweet

```sh
$ gobird reply 1867654321098765432 "Great news!"
1867700000000000002
```

### Search

```sh
$ gobird search "golang generics" -n 5
@GopherAcademy (Gopher Academy) [Tue Dec 17 09:11:22 +0000 2024]
Generics in Go 1.24: what changed and what didn't
replies:8 retweets:37 likes:204
---
@go_trending (Go Trending) [Tue Dec 17 08:55:01 +0000 2024]
Five patterns for type-safe collections using generics
replies:3 retweets:21 likes:98
---
```

### Home timeline

```sh
$ gobird home -n 10
$ gobird home --latest -n 20   # chronological (Following tab)
```

### Mentions

```sh
$ gobird mentions -n 20
```

### Bookmarks

```sh
$ gobird bookmarks -n 50
$ gobird bookmarks --folder 1234567890123456789
```

### User tweets

```sh
$ gobird user-tweets @golang -n 20
```

### Fetch a thread

```sh
$ gobird thread 1867654321098765432
$ gobird thread 1867654321098765432 --filter full
```

### Check who you are

```sh
$ gobird whoami
ID: 783214
Username: @Twitter
Name: Twitter
```

### Follow / unfollow

```sh
$ gobird follow @golang
followed golang
$ gobird unfollow 13334762
unfollowed 13334762
```

### Trending and news

```sh
$ gobird trending
$ gobird news --tabs forYou,news
```

### JSON output

```sh
$ gobird read 1867654321098765432 --json
{
  "id": "1867654321098765432",
  "text": "Go 1.24 is out! ...",
  "author": { "username": "golang", "name": "The Go Programming Language" },
  "likeCount": 4201
}

# Include raw API response
$ gobird search "golang" --json-full | jq '.[0]._raw'
```

---

## Go Library Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mudrii/gobird/pkg/bird"
)

func main() {
    ctx := context.Background()

    // Option A: resolve credentials automatically (browser / env / flags)
    creds, err := bird.ResolveCredentials(bird.ResolveOptions{})
    if err != nil {
        log.Fatal(err)
    }

    client, err := bird.New(creds, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Option B: supply tokens directly
    // client, err := bird.NewWithTokens("your_auth_token", "your_ct0", nil)

    // Search (returns a single page; use GetAllSearchResults for pagination)
    page := client.Search(ctx, "golang", &bird.SearchOptions{
        Product: "Latest",
        FetchOptions: bird.FetchOptions{Limit: 10},
    })
    if page.Error != nil {
        log.Fatal(page.Error)
    }
    for _, t := range page.Items {
        fmt.Printf("@%s: %s\n", t.Author.Username, t.Text)
    }

    // Fetch a single tweet
    tweet, err := client.GetTweet(ctx, "1867654321098765432", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("@%s (%d likes): %s\n", tweet.Author.Username, tweet.LikeCount, tweet.Text)

    // Post a tweet
    id, err := client.Tweet(ctx, "Hello from gobird!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("posted:", id)
}
```

### Result types

Some read methods return a result struct instead of `(T, error)`. Check `result.Error` and `result.Success`:

```go
result := client.GetHomeTimeline(ctx, &bird.FetchOptions{Limit: 20})
if result.Error != nil {
    log.Fatal(result.Error)
}
for _, t := range result.Items {
    fmt.Println(t.Text)
}
```

Methods that follow this pattern: `Search`, `GetAllSearchResults`, `GetHomeTimeline`, `GetHomeLatestTimeline`, `GetBookmarks`, `GetBookmarkFolderTimeline`, `GetLikes`.

---

## Configuration

Config is loaded from the following locations:

- When `$BIRD_CONFIG` or `--config` is set: **only** that file is loaded (replaces default search)
- Otherwise, in order (later entries override earlier ones):
  1. `~/.config/gobird/config.json5` — global
  2. `./.gobirdrc.json5` — project-local

Config files use [JSON5](https://json5.org/) syntax (comments and trailing commas are allowed).

### Minimal example

```json5
// ~/.config/gobird/config.json5
{
  // Credentials (prefer env vars instead of storing tokens in a file)
  "authToken": "",
  "ct0": "",

  // Default browser for cookie extraction: "safari", "chrome", or "firefox"
  "defaultBrowser": "chrome",

  // Chrome profile name (optional)
  "chromeProfile": "Default",

  // HTTP request timeout in milliseconds (0 = no timeout)
  "timeoutMs": 10000,

  // Cookie extraction timeout in milliseconds (0 = no timeout)
  "cookieTimeoutMs": 5000,

  // Quoted tweet expansion depth (default: 1)
  "quoteDepth": 1,
}
```

All config fields can also be set via environment variables (see table below). Environment variables take precedence over file values.

---

## All Commands

| Command | Description |
|---|---|
| `tweet <text>` | Post a new tweet |
| `reply <id-or-url> <text>` | Reply to a tweet |
| `read <id-or-url>` | Read a single tweet |
| `replies <id-or-url>` | Fetch replies to a tweet |
| `thread <id-or-url>` | Fetch a full tweet thread |
| `search <query>` | Search tweets |
| `mentions` | Fetch mentions of the authenticated user |
| `home` | Fetch home timeline (algorithmic) |
| `bookmarks` | Fetch bookmarks |
| `unbookmark <id-or-url>` | Remove a tweet from bookmarks |
| `likes` | Fetch liked tweets |
| `following` | List accounts the user follows |
| `followers` | List accounts following the user |
| `follow <@handle-or-id>` | Follow a user |
| `unfollow <@handle-or-id>` | Unfollow a user |
| `user-tweets <@handle>` | Fetch a user's tweet timeline |
| `lists` | List owned lists (or memberships with `--memberships`) |
| `list-timeline <id-or-url>` | Fetch tweets from a list |
| `news` | Fetch explore news tabs |
| `trending` | Fetch trending topics |
| `whoami` | Print the authenticated user |
| `about <@handle>` | Show account info for a user |
| `check` | Verify credentials are valid |
| `query-ids` | Show active GraphQL query IDs |
| `version` | Print version information |

Pass a tweet ID or URL to `gobird` without a subcommand to read that tweet directly:

```sh
gobird 1867654321098765432
```

---

## Output Formats

All commands accept mutually exclusive output flags:

| Flag | Output |
|---|---|
| (none) | Human-readable, ANSI-coloured |
| `--plain` | Human-readable, no colour, no emoji |
| `--json` | Normalised JSON array / object |
| `--json-full` | Normalised JSON with `_raw` field containing the raw API response |

Use `--no-color` to disable ANSI colour while keeping emoji. Use `--no-emoji` to disable emoji while keeping colour.

---

## Environment Variables

| Variable | Description |
|---|---|
| `AUTH_TOKEN` | Twitter `auth_token` cookie (preferred) |
| `TWITTER_AUTH_TOKEN` | Twitter `auth_token` cookie (alias) |
| `CT0` | Twitter `ct0` cookie (preferred) |
| `TWITTER_CT0` | Twitter `ct0` cookie (alias) |
| `CHROME_SAFE_STORAGE_PASSWORD` | Optional macOS Chrome keychain password override for browser cookie decryption when Keychain subprocess access is denied |
| `BIRD_CONFIG` | Explicit path to config file |
| `BIRD_TIMEOUT_MS` | HTTP request timeout in milliseconds |
| `BIRD_COOKIE_TIMEOUT_MS` | Browser cookie extraction timeout in milliseconds |
| `BIRD_QUOTE_DEPTH` | Quoted tweet expansion depth |

---

## Requirements

- Go 1.24 or later (module currently declares `go 1.24.0`)
- macOS or Linux
- For browser cookie extraction: Safari, Chrome / Chromium, or Firefox must be installed and logged in to x.com

---

## Development

```sh
# Build the binary to bin/gobird
make build

# Run all tests
make test

# Run tests with race detector
make test-race

# Run fmt-check, vet, tests, race-detector, lint, and build (mirrors CI)
make ci

# Run linter (requires golangci-lint)
make lint

# Format source
make fmt

# Generate coverage report (coverage.html)
make coverage

# Remove build artefacts
make clean
```

Install `golangci-lint`:

```sh
brew install golangci-lint          # macOS
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

## Open source project docs

- [LICENSE](LICENSE)
- [CHANGELOG](CHANGELOG.md)
- [CONTRIBUTING](CONTRIBUTING.md)
- [CODE OF CONDUCT](CODE_OF_CONDUCT.md)
- [SECURITY](SECURITY.md)

---

## License

MIT. See [LICENSE](LICENSE).

---

## Related

- [`docs/`](docs/) — architecture notes, API correction log, development guide, and agent context
- [`pkg/bird/`](pkg/bird/) — public Go library (importable by other projects)
- [`internal/client/constants.go`](internal/client/constants.go) — query ID maps and API base URLs
