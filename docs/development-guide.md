# Development Guide

## Prerequisites

- **Go**: 1.26.1 or later (see `go.mod`)
- **golangci-lint**: for linting (`go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest` or via Homebrew)
- **git**: for version injection into the binary
- **macOS**: required for browser cookie extraction features (Chrome AES-CBC decryption via Keychain, Safari's `Cookies.db`, Firefox SQLite stores)

No other tooling is required. The project uses only the Go standard library plus a small set of direct dependencies: `cobra`, `hujson`, `sqlite`, `uuid`.

---

## Repository Structure

```
gobird/
├── cmd/gobird/           # main package — entry point, version injection
├── internal/
│   ├── auth/             # browser cookie extraction (chrome, safari, firefox) + credential resolution
│   ├── cli/              # cobra command definitions (one file per command group)
│   ├── client/           # Twitter/X API client — all HTTP, GraphQL, REST logic
│   ├── config/           # JSON5 config file loading, env var overlay
│   ├── output/           # terminal formatting (color, emoji, JSON, plain text)
│   ├── parsing/          # GraphQL response parsing — cursors, tweets, users, news
│   ├── runtime/          # runtime support (feature override env vars)
│   ├── testutil/         # shared test helpers (mock HTTP server, golden files)
│   └── types/            # all shared types: wire types, models, options, results
├── pkg/bird/             # public Go API — thin wrapper around internal/client
├── tests/
│   ├── acceptance/       # CLI acceptance tests
│   ├── fixtures/         # JSON wire fixtures used by tests
│   ├── golden/           # golden output files (regenerated with -update)
│   └── integration/      # integration tests (require real credentials)
├── docs/                 # this directory
├── Makefile
├── go.mod
└── .golangci.yml
```

### Directory Details

**`cmd/gobird/`**: The `main` package. Sets `buildVersion` and `buildGitSHA` via ldflags, constructs the root cobra command via `cli.NewRootCmd()`, and calls `os.Exit(cli.ExitCode(err))`.

**`internal/auth/`**: Credential resolution. `resolve.go` orchestrates a three-tier priority: CLI flags → environment variables → browser cookie extraction. `chrome.go` implements AES-128-CBC decryption using a key derived from macOS Keychain via PBKDF2-SHA1. `safari.go` reads `Cookies.db`. `firefox.go` reads `moz_cookies`. All use `modernc.org/sqlite` opened read-only with `?mode=ro&immutable=1`.

**`internal/cli/`**: One file per command group. `root.go` defines global persistent flags and registers all subcommands. `shared.go` contains `resolveClient()` (the auth resolution + client construction chain used by every command). New commands are registered in `root.go` and implemented in their own file.

**`internal/client/`**: The core API client. `client.go` defines `Client` and `Options`. All HTTP dispatch goes through `http.go` (`doGET`, `doPOSTJSON`, `doPOSTForm`, `fetchWithRetry`). Query ID management lives in `query_ids.go`. Feature maps live in `features.go`. Domain operations are split: `post.go` (tweet creation), `engagement.go` (like/retweet/bookmark), `follow.go`, `following.go`, `home.go`, `search.go`, `tweet_detail.go`, `user_tweets.go`, `bookmarks.go`, `lists.go`, `news.go`, `timelines.go`, `media.go`, `users.go`, `user_lookup.go`. Pagination logic is in `pagination.go`. Constants and all hardcoded query IDs are in `constants.go`.

**`internal/parsing/`**: Stateless parsing functions. Receives `[]types.WireTimelineInstruction` and returns normalized `[]types.TweetData` or related types. Cursor extraction is in `cursors.go`. Thread filtering is in `thread_filters.go`. Article Draft.js rendering is in `article.go`.

**`internal/types/`**: All shared type definitions. `models.go` defines normalized output types (`TweetData`, `TwitterUser`, etc.). `wire.go` defines raw GraphQL response structs. `options.go` defines fetch option structs. `results.go` defines generic `PageResult[T]` and `PaginatedResult[T]`.

**`internal/config/`**: JSON5 config file loading. Uses `tailscale/hujson` to normalize JSON5 to standard JSON. Searches `~/.config/bird/config.json5` then `./.birdrc.json5`. Environment variables (`AUTH_TOKEN`, `CT0`, `BIRD_TIMEOUT_MS`, etc.) overlay the file config.

**`internal/output/`**: Terminal output formatting. Provides `PrintJSON`, colored tweet rendering, and `FormatOptions` (plain/no-color/no-emoji).

**`internal/testutil/`**: Shared test helpers. `httpmock.go` provides `NewTestServer`, `StaticHandler`, and `NewHTTPClientForServer`. `golden.go` provides `AssertGolden` and the `-update` flag.

**`pkg/bird/`**: Public API surface. `client.go` defines `Client` as a wrapper around `internal/client.Client`. `client_methods.go` exposes all public methods as one-line delegating wrappers. `types.go` re-exports the internal types. `auth.go` exposes `ResolveCredentials`. `errors.go` defines public sentinel errors.

---

## Build Instructions

```bash
# Build the binary to bin/bird
make build

# Direct go build (equivalent)
go build -ldflags "-X main.version=$(git describe --tags --always --dirty) -X main.gitSHA=$(git rev-parse --short HEAD)" -o bin/bird ./cmd/gobird
```

The binary embeds `version` and `gitSHA` at link time. Without git, these default to `dev` and `unknown`.

---

## Running Tests

```bash
# Run all tests
make test
# Equivalent: go test ./...

# Run with race detector (always use this before committing)
make test-race
# Equivalent: go test -race ./...

# Generate coverage report
make coverage
# Produces coverage.out and opens coverage.html
```

Tests that require real credentials (integration tests under `tests/integration/`) are guarded and do not run in normal `go test ./...` unless explicitly invoked. Browser cookie extraction tests (`internal/auth/`) are not unit-testable without a real browser profile present.

---

## Linting

```bash
make lint        # runs golangci-lint run
make vet         # runs go vet ./...
make fmt         # runs gofmt -w .
```

The `.golangci.yml` enables five linters: `errcheck`, `govet`, `staticcheck`, `unused`, `revive`. The `revive` `exported` rule is configured with `disableStutteringCheck`. Test files are exempted from `errcheck` and `revive`. `errcheck` is also suppressed for `.Close()` return values (annotated with `//nolint:errcheck`).

---

## Adding a New API Operation

Follow these steps in order. All steps are required for a complete, correctly wired operation.

### Step 1: Add to `FallbackQueryIDs` in `internal/client/constants.go`

`FallbackQueryIDs` must have exactly one entry per supported operation. Add a new key matching the Twitter GraphQL operation name exactly (case-sensitive, matches the URL path segment):

```go
var FallbackQueryIDs = map[string]string{
    // ... existing entries ...
    "MyNewOperation": "HARDCODED_QUERY_ID_HERE",
}
```

The operation name must match the Twitter URL path: `.../graphql/<queryId>/MyNewOperation`.

### Step 2: Add to `BundledBaselineQueryIDs` if the ID is known

If you have a verified, more current query ID than the fallback, add it here as well. If uncertain, skip this step — the fallback will be used.

```go
var BundledBaselineQueryIDs = map[string]string{
    // ... existing entries ...
    "MyNewOperation": "NEWER_QUERY_ID",
}
```

### Step 3: Add to `PerOperationFallbackIDs` if multiple IDs are known

For operations with multiple known valid query IDs to try in sequence:

```go
var PerOperationFallbackIDs = map[string][]string{
    // ... existing entries ...
    "MyNewOperation": {"PRIMARY_ID", "SECONDARY_ID"},
}
```

If the operation has only one known ID, you can omit it from `PerOperationFallbackIDs` — `getQueryID` will fall back to `FallbackQueryIDs` automatically.

### Step 4: Create the client method

Create or add to an appropriate file in `internal/client/`. For GET-based GraphQL operations, follow this pattern:

```go
// GetMyNewData fetches data from MyNewOperation.
func (c *Client) GetMyNewData(ctx context.Context, someID string, opts *types.FetchOptions) (SomeResult, error) {
    queryID := c.getQueryID("MyNewOperation")
    vars := map[string]any{
        "some_id": someID,
        // ... other variables
    }
    varsJSON, err := json.Marshal(vars)
    if err != nil {
        return SomeResult{}, fmt.Errorf("client: GetMyNewData: marshal vars: %w", err)
    }
    reqURL := fmt.Sprintf("%s/%s/MyNewOperation?variables=%s&features=%s",
        GraphQLBaseURL, queryID,
        url.QueryEscape(string(varsJSON)),
        url.QueryEscape(string(featuresJSON)),
    )
    body, err := c.doGET(ctx, reqURL, c.getJSONHeaders())
    if err != nil {
        return SomeResult{}, fmt.Errorf("client: GetMyNewData: %w", err)
    }
    return parseMyNewData(body)
}
```

For POST operations (mutations like CreateTweet), use `c.doPOSTJSON`. Always use `fmt.Errorf("package: operation: %w", err)` for error wrapping.

### Step 5: Add feature flags via `buildXxxFeatures()`

In `internal/client/features.go`, add a new builder if the operation has a distinct feature set. Always start from `cloneFeatures()` of the most similar existing set — never mutate a shared map in place:

```go
// buildMyNewFeatures returns features for MyNewOperation.
func buildMyNewFeatures() map[string]any {
    f := cloneFeatures(buildArticleFeatures()) // choose appropriate base
    f["some_new_feature_flag"] = true
    return applyFeatureOverrides("myNew", f)
}
```

If the operation uses the same features as an existing operation, reuse its builder directly.

### Step 6: Add parsing in `internal/parsing/`

Add parsing logic for the new response shape. Parsing functions must be pure/stateless — they take `[]byte` or `[]types.WireTimelineInstruction` and return normalized types. Never call HTTP from parsing.

```go
// In internal/parsing/mynew.go
func ParseMyNewResponse(body []byte) ([]types.SomeType, error) {
    var env struct {
        Data struct {
            MyNewOperation struct {
                Result SomeWireShape `json:"result"`
            } `json:"my_new_operation"`
        } `json:"data"`
    }
    if err := json.Unmarshal(body, &env); err != nil {
        return nil, fmt.Errorf("parsing: ParseMyNewResponse: %w", err)
    }
    // ... map wire types to normalized types
}
```

If the response returns wire tweets via timeline instructions, use the existing `parsing.ParseTweetsFromInstructions` and `parsing.ExtractCursorFromInstructions`.

### Step 7: Add the CLI command in `internal/cli/`

Create a new file (e.g., `internal/cli/mynew.go`) with the command constructor:

```go
package cli

import (
    "github.com/spf13/cobra"
)

func newMyNewCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "mynew <arg>",
        Short: "Short description",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            c, err := resolveClient()
            if err != nil {
                return fmt.Errorf("auth: %w", err)
            }
            result, err := c.GetMyNewData(cmd.Context(), args[0], nil)
            if err != nil {
                return fmt.Errorf("mynew: %w", err)
            }
            // format and print using output package
            return nil
        },
    }
}
```

Then register it in `internal/cli/root.go`:

```go
root.AddCommand(newMyNewCmd())
```

Always use `cmd.Context()` — never `context.Background()` — to propagate cancellation from the CLI.

### Step 8: Export via `pkg/bird/`

Add a delegating wrapper to `pkg/bird/client_methods.go`:

```go
// GetMyNewData fetches data using MyNewOperation.
func (c *Client) GetMyNewData(ctx context.Context, someID string, opts *SomeOptions) (SomeResult, error) {
    return c.c.GetMyNewData(ctx, someID, opts)
}
```

If new option or result types are needed, add them to `internal/types/` and re-export them from `pkg/bird/types.go`.

### Step 9: Add tests

Add a `_test.go` file in `internal/client/` using the mock HTTP pattern. See the [Testing Guide](testing-guide.md) for complete patterns.

---

## Adding a New CLI Command

1. Create `internal/cli/mycommand.go` with `func newMyCommandCmd() *cobra.Command { ... }`.
2. Register it with `root.AddCommand(newMyCommandCmd())` in `root.go`.
3. Always call `resolveClient()` at the start of `RunE` to get an authenticated client.
4. Always pass `cmd.Context()` to client methods — never `context.Background()`.
5. Always call `validateOutputFlags()` if the command supports `--json` / `--plain`.
6. Use `output.PrintJSON(cmd.OutOrStdout(), ...)` for JSON output.
7. Use `cmd.Println(...)` for plain text output to ensure proper test capture.

---

## Debugging API Calls

### Query ID Issues

When an operation returns HTTP 404 or a GraphQL `GRAPHQL_VALIDATION_FAILED` error, the query ID is stale. The client handles this automatically via `refreshQueryIDs()`, but if you suspect a stale ID:

```bash
# Check the currently active query ID for any operation
gobird query-ids

# Force a refresh by running a no-op that triggers the scraper
BIRD_FEATURES_JSON='{}' gobird check
```

The scraper in `query_ids.go` fetches `https://x.com/home`, `https://x.com/i/bookmarks`, `https://x.com/explore`, and `https://x.com/settings/account`, then scans embedded JS bundles for patterns like `<queryId>/OperationName`.

To debug a specific operation's query ID resolution order, check `PerOperationFallbackIDs` in `constants.go` — the client tries the primary (runtime-cached or bundled) ID first, then any additional IDs listed.

### Feature Flag Mismatches

Feature flag mismatches typically manifest as empty result sets or GraphQL errors about unknown fields. To test with different feature sets without recompiling:

```bash
# Apply a global feature override
export BIRD_FEATURES_JSON='{"global": {"some_flag": true}}'

# Apply operation-specific overrides
export BIRD_FEATURES_JSON='{"sets": {"search": {"rweb_video_timestamps_enabled": false}}}'

# Load from a file
export BIRD_FEATURES_PATH=/path/to/features.json
```

The set names correspond to the second argument of `applyFeatureOverrides()` in `features.go`: `"article"`, `"tweetDetail"`, `"search"`, `"tweetCreate"`, `"timeline"`, `"bookmarks"`, `"likes"`, `"homeTimeline"`, `"lists"`, `"userTweets"`, `"following"`, `"explore"`, `"myNew"` (for new operations you add).

---

## Browser Cookie Extraction Debugging

All browser extractors open the cookie database read-only with `?mode=ro&immutable=1`. If extraction fails:

**Chrome**: The extractor calls `security find-generic-password -w -a Chrome -s "Chrome Safe Storage"` to get the AES key from macOS Keychain. If Chrome is running, the database may be locked — close Chrome or use the `--chrome-profile-dir` flag to point at a backup copy of the Cookies file.

Candidate paths searched (in order):
1. Explicit `--chrome-profile-dir` or `--chrome-profile` if provided
2. `~/Library/Application Support/Google/Chrome/Default/Cookies`
3. `~/Library/Application Support/Google/Chrome/Profile 1/Cookies`
4. `~/Library/Application Support/Chromium/Default/Cookies`

**Safari**: Reads from `~/Library/Containers/com.apple.Safari/Data/Library/Cookies/Cookies.db` (sandboxed), falling back to `~/Library/Cookies/Cookies.db`.

**Firefox**: Scans all subdirectories of `~/Library/Application Support/Firefox/Profiles/` for `cookies.sqlite`. Use `--firefox-profile <name>` to narrow to a specific profile.

Common failure reasons:
- Browser is open and has exclusive lock on the database
- `auth_token` or `ct0` cookies are missing (not logged in to x.com/twitter.com)
- Sandboxing permissions preventing database read (grant Full Disk Access in System Preferences)
