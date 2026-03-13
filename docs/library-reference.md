# gobird Go Library Reference

`pkg/bird` is the public Go API for gobird. It wraps the internal Twitter/X client and exposes a clean, typed surface for use in other Go programs.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Constructors](#constructors)
- [ClientOptions](#clientoptions)
- [Auth Helpers](#auth-helpers)
- [Methods by Category](#methods-by-category)
  - [Tweet Operations](#tweet-operations)
  - [Timeline Operations](#timeline-operations)
  - [Search](#search)
  - [User Operations](#user-operations)
  - [Post Operations](#post-operations)
  - [Engagement Operations](#engagement-operations)
  - [List Operations](#list-operations)
  - [News and Trending](#news-and-trending)
- [Options Types](#options-types)
- [Result and Model Types](#result-and-model-types)
- [Error Handling](#error-handling)
- [Pagination](#pagination)

---

## Installation

```sh
go get github.com/mudrii/gobird
```

The minimum Go version is **1.26.1** (as declared in `go.mod`).

Import path:

```go
import "github.com/mudrii/gobird/pkg/bird"
```

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mudrii/gobird/pkg/bird"
)

func main() {
    // Resolve credentials from env vars or browser cookies automatically.
    creds, err := bird.ResolveCredentials(bird.ResolveOptions{
        Browser: "safari", // or "chrome", "firefox"
    })
    if err != nil {
        log.Fatalf("auth: %v", err)
    }

    client, err := bird.New(creds, nil)
    if err != nil {
        log.Fatalf("client: %v", err)
    }

    ctx := context.Background()

    // Read a tweet.
    tweet, err := client.GetTweet(ctx, "1234567890123456789", nil)
    if err != nil {
        log.Fatalf("get tweet: %v", err)
    }
    fmt.Printf("@%s: %s\n", tweet.Author.Username, tweet.Text)

    // Search tweets.
    result := client.GetAllSearchResults(ctx, "golang", &bird.SearchOptions{
        FetchOptions: bird.FetchOptions{Limit: 20},
    })
    for _, t := range result.Items {
        fmt.Printf("@%s: %s\n", t.Author.Username, t.Text)
    }
}
```

---

## Constructors

### New

```go
func New(creds *TwitterCookies, opts *ClientOptions) (*Client, error)
```

Creates a `Client` from a `*TwitterCookies` struct returned by `ResolveCredentials` or one of the browser extraction helpers.

Returns `error` if `creds` is nil or if either `AuthToken` or `Ct0` is empty.

```go
creds, _ := bird.ResolveCredentials(bird.ResolveOptions{Browser: "safari"})
client, err := bird.New(creds, &bird.ClientOptions{
    TimeoutMs: 60000,
})
```

### NewWithTokens

```go
func NewWithTokens(authToken, ct0 string, opts *ClientOptions) (*Client, error)
```

Creates a `Client` from bare token strings. Equivalent to `New` with a manually constructed `TwitterCookies`.

Returns `error` if either token is empty.

```go
client, err := bird.NewWithTokens(
    "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
    "abc123def456abc123def456abc123def456abc123",
    nil,
)
```

---

## ClientOptions

```go
type ClientOptions struct {
    // HTTPClient overrides the default http.Client.
    // Useful for testing, proxies, or custom transport configurations.
    HTTPClient *http.Client

    // QueryIDCache seeds the runtime query ID cache.
    // Primarily used in tests to avoid live scraping.
    QueryIDCache map[string]string

    // TimeoutMs sets the HTTP request timeout in milliseconds.
    // Ignored when HTTPClient is provided.
    // Default: 30000 (30 seconds).
    TimeoutMs int
}
```

Pass `nil` to accept all defaults.

---

## Auth Helpers

These functions live in `pkg/bird` and are thin wrappers around the internal `auth` package.

### ResolveCredentials

```go
func ResolveCredentials(opts ResolveOptions) (*TwitterCookies, error)
```

Resolves `auth_token` and `ct0` in priority order:

1. `opts.FlagAuthToken` + `opts.FlagCt0` (both must be non-empty)
2. Environment variables: `AUTH_TOKEN` > `TWITTER_AUTH_TOKEN`; `CT0` > `TWITTER_CT0`
3. Browser cookie extraction (using `opts.Browser` or `opts.CookieSources`)

Returns an error if no valid credentials are found after all tiers.

```go
// From Safari cookies:
creds, err := bird.ResolveCredentials(bird.ResolveOptions{
    Browser: "safari",
})

// From environment variables (AUTH_TOKEN + CT0 must be set):
creds, err := bird.ResolveCredentials(bird.ResolveOptions{})

// Explicit tokens:
creds, err := bird.ResolveCredentials(bird.ResolveOptions{
    FlagAuthToken: "a1b2c3...",
    FlagCt0:       "abc123...",
})

// Try Chrome first, then Firefox, with a 5-second timeout:
creds, err := bird.ResolveCredentials(bird.ResolveOptions{
    CookieSources:   []string{"chrome", "firefox"},
    ChromeProfile:   "Default",
    CookieTimeoutMs: 5000,
})
```

### ResolveOptions

```go
type ResolveOptions struct {
    // FlagAuthToken is the explicit auth_token value (highest priority).
    FlagAuthToken string

    // FlagCt0 is the explicit ct0 value (highest priority, paired with FlagAuthToken).
    FlagCt0 string

    // Browser selects a single browser: "safari", "chrome", or "firefox".
    // Empty means try all in default order (safari → chrome → firefox).
    Browser string

    // CookieSources overrides the browser extraction order.
    // Valid values: "safari", "chrome", "firefox".
    CookieSources []string

    // ChromeProfile is a Chrome profile name or directory path hint.
    ChromeProfile string

    // FirefoxProfile is a Firefox profile name or directory name hint.
    FirefoxProfile string

    // CookieTimeoutMs aborts browser cookie extraction after this many ms (0 = no timeout).
    CookieTimeoutMs int
}
```

### ExtractSafariCookies

```go
func ExtractSafariCookies() (*TwitterCookies, error)
```

Directly extracts cookies from Safari's SQLite cookie store. macOS only. Looks for the DB at:
- `~/Library/Containers/com.apple.Safari/Data/Library/Cookies/Cookies.db`
- Fallback: `~/Library/Cookies/Cookies.db`

### ExtractChromeCookies

```go
func ExtractChromeCookies(profileHint string) (*TwitterCookies, error)
```

Extracts cookies from Chrome or Chromium. The `profileHint` parameter selects a specific profile:

- `""` — uses `Default` profile
- `"Profile 1"` — uses that named profile
- `"/absolute/path"` — treats as a directory, appends `Cookies`
- `"/abs/path/Cookies"` or `"/abs/path/file.sqlite"` — uses that file directly

macOS only. Decrypts cookie values using AES-128-CBC with a key from the macOS Keychain (`Chrome Safe Storage`).

### ExtractFirefoxCookies

```go
func ExtractFirefoxCookies(profileHint string) (*TwitterCookies, error)
```

Extracts cookies from Firefox's `cookies.sqlite`. The `profileHint` filters profiles by name substring. Reads all matching profiles and merges cookies.

---

## Methods by Category

All methods accept a `context.Context` as the first argument. Canceling the context aborts the HTTP request.

### Tweet Operations

#### GetTweet

```go
func (c *Client) GetTweet(ctx context.Context, tweetID string, opts *TweetDetailOptions) (*TweetData, error)
```

Fetches a single tweet by its numeric ID. Returns `*TweetData` or an error.

```go
tweet, err := client.GetTweet(ctx, "1234567890123456789", &bird.TweetDetailOptions{
    QuoteDepth: 2,     // expand quoted tweets 2 levels deep
    IncludeRaw: false, // set true to populate tweet._raw
})
```

#### GetReplies

```go
func (c *Client) GetReplies(ctx context.Context, tweetID string, opts *ThreadOptions) (*TweetResult, error)
```

Returns all replies to the given tweet. Paginates automatically using `opts.FetchOptions`.

```go
result, err := client.GetReplies(ctx, "1234567890123456789", &bird.ThreadOptions{
    FetchOptions: bird.FetchOptions{Limit: 50},
})
for _, t := range result.Items {
    fmt.Printf("@%s: %s\n", t.Author.Username, t.Text)
}
```

#### GetThread

```go
func (c *Client) GetThread(ctx context.Context, tweetID string, opts *ThreadOptions) ([]TweetWithMeta, error)
```

Fetches all tweets in a thread. `opts.FilterMode` selects filtering behavior:

| FilterMode | Behavior |
|-----------|----------|
| `"author_chain"` (default) | Only tweets by the thread's original author |
| `"full_chain"` | All tweets in the conversation chain |

Returns `[]TweetWithMeta` where each element includes thread position metadata.

```go
tweets, err := client.GetThread(ctx, "1234567890123456789", &bird.ThreadOptions{
    FetchOptions: bird.FetchOptions{Limit: 100},
    FilterMode:   "full_chain",
})
for _, t := range tweets {
    fmt.Printf("[%s] @%s: %s\n", t.ThreadPosition, t.Author.Username, t.Text)
}
```

---

### Timeline Operations

#### GetHomeTimeline

```go
func (c *Client) GetHomeTimeline(ctx context.Context, opts *FetchOptions) TweetResult
```

Fetches the authenticated user's home timeline (algorithmic feed). Returns `TweetResult` (not a pointer). Does not expose pagination cursors to callers.

```go
result := client.GetHomeTimeline(ctx, &bird.FetchOptions{Limit: 30})
if result.Error != nil {
    log.Fatal(result.Error)
}
for _, t := range result.Items {
    fmt.Println(t.Text)
}
```

#### GetHomeLatestTimeline

```go
func (c *Client) GetHomeLatestTimeline(ctx context.Context, opts *FetchOptions) TweetResult
```

Fetches the authenticated user's chronological "Following" timeline.

```go
result := client.GetHomeLatestTimeline(ctx, &bird.FetchOptions{Limit: 20})
```

#### GetBookmarks

```go
func (c *Client) GetBookmarks(ctx context.Context, opts *FetchOptions) TweetResult
```

Fetches bookmarked tweets for the authenticated user.

```go
result := client.GetBookmarks(ctx, &bird.FetchOptions{Limit: 100})
```

#### GetBookmarkFolderTimeline

```go
func (c *Client) GetBookmarkFolderTimeline(ctx context.Context, opts *BookmarkFolderOptions) TweetResult
```

Fetches tweets from a specific bookmark folder.

```go
result := client.GetBookmarkFolderTimeline(ctx, &bird.BookmarkFolderOptions{
    FetchOptions: bird.FetchOptions{Limit: 50},
    FolderID:     "VjlhbGxhYnVzAAA=",
})
```

#### GetLikes

```go
func (c *Client) GetLikes(ctx context.Context, opts *FetchOptions) TweetResult
```

Fetches tweets liked by the authenticated user.

```go
result := client.GetLikes(ctx, &bird.FetchOptions{Limit: 50})
```

#### GetUserTweets

```go
func (c *Client) GetUserTweets(ctx context.Context, userID string, opts *UserTweetsOptions) (*TweetResult, error)
```

Fetches tweets from the specified user's timeline. Requires a numeric user ID (use `GetUserIDByUsername` to resolve a handle first).

```go
userID, err := client.GetUserIDByUsername(ctx, "golang")
if err != nil {
    log.Fatal(err)
}
result, err := client.GetUserTweets(ctx, userID, &bird.UserTweetsOptions{
    FetchOptions:   bird.FetchOptions{Limit: 50},
    IncludeReplies: false,
})
```

---

### Search

#### Search

```go
func (c *Client) Search(ctx context.Context, query string, opts *SearchOptions) (*TweetPage, error)
```

Fetches a single page of search results. Returns a `*TweetPage` with a `NextCursor` for manual pagination.

```go
page, err := client.Search(ctx, "golang", &bird.SearchOptions{
    FetchOptions: bird.FetchOptions{Count: 20},
    Product:      "Latest", // "Top" | "Latest" | "People" | "Photos" | "Videos"
})
```

#### GetAllSearchResults

```go
func (c *Client) GetAllSearchResults(ctx context.Context, query string, opts *SearchOptions) TweetResult
```

Fetches all pages of search results up to `opts.Limit` or `opts.MaxPages`. Accumulates items across pages and deduplicates by ID.

```go
result := client.GetAllSearchResults(ctx, "#golang", &bird.SearchOptions{
    FetchOptions: bird.FetchOptions{
        Limit:    200,
        MaxPages: 10,
    },
})
if result.Error != nil {
    log.Fatal(result.Error)
}
fmt.Printf("Found %d tweets\n", len(result.Items))
```

---

### User Operations

#### GetUserIDByUsername

```go
func (c *Client) GetUserIDByUsername(ctx context.Context, username string) (string, error)
```

Resolves a Twitter handle to a numeric user ID. Does not accept the `@` prefix in `username`.

```go
id, err := client.GetUserIDByUsername(ctx, "golang")
// id = "114751940"
```

#### GetUserAboutAccount

```go
func (c *Client) GetUserAboutAccount(ctx context.Context, username string) (*TwitterUser, error)
```

Fetches profile information for the given handle using the `AboutAccountQuery` endpoint. Returns a `*TwitterUser` with ID, username, name, description, follower/following counts, profile image URL, and creation date.

```go
user, err := client.GetUserAboutAccount(ctx, "golang")
fmt.Printf("@%s has %d followers\n", user.Username, user.FollowersCount)
```

#### GetCurrentUser

```go
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUserResult, error)
```

Returns the authenticated user's ID, username, and name. The result is cached internally after the first successful call.

```go
me, err := client.GetCurrentUser(ctx)
fmt.Printf("Authenticated as @%s (ID: %s)\n", me.Username, me.ID)
```

#### GetFollowing

```go
func (c *Client) GetFollowing(ctx context.Context, userID string, opts *FetchOptions) (*UserResult, error)
```

Returns users that the given account follows. Paginates automatically up to `opts.MaxPages` (default: 10).

```go
result, err := client.GetFollowing(ctx, "114751940", &bird.FetchOptions{Limit: 500})
for _, u := range result.Items {
    fmt.Printf("@%s\n", u.Username)
}
```

#### GetFollowers

```go
func (c *Client) GetFollowers(ctx context.Context, userID string, opts *FetchOptions) (*UserResult, error)
```

Returns users who follow the given account.

```go
result, err := client.GetFollowers(ctx, "114751940", &bird.FetchOptions{Limit: 100})
```

#### Follow

```go
func (c *Client) Follow(ctx context.Context, userID string) error
```

Follows the user with the given numeric ID.

```go
err := client.Follow(ctx, "114751940")
```

#### Unfollow

```go
func (c *Client) Unfollow(ctx context.Context, userID string) error
```

Unfollows the user with the given numeric ID.

```go
err := client.Unfollow(ctx, "114751940")
```

---

### Post Operations

#### Tweet

```go
func (c *Client) Tweet(ctx context.Context, text string) (string, error)
```

Posts a new tweet with the given text. Returns the new tweet's numeric ID string.

```go
id, err := client.Tweet(ctx, "Hello from gobird!")
fmt.Println("Posted:", id)
```

#### Reply

```go
func (c *Client) Reply(ctx context.Context, text, inReplyToID string) (string, error)
```

Posts a reply to the tweet with ID `inReplyToID`. Returns the new tweet ID.

```go
id, err := client.Reply(ctx, "Great point!", "1234567890123456789")
```

#### TweetWithMedia

```go
func (c *Client) TweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error)
```

Posts a tweet with pre-uploaded media. The `mediaIDs` are obtained from `UploadMedia`. Up to 4 media IDs may be attached.

```go
data, _ := os.ReadFile("photo.jpg")
mediaID, err := client.UploadMedia(ctx, data, "image/jpeg", "A photo")
if err != nil {
    log.Fatal(err)
}
id, err := client.TweetWithMedia(ctx, "Look at this!", []string{mediaID})
```

#### ReplyWithMedia

```go
func (c *Client) ReplyWithMedia(ctx context.Context, text, inReplyToID string, mediaIDs []string) (string, error)
```

Posts a reply with pre-uploaded media.

#### UploadMedia

```go
func (c *Client) UploadMedia(ctx context.Context, data []byte, mimeType, altText string) (string, error)
```

Uploads media bytes and returns a media ID string for use with `TweetWithMedia` or `ReplyWithMedia`.

- `mimeType`: must start with `image/`, `video/`, or `audio/`
- `altText`: accessibility description (may be empty)
- Maximum file size enforced by the CLI: 512 MiB

```go
data, _ := os.ReadFile("video.mp4")
mediaID, err := client.UploadMedia(ctx, data, "video/mp4", "")
```

---

### Engagement Operations

#### Like

```go
func (c *Client) Like(ctx context.Context, tweetID string) error
```

Likes the given tweet.

```go
err := client.Like(ctx, "1234567890123456789")
```

#### Unlike

```go
func (c *Client) Unlike(ctx context.Context, tweetID string) error
```

Removes a like from the given tweet.

#### Retweet

```go
func (c *Client) Retweet(ctx context.Context, tweetID string) (string, error)
```

Retweets the given tweet. Returns the retweet's ID.

```go
retweetID, err := client.Retweet(ctx, "1234567890123456789")
```

#### Unretweet

```go
func (c *Client) Unretweet(ctx context.Context, tweetID string) error
```

Undoes a retweet. `tweetID` is the original source tweet ID, not the retweet ID.

```go
err := client.Unretweet(ctx, "1234567890123456789")
```

#### Bookmark

```go
func (c *Client) Bookmark(ctx context.Context, tweetID string) error
```

Adds the tweet to the authenticated user's bookmarks.

```go
err := client.Bookmark(ctx, "1234567890123456789")
```

#### Unbookmark

```go
func (c *Client) Unbookmark(ctx context.Context, tweetID string) error
```

Removes the tweet from the authenticated user's bookmarks.

---

### List Operations

#### GetOwnedLists

```go
func (c *Client) GetOwnedLists(ctx context.Context, opts *FetchOptions) (*ListResult, error)
```

Returns Twitter lists owned by the authenticated user.

```go
result, err := client.GetOwnedLists(ctx, nil)
for _, l := range result.Items {
    fmt.Printf("%s [%s] - %d members\n", l.Name, l.ID, l.MemberCount)
}
```

#### GetListMemberships

```go
func (c *Client) GetListMemberships(ctx context.Context, opts *FetchOptions) (*ListResult, error)
```

Returns lists that the authenticated user is a member of.

```go
result, err := client.GetListMemberships(ctx, nil)
```

#### GetListTimeline

```go
func (c *Client) GetListTimeline(ctx context.Context, listID string, opts *FetchOptions) (*TweetResult, error)
```

Fetches the tweet timeline for the given list.

```go
result, err := client.GetListTimeline(ctx, "1234567890123456789", &bird.FetchOptions{
    Limit: 50,
})
```

---

### News and Trending

#### GetNews

```go
func (c *Client) GetNews(ctx context.Context, opts *NewsOptions) ([]NewsItem, error)
```

Fetches news items from Explore tabs. Items are deduplicated across tabs.

Default tabs when `opts.Tabs` is empty: `["forYou", "news", "sports", "entertainment"]`.

Available tab names and their meanings:

| Tab name | Content |
|----------|---------|
| `forYou` | Personalized For You explore feed |
| `news` | News tab |
| `sports` | Sports tab |
| `entertainment` | Entertainment tab |
| `trending` | Trending topics |

```go
items, err := client.GetNews(ctx, &bird.NewsOptions{
    Tabs:     []string{"news", "trending"},
    MaxCount: 20,
})
for _, item := range items {
    fmt.Printf("%s (%s)\n", item.Headline, item.Category)
}
```

---

## Options Types

### FetchOptions

Controls pagination and output for most listing operations.

```go
type FetchOptions struct {
    // Cursor is the pagination cursor from a previous page's NextCursor.
    Cursor string

    // Count is the per-page item count sent to the API. Default depends on operation.
    Count int

    // Limit caps the total number of items accumulated across all pages.
    // 0 means unlimited (fetch all available pages up to MaxPages).
    Limit int

    // MaxPages caps the number of API pages fetched.
    // 0 means unlimited. Most operations default to 10 pages when MaxPages is 0.
    MaxPages int

    // PageDelayMs sleeps this many milliseconds before each page after the first.
    // Useful for rate limit avoidance.
    PageDelayMs int

    // IncludeRaw attaches the raw API JSON response to each result item's _raw field.
    IncludeRaw bool

    // QuoteDepth controls recursive expansion of quoted tweets.
    // 0 = no expansion, 1 = expand one level (default), 2+ = deeper.
    QuoteDepth int
}
```

### SearchOptions

Extends `FetchOptions` for search operations.

```go
type SearchOptions struct {
    FetchOptions

    // Product selects the search timeline type.
    // Valid values: "Latest", "Top", "People", "Photos", "Videos"
    // Default (empty string): API default behavior ("Latest").
    Product string
}
```

### TweetDetailOptions

Controls single-tweet fetching behavior.

```go
type TweetDetailOptions struct {
    // IncludeRaw populates the _raw field with the raw API response.
    IncludeRaw bool

    // QuoteDepth controls quoted tweet expansion depth.
    QuoteDepth int
}
```

### ThreadOptions

Extends `FetchOptions` for thread and reply operations.

```go
type ThreadOptions struct {
    FetchOptions

    // FilterMode selects the thread filtering algorithm.
    // "author_chain" (default): only tweets by the thread's original author.
    // "full_chain": all tweets in the conversation.
    FilterMode string
}
```

### UserTweetsOptions

Extends `FetchOptions` for user timeline fetching.

```go
type UserTweetsOptions struct {
    FetchOptions

    // IncludeReplies includes reply tweets in the results.
    IncludeReplies bool
}
```

### NewsOptions

Controls news/trending fetches.

```go
type NewsOptions struct {
    // Tabs lists which Explore tab names to fetch from.
    // Default: ["forYou", "news", "sports", "entertainment"]
    Tabs []string

    // MaxCount is the per-tab item count sent to the API.
    // Default: 20.
    MaxCount int

    // IncludeRaw attaches the raw API response to each NewsItem's _raw field.
    IncludeRaw bool
}
```

### BookmarkFolderOptions

Extends `FetchOptions` for bookmark folder timelines.

```go
type BookmarkFolderOptions struct {
    FetchOptions

    // FolderID is the bookmark collection ID (opaque string from the API).
    FolderID string
}
```

---

## Result and Model Types

### TweetData

The normalized representation of a single tweet.

```go
type TweetData struct {
    ID                string       `json:"id"`
    Text              string       `json:"text"`
    CreatedAt         string       `json:"createdAt,omitempty"`
    ReplyCount        int          `json:"replyCount,omitempty"`
    RetweetCount      int          `json:"retweetCount,omitempty"`
    LikeCount         int          `json:"likeCount,omitempty"`
    ConversationID    string       `json:"conversationId,omitempty"`
    InReplyToStatusID *string      `json:"inReplyToStatusId,omitempty"`
    Author            TweetAuthor  `json:"author"`
    AuthorID          string       `json:"authorId,omitempty"`
    IsBlueVerified    bool         `json:"isBlueVerified,omitempty"`
    QuotedTweet       *TweetData   `json:"quotedTweet,omitempty"`
    Media             []TweetMedia `json:"media,omitempty"`
    Article           *TweetArticle `json:"article,omitempty"`
    Raw               any          `json:"_raw,omitempty"` // populated when IncludeRaw=true
}
```

### TweetAuthor

```go
type TweetAuthor struct {
    Username string `json:"username"`
    Name     string `json:"name"`
}
```

### TweetMedia

```go
type TweetMedia struct {
    Type       string `json:"type"`        // "photo", "video", "animated_gif"
    URL        string `json:"url"`
    Width      int    `json:"width,omitempty"`
    Height     int    `json:"height,omitempty"`
    PreviewURL string `json:"previewUrl,omitempty"` // thumbnail for video/gif
    VideoURL   string `json:"videoUrl,omitempty"`   // direct video URL
    DurationMs *int   `json:"durationMs,omitempty"` // video duration
}
```

### TweetArticle

```go
type TweetArticle struct {
    Title       string `json:"title"`
    PreviewText string `json:"previewText,omitempty"`
}
```

### TweetWithMeta

Wraps `TweetData` with thread-awareness metadata, returned by `GetThread`.

```go
type TweetWithMeta struct {
    TweetData
    IsThread       bool    `json:"isThread"`
    ThreadPosition string  `json:"threadPosition"` // "standalone", "root", "middle", "end"
    HasSelfReplies bool    `json:"hasSelfReplies"`
    ThreadRootID   *string `json:"threadRootId,omitempty"`
}
```

### TwitterUser

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

### TwitterList

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

type ListOwner struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Name     string `json:"name"`
}
```

### NewsItem

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

### TwitterCookies

```go
type TwitterCookies struct {
    AuthToken    string `json:"authToken"`
    Ct0          string `json:"ct0"`
    CookieHeader string `json:"cookieHeader,omitempty"` // pre-formatted "auth_token=...; ct0=..."
}
```

### PageResult

Single page from a paginated operation.

```go
type PageResult[T any] struct {
    Items      []T
    NextCursor string // pass to FetchOptions.Cursor for the next page; empty = last page
    Success    bool
    Error      error
}

// Convenience type aliases:
type TweetPage = PageResult[TweetData]
type UserPage  = PageResult[TwitterUser]
type ListPage  = PageResult[TwitterList]
type NewsPage  = PageResult[NewsItem]
```

### PaginatedResult

Accumulates items across multiple pages.

```go
type PaginatedResult[T any] struct {
    Items      []T
    NextCursor string // non-empty when MaxPages was hit before exhausting results
    Success    bool
    Error      error
}

// Convenience type aliases:
type TweetResult = PaginatedResult[TweetData]
type UserResult  = PaginatedResult[TwitterUser]
type ListResult  = PaginatedResult[TwitterList]
```

### CurrentUserResult

```go
type CurrentUserResult struct {
    ID       string `json:"id"`
    Username string `json:"username,omitempty"`
    Name     string `json:"name,omitempty"`
}
```

---

## Error Handling

All methods return standard Go errors. There is one sentinel error at the `pkg/bird` level:

```
bird: auth_token and ct0 are required
```

This is returned by `New` and `NewWithTokens` when either token is empty.

All other errors originate from the internal `client` package and are wrapped with context. Common patterns:

```go
tweet, err := client.GetTweet(ctx, tweetID, nil)
if err != nil {
    // Inspect the error message for context:
    // - "404" or "not found" — tweet doesn't exist or was deleted
    // - "401" or "forbidden" — authentication failed
    // - "rate limit" — too many requests
    // - context.Canceled — context was canceled
    // - context.DeadlineExceeded — timeout reached
    log.Printf("failed to fetch tweet: %v", err)
    return
}
```

For `PaginatedResult` (returned by timeline and search operations), check `result.Error` rather than a function return value:

```go
result := client.GetHomeTimeline(ctx, nil)
if result.Error != nil {
    log.Fatal(result.Error)
}
// result.Items may still be partially populated even when Error != nil
// if some pages succeeded before a failure.
```

---

## Pagination

Most listing operations paginate automatically. The `FetchOptions.Limit` and `FetchOptions.MaxPages` fields control how many items are fetched.

### Automatic pagination (recommended)

```go
// Fetch up to 500 tweets from a user's timeline, across however many pages needed:
result, err := client.GetUserTweets(ctx, userID, &bird.UserTweetsOptions{
    FetchOptions: bird.FetchOptions{
        Limit:    500,
        MaxPages: 25,
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Got %d tweets\n", len(result.Items))
```

### Manual pagination (Search only)

For `Search` (single-page), you can paginate manually using `NextCursor`:

```go
opts := &bird.SearchOptions{
    FetchOptions: bird.FetchOptions{Count: 20},
}

for {
    page, err := client.Search(ctx, "golang", opts)
    if err != nil {
        log.Fatal(err)
    }
    for _, t := range page.Items {
        fmt.Println(t.Text)
    }
    if page.NextCursor == "" {
        break
    }
    opts.FetchOptions.Cursor = page.NextCursor
}
```

### Page delays

To be gentle on rate limits when fetching many pages, use `PageDelayMs`:

```go
result, err := client.GetFollowers(ctx, userID, &bird.FetchOptions{
    Limit:       10000,
    MaxPages:    100,
    PageDelayMs: 500, // wait 500ms between pages
})
```
