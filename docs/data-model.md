# Data Model Reference

## Overview

gobird separates its types into two layers:

- **Normalized types** (`internal/types/models.go`, `options.go`, `results.go`): The public-facing structs that flow out of the client to callers. Clean, camelCase JSON, no Twitter API internals.
- **Wire types** (`internal/types/wire.go`): Structs that map directly to the Twitter/X GraphQL response JSON. snake_case keys, deeply nested, full of optional wrappers.

The `internal/parsing` package is the boundary between the two layers. It reads wire types and produces normalized types. Callers never deal with wire types directly.

All normalized types are re-exported through `pkg/bird/types.go` as Go type aliases, so they are identical types — not copies — when seen by library consumers.

---

## Normalized Types (`internal/types/models.go`)

### `TweetData`

The primary output type for any tweet. Returned by search, home timeline, bookmarks, likes, user tweets, thread detail, and tweet lookup.

```go
type TweetData struct {
    ID                string      `json:"id"`
    Text              string      `json:"text"`
    CreatedAt         string      `json:"createdAt,omitempty"`
    ReplyCount        int         `json:"replyCount,omitempty"`
    RetweetCount      int         `json:"retweetCount,omitempty"`
    LikeCount         int         `json:"likeCount,omitempty"`
    ConversationID    string      `json:"conversationId,omitempty"`
    InReplyToStatusID *string     `json:"inReplyToStatusId,omitempty"`
    Author            TweetAuthor `json:"author"`
    AuthorID          string      `json:"authorId,omitempty"`
    IsBlueVerified    bool        `json:"isBlueVerified,omitempty"`
    QuotedTweet       *TweetData  `json:"quotedTweet,omitempty"`
    Media             []TweetMedia  `json:"media,omitempty"`
    Article           *TweetArticle `json:"article,omitempty"`
    Raw               any           `json:"_raw,omitempty"`
}
```

| Field | Source in wire | Notes |
|---|---|---|
| `ID` | `WireRawTweet.RestID` | Twitter's numeric tweet ID as a string |
| `Text` | Priority: article → note tweet → `legacy.full_text` | See `ExtractTweetText` |
| `CreatedAt` | `legacy.created_at` | RFC822 / Twitter date string |
| `ReplyCount` | `legacy.reply_count` | 0 when omitted |
| `RetweetCount` | `legacy.retweet_count` | 0 when omitted |
| `LikeCount` | `legacy.favorite_count` | Mapped from `favorite_count`, not `like_count` |
| `ConversationID` | `legacy.conversation_id_str` | Numeric string; equals root tweet ID |
| `InReplyToStatusID` | `legacy.in_reply_to_status_id_str` | Pointer; nil when not a reply |
| `Author` | `core.user_results.result.legacy` | Flattened to `TweetAuthor{Username, Name}` |
| `AuthorID` | `legacy.user_id_str` | Numeric string user ID of the author |
| `IsBlueVerified` | `WireRawTweet.IsBlueVerified` (top-level) | See note below |
| `QuotedTweet` | `quoted_status_result.result` | Recursive `*TweetData`; depth limited by `QuoteDepth` |
| `Media` | `legacy.extended_entities.media` | Slice; nil when no media |
| `Article` | `article.article_results.result` | Nil when not an article tweet |
| `Raw` | Full response body | Only populated when `IncludeRaw: true` |

**IsBlueVerified — top-level correction**: The `is_blue_verified` field exists at two places in the wire response: at the top level of `WireRawTweet` and potentially inside `legacy` as various other fields. gobird reads it exclusively from the top-level `WireRawTweet.IsBlueVerified`. The `legacy` object does not contain a reliable `is_blue_verified` field — only the outer result does. Using `legacy` would give wrong (always-false) results for blue-verified accounts.

---

### `TweetAuthor`

```go
type TweetAuthor struct {
    Username string `json:"username"`
    Name     string `json:"name"`
}
```

`Username` is the `@handle` (without the `@`). `Name` is the display name. Sourced from `WireUserLegacy.ScreenName` and `WireUserLegacy.Name` respectively.

---

### `TweetMedia`

