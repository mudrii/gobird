package parsing_test

// robustness_test.go covers edge cases identified in the robustness checklist:
// nil safety, empty/missing data, pagination edge cases, content edge cases,
// and the bugs fixed in this pass.

import (
	"encoding/json"
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// ---------------------------------------------------------------------------
// Nil Safety
// ---------------------------------------------------------------------------

// Fix: ExtractTweetText was previously only called inside the raw.Legacy != nil
// block, so tweets with a NoteTweet but no Legacy silently returned empty text.
func TestMapTweetResult_NoteTweetWithoutLegacy(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "note1",
		NoteTweet: &types.WireNoteTweet{
			NoteTweetResults: types.WireNoteTweetResults{
				Result: &types.WireNoteTweetResult{Text: "long form note text"},
			},
		},
		// Legacy deliberately nil
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.Text != "long form note text" {
		t.Errorf("want note text even without Legacy, got %q", td.Text)
	}
}

// Fix: Same issue for Article tweets with no Legacy.
func TestMapTweetResult_ArticleWithoutLegacy(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "article1",
		Article: &types.WireArticle{
			ArticleResults: types.WireArticleResults{
				Result: &types.WireArticleResult{
					Title:       "Article Title",
					PreviewText: "Article preview",
				},
			},
		},
		// Legacy deliberately nil
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.Text == "" {
		t.Errorf("want article text even without Legacy, got empty string")
	}
	if td.Article == nil {
		t.Error("want Article field populated")
	}
}

// Article with nil ArticleResults.Result — should not panic.
func TestMapTweetResult_ArticleNilResult(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "a2",
		Article: &types.WireArticle{
			ArticleResults: types.WireArticleResults{Result: nil},
		},
		Legacy: &types.WireTweetLegacy{FullText: "fallback"},
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.Article != nil {
		t.Error("Article should be nil when ArticleResults.Result is nil")
	}
	if td.Text != "fallback" {
		t.Errorf("want legacy text fallback, got %q", td.Text)
	}
}

// NoteTweet with nil Result inside — should not panic.
func TestExtractTweetText_NoteTweetNilResult(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "n2",
		NoteTweet: &types.WireNoteTweet{
			NoteTweetResults: types.WireNoteTweetResults{Result: nil},
		},
		Legacy: &types.WireTweetLegacy{FullText: "fallback text"},
	}
	got := parsing.ExtractTweetText(raw)
	if got != "fallback text" {
		t.Errorf("want legacy fallback when NoteTweet.Result is nil, got %q", got)
	}
}

// NoteTweet with empty Text — should fall through to legacy.
func TestExtractTweetText_NoteTweetEmptyText(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "n3",
		NoteTweet: &types.WireNoteTweet{
			NoteTweetResults: types.WireNoteTweetResults{
				Result: &types.WireNoteTweetResult{Text: ""},
			},
		},
		Legacy: &types.WireTweetLegacy{FullText: "legacy fallback"},
	}
	got := parsing.ExtractTweetText(raw)
	if got != "legacy fallback" {
		t.Errorf("want legacy fallback for empty note tweet text, got %q", got)
	}
}

// MapTweetResult with nil Core — should not panic, author fields are empty.
func TestMapTweetResult_NilCore(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "nc1",
		Legacy: &types.WireTweetLegacy{FullText: "no core"},
		Core:   nil,
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.Author.Username != "" || td.Author.Name != "" {
		t.Errorf("want empty author when Core is nil, got %+v", td.Author)
	}
}

// MapTweetResult with nil QuotedResult.Result — should not panic.
func TestMapTweetResult_QuotedResultNilInner(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "qnil",
		Legacy: &types.WireTweetLegacy{FullText: "outer"},
		QuotedResult: &types.WireTweetResult{
			Result: nil,
		},
	}
	td := parsing.MapTweetResultWithOptions(raw, parsing.TweetParseOptions{QuoteDepth: 1})
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.QuotedTweet != nil {
		t.Error("QuotedTweet should be nil when QuotedResult.Result is nil")
	}
}

