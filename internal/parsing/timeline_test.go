package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func makeTweetEntry(id string) *types.WireEntry {
	return &types.WireEntry{
		Content: types.WireContent{
			ItemContent: &types.WireItemContent{
				TweetResult: &types.WireTweetResult{
					Result: &types.WireRawTweet{
						RestID: id,
						Legacy: &types.WireTweetLegacy{FullText: "tweet " + id},
					},
				},
			},
		},
	}
}

func TestCollectTweetResultsFromEntry_Nil(t *testing.T) {
	got := parsing.CollectTweetResultsFromEntry(nil)
	if got != nil {
		t.Errorf("want nil for nil entry, got %v", got)
	}
}

func TestCollectTweetResultsFromEntry_DirectItem(t *testing.T) {
	entry := makeTweetEntry("111")
	results := parsing.CollectTweetResultsFromEntry(entry)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].RestID != "111" {
		t.Errorf("want RestID 111, got %q", results[0].RestID)
	}
}

func TestCollectTweetResultsFromEntry_Items(t *testing.T) {
	entry := &types.WireEntry{
		Content: types.WireContent{
			Items: []types.WireItem{
				{Item: struct {
					ItemContent *types.WireItemContent `json:"itemContent"`
				}{
					ItemContent: &types.WireItemContent{
						TweetResult: &types.WireTweetResult{
							Result: &types.WireRawTweet{RestID: "222", Legacy: &types.WireTweetLegacy{FullText: "tweet 222"}},
						},
					},
				}},
				{Item: struct {
					ItemContent *types.WireItemContent `json:"itemContent"`
				}{
					ItemContent: &types.WireItemContent{
						TweetResult: &types.WireTweetResult{
							Result: &types.WireRawTweet{RestID: "333", Legacy: &types.WireTweetLegacy{FullText: "tweet 333"}},
						},
					},
				}},
			},
		},
	}
	results := parsing.CollectTweetResultsFromEntry(entry)
	if len(results) != 2 {
		t.Fatalf("want 2 results from items, got %d", len(results))
	}
}

func TestCollectTweetResultsFromEntry_Empty(t *testing.T) {
	entry := &types.WireEntry{
		Content: types.WireContent{},
	}
	results := parsing.CollectTweetResultsFromEntry(entry)
	if len(results) != 0 {
		t.Errorf("want 0 results from empty content, got %d", len(results))
	}
}

func TestCollectTweetResultsFromEntry_NilTweetResult(t *testing.T) {
	entry := &types.WireEntry{
		Content: types.WireContent{
			ItemContent: &types.WireItemContent{
				TweetResult: nil,
			},
		},
	}
	results := parsing.CollectTweetResultsFromEntry(entry)
	if len(results) != 0 {
		t.Errorf("want 0 results for nil TweetResult, got %d", len(results))
	}
}

func TestCollectTweetResultsFromEntry_NilResultInside(t *testing.T) {
	entry := &types.WireEntry{
		Content: types.WireContent{
			ItemContent: &types.WireItemContent{
				TweetResult: &types.WireTweetResult{Result: nil},
			},
		},
	}
	results := parsing.CollectTweetResultsFromEntry(entry)
	if len(results) != 0 {
		t.Errorf("want 0 results for nil Result inside TweetResult, got %d", len(results))
	}
}

func TestParseTweetsFromInstructions_Basic(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				*makeTweetEntry("444"),
				*makeTweetEntry("555"),
			},
		},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 2 {
		t.Fatalf("want 2 tweets, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_Deduplication(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				*makeTweetEntry("666"),
				*makeTweetEntry("666"),
			},
		},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 1 {
		t.Errorf("want 1 tweet after deduplication, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_Empty(t *testing.T) {
	tweets := parsing.ParseTweetsFromInstructions(nil)
	if len(tweets) != 0 {
		t.Errorf("want 0 tweets for nil instructions, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_SkipsEmptyRestID(t *testing.T) {
	entry := &types.WireEntry{
		Content: types.WireContent{
			ItemContent: &types.WireItemContent{
				TweetResult: &types.WireTweetResult{
					Result: &types.WireRawTweet{
						RestID: "",
						Legacy: &types.WireTweetLegacy{FullText: "no ID"},
					},
				},
			},
		},
	}
	instructions := []types.WireTimelineInstruction{
		{Entries: []types.WireEntry{*entry}},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 0 {
		t.Errorf("want 0 tweets when RestID is empty, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_DeduplicationAcrossInstructions(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{Entries: []types.WireEntry{*makeTweetEntry("777")}},
		{Entries: []types.WireEntry{*makeTweetEntry("777")}},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 1 {
		t.Errorf("want 1 tweet after cross-instruction deduplication, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_TimelinePinEntry(t *testing.T) {
	// A pinned entry has entryId starting with "tweet-" but is the same tweet as
	// another entry — it must be deduplicated so only one copy appears.
	pinnedEntry := makeTweetEntry("pinned1")
	pinnedEntry.EntryID = "tweet-pinned1"
	regularEntry := makeTweetEntry("pinned1")
	regularEntry.EntryID = "tweet-pinned1-again"

	instructions := []types.WireTimelineInstruction{
		{Entries: []types.WireEntry{*pinnedEntry, *regularEntry}},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 1 {
		t.Errorf("pinned tweet should be deduplicated: want 1, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_TerminateEntrySkipped(t *testing.T) {
	// A TimelineTerminateTimeline instruction has no entries with tweet content;
	// an entry with an empty RestID is skipped.
	terminateEntry := &types.WireEntry{
		EntryID: "terminate-timeline",
		Content: types.WireContent{
			EntryType: "TimelineTerminateTimeline",
		},
	}
	instructions := []types.WireTimelineInstruction{
		{
			Type:    "TimelineTerminateTimeline",
			Entries: []types.WireEntry{*terminateEntry},
		},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 0 {
		t.Errorf("TimelineTerminateTimeline should produce 0 tweets, got %d", len(tweets))
	}
}

func TestParseTweetsFromInstructions_UnwrapsVisibilityResults(t *testing.T) {
	inner := &types.WireRawTweet{
		RestID: "888",
		Legacy: &types.WireTweetLegacy{FullText: "inner tweet"},
	}
	outer := &types.WireRawTweet{
		TypeName: "TweetWithVisibilityResults",
		Tweet:    inner,
	}
	entry := &types.WireEntry{
		Content: types.WireContent{
			ItemContent: &types.WireItemContent{
				TweetResult: &types.WireTweetResult{Result: outer},
			},
		},
	}
	instructions := []types.WireTimelineInstruction{
		{Entries: []types.WireEntry{*entry}},
	}
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) != 1 {
		t.Fatalf("want 1 unwrapped tweet, got %d", len(tweets))
	}
	if tweets[0].ID != "888" {
		t.Errorf("want ID 888 from inner tweet, got %q", tweets[0].ID)
	}
}