```go
type TweetMedia struct {
    Type       string `json:"type"`
    URL        string `json:"url"`
    Width      int    `json:"width,omitempty"`
    Height     int    `json:"height,omitempty"`
    PreviewURL string `json:"previewUrl,omitempty"`
    VideoURL   string `json:"videoUrl,omitempty"`
    DurationMs *int   `json:"durationMs,omitempty"`
}
```

| Field | Source | Notes |
|---|---|---|
| `Type` | `WireMedia.Type` | `"photo"`, `"video"`, or `"animated_gif"` |
| `URL` | `WireMedia.MediaURLHttps` | Base media URL (image or poster) |
| `Width` | `sizes.large.w` → `sizes.medium.w` fallback | Large preferred, medium fallback |
| `Height` | `sizes.large.h` → `sizes.medium.h` fallback | Large preferred, medium fallback |
| `PreviewURL` | `MediaURLHttps + ":small"` when `sizes.small != nil` | Set for ANY media with a small size, not just video |
| `VideoURL` | Highest-bitrate `video/mp4` variant URL | Nil when no video info |
| `DurationMs` | `video_info.duration_millis` | Pointer; nil for photos |

**PreviewURL note**: A common misconception is that `previewUrl` only applies to video and GIFs. gobird sets it for any media item that has a `sizes.small` variant, including photos.

**Dimensions note**: The wire response may have `large`, `medium`, `small`, and `thumb` size variants. gobird picks `large` first for reported dimensions, falling back to `medium` if `large` is absent.

**VideoURL selection**: `bestVideoVariant` filters variants to `video/mp4` content type only (ignoring `application/x-mpegURL` m3u8 playlists), then selects the one with the highest bitrate.

---

### `TweetArticle`

```go
type TweetArticle struct {
    Title       string `json:"title"`
    PreviewText string `json:"previewText,omitempty"`
}
```

Populated when the tweet is an article tweet (`WireRawTweet.Article != nil`). `Title` and `PreviewText` come from `article.article_results.result`. When an article is present, it also takes priority for `TweetData.Text` (text = `title + ": " + previewText` or just title if preview is empty).

---

### `TweetWithMeta`

```go
type TweetWithMeta struct {
    TweetData
    IsThread       bool    `json:"isThread"`
    ThreadPosition string  `json:"threadPosition"`
    HasSelfReplies bool    `json:"hasSelfReplies"`
    ThreadRootID   *string `json:"threadRootId,omitempty"`
}
```

Returned only by `GetThread`. Embeds `TweetData` and adds thread-awareness metadata computed by `internal/parsing/thread_filters.go`.

| `ThreadPosition` value | Meaning |
|---|---|
| `"standalone"` | Single tweet, not part of a thread |
| `"root"` | First tweet in a thread |
| `"middle"` | Middle tweet in a thread |
| `"end"` | Last tweet in a thread |

---

### `TwitterUser`

```go
type TwitterUser struct {
    ID              string `json:"id"`
    Username        string `json:"username"`
    Name            string `json:"name"`
    Description     string `json:"description,omitempty"`
    FollowersCount  int    `json:"followersCount,omitempty"`
    FollowingCount  int    `json:"followingCount,omitempty"`
    IsBlueVerified  bool   `json:"isBlueVerified,omitempty"`
    ProfileImageURL string `json:"profileImageUrl,omitempty"`
    CreatedAt       string `json:"createdAt,omitempty"`
    Raw             any    `json:"_raw,omitempty"`
}
```

| Field | Wire source | Notes |
|---|---|---|
| `ID` | `WireRawUser.RestID` | Numeric string user ID |
| `Username` | `WireUserLegacy.ScreenName` | The `@handle` without `@` |
| `Name` | `WireUserLegacy.Name` | Display name |
| `Description` | `WireUserLegacy.Description` | Bio text |
| `FollowersCount` | `WireUserLegacy.FollowersCount` | |
| `FollowingCount` | `WireUserLegacy.FriendsCount` | Wire field is `friends_count` |
| `IsBlueVerified` | `WireRawUser.IsBlueVerified` (top-level) | Same correction as tweet: top-level only |
| `ProfileImageURL` | `WireUserLegacy.ProfileImageURLHTTPS` | |
| `CreatedAt` | `WireUserLegacy.CreatedAt` | Twitter date string |
| `Raw` | Full response | Only when `IncludeRaw: true` |

