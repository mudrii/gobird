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

func TestMapTweetResult_Nil(t *testing.T) {
	got := parsing.MapTweetResult(nil)
	if got != nil {
		t.Errorf("want nil for nil input, got %+v", got)
	}
}

func TestMapTweetResult_Basic(t *testing.T) {
	authorID := "u1"
	raw := &types.WireRawTweet{
		RestID: "tweet1",
		Legacy: &types.WireTweetLegacy{
			FullText:  "hello world",
			UserIDStr: authorID,
		},
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.ID != "tweet1" {
		t.Errorf("want ID tweet1, got %q", td.ID)
	}
	if td.Text != "hello world" {
		t.Errorf("want text hello world, got %q", td.Text)
	}
	if td.AuthorID != authorID {
		t.Errorf("want AuthorID %q, got %q", authorID, td.AuthorID)
	}
}

func TestMapTweetResult_IsBlueVerified_TopLevel(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID:         "456",
		IsBlueVerified: true,
		Legacy:         &types.WireTweetLegacy{FullText: "verified"},
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if !td.IsBlueVerified {
		t.Error("IsBlueVerified should be true from top-level field")
	}
}

func TestMapTweetResult_WithMedia(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "789",
		Legacy: &types.WireTweetLegacy{
			FullText: "with photo",
			ExtendedEntities: &types.WireMediaEntities{
				Media: []types.WireMedia{
					{Type: "photo", MediaURLHttps: "https://example.com/photo.jpg"},
				},
			},
		},
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if len(td.Media) != 1 {
		t.Fatalf("want 1 media item, got %d", len(td.Media))
	}
	if td.Media[0].Type != "photo" {
		t.Errorf("want media type photo, got %q", td.Media[0].Type)
	}
}

func TestMapTweetResult_QuotedTweet_DepthZero(t *testing.T) {
	quoted := &types.WireRawTweet{
		RestID: "q1",
		Legacy: &types.WireTweetLegacy{FullText: "quoted text"},
	}
	raw := &types.WireRawTweet{
		RestID: "outer",
		Legacy: &types.WireTweetLegacy{FullText: "outer text"},
		QuotedResult: &types.WireTweetResult{
			Result: quoted,
		},
	}
	td := parsing.MapTweetResultWithOptions(raw, parsing.TweetParseOptions{QuoteDepth: 0})
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.QuotedTweet != nil {
		t.Error("at depth 0, QuotedTweet should be nil")
	}
}

func TestMapTweetResult_QuotedTweet_Depth1(t *testing.T) {
	quoted := &types.WireRawTweet{
		RestID: "q2",
		Legacy: &types.WireTweetLegacy{FullText: "nested"},
	}
	raw := &types.WireRawTweet{
		RestID: "outer2",
		Legacy: &types.WireTweetLegacy{FullText: "outer2 text"},
		QuotedResult: &types.WireTweetResult{
			Result: quoted,
		},
	}
	td := parsing.MapTweetResultWithOptions(raw, parsing.TweetParseOptions{QuoteDepth: 1})
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.QuotedTweet == nil {
		t.Fatal("at depth 1, QuotedTweet should be populated")
	}
	if td.QuotedTweet.ID != "q2" {
		t.Errorf("want quoted tweet ID q2, got %q", td.QuotedTweet.ID)
	}
}

func TestExtractTweetText_ArticlePriority(t *testing.T) {
	raw := &types.WireRawTweet{
		Legacy: &types.WireTweetLegacy{FullText: "legacy"},
		NoteTweet: &types.WireNoteTweet{
			NoteTweetResults: types.WireNoteTweetResults{
				Result: &types.WireNoteTweetResult{Text: "note"},
			},
		},
		Article: &types.WireArticle{
			ArticleResults: types.WireArticleResults{
				Result: &types.WireArticleResult{Title: "article title"},
			},
		},
	}
	got := parsing.ExtractTweetText(raw)
	if got != "article title" {
		t.Errorf("article should take priority over note and legacy, got %q", got)
	}
}

func TestExtractTweetText_NotePriority(t *testing.T) {
	raw := &types.WireRawTweet{
		Legacy: &types.WireTweetLegacy{FullText: "legacy text"},
		NoteTweet: &types.WireNoteTweet{
			NoteTweetResults: types.WireNoteTweetResults{
				Result: &types.WireNoteTweetResult{Text: "note text"},
			},
		},
	}
	got := parsing.ExtractTweetText(raw)
	if got != "note text" {
		t.Errorf("note should take priority over legacy, got %q", got)
	}
}

func TestExtractTweetText_Legacy(t *testing.T) {
	raw := &types.WireRawTweet{
		Legacy: &types.WireTweetLegacy{FullText: "only legacy"},
	}
	got := parsing.ExtractTweetText(raw)
	if got != "only legacy" {
		t.Errorf("should fall back to legacy full_text, got %q", got)
	}
}

func TestUnwrapTweetResult_Visibility(t *testing.T) {
	inner := &types.WireRawTweet{RestID: "inner42"}
	outer := &types.WireRawTweet{
		TypeName: "TweetWithVisibilityResults",
		Tweet:    inner,
	}
	got := parsing.UnwrapTweetResult(outer)
	if got == nil {
		t.Fatal("want non-nil from unwrap")
	}
	if got.RestID != "inner42" {
		t.Errorf("want inner42, got %q", got.RestID)
	}
}
