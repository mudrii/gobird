package types_test

import (
	"encoding/json"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

// TestTweetData_JSONRoundtrip marshals a fully-populated TweetData then
// unmarshals it and verifies all fields survive the round-trip.
func TestTweetData_JSONRoundtrip(t *testing.T) {
	inReply := "98765"
	dur := 4000
	quoted := &types.TweetData{
		ID:   "quoted-1",
		Text: "quoted text",
		Author: types.TweetAuthor{
			Username: "quoter",
			Name:     "Quoter Name",
		},
	}
	orig := types.TweetData{
		ID:                "tweet-1",
		Text:              "hello world",
		CreatedAt:         "Mon Jan 01 00:00:00 +0000 2024",
		ReplyCount:        3,
		RetweetCount:      7,
		LikeCount:         42,
		ConversationID:    "conv-1",
		InReplyToStatusID: &inReply,
		Author: types.TweetAuthor{
			Username: "testuser",
			Name:     "Test User",
		},
		AuthorID:       "author-1",
		IsBlueVerified: true,
		QuotedTweet:    quoted,
		Media: []types.TweetMedia{
			{
				Type:       "photo",
				URL:        "https://example.com/photo.jpg",
				Width:      1920,
				Height:     1080,
				PreviewURL: "https://example.com/preview.jpg",
				VideoURL:   "",
				DurationMs: &dur,
			},
		},
		Article: &types.TweetArticle{
			Title:       "Article Title",
			PreviewText: "Preview text here",
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got types.TweetData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID: want %q, got %q", orig.ID, got.ID)
	}
	if got.Text != orig.Text {
		t.Errorf("Text: want %q, got %q", orig.Text, got.Text)
	}
	if got.CreatedAt != orig.CreatedAt {
		t.Errorf("CreatedAt: want %q, got %q", orig.CreatedAt, got.CreatedAt)
	}
	if got.ReplyCount != orig.ReplyCount {
		t.Errorf("ReplyCount: want %d, got %d", orig.ReplyCount, got.ReplyCount)
	}
	if got.RetweetCount != orig.RetweetCount {
		t.Errorf("RetweetCount: want %d, got %d", orig.RetweetCount, got.RetweetCount)
	}
	if got.LikeCount != orig.LikeCount {
		t.Errorf("LikeCount: want %d, got %d", orig.LikeCount, got.LikeCount)
	}
	if got.ConversationID != orig.ConversationID {
		t.Errorf("ConversationID: want %q, got %q", orig.ConversationID, got.ConversationID)
	}
	if got.InReplyToStatusID == nil || *got.InReplyToStatusID != inReply {
		t.Errorf("InReplyToStatusID: want %q, got %v", inReply, got.InReplyToStatusID)
	}
	if got.Author.Username != orig.Author.Username {
		t.Errorf("Author.Username: want %q, got %q", orig.Author.Username, got.Author.Username)
	}
	if got.Author.Name != orig.Author.Name {
		t.Errorf("Author.Name: want %q, got %q", orig.Author.Name, got.Author.Name)
	}
	if got.AuthorID != orig.AuthorID {
		t.Errorf("AuthorID: want %q, got %q", orig.AuthorID, got.AuthorID)
	}
	if got.IsBlueVerified != orig.IsBlueVerified {
		t.Errorf("IsBlueVerified: want %v, got %v", orig.IsBlueVerified, got.IsBlueVerified)
	}
	if got.QuotedTweet == nil || got.QuotedTweet.ID != quoted.ID {
		t.Errorf("QuotedTweet.ID: want %q, got %v", quoted.ID, got.QuotedTweet)
	}
	if len(got.Media) != 1 {
		t.Fatalf("Media: want 1 item, got %d", len(got.Media))
	}
	if got.Media[0].Width != 1920 || got.Media[0].Height != 1080 {
		t.Errorf("Media[0] dimensions: want 1920x1080, got %dx%d", got.Media[0].Width, got.Media[0].Height)
	}
	if got.Media[0].DurationMs == nil || *got.Media[0].DurationMs != dur {
		t.Errorf("Media[0].DurationMs: want %d, got %v", dur, got.Media[0].DurationMs)
	}
	if got.Article == nil || got.Article.Title != "Article Title" {
		t.Errorf("Article.Title: want %q, got %v", "Article Title", got.Article)
	}
}

// TestTweetData_JSONRoundtrip_Zeros verifies that zero-value omitempty fields
// are absent from the marshalled JSON and unmarshal back to zero.
func TestTweetData_JSONRoundtrip_Zeros(t *testing.T) {
	orig := types.TweetData{
		ID:   "zero-tweet",
		Text: "minimal",
		Author: types.TweetAuthor{
			Username: "u",
			Name:     "N",
		},
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// omitempty fields should not appear
	for _, field := range []string{"createdAt", "replyCount", "retweetCount", "likeCount",
		"conversationId", "inReplyToStatusId", "authorId", "isBlueVerified", "quotedTweet", "media", "article"} {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal to map: %v", err)
		}
		if _, present := raw[field]; present {
			// isBlueVerified is bool with omitempty, false should be absent
			// replyCount/retweetCount/likeCount are int with omitempty, 0 absent
			t.Errorf("field %q should be absent for zero value but was present", field)
		}
	}
}

// TestTweetWithMeta_EmbedsTweetData verifies that TweetWithMeta embeds TweetData
// and the embedded fields are accessible directly.
func TestTweetWithMeta_EmbedsTweetData(t *testing.T) {
	twm := types.TweetWithMeta{
		TweetData: types.TweetData{
			ID:   "embed-1",
			Text: "embed text",
			Author: types.TweetAuthor{
				Username: "embedder",
				Name:     "Embedder",
			},
		},
		IsThread:       true,
		ThreadPosition: "root",
		HasSelfReplies: true,
	}

	// Access embedded fields directly.
	if twm.ID != "embed-1" {
		t.Errorf("ID via embed: want %q, got %q", "embed-1", twm.ID)
	}
	if twm.Text != "embed text" {
		t.Errorf("Text via embed: want %q, got %q", "embed text", twm.Text)
	}
	if twm.Author.Username != "embedder" {
		t.Errorf("Author.Username via embed: want %q, got %q", "embedder", twm.Author.Username)
	}
	if !twm.IsThread {
		t.Error("IsThread: want true")
	}
	if twm.ThreadPosition != "root" {
		t.Errorf("ThreadPosition: want %q, got %q", "root", twm.ThreadPosition)
	}
}

// TestTweetWithMeta_JSONRoundtrip verifies that TweetWithMeta marshals and
// unmarshals correctly, including the embedded TweetData fields.
func TestTweetWithMeta_JSONRoundtrip(t *testing.T) {
	rootID := "root-42"
	orig := types.TweetWithMeta{
		TweetData: types.TweetData{
			ID:   "meta-1",
			Text: "meta text",
			Author: types.TweetAuthor{
				Username: "metauser",
				Name:     "Meta User",
			},
		},
		IsThread:       true,
		ThreadPosition: "middle",
		HasSelfReplies: false,
		ThreadRootID:   &rootID,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got types.TweetWithMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != "meta-1" {
		t.Errorf("ID: want %q, got %q", "meta-1", got.ID)
	}
	if got.ThreadPosition != "middle" {
		t.Errorf("ThreadPosition: want %q, got %q", "middle", got.ThreadPosition)
	}
	if got.ThreadRootID == nil || *got.ThreadRootID != rootID {
		t.Errorf("ThreadRootID: want %q, got %v", rootID, got.ThreadRootID)
	}
}

// TestTwitterCookies_JSONTags verifies that TwitterCookies JSON field names
// match the expected tag values.
func TestTwitterCookies_JSONTags(t *testing.T) {
	c := types.TwitterCookies{
		AuthToken:    "tok123",
		Ct0:          "ct0val",
		CookieHeader: "auth_token=tok123; ct0=ct0val",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	cases := []struct{ field, want string }{
		{"authToken", "tok123"},
		{"ct0", "ct0val"},
		{"cookieHeader", "auth_token=tok123; ct0=ct0val"},
	}
	for _, tc := range cases {
		if got := raw[tc.field]; got != tc.want {
			t.Errorf("JSON field %q: want %q, got %q", tc.field, tc.want, got)
		}
	}
}

// TestTwitterCookies_CookieHeader_OmitemptyWhenEmpty verifies that an empty
// CookieHeader is omitted from JSON output.
func TestTwitterCookies_CookieHeader_OmitemptyWhenEmpty(t *testing.T) {
	c := types.TwitterCookies{
		AuthToken: "tok",
		Ct0:       "ct0",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["cookieHeader"]; ok {
		t.Error("cookieHeader should be absent when empty (omitempty)")
	}
}

// TestNewsItem_PostCount_Pointer verifies that nil PostCount is omitted from
// JSON and a non-nil value is included.
func TestNewsItem_PostCount_Pointer(t *testing.T) {
	t.Run("nil omitted", func(t *testing.T) {
		item := types.NewsItem{
			ID:       "n1",
			Headline: "headline",
		}
		data, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal map: %v", err)
		}
		if _, ok := raw["postCount"]; ok {
			t.Error("postCount should be absent when nil")
		}
	})

	t.Run("non-nil included", func(t *testing.T) {
		count := 42
		item := types.NewsItem{
			ID:        "n2",
			Headline:  "headline 2",
			PostCount: &count,
		}
		data, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal map: %v", err)
		}
		if _, ok := raw["postCount"]; !ok {
			t.Error("postCount should be present when non-nil")
		}
		var got int
		if err := json.Unmarshal(raw["postCount"], &got); err != nil {
			t.Fatalf("Unmarshal postCount: %v", err)
		}
		if got != 42 {
			t.Errorf("postCount: want 42, got %d", got)
		}
	})
}

// TestTweetMedia_OmitemptyFields verifies that zero Width/Height are omitted
// from marshalled JSON.
func TestTweetMedia_OmitemptyFields(t *testing.T) {
	t.Run("zero width and height omitted", func(t *testing.T) {
		m := types.TweetMedia{
			Type: "photo",
			URL:  "https://example.com/img.jpg",
		}
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		for _, field := range []string{"width", "height", "previewUrl", "videoUrl", "durationMs"} {
			if _, ok := raw[field]; ok {
				t.Errorf("field %q should be absent for zero/nil value", field)
			}
		}
	})

	t.Run("non-zero width and height present", func(t *testing.T) {
		m := types.TweetMedia{
			Type:   "photo",
			URL:    "https://example.com/img.jpg",
			Width:  800,
			Height: 600,
		}
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		for _, field := range []string{"width", "height"} {
			if _, ok := raw[field]; !ok {
				t.Errorf("field %q should be present for non-zero value", field)
			}
		}
	})
}

// TestTweetAuthor_JSONRoundtrip verifies TweetAuthor JSON field names.
func TestTweetAuthor_JSONRoundtrip(t *testing.T) {
	a := types.TweetAuthor{
		Username: "bird_user",
		Name:     "Bird User",
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if raw["username"] != "bird_user" {
		t.Errorf("username: want %q, got %q", "bird_user", raw["username"])
	}
	if raw["name"] != "Bird User" {
		t.Errorf("name: want %q, got %q", "Bird User", raw["name"])
	}
}

// TestTwitterUser_JSONRoundtrip verifies TwitterUser marshals with correct field names.
func TestTwitterUser_JSONRoundtrip(t *testing.T) {
	u := types.TwitterUser{
		ID:              "user-1",
		Username:        "twitteruser",
		Name:            "Twitter User",
		Description:     "A description",
		FollowersCount:  1000,
		FollowingCount:  500,
		IsBlueVerified:  true,
		ProfileImageURL: "https://example.com/profile.jpg",
		CreatedAt:       "2020-01-01",
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got types.TwitterUser
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != u.ID || got.Username != u.Username || got.FollowersCount != u.FollowersCount {
		t.Errorf("TwitterUser roundtrip mismatch: got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Additional model tests: defaults, validation, wire unmarshaling
// ---------------------------------------------------------------------------

func TestTwitterUser_ZeroValue_OmitemptyFields(t *testing.T) {
	u := types.TwitterUser{
		ID:       "zero-user",
		Username: "u",
		Name:     "N",
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"description", "followersCount", "followingCount",
		"isBlueVerified", "profileImageUrl", "createdAt", "_raw"} {
		if _, ok := raw[field]; ok {
			t.Errorf("zero-value field %q should be absent (omitempty)", field)
		}
	}
}

func TestTwitterList_JSONRoundtrip(t *testing.T) {
	owner := &types.ListOwner{ID: "o1", Username: "owner", Name: "Owner"}
	orig := types.TwitterList{
		ID:              "list-1",
		Name:            "Test List",
		Description:     "A test list",
		MemberCount:     50,
		SubscriberCount: 10,
		IsPrivate:       true,
		CreatedAt:       "2024-01-01",
		Owner:           owner,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got types.TwitterList
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != orig.ID {
		t.Errorf("ID: want %q, got %q", orig.ID, got.ID)
	}
	if got.Name != orig.Name {
		t.Errorf("Name: want %q, got %q", orig.Name, got.Name)
	}
	if !got.IsPrivate {
		t.Error("IsPrivate: want true")
	}
	if got.Owner == nil || got.Owner.ID != "o1" {
		t.Errorf("Owner: want id o1, got %v", got.Owner)
	}
}

func TestTwitterList_ZeroValue_OmitemptyFields(t *testing.T) {
	l := types.TwitterList{
		ID:   "empty-list",
		Name: "E",
	}
	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"description", "memberCount", "subscriberCount",
		"isPrivate", "createdAt", "owner", "_raw"} {
		if _, ok := raw[field]; ok {
			t.Errorf("zero-value field %q should be absent (omitempty)", field)
		}
	}
}

func TestNewsItem_JSONRoundtrip(t *testing.T) {
	count := 99
	orig := types.NewsItem{
		ID:          "news-1",
		Headline:    "Breaking News",
		Category:    "World",
		TimeAgo:     "1h",
		PostCount:   &count,
		Description: "Full description",
		URL:         "https://example.com/news",
		IsAiNews:    true,
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got types.NewsItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != orig.ID {
		t.Errorf("ID: want %q, got %q", orig.ID, got.ID)
	}
	if got.Headline != orig.Headline {
		t.Errorf("Headline: want %q, got %q", orig.Headline, got.Headline)
	}
	if !got.IsAiNews {
		t.Error("IsAiNews: want true")
	}
	if got.PostCount == nil || *got.PostCount != 99 {
		t.Errorf("PostCount: want 99, got %v", got.PostCount)
	}
}

func TestCurrentUserResult_JSONRoundtrip(t *testing.T) {
	orig := types.CurrentUserResult{
		ID:       "cu1",
		Username: "currentuser",
		Name:     "Current User",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got types.CurrentUserResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != orig.ID || got.Username != orig.Username || got.Name != orig.Name {
		t.Errorf("CurrentUserResult roundtrip mismatch: got %+v", got)
	}
}

func TestCurrentUserResult_OmitemptyFields(t *testing.T) {
	cur := types.CurrentUserResult{ID: "cu2"}
	data, err := json.Marshal(cur)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"username", "name"} {
		if _, ok := raw[field]; ok {
			t.Errorf("zero-value field %q should be absent (omitempty)", field)
		}
	}
}

func TestWireResponse_UnmarshalPartialJSON(t *testing.T) {
	input := `{"data":{"key":"value"}}`
	var wr types.WireResponse
	if err := json.Unmarshal([]byte(input), &wr); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if wr.Data == nil {
		t.Fatal("Data should not be nil")
	}
	if wr.Data["key"] != "value" {
		t.Errorf("Data[key]: want value, got %v", wr.Data["key"])
	}
	if wr.Errors != nil {
		t.Errorf("Errors should be nil for response without errors, got %v", wr.Errors)
	}
}

func TestWireResponse_UnmarshalWithErrors(t *testing.T) {
	input := `{"data":null,"errors":[{"message":"rate limited","extensions":{"code":"RATE_LIMITED","retry_after":15}}]}`
	var wr types.WireResponse
	if err := json.Unmarshal([]byte(input), &wr); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(wr.Errors) != 1 {
		t.Fatalf("want 1 error, got %d", len(wr.Errors))
	}
	if wr.Errors[0].Message != "rate limited" {
		t.Errorf("Message: want %q, got %q", "rate limited", wr.Errors[0].Message)
	}
	if wr.Errors[0].Extensions.Code != "RATE_LIMITED" {
		t.Errorf("Extensions.Code: want RATE_LIMITED, got %q", wr.Errors[0].Extensions.Code)
	}
	if wr.Errors[0].Extensions.RetryAfter == nil || *wr.Errors[0].Extensions.RetryAfter != 15 {
		t.Errorf("RetryAfter: want 15, got %v", wr.Errors[0].Extensions.RetryAfter)
	}
}

func TestWireResponse_UnmarshalMalformed(t *testing.T) {
	var wr types.WireResponse
	err := json.Unmarshal([]byte(`{invalid json`), &wr)
	if err == nil {
		t.Error("want error for malformed JSON")
	}
}

func TestWireRawTweet_UnmarshalPartialJSON(t *testing.T) {
	input := `{"__typename":"Tweet","rest_id":"123","is_blue_verified":true}`
	var raw types.WireRawTweet
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if raw.TypeName != "Tweet" {
		t.Errorf("TypeName: want Tweet, got %q", raw.TypeName)
	}
	if raw.RestID != "123" {
		t.Errorf("RestID: want 123, got %q", raw.RestID)
	}
	if !raw.IsBlueVerified {
		t.Error("IsBlueVerified: want true")
	}
	if raw.Legacy != nil {
		t.Error("Legacy should be nil when not in JSON")
	}
	if raw.Core != nil {
		t.Error("Core should be nil when not in JSON")
	}
}

func TestWireRawUser_UnmarshalPartialJSON(t *testing.T) {
	input := `{"__typename":"User","rest_id":"u1","is_blue_verified":false}`
	var raw types.WireRawUser
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if raw.TypeName != "User" {
		t.Errorf("TypeName: want User, got %q", raw.TypeName)
	}
	if raw.RestID != "u1" {
		t.Errorf("RestID: want u1, got %q", raw.RestID)
	}
	if raw.Legacy != nil {
		t.Error("Legacy should be nil when not in JSON")
	}
}

func TestWireTimelineInstruction_UnmarshalWithEntry(t *testing.T) {
	input := `{"type":"TimelineReplaceEntry","entry":{"entryId":"cursor-1","content":{"cursorType":"Bottom","value":"next"}}}`
	var inst types.WireTimelineInstruction
	if err := json.Unmarshal([]byte(input), &inst); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if inst.Type != "TimelineReplaceEntry" {
		t.Errorf("Type: want TimelineReplaceEntry, got %q", inst.Type)
	}
	if inst.Entry == nil {
		t.Fatal("Entry should not be nil")
	}
	if inst.Entry.Content.CursorType != "Bottom" {
		t.Errorf("CursorType: want Bottom, got %q", inst.Entry.Content.CursorType)
	}
}

func TestPageResult_ZeroValue(t *testing.T) {
	var pr types.PageResult[types.TweetData]
	if pr.Items != nil {
		t.Error("zero-value Items should be nil")
	}
	if pr.NextCursor != "" {
		t.Error("zero-value NextCursor should be empty")
	}
	if pr.Success {
		t.Error("zero-value Success should be false")
	}
	if pr.Error != nil {
		t.Error("zero-value Error should be nil")
	}
}

func TestFetchOptions_Defaults(t *testing.T) {
	var opts types.FetchOptions
	if opts.Cursor != "" {
		t.Error("default Cursor should be empty")
	}
	if opts.Count != 0 {
		t.Error("default Count should be 0")
	}
	if opts.Limit != 0 {
		t.Error("default Limit should be 0")
	}
	if opts.MaxPages != 0 {
		t.Error("default MaxPages should be 0")
	}
	if opts.IncludeRaw {
		t.Error("default IncludeRaw should be false")
	}
	if opts.QuoteDepth != 0 {
		t.Error("default QuoteDepth should be 0")
	}
}

func TestSearchOptions_EmbedsFetchOptions(t *testing.T) {
	opts := types.SearchOptions{
		FetchOptions: types.FetchOptions{
			Count: 20,
			Limit: 100,
		},
		Product: "Latest",
	}
	if opts.Count != 20 {
		t.Errorf("Count: want 20, got %d", opts.Count)
	}
	if opts.Product != "Latest" {
		t.Errorf("Product: want Latest, got %q", opts.Product)
	}
}

func TestThreadOptions_FilterMode(t *testing.T) {
	opts := types.ThreadOptions{
		FilterMode: "author_chain",
	}
	if opts.FilterMode != "author_chain" {
		t.Errorf("FilterMode: want author_chain, got %q", opts.FilterMode)
	}
}

func TestTweetMedia_DurationMs_NilVsZero(t *testing.T) {
	t.Run("nil duration omitted", func(t *testing.T) {
		m := types.TweetMedia{Type: "photo", URL: "u"}
		data, _ := json.Marshal(m)
		var raw map[string]json.RawMessage
		json.Unmarshal(data, &raw)
		if _, ok := raw["durationMs"]; ok {
			t.Error("nil durationMs should be omitted")
		}
	})
	t.Run("zero duration included", func(t *testing.T) {
		zero := 0
		m := types.TweetMedia{Type: "gif", URL: "u", DurationMs: &zero}
		data, _ := json.Marshal(m)
		var raw map[string]json.RawMessage
		json.Unmarshal(data, &raw)
		if _, ok := raw["durationMs"]; !ok {
			t.Error("zero durationMs (non-nil pointer) should be included")
		}
	})
}
