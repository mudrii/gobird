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

// ---------------------------------------------------------------------------
// Additional thread filter edge cases
// ---------------------------------------------------------------------------

func TestAddThreadMetadata_TwoTweets_SameAuthor(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "alice", "A"),
	}
	result := parsing.AddThreadMetadata(tweets, "alice")
	if len(result) != 2 {
		t.Fatalf("want 2 results, got %d", len(result))
	}
	byID := map[string]types.TweetWithMeta{}
	for _, r := range result {
		byID[r.ID] = r
	}
	if byID["A"].ThreadPosition != "root" {
		t.Errorf("A should be root, got %q", byID["A"].ThreadPosition)
	}
	if byID["B"].ThreadPosition != "end" {
		t.Errorf("B should be end, got %q", byID["B"].ThreadPosition)
	}
}

func TestAddThreadMetadata_ConversationIDUsedForRootID(t *testing.T) {
	tw := types.TweetData{
		ID:             "1",
		AuthorID:       "alice",
		ConversationID: "conv-root",
	}
	result := parsing.AddThreadMetadata([]types.TweetData{tw}, "alice")
	if len(result) != 1 {
		t.Fatalf("want 1 result, got %d", len(result))
	}
	if result[0].ThreadRootID == nil || *result[0].ThreadRootID != "conv-root" {
		t.Errorf("want ThreadRootID=conv-root, got %v", result[0].ThreadRootID)
	}
}

func TestAddThreadMetadata_FallsBackToIDWhenNoConversationID(t *testing.T) {
	tw := types.TweetData{
		ID:       "1",
		AuthorID: "alice",
	}
	result := parsing.AddThreadMetadata([]types.TweetData{tw}, "alice")
	if len(result) != 1 {
		t.Fatalf("want 1 result, got %d", len(result))
	}
	if result[0].ThreadRootID == nil || *result[0].ThreadRootID != "1" {
		t.Errorf("want ThreadRootID=1 (fallback to ID), got %v", result[0].ThreadRootID)
	}
}

func TestAddThreadMetadata_MixedAuthors_MultipleReplies(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "alice", "A"),
		makeTweet("C", "bob", "A"),
		makeTweet("D", "alice", "B"),
	}
	result := parsing.AddThreadMetadata(tweets, "alice")
	byID := map[string]types.TweetWithMeta{}
	for _, r := range result {
		byID[r.ID] = r
	}
	if byID["A"].ThreadPosition != "root" {
		t.Errorf("A should be root, got %q", byID["A"].ThreadPosition)
	}
	if !byID["A"].HasSelfReplies {
		t.Error("A should have self-replies (B)")
	}
	if byID["B"].ThreadPosition != "middle" {
		t.Errorf("B should be middle, got %q", byID["B"].ThreadPosition)
	}
	if byID["D"].ThreadPosition != "end" {
		t.Errorf("D should be end, got %q", byID["D"].ThreadPosition)
	}
}

func TestFilterAuthorChain_BranchingThread(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "alice", ""),
		makeTweet("B", "alice", "A"),
		makeTweet("C", "alice", "A"),
	}
	result := parsing.FilterAuthorChain(tweets, "alice")
	if len(result) != 3 {
		t.Errorf("want 3 tweets (A plus both branches), got %d", len(result))
	}
}

func TestFilterAuthorChain_BreaksOnDifferentAuthorParent(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("A", "bob", ""),
		makeTweet("B", "alice", "A"),
		makeTweet("C", "alice", "B"),
	}
	result := parsing.FilterAuthorChain(tweets, "alice")
	if len(result) != 2 {
		t.Errorf("want 2 tweets (B and C; chain stops at bob's A), got %d", len(result))
	}
}

func TestFilterAuthorOnly_PreservesOrder(t *testing.T) {
	tweets := []types.TweetData{
		makeTweet("1", "alice", ""),
		makeTweet("2", "bob", ""),
		makeTweet("3", "alice", ""),
		makeTweet("4", "bob", ""),
		makeTweet("5", "alice", ""),
	}
	result := parsing.FilterAuthorOnly(tweets, "alice")
	if len(result) != 3 {
		t.Fatalf("want 3 tweets, got %d", len(result))
	}
	if result[0].ID != "1" || result[1].ID != "3" || result[2].ID != "5" {
		t.Errorf("order not preserved: got IDs %s, %s, %s", result[0].ID, result[1].ID, result[2].ID)
	}
}

func TestAddThreadMetadata_EmptyReplyToStatusID(t *testing.T) {
	emptyReply := ""
	tw := types.TweetData{
		ID:                "1",
		AuthorID:          "alice",
		InReplyToStatusID: &emptyReply,
	}
	result := parsing.AddThreadMetadata([]types.TweetData{tw}, "alice")
	if len(result) != 1 {
		t.Fatalf("want 1 result, got %d", len(result))
	}
	if result[0].ThreadPosition != "standalone" {
		t.Errorf("tweet with empty InReplyToStatusID should be standalone, got %q", result[0].ThreadPosition)
	}
}
