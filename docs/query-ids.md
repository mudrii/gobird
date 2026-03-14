# Query ID System

This document explains what query IDs are, how gobird resolves them, and how to maintain them when Twitter/X rotates their values.

---

## What Query IDs Are

Twitter/X's GraphQL API does not use a public schema. Instead of accepting arbitrary GraphQL queries, the server only accepts pre-compiled query hashes — opaque identifiers that correspond to specific GraphQL operations baked into the X.com web client JavaScript bundles.

A query ID looks like this: `TAJw1rBsjAtdNgTdlo2oeg`

It is a 22-character base64url-encoded string. The server maps this identifier to a specific operation definition server-side. The client cannot construct its own operations; it must use a known ID.

Query IDs are not stable. Twitter/X periodically rotates them when deploying new client bundles. An ID that worked yesterday may return HTTP 404 or `GRAPHQL_VALIDATION_FAILED` today.

---

## The Three-Tier Lookup

When gobird needs a query ID for an operation, it checks three sources in order:

### Tier 1: Runtime Cache

```go
c.queryIDMu.RLock()
id, ok := c.queryIDCache[operation]
c.queryIDMu.RUnlock()
if ok && id != "" {
    return id
}
```

The runtime cache is populated by `refreshQueryIDs`, which scrapes fresh IDs from X.com's JavaScript bundles. It starts empty and is only populated after the first successful scrape. Once populated, runtime cache takes highest priority.

### Tier 2: BundledBaselineQueryIDs

```go
if id, ok := BundledBaselineQueryIDs[operation]; ok {
    return id
}
```

A curated map of 18 operations with known-good IDs compiled into the binary. These are more recently verified than the fallback IDs. Operations in this map skip the fallback tier entirely when the cache is cold.

### Tier 3: FallbackQueryIDs

```go
return FallbackQueryIDs[operation]
```

A map of 29 operations with last-known-good IDs. These are the ultimate fallback when neither the runtime cache nor the bundled baseline has a value. They may be stale after a Twitter/X deployment.

### getQueryIDs (Multiple IDs for Retry)

When making a request, gobird calls `getQueryIDs` (plural), which returns all IDs to try in order:

1. The primary ID from `getQueryID` (cache → bundled baseline → fallback)
2. Additional IDs from `PerOperationFallbackIDs` for the operation

This allows operations to try multiple IDs before triggering a full refresh.

---

## Full Table of FallbackQueryIDs (29 entries)

These are the hardcoded last-resort IDs. Update them when operations begin returning 404 consistently even after a refresh.

| Operation | Fallback Query ID |
|-----------|------------------|
| `CreateTweet` | `TAJw1rBsjAtdNgTdlo2oeg` |
| `CreateRetweet` | `ojPdsZsimiJrUGLR1sjUtA` |
| `DeleteRetweet` | `iQtK4dl5hBmXewYZuEOKVw` |
| `CreateFriendship` | `8h9JVdV8dlSyqyRDJEPCsA` |
| `DestroyFriendship` | `ppXWuagMNXgvzx6WoXBW0Q` |
| `FavoriteTweet` | `lI07N6Otwv1PhnEgXILM7A` |
| `UnfavoriteTweet` | `ZYKSe-w7KEslx3JhSIk5LA` |
| `CreateBookmark` | `aoDbu3RHznuiSkQ9aNM67Q` |
| `DeleteBookmark` | `Wlmlj2-xzyS1GN3a6cj-mQ` |
| `TweetDetail` | `97JF30KziU00483E_8elBA` |
| `SearchTimeline` | `M1jEez78PEfVfbQLvlWMvQ` |
| `UserArticlesTweets` | `8zBy9h4L90aDL02RsBcCFg` |
| `UserTweets` | `Wms1GvIiHXAPBaCr9KblaA` |
| `Bookmarks` | `RV1g3b8n_SGOHwkqKYSCFw` |
| `Following` | `BEkNpEt5pNETESoqMsTEGA` |
| `Followers` | `kuFUYP9eV1FPoEy4N-pi7w` |
| `Likes` | `JR2gceKucIKcVNB_9JkhsA` |
| `BookmarkFolderTimeline` | `KJIQpsvxrTfRIlbaRIySHQ` |
| `ListOwnerships` | `wQcOSjSQ8NtgxIwvYl1lMg` |
| `ListMemberships` | `BlEXXdARdSeL_0KyKHHvvg` |
| `ListLatestTweetsTimeline` | `2TemLyqrMpTeAmysdbnVqw` |
| `ListByRestId` | `wXzyA5vM_aVkBL9G8Vp3kw` |
| `HomeTimeline` | `edseUwk9sP5Phz__9TIRnA` |
| `HomeLatestTimeline` | `iOEZpOdfekFsxSlPQCQtPg` |
| `ExploreSidebar` | `lpSN4M6qpimkF4nRFPE3nQ` |
| `ExplorePage` | `kheAINB_4pzRDqkzG3K-ng` |
| `GenericTimelineById` | `uGSr7alSjR9v6QJAIaqSKQ` |
| `TrendHistory` | `Sj4T-jSB9pr0Mxtsc1UKZQ` |
| `AboutAccountQuery` | `zs_jFPFT78rBpXv9Z3U2YQ` |

