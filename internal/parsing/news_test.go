package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
)

func TestParseNewsItemFromContent_Nil(t *testing.T) {
	if got := parsing.ParseNewsItemFromContent(nil); got != nil {
		t.Errorf("ParseNewsItemFromContent(nil): want nil, got %+v", got)
	}
}

func TestParseNewsItemFromContent_AllFields(t *testing.T) {
	postCount := 42.0
	content := map[string]any{
		"id":          "news1",
		"name":        "Top Headline",
		"category":    "tech",
		"time_ago":    "2h ago",
		"description": "Some news description",
		"url":         "https://example.com/news",
		"is_ai_news":  true,
		"tweet_count": postCount,
	}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("ParseNewsItemFromContent: want non-nil result")
	}
	if got.ID != "news1" {
		t.Errorf("ID: want %q, got %q", "news1", got.ID)
	}
	if got.Headline != "Top Headline" {
		t.Errorf("Headline: want %q, got %q", "Top Headline", got.Headline)
	}
	if got.Category != "tech" {
		t.Errorf("Category: want %q, got %q", "tech", got.Category)
	}
	if got.TimeAgo != "2h ago" {
		t.Errorf("TimeAgo: want %q, got %q", "2h ago", got.TimeAgo)
	}
	if got.Description != "Some news description" {
		t.Errorf("Description: want %q, got %q", "Some news description", got.Description)
	}
	if got.URL != "https://example.com/news" {
		t.Errorf("URL: want %q, got %q", "https://example.com/news", got.URL)
	}
	if !got.IsAiNews {
		t.Error("IsAiNews: want true")
	}
	if got.PostCount == nil || *got.PostCount != 42 {
		t.Errorf("PostCount: want 42, got %v", got.PostCount)
	}
}

func TestParseNewsItemFromContent_HeadlineFallback(t *testing.T) {
	// "headline" key used when "name" is absent.
	content := map[string]any{
		"headline": "Fallback Headline",
	}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("ParseNewsItemFromContent: want non-nil result")
	}
	if got.Headline != "Fallback Headline" {
		t.Errorf("Headline: want %q (from headline key), got %q", "Fallback Headline", got.Headline)
	}
}

func TestParseNewsItemFromContent_NamePriorityOverHeadline(t *testing.T) {
	// "name" takes priority over "headline".
	content := map[string]any{
		"name":     "Name Field",
		"headline": "Headline Field",
	}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("ParseNewsItemFromContent: want non-nil result")
	}
	if got.Headline != "Name Field" {
		t.Errorf("Headline: want %q (name takes priority), got %q", "Name Field", got.Headline)
	}
}

func TestParseNewsItemFromContent_EmptyMap(t *testing.T) {
	got := parsing.ParseNewsItemFromContent(map[string]any{})
	if got == nil {
		t.Fatal("ParseNewsItemFromContent: want non-nil for empty map")
	}
	if got.ID != "" || got.Headline != "" || got.PostCount != nil {
		t.Errorf("empty map should produce zero-value NewsItem, got %+v", got)
	}
}

func TestParseNewsItemFromContent_NoPostCount(t *testing.T) {
	content := map[string]any{"id": "n2"}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("ParseNewsItemFromContent: want non-nil result")
	}
	if got.PostCount != nil {
		t.Errorf("PostCount: want nil when absent, got %v", got.PostCount)
	}
}

func TestParseNewsItemFromContent_IsAiNewsFalse(t *testing.T) {
	content := map[string]any{
		"id":         "n3",
		"is_ai_news": false,
	}
	got := parsing.ParseNewsItemFromContent(content)
	if got == nil {
		t.Fatal("ParseNewsItemFromContent: want non-nil result")
	}
	if got.IsAiNews {
		t.Error("IsAiNews: want false")
	}
}
