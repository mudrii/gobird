# API Client Reference

## Overview

`internal/client` is the implementation core of gobird. It is a single `Client` struct with methods split across domain-specific files. All Twitter/X API communication, query-ID resolution, feature-flag construction, retry logic, and pagination live here. The package is not exported; external callers use `pkg/bird` instead.

---

## Client Struct Fields

```go
type Client struct {
    authToken string   // Twitter auth_token cookie (40 hex chars)
    ct0       string   // Twitter ct0 CSRF token (32–160 alphanumeric chars)

    httpClient *http.Client  // Configured at construction; 30s default timeout

    clientUUID string  // UUID generated at New(); sent as x-client-uuid header
    deviceID   string  // UUID generated at New(); sent as x-twitter-client-deviceid header

    queryIDMu    sync.RWMutex      // Guards queryIDCache and queryIDRefreshAt
    queryIDCache map[string]string  // Runtime operation → queryID map
    queryIDRefreshAt time.Time      // When the cache was last written by refreshQueryIDs

    userIDMu sync.RWMutex  // Guards userID
    userID   string         // Cached numeric ID of the authenticated user (lazy)

    scraper func(ctx context.Context) map[string]string  // Override scrapeQueryIDs in tests
}
```

### Construction (`New`)

```go
func New(authToken, ct0 string, opts *Options) *Client
```

`Options.HTTPClient` replaces the default transport. `Options.QueryIDCache` seeds the runtime cache, useful in tests to inject known IDs. `Options.TimeoutMs` overrides the 30-second default only when `HTTPClient` is nil.

Both `clientUUID` and `deviceID` are freshly generated per-instance with `uuid.NewString()`. This mimics the browser's per-session random identifiers that X uses to correlate requests.

---

## Three-Tier Credential Resolution

Credential resolution is handled by `internal/auth.ResolveCredentials` and called from `pkg/bird.ResolveCredentials`. The client itself only receives already-resolved `authToken` and `ct0` strings.

### Tier 1: CLI Flags

```
--auth-token <value>   maps to ResolveOptions.FlagAuthToken
--ct0 <value>          maps to ResolveOptions.FlagCt0
```

Both flags must be provided together. If only one is set, this tier is skipped and the next tier is tried. On success, credentials are validated immediately by regex before returning.

### Tier 2: Environment Variables

```
AUTH_TOKEN          (preferred)
TWITTER_AUTH_TOKEN  (fallback)
CT0                 (preferred)
TWITTER_CT0         (fallback)
```

`firstNonEmpty` selects the first non-empty value among the preferred and fallback names. Both the token and the ct0 must resolve to non-empty values for this tier to win. Same regex validation applies.

### Tier 3: Browser Cookie Extraction

Tried only when tiers 1 and 2 both fail. The extraction order defaults to `["safari", "chrome", "firefox"]`. This can be overridden via:

- `--browser <name>` (single browser)
- `--cookie-source <name>` (one or more, repeatable)
- `ResolveOptions.CookieSources` (programmatic)

Each browser extractor reads the browser's cookie SQLite database for the `x.com` / `twitter.com` domain and extracts `auth_token` and `ct0` cookie values. `modernc.org/sqlite` is used for a pure-Go SQLite implementation with no system library requirement.

An optional timeout (`--cookie-timeout` / `BIRD_COOKIE_TIMEOUT_MS`) wraps extraction in a goroutine with a context deadline to prevent hanging on locked databases.

### Credential Validation

```
auth_token: /^[0-9a-f]{40}$/      (40 lowercase hex digits)
ct0:        /^[0-9a-zA-Z]{32,160}$/ (32–160 alphanumeric)
```

These constraints reject obviously malformed values before any network call is made.

---

## Query ID System

Twitter/X GraphQL endpoints use the URL pattern:

```
https://x.com/i/api/graphql/<queryId>/<OperationName>
```

The `queryId` component changes when X deploys new frontend bundles. gobird maintains a three-level fallback chain.

### Level 1: BundledBaselineQueryIDs

`constants.go` declares `BundledBaselineQueryIDs`, a map of 18 operations to the IDs that were current at the time of the last gobird release. These take priority over `FallbackQueryIDs` for the operations they cover.

### Level 2: FallbackQueryIDs

`constants.go` declares `FallbackQueryIDs` with 29 operations. These are older known-good IDs used when an operation has no entry in `BundledBaselineQueryIDs`.

### Level 3: Runtime Cache

`queryIDCache` (guarded by `queryIDMu`) starts empty and is populated by `refreshQueryIDs`. The runtime cache takes the highest priority: `getQueryID` checks it first.