There must be exactly 29 entries. The test suite in `constants_test.go` enforces this count.

---

## Full Table of BundledBaselineQueryIDs (18 entries)

These IDs are more recently verified than the fallbacks. Operations listed here use these IDs when the runtime cache is cold.

| Operation | Bundled Baseline Query ID |
|-----------|--------------------------|
| `CreateTweet` | `nmdAQXJDxw6-0KKF2on7eA` |
| `CreateRetweet` | `LFho5rIi4xcKO90p9jwG7A` |
| `CreateFriendship` | `8h9JVdV8dlSyqyRDJEPCsA` |
| `DestroyFriendship` | `ppXWuagMNXgvzx6WoXBW0Q` |
| `FavoriteTweet` | `lI07N6Otwv1PhnEgXILM7A` |
| `DeleteBookmark` | `Wlmlj2-xzyS1GN3a6cj-mQ` |
| `TweetDetail` | `_NvJCnIjOW__EP5-RF197A` |
| `SearchTimeline` | `6AAys3t42mosm_yTI_QENg` |
| `Bookmarks` | `RV1g3b8n_SGOHwkqKYSCFw` |
| `BookmarkFolderTimeline` | `KJIQpsvxrTfRIlbaRIySHQ` |
| `Following` | `mWYeougg_ocJS2Vr1Vt28w` |
| `Followers` | `SFYY3WsgwjlXSLlfnEUE4A` |
| `Likes` | `ETJflBunfqNa1uE1mBPCaw` |
| `ExploreSidebar` | `lpSN4M6qpimkF4nRFPE3nQ` |
| `ExplorePage` | `kheAINB_4pzRDqkzG3K-ng` |
| `GenericTimelineById` | `uGSr7alSjR9v6QJAIaqSKQ` |
| `TrendHistory` | `Sj4T-jSB9pr0Mxtsc1UKZQ` |
| `AboutAccountQuery` | `zs_jFPFT78rBpXv9Z3U2YQ` |

Operations NOT in this table (e.g. `HomeTimeline`, `UserTweets`, `ListOwnerships`) always fall through to `FallbackQueryIDs` when the runtime cache is cold.

---

## PerOperationFallbackIDs Per-Operation Chains

These are the ordered lists of all IDs to try per operation. The first element is derived from the bundled baseline or fallback; subsequent elements are additional hardcoded fallbacks.

