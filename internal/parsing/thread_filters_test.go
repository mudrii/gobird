package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func strPtr(s string) *string { return &s }

func makeTweet(id, authorID, replyTo string) types.TweetData {
	td := types.TweetData{
		ID:       id,
		AuthorID: authorID,
	}
	if replyTo != "" {
		td.InReplyToStatusID = strPtr(replyTo)
	}
	return td
}

func TestAddThreadMetadata_SingleTweet(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("1", "alice", ""),
	}
	result := parsing.AddThreadMetadata(tweets, "alice")
	if len(result) != 1 {
		t.Fatalf("want 1 result, got %d", len(result))
	}
	if result[0].ThreadPosition != "standalone" {
		t.Errorf("single tweet should be standalone, got %q", result[0].ThreadPosition)
	}
	if result[0].IsThread {
		t.Error("single standalone tweet should not be IsThread")
	}
}

func TestAddThreadMetadata_Chain(t *testing.T) {
	// A (root) -> B (middle) -> C (end) by same author.
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "alice", "A"),
		makeTweet("C", "alice", "B"),
	}
	result := parsing.AddThreadMetadata(tweets, "alice")
	if len(result) != 3 {
		t.Fatalf("want 3 results, got %d", len(result))
	}

	byID := map[string]types.TweetWithMeta{}
	for _, r := range result {
		byID[r.ID] = r
	}

	if byID["A"].ThreadPosition != "root" {
		t.Errorf("A should be root, got %q", byID["A"].ThreadPosition)
	}
	if byID["B"].ThreadPosition != "middle" {
		t.Errorf("B should be middle, got %q", byID["B"].ThreadPosition)
	}
	if byID["C"].ThreadPosition != "end" {
		t.Errorf("C should be end, got %q", byID["C"].ThreadPosition)
	}
	if !byID["A"].IsThread {
		t.Error("root A should be IsThread")
	}
	if !byID["B"].IsThread {
		t.Error("middle B should be IsThread")
	}
	if !byID["C"].IsThread {
		t.Error("end C should be IsThread")
	}
}

func TestAddThreadMetadata_Reply_DifferentAuthor(t *testing.T) {
	// Tweet from alice, reply from bob — bob's reply is not part of alice's chain.
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "bob", "A"),
	}
	result := parsing.AddThreadMetadata(tweets, "alice")
	if len(result) != 2 {
		t.Fatalf("want 2 results, got %d", len(result))
	}
	byID := map[string]types.TweetWithMeta{}
	for _, r := range result {
		byID[r.ID] = r
	}
	// Alice's tweet A has no self-replies (bob replied, not alice), so standalone.
	if byID["A"].ThreadPosition != "standalone" {
		t.Errorf("A should be standalone when only other-author replies exist, got %q", byID["A"].ThreadPosition)
	}
}

func TestAddThreadMetadata_Empty(t *testing.T) {
	result := parsing.AddThreadMetadata(nil, "alice")
	if result != nil {
		t.Errorf("want nil for empty input, got %v", result)
	}
}

func TestFilterAuthorChain_ReturnsOnlyAuthor(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "alice", "A"),
		makeTweet("C", "bob", "B"),
	}
	result := parsing.FilterAuthorChain(tweets, "alice")
	for _, t2 := range result {
		if t2.AuthorID != "alice" {
			t.Errorf("FilterAuthorChain returned tweet by %q, want only alice", t2.AuthorID)
		}
	}
	if len(result) != 2 {
		t.Errorf("want 2 tweets from alice chain, got %d", len(result))
	}
}

func TestFilterAuthorChain_Empty(t *testing.T) {
	result := parsing.FilterAuthorChain(nil, "alice")
	if result != nil {
		t.Errorf("want nil for empty input, got %v", result)
	}
}

func TestFilterAuthorChain_NoMatch(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "bob", ""),
	}
	result := parsing.FilterAuthorChain(tweets, "alice")
	if result != nil {
		t.Errorf("want nil when no tweets from author, got %v", result)
	}
}

func TestFilterAuthorOnly_Basic(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("1", "alice", ""),
		makeTweet("2", "bob", ""),
		makeTweet("3", "alice", "1"),
	}
	result := parsing.FilterAuthorOnly(tweets, "alice")
	if len(result) != 2 {
		t.Fatalf("want 2 tweets by alice, got %d", len(result))
	}
	for _, t2 := range result {
		if t2.AuthorID != "alice" {
			t.Errorf("FilterAuthorOnly returned tweet by %q", t2.AuthorID)
		}
	}
}

func TestFilterAuthorOnly_Empty(t *testing.T) {
	result := parsing.FilterAuthorOnly(nil, "alice")
	if len(result) != 0 {
		t.Errorf("want 0 results for nil input, got %d", len(result))
	}
}

func TestFilterAuthorOnly_NoneMatch(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("1", "bob", ""),
	}
	result := parsing.FilterAuthorOnly(tweets, "alice")
	if len(result) != 0 {
		t.Errorf("want 0 results when no tweets by author, got %d", len(result))
	}
}

func TestFilterFullChain_ReturnsAll(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "bob", "A"),
		makeTweet("C", "alice", "B"),
	}
	result := parsing.FilterFullChain(tweets)
	if len(result) != len(tweets) {
		t.Errorf("FilterFullChain: want %d tweets (all), got %d", len(tweets), len(result))
	}
}

func TestFilterFullChain_Empty(t *testing.T) {
	result := parsing.FilterFullChain(nil)
	if len(result) != 0 {
		t.Errorf("FilterFullChain(nil): want 0 items, got %d", len(result))
	}
}

func TestFilterFullChain_Single(t *testing.T) {
	tweets := []types.TweetData{makeTweet("X", "user", "")}
	result := parsing.FilterFullChain(tweets)
	if len(result) != 1 {
		t.Errorf("FilterFullChain: want 1 item for single tweet, got %d", len(result))
	}
}