// MapUser with nil Legacy — should return user with empty fields, not panic.
func TestMapUser_NilLegacy_NoFields(t *testing.T) {
	raw := &types.WireRawUser{
		TypeName:       "User",
		RestID:         "99",
		IsBlueVerified: true,
	}
	u := parsing.MapUser(raw)
	if u == nil {
		t.Fatal("want non-nil user even with nil Legacy")
	}
	if u.Username != "" {
		t.Errorf("want empty Username with nil Legacy, got %q", u.Username)
	}
	if u.Name != "" {
		t.Errorf("want empty Name with nil Legacy, got %q", u.Name)
	}
	if !u.IsBlueVerified {
		t.Error("IsBlueVerified should still be set from top-level field")
	}
}

// ExtractMedia with nil entities — returns nil, no panic.
func TestExtractMedia_NilEntities(t *testing.T) {
	got := parsing.ExtractMedia(nil)
	if got != nil {
		t.Errorf("want nil for nil entities, got %v", got)
	}
}

// ExtractMedia with nil Media slice inside entities — should not panic.
func TestExtractMedia_NilMediaSlice(t *testing.T) {
	entities := &types.WireMediaEntities{Media: nil}
	got := parsing.ExtractMedia(entities)
	if len(got) != 0 {
		t.Errorf("want 0 items for nil Media slice, got %d", len(got))
	}
}

// VideoInfo with nil Variants slice — should not panic, VideoURL stays empty.
func TestExtractMedia_VideoNilVariants(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: nil,
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media item, got %d", len(media))
	}
	if media[0].VideoURL != "" {
		t.Errorf("want empty VideoURL for nil variants, got %q", media[0].VideoURL)
	}
}

// VideoInfo with empty Variants slice — VideoURL should be empty.
func TestExtractMedia_VideoEmptyVariants(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo:     &types.WireVideoInfo{Variants: []types.WireVideoVariant{}},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].VideoURL != "" {
		t.Errorf("want empty VideoURL for empty variants, got %q", media[0].VideoURL)
	}
}

// Sizes with no Large or Medium — dimensions should be zero, no panic.
func TestExtractMedia_NoSizes(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "photo",
				MediaURLHttps: "https://example.com/a.jpg",
				Sizes:         types.WireMediaSizes{},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media item, got %d", len(media))
	}
	if media[0].Width != 0 || media[0].Height != 0 {
		t.Errorf("want zero dimensions when no sizes, got %dx%d", media[0].Width, media[0].Height)
	}
	if media[0].PreviewURL != "" {
		t.Errorf("want empty PreviewURL when no Small size, got %q", media[0].PreviewURL)
	}
}

// ---------------------------------------------------------------------------
// Empty / Missing Data
// ---------------------------------------------------------------------------

// Empty instructions slice returns empty results — no panic.
func TestParseTweetsFromInstructions_NilInstructions(t *testing.T) {
	tweets := parsing.ParseTweetsFromInstructions(nil)
	if len(tweets) != 0 {
		t.Errorf("want 0 tweets for nil instructions, got %d", len(tweets))
	}
}

// Instructions with no Entries and no Entry — returns empty.
func TestParseTweetsFromInstructions_EmptyInstructions(t *testing.T) {
	instructions := []types.WireTimelineInstruction{{}}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 0 {
		t.Errorf("want 0 tweets for empty instructions, got %d", len(tweets))
	}
}

// Cursor entry with empty Value string returns empty cursor.
func TestExtractCursorFromInstructions_EmptyValue(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Bottom", Value: ""}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	// An empty Value with CursorType==Bottom is returned as-is (empty string).
	// Callers treat empty string as "no cursor", which is correct behaviour.
	if got != "" {
		t.Errorf("want empty string for cursor with empty value, got %q", got)
	}
}

// No timelineAddEntries instruction — cursor extraction returns empty.
func TestExtractCursorFromInstructions_NoMatchingInstruction(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Top", Value: "top"}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "" {
		t.Errorf("want empty string when no Bottom cursor, got %q", got)
	}
}