**IsBlueVerified — same top-level correction**: `WireRawUser.IsBlueVerified` is the reliable field. The `legacy` object contains `verified` (legacy checkmark, now largely irrelevant) but not a reliable `is_blue_verified`. gobird reads exclusively from the top-level result field, not `legacy`.

---

### `TwitterList`

```go
type TwitterList struct {
    ID              string     `json:"id"`
    Name            string     `json:"name"`
    Description     string     `json:"description,omitempty"`
    MemberCount     int        `json:"memberCount,omitempty"`
    SubscriberCount int        `json:"subscriberCount,omitempty"`
    IsPrivate       bool       `json:"isPrivate,omitempty"`
    CreatedAt       string     `json:"createdAt,omitempty"`
    Owner           *ListOwner `json:"owner,omitempty"`
    Raw             any        `json:"_raw,omitempty"`
}
```

Sourced from `WireList`. `IsPrivate` is derived from `WireList.Mode == "Private"`.

---

### `ListOwner`

```go
type ListOwner struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Name     string `json:"name"`
}
```

Sourced from the list's embedded `user_results` field.

---

### `NewsItem`

```go
type NewsItem struct {
    ID          string `json:"id"`
    Headline    string `json:"headline"`
    Category    string `json:"category,omitempty"`
    TimeAgo     string `json:"timeAgo,omitempty"`
    PostCount   *int   `json:"postCount,omitempty"`
    Description string `json:"description,omitempty"`
    URL         string `json:"url,omitempty"`
    IsAiNews    bool   `json:"isAiNews,omitempty"`
    Raw         any    `json:"_raw,omitempty"`
}
```

Returned by `GetNews`. Sourced from the explore timeline's trending/news entries. `PostCount` is a pointer because it may be genuinely absent (0 is a valid count but also the zero value, so pointer distinguishes "not provided" from 0).

---

### `CurrentUserResult`

```go
type CurrentUserResult struct {
    ID       string `json:"id"`
    Username string `json:"username,omitempty"`
    Name     string `json:"name,omitempty"`
}
```

Returned by `GetCurrentUser`. Contains only the authenticated user's basic identity. Used internally by `ensureClientUserID` to populate the `userID` cache.

---

### `TwitterCookies`

```go
type TwitterCookies struct {
    AuthToken    string `json:"authToken"`
    Ct0          string `json:"ct0"`
    CookieHeader string `json:"cookieHeader,omitempty"`
}
```

The output of `ResolveCredentials`. `CookieHeader` is the pre-built `Cookie:` header string (`auth_token=...; ct0=...`). It is convenience-only; the client uses `AuthToken` and `Ct0` directly when building headers.

---

## Options Types (`internal/types/options.go`)

### `FetchOptions`

```go
type FetchOptions struct {
    Cursor      string
    Count       int
    Limit       int
    MaxPages    int
    PageDelayMs int
    IncludeRaw  bool
    QuoteDepth  int
}
```

Base options embedded by or used as-is by most fetch operations.

| Field | Default | Effect |
|---|---|---|
| `Cursor` | `""` | Resume pagination from this cursor; empty means start from beginning |
| `Count` | 0 (operation-specific default, usually 20) | Per-page item count sent to API |
| `Limit` | 0 (unlimited) | Total item cap across all pages; 0 means fetch all available |
| `MaxPages` | 0 (operation-specific; e.g., 10 for follow ops) | Page count ceiling |
| `PageDelayMs` | 0 (operation uses its own default) | Sleep between pages (after page 0) |
| `IncludeRaw` | `false` | When `true`, attaches full response JSON to each item's `Raw` field |
| `QuoteDepth` | 0 (config default is 1) | Quoted tweet expansion depth; 0 disables; 1 expands one level |

---

### `SearchOptions`

```go
type SearchOptions struct {
    FetchOptions
    Product string
}
```

Extends `FetchOptions` for search. `Product` controls the search timeline type:
- `"Latest"` (default) — chronological
- `"Top"` — relevance-ranked
- Other values (e.g., `"People"`, `"Photos"`) pass through to the API

