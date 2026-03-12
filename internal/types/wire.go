package types

// WireResponse is the top-level GraphQL response envelope.
type WireResponse struct {
	Data   map[string]any   `json:"data"`
	Errors []WireError      `json:"errors"`
}

// WireError represents a single GraphQL error entry.
type WireError struct {
	Message    string            `json:"message"`
	Locations  []WireLocation    `json:"locations"`
	Path       []any             `json:"path"`
	Extensions WireErrorExt      `json:"extensions"`
}

// WireLocation is a source location within a GraphQL document.
type WireLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// WireErrorExt holds structured extension fields on a GraphQL error.
type WireErrorExt struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Source  string `json:"source"`
	RetryAfter *int `json:"retry_after"`
}

// WireTimelineInstruction is a single instruction within a timeline.
type WireTimelineInstruction struct {
	Type    string          `json:"type"`
	Entries []WireEntry     `json:"entries"`
	Entry   *WireEntry      `json:"entry"`
}

// WireEntry is a single timeline entry.
type WireEntry struct {
	EntryID   string        `json:"entryId"`
	SortIndex string        `json:"sortIndex"`
	Content   WireContent   `json:"content"`
}

// WireContent holds the typed content of a timeline entry.
type WireContent struct {
	EntryType  string            `json:"entryType"`
	TypeName   string            `json:"__typename"`
	CursorType string            `json:"cursorType"`
	Value      string            `json:"value"`
	ItemContent *WireItemContent `json:"itemContent"`
	Items      []WireItem        `json:"items"`
}

// WireItem wraps an item within a module timeline entry.
type WireItem struct {
	EntryID string          `json:"entryId"`
	Item    WireItemContent `json:"item"`
}

// WireItemContent holds the typed item payload.
type WireItemContent struct {
	TypeName    string          `json:"__typename"`
	TweetResult *WireTweetResult `json:"tweet_results"`
	UserResult  *WireUserResult  `json:"user_results"`
}

// WireTweetResult is the outer wrapper around a tweet result.
type WireTweetResult struct {
	Result *WireRawTweet `json:"result"`
}

// WireRawTweet represents either a tweet or a visibility wrapper.
type WireRawTweet struct {
	TypeName      string           `json:"__typename"`
	RestID        string           `json:"rest_id"`
	Core          *WireTweetCore   `json:"core"`
	Legacy        *WireTweetLegacy `json:"legacy"`
	Card          *WireCard        `json:"card"`
	QuotedResult  *WireTweetResult `json:"quoted_status_result"`
	Tweet         *WireRawTweet    `json:"tweet"`  // unwrap TweetWithVisibilityResults (correction #unwrap)
	IsBlueVerified bool            `json:"is_blue_verified"`
	NoteTweet     *WireNoteTweet   `json:"note_tweet"`
	Article       *WireArticle     `json:"article"`
}

// WireTweetCore holds the author user result.
type WireTweetCore struct {
	UserResults WireUserResult `json:"user_results"`
}

// WireTweetLegacy holds the legacy tweet fields.
type WireTweetLegacy struct {
	FullText            string             `json:"full_text"`
	CreatedAt           string             `json:"created_at"`
	ConversationIDStr   string             `json:"conversation_id_str"`
	InReplyToStatusIDStr *string           `json:"in_reply_to_status_id_str"`
	ReplyCount          int                `json:"reply_count"`
	RetweetCount        int                `json:"retweet_count"`
	FavoriteCount       int                `json:"favorite_count"`
	ExtendedEntities    *WireMediaEntities `json:"extended_entities"`
	UserIDStr           string             `json:"user_id_str"`
}

// WireMediaEntities holds the extended media list.
type WireMediaEntities struct {
	Media []WireMedia `json:"media"`
}

