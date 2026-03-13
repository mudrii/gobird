# API Corrections Reference

This document catalogues all known Twitter/X API behaviours that differ from naive expectations. Each entry explains what you would expect, what actually happens, how gobird handles it, and what breaks if you get it wrong.

Anyone modifying API client code must read this document first.

---

### #1 — bookmark_collection_timeline Response Path

**Category**: Response Path
**Operations**: BookmarkFolderTimeline

**Naive expectation**: Bookmark folder timeline responses use the same `bookmark_timeline_v2` path as regular bookmarks.

**Actual behaviour**: BookmarkFolderTimeline returns data under `data.bookmark_collection_timeline.timeline.instructions`, not `data.bookmark_timeline_v2`.

**Implementation**: `timelines.go` `bookmarkFolderPage` unmarshals into a struct with path `Data.BookmarkCollectionTimeline.Timeline.Instructions`.

**Why it matters**: Using the wrong path yields an empty instruction slice; no tweets are returned even on success.

---

### #2 — AboutAccount camelCase screenName

**Category**: Request Format
**Operations**: AboutAccountQuery

**Naive expectation**: All GraphQL variable names for user lookup use snake_case (`screen_name`), consistent with REST.

**Actual behaviour**: AboutAccountQuery variables use camelCase: `screenName` (not `screen_name`).

**Implementation**: `user_lookup.go` `GetUserAboutAccount` builds `vars := map[string]any{"screenName": username}`.

**Why it matters**: Sending `screen_name` causes the query to receive a null variable, returning no result.

---

### #5 — UserUnavailable Stops Immediately

**Category**: Response Path
**Operations**: UserByScreenName

**Naive expectation**: When one query ID fails for UserByScreenName, the loop moves to the next. An unavailable user is just another failure, so all IDs are tried.

**Actual behaviour**: A `UserUnavailable` `__typename` in the response is a definitive server signal that the account is suspended, deactivated, or otherwise blocked. Retrying with a different query ID will return the same result.

**Implementation**: `user_lookup.go` `parseUserByScreenNameResponse` checks `raw.TypeName == "UserUnavailable"` and returns `unavailable=true`. The caller in `fetchUserByScreenName` returns `fmt.Errorf("user %q is unavailable", username)` immediately without trying remaining query IDs. Also, `UserByScreenName` never uses the runtime query-ID cache — it reads only from `PerOperationFallbackIDs["UserByScreenName"]`.

**Why it matters**: Without early exit, unavailable-user lookups waste time trying every query ID. More importantly, the runtime cache is not consulted; always using hardcoded IDs prevents stale cached IDs from poisoning this operation.

---

### #6 — getCurrentUser API URLs

**Category**: Auth / Request Format
**Operations**: getCurrentUser (internal)

**Naive expectation**: There is a single GraphQL endpoint to resolve the authenticated user's identity.

**Actual behaviour**: No GraphQL endpoint is used. gobird tries four REST endpoints in order:
1. `https://x.com/i/api/account/settings.json`
2. `https://api.twitter.com/1.1/account/settings.json`
3. `https://x.com/i/api/account/verify_credentials.json?skip_status=true&include_entities=false`
4. `https://api.twitter.com/1.1/account/verify_credentials.json?skip_status=true&include_entities=false`

If all fail, it falls back to scraping the HTML of `https://x.com/settings/account` and `https://twitter.com/settings/account` with regex.

**Implementation**: `users.go` `getCurrentUser` iterates `apiURLs` then `htmlPages`. `tryGetCurrentUserFromAPI` uses a flexible key scanner across multiple JSON paths. `tryGetCurrentUserFromHTML` uses regexes `htmlUserIDRe`, `htmlScreenRe`, `htmlNameRe`.

**Why it matters**: Using a GraphQL user endpoint to resolve the authenticated user requires a user ID you do not yet have. The settings/credentials REST endpoints return the current session's user without needing an ID. The HTML fallback handles cases where the API tier is unavailable.

---

### #7 — cursorType Location

**Category**: Response Path
**Operations**: All timeline operations

**Naive expectation**: The pagination cursor value is found in `entry.content.entryType` or within module items.

