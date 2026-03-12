// Package types defines all public normalized output types and wire-level GraphQL response structs.
package types

// TweetData is the normalized representation of a single tweet.
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
	// IsBlueVerified is taken from the top-level result field, NOT from legacy (correction #8).
	IsBlueVerified bool          `json:"isBlueVerified,omitempty"`
	QuotedTweet    *TweetData    `json:"quotedTweet,omitempty"`
	Media          []TweetMedia  `json:"media,omitempty"`
	Article        *TweetArticle `json:"article,omitempty"`
	// Raw is only populated when includeRaw is true.
	Raw any `json:"_raw,omitempty"`
}

// TweetAuthor holds the author's display identifiers.
type TweetAuthor struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}

// TweetMedia holds a single media attachment.
// PreviewURL is set for ANY media that has sizes.small (correction #31).
// Dimensions use sizes.large first, sizes.medium fallback (correction #32).
type TweetMedia struct {
	Type       string `json:"type"` // "photo", "video", "animated_gif"
	URL        string `json:"url"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	PreviewURL string `json:"previewUrl,omitempty"`
	VideoURL   string `json:"videoUrl,omitempty"`
	DurationMs *int   `json:"durationMs,omitempty"`
}

// TweetArticle holds metadata extracted from an article tweet.
type TweetArticle struct {
	Title       string `json:"title"`
	PreviewText string `json:"previewText,omitempty"`
}

// TweetWithMeta wraps TweetData with thread-awareness metadata.
type TweetWithMeta struct {
	TweetData
	IsThread       bool    `json:"isThread"`
	ThreadPosition string  `json:"threadPosition"` // "standalone", "root", "middle", "end"
	HasSelfReplies bool    `json:"hasSelfReplies"`
	ThreadRootID   *string `json:"threadRootId,omitempty"`
}

// TwitterUser is the normalized representation of a Twitter/X user.
// IsBlueVerified is top-level on result, NOT inside legacy (correction #8).
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

// TwitterList is the normalized representation of a Twitter/X list.
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

// ListOwner identifies the list owner.
type ListOwner struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// NewsItem is the normalized representation of a trending/news item.
type NewsItem struct {
	ID          string `json:"id"`
	Headline    string `json:"headline"`
	Category    string `json:"category,omitempty"`
	TimeAgo     string `json:"timeAgo,omitempty"`
	PostCount   *int   `json:"postCount,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	IsAiNews    bool   `json:"isAiNews,omitempty"`
	// Raw is only populated when includeRaw is true.
	Raw any `json:"_raw,omitempty"`
}

// CurrentUserResult holds basic identity for the authenticated user.
type CurrentUserResult struct {
	ID       string `json:"id"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name,omitempty"`
}

// TwitterCookies holds resolved authentication credentials.
type TwitterCookies struct {
	AuthToken    string `json:"authToken"`
	Ct0          string `json:"ct0"`
	CookieHeader string `json:"cookieHeader,omitempty"`
}
