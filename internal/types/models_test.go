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