**Actual behaviour**: The pagination cursor is at `entry.content.cursorType == "Bottom"`, and `entry.content.value` holds the cursor string. Module items (nested entries) are never checked for cursors. Additionally, `inst.Entry` (singular, from TimelinePinEntry instructions) is also checked.

**Implementation**: `parsing/cursors.go` `ExtractCursorFromInstructions` checks `entry.Content.CursorType == "Bottom"` for all `inst.Entries`, then checks `inst.Entry` if it is non-nil. Returns the first match's `entry.Content.Value`.

**Why it matters**: Reading `entryType` instead of `cursorType` will miss the cursor. Checking module items adds incorrect matches. Missing the cursor stops pagination after the first page.

---

### #8 — isBlueVerified Field Location

**Category**: Response Path
**Operations**: All tweet/user result parsing

**Naive expectation**: Blue verification status is inside the legacy fields object or in `user.verified`.

**Actual behaviour**: The `is_blue_verified` boolean is a top-level field on the raw tweet (`WireRawTweet`) and raw user (`WireRawUser`) objects, not nested inside `legacy`.

**Implementation**: `types/wire.go` declares `IsBlueVerified bool \`json:"is_blue_verified"\`` on both `WireRawTweet` and `WireRawUser` at the struct top level, not inside `WireTweetLegacy` or `WireUserLegacy`.

**Why it matters**: Looking inside `legacy` for this field always returns false/zero-value. The field was added to the outer result object during the Twitter→X rebrand.

---

### #10 — Chrome Upload Chunk Size (5 MiB)

**Category**: Request Format
**Operations**: UploadMedia (APPEND phase)

**Naive expectation**: Media can be uploaded in any chunk size (e.g. 1 MB, 512 KB, or the full file at once).

**Actual behaviour**: The Twitter/X media upload API enforces a maximum chunk size matching the Chrome browser's upload behaviour: exactly 5 MiB (5 × 1024 × 1024 bytes) per APPEND segment.

**Implementation**: `media.go` defines `mediaChunkSize = 5 * 1024 * 1024`. The APPEND loop slices `data[start:end]` in increments of `mediaChunkSize`. Alt text is only applied for `image/*` MIME types — video alt text is not supported by the API.

**Why it matters**: Chunks larger than 5 MiB are rejected by the upload endpoint. Using an arbitrary chunk size that happens to be larger silently fails the upload.

---

### #11 — Bookmark Folder Count Error Retry

**Category**: Response Path / Request Format
**Operations**: BookmarkFolderTimeline

**Naive expectation**: The `count` variable is always valid for BookmarkFolderTimeline.

**Actual behaviour**: Some bookmark folder schema configurations return a GraphQL error `Variable "$count"` indicating `count` is not a recognised variable for this query ID. In that case the request must be retried without `count` in the variables.

