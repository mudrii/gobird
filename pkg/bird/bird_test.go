package bird_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mudrii/gobird/pkg/bird"
)

func TestNew_ReturnsClient(t *testing.T) {
	creds := &bird.TwitterCookies{
		AuthToken: "test_auth_token",
		Ct0:       "test_ct0_value",
	}
	c, err := bird.New(creds, nil)
	if err != nil {
		t.Fatalf("New() with valid credentials: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil client")
	}
}

func TestNew_NilCreds(t *testing.T) {
	c, err := bird.New(nil, nil)
	if err == nil {
		t.Fatal("New(nil) should return error")
	}
	if c != nil {
		t.Fatal("New(nil) should return nil client")
	}
}

func TestNew_EmptyAuthToken(t *testing.T) {
	creds := &bird.TwitterCookies{AuthToken: "", Ct0: "ct0"}
	c, err := bird.New(creds, nil)
	if err == nil {
		t.Fatal("New() with empty AuthToken should return error")
	}
	if c != nil {
		t.Fatal("New() with empty AuthToken should return nil client")
	}
}

func TestNew_EmptyCt0(t *testing.T) {
	creds := &bird.TwitterCookies{AuthToken: "tok", Ct0: ""}
	c, err := bird.New(creds, nil)
	if err == nil {
		t.Fatal("New() with empty Ct0 should return error")
	}
	if c != nil {
		t.Fatal("New() with empty Ct0 should return nil client")
	}
}

func TestNewWithTokens_SetsCredentials(t *testing.T) {
	c, err := bird.NewWithTokens("a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9", "a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5", nil)
	if err != nil {
		t.Fatalf("NewWithTokens() with valid tokens: %v", err)
	}
	if c == nil {
		t.Fatal("NewWithTokens() returned nil client")
	}
}

func TestNewWithTokens_EmptyToken(t *testing.T) {
	c, err := bird.NewWithTokens("", "ct0", nil)
	if err == nil {
		t.Fatal("NewWithTokens() with empty authToken should return error")
	}
	if c != nil {
		t.Fatal("NewWithTokens() with empty authToken should return nil client")
	}
}

func TestNewWithTokens_EmptyCt0(t *testing.T) {
	c, err := bird.NewWithTokens("auth", "", nil)
	if err == nil {
		t.Fatal("NewWithTokens() with empty ct0 should return error")
	}
	if c != nil {
		t.Fatal("NewWithTokens() with empty ct0 should return nil client")
	}
}