---

### `UserTweetsOptions`

```go
type UserTweetsOptions struct {
    FetchOptions
    IncludeReplies bool
}
```

`IncludeReplies: true` fetches from the `UserArticlesTweets` endpoint instead of `UserTweets`, which includes reply tweets.

---

### `TweetDetailOptions`

```go
type TweetDetailOptions struct {
    IncludeRaw bool
    QuoteDepth int
}
```

Used only for `GetTweet` (single tweet lookup). Does not embed `FetchOptions` because tweet detail is not paginated.

---

### `ThreadOptions`

```go
type ThreadOptions struct {
    FetchOptions
    FilterMode string
}
```

Used by `GetReplies` and `GetThread`. `FilterMode` selects the thread filtering algorithm:

| `FilterMode` value | Behavior |
|---|---|
| `"author_chain"` (default) | Returns only author's chain of self-replies |
| `"author_only"` | Returns only tweets by the thread root's author |
| `"full_chain"` | Returns all replies without filtering |

---

### `NewsOptions`

```go
type NewsOptions struct {
    Tabs       []string
    MaxCount   int
    IncludeRaw bool
}
```

`Tabs` selects which explore timeline tabs to fetch. Default (when nil or empty):
```
["forYou", "news", "sports", "entertainment"]
```

Note: `"trending"` is NOT in the default list. It must be explicitly included.

`MaxCount` limits items per tab. `IncludeRaw` attaches raw responses.

Available tab IDs and their base64-encoded timeline IDs:

| Tab key | Timeline ID |
|---|---|
| `"forYou"` | `VGltZWxpbmU6DAC2CwABAAAAB2Zvcl95b3UAAA==` |
| `"trending"` | `VGltZWxpbmU6DAC2CwABAAAACHRyZW5kaW5nAAA=` |
| `"news"` | `VGltZWxpbmU6DAC2CwABAAAABG5ld3MAAA==` |
| `"sports"` | `VGltZWxpbmU6DAC2CwABAAAABnNwb3J0cwAA` |
| `"entertainment"` | `VGltZWxpbmU6DAC2CwABAAAADWVudGVydGFpbm1lbnQAAA==` |

---

### `BookmarkFolderOptions`

```go
type BookmarkFolderOptions struct {
    FetchOptions
    FolderID string
}
```

`FolderID` is the bookmark collection ID (a string numeric ID). Required for `GetBookmarkFolderTimeline`.

---

## Result Types (`internal/types/results.go`)

### `PageResult[T]`

```go
type PageResult[T any] struct {
    Items      []T
    NextCursor string
    Success    bool
    Error      error
}
```

Returned by single-page fetch methods. `NextCursor` is the opaque cursor string that can be passed as `FetchOptions.Cursor` to fetch the next page. `Success` and `Error` replace the conventional `(T, error)` two-return-value pattern for paginated operations where partial success is meaningful.

Type aliases:
- `TweetPage = PageResult[TweetData]`
- `UserPage = PageResult[TwitterUser]`
- `ListPage = PageResult[TwitterList]`
- `NewsPage = PageResult[NewsItem]`

### `PaginatedResult[T]`

```go
type PaginatedResult[T any] struct {
    Items      []T
    NextCursor string
    Success    bool
    Error      error
}
```

Structurally identical to `PageResult[T]` but semantically different: it accumulates items across multiple pages. `NextCursor` is populated only when pagination was stopped before completion (e.g., `MaxPages` reached or mid-pagination error), indicating where to resume.

Type aliases:
- `TweetResult = PaginatedResult[TweetData]`
- `UserResult = PaginatedResult[TwitterUser]`
- `ListResult = PaginatedResult[TwitterList]`

### When to use each

| Method return type | Meaning |
|---|---|
| `TweetPage` / `UserPage` | Caller gets one page and manages the cursor manually |
| `TweetResult` / `UserResult` | Caller gets all items gobird could collect across pages |
| `(*TweetResult, error)` | Pointer result used by methods that return Go errors (e.g., `GetUserTweets`) |
| `(string, error)` | Simple scalar results like `Tweet` (returns tweet ID) or `GetUserIDByUsername` |