**Implementation**: `timelines.go` `bookmarkFolderPage` receives an `includeCount bool` parameter. On the first call `includeCount=true`. After parsing GraphQL errors, if any error message contains `Variable "$count"` and `includeCount` is true, the function calls itself recursively with `includeCount=false`. An additional check for `Variable "$cursor"` returns an error immediately (correction #23) rather than retrying.

**Why it matters**: Failing to retry without `count` leaves bookmark folder timelines entirely inaccessible when the server rejects the variable.

---

### #12 — HomeTimeline requestContext Variable

**Category**: Request Format
**Operations**: HomeTimeline, HomeLatestTimeline

**Naive expectation**: HomeTimeline only needs `count` and cursor to operate.

**Actual behaviour**: HomeTimeline requires `requestContext: "launch"` in the variables. Without it, the API may return a `query: unspecified` GraphQL error.

**Implementation**: `home.go` `homeTimelinePage` always includes `"requestContext": "launch"` and `"latestControlAvailable": true` alongside `count`, `includePromotedContent`, and `withCommunity` in the variables map.

**Why it matters**: Omitting `requestContext` triggers the `query: unspecified` error path (correction #77), causing unnecessary query ID refresh cycles.

---

### #13 — UserTweets withArticlePlainText fieldToggle

**Category**: Request Format
**Operations**: UserTweets

**Naive expectation**: UserTweets uses the same fieldToggles as TweetDetail, including `withArticlePlainText: true`.

**Actual behaviour**: UserTweets requires `fieldToggles: {"withArticlePlainText": false}`. Sending `true` may cause schema validation errors or unexpected response shapes.

**Implementation**: `features.go` `buildUserTweetsFieldToggles` returns `map[string]any{"withArticlePlainText": false}`. `user_tweets.go` `fetchUserTweetsPage` passes this as the `fieldToggles` URL parameter.

**Why it matters**: The article plain text toggle is a feature-gated field; requesting it on UserTweets causes server-side errors on some query IDs.

---

### #29 — UserTweets No withV2Timeline

**Category**: Request Format
**Operations**: UserTweets, Likes

**Naive expectation**: UserTweets and Likes use `withV2Timeline: true` like many other timeline operations.

**Actual behaviour**: UserTweets and Likes must not include `withV2Timeline` in their variables. Sending it either does nothing or triggers a validation error depending on the query ID.

**Implementation**: `user_tweets.go` `fetchUserTweetsPage` variables: `userId`, `count`, `includePromotedContent`, `withQuickPromoteEligibilityTweetFields`, `withVoice` — no `withV2Timeline`. `timelines.go` `likesPage` variables: `userId`, `count`, `includePromotedContent`, `withClientEventToken`, `withBirdwatchNotes`, `withVoice` — no `withV2Timeline`.

**Why it matters**: Including `withV2Timeline` on these operations causes `GRAPHQL_VALIDATION_FAILED` on some query IDs.

---

### #30 — Lists No Cursor Pagination

**Category**: Request Format / Pagination
**Operations**: ListOwnerships, ListMemberships

**Naive expectation**: List operations paginate using a cursor like all other timeline operations.

**Actual behaviour**: ListOwnerships and ListMemberships do not accept a `cursor` variable. Instead they use `count: 100` and additional membership variables. The cursor parameter is explicitly ignored (`_ string`) in the function signature.

**Implementation**: `lists.go` `fetchListsPage` signature is `(ctx, operation, userID, _ string, includeRaw bool)` — the cursor parameter is discarded. Variables sent: `userId`, `count: 100`, `isListMembershipShown: true`, `isListMemberTargetUserId: <userID>`. No cursor key is added.

**Why it matters**: Sending a cursor causes a schema validation error. The API returns all lists in a single large page (up to 100).

---

### #31 — PreviewURL :small Suffix

**Category**: Response Path
**Operations**: Media extraction (all tweet operations)

**Naive expectation**: The preview URL for media thumbnails is a separate field in the wire response, distinct from the main media URL.

**Actual behaviour**: Twitter/X does not return a separate preview URL field. The preview URL is constructed by appending `:small` to `media_url_https`. This applies to ANY media that has a `sizes.small` entry, not just video or GIF thumbnails.

**Implementation**: `parsing/media.go` `ExtractMedia`: if `m.Sizes.Small != nil`, then `tm.PreviewURL = m.MediaURLHttps + ":small"`.

**Why it matters**: Omitting the suffix serves the full-resolution image where a thumbnail is expected. Applying it only to video/GIF misses photo thumbnails.

---

### #32 — Media Dimensions Large → Medium Fallback

**Category**: Response Path
**Operations**: Media extraction (all tweet operations)

**Naive expectation**: The canonical dimensions for a media item come from whichever size is available, with no preference ordering.

**Actual behaviour**: `sizes.large` is the authoritative size. If `large` is absent (e.g. for some GIFs or older media), `sizes.medium` is used as a fallback. `sizes.small` and `sizes.thumb` are never used for dimensions.

**Implementation**: `parsing/media.go` `ExtractMedia`: `if m.Sizes.Large != nil { tm.Width = m.Sizes.Large.W; tm.Height = m.Sizes.Large.H } else if m.Sizes.Medium != nil { ... }`.

**Why it matters**: Using `small` or `thumb` dimensions incorrectly reports much smaller media dimensions. Using `medium` as primary misreports dimensions when `large` is present.

---

### #35 — Search POST with Variables in URL

**Category**: Request Format
**Operations**: SearchTimeline

**Naive expectation**: SearchTimeline sends all data in the POST body (like CreateTweet) or all in URL query parameters (like a GET request).

**Actual behaviour**: SearchTimeline uses POST but puts `variables` in the URL query string, not the body. The body contains only `features` and `queryId`.

**Implementation**: `search.go` `searchPage` marshals vars to JSON, parses the endpoint URL, sets `q2.Set("variables", string(varsJSON))` on the URL, then calls `c.doPOSTJSON` with a body of `{"features": ..., "queryId": ...}`.

**Why it matters**: Putting variables in the body causes `GRAPHQL_VALIDATION_FAILED` because the server cannot find the `rawQuery` variable. Sending as a GET request is not supported for this operation.

---

### #36 — TweetDetail GET → POST Fallback

**Category**: Request Format
**Operations**: TweetDetail

**Naive expectation**: TweetDetail uses a single HTTP method consistently.

**Actual behaviour**: TweetDetail first tries GET. If GET returns 404, it immediately retries with POST for the same query ID before moving on. If both GET and POST return 404, the next query ID is tried. Only after all query IDs are exhausted with 404s does a query ID refresh occur, followed by one more full retry cycle.

**Implementation**: `tweet_detail.go` `tryQueryIDs` inner loop: `doGET` → if 404 → `doPOSTJSON` with same queryID → if 404 → continue to next ID. After loop: if `had404`, call `refreshQueryIDs` and retry via `tryQueryIDs`.

**Why it matters**: Using GET-only misses tweets that require POST. Using POST-only skips valid GET responses. The fallback ordering must be GET first.

---

### #39/#72 — CreateTweet 4-Step Fallback

**Category**: Request Format
**Operations**: CreateTweet

**Naive expectation**: CreateTweet tries the current query ID; on failure it refreshes and retries once.

**Actual behaviour**: CreateTweet has a 4-step fallback chain:
1. POST `/graphql/<queryId>/CreateTweet` with current query ID
2. If 404: refresh query IDs, POST `/graphql/<newQueryId>/CreateTweet`
3. If still 404: POST `/graphql` (base URL, no operation path) with the same body
4. If any step returns GraphQL error code `226`: fall back to REST `statuses/update.json`

The error code `226` indicates an automated behaviour detection block. The REST fallback uses `StatusUpdateURL` with form encoding.

**Implementation**: `post.go` `createTweet` implements all four steps. Step 4 (`tryStatusUpdateFallback`) uses `doPOSTForm` with `status` and optionally `in_reply_to_status_id` and `auto_populate_reply_metadata=true`. Response prefers `id_str` over numeric `id`.

**Why it matters**: Without the full chain, tweet creation fails silently when query IDs rotate or rate limiting kicks in.

---

### #45 — ListTimeline Response Path

**Category**: Response Path
**Operations**: ListLatestTweetsTimeline

**Naive expectation**: List timeline response uses a path like `data.list_timeline` or `data.timeline`.

**Actual behaviour**: ListLatestTweetsTimeline response path is `data.list.tweets_timeline.timeline.instructions`.

**Implementation**: `lists.go` `parseListTimelineResponse` unmarshals into `Data.List.TweetsTimeline.Timeline.Instructions`.

**Why it matters**: All other timeline paths differ. Using a guessed path yields empty results.

---

### #46 — DefaultNewsTabs No Trending

**Category**: Request Format
**Operations**: GetNews / GenericTimelineById

**Naive expectation**: Default news tabs include `trending` alongside `forYou`, `news`, `sports`, and `entertainment`.

**Actual behaviour**: `trending` is NOT in `DefaultNewsTabs`. The default set is `["forYou", "news", "sports", "entertainment"]`. `trending` is defined in `GenericTimelineTabIDs` but omitted from `DefaultNewsTabs`.

**Implementation**: `constants.go` `DefaultNewsTabs = []string{"forYou", "news", "sports", "entertainment"}`. The `trending` key exists in `GenericTimelineTabIDs` but is absent from `DefaultNewsTabs`.

**Why it matters**: Including `trending` in the default fetch unnecessarily doubles request volume and may trigger rate limiting. Callers must opt in to trending by passing it explicitly in `NewsOptions.Tabs`.

---

### #50 — HomeTimeline No nextCursor

**Category**: Pagination
**Operations**: HomeTimeline, HomeLatestTimeline

**Naive expectation**: Home timeline returns a `NextCursor` for callers to use, like other paginated operations.

**Actual behaviour**: HomeTimeline pagination cursor is consumed entirely internally. The result returned to callers always has `NextCursor: ""`.

**Implementation**: `home.go` `getHomeTimelineInternal`: `result := paginateInline(...)` then `result.NextCursor = ""` before returning.

**Why it matters**: Exposing the home timeline cursor would allow callers to resume mid-stream, but the home feed is designed for infinite scrolling within a single session. Returning a cursor would mislead callers into expecting deterministic pagination.

---

### #51 — Likes "Query: Unspecified" Refresh

**Category**: Response Path / Auth
**Operations**: Likes

**Naive expectation**: All query IDs are tried; if all fail, the error is returned.

**Actual behaviour**: For Likes, GraphQL errors containing exactly `"Query: Unspecified"` (case-sensitive) are non-fatal per query ID — the next ID is tried. After all IDs are exhausted, if the accumulated error messages contain `"Query: Unspecified"`, a query ID refresh is triggered and all IDs are tried once more.

**Implementation**: `timelines.go` `GetLikes`: inside the inner loop, `strings.Contains(lastErr.Error(), "Query: Unspecified")` causes `continue` rather than breaking. After the loop, `strings.Contains(accumulatedErrMsg, "Query: Unspecified") && !refreshed` triggers `refreshQueryIDs` and a second full pass.

**Why it matters**: `Query: Unspecified` from the Likes endpoint is a transient stale-ID signal, not a hard failure. Treating it as fatal would prevent likes from loading when IDs are slightly stale.

---

### #58 — tweetResult camelCase Path

**Category**: Response Path
**Operations**: TweetDetail (GetTweet)

**Naive expectation**: The single-tweet result key uses snake_case: `tweet_result`.

**Actual behaviour**: The key is camelCase: `tweetResult`.

**Implementation**: `tweet_detail.go` `GetTweet` unmarshals into `Data.TweetResult` with JSON tag `json:"tweetResult"`.

**Why it matters**: Using `tweet_result` yields a null pointer; the tweet is not found even when the request succeeds.

---

### #60 — home_timeline_urt Path

**Category**: Response Path
**Operations**: HomeTimeline, HomeLatestTimeline

**Naive expectation**: Home timeline instructions are at `data.home_timeline.timeline.instructions` or `data.timeline.instructions`.

**Actual behaviour**: The path is `data.home.home_timeline_urt.instructions` — with `home_timeline_urt` (URT = Unified Rich Timeline) as the inner key.

**Implementation**: `home.go` `homeTimelinePage` unmarshals into `Data.Home.HomeTimelineUrt.Instructions` with JSON tag `json:"home_timeline_urt"`.

**Why it matters**: Wrong path means empty instruction slice and no tweets returned.

---

### #61 — HomeTimeline Variables

**Category**: Request Format
**Operations**: HomeTimeline, HomeLatestTimeline

**Naive expectation**: HomeTimeline variables are `{"count": 20}` plus cursor.

**Actual behaviour**: HomeTimeline requires five specific variables:
- `count` (int)
- `includePromotedContent: true`
- `latestControlAvailable: true`
- `requestContext: "launch"` (see also correction #12)
- `withCommunity: true`

Plus optional `cursor` when paginating. Both `variables` and `features` are sent as URL query parameters on a GET request, not in the body.

**Implementation**: `home.go` `homeTimelinePage` builds the full variable map and marshals both `varsJSON` and `featJSON` into URL query parameters. Uses `doGET`, not `doPOSTJSON`.

**Why it matters**: Missing `latestControlAvailable` or `withCommunity` causes degraded results. Missing `requestContext` triggers the `query: unspecified` error.

---

### #62 — search_by_raw_query Path

**Category**: Response Path
**Operations**: SearchTimeline

**Naive expectation**: Search results are at `data.search.timeline.instructions` or `data.search_timeline.instructions`.

**Actual behaviour**: The path is `data.search_by_raw_query.search_timeline.timeline.instructions`.

**Implementation**: `search.go` `searchPage` unmarshals into `Data.SearchByRawQuery.SearchTimeline.Timeline.Instructions`.

**Why it matters**: Using any other path produces empty results on a successful 200 response.

---

### #63 — screen_name Snake_case

**Category**: Request Format
**Operations**: UserByScreenName

**Naive expectation**: Since `AboutAccountQuery` uses camelCase `screenName` (correction #2), UserByScreenName might also use camelCase.

**Actual behaviour**: UserByScreenName variables use snake_case: `screen_name` (not `screenName`). The two operations are inconsistent with each other.

**Implementation**: `user_lookup.go` `fetchUserByScreenName` builds `vars := map[string]any{"screen_name": username, "withSafetyModeUserFields": true}`.

**Why it matters**: Mixing up the casing between the two operations causes the query to receive a null user argument.

---

### #67 — News Count Multiplication

**Category**: Request Format
**Operations**: GenericTimelineById

**Naive expectation**: The `count` variable is passed as the user-specified `maxCount` integer.

**Actual behaviour**: `count` is passed as a string (not integer) and is doubled: `strconv.Itoa(maxCount * 2)`. The API uses this as the number of items to fetch per tab; doubling compensates for promoted/filtered content that does not appear in results.

**Implementation**: `news.go` `fetchGenericTimeline`: `vars["count"] = strconv.Itoa(maxCount * 2)`.

**Why it matters**: Passing an integer instead of string causes a type validation error. Passing `maxCount` without doubling returns roughly half the expected number of news items.

---

### #71 — Exactly 29 FallbackQueryIDs

**Category**: Other
**Operations**: All operations with query IDs

**Naive expectation**: The number of fallback query IDs is not significant; it can be any number.

**Actual behaviour**: `FallbackQueryIDs` must contain exactly 29 entries. This count is documented in the code comment and verified in tests (`constants_test.go`). Operations not in `BundledBaselineQueryIDs` fall through to this map, so a missing entry means no query ID at all for that operation.

**Implementation**: `constants.go` comment: `// Correction #71: exactly 29 entries, not 24.` The map has 29 keys.

**Why it matters**: Adding or removing operations without updating `FallbackQueryIDs` causes those operations to silently have no fallback query ID and always fail.

---

### #76 — Search GRAPHQL_VALIDATION_FAILED

**Category**: Response Path / Auth
**Operations**: SearchTimeline

**Naive expectation**: HTTP 4xx means the request is unretryable.

**Actual behaviour**: HTTP 400 or 422 with `GRAPHQL_VALIDATION_FAILED` in the body means the query ID is stale, not that the request is fundamentally wrong. The correct response is to refresh query IDs and retry. Additionally, a 200 response with a GraphQL error whose `extensions.code == "GRAPHQL_VALIDATION_FAILED"` or whose path contains `"rawQuery"` and whose message matches `/must be defined/i` is the same signal.

**Implementation**: `search.go` `isSearchQueryIDMismatch` checks `e.Extensions.Code == "GRAPHQL_VALIDATION_FAILED"` and the `rawQuery` path + message regex. Both `Search` and `GetAllSearchResults` check for `GRAPHQL_VALIDATION_FAILED` in HTTP error bodies for status 400/422, then call `refreshQueryIDs`.

**Why it matters**: Treating `GRAPHQL_VALIDATION_FAILED` as a permanent error permanently breaks search when query IDs rotate.

---

### #77 — Home query: unspecified Refresh

**Category**: Response Path / Auth
**Operations**: HomeTimeline, HomeLatestTimeline

**Naive expectation**: Home timeline errors return a non-200 HTTP status, making failure detection straightforward.

**Actual behaviour**: The server returns HTTP 200 with a GraphQL error whose message matches `(?i)query:\s*unspecified`. This indicates a stale query ID and requires a refresh-and-retry cycle.

**Implementation**: `home.go` defines `queryUnspecifiedRe = regexp.MustCompile(\`(?i)query:\s*unspecified\`)`. `homeTimelinePage` calls `parseGraphQLErrors` and matches each error message against the regex. On match, it returns a failure that triggers `refreshQueryIDs` in the outer loop.

**Why it matters**: Without this detection, the loop exits with an unhelpful error message and the timeline never loads after a query ID rotation.

---

### #80 — withAuxiliaryUserLabels:false

**Category**: Request Format
**Operations**: UserByScreenName

**Naive expectation**: `fieldToggles` can be omitted for user lookup operations.

**Actual behaviour**: UserByScreenName must include `fieldToggles: {"withAuxiliaryUserLabels": false}`. Without it, some query IDs return additional label data that changes the response shape in ways that can break parsing.

**Implementation**: `features.go` `buildUserByScreenNameFieldToggles` returns `map[string]any{"withAuxiliaryUserLabels": false}`. `user_lookup.go` `fetchUserByScreenName` marshals and URL-encodes this as the `fieldToggles` query parameter.

**Why it matters**: Omitting fieldToggles or sending `true` may cause parsing failures or return unexpected fields.

---

### #82 — Following count:20 Variables

**Category**: Request Format
**Operations**: Following, Followers

**Naive expectation**: Following/Followers use the default `count` from user options.

**Actual behaviour**: Following and Followers always use `count: 20` and `includePromotedContent: false`, regardless of caller options. On 404 with a refresh, a REST fallback is used (`friends/list.json` or `followers/list.json`) — but only when `refreshed=true`, not on the first 404.

**Implementation**: `following.go` `fetchFollowPage` hardcodes `"count": 20, "includePromotedContent": false`. `fetchFollowPageWithRefresh` only calls `fetchFollowPageREST` when `refreshed && ar.had404`. Both primary (`x.com/i/api/1.1`) and fallback (`api.twitter.com/1.1`) REST URLs are tried.

**Why it matters**: Sending user-controlled counts can trigger validation errors. The REST fallback must not be used on the first failure — only after a query-ID refresh fails.

---

### #83 — paginateCursor Zero-Item Behaviour

**Category**: Pagination
**Operations**: TweetDetail (GetReplies, GetThread)

**Naive expectation**: Zero items on a page means pagination is complete.

**Actual behaviour**: For cursor-based pagination (`paginateCursor` in `tweet_detail.go`), an empty items page does NOT stop pagination. Only an empty cursor or an unchanged cursor signals termination.

**Implementation**: `tweet_detail.go` `paginateCursor` step 7: `if pageCursor == "" || pageCursor == cursor { return ... }`. There is no check for `len(page.Items) == 0`.

**Why it matters**: Reply threads can have empty intermediate pages (injected items, ads, separators) followed by real replies. Stopping on empty pages truncates threads.

---

### #85 — Bookmarks Variables Outside Loop

**Category**: Request Format
**Operations**: Bookmarks

**Naive expectation**: Variables including the cursor are rebuilt inside the per-query-ID retry loop.

**Actual behaviour**: Bookmark variables (including the baked-in cursor) are built once per page call, before the query-ID loop. All query ID attempts for the same page share the same serialized variables JSON.

**Implementation**: `timelines.go` `GetBookmarks` fetchFn: `vars` and `varsJSON` are constructed before `for _, qid := range queryIDs`. The cursor is included in `vars` if non-empty.

**Why it matters**: Rebuilding variables inside the loop would work correctly for most cases, but building them outside ensures consistent behaviour — the cursor cannot drift between attempts for the same logical page.

---

### #86 — fetchWithRetry 3 Attempts

**Category**: Other
**Operations**: Bookmarks, BookmarkFolderTimeline

**Naive expectation**: `fetchWithRetry` retries indefinitely or has a configurable retry count.

**Actual behaviour**: `fetchWithRetry` makes exactly 3 total attempts (attempt 0, 1, 2): `maxRetries=2`, loop condition `attempt <= maxRetries`. The post-loop `return nil, lastErr` is dead code — the loop always returns before reaching it.

Retryable statuses: 429, 500, 502, 503, 504. Non-retryable statuses (including 404) fail immediately on the first occurrence.

Delay uses `Retry-After` header when present; otherwise exponential backoff: `baseDelay<<attempt + jitter` where `baseDelayMs=500` and jitter is `rand.Intn(500)` ms.

**Implementation**: `http.go` `fetchWithRetry`. The `// Correction #86: this return is dead code` comment marks the unreachable post-loop return.

**Why it matters**: Expecting more than 3 attempts, or expecting 404 to be retried, will result in incorrect assumptions about bookmark availability under rate limiting.
