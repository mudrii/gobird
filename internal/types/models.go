// Package types defines all public normalized output types and wire-level GraphQL response structs.
package types

// TweetData is the normalized representation of a single tweet.
type TweetData struct {
	ID                string
	Text              string
	CreatedAt         string
	ReplyCount        int
	RetweetCount      int
	LikeCount         int
	ConversationID    string
	InReplyToStatusID *string
	Author            TweetAuthor
	AuthorID          string
	// IsBlueVerified is taken from the top-level result field, NOT from legacy (correction #8).
	IsBlueVerified    bool
	QuotedTweet       *TweetData
	Media             []TweetMedia
	Article           *TweetArticle
	// Raw is only populated when includeRaw is true.
	Raw any
}

// TweetAuthor holds the author's display identifiers.
type TweetAuthor struct {
	Username string
	Name     string
}

// TweetMedia holds a single media attachment.
// PreviewURL is set for ANY media that has sizes.small (correction #31).
// Dimensions use sizes.large first, sizes.medium fallback (correction #32).
type TweetMedia struct {
	Type       string // "photo", "video", "animated_gif"
	URL        string
	Width      int
	Height     int
	PreviewURL string // set for any media with sizes.small
	VideoURL   string // only for video / animated_gif
	DurationMs *int
}

// TweetArticle holds metadata extracted from an article tweet.
type TweetArticle struct {
	Title       string
	PreviewText string
}

// TweetWithMeta wraps TweetData with thread-awareness metadata.
type TweetWithMeta struct {
	TweetData
	IsThread       bool
	ThreadPosition string  // "standalone", "root", "middle", "end"
	HasSelfReplies bool
	ThreadRootID   *string
}

// TwitterUser is the normalized representation of a Twitter/X user.
// IsBlueVerified is top-level on result, NOT inside legacy (correction #8).
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

// TwitterList is the normalized representation of a Twitter/X list.
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

// ListOwner identifies the list owner.
type ListOwner struct {
	ID       string
	Username string
	Name     string
}

// NewsItem is the normalized representation of a trending/news item.
type NewsItem struct {
	ID          string
	Headline    string
	Category    string
	TimeAgo     string
	PostCount   *int
	Description string
	URL         string
	IsAiNews    bool
	// Raw is only populated when includeRaw is true.
	Raw any
}

// CurrentUserResult holds basic identity for the authenticated user.
type CurrentUserResult struct {
	ID       string
	Username string
	Name     string
}

// TwitterCookies holds resolved authentication credentials.
type TwitterCookies struct {
	AuthToken    string
	Ct0          string
	CookieHeader string
}
