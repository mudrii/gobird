# Twitter/X GraphQL Wire Protocol

This document describes the Twitter/X GraphQL wire protocol as implemented in gobird. It covers authentication, request encoding, response shapes, and all 29 supported operations.

---

## Authentication Scheme

Every request carries three authentication credentials simultaneously.

### Bearer Token

All requests include a static public bearer token in the `Authorization` header:

```
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA
```

This token is public and identical for all Twitter/X web clients. It identifies the application (the X.com web client) rather than the user.

### Cookie Authentication

User identity is established via two cookies:

- `auth_token` — the session token obtained when logging in
- `ct0` — the CSRF token, a random hex value bound to the session

These are sent as a single `Cookie` header:

```
Cookie: auth_token=<auth_token>; ct0=<ct0>
```

### CSRF Token

The `ct0` value is also sent as a separate header to prevent cross-site request forgery:

```
x-csrf-token: <ct0>
```

The server validates that the `Cookie: ct0` value matches `x-csrf-token`. Requests without this match are rejected.

### Standard Headers

Every request includes the following headers, regardless of operation:

```http
accept: */*
accept-language: en-US,en;q=0.9
authorization: Bearer <BearerToken>
x-twitter-auth-type: OAuth2Session
x-twitter-active-user: yes
x-twitter-client-language: en
x-csrf-token: <ct0>
cookie: auth_token=<auth_token>; ct0=<ct0>
user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36
origin: https://x.com
referer: https://x.com/
x-client-transaction-id: <random-16-byte-hex>
```

POST requests for GraphQL operations also include:

```http
content-type: application/json
```

Media upload APPEND requests use `multipart/form-data` with the boundary set by the multipart writer.

---

## GraphQL Operation URL Format

GraphQL operations use the following URL pattern:

```
https://x.com/i/api/graphql/{queryID}/{operationName}
```

Where:
- `{queryID}` is a 22-character base64url-encoded identifier (e.g. `TAJw1rBsjAtdNgTdlo2oeg`)
- `{operationName}` is the operation name as it appears in the X.com client bundle (e.g. `CreateTweet`, `SearchTimeline`)

The base URL is `https://x.com/i/api/graphql`.

### Example

```
GET https://x.com/i/api/graphql/Wms1GvIiHXAPBaCr9KblaA/UserTweets?variables=...&features=...&fieldToggles=...
```

---

## Variables Encoding

Variables are encoded differently depending on the operation type.

### GET Operations (URL Query Parameters)

For read operations, variables are JSON-encoded and placed in the URL query string:

```
GET /graphql/{queryID}/{operation}?variables={json-encoded-vars}&features={json-encoded-features}
```

Some operations also include `fieldToggles`:

```
GET /graphql/{queryID}/{operation}?variables=...&features=...&fieldToggles=...
```

The values are URL-encoded (`url.QueryEscape` in Go, equivalent to `encodeURIComponent` in JavaScript).

**Operations using this pattern**: UserTweets, UserByScreenName, TweetDetail (primary), HomeTimeline, HomeLatestTimeline, Following, Followers, Bookmarks, BookmarkFolderTimeline, Likes, ListOwnerships, ListMemberships, ListLatestTweetsTimeline, SearchTimeline (variables only — see below), GenericTimelineById, AboutAccountQuery

### POST Operations (JSON Body)

For write operations, all data is sent in the JSON body:

```
POST /graphql/{queryID}/{operation}
Content-Type: application/json

{
  "variables": { ... },
  "features": { ... },
  "queryId": "{queryID}"
}
```

**Operations using this pattern**: CreateTweet, CreateRetweet, DeleteRetweet, FavoriteTweet, UnfavoriteTweet, CreateFriendship, DestroyFriendship, CreateBookmark, DeleteBookmark

### Mixed POST (SearchTimeline Special Case)