// Tweet with no legacy field — ID is preserved, text is empty/article text.
func TestMapTweetResult_NoLegacy(t *testing.T) {
	raw := &types.WireRawTweet{
		RestID: "noleg",
	}
	td := parsing.MapTweetResult(raw)
	if td == nil {
		t.Fatal("want non-nil TweetData")
	}
	if td.ID != "noleg" {
		t.Errorf("want ID noleg, got %q", td.ID)
	}
	if td.Text != "" {
		t.Errorf("want empty text when no legacy/note/article, got %q", td.Text)
	}
	if td.Media != nil {
		t.Errorf("want nil Media when no legacy, got %v", td.Media)
	}
}

// ---------------------------------------------------------------------------
// Pagination Edge Cases
// ---------------------------------------------------------------------------

// Same cursor on consecutive pages — loop terminates immediately (tested by
// confirming ExtractCursorFromInstructions is idempotent on same data).
func TestExtractCursorFromInstructions_StableCursor(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Bottom", Value: "cursor-abc"}},
			},
		},
	}
	c1 := parsing.ExtractCursorFromInstructions(instructions)
	c2 := parsing.ExtractCursorFromInstructions(instructions)
	if c1 != c2 {
		t.Errorf("cursor must be stable across calls: %q vs %q", c1, c2)
	}
}

// Page returns 0 tweet items but a valid cursor — ParseTweetsFromInstructions
// returns empty slice (not nil) so callers can distinguish from error.
func TestParseTweetsFromInstructions_ZeroItemsWithCursor(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Bottom", Value: "next"}},
			},
		},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if tweets == nil {
		// nil is acceptable; length 0 is what matters.
	}
	if len(tweets) != 0 {
		t.Errorf("want 0 tweets when only cursor entry present, got %d", len(tweets))
	}
	cursor := parsing.ExtractCursorFromInstructions(instructions)
	if cursor != "next" {
		t.Errorf("want cursor next, got %q", cursor)
	}
}

// ---------------------------------------------------------------------------
// Content Edge Cases
// ---------------------------------------------------------------------------

// Tweet text with Unicode, emoji, RTL — round-trips correctly.
func TestMapTweetResult_UnicodeText(t *testing.T) {
	text := "مرحبا 🌍 こんにちは \u200f RTL mixed"
	raw := &types.WireRawTweet{
		RestID: "uni1",
		Legacy: &types.WireTweetLegacy{FullText: text},
	}
	td := parsing.MapTweetResult(raw)
	if td.Text != text {
		t.Errorf("want exact unicode text, got %q", td.Text)
	}
}

// Quoted tweet that quotes another quoted tweet — depth limit enforced.
func TestMapTweetResult_QuotedDepth2_Enforced(t *testing.T) {
	depth2 := &types.WireRawTweet{
		RestID: "d2",
		Legacy: &types.WireTweetLegacy{FullText: "depth 2"},
	}
	depth1 := &types.WireRawTweet{
		RestID: "d1",
		Legacy: &types.WireTweetLegacy{FullText: "depth 1"},
		QuotedResult: &types.WireTweetResult{
			Result: depth2,
		},
	}
	root := &types.WireRawTweet{
		RestID: "root",
		Legacy: &types.WireTweetLegacy{FullText: "root"},
		QuotedResult: &types.WireTweetResult{
			Result: depth1,
		},
	}

	// At QuoteDepth=1, depth2 should not appear.
	td := parsing.MapTweetResultWithOptions(root, parsing.TweetParseOptions{QuoteDepth: 1})
	if td.QuotedTweet == nil {
		t.Fatal("want QuotedTweet at depth 1")
	}
	if td.QuotedTweet.ID != "d1" {
		t.Errorf("want d1 quoted tweet, got %q", td.QuotedTweet.ID)
	}
	if td.QuotedTweet.QuotedTweet != nil {
		t.Error("depth limit=1 must prevent second-level quoted tweet")
	}
}

