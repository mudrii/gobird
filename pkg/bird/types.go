package bird

import "github.com/mudrii/gobird/internal/types"

// Re-export public normalized types.

// PageResult is returned by a single paginated fetch.
type PageResult[T any] = types.PageResult[T]

// PaginatedResult accumulates items across pages.
type PaginatedResult[T any] = types.PaginatedResult[T]

// TweetData is the normalized representation of a single tweet.
type TweetData = types.TweetData

// TweetAuthor holds the author's display identifiers.
type TweetAuthor = types.TweetAuthor

// TweetMedia holds a single media attachment.
type TweetMedia = types.TweetMedia

// TweetArticle holds metadata extracted from an article tweet.
type TweetArticle = types.TweetArticle

// TweetWithMeta wraps TweetData with thread-awareness metadata.
type TweetWithMeta = types.TweetWithMeta

// TwitterUser is the normalized representation of a Twitter/X user.
type TwitterUser = types.TwitterUser

// TwitterList is the normalized representation of a Twitter/X list.
type TwitterList = types.TwitterList

// ListOwner identifies the list owner.
type ListOwner = types.ListOwner

// NewsItem is the normalized representation of a trending/news item.
type NewsItem = types.NewsItem

// CurrentUserResult holds basic identity for the authenticated user.
type CurrentUserResult = types.CurrentUserResult

// TweetPage is a page of TweetData items.
type TweetPage = types.TweetPage

// UserPage is a page of TwitterUser items.
type UserPage = types.UserPage

// ListPage is a page of TwitterList items.
type ListPage = types.ListPage

// NewsPage is a page of NewsItem items.
type NewsPage = types.NewsPage

// TweetResult is the full paginated tweet result.
type TweetResult = types.TweetResult

// UserResult is the full paginated user result.
type UserResult = types.UserResult

// ListResult is the full paginated list result.
type ListResult = types.ListResult

// FetchOptions configures common pagination and output options.
type FetchOptions = types.FetchOptions

// SearchOptions extends FetchOptions for search operations.
type SearchOptions = types.SearchOptions

// UserTweetsOptions extends FetchOptions for user tweet timelines.
type UserTweetsOptions = types.UserTweetsOptions

// TweetDetailOptions configures TweetDetail fetches.
type TweetDetailOptions = types.TweetDetailOptions

// ThreadOptions configures thread and reply fetches.
type ThreadOptions = types.ThreadOptions

// NewsOptions configures news/trending fetches.
type NewsOptions = types.NewsOptions

// BookmarkFolderOptions configures bookmark folder timeline fetches.
type BookmarkFolderOptions = types.BookmarkFolderOptions
