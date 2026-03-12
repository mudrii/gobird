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
