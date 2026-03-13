// Package output provides tweet formatting and rendering utilities.
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

// FormatTweet formats a tweet as "@handle: text (replies: N, likes: N, rts: N)".
// When an article is attached its title is appended. Reply count is always shown.
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
	line := fmt.Sprintf("%s%s: %s (replies: %d, likes: %d, rts: %d)",
		prefix, handle, text, t.ReplyCount, t.LikeCount, t.RetweetCount)
	if t.Article != nil && t.Article.Title != "" {
		line += fmt.Sprintf(" [article: %s]", t.Article.Title)
	}
	return line
}

// FormatUser formats a TwitterUser as "@handle (Name) — followers: N, following: N [✓]".
// A verified badge is appended when IsBlueVerified is true.
func FormatUser(u types.TwitterUser, opts FormatOptions) string {
	prefix := ""
	if !opts.NoEmoji && !opts.Plain {
		prefix = "👤 "
	}
	verified := ""
	if u.IsBlueVerified {
		if !opts.NoEmoji && !opts.Plain {
			verified = " ✓"
		} else {
			verified = " [verified]"
		}
	}
	return fmt.Sprintf("%s@%s (%s) - followers: %s, following: %d%s",
		prefix, u.Username, u.Name, formatCount(u.FollowersCount), u.FollowingCount, verified)
}

// formatCount returns a human-readable representation of a large count.
// Values >= 1 000 000 are shown as "1.2M", >= 1 000 as "1.2K".
func formatCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
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

// FormatNewsItem formats a NewsItem as "Headline (Category) [url] [AI]".
// URL is shown when present. An AI badge is appended when IsAiNews is true.
func FormatNewsItem(n types.NewsItem, opts FormatOptions) string {
	prefix := ""
	if !opts.NoEmoji && !opts.Plain {
		prefix = "📰 "
	}
	line := prefix + n.Headline
	if n.Category != "" {
		line += fmt.Sprintf(" (%s)", n.Category)
	}
	if n.URL != "" {
		line += fmt.Sprintf(" [%s]", n.URL)
	}
	if n.IsAiNews {
		if !opts.NoEmoji && !opts.Plain {
			line += " 🤖"
		} else {
			line += " [AI]"
		}
	}
	return line
}