---

## Wire Types (`internal/types/wire.go`)

Wire types are the direct Go representation of Twitter/X GraphQL JSON responses. They are consumed only within `internal/client` and `internal/parsing`. External callers never see them.

### Response Envelope

```go
type WireResponse struct {
    Data   map[string]any `json:"data"`
    Errors []WireError    `json:"errors"`
}
```

The `data` field is typed as `map[string]any` because each operation has a different top-level key. Individual operations unmarshal their own anonymous structs for the specific path they need. `errors` is present even on HTTP 200 responses when GraphQL-level errors occur (e.g., expired query IDs).

```go
type WireError struct {
    Message    string         `json:"message"`
    Locations  []WireLocation `json:"locations"`
    Path       []any          `json:"path"`
    Extensions WireErrorExt   `json:"extensions"`
}

type WireErrorExt struct {
    Code       string `json:"code"`
    Name       string `json:"name"`
    Source     string `json:"source"`
    RetryAfter *int   `json:"retry_after"`
}
```

`Extensions.Code` values relevant to gobird: `GRAPHQL_VALIDATION_FAILED` (triggers search refresh).

---

### Timeline Instruction Types

```go
type WireTimelineInstruction struct {
    Type    string      `json:"type"`
    Entries []WireEntry `json:"entries"`
    Entry   *WireEntry  `json:"entry"`
}
```

A timeline response contains a slice of instructions. Each instruction has a `type` (e.g., `"TimelineAddEntries"`, `"TimelineReplaceEntry"`) and either a bulk `entries` array or a single `entry` pointer. Both paths are searched when extracting cursors.

```go
type WireEntry struct {
    EntryID   string      `json:"entryId"`
    SortIndex string      `json:"sortIndex"`
    Content   WireContent `json:"content"`
}
```

`EntryID` is a string like `"tweet-1234567890"` or `"cursor-top-1234567890"`. `SortIndex` is a numeric string used for ordering.

```go
type WireContent struct {
    EntryType   string           `json:"entryType"`
    TypeName    string           `json:"__typename"`
    CursorType  string           `json:"cursorType"`
    Value       string           `json:"value"`
    ItemContent *WireItemContent `json:"itemContent"`
    Items       []WireItem       `json:"items"`
}
```

A content object is either:
- A tweet entry: `ItemContent != nil` with a tweet result.
- A module entry: `Items` contains multiple `WireItem` each with their own `ItemContent`.
- A cursor entry: `CursorType == "Bottom"` (pagination bottom cursor) or `"Top"`.

Cursor extraction checks `entry.content.cursorType` only — not `entryType`, not module items.

---

### Tweet Result Types

```go
type WireItemContent struct {
    TypeName    string           `json:"__typename"`
    TweetResult *WireTweetResult `json:"tweet_results"`
    UserResult  *WireUserResult  `json:"user_results"`
}

type WireTweetResult struct {
    Result *WireRawTweet `json:"result"`
}

type WireRawTweet struct {
    TypeName       string           `json:"__typename"`
    RestID         string           `json:"rest_id"`
    Core           *WireTweetCore   `json:"core"`
    Legacy         *WireTweetLegacy `json:"legacy"`
    Card           *WireCard        `json:"card"`
    QuotedResult   *WireTweetResult `json:"quoted_status_result"`
    Tweet          *WireRawTweet    `json:"tweet"`  // visibility wrapper
    IsBlueVerified bool             `json:"is_blue_verified"`
    NoteTweet      *WireNoteTweet   `json:"note_tweet"`
    Article        *WireArticle     `json:"article"`
}
```

**Visibility wrapper**: When `__typename == "TweetWithVisibilityResults"`, the actual tweet data is inside the `tweet` field (not at the top level). `UnwrapTweetResult` handles this by checking `raw.Tweet != nil` and returning `raw.Tweet` instead of `raw`. No `__typename` check is performed — only the pointer truthiness check.

