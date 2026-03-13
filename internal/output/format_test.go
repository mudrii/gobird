package output_test

import (
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
)

func makeTweet(id, text, username, name string, likes, rts int) types.TweetData {
	return types.TweetData{
		ID:           id,
		Text:         text,
		LikeCount:    likes,
		RetweetCount: rts,
		Author: types.TweetAuthor{
			Username: username,
			Name:     name,
		},
	}
}

func TestFormatTweet_Basic(t *testing.T) {
	tw := makeTweet("1", "Hello world", "alice", "Alice", 42, 7)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@alice") {
		t.Errorf("missing handle: %q", got)
	}
	if !strings.Contains(got, "Hello world") {
		t.Errorf("missing text: %q", got)
	}
	if !strings.Contains(got, "likes: 42") {
		t.Errorf("missing like count: %q", got)
	}
	if !strings.Contains(got, "rts: 7") {
		t.Errorf("missing retweet count: %q", got)
	}
	if !strings.Contains(got, "replies:") {
		t.Errorf("missing reply count: %q", got)
	}
}

func TestFormatTweet_NoColor(t *testing.T) {
	tw := makeTweet("2", "No ANSI", "bob", "Bob", 0, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true})
	if strings.Contains(got, "\x1b[") {
		t.Errorf("output contains ANSI escape sequences: %q", got)
	}
}

func TestFormatTweet_NoEmoji(t *testing.T) {
	tw := makeTweet("3", "Plain text", "carol", "Carol", 1, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoEmoji: true, NoColor: true})
	if strings.Contains(got, "🐦") {
		t.Errorf("output contains emoji when NoEmoji is set: %q", got)
	}
}

func TestFormatTweet_Plain(t *testing.T) {
	tw := makeTweet("4", "Plain mode", "dave", "Dave", 5, 2)
	got := output.FormatTweet(tw, output.FormatOptions{Plain: true})
	if strings.Contains(got, "\x1b[") {
		t.Errorf("plain output contains ANSI escape sequences: %q", got)
	}
	if strings.Contains(got, "🐦") {
		t.Errorf("plain output contains emoji: %q", got)
	}
}

func TestFormatTweet_WithMedia(t *testing.T) {
	tw := makeTweet("5", "Has media", "eve", "Eve", 10, 3)
	tw.Media = []types.TweetMedia{
		{Type: "photo", URL: "https://example.com/photo.jpg"},
	}
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@eve") {
		t.Errorf("missing handle in media tweet: %q", got)
	}
}

func TestFormatTweet_WithQuotedTweet(t *testing.T) {
	inner := makeTweet("10", "Quoted text", "quoted", "Quoted", 0, 0)
	outer := makeTweet("11", "Outer text", "outer", "Outer", 1, 0)
	outer.QuotedTweet = &inner
	got := output.FormatTweet(outer, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@outer") {
		t.Errorf("missing outer handle: %q", got)
	}
	if !strings.Contains(got, "Outer text") {
		t.Errorf("missing outer text: %q", got)
	}
}

func TestFormatUser_Basic(t *testing.T) {
	u := types.TwitterUser{
		ID:             "100",
		Username:       "frank",
		Name:           "Frank",
		Description:    "Bio here",
		FollowersCount: 500,
		FollowingCount: 200,
	}
	got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@frank") {
		t.Errorf("missing handle: %q", got)
	}
	if !strings.Contains(got, "Frank") {
		t.Errorf("missing name: %q", got)
	}
	if !strings.Contains(got, "500") {
		t.Errorf("missing followers count: %q", got)
	}
	if !strings.Contains(got, "200") {
		t.Errorf("missing following count: %q", got)
	}
}

func TestFormatUser_NoEmoji(t *testing.T) {
	u := types.TwitterUser{Username: "grace", Name: "Grace"}
	got := output.FormatUser(u, output.FormatOptions{NoEmoji: true})
	if strings.Contains(got, "👤") {
		t.Errorf("output contains emoji when NoEmoji is set: %q", got)
	}
}

func TestFormatList_Basic(t *testing.T) {
	l := types.TwitterList{
		ID:          "list-1",
		Name:        "My List",
		MemberCount: 42,
	}
	got := output.FormatList(l, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "My List") {
		t.Errorf("missing list name: %q", got)
	}
	if !strings.Contains(got, "42") {
		t.Errorf("missing member count: %q", got)
	}
}

func TestFormatList_WithOwner(t *testing.T) {
	l := types.TwitterList{
		ID:          "list-2",
		Name:        "Owned List",
		MemberCount: 10,
		Owner:       &types.ListOwner{Username: "henry", Name: "Henry"},
	}
	got := output.FormatList(l, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@henry") {
		t.Errorf("missing owner handle: %q", got)
	}
}

