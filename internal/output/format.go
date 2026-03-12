package output

import (
	"fmt"

	"github.com/mudrii/gobird/internal/types"
)

// FormatOptions controls how output is rendered.
type FormatOptions struct {
	Plain   bool
	NoColor bool
	NoEmoji bool
}

// FormatTweet formats a tweet as "@handle: text (likes: N, rts: N)".
func FormatTweet(t types.TweetData, _ FormatOptions) string {
	return fmt.Sprintf("@%s: %s (likes: %d, rts: %d)",
		t.Author.Username, t.Text, t.LikeCount, t.RetweetCount)
}

// FormatUser formats a TwitterUser as "@handle (Name) — followers: N, following: N".
func FormatUser(u types.TwitterUser, _ FormatOptions) string {
	return fmt.Sprintf("@%s (%s) — followers: %d, following: %d",
		u.Username, u.Name, u.FollowersCount, u.FollowingCount)
}

// FormatList formats a TwitterList as "Name [ID] (members: N)".
func FormatList(l types.TwitterList, _ FormatOptions) string {
	owner := ""
	if l.Owner != nil {
		owner = fmt.Sprintf(", owner: @%s", l.Owner.Username)
	}
	return fmt.Sprintf("%s [%s] (members: %d%s)", l.Name, l.ID, l.MemberCount, owner)
}

// FormatNewsItem formats a NewsItem as "Headline (Category)".
func FormatNewsItem(n types.NewsItem, _ FormatOptions) string {
	if n.Category != "" {
		return fmt.Sprintf("%s (%s)", n.Headline, n.Category)
	}
	return n.Headline
}