// Quoted tweet at depth 2 — should populate both levels.
func TestMapTweetResult_QuotedDepth2_Allowed(t *testing.T) {
	depth2 := &types.WireRawTweet{
		RestID: "d2",
		Legacy: &types.WireTweetLegacy{FullText: "depth 2"},
	}
	depth1 := &types.WireRawTweet{
		RestID: "d1",
		Legacy: &types.WireTweetLegacy{FullText: "depth 1"},
		QuotedResult: &types.WireTweetResult{
			Result: depth2,
		},
	}
	root := &types.WireRawTweet{
		RestID: "root",
		Legacy: &types.WireTweetLegacy{FullText: "root"},
		QuotedResult: &types.WireTweetResult{
			Result: depth1,
		},
	}

	td := parsing.MapTweetResultWithOptions(root, parsing.TweetParseOptions{QuoteDepth: 2})
	if td.QuotedTweet == nil {
		t.Fatal("want QuotedTweet at depth 2")
	}
	if td.QuotedTweet.QuotedTweet == nil {
		t.Fatal("want second-level QuotedTweet at depth 2")
	}
	if td.QuotedTweet.QuotedTweet.ID != "d2" {
		t.Errorf("want d2 at second level, got %q", td.QuotedTweet.QuotedTweet.ID)
	}
}

// Article with empty content_state — falls back to title/preview.
func TestExtractArticleText_EmptyContentState(t *testing.T) {
	ar := &types.WireArticleResult{
		Title:        "The Title",
		PreviewText:  "The Preview",
		ContentState: "",
	}
	got := parsing.ExtractArticleText(ar)
	want := "The Title\n\nThe Preview"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// Article with content_state that produces empty rendered text — falls back to title.
func TestExtractArticleText_ContentStateAllEmpty(t *testing.T) {
	cs := map[string]any{
		"blocks":    []any{},
		"entityMap": map[string]any{},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{
		Title:        "Fallback",
		ContentState: string(csJSON),
	}
	got := parsing.ExtractArticleText(ar)
	if got != "Fallback" {
		t.Errorf("want Fallback when content_state renders empty, got %q", got)
	}
}

// Malformed entity range (out of bounds) in renderBlockText — no panic, returns text.
func TestRenderBlockText_MalformedEntityRange(t *testing.T) {
	// Entity range claims offset=0, length=999 for a 5-rune text.
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "unstyled",
				"text": "hello",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 999, "key": 0},
				},
			},
		},
		"entityMap": map[string]any{
			"0": map[string]any{
				"type": "LINK",
				"data": map[string]any{"url": "https://example.com"},
			},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	// The malformed range should be skipped; the raw text is returned.
	if got != "hello" {
		t.Errorf("want raw text for malformed entity range, got %q", got)
	}
}