SearchTimeline uses POST but puts `variables` in the URL, not the body (correction #35):

```
POST /graphql/{queryID}/SearchTimeline?variables={json-encoded-vars}
Content-Type: application/json

{
  "features": { ... },
  "queryId": "{queryID}"
}
```

### TweetDetail GET → POST Fallback

TweetDetail first attempts GET with all parameters in the URL. If the GET returns 404, it immediately retries with POST, moving the variables into the body (correction #36):

```json
{
  "variables": { ... },
  "features": { ... },
  "queryId": "{queryID}"
}
```

---

## Features Object

The `features` object is a flat map of boolean flags sent with every GraphQL request. It tells the server which response fields to include and which feature flags are active for the client.

Features are operation-specific. Sending the wrong feature set can cause validation errors or unexpected response shapes.

### Feature Set Hierarchy

Feature sets are built by composition:

```
buildArticleFeatures()          — base set (~37 flags)
  └── buildSearchFeatures()     — article + rweb_video_timestamps_enabled
        └── buildTimelineFeatures()   — search + 8 more timeline flags
              ├── buildBookmarksFeatures()     — timeline + graphql_timeline_v2_bookmark_timeline
              ├── buildLikesFeatures()         — timeline (no additions)
              └── buildHomeTimelineFeatures()  — timeline (no additions)
  └── buildExploreFeatures()    — search + 5 Grok/trending flags
buildTweetDetailFeatures()      — article + 3 article-specific flags
buildTweetCreateFeatures()      — article + responsive_web_profile_redirect_enabled=false
buildUserTweetsFeatures()       — standalone hardcoded map
buildListsFeatures()            — standalone hardcoded map
buildFollowingFeatures()        — standalone hardcoded map
buildUserByScreenNameFeatures() — delegates to buildArticleFeatures()
```

### Key Feature Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `rweb_video_screen_enabled` | true | Video player in feed |
| `rweb_video_timestamps_enabled` | true (search/explore) | Video timestamp markers |
| `articles_preview_enabled` | true | Article card previews |
| `longform_notetweets_consumption_enabled` | true | Long-form tweet reading |
| `responsive_web_graphql_timeline_navigation_enabled` | true | Timeline navigation |
| `tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled` | true | Visibility wrappers |
| `graphql_timeline_v2_bookmark_timeline` | true (bookmarks only) | Bookmark timeline v2 path |
| `responsive_web_profile_redirect_enabled` | false (CreateTweet) | Profile redirect on post |
| `verified_phone_label_enabled` | false | Phone verification badge |

### Runtime Override

Features can be overridden at runtime via two environment variables:

- `BIRD_FEATURES_JSON` — inline JSON string of overrides
- `BIRD_FEATURES_PATH` — path to a JSON file of overrides

The JSON schema is:

```json
{
  "global": { "flag_name": true },
  "sets": {
    "article": { "flag_name": false },
    "search": { "flag_name": true }
  }
}
```

Override order: base → global overrides → set-specific overrides.

---

## FieldToggles

`fieldToggles` is a small flat map sent as a URL parameter on GET requests. It enables or disables specific response fields at the per-operation level, as opposed to the broader feature flags.

| Operation | fieldToggles |
|-----------|-------------|
| TweetDetail | `{"withPayments":false,"withAuxiliaryUserLabels":false,"withArticleRichContentState":true,"withArticlePlainText":true,"withGrokAnalyze":false,"withDisallowedReplyControls":false}` |
| UserTweets | `{"withArticlePlainText":false}` |
| UserByScreenName | `{"withAuxiliaryUserLabels":false}` |

Operations not listed above do not send `fieldToggles`.

---

## Timeline Instruction Types

The `instructions` array in timeline responses contains objects of the following types:

### TimelineAddEntries

The most common instruction type. Contains an `entries` array of timeline entries.

```json
{
  "type": "TimelineAddEntries",
  "entries": [ ... ]
}
```

### TimelinePinEntry

Contains a single `entry` (not `entries`) for a pinned tweet. The entry is at the top-level `entry` key on the instruction.

```json
{
  "type": "TimelinePinEntry",
  "entry": { ... }
}
```

Gobird's cursor extractor checks both `inst.Entries` and `inst.Entry` to handle pinned entries.

### TimelineTerminateTimeline

Signals the end of a timeline direction (top or bottom). Contains a `direction` field. When seen with `"direction": "Bottom"`, no further pages exist.

```json
{
  "type": "TimelineTerminateTimeline",
  "direction": "Bottom"
}
```

Gobird does not explicitly parse this type — it relies on the absence of a Bottom cursor instead.

### TimelineReplaceEntry

Replaces an existing entry by ID. Used for live updates. Gobird does not process this type.

---

## Entry Types and Content Paths

Each entry in an instruction has an `entryId`, `sortIndex`, and `content` object.

### Tweet Entry

```json
{
  "entryId": "tweet-1234567890",
  "sortIndex": "1234567890",
  "content": {
    "entryType": "TimelineTimelineItem",
    "__typename": "TimelineTimelineItem",
    "itemContent": {
      "__typename": "TimelineTweet",
      "tweet_results": {
        "result": { ... }
      }
    }
  }
}
```

Path: `entry.content.itemContent.tweet_results.result`

### Module Timeline Entry

A module entry contains multiple items, each with their own tweet result:

```json
{
  "entryId": "conversationthread-1234",
  "content": {
    "entryType": "TimelineTimelineModule",
    "items": [
      {
        "entryId": "conversationthread-1234-tweet-5678",
        "item": {
          "itemContent": {
            "tweet_results": {
              "result": { ... }
            }
          }
        }
      }
    ]
  }
}
```

Path: `entry.content.items[].item.itemContent.tweet_results.result`

### TweetWithVisibilityResults Wrapper

Some tweets are wrapped in a visibility results object. The actual tweet is at `result.tweet`:

```json
{
  "__typename": "TweetWithVisibilityResults",
  "tweet": {
    "__typename": "Tweet",
    "rest_id": "...",
    ...
  }
}
```

Gobird's `UnwrapTweetResult` function traverses this wrapper transparently.

### User Entry

```json
{
  "content": {
    "itemContent": {
      "__typename": "TimelineUser",
      "user_results": {
        "result": { ... }
      }
    }
  }
}
```

Path: `entry.content.itemContent.user_results.result`

### List Entry

```json
{
  "content": {
    "itemContent": {
      "list_results": {
        "result": { ... }
      }
    }
  }
}
```

Path: `entry.content.itemContent.list_results.result`

---

## Cursor Entry Format and Extraction

Pagination cursors appear as special entries within the `instructions` array.

### Cursor Entry Shape

```json
{
  "entryId": "cursor-bottom-1234567890",
  "content": {
    "entryType": "TimelineTimelineCursor",
    "__typename": "TimelineTimelineCursor",
    "cursorType": "Bottom",
    "value": "LBn_____..."
  }
}
```

Key fields:
- `content.cursorType` — either `"Bottom"` (next page) or `"Top"` (previous page)
- `content.value` — opaque cursor string to pass as the `cursor` variable

### Extraction Algorithm

`ExtractCursorFromInstructions` (`parsing/cursors.go`) scans all instructions:

1. For each instruction, scan `inst.Entries` — if any entry has `content.CursorType == "Bottom"`, return `content.Value`.
2. If the instruction has a non-nil `inst.Entry` (TimelinePinEntry), check it for `CursorType == "Bottom"`.
3. Return empty string if no Bottom cursor found.

Only `cursorType` is checked — not `entryType`, not `__typename`, not the `entryId` string (correction #7).

---

## Error Response Format

### GraphQL Errors

GraphQL errors are returned in an `errors` array at the top level of the response. A 200 HTTP status may accompany errors (partial data or query-ID mismatches):

```json
{
  "data": { ... },
  "errors": [
    {
      "message": "The query was not found.",
      "locations": [{ "line": 1, "column": 0 }],
      "path": ["rawQuery"],
      "extensions": {
        "code": "GRAPHQL_VALIDATION_FAILED",
        "name": "GraphqlValidationFailed",
        "source": "CLIENT"
      }
    }
  ]
}
```

### Extensions Codes

| Code | Meaning | Gobird Response |
|------|---------|----------------|
| `GRAPHQL_VALIDATION_FAILED` | Stale or invalid query ID | Refresh query IDs, retry |
| `226` | Automated behaviour detected | Fall back to REST `statuses/update.json` |
| `query: unspecified` (in message) | Stale query ID for home timeline | Refresh query IDs, retry |
| `Query: Unspecified` (in message) | Stale query ID for likes | Try next ID, then refresh |
| `UserUnavailable` (in `__typename`) | Account suspended/deactivated | Return error immediately |

### HTTP Error Responses

Non-2xx responses are wrapped in an `httpError` struct with `StatusCode` and `Body`. Only 404 is special-cased (`is404`); other non-retryable errors are returned directly. For `fetchWithRetry`, status codes 429, 500, 502, 503, 504 trigger exponential-backoff retry.

---

## Media Upload API

Media upload uses the Twitter REST media upload endpoint, not GraphQL:

```
https://upload.twitter.com/i/media/upload.json
```

The upload protocol has four sequential phases.

### Phase 1: INIT

Initialises the upload and obtains a `media_id_string`.

```
POST https://upload.twitter.com/i/media/upload.json
Content-Type: application/x-www-form-urlencoded

command=INIT&total_bytes=<size>&media_type=<mime-type>
```

Response:
```json
{ "media_id_string": "123456789012345678" }
```

### Phase 2: APPEND

Uploads binary data in chunks of exactly 5 MiB (5 × 1024 × 1024 bytes).

```
POST https://upload.twitter.com/i/media/upload.json
Content-Type: multipart/form-data; boundary=...

--boundary
Content-Disposition: form-data; name="command"
APPEND
--boundary
Content-Disposition: form-data; name="media_id"
123456789012345678
--boundary
Content-Disposition: form-data; name="segment_index"
0
--boundary
Content-Disposition: form-data; name="media"
<binary chunk data>
--boundary--
```

Success: HTTP 204 No Content or 200 OK.

Chunks are indexed from 0. The final chunk may be smaller than 5 MiB.

### Phase 3: FINALIZE

Signals that all chunks have been uploaded.

```
POST https://upload.twitter.com/i/media/upload.json
Content-Type: application/x-www-form-urlencoded

command=FINALIZE&media_id=123456789012345678
```

### Phase 4: STATUS Polling

For video and GIF, the server processes the media asynchronously. The client polls until processing completes.

```
GET https://upload.twitter.com/i/media/upload.json?command=STATUS&media_id=123456789012345678
```

Response:
```json
{
  "processing_info": {
    "state": "in_progress",
    "check_after_secs": 3
  }
}
```

Gobird polls up to 20 times (`mediaMaxPolls = 20`). Terminal states are `"succeeded"` and `"failed"`. If `processing_info` is absent, processing is already complete. Delay between polls uses `check_after_secs` from the response, defaulting to 2 seconds.

### Alt Text

After a successful upload, alt text can be attached to images only (not video or GIF):

```
POST https://x.com/i/api/1.1/media/metadata/create.json
Content-Type: application/json

{
  "media_id": "123456789012345678",
  "alt_text": { "text": "Description of the image" }
}
```

---

## REST vs GraphQL Operations

| Operation | Protocol | Endpoint |
|-----------|----------|----------|
| HomeTimeline | GraphQL GET | `/graphql/{id}/HomeTimeline` |
| HomeLatestTimeline | GraphQL GET | `/graphql/{id}/HomeLatestTimeline` |
| SearchTimeline | GraphQL POST (vars in URL) | `/graphql/{id}/SearchTimeline` |
| TweetDetail | GraphQL GET → POST fallback | `/graphql/{id}/TweetDetail` |
| UserTweets | GraphQL GET | `/graphql/{id}/UserTweets` |
| UserArticlesTweets | GraphQL GET | `/graphql/{id}/UserArticlesTweets` |
| UserByScreenName | GraphQL GET | `/graphql/{id}/UserByScreenName` |
| AboutAccountQuery | GraphQL GET | `/graphql/{id}/AboutAccountQuery` |
| Bookmarks | GraphQL GET (fetchWithRetry) | `/graphql/{id}/Bookmarks` |
| BookmarkFolderTimeline | GraphQL GET (fetchWithRetry) | `/graphql/{id}/BookmarkFolderTimeline` |
| CreateBookmark | GraphQL POST | `/graphql/{id}/CreateBookmark` |
| DeleteBookmark | GraphQL POST | `/graphql/{id}/DeleteBookmark` |
| Likes | GraphQL GET | `/graphql/{id}/Likes` |
| FavoriteTweet | GraphQL POST | `/graphql/{id}/FavoriteTweet` |
| UnfavoriteTweet | GraphQL POST | `/graphql/{id}/UnfavoriteTweet` |
| Following | GraphQL GET | `/graphql/{id}/Following` |
| Followers | GraphQL GET | `/graphql/{id}/Followers` |
| CreateFriendship | GraphQL POST | `/graphql/{id}/CreateFriendship` |
| DestroyFriendship | GraphQL POST | `/graphql/{id}/DestroyFriendship` |
| CreateTweet | GraphQL POST | `/graphql/{id}/CreateTweet` |
| CreateRetweet | GraphQL POST | `/graphql/{id}/CreateRetweet` |
| DeleteRetweet | GraphQL POST | `/graphql/{id}/DeleteRetweet` |
| ListOwnerships | GraphQL GET | `/graphql/{id}/ListOwnerships` |
| ListMemberships | GraphQL GET | `/graphql/{id}/ListMemberships` |
| ListLatestTweetsTimeline | GraphQL GET | `/graphql/{id}/ListLatestTweetsTimeline` |
| ListByRestId | GraphQL GET | `/graphql/{id}/ListByRestId` |
| GenericTimelineById | GraphQL GET | `/graphql/{id}/GenericTimelineById` |
| ExploreSidebar | GraphQL GET | `/graphql/{id}/ExploreSidebar` |
| ExplorePage | GraphQL GET | `/graphql/{id}/ExplorePage` |
| TrendHistory | GraphQL GET | `/graphql/{id}/TrendHistory` |
| getCurrentUser | REST GET (multiple) | `account/settings.json`, `verify_credentials.json` |
| Follow/Unfollow | REST POST | `/i/api/1.1/friendships/create.json`, `friendships/destroy.json` |
| Following (fallback) | REST GET | `/i/api/1.1/friends/list.json` |
| Followers (fallback) | REST GET | `/i/api/1.1/followers/list.json` |
| UserByScreenName (fallback) | REST GET | `/i/api/1.1/users/show.json` |
| CreateTweet (fallback) | REST POST | `/i/api/1.1/statuses/update.json` |
| UploadMedia | REST POST/GET | `upload.twitter.com/i/media/upload.json` |
| MediaMetadata | REST POST | `/i/api/1.1/media/metadata/create.json` |

---

## All 29 Operations Table

| # | Operation | Method | Variables | Response Path |
|---|-----------|--------|-----------|---------------|
| 1 | CreateTweet | POST | `tweet_text`, `dark_request`, `media`, `semantic_annotation_ids`, optional `reply` | `data.create_tweet.tweet_results.result.rest_id` |
| 2 | CreateRetweet | POST | `tweet_id`, `dark_request` | `data.create_retweet.retweet_results.result.rest_id` |
| 3 | DeleteRetweet | POST | `source_tweet_id`, `dark_request` | `data.unretweet.source_tweet_results.result.rest_id` |
| 4 | CreateFriendship | POST | `userId` | `data.create_friendship.user.result` |
| 5 | DestroyFriendship | POST | `userId` | `data.destroy_friendship.user.result` |
| 6 | FavoriteTweet | POST | `tweetId` | `data.favorite_tweet` |
| 7 | UnfavoriteTweet | POST | `tweetId` | `data.unfavorite_tweet` |
| 8 | CreateBookmark | POST | `tweet_id` | `data.bookmark_tweet_result.result` |
| 9 | DeleteBookmark | POST | `tweet_id` | `data.tweet_bookmark_delete.tweet_bookmark_delete_result` |
| 10 | TweetDetail | GET→POST | `focalTweetId`, `with_rux_injections`, `rankingMode`, `includePromotedContent`, `withCommunity`, `withQuickPromoteEligibilityTweetFields`, `withBirdwatchNotes`, `withVoice`, optional `cursor` | `data.tweetResult.result` (single tweet) / `data.threaded_conversation_with_injections_v2.instructions` (thread) |
| 11 | SearchTimeline | POST+URL | `rawQuery`, `count`, `querySource`, `product`, optional `cursor` | `data.search_by_raw_query.search_timeline.timeline.instructions` |
| 12 | UserArticlesTweets | GET | `userId`, `count`, optional `cursor` | `data.user.result.timeline.timeline.instructions` |
| 13 | UserTweets | GET | `userId`, `count`, `includePromotedContent`, `withQuickPromoteEligibilityTweetFields`, `withVoice`, optional `cursor` | `data.user.result.timeline.timeline.instructions` |
| 14 | Bookmarks | GET | `count`, `includePromotedContent`, `withDownvotePerspective`, `withReactionsMetadata`, `withReactionsPerspective`, optional `cursor` | `data.bookmark_timeline_v2.timeline.instructions` |
| 15 | Following | GET | `userId`, `count: 20`, `includePromotedContent: false`, optional `cursor` | `data.user.result.timeline.timeline.instructions` |
| 16 | Followers | GET | `userId`, `count: 20`, `includePromotedContent: false`, optional `cursor` | `data.user.result.timeline.timeline.instructions` |
| 17 | Likes | GET | `userId`, `count`, `includePromotedContent`, `withClientEventToken`, `withBirdwatchNotes`, `withVoice`, optional `cursor` | `data.user.result.timeline.timeline.instructions` |
| 18 | BookmarkFolderTimeline | GET | `bookmark_collection_id`, `includePromotedContent`, optional `count`, optional `cursor` | `data.bookmark_collection_timeline.timeline.instructions` |
| 19 | ListOwnerships | GET | `userId`, `count: 100`, `isListMembershipShown: true`, `isListMemberTargetUserId` | `data.user.result.timeline.timeline.instructions` |
| 20 | ListMemberships | GET | `userId`, `count: 100`, `isListMembershipShown: true`, `isListMemberTargetUserId` | `data.user.result.timeline.timeline.instructions` |
| 21 | ListLatestTweetsTimeline | GET | `listId`, `count: 20`, optional `cursor` | `data.list.tweets_timeline.timeline.instructions` |
| 22 | ListByRestId | GET | `listId` | `data.list` |
| 23 | HomeTimeline | GET | `count`, `includePromotedContent: true`, `latestControlAvailable: true`, `requestContext: "launch"`, `withCommunity: true`, optional `cursor` | `data.home.home_timeline_urt.instructions` |
| 24 | HomeLatestTimeline | GET | (same as HomeTimeline) | `data.home.home_timeline_urt.instructions` |
| 25 | ExploreSidebar | GET | varies | `data.explore_sidebar` |
| 26 | ExplorePage | GET | varies | `data.explore_page` |
| 27 | GenericTimelineById | GET | `timelineId`, `count` (string, doubled), `includePromotedContent: false` | `data.timeline.timeline.instructions` |
| 28 | TrendHistory | GET | varies | `data.trend_history` |
| 29 | AboutAccountQuery | GET | `screenName` (camelCase) | `data.user_result_by_screen_name.result.about_profile` |

Note: `UserByScreenName` is used internally by gobird but is not in `FallbackQueryIDs` — it uses only hardcoded IDs from `PerOperationFallbackIDs`.
