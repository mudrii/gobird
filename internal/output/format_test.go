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
