package output

import (
	"fmt"
	"strings"

	"github.com/mudrii/gobird/internal/types"
)

// FormatOptions controls how output is rendered.
type FormatOptions struct {
	Plain   bool
	NoColor bool
	NoEmoji bool
}

// FormatTweet formats a tweet as "@handle: text (likes: N, rts: N)".
func FormatTweet(t types.TweetData, opts FormatOptions) string {
	handle := "@" + t.Author.Username
	if !opts.NoColor && !opts.Plain {
		handle = "\x1b[1m" + handle + "\x1b[0m"
	}
	prefix := ""
	if !opts.NoEmoji && !opts.Plain {
		prefix = "🐦 "
	}
	text := strings.TrimSpace(t.Text)
	return fmt.Sprintf("%s%s: %s (likes: %d, rts: %d)",
		prefix, handle, text, t.LikeCount, t.RetweetCount)
}

// FormatUser formats a TwitterUser as "@handle (Name) — followers: N, following: N".
func FormatUser(u types.TwitterUser, opts FormatOptions) string {
	prefix := ""
	if !opts.NoEmoji && !opts.Plain {
		prefix = "👤 "
	}
	return fmt.Sprintf("%s@%s (%s) - followers: %d, following: %d",
		prefix, u.Username, u.Name, u.FollowersCount, u.FollowingCount)
}

// FormatList formats a TwitterList as "Name [ID] (members: N)".
func FormatList(l types.TwitterList, opts FormatOptions) string {
	owner := ""
	if l.Owner != nil {
		owner = fmt.Sprintf(", owner: @%s", l.Owner.Username)
	}
	prefix := ""
	if !opts.NoEmoji && !opts.Plain {
		prefix = "📋 "
	}
	return fmt.Sprintf("%s%s [%s] (members: %d%s)", prefix, l.Name, l.ID, l.MemberCount, owner)
}

// FormatNewsItem formats a NewsItem as "Headline (Category)".
func FormatNewsItem(n types.NewsItem, opts FormatOptions) string {
	prefix := ""
	if !opts.NoEmoji && !opts.Plain {
		prefix = "📰 "
	}
	if n.Category != "" {
		return fmt.Sprintf("%s%s (%s)", prefix, n.Headline, n.Category)
	}
	return prefix + n.Headline
}
