# GoBird Implementation Plan

Date: 2026-03-12 (revised)
Target repo: `/Users/mudrii/src/gobird`
Module path: `github.com/mudrii/gobird`
Source of truth: `/Users/mudrii/src/bird-docs` (corrections.md takes highest priority)

---

## 1. Inputs and Current State

This plan is derived from reading all eleven source documents in priority order:

1. `corrections.md` — 86 verified source-code corrections (mandatory)
2. `README.md`
3. `cli-commands.md`
4. `graphql-operations.md`
5. `api-endpoints.md`
6. `features-flags.md`
7. `response-parsing.md`
8. `data-types.md`
9. `authentication.md`
10. `architecture.md`
11. `go-implementation-guide.md`

Observed repo state: `/Users/mudrii/src/gobird` contains only `IMPLEMENTATION_PROMPT.md` and this plan. No Go module, source, tests, or CI exist yet.

---

## 2. Source-of-Truth Rules

- `corrections.md` overrides all other documents on every specific behavior it mentions.
- Lower-priority docs are authoritative on behaviors not addressed by higher-priority ones.
- If any doc conflicts with `/Users/mudrii/src/bird` compiled source, follow the source.
- Do not simplify behaviors that look odd but are explicitly documented.
- Never guess. Read the reference source before implementing any ambiguous behavior.