```go
"TweetDetail":              {"_NvJCnIjOW__EP5-RF197A", "97JF30KziU00483E_8elBA", "aFvUsJm2c-oDkJV75blV6g"},
"SearchTimeline":           {"6AAys3t42mosm_yTI_QENg", "M1jEez78PEfVfbQLvlWMvQ", "5h0kNbk3ii97rmfY6CdgAA", "Tp1sewRU1AsZpBWhqCZicQ"},
"HomeTimeline":             {"edseUwk9sP5Phz__9TIRnA"},
"HomeLatestTimeline":       {"iOEZpOdfekFsxSlPQCQtPg"},
"Bookmarks":                {"RV1g3b8n_SGOHwkqKYSCFw", "tmd4ifV8RHltzn8ymGg1aw"},
"BookmarkFolderTimeline":   {"KJIQpsvxrTfRIlbaRIySHQ"},
"Likes":                    {"ETJflBunfqNa1uE1mBPCaw", "JR2gceKucIKcVNB_9JkhsA"},
"UserTweets":               {"Wms1GvIiHXAPBaCr9KblaA"},
"Following":                {"mWYeougg_ocJS2Vr1Vt28w", "BEkNpEt5pNETESoqMsTEGA"},
"Followers":                {"SFYY3WsgwjlXSLlfnEUE4A", "kuFUYP9eV1FPoEy4N-pi7w"},
"CreateFriendship":         {"8h9JVdV8dlSyqyRDJEPCsA", "OPwKc1HXnBT_bWXfAlo-9g"},
"DestroyFriendship":        {"ppXWuagMNXgvzx6WoXBW0Q", "8h9JVdV8dlSyqyRDJEPCsA"},
"ListOwnerships":           {"wQcOSjSQ8NtgxIwvYl1lMg"},
"ListMemberships":          {"BlEXXdARdSeL_0KyKHHvvg"},
"ListLatestTweetsTimeline": {"2TemLyqrMpTeAmysdbnVqw"},
"AboutAccountQuery":        {"zs_jFPFT78rBpXv9Z3U2YQ"},
"UserByScreenName":         {"xc8f1g7BYqr6VTzTbvNlGw", "qW5u-DAuXpMEG0zA1F7UGQ", "sLVLhk0bGj3MVFEKTdax1w"},
```

