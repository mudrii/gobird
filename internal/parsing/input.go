package parsing

import (
	"regexp"
	"strings"
)

var (
	tweetURLRe    = regexp.MustCompile(`https?://(?:twitter\.com|x\.com)/\w+/status/(\d+)`)
	tweetIDRe     = regexp.MustCompile(`^\d{15,20}$`)
	listURLRe     = regexp.MustCompile(`(?:twitter\.com|x\.com)/i/lists/(\d+)`)
	listIDRe      = regexp.MustCompile(`^\d{5,20}$`)
	handleRe      = regexp.MustCompile(`^@?(\w{1,50})$`)
)

// LooksLikeTweetInput returns true if s could be a tweet URL or numeric ID.
func LooksLikeTweetInput(s string) bool {
	return tweetURLRe.MatchString(s) || tweetIDRe.MatchString(s)
}

// ExtractTweetID extracts a tweet ID from a URL or returns the bare numeric ID.
func ExtractTweetID(s string) string {
	if m := tweetURLRe.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if tweetIDRe.MatchString(s) {
		return s
	}
	return ""
}

// ExtractListID extracts a list ID from a URL or bare numeric ID.
func ExtractListID(s string) string {
	if m := listURLRe.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if listIDRe.MatchString(s) {
		return s
	}
	return ""
}

// NormalizeHandle strips a leading @ and lowercases the handle.
func NormalizeHandle(s string) string {
	s = strings.TrimPrefix(s, "@")
	return strings.ToLower(s)
}

// MentionsQueryFromUserOption builds a search query for mentions of a user.
func MentionsQueryFromUserOption(handle string) string {
	h := NormalizeHandle(handle)
	return "@" + h
}