Known prompt/doc gaps already resolved:
- `follow` and `unfollow` appear in CLI docs and library scope; implement them.
- `bird read` supports `--json-full`; preserve it.
- `paginateCursor` stops only on cursor change, not on zero items.
- `FALLBACK_QUERY_IDS` has 29 entries, not 24 (correction #71).
- `getHeaders()` delegates to `getJsonHeaders()`, not `getBaseHeaders()` (correction #70).
- Default news tabs are `forYou`, `news`, `sports`, `entertainment` — NOT `trending` (correction #46).
- `fetchWithRetry` post-loop return is dead code (correction #86); 3 total attempts: 0, 1, 2.

---

## 3. Success Criteria

Done when all of the following are true:
- All documented CLI commands, aliases, shorthand dispatch, and help flows work.
- All global flags, per-command flags, config files, and env precedence rules match the docs.
- Authentication, headers, cookie resolution, current-user lookup, and request signing work.
- GraphQL query ID loading, runtime refresh, cache persistence, and fallback sequences work.
- Feature maps, field toggles, and runtime overrides match the docs per operation.
- Response parsing, normalization, quote-depth, article rendering, media, thread metadata, and cursor handling are correct.
- JSON and human-readable output contracts match the reference.
- Exit codes: `0` success, `1` runtime/auth/API/network failure, `2` usage/validation failure.
- Public Go API exposes the documented library surface with strong typing and doc comments.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, `golangci-lint run`, `go build ./...` all pass.

---

## 4. Non-Negotiable Constraints

- Go 1.23+; module `github.com/mudrii/gobird`
- Strong typing on all public APIs and normalized models
- `context.Context` as first parameter for every networked operation
- Never log `auth_token`, `ct0`, raw cookie headers, or any equivalent secret
- All exported identifiers must have doc comments
- No dead code, placeholder stubs, or speculative commands
- TDD for units; ATDD for CLI acceptance and end-to-end parity
- Exact observable behavior preserved over "cleaner" abstractions

---

## 5. Repository Layout

```
/Users/mudrii/src/gobird
├── cmd/
│   └── bird/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── cookies.go          # browser extraction: Safari, Chrome, Firefox
│   │   ├── resolve.go          # resolveCredentials: CLI → env → browser
│   │   └── resolve_test.go
│   ├── cli/
│   │   ├── root.go             # root command, global flags, shorthand injection
│   │   ├── shared.go           # resolveCredentialsFromOptions, loadMedia, detectMime
│   │   ├── pagination.go       # shared CLI pagination output helpers
│   │   ├── input.go            # looksLikeTweetInput, extractTweetID, extractListID, etc.
│   │   ├── help.go
│   │   ├── tweet.go            # tweet, reply commands
│   │   ├── read.go             # read, replies, thread commands
│   │   ├── search.go           # search, mentions commands
│   │   ├── bookmarks.go        # bookmarks, unbookmark commands
│   │   ├── home.go             # home command
│   │   ├── users.go            # following, followers, likes, whoami, about, follow, unfollow
│   │   ├── lists.go            # lists, list-timeline commands
│   │   ├── news.go             # news, trending commands
│   │   ├── check.go            # check command
│   │   └── query_ids.go        # query-ids command
│   ├── client/
│   │   ├── client.go           # Client struct, New(), ensureClientUserID()
│   │   ├── constants.go        # API URLs, FallbackQueryIDs (29), TargetQueryIDOperations
│   │   ├── headers.go          # baseHeaders(), jsonHeaders(), uploadHeaders()
│   │   ├── http.go             # fetchWithTimeout(), doGET(), doPOSTJSON(), doPOSTForm(), fetchWithRetry()
│   │   ├── query_ids.go        # getQueryID(), refreshQueryIDs(), withRefreshedQueryIDsOn404()
│   │   ├── features.go         # all 12 buildXxxFeatures() functions + applyFeatureOverrides()
│   │   ├── post.go             # Tweet(), Reply(), createTweet(), tryStatusUpdateFallback()
│   │   ├── search.go           # Search(), GetAllSearchResults()
│   │   ├── tweet_detail.go     # GetTweet(), GetReplies(), GetThread(), paged variants
│   │   ├── home.go             # GetHomeTimeline(), GetHomeLatestTimeline()
│   │   ├── timelines.go        # GetBookmarks(), GetLikes(), GetBookmarkFolderTimeline()
│   │   ├── bookmarks.go        # Unbookmark()
│   │   ├── engagement.go       # Like(), Unlike(), Retweet(), Unretweet(), Bookmark()
│   │   ├── follow.go           # Follow(), Unfollow(), followViaREST(), followViaGraphQL()
│   │   ├── user_tweets.go      # GetUserTweets(), GetUserTweetsPaged()
│   │   ├── users.go            # GetCurrentUser(), GetFollowing(), GetFollowers()
│   │   ├── user_lookup.go      # GetUserIDByUsername(), GetUserAboutAccount()
│   │   ├── lists.go            # GetOwnedLists(), GetListMemberships(), GetListTimeline()
│   │   ├── news.go             # GetNews()
│   │   └── media.go            # UploadMedia()
│   ├── config/
│   │   ├── config.go           # JSON5 config loading, env var resolution
│   │   └── config_test.go
│   ├── output/
│   │   ├── format.go           # rich text, --plain, --no-emoji, --no-color, OSC 8 hyperlinks
│   │   ├── json.go             # --json, --json-full rendering
│   │   └── format_test.go
│   ├── parsing/
│   │   ├── timeline.go         # parseTweetsFromInstructions, collectTweetResultsFromEntry
│   │   ├── tweet.go            # mapTweetResult, unwrapTweetResult, extractTweetText
│   │   ├── article.go          # extractArticleText, extractArticleMetadata
│   │   ├── draftjs.go          # renderContentState, renderBlockText, renderAtomicBlock
│   │   ├── cursors.go          # extractCursorFromInstructions
│   │   ├── users.go            # parseUsersFromInstructions
│   │   ├── lists.go            # parseListsFromInstructions
│   │   ├── news.go             # parseTimelineTabItems, parseNewsItemFromContent
│   │   ├── thread_filters.go   # filterAuthorChain, filterAuthorOnly, filterFullChain, addThreadMetadata
│   │   ├── media.go            # extractMedia, video variant selection
│   │   └── *_test.go
│   ├── runtime/
│   │   ├── query_id_store.go   # disk cache, memory cache, TTL, coalesced refresh
│   │   ├── query_ids.go        # scraping, bundle regex, 4 extraction patterns
│   │   ├── features.go         # loadFeatureOverrides, applyFeatureOverrides
│   │   ├── overrides.go        # mergeOverrides, normalization, BIRD_FEATURES_JSON
│   │   └── *_test.go
│   ├── types/
│   │   ├── wire.go             # raw GraphQL/REST response structs
│   │   ├── models.go           # normalized output types
│   │   ├── results.go          # result unions, partial pagination results
│   │   └── options.go          # fetch option structs
│   └── testutil/
│       ├── httpmock.go
│       ├── fixtures.go
│       ├── golden.go
│       └── clock.go
├── pkg/
│   └── bird/
│       ├── client.go           # exported TwitterClient wrapper
│       ├── auth.go             # exported ResolveCredentials, Extract* helpers
│       ├── types.go            # re-exported public types
│       └── doc.go
├── tests/
│   ├── acceptance/             # CLI acceptance tests
│   ├── integration/            # mocked-HTTP integration tests
│   ├── fixtures/               # raw response JSON fixtures by operation family
│   └── golden/                 # golden output files (text, JSON, plain)
├── .github/workflows/ci.yml
├── .golangci.yml
├── Makefile
├── go.mod
└── README.md
```

---

## 6. Core Architectural Decisions

- Single `Client` struct, methods split across files by domain. No TypeScript mixin emulation.
- Separate layers: HTTP transport → runtime metadata → parsing → CLI rendering.
- Raw wire structs (in `internal/types/wire.go`) kept strictly separate from normalized output structs.
- Centralize 404 refresh logic in `withRefreshedQueryIDsOn404`; do not re-implement per operation.
- Injectable `http.Client` or `http.RoundTripper` for deterministic tests.
- CLI commands call typed client methods; no API logic in command handlers.
- `getHeaders()` = `getJsonHeaders()` (includes `content-type: application/json`). Correction #70.
- `getUploadHeaders()` = `getBaseHeaders()` only — no content-type override. Correction #70.

---

## 7. Critical Parity Traps (Memorize These)

Every item here has a regression test requirement.

### Response Path Corrections (corrections.md §1)

| Operation | CORRECT path |
|---|---|
| BookmarkFolderTimeline | `data.bookmark_collection_timeline.timeline.instructions` |
| Likes | `data.user.result.timeline.timeline.instructions` |
| UserTweets | `data.user.result.timeline.timeline.instructions` |
| Following/Followers | `data.user.result.timeline.timeline.instructions` |
| ListOwnerships | `data.user.result.timeline.timeline.instructions` |
| ListMemberships | `data.user.result.timeline.timeline.instructions` |
| GenericTimelineById | `data.timeline.timeline.instructions` |
| AboutAccountQuery | `data.user_result_by_screen_name.result.about_profile` |
| HomeTimeline | `data.home.home_timeline_urt.instructions` (with `_urt` suffix, correction #60) |
| SearchTimeline | `data.search_by_raw_query.search_timeline.timeline.instructions` (correction #62) |
| TweetDetail single | `data.tweetResult.result` (camelCase; correction #58) |
| TweetDetail thread | `data.threaded_conversation_with_injections_v2.instructions` |
| CreateTweet response | `data.create_tweet.tweet_results.result.rest_id` (all snake_case; correction #38) |
| Bookmarks | `data.bookmark_timeline_v2.timeline.instructions` |
| ListLatestTweetsTimeline | `data.list.tweets_timeline.timeline.instructions` (correction #45) |
| UserArticlesTweets | `data.user.result.timeline.timeline.instructions` |

### Variable Name Corrections (corrections.md §2)

| Operation | CORRECT variable |
|---|---|
| AboutAccountQuery | `screenName` (camelCase) |
| UserByScreenName | `screen_name` (snake_case) — opposite of AboutAccountQuery (correction #63) |
| CreateFriendship (GraphQL) | `user_id` (snake_case) |
| DestroyFriendship (GraphQL) | `user_id` (snake_case) |

### Behavior Corrections (critical subset)

- **DeleteRetweet**: both `tweet_id` AND `source_tweet_id` set to same ID; no `dark_request`. No `features` in body (correction #3).
- **Engagement mutations**: body is `{ variables, queryId }` — NO `features` (correction #3, #68).
- **Follow/Unfollow**: REST-first always. x.com/i/api then api.twitter.com, then GraphQL (correction #4).
- **Follow REST error codes**: 160 = already following (success), 162 = blocked, 108 = not found (correction #4).
- **UserByScreenName**: hardcoded IDs `xc8f1g7BYqr6VTzTbvNlGw`, `qW5u-DAuXpMEG0zA1F7UGQ`, `sLVLhk0bGj3MVFEKTdax1w`; no runtime cache; `UserUnavailable` → stop immediately (correction #5).
- **getCurrentUser**: 4 API URLs + 2 HTML pages; does NOT call UserByScreenName (correction #6). `data.user.id` only accepted when type is string (correction #78).
- **Cursor extraction**: check `entry.content.cursorType`, NOT `entryType`, NOT module items (correction #7).
- **is_blue_verified**: top-level on result, NOT inside `legacy` (correction #8).
- **Pagination delay**: BEFORE fetch, skip page 0 (correction #9).
- **Media upload**: 5 MiB chunks exactly (`5*1024*1024`); max 20 STATUS polls; alt text image/* only (correction #10).
- **BookmarkFolderTimeline**: retry without `count` on `Variable "$count"` error; return error on `Variable "$cursor"` error (corrections #11, #23).
- **HomeTimeline variables**: `withCommunity:true`, `latestControlAvailable:true`, `requestContext:"launch"` (corrections #12, #61).
- **UserTweets**: `includePromotedContent:false`; `fieldToggles.withArticlePlainText:false`; hard max 10 pages (correction #13).
- **Following/Followers REST fallback**: only after 404-triggered refresh; two URLs each (correction #14).
- **Media previewUrl**: set for ANY media with `sizes.small` (not just video/GIF) (correction #31).
- **Media dimensions**: `sizes.large` first, `sizes.medium` fallback (correction #32).
- **Bookmarks retry**: `maxRetries=2`, `baseDelayMs=500`, backoff = `500*2^attempt + random(0,500)` ms (correction #16).
- **Bookmarks variables**: all six fields including `includePromotedContent:false` (correction #22).
- **BookmarkFolderTimeline**: `includePromotedContent:true` (different from regular bookmarks; correction #23).
- **Likes**: `includePromotedContent:false`; uses `fetchWithTimeout` (no retry); also triggers refresh on `"Query: Unspecified"` string (exact case; corrections #49, #51).
- **HomeTimeline "Query: Unspecified"**: case-insensitive regex `/query:\s*unspecified/i` (correction #77).
- **UserTweets fatal errors**: `"User has been suspended"` and `"User not found"` (exact strings; correction #24).
- **unwrapTweetResult**: checks `result.tweet` truthy; no `__typename` check (correction #25).
- **collectTweetResultsFromEntry**: checks exactly 5 paths (correction #26).
- **paginateCursor stop conditions**: empty/unchanged cursor ONLY; no zero-items stop (correction #18).
- **paginateCursor error with no accumulated items**: return raw page failure object (correction #83).
- **Inline paginator**: also stops on `page.tweets.length === 0` OR `added === 0` (correction #33).
- **Tweet text priority**: article first, then note tweet, then legacy.full_text (correction #19).
- **Article text**: `content_state` (Draft.js rich rendering) tried BEFORE `plain_text` (correction #54).
- **Draft.js entities**: `MARKDOWN` (not `code-block`); `IMAGE` outputs `[Image]` with no URL (correction #20).
- **ContentState block join**: double newline `\n\n` between blocks; result trimmed (correction #53).
- **normalizeHandle**: regex `/^[A-Za-z0-9_]{1,15}$/` strictly; returns null on failure (correction #52).
- **TweetDetail variables**: no `referrer`, no `count` (correction #28); see exact variables in §12.
- **TweetDetail GET→POST fallback**: per-queryId GET first, 404 → POST same queryId, 404 → next queryId (correction #36).
- **TweetDetail partial errors**: non-fatal if usable data present (correction #37).
- **SearchTimeline**: POST with variables in URL query string, features+queryId in body (correction #35).
- **SearchTimeline product**: `"Latest"` hardcoded (not `"Top"`); `querySource`: `"typed_query"` (correction #35).
- **CreateTweet fallback**: code 226 only (not any error); `statuses/update.json` form-encoded (correction #39).
- **statuses/update.json response**: `id_str` first, `String(id)` fallback (correction #40).
- **Env vars**: `AUTH_TOKEN` > `TWITTER_AUTH_TOKEN`; `CT0` > `TWITTER_CT0` (correction #41).
- **Domain preference**: x.com > twitter.com > first match (correction #42).
- **Query IDs cache**: `~/.config/bird/query-ids-cache.json`; TTL 24h; env `BIRD_QUERY_IDS_CACHE` (correction #43).
- **Features cache**: `BIRD_FEATURES_CACHE` > `BIRD_FEATURES_PATH` > `~/.config/bird/features.json` (correction #44).
- **News default tabs**: `forYou`, `news`, `sports`, `entertainment` — NOT `trending` (correction #46).
- **News deduplication key**: `headline` string (not URL or ID; correction #47).
- **News item ID when no URL**: `entryId ? "${entryId}-${headline}" : "${tabName}-${headline}"` (correction #48).
- **Likes `includePromotedContent`**: false (not absent); no `withV2Timeline` (corrections #49, #29).
- **UserTweets `withV2Timeline`**: NOT included (correction #29).
- **ListOwnerships variables**: `isListMembershipShown:true` and `isListMemberTargetUserId` both required (correction #30).
- **ListLatestTweetsTimeline variables**: only `listId`, `count`, `cursor` (correction #84).
- **GenericTimelineById**: `count = maxCount * 2`; `includePromotedContent:false` (corrections #34, #67).
- **ExploreSidebar/ExplorePage/TrendHistory**: in FALLBACK_QUERY_IDS but never called (corrections #64, #20).
- **ListByRestId**: in FALLBACK_QUERY_IDS but never called (correction #65).
- **UserArticlesTweets**: only called as article-text fallback inside getTweet (correction #66).
- **getFollowing/getFollowers variables**: include `includePromotedContent:false` (correction #82).
- **refreshQueryIds full**: targets all 29 FALLBACK_QUERY_IDS keys (correction #71).
- **query-ids CLI command**: refreshes only 10 operations (correction #69).
- **Double-404 fallback**: TWITTER_GRAPHQL_POST_URL = TWITTER_API_BASE = `https://x.com/i/api/graphql` (correction #72).
- **SearchTimeline query IDs**: 4 IDs total (correction #73).
- **TweetDetail query IDs**: 3 IDs total (correction #74).
- **BookmarkFolderTimeline query IDs**: 2 IDs total (correction #75).
- **GRAPHQL_VALIDATION_FAILED**: HTTP 400/422 also triggers refresh for SearchTimeline (correction #76).
- **UserByScreenName REST fallback**: `users/show.json` (not `users/lookup.json`; correction #55).
- **UserByScreenName response**: also checks `core.screen_name` and `core.name` (correction #79).
- **UserByScreenName fieldToggles**: `withAuxiliaryUserLabels:false` (correction #80).
- **detectMime returns null**: the CLI caller throws, not detectMime itself (correction #81).
- **Bookmarks variables**: built once outside per-queryId loop (correction #85).
- **fetchWithRetry dead code**: post-loop return is unreachable; 3 total attempts correct (correction #86).
- **HomeTimeline returns no nextCursor** to caller (correction #50).
- **Media poll delay clamp**: default 2s when not finite; clamp to ≥1s (correction #21).
- **CreateTweet media body**: always present even when no media; `media_entities:[]` when empty (graphql-operations.md §5).

---

## 8. Exact Constants

### Bearer Token
```
AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA
```

### User Agent
```
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36
```

### API Base URLs

```
GraphQL:          https://x.com/i/api/graphql
GraphQL POST:     https://x.com/i/api/graphql     (same — correction #72)
REST v1:          https://x.com/i/api/1.1
Media upload:     https://upload.twitter.com/i/media/upload.json
Media metadata:   https://x.com/i/api/1.1/media/metadata/create.json
Status update:    https://x.com/i/api/1.1/statuses/update.json
Follow REST:      https://x.com/i/api/1.1/friendships/create.json
Unfollow REST:    https://x.com/i/api/1.1/friendships/destroy.json
Followers REST:   https://x.com/i/api/1.1/followers/list.json
Following REST:   https://x.com/i/api/1.1/friends/list.json
UserLookup REST:  https://x.com/i/api/1.1/users/show.json
Settings:         https://x.com/i/api/account/settings.json
Credentials:      https://x.com/i/api/account/verify_credentials.json
Settings page:    https://x.com/settings/account
```

All have `api.twitter.com` mirrors tried as alternates.

### All 29 FALLBACK_QUERY_IDS

```
CreateTweet:              TAJw1rBsjAtdNgTdlo2oeg
CreateRetweet:            ojPdsZsimiJrUGLR1sjUtA
DeleteRetweet:            iQtK4dl5hBmXewYZuEOKVw
CreateFriendship:         8h9JVdV8dlSyqyRDJEPCsA
DestroyFriendship:        ppXWuagMNXgvzx6WoXBW0Q
FavoriteTweet:            lI07N6Otwv1PhnEgXILM7A
UnfavoriteTweet:          ZYKSe-w7KEslx3JhSIk5LA
CreateBookmark:           aoDbu3RHznuiSkQ9aNM67Q
DeleteBookmark:           Wlmlj2-xzyS1GN3a6cj-mQ
TweetDetail:              97JF30KziU00483E_8elBA
SearchTimeline:           M1jEez78PEfVfbQLvlWMvQ
UserArticlesTweets:       8zBy9h4L90aDL02RsBcCFg
UserTweets:               Wms1GvIiHXAPBaCr9KblaA
Bookmarks:                RV1g3b8n_SGOHwkqKYSCFw
Following:                BEkNpEt5pNETESoqMsTEGA
Followers:                kuFUYP9eV1FPoEy4N-pi7w
Likes:                    JR2gceKucIKcVNB_9JkhsA
BookmarkFolderTimeline:   KJIQpsvxrTfRIlbaRIySHQ
ListOwnerships:           wQcOSjSQ8NtgxIwvYl1lMg
ListMemberships:          BlEXXdARdSeL_0KyKHHvvg
ListLatestTweetsTimeline: 2TemLyqrMpTeAmysdbnVqw
ListByRestId:             wXzyA5vM_aVkBL9G8Vp3kw
HomeTimeline:             edseUwk9sP5Phz__9TIRnA
HomeLatestTimeline:       iOEZpOdfekFsxSlPQCQtPg
ExploreSidebar:           lpSN4M6qpimkF4nRFPE3nQ
ExplorePage:              kheAINB_4pzRDqkzG3K-ng
GenericTimelineById:      uGSr7alSjR9v6QJAIaqSKQ
TrendHistory:             Sj4T-jSB9pr0Mxtsc1UKZQ
AboutAccountQuery:        zs_jFPFT78rBpXv9Z3U2YQ
```

### Bundled Baseline Query IDs (query-ids.json overrides)

These override fallback values when present:
```
CreateTweet:              nmdAQXJDxw6-0KKF2on7eA
CreateRetweet:            LFho5rIi4xcKO90p9jwG7A
CreateFriendship:         8h9JVdV8dlSyqyRDJEPCsA
DestroyFriendship:        ppXWuagMNXgvzx6WoXBW0Q
FavoriteTweet:            lI07N6Otwv1PhnEgXILM7A
DeleteBookmark:           Wlmlj2-xzyS1GN3a6cj-mQ
TweetDetail:              _NvJCnIjOW__EP5-RF197A
SearchTimeline:           6AAys3t42mosm_yTI_QENg
Bookmarks:                RV1g3b8n_SGOHwkqKYSCFw
BookmarkFolderTimeline:   KJIQpsvxrTfRIlbaRIySHQ
Following:                mWYeougg_ocJS2Vr1Vt28w
Followers:                SFYY3WsgwjlXSLlfnEUE4A
Likes:                    ETJflBunfqNa1uE1mBPCaw
ExploreSidebar:           lpSN4M6qpimkF4nRFPE3nQ
ExplorePage:              kheAINB_4pzRDqkzG3K-ng
GenericTimelineById:      uGSr7alSjR9v6QJAIaqSKQ
TrendHistory:             Sj4T-jSB9pr0Mxtsc1UKZQ
AboutAccountQuery:        zs_jFPFT78rBpXv9Z3U2YQ
```

NOT in query-ids.json (use fallback): `DeleteRetweet`, `UnfavoriteTweet`, `CreateBookmark`, `UserArticlesTweets`, `UserTweets`, `ListOwnerships`, `ListMemberships`, `ListLatestTweetsTimeline`, `ListByRestId`, `HomeTimeline`, `HomeLatestTimeline`.

### Per-Operation Fallback ID Lists (all deduplicated via Set)

```
TweetDetail:              [primary, "97JF30KziU00483E_8elBA", "aFvUsJm2c-oDkJV75blV6g"]
SearchTimeline:           [primary, "M1jEez78PEfVfbQLvlWMvQ", "5h0kNbk3ii97rmfY6CdgAA", "Tp1sewRU1AsZpBWhqCZicQ"]
HomeTimeline:             [primary, "edseUwk9sP5Phz__9TIRnA"]
HomeLatestTimeline:       [primary, "iOEZpOdfekFsxSlPQCQtPg"]
Bookmarks:                [primary, "RV1g3b8n_SGOHwkqKYSCFw", "tmd4ifV8RHltzn8ymGg1aw"]
BookmarkFolderTimeline:   [primary, "KJIQpsvxrTfRIlbaRIySHQ"]
Likes:                    [primary, "JR2gceKucIKcVNB_9JkhsA"]
UserTweets:               [primary, "Wms1GvIiHXAPBaCr9KblaA"]
Following:                [primary, "BEkNpEt5pNETESoqMsTEGA"]
Followers:                [primary, "kuFUYP9eV1FPoEy4N-pi7w"]
CreateFriendship:         [primary, "8h9JVdV8dlSyqyRDJEPCsA", "OPwKc1HXnBT_bWXfAlo-9g"]
DestroyFriendship:        [primary, "ppXWuagMNXgvzx6WoXBW0Q", "8h9JVdV8dlSyqyRDJEPCsA"]
ListOwnerships:           [primary, "wQcOSjSQ8NtgxIwvYl1lMg"]
ListMemberships:          [primary, "BlEXXdARdSeL_0KyKHHvvg"]
ListLatestTweetsTimeline: [primary, "2TemLyqrMpTeAmysdbnVqw"]
AboutAccountQuery:        [primary, "zs_jFPFT78rBpXv9Z3U2YQ"]
UserByScreenName:         ["xc8f1g7BYqr6VTzTbvNlGw", "qW5u-DAuXpMEG0zA1F7UGQ", "sLVLhk0bGj3MVFEKTdax1w"]
  (hardcoded only — never uses runtime cache or FALLBACK_QUERY_IDS)
```

### GenericTimelineById Timeline IDs

```
forYou:          VGltZWxpbmU6DAC2CwABAAAAB2Zvcl95b3UAAA==
trending:        VGltZWxpbmU6DAC2CwABAAAACHRyZW5kaW5nAAA=
news:            VGltZWxpbmU6DAC2CwABAAAABG5ld3MAAA==
sports:          VGltZWxpbmU6DAC2CwABAAAABnNwb3J0cwAA
entertainment:   VGltZWxpbmU6DAC2CwABAAAADWVudGVydGFpbm1lbnQAAA==
```

Default tabs: `forYou`, `news`, `sports`, `entertainment` (NOT `trending`).

---

## 9. Exact Feature Flag Maps

These are the hardcoded baselines. `applyFeatureOverrides(setName, base)` applies on top:
`result = { ...base, ...globalOverrides, ...setOverrides[setName] }`.

### Article Features (`buildArticleFeatures`) — set name: `article`

```json
{
  "rweb_video_screen_enabled": true,
  "profile_label_improvements_pcf_label_in_post_enabled": true,
  "responsive_web_profile_redirect_enabled": true,
  "rweb_tipjar_consumption_enabled": true,
  "verified_phone_label_enabled": false,
  "creator_subscriptions_tweet_preview_api_enabled": true,
  "responsive_web_graphql_timeline_navigation_enabled": true,
  "responsive_web_graphql_exclude_directive_enabled": true,
  "responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
  "premium_content_api_read_enabled": false,
  "communities_web_enable_tweet_community_results_fetch": true,
  "c9s_tweet_anatomy_moderator_badge_enabled": true,
  "responsive_web_grok_analyze_button_fetch_trends_enabled": false,
  "responsive_web_grok_analyze_post_followups_enabled": false,
  "responsive_web_grok_annotations_enabled": false,
  "responsive_web_jetfuel_frame": true,
  "post_ctas_fetch_enabled": true,
  "responsive_web_grok_share_attachment_enabled": true,
  "articles_preview_enabled": true,
  "responsive_web_edit_tweet_api_enabled": true,
  "graphql_is_translatable_rweb_tweet_is_translatable_enabled": true,
  "view_counts_everywhere_api_enabled": true,
  "longform_notetweets_consumption_enabled": true,
  "responsive_web_twitter_article_tweet_consumption_enabled": true,
  "tweet_awards_web_tipping_enabled": false,
  "responsive_web_grok_show_grok_translated_post": false,
  "responsive_web_grok_analysis_button_from_backend": true,
  "creator_subscriptions_quote_tweet_preview_enabled": false,
  "freedom_of_speech_not_reach_fetch_enabled": true,
  "standardized_nudges_misinfo": true,
  "tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
  "longform_notetweets_rich_text_read_enabled": true,
  "longform_notetweets_inline_media_enabled": true,
  "responsive_web_grok_image_annotation_enabled": true,
  "responsive_web_grok_imagine_annotation_enabled": true,
  "responsive_web_grok_community_note_auto_translation_is_enabled": false,
  "responsive_web_enhance_cards_enabled": false
}
```

### TweetDetail Features (`buildTweetDetailFeatures`) — set name: `tweetDetail`

Spreads `buildArticleFeatures()` plus adds 3 flags:
```
"responsive_web_twitter_article_plain_text_enabled": true
"responsive_web_twitter_article_seed_tweet_detail_enabled": true
"responsive_web_twitter_article_seed_tweet_summary_enabled": true
```

`fetchTweetDetail` also adds these on top before sending (net new: `articles_rest_api_enabled`, `rweb_video_timestamps_enabled`):
```json
{
  "articles_rest_api_enabled": true,
  "rweb_video_timestamps_enabled": true
}
```

### Article Field Toggles (`buildArticleFieldToggles`)

```json
{
  "withPayments": false,
  "withAuxiliaryUserLabels": false,
  "withArticleRichContentState": true,
  "withArticlePlainText": true,
  "withGrokAnalyze": false,
  "withDisallowedReplyControls": false
}
```

UserTweets fieldToggles (separate, different):
```json
{ "withArticlePlainText": false }
```

UserByScreenName fieldToggles (correction #80):
```json
{ "withAuxiliaryUserLabels": false }
```

### Search Features (`buildSearchFeatures`) — set name: `search`

Same as article plus `"rweb_video_timestamps_enabled": true`, minus `articles_preview_enabled` (no — actually it is included in search too). The key unique addition is `"rweb_video_timestamps_enabled": true`.

### Tweet Create Features (`buildTweetCreateFeatures`) — set name: `tweetCreate`

Notable difference: `"responsive_web_profile_redirect_enabled": false` (not true like article/search). Does not include `rweb_video_timestamps_enabled`.

### Timeline Features (`buildTimelineFeatures`) — set name: `timeline`

Spreads `buildSearchFeatures()` plus adds:
```
"blue_business_profile_image_shape_enabled": true
"responsive_web_text_conversations_enabled": false
"tweetypie_unmention_optimization_enabled": true
"vibe_api_enabled": true
"responsive_web_twitter_blue_verified_badge_is_enabled": true
"interactive_text_enabled": true
"longform_notetweets_richtext_consumption_enabled": true
"responsive_web_media_download_video_enabled": false
```

### Bookmarks Features (`buildBookmarksFeatures`) — set name: `bookmarks`

Spreads `buildTimelineFeatures()` plus:
```
"graphql_timeline_v2_bookmark_timeline": true
```

### Likes Features (`buildLikesFeatures`) — set name: `likes`

Identical to `buildTimelineFeatures()` with no additions.

### Lists Features (`buildListsFeatures`) — set name: `lists`

Hardcoded directly (does NOT spread search/timeline). Notable differences:
- `blue_business_profile_image_shape_enabled`: false (bundled features.json overrides to true)
- `vibe_api_enabled`: false (bundled overrides to true)
- `interactive_text_enabled`: false (bundled overrides to true)
- `responsive_web_twitter_blue_verified_badge_is_enabled`: absent (bundled adds as true)
- Does NOT include `rweb_video_timestamps_enabled`
- Does NOT include `responsive_web_graphql_exclude_directive_enabled`

### Home Timeline Features (`buildHomeTimelineFeatures`) — set name: `homeTimeline`

Identical to `buildTimelineFeatures()` with no additions.

### User Tweets Features (`buildUserTweetsFeatures`) — set name: `userTweets`

Hardcoded directly. Key differences from search/timeline:
- `rweb_video_screen_enabled`: false
- `responsive_web_profile_redirect_enabled`: false
- `responsive_web_grok_analyze_post_followups_enabled`: true
- `responsive_web_grok_show_grok_translated_post`: true
- Does NOT include `responsive_web_graphql_exclude_directive_enabled`
- Does NOT include `rweb_video_timestamps_enabled`

### Following/Followers Features (`buildFollowingFeatures`) — set name: `following`

Hardcoded directly. Key differences:
- `profile_label_improvements_pcf_label_in_post_enabled`: false
- `premium_content_api_read_enabled`: true
- `responsive_web_jetfuel_frame`: false
- `responsive_web_grok_share_attachment_enabled`: false
- `tweet_awards_web_tipping_enabled`: true
- `responsive_web_grok_analysis_button_from_backend`: false
- `responsive_web_grok_image_annotation_enabled`: false
- `responsive_web_grok_imagine_annotation_enabled`: false
- Does NOT include `responsive_web_graphql_exclude_directive_enabled`
- Does NOT include `rweb_video_timestamps_enabled`

### Explore Features (`buildExploreFeatures`) — set name: `explore`

Used for `GenericTimelineById` (news/trending). Key additions vs search:
- `responsive_web_grok_analyze_button_fetch_trends_enabled`: true
- `responsive_web_grok_analyze_post_followups_enabled`: true
- `responsive_web_grok_annotations_enabled`: true
- `responsive_web_grok_show_grok_translated_post`: true
- `responsive_web_grok_community_note_auto_translation_is_enabled`: true

### Bundled features.json (applied first in override chain)

```json
{
  "global": {
    "responsive_web_grok_annotations_enabled": false,
    "post_ctas_fetch_enabled": true,
    "responsive_web_graphql_exclude_directive_enabled": true
  },
  "sets": {
    "lists": {
      "blue_business_profile_image_shape_enabled": true,
      "tweetypie_unmention_optimization_enabled": true,
      "responsive_web_text_conversations_enabled": false,
      "interactive_text_enabled": true,
      "vibe_api_enabled": true,
      "responsive_web_twitter_blue_verified_badge_is_enabled": true
    }
  }
}
```

---

## 10. Exact Request Specifications

### TweetDetail Variables (correction #28)
```json
{
  "focalTweetId": "<tweetId>",
  "with_rux_injections": false,
  "rankingMode": "Relevance",
  "includePromotedContent": true,
  "withCommunity": true,
  "withQuickPromoteEligibilityTweetFields": true,
  "withBirdwatchNotes": true,
  "withVoice": true
}
```
No `referrer`, no `count`. `cursor` added only when paginating.

### HomeTimeline / HomeLatestTimeline Variables (corrections #12, #61)
```json
{
  "count": 20,
  "includePromotedContent": true,
  "latestControlAvailable": true,
  "requestContext": "launch",
  "withCommunity": true
}
```
`cursor` added only when paginating.

### Bookmarks Variables (correction #22)
```json
{
  "count": 20,
  "includePromotedContent": false,
  "withDownvotePerspective": false,
  "withReactionsMetadata": false,
  "withReactionsPerspective": false
}
```
`cursor` added only when paginating. Variables built once outside the per-queryId loop (correction #85).

### BookmarkFolderTimeline Variables
```json
{
  "bookmark_collection_id": "<folderId>",
  "includePromotedContent": true,
  "count": 20
}
```
`cursor` only when paginating. Retry without `count` on `Variable "$count"` error.

### Likes Variables (correction #49)
```json
{
  "userId": "<currentUserId>",
  "count": 20,
  "includePromotedContent": false,
  "withClientEventToken": false,
  "withBirdwatchNotes": false,
  "withVoice": true
}
```
No `withV2Timeline` (correction #29).

### UserTweets Variables (corrections #13, #29)
```json
{
  "userId": "<userId>",
  "count": 20,
  "includePromotedContent": false,
  "withQuickPromoteEligibilityTweetFields": true,
  "withVoice": true
}
```
No `withV2Timeline`. fieldToggles: `{"withArticlePlainText":false}`.

### Following/Followers Variables (correction #82)
```json
{
  "userId": "<userId>",
  "count": 20,
  "includePromotedContent": false
}
```

### ListOwnerships Variables (correction #30)
```json
{
  "userId": "<currentUserId>",
  "count": 100,
  "isListMembershipShown": true,
  "isListMemberTargetUserId": "<currentUserId>"
}
```
No cursor even for pagination.

### ListLatestTweetsTimeline Variables (correction #84)
```json
{
  "listId": "<listId>",
  "count": 20
}
```
`cursor` only when paginating. No extra fields.

### GenericTimelineById Variables (corrections #34, #67)
```json
{
  "timelineId": "<tab-timeline-id>",
  "count": "<maxCount * 2>",
  "includePromotedContent": false
}
```

### SearchTimeline (correction #35)
URL: `POST /graphql/<queryId>/SearchTimeline?variables=<url-encoded-json>`
URL variables:
```json
{
  "rawQuery": "<query>",
  "count": 20,
  "querySource": "typed_query",
  "product": "Latest"
}
```
POST body:
```json
{ "features": { }, "queryId": "<queryId>" }
```

### UserByScreenName Variables (correction #63)
```json
{ "screen_name": "<handle>", "withSafetyModeUserFields": true }
```

### AboutAccountQuery Variables (correction #2)
```json
{ "screenName": "<handle>" }
```

### CreateTweet Body
```json
{
  "variables": {
    "tweet_text": "<text>",
    "dark_request": false,
    "media": {
      "media_entities": [],
      "possibly_sensitive": false
    },
    "semantic_annotation_ids": []
  },
  "features": { },
  "queryId": "<queryId>"
}
```
For reply, add `"reply": { "in_reply_to_tweet_id": "<id>", "exclude_reply_user_ids": [] }` to variables.
Response path: `data.create_tweet.tweet_results.result.rest_id` (correction #38).
Extra header: `referer: https://x.com/compose/post`.

### DeleteRetweet Body (correction #3)
```json
{
  "variables": {
    "tweet_id": "<retweetId>",
    "source_tweet_id": "<retweetId>"
  },
  "queryId": "<queryId>"
}
```
No `dark_request`. No `features`.

### Engagement Mutations Body (FavoriteTweet, UnfavoriteTweet, CreateRetweet, CreateBookmark, DeleteBookmark)
```json
{
  "variables": { "tweet_id": "<tweetId>" },
  "queryId": "<queryId>"
}
```
No `features`.
Extra header: `referer: https://x.com/i/status/<tweetId>`.

### Follow/Unfollow REST (correction #4)
```
POST https://x.com/i/api/1.1/friendships/create.json
Content-Type: application/x-www-form-urlencoded
Body: user_id=<userId>&skip_status=true
```
Fallback: `api.twitter.com` mirror.
GraphQL CreateFriendship body: `{ "variables": { "user_id": "<userId>" }, "queryId": "<queryId>" }` — no `features`.

### statuses/update.json Fallback (correction #39)
```
POST https://x.com/i/api/1.1/statuses/update.json
Content-Type: application/x-www-form-urlencoded
Headers: getBaseHeaders() + content-type override
Body: status=<text>[&in_reply_to_status_id=<id>&auto_populate_reply_metadata=true][&media_ids=<csv>]
```
Response: `id_str` first, then `String(id)` (correction #40).

---

## 11. Pagination Specifications

### Pattern 1: Generic `paginateCursor` (replies and threads ONLY)

```
loop:
  1. if pagesFetched > 0 AND pageDelayMs > 0 → sleep(pageDelayMs)  ← delay BEFORE fetch
  2. page = fetchPage(cursor)
  3. if !page.success:
       if len(items) > 0 → return { success:false, error:page.error, items, nextCursor:cursor }
       else → return raw page failure object  ← correction #83
  4. pagesFetched++
  5. deduplicate items by getKey()
  6. pageCursor = page.cursor
  7. if !pageCursor || pageCursor == cursor → return { success:true, items, nextCursor:undefined }
  8. if maxPages set && pagesFetched >= maxPages → return { success:true, items, nextCursor:pageCursor }
  9. cursor = pageCursor → continue
```

STOP CONDITIONS: empty cursor OR unchanged cursor ONLY. Zero items does NOT stop. Zero additions does NOT stop.

### Pattern 2: Inline Loops (all other paginated methods)

```
stop when ANY of:
  - !nextCursor
  - nextCursor == cursor
  - page.tweets.length == 0
  - added == 0 (all items already seen)
  - maxPages reached
  - count/limit reached
```

Page delay only in `getUserTweetsPaged` (before pages after first), not in others.

### Page Delay Default: 1000ms (for replies, thread, userTweets)

### UserTweets Hard Cap: 10 pages

`hardMaxPages = min(10, maxPages ?? ceil(limit/20))`. If caller supplies `maxPages`, use that (still capped at 10).

---

## 12. Retry and Fallback Specifications

### fetchWithRetry (bookmarks and bookmark folder timeline ONLY)

```
maxRetries = 2 → 3 total attempts: 0, 1, 2
baseDelayMs = 500
retryable: 429, 500, 502, 503, 504

For attempt in 0..2:
  response = fetch(url)
  if response.status not in retryable OR attempt == maxRetries:
    return response
  retryAfter = response.headers["retry-after"]
  if retryAfter is valid integer:
    delay = retryAfter * 1000
  else:
    delay = baseDelayMs * 2^attempt + random(0, baseDelayMs)
  sleep(delay)
  (loop continues)
```

Post-loop return is dead code (correction #86).

### 404 Refresh Pattern (universal for most operations)

```
for each queryId in getXxxQueryIDs():
  result = attempt(queryId)
  if result.had404: continue to next queryId
  else: return result
if had404:
  refreshQueryIDs()
  for each queryId in getXxxQueryIDs():
    result = attempt(queryId)
    if result.had404: continue
    else: return result
return lastResult
```

### TweetDetail GET→POST Pattern (correction #36)

```
for each queryId in getTweetDetailQueryIDs():
  response = GET /graphql/<queryId>/TweetDetail?...
  if response.status == 404:
    response = POST /graphql/<queryId>/TweetDetail { variables, features, queryId }
    if response.status == 404:
      had404 = true; continue to next queryId
  parse and return response
if had404: refreshQueryIDs() then retry entire loop once
```

### CreateTweet Fallback Chain (correction #39, #72)

```
1. POST /graphql/<queryId>/CreateTweet
2. On 404: refreshQueryIDs(), POST /graphql/<newQueryId>/CreateTweet
3. On second 404: POST https://x.com/i/api/graphql { variables, features, queryId }
4. On any response with errors[].code == 226: POST statuses/update.json (form-encoded)
```

### withRefreshedQueryIDsOn404

```go
func (c *Client) withRefreshedQueryIDsOn404(ctx context.Context, attempt func() attemptResult) (result, refreshed bool) {
    r := attempt()
    if r.success || !r.had404 {
        return r, false
    }
    c.refreshQueryIDs(ctx)
    return attempt(), true
}
```

Used by: `getFollowing`, `getFollowers`, `getUserAboutAccount`. `refreshed` flag gates REST fallback.

### Following/Followers REST Fallback (correction #14)

Only attempted when `refreshed == true` after 404-triggered refresh:
```
GET https://x.com/i/api/1.1/friends/list.json?user_id=<id>&count=<n>&skip_status=true&include_user_entities=false[&cursor=<cursor>]
GET https://api.twitter.com/1.1/friends/list.json (same params)
```

### SearchTimeline Refresh Triggers (corrections #35, #76)

Refresh triggered by:
- HTTP 404
- HTTP 400 or 422 where body contains `GRAPHQL_VALIDATION_FAILED`
- JSON response where `errors[].extensions.code == "GRAPHQL_VALIDATION_FAILED"`
- JSON response where `errors[].path` includes `"rawQuery"` AND message matches `/must be defined/i`

### HomeTimeline Refresh Trigger (correction #77)

Refresh triggered by `errors[].message` matching `/query:\s*unspecified/i` (case-insensitive regex).

### Likes Refresh Trigger (correction #51)

After exhausting all query IDs, if accumulated error string `.includes("Query: Unspecified")` (exact case), refresh and retry.

---

## 13. Implementation Phases

### Phase 0 — Bootstrap and Tooling

Deliverables:
- `go.mod`: `module github.com/mudrii/gobird`, `go 1.23`
- Directory structure per §5
- `Makefile` with targets: `build`, `test`, `test-race`, `lint`, `vet`, `fmt`, `clean`, `coverage`, `ci`
- Build uses ldflags: `-X main.version=$(VERSION) -X main.gitSHA=$(GIT_SHA)`
- `.golangci.yml` configured with `errcheck`, `govet`, `staticcheck`, `unused`, `revive`
- `.github/workflows/ci.yml`: checkout → setup-go 1.23 → mod tidy → mod verify → test → race test → vet → lint → build
- Dependencies: `github.com/spf13/cobra`, `github.com/google/uuid`, `github.com/tailscale/hujson`, `modernc.org/sqlite`

Tests first:
- `go test ./...` smoke
- CLI root help smoke test

Exit: clean build; CI green on empty skeleton.

---

### Phase 1 — Contracts, Types, and Error Taxonomy

Deliverables: `internal/types/{wire,models,results,options}.go`

#### Normalized Models (`models.go`)

```go
type TweetData struct {
    ID               string
    Text             string
    CreatedAt        string
    ReplyCount       int
    RetweetCount     int
    LikeCount        int
    ConversationID   string
    InReplyToStatusID *string
    Author           TweetAuthor
    AuthorID         string
    QuotedTweet      *TweetData
    Media            []TweetMedia
    Article          *TweetArticle
    Raw              any  // only when includeRaw=true
}

type TweetAuthor struct {
    Username string
    Name     string
}

type TweetMedia struct {
    Type       string  // "photo", "video", "animated_gif"
    URL        string
    Width      int
    Height     int
    PreviewURL string  // set for ANY media with sizes.small (correction #31)
    VideoURL   string  // only for video/gif
    DurationMs *int
}

type TweetArticle struct {
    Title       string
    PreviewText string
}

type TweetWithMeta struct {
    TweetData
    IsThread        bool
    ThreadPosition  string  // "standalone","root","middle","end"
    HasSelfReplies  bool
    ThreadRootID    *string
}

type TwitterUser struct {
    ID              string
    Username        string
    Name            string
    Description     string
    FollowersCount  int
    FollowingCount  int
    IsBlueVerified  bool
    ProfileImageURL string
    CreatedAt       string
}

type TwitterList struct {
    ID              string
    Name            string
    Description     string
    MemberCount     int
    SubscriberCount int
    IsPrivate       bool
    CreatedAt       string
    Owner           *ListOwner
}

type ListOwner struct {
    ID       string
    Username string
    Name     string
}

type NewsItem struct {
    ID          string
    Headline    string
    Category    string
    TimeAgo     string
    PostCount   *int
    Description string
    URL         string
    IsAiNews    bool
    Raw         any  // only when includeRaw
}

type CurrentUserResult struct {
    ID       string
    Username string
    Name     string
}

type TwitterCookies struct {
    AuthToken    string
    Ct0          string
    CookieHeader string
    Source       string
}

type AboutAccountProfile struct {
    AccountBasedIn       string
    Source               string
    CreatedCountryAccurate bool
    LocationAccurate     bool
    LearnMoreURL         string
}

type UploadMediaResult struct {
    MediaID string
}
```

#### Result Types (`results.go`)

```go
type GetTweetResult struct {
    Success bool
    Tweet   *TweetData
    Error   string
}

type SearchResult struct {
    Success    bool
    Tweets     []TweetData
    NextCursor string
    Error      string
}

type TweetResult struct {
    Success bool
    TweetID string
    Error   string
}

type MutationResult struct {
    Success bool
    Error   string
}

type FollowMutationResult struct {
    Success  bool
    UserID   string
    Username string
    Error    string
}

type FollowingResult struct {
    Success    bool
    Users      []TwitterUser
    NextCursor string
    Error      string
}

type UserLookupResult struct {
    Success  bool
    UserID   string
    Username string
    Name     string
    Error    string
}

type ListsResult struct {
    Success bool
    Lists   []TwitterList
    Error   string
}

type NewsResult struct {
    Success bool
    Items   []NewsItem
    Error   string
}

type AboutAccountResult struct {
    Success  bool
    Profile  *AboutAccountProfile
    Error    string
}

type CursorPaginationResult[T any] struct {
    Success    bool
    Items      []T
    NextCursor string
    Error      string
}
```

#### Error taxonomy (sentinel errors):
- `ErrMissingCredentials` — exit 1
- `ErrAuthFailure` — exit 1
- `ErrNetworkFailure` — exit 1
- `ErrStaleQueryIDs` — exit 1
- `ErrParseFailure` — exit 1
- `ErrPartialPagination` — exit 1 (items accumulated + error)
- `ErrInvalidUsage` — exit 2
- `ErrInvalidHandle` — exit 2

Tests first: JSON marshal tests for all models; error classification tests.

---

### Phase 2 — Config, Env, and Input Normalization

Deliverables: `internal/config/config.go`, `internal/cli/input.go`

#### Config files (JSON5 via hujson):
- `~/.config/bird/config.json5` (global)
- `./.birdrc.json5` (local, overrides global)
- Permissions check: warn if not 0600

#### Config schema:
```json5
{
  cookieSource: ["safari","chrome","firefox"],
  chromeProfile: "Default",
  chromeProfileDir: "/path",
  firefoxProfile: "default-release",
  cookieTimeoutMs: 30000,
  timeoutMs: 20000,
  quoteDepth: 1
}
```

#### Env var resolution precedence (highest to lowest):
1. CLI flags
2. `AUTH_TOKEN` or `TWITTER_AUTH_TOKEN` (correction #41)
3. `CT0` or `TWITTER_CT0`
4. `BIRD_TIMEOUT_MS`, `BIRD_COOKIE_TIMEOUT_MS`, `BIRD_QUOTE_DEPTH`
5. `BIRD_QUERY_IDS_CACHE` — query IDs cache path (correction #43)
6. `BIRD_FEATURES_CACHE` > `BIRD_FEATURES_PATH` — features file path (correction #44)
7. `BIRD_FEATURES_JSON` — inline JSON override
8. `NO_COLOR` — disables color (same as `--no-color`)
9. `BIRD_DEBUG_BOOKMARKS=1` — verbose bookmark retry logs
10. Config file values
11. Defaults

#### Input utilities:

`looksLikeTweetInput(value)`:
- URL regex: `/^(?:https?:\/\/)?(?:www\.)?(?:twitter\.com|x\.com)\/[^/]+\/status\/\d+/i`
- ID regex: `/^\d{8,}$/` (8+ digits, correction from cli-commands.md)
- Returns true if either matches

`extractTweetID(input)`:
- Regex: `/(?:twitter\.com|x\.com)\/(?:\w+\/status|i\/web\/status)\/(\d+)/i`
- Returns captured ID or `input` unchanged

`extractListID(input)`:
- URL regex: `/(?:twitter\.com|x\.com)\/i\/lists\/(\d+)/i`
- Bare ID regex: `/^\d{5,}$/`
- Returns string or nil

`extractBookmarkFolderID(input)`:
- URL regex: `/(?:twitter\.com|x\.com)\/i\/bookmarks\/(\d+)/i`
- Bare ID regex: `/^\d{5,}$/`
- Returns string or nil

`normalizeHandle(input)`:
- Strip leading `@`, trim
- Regex: `/^[A-Za-z0-9_]{1,15}$/`
- Returns nil on failure (correction #52)
- Max 15 chars; must match exactly

`mentionsQueryFromUserOption(userOption)`:
- nil input → `{ query: nil, error: nil }`
- valid → `{ query: "@<handle>", error: nil }`
- invalid → `{ query: nil, error: "Invalid --user handle..." }`

`resolveCliInvocation(args)`:
- Strip leading `--` separator
- If no args: return showHelp=true
- If no known command found, scan for `looksLikeTweetInput` and insert `"read"` before it
- Shorthand dispatch is done before Cobra parses

Tests first: config precedence, shorthand dispatch, all extractor table tests, handle normalization edge cases.

---

### Phase 3 — Credentials and Browser Cookie Resolution

Deliverables: `internal/auth/{cookies,resolve}.go`

#### Resolution order (correction #57):
1. CLI flags `--auth-token` / `--ct0`
2. Env: `AUTH_TOKEN` > `TWITTER_AUTH_TOKEN`; `CT0` > `TWITTER_CT0`
3. If both auth_token AND ct0 resolved: skip browser extraction entirely
4. Browser sources in configured order (default: safari → chrome → firefox)

#### Browser extraction:
- Query origins: `https://x.com/` and `https://twitter.com/`
- Domain preference: x.com > twitter.com > first match (correction #42)
- Cookie extraction timeout: `cookieTimeoutMs` (default 30000ms on macOS; nil on other platforms)

#### Chrome cookie extraction (macOS):
- Cookie DB: `~/Library/Application Support/Google/Chrome/<Profile>/Cookies` (SQLite)
- Key storage: macOS Keychain, service `Chrome Safe Storage`
- Decryption: AES-256-CBC with PBKDF2-derived key from Keychain secret
- Profile resolution: chromeProfileDir > chromeProfile > `Default`
- Driver: `modernc.org/sqlite` (pure Go, no CGo)

#### Safari cookie extraction (macOS):
- File: `~/Library/Cookies/Cookies.binarycookies` (proprietary binary format, NOT SQLite)
- Requires custom binary parser (see binary format spec in authentication.md)

#### Firefox cookie extraction (macOS):
- Cookie DB: `~/Library/Application Support/Firefox/Profiles/<profile>/cookies.sqlite`
- Plain SQLite, no decryption
- Profile resolution: firefoxProfile name > first profile found

#### Cookie header format:
```
auth_token=<token>; ct0=<ct0>
```

Source labels: `"CLI argument"`, `"env AUTH_TOKEN"`, `"Safari"`, `"Chrome default profile"`, `"Chrome profile \"<name>\""`, `"Firefox default profile"`, `"Firefox profile \"<name>\""`.

Tests first: source-order resolution, domain preference, cookie header construction, missing-credential classification.

---

### Phase 4 — Client Core, Headers, and HTTP Transport

Deliverables: `internal/client/{client,headers,http,constants}.go`

#### Client struct:
```go
type Client struct {
    AuthToken    string
    Ct0          string
    CookieHeader string
    UserAgent    string          // default: Chrome 131 macOS UA
    ClientUUID   string         // uuid.New().String()
    DeviceID     string         // uuid.New().String()
    UserID       string         // set by getCurrentUser() or ensureClientUserID()
    TimeoutMs    int            // 0 = no timeout
    QuoteDepth   int            // default 1
    QueryIDs     *runtime.QueryIDStore
    httpClient   *http.Client
    features     *runtime.FeatureOverrides
}
```

Validation: fail at construction if authToken or ct0 is empty.

#### `baseHeaders()` (generated per-request):
```
accept:                    */*
accept-language:           en-US,en;q=0.9
authorization:             Bearer <BearerToken>
x-csrf-token:              <ct0>
x-twitter-auth-type:       OAuth2Session
x-twitter-active-user:     yes
x-twitter-client-language: en
x-client-uuid:             <ClientUUID>
x-twitter-client-deviceid: <DeviceID>
x-client-transaction-id:   <16 random bytes hex>  ← per-request
cookie:                    <CookieHeader>
user-agent:                <UserAgent>
origin:                    https://x.com
referer:                   https://x.com/
```
`x-twitter-client-user-id: <UserID>` added only when `UserID != ""`

#### `jsonHeaders()` = `baseHeaders()` + `content-type: application/json`
#### `getHeaders()` = alias for `jsonHeaders()` (correction #70)
#### `uploadHeaders()` = `baseHeaders()` only — NO content-type override (correction #70)

#### `createTransactionID()`:
```go
func createTransactionID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b) // 32-char lowercase hex
}
```

#### `fetchWithTimeout(ctx, url, init)`:
- If `TimeoutMs > 0`: use `context.WithTimeout`
- Catch all errors and return `{success:false, error:err.Error()}`
- Non-2xx: return `{success:false, error:"HTTP <status>: <first 200 chars>"}`

Tests first: header snapshots, timeout propagation, request construction, secret-redaction in error messages.

---

### Phase 5 — Runtime Query IDs and Feature Overrides

Deliverables: `internal/runtime/{query_id_store,query_ids,features,overrides}.go`

#### QueryIDStore behavior:
- Default cache: `~/.config/bird/query-ids-cache.json` (override: `BIRD_QUERY_IDS_CACHE`)
- TTL: `86400000` ms (24 hours)
- Memory cache loaded lazily on first `getQueryID()` call
- Concurrent `refresh()` calls coalesce to same in-flight promise
- `clearMemory()` resets both memory snapshot and load-once guard
- In `NODE_ENV=test` (or Go equivalent `GOBIRD_TEST=1`): skip `refreshQueryIDs()`

#### `getQueryID(operationName) string`:
1. Check runtime memory/disk cache
2. Fall back to `QUERY_IDS[operationName]` = merged FALLBACK + bundled baseline

#### Cache schema (JSON):
```json
{
  "fetchedAt": "2026-01-19T12:00:00.000Z",
  "ttlMs": 86400000,
  "ids": { "CreateTweet": "<id>", ... },
  "discovery": {
    "pages": ["https://x.com/?lang=en", "..."],
    "bundles": ["loader.abc123.js", "..."]
  }
}
```
`discovery.bundles` stores filename only (last path segment).

#### Refresh process:
1. Scrape 4 pages: `https://x.com/?lang=en`, `/explore`, `/notifications`, `/settings/profile`
2. Extract bundle URLs matching: `https://abs.twimg.com/responsive-web/client-web(-legacy)?/[A-Za-z0-9.-]+\.js`
3. If no bundles: error `"No client bundles discovered; x.com layout may have changed."`
4. Fetch bundles parallel, concurrency 6; stop early when all 29 targets found
5. Apply 4 regex patterns per bundle:
   - `e.exports={queryId:"<id>",operationName:"<name>"`
   - `e.exports={operationName:"<name>",queryId:"<id>"`
   - `operationName[:"=]"<name>"` ... ≤4000 chars ... `queryId[:"=]"<id>"`
   - `queryId[:"=]"<id>"` ... ≤4000 chars ... `operationName[:"=]"<name>"`
6. Keep only operations in `TARGET_QUERY_ID_OPERATIONS` (all 29 keys of FALLBACK_QUERY_IDS)
7. Validate queryId: `/^[a-zA-Z0-9_-]+$/`
8. First match per operation wins
9. Write snapshot to disk, update memory

#### `query-ids` CLI command refresh targets (only 10, not 29):
`CreateTweet`, `CreateRetweet`, `FavoriteTweet`, `TweetDetail`, `SearchTimeline`, `UserArticlesTweets`, `Bookmarks`, `Following`, `Followers`, `Likes`

#### Feature Override system:
Load order (later wins):
1. Bundled `features.json` (embedded in binary)
2. File at resolved cache path (`BIRD_FEATURES_CACHE` > `BIRD_FEATURES_PATH` > `~/.config/bird/features.json`)
3. `BIRD_FEATURES_JSON` env var (JSON string, correction #44)

`applyFeatureOverrides(setName, base) map[string]bool`:
```
{ ...base, ...globalOverrides, ...setOverrides[setName] }
```

Tests first: query ID resolution order, TTL freshness, refresh trigger conditions, feature override precedence, serialized feature map snapshot tests.

---

### Phase 6 — Parsing Engine and Normalization

Deliverables: `internal/parsing/*.go`

#### `extractCursorFromInstructions(instructions, cursorType="Bottom")`:
```
for each instruction in instructions:
  for each entry in instruction.entries:
    if entry.content.cursorType == cursorType AND entry.content.value != "":
      return entry.content.value
return ""
```
ONLY checks `entry.content.cursorType`. Does NOT check `entryType`. Does NOT check module items (correction #7).

#### `collectTweetResultsFromEntry(entry)` — exactly 5 paths (correction #26):
1. `entry.content.itemContent.tweet_results.result`
2. `entry.content.item.itemContent.tweet_results.result`
3. For each `item` in `entry.content.items`:
   a. `item.item.itemContent.tweet_results.result`
   b. `item.itemContent.tweet_results.result`
   c. `item.content.itemContent.tweet_results.result`
Result only added if `result.rest_id != ""`

#### `unwrapTweetResult(result)`:
- If `result.tweet` is present (truthy): return `result.tweet`
- Otherwise: return `result`
No `__typename` check (correction #25).

#### `parseTweetsFromInstructions(instructions, opts)`:
- Iterates all instructions' `.entries` arrays (no instruction type filter)
- For each entry: `collectTweetResultsFromEntry` → `mapTweetResult`
- Deduplicates by tweet ID (first occurrence wins)

#### `mapTweetResult(result, opts)`:
Field mapping:
- `id`: `result.rest_id`
- `text`: `extractTweetText(result)` — article first, note tweet, legacy (correction #19)
- `createdAt`: `result.legacy.created_at`
- `replyCount`: `result.legacy.reply_count`
- `retweetCount`: `result.legacy.retweet_count`
- `likeCount`: `result.legacy.favorite_count`
- `conversationId`: `result.legacy.conversation_id_str`
- `inReplyToStatusId`: `result.legacy.in_reply_to_status_id_str` (nil when null)
- `author.username`: `result.core.user_results.result.legacy.screen_name` → `.core.screen_name` (correction #79)
- `author.name`: `.legacy.name` → `.core.name` → username
- `authorId`: `result.core.user_results.result.rest_id`
- `isBlueVerified`: `result.is_blue_verified` (top-level, NOT in legacy; correction #8)
- `quotedTweet`: recursive call on `unwrapTweetResult(result.quoted_status_result.result)` when quoteDepth > 0
- `media`: `extractMedia(result)`
- `article`: `extractArticleMetadata(result)`
- `_raw`: `result` when includeRaw=true

Discard result if `rest_id` empty, `username` empty, or text extraction yields empty.

#### `extractTweetText(result)` priority (correction #19):
1. `extractArticleText(result)`
2. `extractNoteTweetText(result)`
3. `firstText(result.legacy.full_text)`

#### `extractArticleText(result)` (correction #54):
1. Check `result.article` — absent → nil
2. `articleResult = article.article_results?.result ?? article`
3. `title` = first non-empty of `articleResult.title`, `article.title`
4. **Rich path first**: render `article.article_results.result.content_state` via `renderContentState`
5. If rich path yields output: prepend title if not already present
6. **Plain text fallback**: try many paths: `articleResult.plain_text`, `article.plain_text`, `articleResult.body.text`, ... (19 paths total)
7. If body == title exactly: clear body
8. **Recursive text collection fallback**: walk entire object tree collecting `text`/`title` keys
9. Combine: `title\n\nbody` if both set and body doesn't start with title; else `body ?? title`

#### `extractNoteTweetText(result)`:
`note = result.note_tweet.note_tweet_results.result`
Try: `.text`, `.richtext.text`, `.rich_text.text`, `.content.text`, `.content.richtext.text`, `.content.rich_text.text`

#### `renderContentState(contentState)`:
- Returns nil if no blocks
- Entity map: supports both array `[{key,value}]` and object `{"0":{...}}` formats; keys parsed as int
- Block types: `unstyled`, `header-one/two/three`, `unordered-list-item`, `ordered-list-item`, `blockquote`, `atomic`
- Ordered list counter resets when previous block type was `ordered-list-item` and current is different
- Blocks joined with `\n\n` (correction #53); result trimmed; nil if empty

#### `renderAtomicBlock(block, entityMap)`:
- Take first `block.entityRanges[0]`; look up entity
- `MARKDOWN` → `entity.data.markdown.trim()` (correction #20)
- `DIVIDER` → `"---"`
- `TWEET` → `"[Embedded Tweet: https://x.com/i/status/<tweetId>]"` (only if tweetId truthy)
- `LINK` → `"[Link: <url>]"` (only if url truthy)
- `IMAGE` → `"[Image]"` — NO URL (correction #20)
- Anything else → nil (block omitted)

#### `renderBlockText(block, entityMap)`:
- Filter entityRanges to type `LINK` with `entity.data.url`
- Sort by offset descending (reverse order to preserve string positions)
- Rewrite each range as `[linkText](url)`
- Trim result

#### `extractMedia(result)`:
- Source: `result.legacy.extended_entities.media` preferred, fallback `result.legacy.entities.media`
- Skip items missing `type` or `media_url_https`
- Per item:
  - `type`: item.type
  - `url`: item.media_url_https
  - `width`: `sizes.large.w` → `sizes.medium.w` (correction #32)
  - `height`: `sizes.large.h` → `sizes.medium.h`
  - `previewUrl`: `media_url_https + ":small"` when `sizes.small` exists — for ANY media type (correction #31)
  - `videoUrl`: best MP4 variant by highest numeric bitrate; fallback to first MP4 if no numeric bitrates
  - `durationMs`: `video_info.duration_millis` when type is number

#### `parseUsersFromInstructions(instructions)`:
- Path: `entry.content.itemContent.user_results.result`
- Unwrap `UserWithVisibilityResults` via `result.user` when `__typename == "UserWithVisibilityResults"`
- Skip unless `userResult.__typename == "User"` after unwrapping
- `isBlueVerified`: `userResult.is_blue_verified` (top-level; correction #8)
- `profileImageUrl`: `legacy.profile_image_url_https` → `avatar.image_url`
- `createdAt`: `legacy.created_at` → `core.created_at`

#### Thread Filters (exported, from `thread_filters.go`):

`filterAuthorChain(tweets, bookmarkedTweet)`:
1. Map tweets by ID
2. Walk backwards via `inReplyToStatusId` collecting same-author chain
3. Forward expansion: repeatedly add tweets by same author replying to chain IDs (fixed-point)
4. Filter + sort ascending by `createdAt`

`filterAuthorOnly(tweets, bookmarkedTweet)`:
Filter by matching `author.username`. No sort.

`filterFullChain(tweets, bookmarkedTweet, options)`:
1. Map tweets by ID; map replies by parent ID
2. Seed with bookmarkedTweet.id
3. Walk backwards collecting all ancestors
4. BFS expanding all descendants from bookmarkedTweet
5. If `includeAncestorBranches`: expand branches from each ancestor
6. Filter + sort ascending by `createdAt`

`addThreadMetadata(tweet, allConversationTweets)`:
- `hasSelfReplies`: any tweet in set has `inReplyToStatusId == tweet.id` AND same author
- `isRoot`: `!tweet.inReplyToStatusId`
- `isThread`: `hasSelfReplies || !isRoot`
- `threadPosition`: standalone/root/middle/end per matrix
- `threadRootId`: `tweet.conversationId ?? nil`

Tests first: parser fixture tests per timeline family; article rendering tests; cursor extraction tests; media selection tests; thread filter tests.

---

### Phase 7 — Current User and User Lookup

Deliverables: `internal/client/{users,user_lookup}.go`

#### `getCurrentUser(ctx)`:
Phase 1 — try in order:
1. `GET https://x.com/i/api/account/settings.json`
2. `GET https://api.twitter.com/1.1/account/settings.json`
3. `GET https://x.com/i/api/account/verify_credentials.json?skip_status=true&include_entities=false`
4. `GET https://api.twitter.com/1.1/account/verify_credentials.json?skip_status=true&include_entities=false`

Parse per response (with typeof-string guard; correction #78):
- username: `data.screen_name`, then `data.user.screen_name`
- userId: `data.user_id` (string), `data.user_id_str` (string), `data.user.id_str` (string), `data.user.id` (string only — reject numeric)
- name: `data.name`, `data.user.name`, else `username`

Both username AND userId must be non-empty for success. Does NOT call UserByScreenName (correction #6).

Phase 2 — HTML fallback (if all Phase 1 fail):
- `GET https://x.com/settings/account` — cookie + user-agent headers ONLY
- `GET https://twitter.com/settings/account`
- Regexes: `/"screen_name":"([^"]+)"/`, `/"user_id"\s*:\s*"(\d+)"/`, `/"name":"([^"\\]*(?:\\.[^"\\]*)*)"/`

On success: set `client.UserID = userId`.

#### `getUserIDByUsername(ctx, handle)`:
1. `normalizeHandle(handle)` → nil means exit with error
2. Try 3 hardcoded UserByScreenName GraphQL IDs in sequence:
   `xc8f1g7BYqr6VTzTbvNlGw`, `qW5u-DAuXpMEG0zA1F7UGQ`, `sLVLhk0bGj3MVFEKTdax1w`
   - On `__typename == "UserUnavailable"`: return error immediately, NO REST fallback
   - On any non-ok response: try next ID
3. If all 3 fail and error doesn't include "not found or unavailable":
   - `GET https://x.com/i/api/1.1/users/show.json?screen_name=<handle>` (correction #55)
   - `GET https://api.twitter.com/1.1/users/show.json?screen_name=<handle>`

Request params (correction #63, #80):
- variables: `{ "screen_name": "<handle>", "withSafetyModeUserFields": true }`
- fieldToggles: `{ "withAuxiliaryUserLabels": false }`
- features: UserByScreenName-specific feature map

Response: `data.user.result.rest_id`, `.legacy.screen_name` → `.core.screen_name` (correction #79).

#### `getUserAboutAccount(ctx, handle)`:
- variables: `{ "screenName": "<handle>" }` (camelCase, correction #2)
- Response path: `data.user_result_by_screen_name.result.about_profile` (correction #1)
- Uses `withRefreshedQueryIDsOn404` pattern

Tests first: current-user state machine tests, HTML scrape fallback tests, UserByScreenName rotation tests, UserUnavailable short-circuit, about-account response path.

---

### Phase 8 — Core Read Operations

Deliverables: `internal/client/{tweet_detail,search,home}.go`

#### `fetchTweetDetail(ctx, tweetId, cursor, queryIds)` (correction #36):
```
for each queryId in getTweetDetailQueryIDs():
  response = GET /graphql/<queryId>/TweetDetail?variables=<json>&features=<json>&fieldToggles=<json>
  if response.status == 404:
    response = POST /graphql/<queryId>/TweetDetail { variables, features, queryId }
    if response.status == 404:
      had404 = true; continue
  parse response; return
if had404: refreshQueryIDs(); retry loop once
```

Variables (correction #28): `focalTweetId`, `with_rux_injections:false`, `rankingMode:"Relevance"`, `includePromotedContent:true`, `withCommunity:true`, `withQuickPromoteEligibilityTweetFields:true`, `withBirdwatchNotes:true`, `withVoice:true`; `cursor` only when paginating. No `referrer`, no `count`.

Features: `buildTweetDetailFeatures()` + `articles_rest_api_enabled:true` + `rweb_video_timestamps_enabled:true`.
FieldToggles: `buildArticleFieldToggles()` (POST body does NOT include fieldToggles).

Partial errors non-fatal: if `data.tweetResult.result` OR `data.threaded_conversation_with_injections_v2.instructions` is non-empty, ignore errors (correction #37).

Response paths (correction #58):
- Single tweet: `data.tweetResult.result`
- Thread: `data.threaded_conversation_with_injections_v2.instructions`
- Try single first; fallback to `findTweetInInstructions(instructions, tweetId)`.

`fetchUserArticlePlainText`: called ONLY when getTweet finds an article tweet AND article text extraction returns only title. Variables are the 17-field UserArticlesTweets set (correction #66).

#### `Search(ctx, query, opts)`:
- POST with variables in URL query string, features+queryId in body (correction #35)
- variables: `rawQuery`, `count:20`, `querySource:"typed_query"`, `product:"Latest"`, `cursor?`
- Response path: `data.search_by_raw_query.search_timeline.timeline.instructions` (correction #62)
- Refresh triggers: 404, HTTP 400/422 with GRAPHQL_VALIDATION_FAILED, JSON 200 with GRAPHQL_VALIDATION_FAILED, rawQuery-must-be-defined (correction #76)
- Uses 4-ID list: `[primary, "M1jEez78PEfVfbQLvlWMvQ", "5h0kNbk3ii97rmfY6CdgAA", "Tp1sewRU1AsZpBWhqCZicQ"]`

#### `GetHomeTimeline / GetHomeLatestTimeline(ctx, opts)`:
- Variables (corrections #12, #61): `count:20`, `includePromotedContent:true`, `latestControlAvailable:true`, `requestContext:"launch"`, `withCommunity:true`, `cursor?`
- Response path: `data.home.home_timeline_urt.instructions` (correction #60)
- Refresh trigger: `/query:\s*unspecified/i` regex on errors[].message (correction #77)
- Returns NO nextCursor to caller (correction #50)
- Uses 2-ID lists: HomeTimeline `[primary, "edseUwk9sP5Phz__9TIRnA"]`; HomeLatestTimeline `[primary, "iOEZpOdfekFsxSlPQCQtPg"]`

#### `paginateCursor` (generic paginator, for replies and threads only):
Exactly as described in §11 Pattern 1.

Tests first: TweetDetail happy path and GET→POST fallback, partial error handling, search refresh triggers, HomeTimeline request serialization, generic paginator exhaustion and error-with-items cases.

---

### Phase 9 — Timeline Families and Paginated Reads

Deliverables: `internal/client/{timelines,user_tweets,lists,news}.go`

#### Bookmarks:
- GET with variables (correction #22): `count:20`, `includePromotedContent:false`, `withDownvotePerspective:false`, `withReactionsMetadata:false`, `withReactionsPerspective:false`
- Variables built ONCE per page, not per queryId retry (correction #85)
- Response path: `data.bookmark_timeline_v2.timeline.instructions`
- 3-ID list: `[primary, "RV1g3b8n_SGOHwkqKYSCFw", "tmd4ifV8RHltzn8ymGg1aw"]`
- Uses `fetchWithRetry` (maxRetries=2, baseDelayMs=500; correction #16)
- Features: `buildBookmarksFeatures()`
- Inline paginator stop conditions (Pattern 2)
- Returns `nextCursor`

#### BookmarkFolderTimeline:
- Variables: `bookmark_collection_id`, `includePromotedContent:true`, `count:20`, `cursor?` (correction #23)
- Response path: `data.bookmark_collection_timeline.timeline.instructions` (correction #1)
- 2-ID list: `[primary, "KJIQpsvxrTfRIlbaRIySHQ"]`
- Retry without `count` on `Variable "$count"` error; return error on `Variable "$cursor"` error (corrections #11, #23)
- Also uses `fetchWithRetry`
- Supports thread filters and thread metadata when requested

#### Likes:
- GET with variables (correction #49): `userId`, `count:20`, `includePromotedContent:false`, `withClientEventToken:false`, `withBirdwatchNotes:false`, `withVoice:true`
- No `withV2Timeline` (correction #29)
- Response path: `data.user.result.timeline.timeline.instructions` (correction #1)
- 2-ID list: `[primary, "JR2gceKucIKcVNB_9JkhsA"]`
- Uses `fetchWithTimeout` (NOT `fetchWithRetry`; correction #49)
- Refresh trigger: `"Query: Unspecified"` exact string in errors[].message (correction #51)
- Returns `nextCursor`

#### UserTweets:
- Variables (correction #13, #29): `userId`, `count:20`, `includePromotedContent:false`, `withQuickPromoteEligibilityTweetFields:true`, `withVoice:true`, `cursor?`
- No `withV2Timeline` (correction #29)
- fieldToggles: `{"withArticlePlainText":false}`
- Response path: `data.user.result.timeline.timeline.instructions` (correction #1)
- 2-ID list: `[primary, "Wms1GvIiHXAPBaCr9KblaA"]`
- Hard max 10 pages (correction #13)
- Delay 1000ms before each page after first (inline paginator with delay)
- Fatal errors: `"User has been suspended"`, `"User not found"` (exact strings; correction #24)

#### Following/Followers:
- Variables (correction #82): `userId`, `count`, `includePromotedContent:false`, `cursor?`
- Response path: `data.user.result.timeline.timeline.instructions` (correction #1)
- 2-ID lists each: Following `[primary, "BEkNpEt5pNETESoqMsTEGA"]`; Followers `[primary, "kuFUYP9eV1FPoEy4N-pi7w"]`
- Uses `withRefreshedQueryIDsOn404`
- REST fallback only when `refreshed == true` (correction #14):
  - Following: `friends/list.json` x.com then api.twitter.com
  - Followers: `followers/list.json` x.com then api.twitter.com
  - Params: `user_id`, `count`, `skip_status:true`, `include_user_entities:false`, `cursor?`
  - Map REST fields: `friends_count` → followingCount; `verified` → isBlueVerified fallback (correction #14)

#### ListOwnerships / ListMemberships:
- Variables (correction #30): `userId`, `count:100`, `isListMembershipShown:true`, `isListMemberTargetUserId:userId`
- No cursor even when paginating
- Response path: `data.user.result.timeline.timeline.instructions` (corrections #1, #17)
- Parse via `parseListsFromInstructions`: `entry.content.itemContent.list` (NOT module items)
- List owner fields from `legacy` only (no `.core` fallback for list owner; response-parsing.md §List Response Parsing)

#### ListLatestTweetsTimeline:
- Variables (correction #84): `listId`, `count:20`, `cursor?` — no extra fields
- Response path: `data.list.tweets_timeline.timeline.instructions` (correction #45)

#### News (GetNews):
- Uses `GenericTimelineById` for each tab
- Default tabs: `forYou`, `news`, `sports`, `entertainment` (NOT `trending`; correction #46)
- `count = maxCount * 2` (correction #67); `includePromotedContent:false` (correction #34)
- Features: `buildExploreFeatures()`
- Response path: `data.timeline.timeline.instructions` (correction #1 for GenericTimelineById)
- Parse with `parseTimelineTabItems`: handles both `.entries` (array) and `.entry` (single)
- Headline: `itemContent.name || itemContent.title` (correction #27)
- URL: `itemContent.trend_url?.url || trend_metadata?.url?.url` (nested; correction #27)
- ID: `trendUrl ?? (entryId ? "${entryId}-${headline}" : "${tabName}-${headline}")` (correction #48)
- Deduplication by `headline` string across and within tabs (correction #47)
- AI detection: `is_ai_trend == true || (words>=5 && (includes("News") || includes("hours ago")))`
- Category parsing: split `social_context.text` on `"·"`; override with `domain_context` when category is `"Trending"` or `"News"`

Tests first: bookmarks retry, bookmark folder count-retry, UserTweets page cap, follower REST fallback gating, list path corrections, news parsing and deduplication.

---

### Phase 10 — Mutations and Media Upload

Deliverables: `internal/client/{post,bookmarks,engagement,follow,media}.go`

#### Tweet / Reply:
- `createTweet` POST body always includes `media` field (even empty `media_entities:[]`; graphql-operations.md §5)
- Referer: `https://x.com/compose/post`
- Features: `buildTweetCreateFeatures()`
- Response path: `data.create_tweet.tweet_results.result.rest_id` (correction #38)
- 404 fallback → refresh → retry same URL → double-404 → POST bare graphql URL (correction #72)
- Error code 226 → `tryStatusUpdateFallback()` (correction #39)

#### statuses/update.json fallback (correction #39, #40):
- Headers: `getBaseHeaders()` + explicit `content-type: application/x-www-form-urlencoded`
- Body: `status=<text>` + optional `in_reply_to_status_id` + `auto_populate_reply_metadata=true` + optional `media_ids`
- Parse: `id_str` first, `String(id)` fallback

#### Engagement mutations (Like, Unlike, Retweet, Unretweet, Bookmark):
- Body: `{ variables: { tweet_id }, queryId }` — NO `features` (correction #3)
- Referer: `https://x.com/i/status/<tweetId>`
- 404 fallback: refresh + retry; double-404 → POST bare graphql URL (correction #72)

#### DeleteRetweet (correction #3):
- Body: `{ variables: { tweet_id: id, source_tweet_id: id }, queryId }` — both set to same ID
- No `dark_request`. No `features`.

#### Unbookmark (DeleteBookmark):
- Implemented in bookmarks.go (not engagement.go; graphql-operations.md §27)
- Same fallback pattern as engagement mutations

#### Follow / Unfollow (correction #4):
1. POST `https://x.com/i/api/1.1/friendships/create.json` (form-encoded: `user_id=<id>&skip_status=true`)
2. POST `https://api.twitter.com/1.1/friendships/create.json`
3. Only then: GraphQL CreateFriendship/DestroyFriendship
- REST error 160 = success (already following)
- REST errors 162 (blocked), 108 (not found) = error
- GraphQL body: `{ variables: { user_id: "<id>" }, queryId }` — no `features` (correction #68)
- GraphQL fallback IDs: CreateFriendship `[primary, "8h9JVdV8dlSyqyRDJEPCsA", "OPwKc1HXnBT_bWXfAlo-9g"]`; DestroyFriendship `[primary, "ppXWuagMNXgvzx6WoXBW0Q", "8h9JVdV8dlSyqyRDJEPCsA"]`

#### Media Upload:
1. **INIT** (form-encoded): `command=INIT&total_bytes=<n>&media_type=<mime>&media_category=<cat>`
   - Category: `image/gif`→`tweet_gif`, `video/*`→`tweet_video`, `image/*`→`tweet_image`
   - Response: `media_id_string` or `String(media_id)`
2. **APPEND** (multipart form-data): chunks of exactly `5*1024*1024` bytes (`5<<20`); `segment_index` increments
3. **FINALIZE** (form-encoded): `command=FINALIZE&media_id=<id>`
   - If `processing_info.state` present and not `"succeeded"`: poll STATUS
4. **STATUS polling** (GET with `command=STATUS&media_id=<id>`):
   - Max 20 polls (correction #10)
   - Delay: `check_after_secs` from response, default 2 when not finite, clamp to ≥1 (correction #21)
   - Stop on `"succeeded"` or `"failed"`; also stop when `processing_info` absent
5. **Alt text** (images only; correction #10):
   `POST https://x.com/i/api/1.1/media/metadata/create.json` with JSON headers
   `{"media_id":"<id>","alt_text":{"text":"<alt>"}}`
   Only when `alt != ""` AND `mimeType.hasPrefix("image/")`
6. All INIT/APPEND/FINALIZE use `uploadHeaders()` (no content-type; correction #70)
7. CLI `detectMime` returns nil for unsupported types; caller throws (correction #81)
   Supported extensions: `.jpg/.jpeg`→`image/jpeg`, `.png`→`image/png`, `.webp`→`image/webp`, `.gif`→`image/gif`, `.mp4/.m4v`→`video/mp4`, `.mov`→`video/quicktime`
   Error message lists `mov` but not `m4v` (correction #81 nuance)

Tests first: CreateTweet fallback chain, DeleteRetweet payload, engagement mutation no-features, follow REST-first, media chunk boundaries, media poll max attempts, alt-text image-only.

---

### Phase 11 — CLI Wiring and Output Parity

Deliverables: `internal/cli/*.go`, `internal/output/*.go`, `cmd/bird/main.go`

#### Global Flags (all documented, no extras):
```
--auth-token <token>
--ct0 <token>
--cookie-source <src>    repeatable; values: safari, chrome, firefox
--chrome-profile <name>
--chrome-profile-dir <path>
--firefox-profile <name>
--cookie-timeout <ms>
--timeout <ms>
--quote-depth <n>
--plain
--no-emoji
--no-color
--media <path>           repeatable
--alt <text>             repeatable
--version
--help
```

#### CLI Commands:
```
bird tweet "<text>"
bird reply <tweet-id-or-url> "<text>"
bird read <tweet-id-or-url>                  --json, --json-full
bird <tweet-id-or-url>                       shorthand for read
bird replies <tweet-id-or-url>               --all, --max-pages, --cursor, --delay, --json
bird thread <tweet-id-or-url>                --all, --max-pages, --cursor, --delay, --json
bird search "<query>"                        -n/--count, --all, --max-pages, --cursor, --json
bird mentions                                -n, --all, --max-pages, --cursor, --json, --user <@handle>
bird home                                    -n, --following, --json, --json-full
bird bookmarks                               -n, --all, --max-pages, --cursor, --json,
                                             --folder-id, --expand-root-only, --author-chain,
                                             --author-only, --full-chain-only, --include-ancestor-branches,
                                             --include-parent, --thread-meta, --sort-chronological
bird unbookmark <tweet-id-or-url...>
bird likes                                   -n, --all, --max-pages, --cursor, --json, --json-full
bird news                                    -n, --json, --ai-only, --with-tweets, --tweets-per-item,
                                             --for-you, --news-only, --sports, --entertainment, --trending-only
bird trending                                alias for news
bird user-tweets <@handle>                   -n, --all, --max-pages, --cursor, --delay, --json
bird lists                                   -n, --json, --member-of
bird list-timeline <list-id-or-url>          -n, --all, --max-pages, --cursor, --json
bird following                               -n, --all, --max-pages, --cursor, --json, --user <userId>
bird followers                               -n, --all, --max-pages, --cursor, --json, --user <userId>
bird follow <@handle-or-userId>
bird unfollow <@handle-or-userId>
bird about <@handle>                         --json
bird whoami                                  --json
bird check
bird query-ids                               --fresh
bird help [command]
```

Note: `--user` on `mentions` takes `@handle`; `--user` on `following`/`followers` takes numeric userId.

#### Exit Codes:
- `0`: success
- `1`: runtime/auth/API/network failure; also: `check` with missing credentials; partial pagination after printing items
- `2`: invalid flag values, missing required args, malformed handles (pre-network), unknown command in `help`

Partial pagination: print accumulated items to stdout FIRST, then print error to stderr, then exit 1.

#### Output Modes:
- Rich text (TTY default): color, emoji, OSC 8 hyperlinks
- `--plain`: strip all formatting, OSC 8, emoji
- `--no-emoji`: strip emoji only
- `--no-color` / `NO_COLOR=1`: strip ANSI colors
- `TERM=dumb`: same as `--no-color`
- `--json`: compact JSON per line
- `--json-full`: includes `_raw` field (on `home`, `likes`, `news`, `read`)

#### Version output:
`bird --version` → `<version> (<gitSHA>)` e.g. `0.1.0 (3df7969b)`

Tests first: root help/version, all command argument validation, golden output (text, plain, JSON, JSON-full), partial pagination exit-code tests, shorthand dispatch.

---

### Phase 12 — Public Package Surface

Deliverables: `pkg/bird/{client,auth,types,doc}.go`

Export:
- `TwitterClient` with all documented methods
- `ResolveCredentials()`, `ExtractCookiesFromSafari()`, `ExtractCookiesFromChrome()`, `ExtractCookiesFromFirefox()`
- `RuntimeQueryIDs` singleton store access
- Re-export all public types from `internal/types`

Every exported identifier must have a doc comment. Add usage examples where practical.

Tests first: public package compile tests, example tests.

---

### Phase 13 — Hardening, Fixtures, and Release Readiness

Deliverables: expanded fixture corpus, full acceptance suite, CI cleanup.

#### Fixture corpus organization:
```
tests/fixtures/
├── auth/
├── query_ids/
├── tweet_detail/        # happy path, article, note tweet, quoted, partial-error, 404-retry
├── search/              # happy path, GRAPHQL_VALIDATION_FAILED, rawQuery-must-be-defined
├── home/                # query-unspecified refresh
├── bookmarks/           # retry-429, retry-500, bookmark-folder-count-error, cursor-error
├── likes/               # query-unspecified
├── user_tweets/         # page-cap, user-suspended, user-not-found
├── following/           # REST-fallback after refresh
├── followers/
├── lists/               # ownerships, memberships, timeline
├── news/                # per tab, deduplication, AI detection
├── about/
├── media/               # init, append chunks, finalize, status-polling, alt-text
├── mutations/           # CreateTweet, fallback-226, DeleteRetweet, engagements
└── user_lookup/         # UserByScreenName rotation, UserUnavailable, REST fallback
```

Each fixture family should include:
- Raw upstream response JSON
- Expected normalized result snapshot
- At least one correction/fallback regression case for high-risk operations
- Empty/edge-case variants (empty results, repeated cursor, no cursor, UserUnavailable)

#### Race tests:
- Runtime query ID store concurrent reads/writes
- Feature override cache concurrent access
- Any shared memoized state

#### Golden files:
- Tweet text output (rich, plain, no-color)
- JSON output (--json, --json-full)
- `query-ids` reporting

Tests to add:
- Acceptance tests for every command
- Cross-platform config and output sanity
- All 86 corrections have explicit regression test coverage

---

## 14. Command Implementation Matrix

| CLI command | Primary client methods | Key requirements |
|---|---|---|
| `bird tweet` | `Tweet`, `UploadMedia` | media validation, alt text, CreateTweet fallback chain |
| `bird reply` | `Reply`, `UploadMedia` | tweet ID extraction, CreateTweet fallback |
| `bird read` | `GetTweet` | `--json`, `--json-full`; UserArticlesTweets fallback |
| `bird <url/id>` | `GetTweet` | shorthand insertion before Cobra parse |
| `bird replies` | `GetRepliesPaged` | generic paginateCursor, delay before fetch |
| `bird thread` | `GetThreadPaged` | generic paginateCursor, deduplicate |
| `bird search` | `Search` | POST search, 4-ID rotation, validation-failed refresh |
| `bird mentions` | `Search` | mentionsQueryFromUserOption, handle normalization |
| `bird home` | `GetHomeTimeline/GetHomeLatestTimeline` | `--following`, `--json-full`, no nextCursor |
| `bird bookmarks` | `GetBookmarks/GetAllBookmarks/GetBookmarkFolderTimeline` | retry, thread filters, folder mode |
| `bird unbookmark` | `Unbookmark` | multi-arg loop, partial failure |
| `bird likes` | `GetLikes/GetAllLikes` | no retry, `--json-full`, Query: Unspecified refresh |
| `bird news` | `GetNews` | tab IDs, GenericTimelineById, dedup by headline |
| `bird trending` | `GetNews` | alias of news |
| `bird user-tweets` | `GetUserTweetsPaged` | page cap 10, fatal error strings |
| `bird lists` | `GetOwnedLists/GetListMemberships` | corrected variable set, corrected response path |
| `bird list-timeline` | `GetListTimeline/GetAllListTimeline` | list ID extraction, simple variables |
| `bird following` | `GetFollowing` | userId input, REST fallback after refresh only |
| `bird followers` | `GetFollowers` | userId input, REST fallback after refresh only |
| `bird follow` | `Follow`, `GetUserIDByUsername` | REST-first, handle or userId input |
| `bird unfollow` | `Unfollow`, `GetUserIDByUsername` | REST-first |
| `bird whoami` | `GetCurrentUser` | API chain then HTML fallback |
| `bird about` | `GetUserAboutAccount` | corrected variable and response path |
| `bird check` | credential resolution, `GetCurrentUser` | exit 1 on missing credentials |
| `bird query-ids` | RuntimeQueryIDs store | refresh 10 ops (not 29), cache reporting |
| `bird help` | CLI only | command-aware help, unknown command → exit 2 |

---

## 15. Testing Strategy

#### Unit tests (alongside source files as `_test.go`):
- Header construction and secret redaction
- Config/env precedence
- Input extraction and normalization
- Feature map serialization (snapshot all 12 maps)
- Query ID lookup, cache freshness, refresh decisions
- Parser behavior per fixture family
- Thread filter algorithms
- Output formatting helpers
- Correction regression tests (one test per correction section)

#### Service tests with mocked transport:
- Request URL generation per operation
- Request body serialization per operation
- Fallback sequencing (TweetDetail GET→POST, CreateTweet chain, follow REST-first)
- Retry behavior (bookmarks 429/500/502/503/504)
- Partial-success handling
- Error classification (exit 1 vs exit 2)

#### Acceptance tests (`tests/acceptance/`):
- Every CLI command with representative inputs
- Shorthand dispatch
- Help and version output
- JSON vs text output modes
- Exit code mapping
- Partial pagination semantics

#### Golden tests (`tests/golden/`):
- Rich tweet rendering
- Plain mode
- No-color mode
- JSON output
- Query-ID reporting

#### Race tests:
- Runtime query ID store
- Feature override cache
- Shared memoized state

---

## 16. Makefile

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -ldflags "-X main.version=$(VERSION) -X main.gitSHA=$(GIT_SHA)"

.PHONY: build test test-race lint vet fmt clean coverage ci

build:
	go build $(LDFLAGS) -o bin/bird ./cmd/bird

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean:
	rm -rf bin/ coverage.out

ci: vet lint test test-race build
```

---

## 17. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Upstream query IDs change, refresh scraping breaks | Isolate refresh logic; test against captured bundle assets; fallback ID lists |
| Feature maps silently drift → empty responses | Snapshot all 12 feature maps in tests; compare per operation family |
| Parser overfits one payload family | Maintain fixtures for item, module, replace-entry, and multi-timeline variants |
| Output regressions when JSON tests pass | Require golden files for text output and partial pagination messages |
| Follow/followers behavior implemented too generically | Dedicated tests for refresh-only REST fallback and REST error codes |
| Media upload accumulates off-by-one bugs | Isolated tests with fake server asserting chunk boundaries and poll counts |
| Chrome decryption fails on newer macOS | Wrap in timeout; fall through to next browser source |
| Safari binary format changes | Treat as non-fatal extraction failure; log warning |
| Pagination delay placed after fetch (common mistake) | Assert delay position in paginator unit tests |
| Engagement mutations accidentally include `features` | Test request body serialization for each mutation type |
| TweetDetail fieldToggles included in POST body | Test that POST body excludes fieldToggles (correction #36) |

---

## 18. Immediate Execution Order

1. Bootstrap module, Makefile, lint, vet, CI (Phase 0)
2. Define types and error taxonomy (Phase 1)
3. Config loading, env precedence, input extraction (Phase 2)
4. Credential resolution and browser cookie extraction (Phase 3)
5. Client core, headers, transport, request builders (Phase 4)
6. Runtime query IDs store and feature override system (Phase 5)
7. Parsing engine with fixture-backed tests (Phase 6)
8. Current-user and user-lookup flows (Phase 7)
9. TweetDetail, SearchTimeline, HomeTimeline (Phase 8)
10. Bookmarks, Likes, UserTweets, Following/Followers, Lists, News (Phase 9)
11. Mutations, Follow/Unfollow, Media Upload (Phase 10)
12. CLI commands, global flags, output formatting (Phase 11)
13. Public package surface (Phase 12)
14. Acceptance tests, goldens, race tests, CI cleanup (Phase 13)

---

## 19. Final Go/No-Go Checklist

Do not mark `gobird` done until:

- [ ] All 26 CLI commands implemented and tested
- [ ] All global flags implemented and tested
- [ ] All per-command flags implemented and tested
- [ ] All 86 correction sections have explicit regression test coverage
- [ ] All 29 FALLBACK_QUERY_IDS embedded as constants
- [ ] Query ID refresh scrapes all 29 operations; CLI command refreshes 10
- [ ] Per-operation fallback ID lists all correct and tested
- [ ] All 12 feature flag maps are snapshot-tested
- [ ] Feature override loading (bundled → file → env) is tested
- [ ] Bookmark retry (maxRetries=2, baseDelayMs=500) is tested
- [ ] BookmarkFolderTimeline count-retry and cursor-error behavior is tested
- [ ] UserTweets hard 10-page cap is tested
- [ ] UserByScreenName hardcoded ID rotation + UserUnavailable short-circuit is tested
- [ ] getCurrentUser 4-API + 2-HTML-fallback chain is tested
- [ ] Following/Followers REST fallback gated on `refreshed==true` is tested
- [ ] TweetDetail GET→POST per-queryId fallback is tested
- [ ] CreateTweet error-226 → statuses/update.json fallback is tested
- [ ] DeleteRetweet both-IDs payload (no dark_request, no features) is tested
- [ ] Engagement mutations no-features body is tested
- [ ] Follow REST-first (error 160 = success) is tested
- [ ] Media upload: chunk size 5 MiB, max 20 polls, alt-text image-only is tested
- [ ] paginateCursor stops only on cursor change; inline loops stop on zero-items too
- [ ] HomeTimeline returns no nextCursor to caller
- [ ] News deduplicates by headline; default 4 tabs (not trending)
- [ ] Text, plain, no-color, JSON, JSON-full outputs have golden files
- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...`, `golangci-lint run` all pass
- [ ] `go build ./cmd/bird` produces binary with correct `--version` output
- [ ] No unresolved high-severity parity gaps remain

---

This plan is the working execution contract. Update it as the repo acquires code and tests.