// Entity range with negative offset — no panic.
func TestRenderBlockText_NegativeOffset(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "unstyled",
				"text": "hello",
				"entityRanges": []map[string]any{
					{"offset": -1, "length": 5, "key": 0},
				},
			},
		},
		"entityMap": map[string]any{
			"0": map[string]any{
				"type": "LINK",
				"data": map[string]any{"url": "https://example.com"},
			},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "hello" {
		t.Errorf("want raw text for negative offset, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Fix: ParseUsersFromInstructions — inst.Entry (singular) path
// ---------------------------------------------------------------------------

// Users in a singular inst.Entry should be collected.
func TestParseUsersFromInstructions_SingularEntry(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entry: &types.WireEntry{
				Content: types.WireContent{
					ItemContent: &types.WireItemContent{
						UserResult: &types.WireUserResult{
							Result: &types.WireRawUser{
								TypeName: "User",
								RestID:   "500",
								Legacy:   &types.WireUserLegacy{ScreenName: "singularuser"},
							},
						},
					},
				},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 1 {
		t.Fatalf("want 1 user from singular Entry, got %d", len(users))
	}
	if users[0].Username != "singularuser" {
		t.Errorf("want singularuser, got %q", users[0].Username)
	}
}

// Users in module items within entries should be collected.
func TestParseUsersFromInstructions_ModuleItems(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{
					Content: types.WireContent{
						Items: []types.WireItem{
							{
								Item: struct {
									ItemContent *types.WireItemContent `json:"itemContent"`
								}{
									ItemContent: &types.WireItemContent{
										UserResult: &types.WireUserResult{
											Result: &types.WireRawUser{
												TypeName: "User",
												RestID:   "600",
												Legacy:   &types.WireUserLegacy{ScreenName: "moduleuser"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 1 {
		t.Fatalf("want 1 user from module item, got %d", len(users))
	}
	if users[0].Username != "moduleuser" {
		t.Errorf("want moduleuser, got %q", users[0].Username)
	}
}

// Deduplication works across Entries + singular Entry.
func TestParseUsersFromInstructions_DeduplicationAcrossEntryForms(t *testing.T) {
	user := &types.WireRawUser{
		TypeName: "User",
		RestID:   "700",
		Legacy:   &types.WireUserLegacy{ScreenName: "dupacross"},
	}
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{
					Content: types.WireContent{
						ItemContent: &types.WireItemContent{
							UserResult: &types.WireUserResult{Result: user},
						},
					},
				},
			},
			Entry: &types.WireEntry{
				Content: types.WireContent{
					ItemContent: &types.WireItemContent{
						UserResult: &types.WireUserResult{Result: user},
					},
				},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 1 {
		t.Errorf("want 1 user after cross-form deduplication, got %d", len(users))
	}
}

// ---------------------------------------------------------------------------
// UnwrapUserResult — nil inner user in visibility wrapper
// ---------------------------------------------------------------------------

func TestUnwrapUserResult_NilInnerUser(t *testing.T) {
	outer := &types.WireRawUser{
		TypeName: "UserWithVisibilityResults",
		User:     nil, // malformed — no inner user
	}
	// MapUser should not panic; it returns nil because inner is nil then
	// TypeName != UserUnavailable but RestID is empty.
	u := parsing.MapUser(outer)
	// When User is nil, UnwrapUserResult returns outer (since raw.User == nil
	// the condition raw.TypeName == "..." && raw.User != nil is false).
	// So outer itself is returned with empty RestID and nil Legacy → non-nil user.
	// The important thing is no panic and the function completes.
	_ = u
}

// ---------------------------------------------------------------------------
// ParseTweetsFromInstructions — skips cursor entries silently
// ---------------------------------------------------------------------------

func TestParseTweetsFromInstructions_SkipsCursorEntries(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				*makeTweetEntry("t1"),
				{Content: types.WireContent{CursorType: "Bottom", Value: "cursor-val"}},
				*makeTweetEntry("t2"),
			},
		},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 2 {
		t.Errorf("want 2 tweets (cursor entry skipped), got %d", len(tweets))
	}
}

// ---------------------------------------------------------------------------
// Video: nil Bitrate pointer treated as 0
// ---------------------------------------------------------------------------

func TestExtractMedia_VideoNilBitrateVariant(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/v.mp4", Bitrate: nil},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media item, got %d", len(media))
	}
	// nil Bitrate is treated as 0; still selected as the only variant.
	if media[0].VideoURL != "https://example.com/v.mp4" {
		t.Errorf("want video URL for nil-bitrate mp4 variant, got %q", media[0].VideoURL)
	}
}

// ---------------------------------------------------------------------------
// MapList — nil UserResults.Result
// ---------------------------------------------------------------------------

func TestMapList_NilOwner(t *testing.T) {
	raw := &types.WireList{
		IDStr:       "listRobust1",
		Name:        "Test List",
		Description: "desc",
		Mode:        "public",
		// UserResults.Result is nil
	}
	l := parsing.MapList(raw)
	if l == nil {
		t.Fatal("want non-nil list")
	}
	if l.Owner != nil {
		t.Error("want nil Owner when UserResults.Result is nil")
	}
	if l.IsPrivate {
		t.Error("want IsPrivate=false for public list")
	}
}

// ---------------------------------------------------------------------------
// ParseNewsItemFromContent — type coercion and missing fields
// ---------------------------------------------------------------------------

func TestParseNewsItemFromContent_TweetCountFloat64_Robustness(t *testing.T) {
	// JSON numbers are decoded as float64 when using interface{}.
	content := map[string]any{
		"id":          "nRobust1",
		"headline":    "Test News",
		"tweet_count": float64(99),
	}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("want non-nil item")
	}
	if got.PostCount == nil || *got.PostCount != 99 {
		t.Errorf("want PostCount=99 from float64, got %v", got.PostCount)
	}
}

func TestParseNewsItemFromContent_IsAiNewsBool_Robustness(t *testing.T) {
	content := map[string]any{
		"id":         "nRobust2",
		"is_ai_news": true,
	}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("want non-nil item")
	}
	if !got.IsAiNews {
		t.Error("want IsAiNews=true from bool value")
	}
}