```
Priority order in getQueryID:
  1. queryIDCache[operation]   (runtime, from scraping)
  2. BundledBaselineQueryIDs[operation]
  3. FallbackQueryIDs[operation]
```

### Per-Operation Fallback List

`PerOperationFallbackIDs` provides a slice of IDs to try per operation, in order. `getQueryIDs` builds this list by placing the current primary (from `getQueryID`) first, deduplicating against the hardcoded list, and returning all candidates. Operations not in this map use only their single primary.

### Special Case: UserByScreenName

`UserByScreenName` is declared only in `PerOperationFallbackIDs` (with three hardcoded IDs) and never appears in `BundledBaselineQueryIDs` or `FallbackQueryIDs`. Consequently, `getQueryID("UserByScreenName")` returns `""`, and `getQueryIDs("UserByScreenName")` returns the three hardcoded IDs directly. The runtime cache is never consulted for this operation.

### Scrape Refresh (`refreshQueryIDs`)

```go
func (c *Client) refreshQueryIDs(ctx context.Context)
```

Steps:
1. Fetches four X.com page URLs: `/home`, `/i/bookmarks`, `/explore`, `/settings/account`.
2. Finds all `https://abs.twimg.com/...js` script URLs in each page response.
3. Deduplicates script URLs across pages.
4. For each unseen script URL, fetches the JS bundle content.
5. For each operation in `FallbackQueryIDs`, applies a pre-compiled regex `([A-Za-z0-9_-]{20,})/<OperationName>\b` to extract the query ID from bundle text.
6. Writes results to `queryIDCache` under `queryIDMu.Lock()`.
7. Also seeds `BundledBaselineQueryIDs` entries into the cache so they are present for fast-path reads.

Errors (network failures, script fetch failures) are silently ignored. Whatever IDs were found are stored. The refresh timestamp is updated via `queryIDRefreshAt`.

The `c.scraper` field allows tests to inject a replacement function instead of hitting the real X.com.

---

## HTTP Request Lifecycle

### Base Headers

Every request starts with `baseHeaders`, which sets:

| Header | Value |
|---|---|
| `accept` | `*/*` |
| `accept-language` | `en-US,en;q=0.9` |
| `authorization` | `Bearer AAAAAAAAAAAAAAAAAAAAANRILgA...` (public bearer token) |
| `x-twitter-auth-type` | `OAuth2Session` |
| `x-twitter-active-user` | `yes` |
| `x-twitter-client-language` | `en` |
| `x-csrf-token` | `ct0` value |
| `x-client-uuid` | client UUID (per-instance random) |
| `x-twitter-client-deviceid` | device ID (per-instance random) |
| `x-client-transaction-id` | 16-byte random hex (per-request) |
| `x-twitter-client-user-id` | authenticated user's numeric ID (once resolved) |
| `cookie` | `auth_token=<token>; ct0=<ct0>` |
| `user-agent` | Chrome 131 macOS UA string |
| `origin` | `https://x.com` |
| `referer` | `https://x.com/` |

### Header Variants

- **JSON requests** (`getJSONHeaders`): base headers + `content-type: application/json`. Used for all GraphQL POST requests and most GET requests.
- **Upload requests** (`getUploadHeaders`): base headers only (no content-type override). Used for media upload multipart requests.

### CSRF Mechanism

The `ct0` cookie value is sent in two places simultaneously: as `Cookie: ... ct0=<value>` and as `X-Csrf-Token: <value>`. This is the standard X.com CSRF pattern. The `auth_token` cookie provides the session identity, while `ct0` prevents cross-site request forgery.

### Bearer Token

The bearer token (`AAAAAAAAAAAAAAAAAAAAANRILgA...`) is X.com's public client-side token embedded in the JavaScript bundle. It is not a secret. All authenticated requests require it in the `Authorization: Bearer` header.

### Transaction ID

`createTransactionID` generates a fresh 16-byte cryptographically random hex string for each request and sets it as `x-client-transaction-id`. This is X's per-request idempotency/tracing token.

---

## Retry and Refresh Strategies

### `fetchWithRetry` — Bookmarks and BookmarkFolderTimeline Only

```
Attempts: 3 total (attempt 0, 1, 2)
Retryable status codes: 429, 500, 502, 503, 504
Delay: exponential backoff starting at 500 ms with random jitter
  delay = (500 << attempt) ms + rand(0..499) ms
Retry-After header: used directly if present (overrides calculated delay)
Context cancellation: checked between attempts
Non-retryable errors: returned immediately (no retry)
```

