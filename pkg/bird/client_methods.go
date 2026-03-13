package bird

import "context"

// Tweet creates a new tweet and returns its tweet ID.
func (c *Client) Tweet(ctx context.Context, text string) (string, error) {
	return c.c.Tweet(ctx, text)
}

// Reply creates a reply tweet and returns its tweet ID.
func (c *Client) Reply(ctx context.Context, text, inReplyToID string) (string, error) {
	return c.c.Reply(ctx, text, inReplyToID)
}

// UploadMedia uploads media and returns its media ID.
func (c *Client) UploadMedia(ctx context.Context, data []byte, mimeType, altText string) (string, error) {
	return c.c.UploadMedia(ctx, data, mimeType, altText)
}

// GetTweet returns a single tweet by ID.
func (c *Client) GetTweet(ctx context.Context, tweetID string, opts *TweetDetailOptions) (*TweetData, error) {
	return c.c.GetTweet(ctx, tweetID, opts)
}

// GetReplies returns replies for a tweet.
func (c *Client) GetReplies(ctx context.Context, tweetID string, opts *ThreadOptions) (*TweetResult, error) {
	return c.c.GetReplies(ctx, tweetID, opts)
}

// GetThread returns the normalized thread for a tweet.
func (c *Client) GetThread(ctx context.Context, tweetID string, opts *ThreadOptions) ([]TweetWithMeta, error) {
	return c.c.GetThread(ctx, tweetID, opts)
}

// Search fetches a single page of search results.
func (c *Client) Search(ctx context.Context, q string, opts *SearchOptions) TweetPage {
	return c.c.Search(ctx, q, opts)
}

// GetAllSearchResults fetches all available search results.
func (c *Client) GetAllSearchResults(ctx context.Context, q string, opts *SearchOptions) TweetResult {
	return c.c.GetAllSearchResults(ctx, q, opts)
}

// GetHomeTimeline fetches the authenticated user's algorithmic home timeline.
func (c *Client) GetHomeTimeline(ctx context.Context, opts *FetchOptions) TweetResult {
	return c.c.GetHomeTimeline(ctx, opts)
}

// GetHomeLatestTimeline fetches the authenticated user's latest/following home timeline.
func (c *Client) GetHomeLatestTimeline(ctx context.Context, opts *FetchOptions) TweetResult {
	return c.c.GetHomeLatestTimeline(ctx, opts)
}

// GetBookmarks fetches the authenticated user's bookmarks.
func (c *Client) GetBookmarks(ctx context.Context, opts *FetchOptions) TweetResult {
	return c.c.GetBookmarks(ctx, opts)
}

// GetBookmarkFolderTimeline fetches tweets from a bookmark folder.
func (c *Client) GetBookmarkFolderTimeline(ctx context.Context, opts *BookmarkFolderOptions) TweetResult {
	return c.c.GetBookmarkFolderTimeline(ctx, opts)
}

// GetLikes fetches the authenticated user's liked tweets.
func (c *Client) GetLikes(ctx context.Context, opts *FetchOptions) TweetResult {
	return c.c.GetLikes(ctx, opts)
}

// Like likes a tweet.
func (c *Client) Like(ctx context.Context, tweetID string) error {
	return c.c.Like(ctx, tweetID)
}

// Unlike removes a like from a tweet.
func (c *Client) Unlike(ctx context.Context, tweetID string) error {
	return c.c.Unlike(ctx, tweetID)
}

// Retweet retweets a tweet and returns the retweet ID.
func (c *Client) Retweet(ctx context.Context, tweetID string) (string, error) {
	return c.c.Retweet(ctx, tweetID)
}

// Unretweet removes a retweet from a tweet.
func (c *Client) Unretweet(ctx context.Context, tweetID string) error {
	return c.c.Unretweet(ctx, tweetID)
}

// Bookmark bookmarks a tweet.
func (c *Client) Bookmark(ctx context.Context, tweetID string) error {
	return c.c.Bookmark(ctx, tweetID)
}

// Unbookmark removes a bookmark from a tweet.
func (c *Client) Unbookmark(ctx context.Context, tweetID string) error {
	return c.c.Unbookmark(ctx, tweetID)
}

// GetCurrentUser returns the authenticated user's identity.
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUserResult, error) {
	return c.c.GetCurrentUser(ctx)
}

// GetUserIDByUsername resolves a screen name to a numeric user ID.
func (c *Client) GetUserIDByUsername(ctx context.Context, username string) (string, error) {
	return c.c.GetUserIDByUsername(ctx, username)
}

// GetUserAboutAccount returns normalized profile data for a username.
func (c *Client) GetUserAboutAccount(ctx context.Context, username string) (*TwitterUser, error) {
	return c.c.GetUserAboutAccount(ctx, username)
}

// GetUserTweets fetches tweets for a user across multiple pages.
func (c *Client) GetUserTweets(ctx context.Context, userID string, opts *UserTweetsOptions) (*TweetResult, error) {
	return c.c.GetUserTweets(ctx, userID, opts)
}

// GetUserTweetsPaged fetches a single page of tweets for a user.
func (c *Client) GetUserTweetsPaged(ctx context.Context, userID string, cursor string) (*TweetPage, error) {
	return c.c.GetUserTweetsPaged(ctx, userID, cursor)
}

// GetFollowing returns accounts followed by a user.
func (c *Client) GetFollowing(ctx context.Context, userID string, opts *FetchOptions) (*UserResult, error) {
	return c.c.GetFollowing(ctx, userID, opts)
}

// GetFollowers returns accounts following a user.
func (c *Client) GetFollowers(ctx context.Context, userID string, opts *FetchOptions) (*UserResult, error) {
	return c.c.GetFollowers(ctx, userID, opts)
}

// Follow follows a user by numeric user ID.
func (c *Client) Follow(ctx context.Context, userID string) error {
	return c.c.Follow(ctx, userID)
}

// Unfollow unfollows a user by numeric user ID.
func (c *Client) Unfollow(ctx context.Context, userID string) error {
	return c.c.Unfollow(ctx, userID)
}

// GetOwnedLists returns the authenticated user's owned lists.
func (c *Client) GetOwnedLists(ctx context.Context, opts *FetchOptions) (*ListResult, error) {
	return c.c.GetOwnedLists(ctx, opts)
}

// GetListMemberships returns the authenticated user's list memberships.
func (c *Client) GetListMemberships(ctx context.Context, opts *FetchOptions) (*ListResult, error) {
	return c.c.GetListMemberships(ctx, opts)
}

// GetListTimeline returns the timeline for a list.
func (c *Client) GetListTimeline(ctx context.Context, listID string, opts *FetchOptions) (*TweetResult, error) {
	return c.c.GetListTimeline(ctx, listID, opts)
}

// GetNews fetches normalized news/trending items.
func (c *Client) GetNews(ctx context.Context, opts *NewsOptions) ([]NewsItem, error) {
	return c.c.GetNews(ctx, opts)
}

// ActiveQueryID returns the active query ID for the given operation.
func (c *Client) ActiveQueryID(operation string) string {
	return c.c.ActiveQueryID(operation)
}

// AllQueryIDs returns all query IDs to try for the given operation.
func (c *Client) AllQueryIDs(operation string) []string {
	return c.c.AllQueryIDs(operation)
}

// RefreshQueryIDs refreshes runtime query IDs from the X.com bundle.
func (c *Client) RefreshQueryIDs(ctx context.Context) {
	c.c.RefreshQueryIDs(ctx)
}