```go
type WireTweetLegacy struct {
    FullText             string             `json:"full_text"`
    CreatedAt            string             `json:"created_at"`
    ConversationIDStr    string             `json:"conversation_id_str"`
    InReplyToStatusIDStr *string            `json:"in_reply_to_status_id_str"`
    ReplyCount           int                `json:"reply_count"`
    RetweetCount         int                `json:"retweet_count"`
    FavoriteCount        int                `json:"favorite_count"`
    ExtendedEntities     *WireMediaEntities `json:"extended_entities"`
    UserIDStr            string             `json:"user_id_str"`
}
```

`FavoriteCount` maps to `TweetData.LikeCount`. `InReplyToStatusIDStr` is a pointer because it is absent (not `null`) when the tweet is not a reply.

---

### User Result Types

```go
type WireUserResult struct {
    Result *WireRawUser `json:"result"`
}

type WireRawUser struct {
    TypeName       string          `json:"__typename"`
    RestID         string          `json:"rest_id"`
    Legacy         *WireUserLegacy `json:"legacy"`
    IsBlueVerified bool            `json:"is_blue_verified"`
    User           *WireRawUser    `json:"user"`  // visibility wrapper
}
```

Same visibility wrapper pattern as tweets: `User != nil` means unwrap. `TypeName == "UserUnavailable"` means the account is suspended, deactivated, or blocked — stop immediately (do not attempt further fallbacks).

```go
type WireUserLegacy struct {
    ScreenName           string `json:"screen_name"`
    Name                 string `json:"name"`
    Description          string `json:"description"`
    FollowersCount       int    `json:"followers_count"`
    FriendsCount         int    `json:"friends_count"`
    ProfileImageURLHTTPS string `json:"profile_image_url_https"`
    CreatedAt            string `json:"created_at"`
}
```

`FriendsCount` maps to `TwitterUser.FollowingCount`. This is the Twitter legacy naming for "following count".

---

### Media Types

```go
type WireMediaEntities struct {
    Media []WireMedia `json:"media"`
}

type WireMedia struct {
    Type          string         `json:"type"`
    MediaURLHttps string         `json:"media_url_https"`
    Sizes         WireMediaSizes `json:"sizes"`
    VideoInfo     *WireVideoInfo `json:"video_info"`
}

type WireMediaSizes struct {
    Large  *WireMediaSize `json:"large"`
    Medium *WireMediaSize `json:"medium"`
    Small  *WireMediaSize `json:"small"`
    Thumb  *WireMediaSize `json:"thumb"`
}

type WireMediaSize struct {
    W      int    `json:"w"`
    H      int    `json:"h"`
    Resize string `json:"resize"`
}

type WireVideoInfo struct {
    DurationMillis *int               `json:"duration_millis"`
    Variants       []WireVideoVariant `json:"variants"`
}

type WireVideoVariant struct {
    ContentType string `json:"content_type"`
    URL         string `json:"url"`
    Bitrate     *int   `json:"bitrate"`
}
```

All size variants are pointers because any of them may be absent. `DurationMillis` and `Bitrate` are pointers for the same reason.

---

### Card Types (Article Detection)

```go
type WireCard struct {
    RestID string          `json:"rest_id"`
    Legacy *WireCardLegacy `json:"legacy"`
}

type WireCardLegacy struct {
    BindingValues []WireBindingValue `json:"binding_values"`
}

type WireBindingValue struct {
    Key   string          `json:"key"`
    Value WireBindingData `json:"value"`
}

type WireBindingData struct {
    StringValue  *string         `json:"string_value"`
    ImageValue   *WireImageValue `json:"image_value"`
    BooleanValue *bool           `json:"boolean_value"`
}
```

Twitter cards are used for link previews and article tweets. The `binding_values` array is a key-value list where each value is a tagged union (`string_value`, `image_value`, or `boolean_value`). `internal/parsing/article.go` extracts article title and preview text from specific card binding keys.

---

### Note Tweet Types (Long-form)

```go
type WireNoteTweet struct {
    NoteTweetResults WireNoteTweetResults `json:"note_tweet_results"`
}

type WireNoteTweetResults struct {
    Result *WireNoteTweetResult `json:"result"`
}

type WireNoteTweetResult struct {
    Text string `json:"text"`
}
```

