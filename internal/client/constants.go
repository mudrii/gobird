// Package client implements the Twitter/X API client.
package client

// BearerToken is the public bearer token used for all GraphQL and REST requests.
const BearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

// UserAgent is the browser UA string sent on every request.
const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// API base URLs.
const (
	GraphQLBaseURL     = "https://x.com/i/api/graphql"
	RESTV1BaseURL      = "https://x.com/i/api/1.1"
	MediaUploadURL     = "https://upload.twitter.com/i/media/upload.json"
	MediaMetadataURL   = "https://x.com/i/api/1.1/media/metadata/create.json"
	StatusUpdateURL    = "https://x.com/i/api/1.1/statuses/update.json"
	FollowRESTURL      = "https://x.com/i/api/1.1/friendships/create.json"
	UnfollowRESTURL    = "https://x.com/i/api/1.1/friendships/destroy.json"
	FollowersRESTURL   = "https://x.com/i/api/1.1/followers/list.json"
	FollowingRESTURL   = "https://x.com/i/api/1.1/friends/list.json"
	UserLookupRESTURL  = "https://x.com/i/api/1.1/users/show.json"
	SettingsURL        = "https://x.com/i/api/account/settings.json"
	CredentialsURL     = "https://x.com/i/api/account/verify_credentials.json"
	SettingsPageURL    = "https://x.com/settings/account"
)

// FallbackQueryIDs contains the 29 hardcoded fallback query IDs used when
// runtime scraping has not yet populated the cache.
// Correction #71: exactly 29 entries, not 24.
var FallbackQueryIDs = map[string]string{
	"CreateTweet":              "TAJw1rBsjAtdNgTdlo2oeg",
	"CreateRetweet":            "ojPdsZsimiJrUGLR1sjUtA",
	"DeleteRetweet":            "iQtK4dl5hBmXewYZuEOKVw",
	"CreateFriendship":         "8h9JVdV8dlSyqyRDJEPCsA",
	"DestroyFriendship":        "ppXWuagMNXgvzx6WoXBW0Q",
	"FavoriteTweet":            "lI07N6Otwv1PhnEgXILM7A",
	"UnfavoriteTweet":          "ZYKSe-w7KEslx3JhSIk5LA",
	"CreateBookmark":           "aoDbu3RHznuiSkQ9aNM67Q",
	"DeleteBookmark":           "Wlmlj2-xzyS1GN3a6cj-mQ",
	"TweetDetail":              "97JF30KziU00483E_8elBA",
	"SearchTimeline":           "M1jEez78PEfVfbQLvlWMvQ",
	"UserArticlesTweets":       "8zBy9h4L90aDL02RsBcCFg",
	"UserTweets":               "Wms1GvIiHXAPBaCr9KblaA",
	"Bookmarks":                "RV1g3b8n_SGOHwkqKYSCFw",
	"Following":                "BEkNpEt5pNETESoqMsTEGA",
	"Followers":                "kuFUYP9eV1FPoEy4N-pi7w",
	"Likes":                    "JR2gceKucIKcVNB_9JkhsA",
	"BookmarkFolderTimeline":   "KJIQpsvxrTfRIlbaRIySHQ",
	"ListOwnerships":           "wQcOSjSQ8NtgxIwvYl1lMg",
	"ListMemberships":          "BlEXXdARdSeL_0KyKHHvvg",
	"ListLatestTweetsTimeline": "2TemLyqrMpTeAmysdbnVqw",
	"ListByRestId":             "wXzyA5vM_aVkBL9G8Vp3kw",
	"HomeTimeline":             "edseUwk9sP5Phz__9TIRnA",
	"HomeLatestTimeline":       "iOEZpOdfekFsxSlPQCQtPg",
	"ExploreSidebar":           "lpSN4M6qpimkF4nRFPE3nQ",
	"ExplorePage":              "kheAINB_4pzRDqkzG3K-ng",
	"GenericTimelineById":      "uGSr7alSjR9v6QJAIaqSKQ",
	"TrendHistory":             "Sj4T-jSB9pr0Mxtsc1UKZQ",
	"AboutAccountQuery":        "zs_jFPFT78rBpXv9Z3U2YQ",
}

