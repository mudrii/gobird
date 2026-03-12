package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func TestExtractTweetTextPriority(t *testing.T) {
	// Article title takes highest priority.
	raw := &types.WireRawTweet{
		Legacy: &types.WireTweetLegacy{FullText: "legacy text"},
		NoteTweet: &types.WireNoteTweet{
			NoteTweetResults: types.WireNoteTweetResults{
				Result: &types.WireNoteTweetResult{Text: "note text"},
			},
		},
		Article: &types.WireArticle{
			ArticleResults: types.WireArticleResults{
				Result: &types.WireArticleResult{Title: "article title"},
			},
		},
	}
	if got := parsing.ExtractTweetText(raw); got != "article title" {
		t.Errorf("want article title, got %q", got)
	}

	// Note tweet wins over legacy when no article.
	raw.Article = nil
	if got := parsing.ExtractTweetText(raw); got != "note text" {
		t.Errorf("want note text, got %q", got)
	}

	// Legacy wins when no note tweet or article.
	raw.NoteTweet = nil
	if got := parsing.ExtractTweetText(raw); got != "legacy text" {
		t.Errorf("want legacy text, got %q", got)
	}
}

func TestUnwrapTweetResult(t *testing.T) {
	inner := &types.WireRawTweet{RestID: "inner"}
	outer := &types.WireRawTweet{Tweet: inner, RestID: "outer"}
	got := parsing.UnwrapTweetResult(outer)
	if got.RestID != "inner" {
		t.Errorf("want inner, got %q", got.RestID)
	}
}

func TestIsBlueVerifiedTopLevel(t *testing.T) {
	// Correction #8: is_blue_verified is top-level on result, NOT inside legacy.
	raw := &types.WireRawTweet{
		RestID:         "123",
		IsBlueVerified: true,
		Legacy:         &types.WireTweetLegacy{FullText: "hi"},
	}
	td := parsing.MapTweetResult(raw)
	if !td.IsBlueVerified {
		t.Error("IsBlueVerified should be true from top-level field")
	}
}
