package types

// PageResult is returned by a single paginated fetch.
type PageResult[T any] struct {
	Items      []T
	NextCursor string
	Success    bool
	Error      error
}

// TweetPage is a page of TweetData items.
type TweetPage = PageResult[TweetData]

// UserPage is a page of TwitterUser items.
type UserPage = PageResult[TwitterUser]

// ListPage is a page of TwitterList items.
type ListPage = PageResult[TwitterList]

// NewsPage is a page of NewsItem items.
type NewsPage = PageResult[NewsItem]

// PaginatedResult accumulates items across pages.
type PaginatedResult[T any] struct {
	Items      []T
	NextCursor string
	Success    bool
	Error      error
}

// TweetResult is the full paginated tweet result.
type TweetResult = PaginatedResult[TweetData]

// UserResult is the full paginated user result.
type UserResult = PaginatedResult[TwitterUser]

// ListResult is the full paginated list result.
type ListResult = PaginatedResult[TwitterList]