// BundledBaselineQueryIDs are the query IDs embedded in query-ids.json.
// They override FallbackQueryIDs for the operations listed here.
// Operations NOT in this map use FallbackQueryIDs directly.
var BundledBaselineQueryIDs = map[string]string{
	"CreateTweet":            "nmdAQXJDxw6-0KKF2on7eA",
	"CreateRetweet":          "LFho5rIi4xcKO90p9jwG7A",
	"CreateFriendship":       "8h9JVdV8dlSyqyRDJEPCsA",
	"DestroyFriendship":      "ppXWuagMNXgvzx6WoXBW0Q",
	"FavoriteTweet":          "lI07N6Otwv1PhnEgXILM7A",
	"DeleteBookmark":         "Wlmlj2-xzyS1GN3a6cj-mQ",
	"TweetDetail":            "_NvJCnIjOW__EP5-RF197A",
	"SearchTimeline":         "6AAys3t42mosm_yTI_QENg",
	"Bookmarks":              "RV1g3b8n_SGOHwkqKYSCFw",
	"BookmarkFolderTimeline": "KJIQpsvxrTfRIlbaRIySHQ",
	"Following":              "mWYeougg_ocJS2Vr1Vt28w",
	"Followers":              "SFYY3WsgwjlXSLlfnEUE4A",
	"Likes":                  "ETJflBunfqNa1uE1mBPCaw",
	"ExploreSidebar":         "lpSN4M6qpimkF4nRFPE3nQ",
	"ExplorePage":            "kheAINB_4pzRDqkzG3K-ng",
	"GenericTimelineById":    "uGSr7alSjR9v6QJAIaqSKQ",
	"TrendHistory":           "Sj4T-jSB9pr0Mxtsc1UKZQ",
	"AboutAccountQuery":      "zs_jFPFT78rBpXv9Z3U2YQ",
}

// PerOperationFallbackIDs lists all query IDs to try per operation, in order.
// The first element is derived from BundledBaselineQueryIDs (if present) or
// FallbackQueryIDs; subsequent elements are additional hardcoded fallbacks.
// UserByScreenName has hardcoded-only IDs and never uses the runtime cache.
var PerOperationFallbackIDs = map[string][]string{
	"TweetDetail":              {"_NvJCnIjOW__EP5-RF197A", "97JF30KziU00483E_8elBA", "aFvUsJm2c-oDkJV75blV6g"},
	"SearchTimeline":           {"6AAys3t42mosm_yTI_QENg", "M1jEez78PEfVfbQLvlWMvQ", "5h0kNbk3ii97rmfY6CdgAA", "Tp1sewRU1AsZpBWhqCZicQ"},
	"HomeTimeline":             {"edseUwk9sP5Phz__9TIRnA"},
	"HomeLatestTimeline":       {"iOEZpOdfekFsxSlPQCQtPg"},
	"Bookmarks":                {"RV1g3b8n_SGOHwkqKYSCFw", "tmd4ifV8RHltzn8ymGg1aw"},
	"BookmarkFolderTimeline":   {"KJIQpsvxrTfRIlbaRIySHQ"},
	"Likes":                    {"ETJflBunfqNa1uE1mBPCaw", "JR2gceKucIKcVNB_9JkhsA"},
	"UserTweets":               {"Wms1GvIiHXAPBaCr9KblaA"},
	"Following":                {"mWYeougg_ocJS2Vr1Vt28w", "BEkNpEt5pNETESoqMsTEGA"},
	"Followers":                {"SFYY3WsgwjlXSLlfnEUE4A", "kuFUYP9eV1FPoEy4N-pi7w"},
	"CreateFriendship":         {"8h9JVdV8dlSyqyRDJEPCsA", "OPwKc1HXnBT_bWXfAlo-9g"},
	"DestroyFriendship":        {"ppXWuagMNXgvzx6WoXBW0Q", "8h9JVdV8dlSyqyRDJEPCsA"},
	"ListOwnerships":           {"wQcOSjSQ8NtgxIwvYl1lMg"},
	"ListMemberships":          {"BlEXXdARdSeL_0KyKHHvvg"},
	"ListLatestTweetsTimeline": {"2TemLyqrMpTeAmysdbnVqw"},
	"AboutAccountQuery":        {"zs_jFPFT78rBpXv9Z3U2YQ"},
	// Hardcoded only — never uses runtime cache (correction #5).
	"UserByScreenName": {"xc8f1g7BYqr6VTzTbvNlGw", "qW5u-DAuXpMEG0zA1F7UGQ", "sLVLhk0bGj3MVFEKTdax1w"},
}

// GenericTimelineTabIDs maps news tab names to their timeline IDs.
// Default tabs: forYou, news, sports, entertainment (NOT trending). Correction #46.
var GenericTimelineTabIDs = map[string]string{
	"forYou":        "VGltZWxpbmU6DAC2CwABAAAAB2Zvcl95b3UAAA==",
	"trending":      "VGltZWxpbmU6DAC2CwABAAAACHRyZW5kaW5nAAA=",
	"news":          "VGltZWxpbmU6DAC2CwABAAAABG5ld3MAAA==",
	"sports":        "VGltZWxpbmU6DAC2CwABAAAABnNwb3J0cwAA",
	"entertainment": "VGltZWxpbmU6DAC2CwABAAAADWVudGVydGFpbm1lbnQAAA==",
}

// DefaultNewsTabs are the tabs fetched when none are specified.
// Correction #46: does NOT include "trending".
var DefaultNewsTabs = []string{"forYou", "news", "sports", "entertainment"}
