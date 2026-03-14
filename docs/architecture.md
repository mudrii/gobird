# gobird Architecture

## System Overview and Purpose

gobird is a Twitter/X client implemented in Go with two public surfaces:

1. **CLI binary** (`gobird`) — a command-line tool that reads tweets, searches, manages bookmarks, follows/unfollows accounts, posts tweets, and inspects news/trending content. It is the primary end-user interface.
2. **Go library** (`pkg/bird`) — a fully exported package that external Go programs can import to drive the same API operations programmatically.

Both surfaces share one implementation path. The CLI constructs a `bird.Client`, calls the same methods any library consumer would call, and formats the results for the terminal.

The underlying protocol is Twitter/X's private GraphQL and REST v1.1 APIs. gobird authenticates using session cookies extracted from a real browser (`auth_token` + `ct0`), mirroring what the x.com browser client sends. No official developer API key is required.

---

## Component Diagram

```
┌───────────────────────────────────────────────────────────────────────────┐
│ cmd/gobird/main.go                                                        │
│  (binary entrypoint, wires SetBuildInfo + NewRootCmd)                    │
└──────────────────────────────────┬────────────────────────────────────────┘
                                   │ calls
┌──────────────────────────────────▼────────────────────────────────────────┐
│ internal/cli/                                                             │
│  root.go, search.go, home.go, bookmarks.go, users.go,                    │
│  lists.go, news.go, tweet.go, read.go, user_tweets.go, check.go,         │
│  query_ids.go, shared.go                                                  │
│  (cobra commands; resolves config + credentials, builds bird.Client,      │
│   formats output via internal/output)                                     │
└──────────────────────────────────┬────────────────────────────────────────┘
                                   │ imports (public API)
┌──────────────────────────────────▼────────────────────────────────────────┐
│ pkg/bird/                                                                 │
│  client.go, client_methods.go, auth.go, types.go, errors.go, doc.go      │
│  (thin facade; type aliases to internal/types; wraps internal/client)     │
└──────────────────────────────────┬────────────────────────────────────────┘
                                   │ wraps
┌──────────────────────────────────▼────────────────────────────────────────┐
│ internal/client/                                                          │
│  client.go, http.go, headers.go, features.go, query_ids.go,              │
│  pagination.go, constants.go, search.go, home.go, bookmarks.go,          │
│  following.go, user_lookup.go, user_tweets.go, tweet_detail.go,          │
│  engagement.go, follow.go, post.go, media.go, news.go, lists.go,         │
│  timelines.go, raw.go, users.go                                           │
│  (all API logic: HTTP, auth headers, retry, query-ID resolution,          │
│   feature flags, pagination)                                              │
└──────────┬───────────────────────┬────────────────────────────────────────┘
           │ calls                 │ uses types from
┌──────────▼───────────┐  ┌───────▼──────────────────────────────────────┐
│ internal/parsing/    │  │ internal/types/                              │
│  tweet.go            │  │  models.go   — public normalized structs     │
│  timeline.go         │  │  wire.go     — GraphQL wire shapes           │
│  cursors.go          │  │  options.go  — fetch/search/thread options   │
│  media.go            │  │  results.go  — generic PageResult/           │
│  article.go          │  │               PaginatedResult                │
│  users.go            │  └──────────────────────────────────────────────┘
│  lists.go            │
│  news.go             │
│  thread_filters.go   │
│  input.go            │
└──────────────────────┘

┌─────────────────────────────────────────┐
│ internal/auth/                          │
│  resolve.go, safari.go, chrome.go,      │
│  firefox.go, cookies.go                 │
│  (three-tier credential resolution)     │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ internal/config/                        │
│  config.go                              │
│  (JSON5 config loading + env overlay)   │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ internal/output/                        │
│  json.go, format.go                     │
│  (JSON + terminal formatted output)     │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ internal/testutil/                      │
│  golden.go                              │
│  (golden-file test helpers)             │
└─────────────────────────────────────────┘
```

---

## Layer Architecture

The codebase follows a strict layered dependency graph with no upward imports:

```
cmd  →  internal/cli  →  pkg/bird  →  internal/client  →  internal/parsing  →  internal/types
                    ↘                ↗
                  internal/auth
                  internal/config
                  internal/output
```

Each layer may only import layers below it. `internal/types` has no internal imports. `internal/parsing` imports only `internal/types`. `internal/client` imports `internal/parsing` and `internal/types`. `pkg/bird` wraps `internal/client` and re-exports `internal/types` and `internal/auth`. `internal/cli` uses `pkg/bird` exclusively for API calls.

---

## Package Responsibilities

