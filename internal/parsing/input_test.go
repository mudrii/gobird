package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
)

func TestExtractTweetID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1234567890123456789", "1234567890123456789"},
		{"https://x.com/user/status/1234567890123456789", "1234567890123456789"},
		{"https://twitter.com/user/status/9876543210987654321", "9876543210987654321"},
		{"not-an-id", ""},
	}
	for _, tc := range cases {
		got := parsing.ExtractTweetID(tc.in)
		if got != tc.want {
			t.Errorf("ExtractTweetID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeHandle(t *testing.T) {
	if got := parsing.NormalizeHandle("@User"); got != "user" {
		t.Errorf("want user, got %q", got)
	}
	if got := parsing.NormalizeHandle("User"); got != "user" {
		t.Errorf("want user, got %q", got)
	}
}

func TestLooksLikeTweetInput_URL_Twitter(t *testing.T) {
	if !parsing.LooksLikeTweetInput("https://twitter.com/user/status/1234567890123456789") {
		t.Error("twitter.com URL should be a tweet input")
	}
}

func TestLooksLikeTweetInput_URL_X(t *testing.T) {
	if !parsing.LooksLikeTweetInput("https://x.com/user/status/9876543210987654") {
		t.Error("x.com URL should be a tweet input")
	}
}

func TestLooksLikeTweetInput_NumericID(t *testing.T) {
	if !parsing.LooksLikeTweetInput("1234567890123456789") {
		t.Error("19-digit numeric string should be a tweet input")
	}
}

func TestLooksLikeTweetInput_Handle(t *testing.T) {
	if parsing.LooksLikeTweetInput("@user") {
		t.Error("@user handle should not be a tweet input")
	}
}

func TestLooksLikeTweetInput_ShortNumber(t *testing.T) {
	if parsing.LooksLikeTweetInput("12345") {
		t.Error("short numeric string (< 15 digits) should not be a tweet input")
	}
}

func TestLooksLikeTweetInput_EmptyString(t *testing.T) {
	if parsing.LooksLikeTweetInput("") {
		t.Error("empty string should not be a tweet input")
	}
}

func TestExtractTweetID_FromURL_Twitter(t *testing.T) {
	got := parsing.ExtractTweetID("https://twitter.com/someuser/status/1234567890123456789")
	if got != "1234567890123456789" {
		t.Errorf("want 1234567890123456789, got %q", got)
	}
}

func TestExtractTweetID_FromURL_X(t *testing.T) {
	got := parsing.ExtractTweetID("https://x.com/anotheruser/status/9876543210987654321")
	if got != "9876543210987654321" {
		t.Errorf("want 9876543210987654321, got %q", got)
	}
}

func TestExtractTweetID_FromID(t *testing.T) {
	got := parsing.ExtractTweetID("1234567890123456789")
	if got != "1234567890123456789" {
		t.Errorf("want 1234567890123456789, got %q", got)
	}
}

func TestExtractTweetID_Invalid(t *testing.T) {
	cases := []string{"", "hello", "@user", "123", "https://twitter.com/user"}
	for _, c := range cases {
		if got := parsing.ExtractTweetID(c); got != "" {
			t.Errorf("ExtractTweetID(%q) = %q, want empty", c, got)
		}
	}
}

func TestNormalizeHandle_WithAt(t *testing.T) {
	got := parsing.NormalizeHandle("@user")
	if got != "user" {
		t.Errorf("want user, got %q", got)
	}
}

func TestNormalizeHandle_WithoutAt(t *testing.T) {
	got := parsing.NormalizeHandle("user")
	if got != "user" {
		t.Errorf("want user, got %q", got)
	}
}

func TestNormalizeHandle_Lowercases(t *testing.T) {
	got := parsing.NormalizeHandle("@UPPERCASE")
	if got != "uppercase" {
		t.Errorf("want uppercase, got %q", got)
	}
}

func TestExtractListID_FromURL(t *testing.T) {
	got := parsing.ExtractListID("https://twitter.com/i/lists/123456")
	if got != "123456" {
		t.Errorf("want 123456, got %q", got)
	}
}

func TestExtractListID_FromBareID(t *testing.T) {
	got := parsing.ExtractListID("123456")
	if got != "123456" {
		t.Errorf("want 123456, got %q", got)
	}
}

func TestExtractListID_Invalid(t *testing.T) {
	got := parsing.ExtractListID("not-a-list")
	if got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestLooksLikeTweetInput_EmbeddedURLRejected(t *testing.T) {
	if parsing.LooksLikeTweetInput("prefix https://x.com/user/status/1234567890123456789 suffix") {
		t.Error("embedded URL should not be treated as a tweet input")
	}
}

func TestExtractTweetID_EmbeddedURLRejected(t *testing.T) {
	if got := parsing.ExtractTweetID("prefix https://x.com/user/status/1234567890123456789 suffix"); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestExtractListID_XComURL(t *testing.T) {
	got := parsing.ExtractListID("https://x.com/i/lists/123456?s=20")
	if got != "123456" {
		t.Errorf("want 123456, got %q", got)
	}
}

func TestExtractTweetID_XComURL(t *testing.T) {
	got := parsing.ExtractTweetID("https://x.com/alice/status/1234567890123456789")
	if got != "1234567890123456789" {
		t.Errorf("want 1234567890123456789, got %q", got)
	}
}

func TestExtractTweetID_TwitterComURL(t *testing.T) {
	got := parsing.ExtractTweetID("https://twitter.com/bob/status/9876543210987654321")
	if got != "9876543210987654321" {
		t.Errorf("want 9876543210987654321, got %q", got)
	}
}

func TestExtractTweetID_WithQueryParams(t *testing.T) {
	// Query string after the ID is not part of the ID; the regex stops at the
	// digit run so ?s=20 is ignored and the ID is still extracted correctly.
	got := parsing.ExtractTweetID("https://twitter.com/user/status/1234567890123456789?s=20")
	if got != "1234567890123456789" {
		t.Errorf("want 1234567890123456789, got %q", got)
	}
}

func TestMentionsQueryFromUserOption_WithAt(t *testing.T) {
	got := parsing.MentionsQueryFromUserOption("@alice")
	if got != "to:alice" {
		t.Errorf("want to:alice, got %q", got)
	}
}

func TestMentionsQueryFromUserOption_WithoutAt(t *testing.T) {
	got := parsing.MentionsQueryFromUserOption("alice")
	if got != "to:alice" {
		t.Errorf("want to:alice, got %q", got)
	}
}