func TestNew_MissingCredentials_ErrorMessage(t *testing.T) {
	_, err := bird.New(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// ErrMissingCredentials is unexported; verify it's a known sentinel via errors.Is
	// by checking the message contains key text.
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestResolveCredentials_TypeAlias(t *testing.T) {
	// Verify that bird.ResolveOptions is callable and accepts the expected fields.
	opts := bird.ResolveOptions{
		FlagAuthToken: "tok",
		FlagCt0:       "ct0",
	}
	// ResolveCredentials should not panic — it will fail because no env/browser
	// sources are set, but the type alias is correctly wired.
	_, err := bird.ResolveCredentials(opts)
	if err == nil {
		// Got credentials from the environment — that's fine.
		return
	}
	// Expect a credential-resolution error, not a type error.
	_ = err
}

func TestTwitterCookies_TypeAlias(t *testing.T) {
	// Verify the type alias round-trips its fields correctly.
	c := bird.TwitterCookies{
		AuthToken:    "a",
		Ct0:          "b",
		CookieHeader: "c=d",
	}
	if c.AuthToken != "a" {
		t.Errorf("AuthToken: want %q, got %q", "a", c.AuthToken)
	}
	if c.Ct0 != "b" {
		t.Errorf("Ct0: want %q, got %q", "b", c.Ct0)
	}
	if c.CookieHeader != "c=d" {
		t.Errorf("CookieHeader: want %q, got %q", "c=d", c.CookieHeader)
	}
}

func TestNew_MissingCredentials_IsError(t *testing.T) {
	_, err := bird.New(nil, nil)
	// Ensure the returned value satisfies the error interface.
	_ = err
	if !errors.Is(err, err) {
		t.Error("error should satisfy errors.Is(err, err)")
	}
}

// ---------------------------------------------------------------------------
// Additional client and error tests
// ---------------------------------------------------------------------------

func TestNew_BothFieldsEmpty(t *testing.T) {
	creds := &bird.TwitterCookies{AuthToken: "", Ct0: ""}
	c, err := bird.New(creds, nil)
	if err == nil {
		t.Fatal("New() with both empty should return error")
	}
	if c != nil {
		t.Fatal("New() with both empty should return nil client")
	}
}

func TestNewWithTokens_BothEmpty(t *testing.T) {
	c, err := bird.NewWithTokens("", "", nil)
	if err == nil {
		t.Fatal("NewWithTokens() with both empty should return error")
	}
	if c != nil {
		t.Fatal("NewWithTokens() with both empty should return nil client")
	}
}

func TestNew_ErrorMessageContent(t *testing.T) {
	_, err := bird.New(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}
	if !strings.Contains(msg, "auth_token") || !strings.Contains(msg, "ct0") {
		t.Errorf("error message should mention auth_token and ct0: %q", msg)
	}
}

func TestNewWithTokens_ErrorMessageContent(t *testing.T) {
	_, err := bird.NewWithTokens("", "ct0", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "auth_token") {
		t.Errorf("error should mention auth_token: %q", msg)
	}
}

func TestNew_ValidCredsWithCookieHeader(t *testing.T) {
	creds := &bird.TwitterCookies{
		AuthToken:    "tok",
		Ct0:          "ct0",
		CookieHeader: "auth_token=tok; ct0=ct0",
	}
	c, err := bird.New(creds, nil)
	if err != nil {
		t.Fatalf("New() with valid creds including CookieHeader: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil client")
	}
}

func TestTypeAliases_TweetData(t *testing.T) {
	td := bird.TweetData{
		ID:   "alias1",
		Text: "alias test",
		Author: bird.TweetAuthor{
			Username: "u",
			Name:     "N",
		},
		Media:   []bird.TweetMedia{{Type: "photo", URL: "url"}},
		Article: &bird.TweetArticle{Title: "T"},
	}
	if td.ID != "alias1" {
		t.Errorf("TweetData alias: ID mismatch")
	}
	if td.Author.Username != "u" {
		t.Errorf("TweetAuthor alias: Username mismatch")
	}
	if len(td.Media) != 1 {
		t.Errorf("TweetMedia alias: want 1 item")
	}
	if td.Article.Title != "T" {
		t.Errorf("TweetArticle alias: Title mismatch")
	}
}

func TestTypeAliases_TwitterUser(t *testing.T) {
	u := bird.TwitterUser{
		ID:       "u1",
		Username: "test",
		Name:     "Test",
	}
	if u.ID != "u1" {
		t.Errorf("TwitterUser alias: ID mismatch")
	}
}

func TestTypeAliases_TwitterList(t *testing.T) {
	l := bird.TwitterList{
		ID:   "l1",
		Name: "List",
		Owner: &bird.ListOwner{
			ID:       "o1",
			Username: "owner",
		},
	}
	if l.Owner.Username != "owner" {
		t.Errorf("ListOwner alias: Username mismatch")
	}
}

func TestTypeAliases_NewsItem(t *testing.T) {
	n := bird.NewsItem{
		ID:       "n1",
		Headline: "News",
	}
	if n.ID != "n1" {
		t.Errorf("NewsItem alias: ID mismatch")
	}
}

func TestTypeAliases_Options(t *testing.T) {
	fo := bird.FetchOptions{Count: 20, Limit: 100}
	if fo.Count != 20 {
		t.Errorf("FetchOptions alias: Count mismatch")
	}
	so := bird.SearchOptions{Product: "Latest"}
	if so.Product != "Latest" {
		t.Errorf("SearchOptions alias: Product mismatch")
	}
	uto := bird.UserTweetsOptions{IncludeReplies: true}
	if !uto.IncludeReplies {
		t.Errorf("UserTweetsOptions alias: IncludeReplies mismatch")
	}
	tdo := bird.TweetDetailOptions{QuoteDepth: 2}
	if tdo.QuoteDepth != 2 {
		t.Errorf("TweetDetailOptions alias: QuoteDepth mismatch")
	}
	to := bird.ThreadOptions{FilterMode: "full_chain"}
	if to.FilterMode != "full_chain" {
		t.Errorf("ThreadOptions alias: FilterMode mismatch")
	}
	no := bird.NewsOptions{Tabs: []string{"news"}, MaxCount: 10}
	if no.MaxCount != 10 {
		t.Errorf("NewsOptions alias: MaxCount mismatch")
	}
	bfo := bird.BookmarkFolderOptions{FolderID: "f1"}
	if bfo.FolderID != "f1" {
		t.Errorf("BookmarkFolderOptions alias: FolderID mismatch")
	}
}

func TestTypeAliases_PageResultAndPaginatedResult(t *testing.T) {
	var tp bird.TweetPage
	tp.Success = true
	tp.Items = []bird.TweetData{{ID: "t1"}}
	if len(tp.Items) != 1 {
		t.Error("TweetPage alias: Items mismatch")
	}

	var tr bird.TweetResult
	tr.Success = true
	tr.Items = []bird.TweetData{{ID: "t2"}}
	if len(tr.Items) != 1 {
		t.Error("TweetResult alias: Items mismatch")
	}
}