`UserByScreenName` is hardcoded only — it never consults the runtime cache (correction #5). Its IDs appear only in `PerOperationFallbackIDs`, not in `FallbackQueryIDs` or `BundledBaselineQueryIDs`.

`getQueryIDs` deduplicates the combined list: primary ID first, then any additional IDs from `PerOperationFallbackIDs` that are not already included.

---

## How scrapeQueryIDs Works

`scrapeQueryIDs` (`client/query_ids.go`) extracts fresh IDs from Twitter/X's JavaScript bundles.

### Algorithm

1. **Fetch HTML pages**: GET four pages using the `UserAgent` header (no auth cookies):
   - `https://x.com/home`
   - `https://x.com/i/bookmarks`
   - `https://x.com/explore`
   - `https://x.com/settings/account`

2. **Extract script URLs**: Find all `https://abs.twimg.com/...js` URLs in the HTML using:
   ```
   https://abs\.twimg\.com/[^"' )]+\.js
   ```

3. **Fetch JS bundles**: For each unique script URL (deduped across all pages), GET the bundle contents.

4. **Apply per-operation regexes**: For each operation name in `FallbackQueryIDs`, scan the bundle for:
   ```
   ([A-Za-z0-9_-]{20,})/<operationName>\b
   ```
   The first capture group is the query ID. The `{20,}` constraint filters out short strings that are not valid IDs.

5. **Stop early**: Once an operation has been found, subsequent bundles skip it (`if _, ok := found[operation]; ok { continue }`).

6. **Return the map**: Any operation not found in the bundles remains absent from the returned map; the caller falls through to bundled baseline or fallback.

### Constraints

- 15-second HTTP timeout per request
- No auth credentials sent (unauthenticated scrape)
- Errors per page/bundle are silently ignored
- Scripts are deduped across pages

---

## When Refresh Is Triggered Per Operation

| Operation | Trigger |
|-----------|---------|
| HomeTimeline / HomeLatestTimeline | HTTP 404 or GraphQL message matching `query:\s*unspecified` |
| SearchTimeline | HTTP 404, HTTP 400/422 with `GRAPHQL_VALIDATION_FAILED` in body, or GraphQL `GRAPHQL_VALIDATION_FAILED` code |
| Likes | HTTP 404; or after all IDs exhausted and `Query: Unspecified` in accumulated errors |
| Bookmarks | HTTP 404 |
| BookmarkFolderTimeline | HTTP 404 |
| CreateTweet | HTTP 404 on attempt 1 (triggers refresh before attempt 2) |
| TweetDetail | HTTP 404 on both GET and POST for all query IDs |
| Following / Followers | HTTP 404 (via `withRefreshedQueryIDsOn404`) |
| AboutAccountQuery | HTTP 404 (via `withRefreshedQueryIDsOn404`) |
| GenericTimelineById | HTTP 404 |
| UserByScreenName | Never — uses hardcoded IDs only |

A refresh is only performed once per logical request across all operations, enforced by a `refreshed bool` flag. This prevents infinite refresh loops.

---

## How to Update Query IDs When They Change

When an operation consistently returns 404 or `GRAPHQL_VALIDATION_FAILED` even after automatic refresh, the hardcoded IDs need updating.

### Step 1: Find the New ID

Run gobird's `gobird query-ids` command (or call `RefreshQueryIDs` manually) to attempt a live scrape. If successful, the new ID is cached at runtime.

Alternatively, open X.com in a browser with Developer Tools open, filter network traffic to `graphql`, and look for the failing operation name in request URLs. Copy the 22-character segment before the operation name.

### Step 2: Update BundledBaselineQueryIDs

If the operation is in `BundledBaselineQueryIDs`, update its value in `constants.go`:

```go
var BundledBaselineQueryIDs = map[string]string{
    "TweetDetail": "<new-id-here>",
    // ...
}
```

### Step 3: Update PerOperationFallbackIDs

Prepend the new ID to the operation's fallback chain in `PerOperationFallbackIDs`. Keep old IDs as fallbacks:

```go
"TweetDetail": {"<new-id>", "_NvJCnIjOW__EP5-RF197A", "97JF30KziU00483E_8elBA", "aFvUsJm2c-oDkJV75blV6g"},
```

### Step 4: Update FallbackQueryIDs (if needed)

For operations that were previously working via `FallbackQueryIDs` but are now broken, update the value there too. Maintain the 29-entry count.

### Step 5: Verify

Run the test suite. `constants_test.go` validates the 29-entry count and that all operations have at least one ID.

---

## The 404-Refresh Cycle Explained

The standard flow when a query ID is stale:

```
Request with current ID
    → HTTP 404
    → refreshQueryIDs()
        → scrapeQueryIDs() fetches X.com JS bundles
        → Updates runtime cache with fresh IDs
        → Applies BundledBaselineQueryIDs as floor
    → Retry with new ID from cache
        → Success  → return result
        → HTTP 404 → try additional IDs from PerOperationFallbackIDs
                   → all fail → return error to caller
```

Key properties:
- Refresh is performed at most once per top-level operation call
- The runtime cache is shared across all goroutines (mutex-protected)
- After refresh, the cache contains the union of scraped IDs and `BundledBaselineQueryIDs`
- `FallbackQueryIDs` are NOT written into the cache by refresh — they remain as the last-resort tier only

---

## Testing with Mock Scrapers

The `Client` struct has a `scraper` field of type `func(context.Context) map[string]string`. When non-nil, it replaces `scrapeQueryIDs` during `refreshQueryIDs`.

Tests inject a mock scraper to control which IDs are returned without making network requests:

```go
client := &Client{
    scraper: func(ctx context.Context) map[string]string {
        return map[string]string{
            "TweetDetail": "mock-query-id-here",
        }
    },
}
```

This allows tests to verify:
- The 404-refresh cycle triggers correctly
- The new ID from the scraper is used on the second attempt
- Operations not returned by the scraper fall back to `BundledBaselineQueryIDs`
- The refresh is only called once per request

See `internal/client/query_ids_test.go` for examples.
