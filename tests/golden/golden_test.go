//go:build acceptance

package golden_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mudrii/gobird/internal/cli"
	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

func goldenDir() string {
	return filepath.Join("testdata")
}

func goldenPath(name string) string {
	return filepath.Join(goldenDir(), name)
}

func TestGolden_VersionOutput(t *testing.T) {
	cli.SetBuildInfo("1.0.0-golden", "abc1234")
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	testutil.AssertGolden(t, goldenPath("version.txt"), buf.Bytes())
}

func TestGolden_HelpOutput(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	testutil.AssertGolden(t, goldenPath("help.txt"), buf.Bytes())
}

func TestGolden_FormatTweet_Plain(t *testing.T) {
	tw := types.TweetData{
		ID:   "1001",
		Text: "Hello from the golden test!",
		Author: types.TweetAuthor{
			Username: "goldenuser",
			Name:     "Golden User",
		},
		ReplyCount:   5,
		RetweetCount: 10,
		LikeCount:    100,
		CreatedAt:    "Mon Jan 01 12:00:00 +0000 2024",
	}
	got := output.FormatTweet(tw, output.FormatOptions{
		Plain:   true,
		NoColor: true,
		NoEmoji: true,
	})
	testutil.AssertGolden(t, goldenPath("format_tweet_plain.txt"), []byte(got))
}

func TestGolden_FormatTweet_NoColor(t *testing.T) {
	tw := types.TweetData{
		ID:   "1001",
		Text: "Hello from the golden test!",
		Author: types.TweetAuthor{
			Username: "goldenuser",
			Name:     "Golden User",
		},
		ReplyCount:   5,
		RetweetCount: 10,
		LikeCount:    100,
	}
	got := output.FormatTweet(tw, output.FormatOptions{
		NoColor: true,
		NoEmoji: true,
	})
	testutil.AssertGolden(t, goldenPath("format_tweet_nocolor.txt"), []byte(got))
}

func TestGolden_FormatUser_Plain(t *testing.T) {
	u := types.TwitterUser{
		ID:             "u100",
		Username:       "goldenuser",
		Name:           "Golden User",
		Description:    "A test user for golden files",
		FollowersCount: 1500,
		FollowingCount: 200,
		IsBlueVerified: true,
	}
	got := output.FormatUser(u, output.FormatOptions{
		Plain:   true,
		NoColor: true,
		NoEmoji: true,
	})
	testutil.AssertGolden(t, goldenPath("format_user_plain.txt"), []byte(got))
}

func TestGolden_FormatList_Plain(t *testing.T) {
	l := types.TwitterList{
		ID:          "list-42",
		Name:        "My Golden List",
		MemberCount: 42,
		Owner: &types.ListOwner{
			ID:       "u100",
			Username: "listowner",
			Name:     "List Owner",
		},
	}
	got := output.FormatList(l, output.FormatOptions{
		Plain:   true,
		NoColor: true,
		NoEmoji: true,
	})
	testutil.AssertGolden(t, goldenPath("format_list_plain.txt"), []byte(got))
}

func TestGolden_FormatNewsItem_Plain(t *testing.T) {
	n := types.NewsItem{
		ID:       "n1",
		Headline: "Breaking: Golden Tests Pass",
		Category: "Technology",
		URL:      "https://example.com/golden-news",
		IsAiNews: true,
	}
	got := output.FormatNewsItem(n, output.FormatOptions{
		Plain:   true,
		NoColor: true,
		NoEmoji: true,
	})
	testutil.AssertGolden(t, goldenPath("format_news_plain.txt"), []byte(got))
}

func TestGolden_JSONOutput_Tweet(t *testing.T) {
	tw := types.TweetData{
		ID:   "1001",
		Text: "JSON golden test",
		Author: types.TweetAuthor{
			Username: "goldenuser",
			Name:     "Golden User",
		},
		ReplyCount:   5,
		RetweetCount: 10,
		LikeCount:    100,
		CreatedAt:    "Mon Jan 01 12:00:00 +0000 2024",
	}
	data, err := json.MarshalIndent(tw, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	testutil.AssertGolden(t, goldenPath("json_tweet.json"), data)
}

func TestGolden_JSONOutput_TweetList(t *testing.T) {
	tweets := []types.TweetData{
		{
			ID:   "1001",
			Text: "First tweet",
			Author: types.TweetAuthor{
				Username: "alice",
				Name:     "Alice",
			},
			LikeCount: 10,
		},
		{
			ID:   "1002",
			Text: "Second tweet",
			Author: types.TweetAuthor{
				Username: "bob",
				Name:     "Bob",
			},
			LikeCount: 20,
		},
	}
	data, err := json.MarshalIndent(tweets, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	testutil.AssertGolden(t, goldenPath("json_tweet_list.json"), data)
}