| Package | Responsibility |
|---|---|
| `cmd/gobird` | Binary entrypoint. Calls `cli.SetBuildInfo` and `cli.NewRootCmd`, runs cobra, maps errors to exit codes. |
| `internal/cli` | All cobra command definitions. Reads config, resolves credentials, builds `bird.Client`, dispatches API calls, formats output. |
| `internal/auth` | Three-tier credential resolution: CLI flags → environment variables → browser cookie extraction (Safari, Chrome, Firefox). Validates `auth_token` (40 hex chars) and `ct0` (32–160 alphanumeric chars). |
| `internal/config` | Loads `~/.config/gobird/config.json5` and `./.gobirdrc.json5` (local overrides global). Accepts JSON5 via `hujson`. Applies env var overlay on top. |
| `pkg/bird` | Public stable API surface. Contains type aliases (not re-declarations) for every type in `internal/types`. Wraps `internal/client.Client` in `bird.Client`. Delegates all methods one-for-one. |
| `internal/client` | All Twitter/X API logic. HTTP request construction, credential headers, query-ID resolution and refresh, feature flag sets, pagination patterns, retry strategies. Splits into domain files per operation group. |
| `internal/parsing` | Pure transformation functions: wire GraphQL structs → normalized `types.*` structs. No HTTP, no state. |
| `internal/types` | All data types: normalized output models, wire (GraphQL response) shapes, option structs, generic result types. No logic. |
| `internal/output` | Terminal and JSON rendering. |
| `internal/testutil` | Golden-file test infrastructure. |

---

## Data Flow

```
User invokes CLI
       │
       ▼
internal/cli command handler
  ├─ Load config (internal/config.Load)
  ├─ Resolve credentials (internal/auth.ResolveCredentials)
  │     ├─ Tier 1: --auth-token / --ct0 flags
  │     ├─ Tier 2: AUTH_TOKEN / CT0 env vars
  │     └─ Tier 3: Browser SQLite cookie DB (Safari/Chrome/Firefox)
  ├─ Build client: bird.New(creds, opts)  →  internal/client.New(authToken, ct0, opts)
  │     └─ Generates clientUUID and deviceID (uuid.NewString)
  │     └─ Seeds queryIDCache from opts.QueryIDCache if provided
  └─ Call API method (e.g., client.GetAllSearchResults)

API method (internal/client)
  ├─ Resolve query ID(s): runtime cache → BundledBaseline → FallbackQueryIDs
  ├─ Build feature flags map (buildXxxFeatures)
  ├─ Serialize variables to JSON
  ├─ Build URL: GraphQLBaseURL + "/" + queryID + "/" + operation
  ├─ Build headers (baseHeaders + authToken + ct0 + clientUUID + deviceID)
  ├─ HTTP request (doGET or doPOSTJSON)
  │     └─ On retryable status (429/500-504): fetchWithRetry (bookmarks only)
  │     └─ On 404: withRefreshedQueryIDsOn404 → scrapeQueryIDs → retry
  │     └─ On GRAPHQL_VALIDATION_FAILED: refreshQueryIDs → retry (search)
  │     └─ On "query: unspecified": refreshQueryIDs → retry (home timeline)
  ├─ Unmarshal JSON body into wire structs (types.WireXxx)
  └─ Parse: internal/parsing functions
        ├─ CollectTweetResultsFromEntry (timeline → []WireRawTweet)
        ├─ UnwrapTweetResult (handles TweetWithVisibilityResults)
        ├─ MapTweetResultWithOptions (WireRawTweet → TweetData)
        │     ├─ Text: article title → note tweet text → legacy full_text
        │     ├─ IsBlueVerified: top-level result field (NOT legacy)
        │     └─ QuotedTweet: recursive with QuoteDepth-1
        └─ ExtractCursorFromInstructions (Bottom cursorType)

Result returned to CLI
  └─ internal/output formats to terminal or JSON
  └─ ExitCode maps error type to exit code (0/1/2)
```

---

## Key Design Decisions

### Why `internal/` is not exported

All packages under `internal/` are Go's standard mechanism for restricting import access to within the module. This allows the module authors to change wire types, parsing algorithms, and client internals without making any breaking API promises. Only `pkg/bird` is the stable surface.

### Why `pkg/bird` re-exports with type aliases

`pkg/bird` uses Go type aliases (`type TweetData = types.TweetData`) rather than separate type declarations. This means a consumer who imports `pkg/bird` gets the exact same concrete types as those flowing through the entire pipeline. There is no conversion cost and no duplication of struct definitions. The only code in `pkg/bird` is the public constructor, method delegation, and the alias list.

### Why the `bird.Client` wraps `internal/client.Client`

The wrapping exists to allow the internal client to evolve independently. The `bird.Client` holds a single unexported `*client.Client` field and delegates every method verbatim. This pattern keeps the public API surface clean while the implementation layer is free to add, rename, or restructure internal methods.

### Why credentials are validated with regex at resolution time

Malformed cookies would produce confusing HTTP 401/403 errors deep inside request logic. Early regex validation in `internal/auth` (`auth_token`: 40 hex chars; `ct0`: 32–160 alphanumeric) surfaces the problem immediately with an actionable message before any network call is made.

### Why query IDs are scraped at runtime