func TestFormatNewsItem_Basic(t *testing.T) {
	n := types.NewsItem{
		ID:       "n1",
		Headline: "Big story today",
		Category: "World",
	}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "Big story today") {
		t.Errorf("missing headline: %q", got)
	}
	if !strings.Contains(got, "World") {
		t.Errorf("missing category: %q", got)
	}
}

func TestFormatNewsItem_NoCategory(t *testing.T) {
	n := types.NewsItem{ID: "n2", Headline: "No category"}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "No category") {
		t.Errorf("missing headline: %q", got)
	}
}

func TestFormatNewsItem_NoEmoji(t *testing.T) {
	n := types.NewsItem{Headline: "Test"}
	got := output.FormatNewsItem(n, output.FormatOptions{NoEmoji: true})
	if strings.Contains(got, "📰") {
		t.Errorf("output contains emoji when NoEmoji is set: %q", got)
	}
}

func TestFormatTweet_WithArticle(t *testing.T) {
	tw := makeTweet("20", "Read this article", "writer", "Writer", 0, 0)
	tw.Article = &types.TweetArticle{Title: "Deep Dive Into Go"}
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "Deep Dive Into Go") {
		t.Errorf("article title not shown in output: %q", got)
	}
}

func TestFormatTweet_LikeCount_Zero(t *testing.T) {
	tw := makeTweet("21", "Unpopular opinion", "anon", "Anon", 0, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "likes: 0") {
		t.Errorf("zero like count should still be shown: %q", got)
	}
}

func TestFormatTweet_ReplyCount(t *testing.T) {
	tw := makeTweet("22", "Discussion starter", "host", "Host", 5, 1)
	tw.ReplyCount = 17
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "replies: 17") {
		t.Errorf("reply count not shown: %q", got)
	}
}

func TestFormatUser_BlueVerified(t *testing.T) {
	u := types.TwitterUser{
		Username:       "verified_user",
		Name:           "Verified",
		IsBlueVerified: true,
	}
	got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "[verified]") {
		t.Errorf("verified badge not shown: %q", got)
	}
}

func TestFormatUser_FollowerCount(t *testing.T) {
	u := types.TwitterUser{
		Username:       "popular",
		Name:           "Popular",
		FollowersCount: 1_500_000,
		FollowingCount: 500,
	}
	got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "1.5M") {
		t.Errorf("large follower count not formatted readably (want 1.5M): %q", got)
	}
}

func TestFormatNewsItem_WithURL(t *testing.T) {
	n := types.NewsItem{
		ID:       "n3",
		Headline: "Breaking news",
		URL:      "https://example.com/story",
	}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "https://example.com/story") {
		t.Errorf("URL not shown in output: %q", got)
	}
}

func TestFormatNewsItem_AiNews(t *testing.T) {
	n := types.NewsItem{
		ID:       "n4",
		Headline: "AI writes code",
		IsAiNews: true,
	}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "[AI]") {
		t.Errorf("AI badge not shown: %q", got)
	}
}

// ---------------------------------------------------------------------------
// Additional formatting edge cases
// ---------------------------------------------------------------------------

func TestFormatTweet_ZeroValueFields(t *testing.T) {
	tw := types.TweetData{}
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@") {
		t.Errorf("should contain @ prefix even for empty username: %q", got)
	}
	if !strings.Contains(got, "replies: 0") {
		t.Errorf("should show zero reply count: %q", got)
	}
	if !strings.Contains(got, "likes: 0") {
		t.Errorf("should show zero like count: %q", got)
	}
	if !strings.Contains(got, "rts: 0") {
		t.Errorf("should show zero retweet count: %q", got)
	}
}

func TestFormatTweet_EmptyText(t *testing.T) {
	tw := makeTweet("1", "", "user", "User", 0, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@user") {
		t.Errorf("handle should appear even with empty text: %q", got)
	}
}

func TestFormatTweet_TextWithWhitespace(t *testing.T) {
	tw := makeTweet("1", "  spaced out  ", "user", "User", 0, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if strings.Contains(got, "  spaced out  ") {
		t.Errorf("text should be trimmed: %q", got)
	}
	if !strings.Contains(got, "spaced out") {
		t.Errorf("trimmed text should be present: %q", got)
	}
}

func TestFormatTweet_ArticleEmptyTitle(t *testing.T) {
	tw := makeTweet("1", "text", "user", "User", 0, 0)
	tw.Article = &types.TweetArticle{Title: ""}
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if strings.Contains(got, "[article:") {
		t.Errorf("empty article title should not produce [article:] tag: %q", got)
	}
}

func TestFormatTweet_EmptyMedia(t *testing.T) {
	tw := makeTweet("1", "text", "user", "User", 0, 0)
	tw.Media = []types.TweetMedia{}
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@user") {
		t.Errorf("should render normally with empty media slice: %q", got)
	}
}