// WireMedia is a single media item in the tweet payload.
type WireMedia struct {
	Type             string              `json:"type"`
	MediaURLHttps    string              `json:"media_url_https"`
	Sizes            WireMediaSizes      `json:"sizes"`
	VideoInfo        *WireVideoInfo      `json:"video_info"`
}

// WireMediaSizes holds the available size variants.
type WireMediaSizes struct {
	Large  *WireMediaSize `json:"large"`
	Medium *WireMediaSize `json:"medium"`
	Small  *WireMediaSize `json:"small"`
	Thumb  *WireMediaSize `json:"thumb"`
}

// WireMediaSize is a single size variant.
type WireMediaSize struct {
	W      int    `json:"w"`
	H      int    `json:"h"`
	Resize string `json:"resize"`
}

// WireVideoInfo holds video-specific metadata.
type WireVideoInfo struct {
	DurationMillis *int              `json:"duration_millis"`
	Variants       []WireVideoVariant `json:"variants"`
}

// WireVideoVariant is a single video bitrate variant.
type WireVideoVariant struct {
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
	Bitrate     *int   `json:"bitrate"`
}

// WireUserResult wraps a user result with possible visibility wrapper.
type WireUserResult struct {
	Result *WireRawUser `json:"result"`
}

// WireRawUser represents either a User or UserUnavailable.
type WireRawUser struct {
	TypeName       string          `json:"__typename"`
	RestID         string          `json:"rest_id"`
	Legacy         *WireUserLegacy `json:"legacy"`
	IsBlueVerified bool            `json:"is_blue_verified"`
}

// WireUserLegacy holds legacy user fields.
type WireUserLegacy struct {
	ScreenName      string `json:"screen_name"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	FollowersCount  int    `json:"followers_count"`
	FriendsCount    int    `json:"friends_count"`
	ProfileImageURLHTTPS string `json:"profile_image_url_https"`
	CreatedAt       string `json:"created_at"`
}

// WireCard holds Twitter card data (used for articles).
type WireCard struct {
	RestID string      `json:"rest_id"`
	Legacy *WireCardLegacy `json:"legacy"`
}

// WireCardLegacy holds the card binding values.
type WireCardLegacy struct {
	BindingValues []WireBindingValue `json:"binding_values"`
}

// WireBindingValue is a single card binding key-value pair.
type WireBindingValue struct {
	Key   string          `json:"key"`
	Value WireBindingData `json:"value"`
}

// WireBindingData holds the typed value of a card binding.
type WireBindingData struct {
	StringValue  *string         `json:"string_value"`
	ImageValue   *WireImageValue `json:"image_value"`
	BooleanValue *bool           `json:"boolean_value"`
}

// WireImageValue holds a card image with dimensions.
type WireImageValue struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// WireNoteTweet holds note tweet (long-form) content.
type WireNoteTweet struct {
	NoteTweetResults WireNoteTweetResults `json:"note_tweet_results"`
}

// WireNoteTweetResults wraps the note tweet result.
type WireNoteTweetResults struct {
	Result *WireNoteTweetResult `json:"result"`
}

// WireNoteTweetResult holds the note tweet text.
type WireNoteTweetResult struct {
	Text string `json:"text"`
}

// WireArticle holds article-specific metadata.
type WireArticle struct {
	ArticleResults WireArticleResults `json:"article_results"`
}

// WireArticleResults wraps the article result.
type WireArticleResults struct {
	Result *WireArticleResult `json:"result"`
}

// WireArticleResult holds the article content.
type WireArticleResult struct {
	Title        string `json:"title"`
	PreviewText  string `json:"preview_text"`
	ContentState string `json:"content_state"`
}

// WireList is the wire-level Twitter list object.
type WireList struct {
	IDStr           string        `json:"id_str"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	MemberCount     int           `json:"member_count"`
	SubscriberCount int           `json:"subscriber_count"`
	Mode            string        `json:"mode"`
	CreatedAt       string        `json:"created_at"`
	UserResults     WireUserResult `json:"user_results"`
}