Twitter/X's private GraphQL uses operation-specific query IDs (`queryId` field in request body, also embedded in the URL). These IDs rotate when X deploys new frontend bundles. The hardcoded fallback IDs (`FallbackQueryIDs`, `BundledBaselineQueryIDs`) become stale over time. `scrapeQueryIDs` fetches four known X.com page URLs, extracts all `abs.twimg.com/*.js` bundle URLs, fetches each bundle, and regex-matches `<queryId>/<OperationName>` patterns to populate the runtime cache. This keeps the tool functional after X frontend deploys without requiring a gobird update.

### Why pagination is a shared helper

`paginateInline` in `pagination.go` encodes the complete stop-condition logic for timeline pagination (empty cursor, cursor unchanged, empty page, zero new items, max-pages, limit). Centralising this eliminates copy-paste bugs across Search, Home, Bookmarks, Likes, and similar operations that all follow the same loop shape.

---

## Dependency Graph

| Dependency | Why |
|---|---|
| `github.com/spf13/cobra` | CLI command tree, persistent flags, usage generation |
| `github.com/tailscale/hujson` | JSON5 parsing for config files (allows comments and trailing commas) |
| `modernc.org/sqlite` | Pure-Go SQLite driver used by Firefox and Chrome cookie extractors to read browser cookie databases without requiring a system `libsqlite3` |
| `github.com/google/uuid` | Generates `clientUUID` and `deviceID` on each client construction |
| `github.com/mattn/go-isatty` | Detects whether stdout is a terminal (for color/emoji output decisions) |
| `github.com/spf13/pflag` | Cobra dependency: POSIX-style flag parsing |
| `golang.org/x/sys` | Required by `modernc.org/sqlite` for low-level OS calls |

---

## Concurrency Model

The `internal/client.Client` struct contains three categories of shared state protected by separate mutexes:

### Query ID cache (`queryIDMu sync.RWMutex`)

`queryIDCache` is a `map[string]string` that maps operation names to their current query IDs. All reads use `queryIDMu.RLock()` and go through `getQueryID`. All writes happen inside `refreshQueryIDs`, which holds `queryIDMu.Lock()` while updating the map and recording `queryIDRefreshAt`. This is the most frequently contended lock: every API call reads it, and scrape refreshes write it.

### User ID cache (`userIDMu sync.RWMutex`)

`userID` (the authenticated account's numeric ID) is resolved lazily by `ensureClientUserID`. The fast path holds only a read lock. The slow path calls `getCurrentUser` without holding any lock (because `getCurrentUser` itself needs to read-lock `userIDMu` via `cachedUserID`), then acquires the write lock only to store the result. A double-checked pattern guards against redundant writes: if another goroutine resolved it first, the second write is a no-op. Only successful resolutions are cached; errors are not, allowing callers to retry.

### Feature overrides (`featureOverridesOnce sync.Once`)

`loadFeatureOverrides` uses a package-level `sync.Once` to parse `BIRD_FEATURES_JSON` or `BIRD_FEATURES_PATH` exactly once per process lifetime. The result is stored in a package-level `featureOverrides` variable and never mutated after initialization.

The HTTP client itself (`*http.Client`) is safe for concurrent use by the standard library's contract.

Pagination loops run sequentially within a single goroutine. Browser cookie extraction can optionally run with a timeout by spawning a goroutine and selecting on a context deadline channel.

---

## Error Propagation Strategy

```
HTTP transport error
  └─ returned as Go error from net/http
       └─ wrapped in context.Err() if context cancelled
            └─ returned to internal/client method caller
                 └─ returned to pkg/bird method caller
                      └─ returned to internal/cli command handler
                           └─ printed to stderr
                                └─ mapped by ExitCode() to exit code 1

HTTP non-2xx response
  └─ internal/client wraps as *httpError{StatusCode, Body}
       ├─ is404(err) checks for 404 → triggers query ID refresh
       ├─ retryableStatus(code) checks 429/500-504 → fetchWithRetry
       └─ all other status codes propagate as-is

GraphQL-level errors (200 OK with errors[] in body)
  └─ parseGraphQLErrors extracts []graphqlError from body
       ├─ isSearchQueryIDMismatch → GRAPHQL_VALIDATION_FAILED → refresh
       ├─ queryUnspecifiedRe match → HomeTimeline refresh
       └─ graphQLError() → fmt.Errorf wrapping first error message

Validation errors (bad credentials, bad flags)
  └─ returned synchronously from auth.ResolveCredentials or cobra PreRunE
       └─ ExitCode maps "invalid" substring → exit code 2

Partial success (some pages fetched, then error)
  └─ PaginatedResult{Items: accumulated, Success: false, Error: err}
       └─ CLI treats as partial success: prints items, prints error to stderr

Exit codes:
  0 — success
  1 — API error, network error, credential failure
  2 — usage error (unknown flag, invalid argument, missing required value)
```

Errors are never silently swallowed except in two documented places:
- `refreshQueryIDs` silently ignores scraping errors to preserve availability (the old IDs remain in cache).
- `scrapeQueryIDs` silently skips individual page or JS bundle fetch failures and returns whatever was found.
