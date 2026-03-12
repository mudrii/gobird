package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// AddThreadMetadata annotates tweets in a thread with position metadata.
func AddThreadMetadata(tweets []types.TweetData, authorID string) []types.TweetWithMeta {
	if len(tweets) == 0 {
		return nil
	}
	out := make([]types.TweetWithMeta, len(tweets))
	for i, t := range tweets {
		wm := types.TweetWithMeta{TweetData: t}
		if len(tweets) == 1 {
			wm.ThreadPosition = "standalone"
		} else {
			switch i {
			case 0:
				wm.ThreadPosition = "root"
				wm.IsThread = true
			case len(tweets) - 1:
				wm.ThreadPosition = "end"
				wm.IsThread = true
			default:
				wm.ThreadPosition = "middle"
				wm.IsThread = true
			}
		}
		rootID := tweets[0].ID
		wm.ThreadRootID = &rootID
		out[i] = wm
	}
	return out
}

// FilterAuthorChain keeps tweets that form a self-reply chain from the author.
func FilterAuthorChain(tweets []types.TweetData, authorID string) []types.TweetData {
	var chain []types.TweetData
	for _, t := range tweets {
		if t.AuthorID == authorID {
			chain = append(chain, t)
		}
	}
	return chain
}

// FilterAuthorOnly keeps only tweets authored by authorID.
func FilterAuthorOnly(tweets []types.TweetData, authorID string) []types.TweetData {
	var out []types.TweetData
	for _, t := range tweets {
		if t.AuthorID == authorID {
			out = append(out, t)
		}
	}
	return out
}

// FilterFullChain returns all tweets in the thread regardless of author.
func FilterFullChain(tweets []types.TweetData) []types.TweetData {
	return tweets
}