This retry is applied only to GET requests for the Bookmarks and BookmarkFolderTimeline operations, which are more likely to receive rate-limit responses.

### `withRefreshedQueryIDsOn404` — Universal 404 Refresh

```go
func (c *Client) withRefreshedQueryIDsOn404(
    ctx context.Context,
    attempt func() attemptResult,
) (attemptResult, bool)
```

Used by: `fetchFollowPageWithRefresh` (Following/Followers), `GetUserAboutAccount`.

Pattern:
1. Call `attempt()`.
2. If success: return immediately.
3. If the error is a 404: call `refreshQueryIDs(ctx)`, then call `attempt()` again.
4. Return the second result and whether a refresh occurred.

The second `attempt` uses whatever query IDs are now in the cache (just populated by scraping). If the second attempt also fails with 404 and the operation has a REST fallback (Following/Followers), the REST endpoint is tried.

### Search `GRAPHQL_VALIDATION_FAILED` Refresh

Search uses a two-axis refresh strategy within its own loop (not `withRefreshedQueryIDsOn404`):

**Trigger conditions:**
- HTTP 400 or 422 with `GRAPHQL_VALIDATION_FAILED` in the response body.
- A GraphQL error whose `extensions.code == "GRAPHQL_VALIDATION_FAILED"`.
- A GraphQL error where path contains `"rawQuery"` AND message matches `/must be defined/i`.
- HTTP 404.

**Behavior:**
- On first trigger: calls `refreshQueryIDs`, updates the local `queryIDs` slice, sets `refreshed = true`, and retries from the beginning of the query-ID list.
- On subsequent triggers with `refreshed == true`: no further refresh; returns last error.

This is done independently for `Search` (single-page) and `GetAllSearchResults` (pagination loop).

### Home Timeline `"query: unspecified"` Refresh

HomeTimeline and HomeLatestTimeline check each page response body for GraphQL errors matching `(?i)query:\s*unspecified`. If found:

- The page is treated as a failure.
- If not yet refreshed: `refreshQueryIDs` is called, query IDs are updated, and the page is retried.
- If already refreshed: the error is returned.

Both `is404` and `isQueryUnspecifiedError` trigger the same refresh path.

---

## Feature Flag System

Each GraphQL operation requires a `features` parameter (a JSON object of boolean flags) that controls which fields the server includes in the response. gobird maintains 12 named feature-map builders.

### Inheritance Chain

```
buildArticleFeatures (base, ~37 flags)
    ├── buildTweetDetailFeatures (+ 3 article-related flags)
    ├── buildSearchFeatures (+ rweb_video_timestamps_enabled)
    │       ├── buildTimelineFeatures (+ 8 timeline-specific flags)
    │       │       ├── buildBookmarksFeatures (+ graphql_timeline_v2_bookmark_timeline)
    │       │       ├── buildLikesFeatures (alias)
    │       │       └── buildHomeTimelineFeatures (alias)
    │       └── buildExploreFeatures (+ 5 Grok/explore flags)
    ├── buildTweetCreateFeatures (responsive_web_profile_redirect_enabled=false)
    ├── buildListsFeatures (independent map, similar base, slightly different values)
    ├── buildUserTweetsFeatures (independent map, rweb_video_screen_enabled=false)
    ├── buildFollowingFeatures (independent map, premium_content_api_read_enabled=true)
    └── buildUserByScreenNameFeatures (delegates to buildArticleFeatures)
```

### env Override

`BIRD_FEATURES_JSON` or `BIRD_FEATURES_PATH` can inject override values. The payload is parsed once via `sync.Once` and stored in `featureOverrides`. Its structure:

```json
{
  "global": { "some_flag": true },
  "sets": {
    "search": { "another_flag": false }
  }
}
```

`global` overrides apply to every feature set. `sets.<name>` overrides apply only to the named set. The set names match the `applyFeatureOverrides` call in each builder:

| Set name | Operations |
|---|---|
| `article` | Base set, article operations |
| `tweetDetail` | TweetDetail |
| `search` | SearchTimeline |
| `tweetCreate` | CreateTweet |
| `timeline` | Bookmarks, Home, Likes (base) |
| `bookmarks` | Bookmarks, BookmarkFolderTimeline |
| `likes` | Likes |
| `homeTimeline` | HomeTimeline, HomeLatestTimeline |
| `lists` | ListOwnerships, ListMemberships, ListLatestTweetsTimeline |
| `userTweets` | UserTweets, UserArticlesTweets |
| `following` | Following, Followers |
| `explore` | GenericTimelineById |

### Field Toggles

