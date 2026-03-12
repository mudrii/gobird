package types

// FetchOptions configures common pagination and output options.
type FetchOptions struct {
	// Cursor is the pagination cursor from the previous page.
	Cursor string
	// Count is the per-page item count (default depends on operation).
	Count int
	// Limit caps the total number of items across all pages.
	Limit int
	// MaxPages caps the number of fetched pages.
	MaxPages int
	// PageDelayMs is the sleep duration before each page after the first.
	PageDelayMs int
	// IncludeRaw attaches the raw API response to each result item.
	IncludeRaw bool
	// QuoteDepth controls quoted tweet expansion depth. 0 disables recursion.
	QuoteDepth int
}

// SearchOptions extends FetchOptions for search operations.
type SearchOptions struct {
	FetchOptions
	// Product controls the search timeline type ("Latest", "Top", etc.).
	Product string
}

// UserTweetsOptions extends FetchOptions for user tweet timelines.
type UserTweetsOptions struct {
	FetchOptions
	// IncludeReplies includes reply tweets in results.
	IncludeReplies bool
}

// TweetDetailOptions configures TweetDetail fetches.
type TweetDetailOptions struct {
	// IncludeRaw attaches the raw API response.
	IncludeRaw bool
	// QuoteDepth controls quoted tweet expansion depth. 0 disables recursion.
	QuoteDepth int
}

// ThreadOptions configures thread and reply fetches.
type ThreadOptions struct {
	FetchOptions
	// FilterMode selects the thread filtering algorithm.
	// Values: "author_chain" | "author_only" | "full_chain"
	FilterMode string
}

// NewsOptions configures news/trending fetches.
type NewsOptions struct {
	// Tabs lists which news tab IDs to fetch.
	// Defaults: ["forYou", "news", "sports", "entertainment"] (correction #46).
	Tabs []string
	// MaxCount is the per-tab item limit.
	MaxCount int
	// IncludeRaw attaches the raw API response.
	IncludeRaw bool
}

// BookmarkFolderOptions configures bookmark folder timeline fetches.
type BookmarkFolderOptions struct {
	FetchOptions
	// FolderID is the bookmark collection ID.
	FolderID string
}