func TestFormatTweet_WithColor(t *testing.T) {
	tw := makeTweet("1", "text", "user", "User", 0, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: false, NoEmoji: true})
	if !strings.Contains(got, "\x1b[1m") {
		t.Errorf("should contain bold ANSI escape when NoColor is false: %q", got)
	}
}

func TestFormatTweet_WithEmoji(t *testing.T) {
	tw := makeTweet("1", "text", "user", "User", 0, 0)
	got := output.FormatTweet(tw, output.FormatOptions{NoColor: true, NoEmoji: false})
	if !strings.HasPrefix(got, "\xf0\x9f\x90\xa6") { // bird emoji bytes
		t.Errorf("should start with bird emoji when NoEmoji is false: %q", got)
	}
}

func TestFormatUser_ZeroValueFields(t *testing.T) {
	u := types.TwitterUser{}
	got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "@") {
		t.Errorf("should contain @ even with empty username: %q", got)
	}
	if !strings.Contains(got, "followers:") {
		t.Errorf("should contain followers label: %q", got)
	}
}

func TestFormatUser_NotVerified(t *testing.T) {
	u := types.TwitterUser{
		Username:       "regular",
		Name:           "Regular",
		IsBlueVerified: false,
	}
	got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: true})
	if strings.Contains(got, "verified") || strings.Contains(got, "\xe2\x9c\x93") {
		t.Errorf("unverified user should not have verified badge: %q", got)
	}
}

func TestFormatUser_FollowerCountFormatting(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		contains string
	}{
		{"small", 42, "42"},
		{"thousands", 1500, "1.5K"},
		{"millions", 2500000, "2.5M"},
		{"exactly_1000", 1000, "1.0K"},
		{"exactly_1M", 1000000, "1.0M"},
		{"zero", 0, "0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := types.TwitterUser{
				Username:       "u",
				Name:           "U",
				FollowersCount: tc.count,
			}
			got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: true})
			if !strings.Contains(got, tc.contains) {
				t.Errorf("want %q in output, got %q", tc.contains, got)
			}
		})
	}
}

func TestFormatList_ZeroMemberCount(t *testing.T) {
	l := types.TwitterList{
		ID:          "l1",
		Name:        "Empty List",
		MemberCount: 0,
	}
	got := output.FormatList(l, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "members: 0") {
		t.Errorf("zero member count should be shown: %q", got)
	}
}

func TestFormatList_NoOwner(t *testing.T) {
	l := types.TwitterList{
		ID:   "l2",
		Name: "Ownerless",
	}
	got := output.FormatList(l, output.FormatOptions{NoColor: true, NoEmoji: true})
	if strings.Contains(got, "owner:") {
		t.Errorf("should not show owner when nil: %q", got)
	}
}

func TestFormatNewsItem_AllFieldsPopulated(t *testing.T) {
	n := types.NewsItem{
		ID:       "n5",
		Headline: "Full news",
		Category: "Tech",
		URL:      "https://example.com",
		IsAiNews: true,
	}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(got, "Full news") {
		t.Errorf("missing headline: %q", got)
	}
	if !strings.Contains(got, "(Tech)") {
		t.Errorf("missing category: %q", got)
	}
	if !strings.Contains(got, "https://example.com") {
		t.Errorf("missing URL: %q", got)
	}
	if !strings.Contains(got, "[AI]") {
		t.Errorf("missing AI badge: %q", got)
	}
}

func TestFormatNewsItem_EmptyFields(t *testing.T) {
	n := types.NewsItem{Headline: "Just headline"}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: true})
	if strings.Contains(got, "(") {
		t.Errorf("no category should mean no parens: %q", got)
	}
	if strings.Contains(got, "[") && !strings.Contains(got, "[AI]") {
		t.Errorf("no URL should mean no brackets: %q", got)
	}
}

func TestFormatNewsItem_AiNewsWithEmoji(t *testing.T) {
	n := types.NewsItem{
		Headline: "AI stuff",
		IsAiNews: true,
	}
	got := output.FormatNewsItem(n, output.FormatOptions{NoColor: true, NoEmoji: false})
	if !strings.Contains(got, "\xf0\x9f\xa4\x96") { // robot emoji bytes
		t.Errorf("AI news with emoji enabled should show robot emoji: %q", got)
	}
}

func TestFormatUser_VerifiedWithEmoji(t *testing.T) {
	u := types.TwitterUser{
		Username:       "vuser",
		Name:           "V",
		IsBlueVerified: true,
	}
	got := output.FormatUser(u, output.FormatOptions{NoColor: true, NoEmoji: false})
	if !strings.Contains(got, "\xe2\x9c\x93") { // checkmark bytes
		t.Errorf("verified user with emoji should show checkmark: %q", got)
	}
}
