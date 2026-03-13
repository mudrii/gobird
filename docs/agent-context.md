# Agent Context: gobird Codebase

**Read this document before making any changes to this repository.**

This document records verified behavioral facts about the gobird codebase, including corrections to common misconceptions about the Twitter/X GraphQL API. Every section marked "correction" has been empirically verified against real API responses.

---

## Table of Contents

1. [Critical Corrections Index](#critical-corrections-index)
2. [Package Boundaries](#package-boundaries)
3. [Naming Conventions](#naming-conventions)
4. [Mutex Discipline Rules](#mutex-discipline-rules)
5. [Error Wrapping Rules](#error-wrapping-rules)
6. [Common Pitfalls](#common-pitfalls)
7. [Query ID System](#query-id-system)
8. [Feature Flag System](#feature-flag-system)
9. [Wire Type System](#wire-type-system)
10. [Pagination Patterns](#pagination-patterns)
11. [How Tests Are Structured](#how-tests-are-structured)
12. [What Staticcheck and Golangci-lint Will Flag](#what-staticcheck-and-golangci-lint-will-flag)
13. [Operation-Specific Notes](#operation-specific-notes)

---

## Critical Corrections Index

These are the most important behavioral facts. Violating them will produce silent failures, empty results, or wrong data — often with no error returned.

### #7: `cursorType` comes from `entry.content`, NOT `entry.content.itemContent`

The Bottom cursor for pagination is at `entry.content.cursorType` and `entry.content.value`.

Correct path: `WireEntry.Content.CursorType` (field `cursorType` on `WireContent`)

Wrong paths (do not use):
- `entry.content.itemContent.cursorType` — `itemContent` is only for tweet/user items
- `entry.entryType` — this is a different field
- Checking `__typename == "TimelineTimelineCursor"` in addition to `cursorType` — `cursorType` alone is sufficient

Code in `internal/parsing/cursors.go`:
```go
if entry.Content.CursorType == "Bottom" {
    return entry.Content.Value
}
```

### #8: `isBlueVerified` is a top-level field on `WireRawTweet`, NOT inside `legacy`

```go
// Correct:
td.IsBlueVerified = raw.IsBlueVerified  // field at root of result object

// Wrong:
td.IsBlueVerified = raw.Legacy.IsBlueVerified  // does not exist
```

Same applies to `WireRawUser` — `is_blue_verified` is top-level, not inside `legacy`.

Wire JSON:
```json
{
  "result": {
    "rest_id": "...",
    "is_blue_verified": true,    ← here, at the top level
    "legacy": {
      "full_text": "...",
      // is_blue_verified is NOT here
    }
  }
}
```

### #29: `UserTweets` does NOT use `withV2Timeline`

The `UserTweets` variables map must NOT include a `withV2Timeline` key. Adding it causes the API to return a different (incompatible) response shape.

Correct variables:
```go
vars := map[string]any{
    "userId":                                 userID,
    "count":                                  20,
    "includePromotedContent":                 false,
    "withQuickPromoteEligibilityTweetFields": true,
    "withVoice":                              true,
}
```

`UserTweets` also uses `fieldToggles: {"withArticlePlainText": false}` — omitting this causes some responses to include unexpected article data.

### #35: `SearchTimeline` uses POST but variables go in the URL query string

`SearchTimeline` is a POST request. But unlike other POST endpoints, the `variables` are passed in the URL query string, not in the POST body. The POST body contains only `features` and `queryId`.

```go
// Correct:
u.RawQuery = q2.Encode()  // variables JSON-encoded in URL query param
body := map[string]any{
    "features": buildSearchFeatures(),
    "queryId":  queryID,
}
raw, err := c.doPOSTJSON(ctx, u.String(), headers, body)

// Wrong:
body := map[string]any{
    "variables": vars,  // do NOT put variables in the POST body for Search
    "features":  buildSearchFeatures(),
    "queryId":   queryID,
}
```

### #36: `TweetDetail` tries GET first, then falls back to POST on 404

For each query ID, the `fetchTweetDetail` function:
1. Attempts GET with all params in the URL (`?variables=...&features=...&fieldToggles=...`)
2. If GET returns 404, attempts POST with the same data in the request body
3. If POST also returns 404, marks `had404=true` and continues to the next query ID
4. After exhausting all query IDs, if `had404` is true, calls `refreshQueryIDs` and retries once

This is a GET-primary, POST-fallback pattern. Other operations that return 404 trigger a refresh and retry but do not attempt both methods.

### #46: `DefaultNewsTabs` does NOT include `"trending"`

```go
// Correct:
var DefaultNewsTabs = []string{"forYou", "news", "sports", "entertainment"}

// Wrong: do not add "trending" to this list
// "trending" exists as a tab ID in GenericTimelineTabIDs but is NOT fetched by default
```

`GenericTimelineTabIDs` contains the `"trending"` key and its ID for callers who explicitly request it, but the default tab list intentionally excludes it.

### #50: `HomeTimeline` does NOT return `nextCursor` to callers

The `GetHomeTimeline` and `GetHomeLatestTimeline` methods consume the cursor internally (for multi-page fetching) and strip it from the returned `TweetResult`:

```go
result := paginateInline(ctx, *opts, 0, fetchFn)
result.NextCursor = ""  // correction #50
return result
```

Do not add `result.NextCursor` back. If you need paginated access to the home timeline, callers should use `MaxPages` or `Limit` in `FetchOptions`.

### #58: `GetTweet` response path is `data.tweetResult.result` (camelCase)

The JSON key is `tweetResult` (camelCase), not `tweet_result` (snake_case).

```go
var env struct {
    Data struct {
        TweetResult *types.WireTweetResult `json:"tweetResult"`  // camelCase
    } `json:"data"`
}
```

Compare with UserTweets which uses `data.user.result.timeline.timeline.instructions` (all snake_case).

### #60: `HomeTimeline` response path is `data.home.home_timeline_urt.instructions`

```go
var resp struct {
    Data struct {
        Home struct {
            HomeTimelineUrt struct {
                Instructions []types.WireTimelineInstruction `json:"instructions"`
            } `json:"home_timeline_urt"`  // snake_case
        } `json:"home"`
    } `json:"data"`
}
```

Note that `home_timeline_urt` is snake_case while `tweetResult` (correction #58) is camelCase — these are different operations with different response shapes.

### #63: `UserByScreenName` uses snake_case `screen_name` in variables

```go
vars := map[string]any{
    "screen_name": username,  // snake_case
}
```

Not `screenName` (camelCase). This is inconsistent with how some other operations use camelCase variable names, but has been verified against the actual API.

`UserByScreenName` also uses hardcoded-only query IDs — it never reads from the runtime cache. See `PerOperationFallbackIDs["UserByScreenName"]`.

### #83: `paginateCursor` does NOT stop on zero items

`paginateCursor` (used for TweetDetail replies and thread fetching) stops ONLY when:
- The next cursor is empty
- The next cursor equals the current cursor (no progress)

An empty page (zero tweets) does NOT terminate the loop. This differs from `paginateInline`, which does stop on zero items. The distinction is intentional: thread pages can be empty (e.g., all items filtered out as promoted content) while still having a valid cursor.

---

## Package Boundaries

### What goes where

| Package | Allowed to contain |
|---|---|
| `internal/types/` | Struct definitions only. No logic, no HTTP, no parsing. |
| `internal/parsing/` | Stateless pure functions. Input: raw bytes or wire types. Output: normalized types. No HTTP, no client methods. |
| `internal/client/` | All HTTP dispatch, query ID management, pagination loops, feature map construction. |
| `internal/auth/` | Credential resolution and browser cookie extraction. No client methods. |
| `internal/cli/` | Cobra command definitions, flag handling, output formatting dispatch. Calls `resolveClient()` from `shared.go`. |
| `internal/config/` | Config file loading and env var overlay only. |
| `internal/output/` | Terminal formatting only. No HTTP, no parsing. |
| `internal/testutil/` | Test helpers only. Must not be imported by non-test code. |
| `pkg/bird/` | Public API surface only. All methods delegate to `internal/client`. No business logic. |

### What MUST stay internal

The following types and functions are intentionally unexported and must NOT be moved to `pkg/bird/`:

- `client.httpError` — internal HTTP error type; callers receive `error`
- `client.inlinePageResult` — internal pagination callback type
- `client.graphqlURL`, `client.is404` — internal HTTP helpers
- `client.scrapeQueryIDs` — internal scraper; only the `Client.RefreshQueryIDs` public wrapper is exported
- All `buildXxxFeatures()` functions — implementation detail of the client
- `parsing.ExtractCursorFromInstructions` — used internally by client; do not expose directly in `pkg/bird`
- `auth.extractChrome`, `auth.extractSafari`, `auth.extractFirefox` — use the exported `auth.ExtractChromeCookies` etc. wrappers

---

## Naming Conventions

### Operation names must match Twitter GraphQL exactly

Operation names in code must match the URL path segment used in `https://x.com/i/api/graphql/<queryId>/<OperationName>` exactly. They are case-sensitive:

- `CreateTweet` not `create_tweet` or `CreateTweets`
- `FavoriteTweet` not `LikeTweet` or `FavoriteTweets`
- `UnfavoriteTweet` not `UnlikeTweet`
- `CreateRetweet` not `Retweet`
- `DeleteRetweet` not `Unretweet`
- `CreateBookmark` not `AddBookmark`
- `TweetDetail` not `TweetDetails` or `GetTweet`
- `SearchTimeline` not `Search`
- `UserTweets` not `GetUserTweets`
- `HomeTimeline` not `Home`
- `GenericTimelineById` not `GenericTimelineByID` (lowercase 'd')

### Go function names follow Go conventions

The Go function names (camelCase, exported or unexported) are independent of the operation names. `GetTweet` calls `TweetDetail`. `GetBookmarks` calls `Bookmarks`. This mapping is internal and callers need not know the GraphQL operation name.

### File naming

One file per domain in `internal/client/`. Files are named after their primary concern:
- `post.go` — tweet creation
- `engagement.go` — likes, retweets, bookmarks
- `follow.go` — follow/unfollow mutations
- `following.go` — following/followers read operations

---

## Mutex Discipline Rules

The `Client` struct has two mutex-guarded fields:

```go
type Client struct {
    queryIDMu    sync.RWMutex
    queryIDCache map[string]string
    queryIDRefreshAt time.Time

    userIDMu sync.RWMutex
    userID   string
    // ...
}
```

**Rule 1: Never access `c.userID` directly.** Always use `c.cachedUserID()`:

```go
// Correct:
id := c.cachedUserID()

// Wrong — data race:
id := c.userID
```

`cachedUserID()` acquires `userIDMu.RLock()` and releases it. The pattern exists because the user ID is resolved lazily and may be written by `ensureClientUserID()` concurrently.

**Rule 2: Never hold the write lock while making HTTP calls.** The pattern in `ensureClientUserID` is:

```go
// Slow path: resolve WITHOUT holding lock
user, err := c.getCurrentUser(ctx)  // HTTP call, no lock held
if err != nil {
    return err
}
// Write lock acquired only to store result
c.userIDMu.Lock()
if c.userID == "" {
    c.userID = user.ID
}
c.userIDMu.Unlock()
```

**Rule 3: `refreshQueryIDs` holds `queryIDMu.Lock()` only during the map write, not during the scrape.** The scraper runs without any lock held.

**Rule 4: Test code must inject `c.scraper`** to prevent the test from making real network calls during `refreshQueryIDs`:

```go
c.scraper = func(_ context.Context) map[string]string { return nil }
```

---

## Error Wrapping Rules

All errors must be wrapped with context using `fmt.Errorf("package: operation: %w", err)`.

The convention is `"package: operation: underlying error"`:

```go
// Correct:
return nil, fmt.Errorf("client: GetTweet: %w", err)
return nil, fmt.Errorf("auth: resolve: %w", err)
return nil, fmt.Errorf("parsing: ParseMyNew: %w", err)

// Wrong — loses context:
return nil, err

// Wrong — error strings must NOT start with a capital letter (staticcheck SA1006):
return nil, fmt.Errorf("Client GetTweet failed: %w", err)
return nil, errors.New("Not found")  // capital N
```

**Staticcheck rule**: Error strings passed to `fmt.Errorf` or `errors.New` must start with a lowercase letter and must not end with punctuation. The linter enforces this.

---

## Common Pitfalls

### 1. Adding new features to a feature map — always clone first

Feature maps are built fresh on each call by `buildXxxFeatures()`. Never store them in a variable and mutate them across calls. Inside a feature builder, always call `cloneFeatures()` before modifying:

```go
// Correct:
func buildMyFeatures() map[string]any {
    f := cloneFeatures(buildArticleFeatures())  // deep copy
    f["my_new_flag"] = true
    return applyFeatureOverrides("mySet", f)
}

// Wrong — mutates the base map, affecting all callers:
func buildMyFeatures() map[string]any {
    f := buildArticleFeatures()  // returns a new map, but...
    // Actually safe here since buildArticleFeatures creates a new map each call.
    // The pitfall is passing a map reference around and mutating it.
}

// Also wrong — mutating a received map:
func modifyFeatures(f map[string]any) {
    f["flag"] = true  // mutates caller's map
}
```

`cloneFeatures` does a shallow copy (string keys, scalar values) which is correct for feature flag maps.

### 2. Adding new query IDs — must add to ALL THREE maps

When adding a new operation, you must add its query ID to:
1. `FallbackQueryIDs` — always required
2. `BundledBaselineQueryIDs` — if you have a verified current ID
3. `PerOperationFallbackIDs` — if multiple IDs need to be tried in sequence

Missing any of these causes silent fallback to wrong IDs or `getQueryID` returning empty string.

The client does NOT panic or error on a missing query ID — it returns an empty string, which produces a malformed URL like `.../graphql//TweetDetail` that gets a 404.

### 3. Adding new CLI commands — two registration points

Every new command requires changes in two places:
1. The command constructor file: `internal/cli/mycommand.go` with `func newMyCommandCmd() *cobra.Command { ... }`
2. `internal/cli/root.go`: `root.AddCommand(newMyCommandCmd())`

Forgetting step 2 means the command is silently absent from the binary. It will not appear in `--help` and will not be executed.

### 4. Always use `cmd.Context()` in CLI commands

```go
// Correct — propagates cancellation from the CLI (Ctrl-C, timeout):
result, err := c.GetTweet(cmd.Context(), tweetID, nil)

// Wrong — ignores CLI cancellation:
result, err := c.GetTweet(context.Background(), tweetID, nil)
```

The cobra command's context is wired to OS signal handling.

### 5. Never remove wire type fields — only add

Wire types in `internal/types/wire.go` are deserialized from JSON. Removing a field is safe for deserialization (the JSON field is silently ignored) but may break existing callers that reference the field. Always add fields; never remove them. Mark deprecated fields with a comment if needed.

### 6. `UserByScreenName` never uses the runtime query ID cache

`PerOperationFallbackIDs["UserByScreenName"]` is a hardcoded list. The function `getUserByScreenNameQueryIDs()` returns this list directly and does NOT call `c.getQueryID("UserByScreenName")`. This means `refreshQueryIDs` has no effect on `UserByScreenName` — it will always use the hardcoded IDs.

This is intentional: the Twitter API has rotated UserByScreenName IDs frequently, so multiple hardcoded fallbacks are maintained.

### 7. The `scraper` field is for test injection only

`Client.scraper` is an unexported field of type `func(ctx context.Context) map[string]string`. When nil, `refreshQueryIDs` uses the real `scrapeQueryIDs` function. When non-nil (set by tests), it uses the injected function. This is the ONLY mechanism for preventing network calls from `RefreshQueryIDs` in tests.

Do not expose `scraper` publicly or use it for anything other than test injection.

### 8. `DeleteRetweet` takes the source tweet ID, not the retweet ID

Correction #3: The `DeleteRetweet` GraphQL mutation takes both `tweet_id` and `source_tweet_id`, both set to the original tweet's ID (not the retweet's ID):

```go
body := map[string]any{
    "variables": map[string]any{
        "tweet_id":        tweetID,         // source tweet
        "source_tweet_id": tweetID,         // same as tweet_id
    },
}
```

### 9. `Follow`/`Unfollow` uses REST-first, not GraphQL-first

The `Follow` and `Unfollow` methods try two REST endpoints before falling back to GraphQL:
1. `https://x.com/i/api/1.1/friendships/create.json` (REST)
2. `https://api.twitter.com/1.1/friendships/create.json` (REST)
3. `CreateFriendship` GraphQL (fallback only)

REST error codes: 160 = already following (treat as success), 162 = blocked (return error), 108 = not found (return error).

### 10. `CreateTweet` has a three-attempt chain

Correction #39, #72: The `createTweet` method:
1. POST to `graphql/<queryId>/CreateTweet`
2. If 404: refresh query IDs, POST to `graphql/<newQueryId>/CreateTweet`
3. If 404 again: POST to `https://x.com/i/api/graphql` (the base URL, no operation path)
4. If GraphQL error code 226: fall back to `statuses/update.json` (v1.1 REST)

Error code 226 means the GraphQL endpoint rejected the tweet (rate limit or policy), requiring the v1.1 fallback. Code #40: prefer `id_str` over numeric `id` in the v1.1 response.

---

## Query ID System

### Three levels of query IDs

```
Priority (highest first):
  1. Runtime cache (c.queryIDCache) — populated by refreshQueryIDs
  2. BundledBaselineQueryIDs        — compiled-in, verified IDs
  3. FallbackQueryIDs               — fallback constants

c.getQueryID("Op") returns the first non-empty ID from this chain.
```

### `PerOperationFallbackIDs` for multi-ID operations

For operations where multiple query IDs should be tried sequentially:

```go
var PerOperationFallbackIDs = map[string][]string{
    "TweetDetail": {"_NvJCn...", "97JF30...", "aFvUsJ..."},
    // ...
}
```

`c.getQueryIDs("TweetDetail")` returns the runtime/bundled primary first, then deduplicates against the fallback list. The client tries each ID in sequence, stopping on the first success.

### How `refreshQueryIDs` works

1. Calls the scraper (real or injected) to fetch fresh IDs from x.com JS bundles
2. Acquires `queryIDMu.Lock()`
3. Seeds the cache with `BundledBaselineQueryIDs` first (so bundled IDs are always present)
4. Overlays scraped IDs on top (scraped IDs win if non-empty)
5. Updates `queryIDRefreshAt`
6. Releases the lock

Errors from the scraper are silently ignored to preserve availability.

### When refreshes are triggered

Operations trigger `refreshQueryIDs` differently:

| Operation | Trigger |
|---|---|
| `CreateTweet` | HTTP 404 on attempt 1 |
| `TweetDetail` | HTTP 404 on all query IDs (`had404=true`) |
| `Search` | HTTP 404 or `GRAPHQL_VALIDATION_FAILED` |
| `HomeTimeline` | HTTP 404 or `query: unspecified` in GraphQL error |
| `Following`, `Followers`, `AboutAccountQuery` | HTTP 404 (via `withRefreshedQueryIDsOn404`) |

Refreshes are rate-limited by a `refreshed` boolean within each paginating fetch — at most one refresh per paginated operation.

---

## Feature Flag System

### Feature map inheritance hierarchy

```
buildArticleFeatures()          ← base set (~38 flags)
├── buildTweetDetailFeatures()  ← article + 3 extra flags
├── buildSearchFeatures()       ← article + rweb_video_timestamps_enabled
│   └── buildTimelineFeatures() ← search + 8 extra flags
│       ├── buildBookmarksFeatures() ← timeline + graphql_timeline_v2_bookmark_timeline
│       ├── buildLikesFeatures()     ← timeline (no extras)
│       ├── buildHomeTimelineFeatures() ← timeline (no extras)
│       └── buildExploreFeatures()   ← search + 5 grok flags
├── buildTweetCreateFeatures()  ← article but responsive_web_profile_redirect_enabled=false
```

Separate (not derived from article):
- `buildListsFeatures()` — own full map
- `buildUserTweetsFeatures()` — own full map
- `buildFollowingFeatures()` — own full map

### Runtime overrides via environment

Feature maps can be overridden at runtime without recompiling:

```bash
# Override a single flag globally
BIRD_FEATURES_JSON='{"global":{"rweb_video_timestamps_enabled":false}}' gobird search "golang"

# Override for a specific operation set
BIRD_FEATURES_JSON='{"sets":{"search":{"verified_phone_label_enabled":true}}}' gobird search "hello"

# Load from file (useful for complex overrides)
BIRD_FEATURES_PATH=/path/to/overrides.json gobird home
```

The set names are the first argument to `applyFeatureOverrides()` in `features.go`:
`"article"`, `"tweetDetail"`, `"search"`, `"tweetCreate"`, `"timeline"`, `"bookmarks"`, `"likes"`, `"homeTimeline"`, `"lists"`, `"userTweets"`, `"following"`, `"explore"`.

---

## Wire Type System

### Understanding `WireRawTweet`

The `WireRawTweet` type is used for both normal tweets AND `TweetWithVisibilityResults` wrappers. When the API returns a visibility wrapper, the actual tweet is in `raw.Tweet`. Always call `parsing.UnwrapTweetResult(raw)` before accessing tweet fields:

```go
raw = UnwrapTweetResult(raw)
// Now raw is guaranteed to be the actual tweet, not a wrapper
```

`UnwrapTweetResult` checks `raw.Tweet != nil` — it does NOT check `__typename`. This is intentional per correction §unwrap.

### `WireContent` field layout

```go
type WireContent struct {
    EntryType   string           // "TimelineTimelineItem", "TimelineTimelineCursor", etc.
    TypeName    string           // __typename
    CursorType  string           // "Top" or "Bottom" — for cursor entries (correction #7)
    Value       string           // cursor value string
    ItemContent *WireItemContent // tweet or user content (nil for cursor entries)
    Items       []WireItem       // module items (for conversation modules)
}
```

For cursor entries: `CursorType` and `Value` are populated; `ItemContent` is nil.
For tweet entries: `ItemContent` is populated; `CursorType` and `Value` are empty.

### Text extraction priority

For any tweet, text is extracted in this order (`ExtractTweetText`):
1. Article content (Draft.js `ContentState` → plain text, or `Title + PreviewText`)
2. Note tweet text (`note_tweet.note_tweet_results.result.text`)
3. `legacy.full_text`

This order matters: article tweets may have a `legacy.full_text` that is truncated or a stub. Always use `ExtractTweetText`, never read `legacy.full_text` directly.

---

## Pagination Patterns

### Pattern 1: `paginateCursor` (thread/replies)

Used by `paginateCursor` in `tweet_detail.go`. Stop conditions:
- `pageCursor == ""` OR `pageCursor == cursor` (cursor unchanged)

Does NOT stop on: zero items on the page.

Returns `NextCursor` in result when `maxPages` reached (caller can resume).

### Pattern 2: `paginateInline` (search, home, bookmarks, likes)

Used by `paginateInline` in `pagination.go`. Stop conditions (any one triggers stop):
1. `nextCursor == ""`
2. `nextCursor == cursor`
3. `len(result.tweets) == 0` (empty page)
4. `added == 0` (all items already seen/deduped)
5. `maxPages` reached
6. `len(accumulated) >= limit` (when `limit > 0`)

Page delay is applied BEFORE each page fetch, skipped for page 0 (correction #9).

### Pattern 3: UserTweets loop

`GetUserTweets` uses a manual loop (not `paginateInline`) with a hard limit of 10 pages regardless of settings.

### Cursor extraction

Always use `parsing.ExtractCursorFromInstructions(instructions)`. This scans for the first entry whose `content.cursorType == "Bottom"`. Do not write custom cursor extraction.

---

## How Tests Are Structured

### Mock scraper injection

Every `internal/client/` test that exercises code paths that could trigger `refreshQueryIDs` must inject a no-op scraper:

```go
c.scraper = func(_ context.Context) map[string]string { return nil }
```

Without this, the test would attempt real HTTP calls to x.com during the test.

### Pre-cancelled context for `RefreshQueryIDs` tests

When testing `RefreshQueryIDs` itself, pass a pre-cancelled context:

```go
ctx, cancel := context.WithCancel(context.Background())
cancel() // pre-cancel
c.RefreshQueryIDs(ctx) // safe: scraper's HTTP call sees cancelled context immediately
```

This ensures the test does not make real network calls even if `c.scraper` is nil.

### Test client construction pattern

The pattern in `internal/client/*_test.go`:

```go
func newTestClient(handler http.Handler) (*Client, *httptest.Server) {
    srv := httptest.NewServer(handler)
    c := New("tok", "ct0", &Options{
        HTTPClient: &http.Client{},
        QueryIDCache: map[string]string{
            "OperationName": "testQueryID",
        },
    })
    c.httpClient = &http.Client{Transport: redirectTransport(srv.URL)}
    c.scraper = func(_ context.Context) map[string]string { return nil }
    return c, srv
}
```

The `redirectTransport` rewrites the host portion of every request URL to the test server's URL while preserving the path and query string.

### Golden file tests

Used for complex serialization output. The `testutil.AssertGolden` function compares bytes. To regenerate:

```bash
go test ./... -update
```

Always verify the regenerated output manually before committing.

---

## What Staticcheck and Golangci-lint Will Flag

The linters enabled are: `errcheck`, `govet`, `staticcheck`, `unused`, `revive`.

### `errcheck` flags

- Not checking the return value of any function that returns `error`
- Exception: `.Close()` return values are suppressed (annotated `//nolint:errcheck`)
- Exception: test files are fully excluded from `errcheck`

Standard pattern for suppressing in non-test production code:
```go
defer rows.Close() //nolint:errcheck
```

### `staticcheck` flags

- Error strings beginning with a capital letter: `errors.New("Something")` → `errors.New("something")`
- Error strings ending with punctuation: `fmt.Errorf("operation failed.")` → `fmt.Errorf("operation failed")`
- Use of deprecated functions
- Unreachable code (correction #86: the post-loop return in `fetchWithRetry` is dead code but preserved with a comment)

### `revive` exported rule flags

- Exported functions, types, and methods must have doc comments
- The `disableStutteringCheck` argument is set, so `bird.BirdClient` is allowed (no stutter check)

### `unused` flags

- Any unexported function/type/variable/constant that is never referenced

### `govet` flags

- Printf format mismatches
- Struct field alignment issues (not enforced by this config, but `govet` checks correctness)
- Copies of sync types (`sync.Mutex`, `sync.RWMutex`) — never copy a struct that embeds a mutex

---

## Operation-Specific Notes

### `UserByScreenName`

- Uses hardcoded-only IDs from `PerOperationFallbackIDs["UserByScreenName"]`
- Never consults the runtime query ID cache
- Variable key is `screen_name` (snake_case, correction #63)
- Response path: `data.user.result` (GraphQL) or `data.user_result_by_screen_name.result` (AboutAccountQuery variant)

### `HomeTimeline` / `HomeLatestTimeline`

- Both return `TweetResult` with `NextCursor` stripped (correction #50)
- Variables include `latestControlAvailable: true` and `requestContext: "launch"` (correction #61)
- Uses GET, not POST (unlike Search which uses POST with variables in URL)
- Refresh trigger: "query: unspecified" GraphQL error (case-insensitive, allows whitespace, correction #77)

### `TweetDetail`

- GET first, then POST fallback per query ID (correction #36)
- Variables: `focalTweetId` (camelCase), `with_rux_injections: false`, `rankingMode: "Relevance"` (correction #28)
- Response for single tweet: `data.tweetResult.result` (camelCase `tweetResult`, correction #58)
- Response for thread: `data.threaded_conversation_with_injections_v2.instructions`
- Field toggles: `buildArticleFieldToggles()` (includes `withArticleRichContentState: true`)

### `SearchTimeline`

- POST request with variables in URL query string (correction #35)
- Refresh trigger: `GRAPHQL_VALIDATION_FAILED` in GraphQL error OR HTTP 400/422 with that body (correction #76)
- Product defaults to `"Latest"` when not specified
- Response path: `data.search_by_raw_query.search_timeline.timeline.instructions`

### `CreateTweet`

- Three-attempt chain: POST with queryId → 404 → refresh + retry → 404 → POST to base GraphQL URL (corrections #39, #72)
- Error code 226 triggers v1.1 `statuses/update.json` fallback (corrections #39, #40)
- Response path: `data.create_tweet.tweet_results.result.rest_id` (correction #38)
- Prefers `id_str` over numeric `id` in v1.1 fallback response (correction #40)

### `UserTweets`

- No `withV2Timeline` variable (correction #29)
- Field toggles: `{"withArticlePlainText": false}` (correction #13)
- Response path: `data.user.result.timeline.timeline.instructions`
- Hard page limit: 10 pages regardless of `MaxPages` setting

### `Bookmarks` / `BookmarkFolderTimeline`

- Uses `fetchWithRetry` for retry on 429/500/502/503/504 (correction #86)
- Up to 3 total attempts with exponential backoff
- Respects `Retry-After` header when present
- Feature flag: `graphql_timeline_v2_bookmark_timeline: true` (bookmarks-specific)

### `GenericTimelineById` (news/trending)

- Default tabs: `["forYou", "news", "sports", "entertainment"]` (no `"trending"`, correction #46)
- Tab IDs in `GenericTimelineTabIDs` map — base64-encoded opaque identifiers
- Uses explore feature set (`buildExploreFeatures()`)