Separate from feature flags, some operations also send a `fieldToggles` parameter:

- **Article operations**: `withPayments`, `withAuxiliaryUserLabels`, `withArticleRichContentState`, `withArticlePlainText`, `withGrokAnalyze`, `withDisallowedReplyControls`
- **UserTweets**: `withArticlePlainText: false`
- **UserByScreenName**: `withAuxiliaryUserLabels: false`

---

## Pagination Patterns

gobird uses two distinct pagination patterns.

### Pattern 1: Manual Page Loop

Used by: `GetFollowing`, `GetFollowers`, `GetOwnedLists`, `GetListMemberships`, `GetListTimeline`, `GetUserTweets`.

Each of these implements its own for-loop that:
1. Fetches a page via a `fetchXxxPage` function.
2. Deduplicates items by ID into an accumulator.
3. Checks stop conditions manually.
4. Applies a page delay between pages (skipping page 0).
5. Returns `*UserResult` / `*ListResult` / `*TweetResult` with the accumulated items.

Stop conditions per operation:
- `nextCursor == ""`: no more pages.
- `nextCursor == cursor`: cursor did not advance.
- `pagesFetched >= maxPages`: ceiling reached (default 10 for follow ops).

### Pattern 2: `paginateInline` Helper

Used by: `Search`, `GetAllSearchResults`, `GetHomeTimeline`, `GetHomeLatestTimeline`, `GetBookmarks`, `GetBookmarkFolderTimeline`, `GetLikes`.

```go
func paginateInline(
    ctx context.Context,
    opts types.FetchOptions,
    defaultDelayMs int,
    fetch fetchPageFn,
) types.TweetResult
```

The caller provides a `fetch(ctx, cursor) inlinePageResult` callback. `paginateInline` manages the loop, accumulator, dedup map, and all stop conditions:

| Stop condition | Description |
|---|---|
| `nextCursor == ""` | No cursor in response |
| `nextCursor == cursor` | Cursor did not advance |
| `len(result.tweets) == 0` | Empty page |
| `added == 0` | All items on page were already seen (full dedup) |
| `maxPages > 0 && page >= maxPages` | Page count ceiling |
| `limit > 0 && len(accumulated) >= limit` | Item count ceiling |

Page delay is applied before the fetch and is skipped for page 0. If `opts.PageDelayMs == 0`, the `defaultDelayMs` argument is used. Callers pass `defaultDelayMs = 0` to suppress delay entirely (e.g., home timeline), or a positive value to document the intended default (e.g., 1000 ms for search).

If the fetch fails after some items have been accumulated, `paginateInline` returns a partial success (`Success: false`, `Error: err`, `Items: accumulated`, `NextCursor: last successful cursor`). If the fetch fails on the very first page with no items, it returns `Success: false` with no items.

---

## All 29 GraphQL Operations

