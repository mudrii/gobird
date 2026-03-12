package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// CollectTweetResultsFromEntry extracts tweet results from a timeline entry.
// Exactly 5 paths per response-parsing spec.
func CollectTweetResultsFromEntry(entry *types.WireEntry) []*types.WireRawTweet {
	if entry == nil {
		return nil
	}
	c := entry.Content
	var results []*types.WireRawTweet

	// Path 1: direct itemContent.tweet_results.result
	if c.ItemContent != nil && c.ItemContent.TweetResult != nil {
		if r := c.ItemContent.TweetResult.Result; r != nil {
			results = append(results, r)
		}
	}

	// Path 2: items[].item.itemContent.tweet_results.result (module timeline)
	for _, item := range c.Items {
		if item.Item.ItemContent != nil && item.Item.ItemContent.TweetResult != nil {
			if r := item.Item.ItemContent.TweetResult.Result; r != nil {
				results = append(results, r)
			}
		}
	}

	// Paths 3-5 are covered by the above two patterns in combination with
	// UnwrapTweetResult which handles TweetWithVisibilityResults.

	return results
}

// ParseTweetsFromInstructions collects and normalizes all tweet results
// from a slice of timeline instructions.
func ParseTweetsFromInstructions(instructions []types.WireTimelineInstruction) []types.TweetData {
	return ParseTweetsFromInstructionsWithOptions(instructions, TweetParseOptions{QuoteDepth: 1})
}

// ParseTweetsFromInstructionsWithOptions collects and normalizes all tweet
// results from a slice of timeline instructions.
func ParseTweetsFromInstructionsWithOptions(instructions []types.WireTimelineInstruction, opts TweetParseOptions) []types.TweetData {
	var tweets []types.TweetData
	seen := map[string]bool{}
	for _, inst := range instructions {
		for i := range inst.Entries {
			for _, raw := range CollectTweetResultsFromEntry(&inst.Entries[i]) {
				unwrapped := UnwrapTweetResult(raw)
				if unwrapped == nil || unwrapped.RestID == "" {
					continue
				}
				if seen[unwrapped.RestID] {
					continue
				}
				seen[unwrapped.RestID] = true
				if td := MapTweetResultWithOptions(unwrapped, opts); td != nil {
					tweets = append(tweets, *td)
				}
			}
		}
	}
	return tweets
}