Note tweets (Twitter's long-form post type) carry their full text here instead of in `legacy.full_text` (which is truncated). `ExtractTweetText` checks note tweet before falling back to legacy.

---

### Article Types

```go
type WireArticle struct {
    ArticleResults WireArticleResults `json:"article_results"`
}

type WireArticleResults struct {
    Result *WireArticleResult `json:"result"`
}

type WireArticleResult struct {
    Title        string `json:"title"`
    PreviewText  string `json:"preview_text"`
    ContentState string `json:"content_state"`
}
```

Article tweets (distinct from linked articles in cards) carry their metadata here. When present, `ExtractTweetText` uses the article title (and preview text) as the tweet's text, giving priority over note tweet and legacy text.

---

### List Wire Type

```go
type WireList struct {
    IDStr           string         `json:"id_str"`
    Name            string         `json:"name"`
    Description     string         `json:"description"`
    MemberCount     int            `json:"member_count"`
    SubscriberCount int            `json:"subscriber_count"`
    Mode            string         `json:"mode"`
    CreatedAt       string         `json:"created_at"`
    UserResults     WireUserResult `json:"user_results"`
}
```

`Mode` is `"Public"` or `"Private"`. Mapped to `TwitterList.IsPrivate = (Mode == "Private")`.

---

## JSON Tag Conventions

### Normalized types

- camelCase keys: `"id"`, `"createdAt"`, `"replyCount"`, `"isBlueVerified"`
- `omitempty` on all optional fields to keep JSON output clean
- Pointer fields for optional values: `InReplyToStatusID *string`, `QuotedTweet *TweetData`, `DurationMs *int`
- `_raw` prefix on the raw payload field to signal it is internal / non-standard

### Wire types

- snake_case keys matching the Twitter/X API: `"full_text"`, `"created_at"`, `"conversation_id_str"`, `"is_blue_verified"`
- No `omitempty` — wire types are only read (unmarshaled), never written
- Pointer fields for genuinely optional nested objects: `*WireTweetCore`, `*WireTweetLegacy`, `*WireCard`

---

## Pointer vs Value Semantics for Optional Fields

| Pattern | Example | Reason |
|---|---|---|
| `*string` for optional IDs | `InReplyToStatusID *string` | Must distinguish "no reply" (nil) from empty string (invalid but possible) |
| `*int` for optional counts | `PostCount *int`, `DurationMs *int`, `Bitrate *int` | Must distinguish "absent" from 0 (which is a valid value) |
| `*TweetData` for embedded | `QuotedTweet *TweetData` | Tree structure; nil = no quoted tweet |
| `*TweetArticle` | `Article *TweetArticle` | nil = not an article tweet |
| `*WireMediaSize` | `Large, Medium, Small, Thumb *WireMediaSize` | Any size variant may be absent from the API response |
| `*int` `DurationMillis` | `WireVideoInfo.DurationMillis` | Photos have no duration; API omits the field |
| Value for required | `ID string`, `Text string`, `Author TweetAuthor` | Always present on a valid tweet |
| Value for nested structs with all-optional content | `WireTweetCore`, `WireMediaSizes` | Struct presence implies the parent was present |

---

## `IsBlueVerified` — Why It Differs from Naive Expectation

A developer reading the Twitter API response for the first time typically notices a `legacy` object and expects all user/tweet flags to live there. The `legacy` object does contain `verified` (the old blue checkmark from before 2023) but does NOT contain a reliable `is_blue_verified` field.

The `is_blue_verified` field in gobird is sourced from the **top-level result object**:

For tweets:
```
WireRawTweet.IsBlueVerified  (is_blue_verified at top level)
NOT: WireRawTweet.Legacy.something
```

For users:
```
WireRawUser.IsBlueVerified   (is_blue_verified at top level)
NOT: WireRawUser.Legacy.verified
```

`legacy.verified` is the pre-2023 legacy verification checkmark (now largely unused). `is_blue_verified` at the top level reflects current Twitter Blue / X Premium subscription status.

If you read from `legacy.verified`, blue-verified accounts will appear unverified. If you read from the top-level field, you get the correct current status.

This is documented in the source as "correction #8" and applies identically to both `TweetData.IsBlueVerified` and `TwitterUser.IsBlueVerified`.