| Operation | File | Method | HTTP | Notes |
|---|---|---|---|---|
| `CreateTweet` | `post.go` | `Tweet`, `Reply` | POST JSON | Variables: `tweet_text`, optional `reply.in_reply_to_tweet_id`, optional `media.media_ids` |
| `CreateRetweet` | `engagement.go` | `Retweet` | POST JSON | Variables: `tweet_id`, `tweet_id` (retweet target) |
| `DeleteRetweet` | `engagement.go` | `Unretweet` | POST JSON | Variables: `source_tweet_id` |
| `FavoriteTweet` | `engagement.go` | `Like` | POST JSON | Variables: `tweet_id` |
| `UnfavoriteTweet` | `engagement.go` | `Unlike` | POST JSON | Variables: `tweet_id` |
| `CreateBookmark` | `engagement.go` | `Bookmark` | POST JSON | Variables: `tweet_id` |
| `DeleteBookmark` | `bookmarks.go` | `Unbookmark` | POST JSON | Variables: `tweet_id`; custom referer header |
| `CreateFriendship` | `follow.go` | `Follow` | POST JSON | Variables: `userId` |
| `DestroyFriendship` | `follow.go` | `Unfollow` | POST JSON | Variables: `userId` |
| `TweetDetail` | `tweet_detail.go` | `GetTweet`, `GetReplies`, `GetThread` | GET | Variables: `focalTweetId`, `count`, `referrer`; fieldToggles |
| `SearchTimeline` | `search.go` | `Search`, `GetAllSearchResults` | POST JSON (vars in URL) | Variables in query string, features+queryId in body |
| `UserArticlesTweets` | `user_tweets.go` | internal | GET | Variables: `userId`, `count`, optional `cursor` |
| `UserTweets` | `user_tweets.go` | `GetUserTweets`, `GetUserTweetsPaged` | GET | Variables: `userId`, `count`, optional `cursor`, optional `includePromotedContent` |
| `Bookmarks` | `bookmarks.go` | `GetBookmarks` | GET + fetchWithRetry | Variables: `count`; features+queryId in URL |
| `Following` | `following.go` | `GetFollowing` | GET | Variables: `userId`, `count`, `includePromotedContent: false` |
| `Followers` | `following.go` | `GetFollowers` | GET | Variables: `userId`, `count`, `includePromotedContent: false` |
| `Likes` | `timelines.go` | `GetLikes` | GET | Variables: `userId`, `count` |
| `BookmarkFolderTimeline` | `bookmarks.go` | `GetBookmarkFolderTimeline` | GET + fetchWithRetry | Variables: `bookmark_collection_id`, `count` |
| `ListOwnerships` | `lists.go` | `GetOwnedLists` | GET | Variables: `id` (user ID), `count` |
| `ListMemberships` | `lists.go` | `GetListMemberships` | GET | Variables: `id` (user ID), `count` |
| `ListLatestTweetsTimeline` | `lists.go` | `GetListTimeline` | GET | Variables: `listId`, `count` |
| `ListByRestId` | `lists.go` | internal lookup | GET | Variables: `listId` |
| `HomeTimeline` | `home.go` | `GetHomeTimeline` | GET | Variables: `count`, `includePromotedContent`, `latestControlAvailable`, `requestContext`, `withCommunity`; never returns nextCursor to caller |
| `HomeLatestTimeline` | `home.go` | `GetHomeLatestTimeline` | GET | Same variables; chronological feed |
| `ExploreSidebar` | `news.go` | internal | GET | Returns sidebar trending data |
| `ExplorePage` | `news.go` | internal | GET | Returns explore page content |
| `GenericTimelineById` | `timelines.go` | `GetNews` (news tabs) | GET | Variables: `timelineId` from `GenericTimelineTabIDs`; one call per tab |
| `TrendHistory` | `news.go` | internal | GET | Trend history data |
| `AboutAccountQuery` | `user_lookup.go` | `GetUserAboutAccount` | GET | Variables: `screenName`; uses `withRefreshedQueryIDsOn404` |
| `UserByScreenName` | `user_lookup.go` | `GetUserIDByUsername` | GET | Hardcoded IDs only (never runtime cache); variables: `screen_name`, `withSafetyModeUserFields` |

### REST Endpoints (fallback or primary)

| Endpoint | Purpose | Trigger |
|---|---|---|
| `POST /i/api/1.1/statuses/update.json` | Legacy tweet creation fallback | Not used by default |
| `GET /i/api/1.1/friends/list.json` | Following REST fallback | GraphQL 404 after refresh |
| `GET /i/api/1.1/followers/list.json` | Followers REST fallback | GraphQL 404 after refresh |
| `GET /i/api/1.1/users/show.json` | User lookup REST fallback | All GraphQL query IDs exhausted |
| `POST https://upload.twitter.com/i/media/upload.json` | Media upload (init/append/finalize) | `UploadMedia` always |
| `POST /i/api/1.1/media/metadata/create.json` | Alt text for uploaded media | After media upload if altText provided |
| `POST /i/api/1.1/friendships/create.json` | Follow REST | Used by `Follow` |
| `POST /i/api/1.1/friendships/destroy.json` | Unfollow REST | Used by `Unfollow` |
| `GET /i/api/account/settings.json` or `api.twitter.com` mirror | Current user settings | `GetCurrentUser` |
| `GET /i/api/account/verify_credentials.json` or `api.twitter.com` mirror | Credential verification | `GetCurrentUser` |

---

## Query ID Resolution Flow (Complete)

```
getQueryID(operation)
  │
  ├── queryIDCache[operation] != ""  →  return cached
  │
  ├── BundledBaselineQueryIDs[operation] exists  →  return bundled
  │
  └── FallbackQueryIDs[operation]  →  return fallback (may be "")

getQueryIDs(operation)
  │
  ├── primary = getQueryID(operation)
  │
  ├── PerOperationFallbackIDs[operation] exists?
  │     YES: deduplicate [primary] + fallback list  →  return slice
  │     NO:  return [primary] if non-empty, else nil

refreshQueryIDs(ctx)
  ├── Run scraper (real or injected test stub)
  ├── Lock queryIDMu.Lock()
  ├── Seed BundledBaselineQueryIDs into cache
  ├── Merge scraped IDs (non-empty only) into cache
  └── Record queryIDRefreshAt = time.Now()
```
