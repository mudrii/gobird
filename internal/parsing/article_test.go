package parsing_test

import (
	"encoding/json"
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func TestExtractArticleText_Nil(t *testing.T) {
	if got := parsing.ExtractArticleText(nil); got != "" {
		t.Errorf("want empty string for nil, got %q", got)
	}
}

func TestExtractArticleText_TitleOnly(t *testing.T) {
	ar := &types.WireArticleResult{Title: "My Title"}
	got := parsing.ExtractArticleText(ar)
	if got != "My Title" {
		t.Errorf("want %q, got %q", "My Title", got)
	}
}

func TestExtractArticleText_TitleAndPreview(t *testing.T) {
	ar := &types.WireArticleResult{
		Title:       "My Title",
		PreviewText: "Preview text here",
	}
	got := parsing.ExtractArticleText(ar)
	want := "My Title\n\nPreview text here"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractArticleText_PreviewOnlyWhenSameAsTitle(t *testing.T) {
	ar := &types.WireArticleResult{
		Title:       "Same",
		PreviewText: "Same",
	}
	got := parsing.ExtractArticleText(ar)
	if got != "Same" {
		t.Errorf("want %q, got %q", "Same", got)
	}
}

func TestExtractArticleText_WithBlocks(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{"type": "unstyled", "text": "First block", "entityRanges": []any{}},
			{"type": "unstyled", "text": "Second block", "entityRanges": []any{}},
		},
		"entityMap": map[string]any{},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{
		Title:        "My Title",
		ContentState: string(csJSON),
	}
	got := parsing.ExtractArticleText(ar)
	want := "My Title\n\nFirst block\n\nSecond block"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractArticleText_WithAtomic_Divider(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{"type": "unstyled", "text": "Before", "entityRanges": []any{}},
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": 0},
				},
			},
			{"type": "unstyled", "text": "After", "entityRanges": []any{}},
		},
		"entityMap": map[string]any{
			"0": map[string]any{
				"type": "DIVIDER",
				"data": map[string]any{},
			},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	want := "Before\n\n---\n\nAfter"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractArticleText_WithHeaderBlocks(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{"type": "header-one", "text": "Big Header", "entityRanges": []any{}},
			{"type": "header-two", "text": "Sub Header", "entityRanges": []any{}},
		},
		"entityMap": map[string]any{},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	want := "Big Header\n\nSub Header"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractArticleText_EmptyBlocks(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{"type": "unstyled", "text": "Real content", "entityRanges": []any{}},
			{"type": "unstyled", "text": "", "entityRanges": []any{}},
			{"type": "unstyled", "text": "More content", "entityRanges": []any{}},
		},
		"entityMap": map[string]any{},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	want := "Real content\n\nMore content"
	if got != want {
		t.Errorf("empty blocks should be skipped; want %q, got %q", want, got)
	}
}

func TestExtractArticleText_AtomicLink(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": 0},
				},
			},
		},
		"entityMap": map[string]any{
			"0": map[string]any{
				"type": "LINK",
				"data": map[string]any{"url": "https://example.com"},
			},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "https://example.com" {
		t.Errorf("want https://example.com, got %q", got)
	}
}

func TestExtractArticleText_InvalidContentState_FallsBackToTitle(t *testing.T) {
	ar := &types.WireArticleResult{
		Title:        "Fallback Title",
		ContentState: "not-valid-json",
	}
	got := parsing.ExtractArticleText(ar)
	if got != "Fallback Title" {
		t.Errorf("want %q on invalid JSON, got %q", "Fallback Title", got)
	}
}

func TestExtractArticleText_ContentStateSameAsTitle(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{"type": "unstyled", "text": "My Title", "entityRanges": []any{}},
		},
		"entityMap": map[string]any{},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{
		Title:        "My Title",
		ContentState: string(csJSON),
	}
	got := parsing.ExtractArticleText(ar)
	if got != "My Title" {
		t.Errorf("want %q when content equals title, got %q", "My Title", got)
	}
}

func TestExtractArticleText_MultipleAtomicBlocks(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": 0},
				},
			},
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": 1},
				},
			},
		},
		"entityMap": map[string]any{
			"0": map[string]any{"type": "DIVIDER", "data": map[string]any{}},
			"1": map[string]any{"type": "DIVIDER", "data": map[string]any{}},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	want := "---\n\n---"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractArticleText_InlineEntityLink(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "unstyled",
				"text": "https://example.com/article",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 27, "key": 0},
				},
			},
		},
		"entityMap": map[string]any{
			"0": map[string]any{
				"type": "LINK",
				"data": map[string]any{"url": "https://example.com/article"},
			},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "https://example.com/article" {
		t.Errorf("want URL from inline entity link, got %q", got)
	}
}

func TestRenderContentState_NilBlocks(t *testing.T) {
	cs := map[string]any{
		"blocks":    nil,
		"entityMap": map[string]any{},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "" {
		t.Errorf("content_state with nil blocks should yield empty string, got %q", got)
	}
}

func TestEntityKeyToString_Int(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": 5},
				},
			},
		},
		"entityMap": map[string]any{
			"5": map[string]any{"type": "DIVIDER", "data": map[string]any{}},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "---" {
		t.Errorf("numeric key 5 should look up entityMap[\"5\"], got %q", got)
	}
}

func TestEntityKeyToString_StringKey(t *testing.T) {
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": "mykey"},
				},
			},
		},
		"entityMap": map[string]any{
			"mykey": map[string]any{"type": "DIVIDER", "data": map[string]any{}},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "---" {
		t.Errorf("string key should look up entityMap[\"mykey\"], got %q", got)
	}
}

func TestEntityKeyToString_Float64(t *testing.T) {
	// JSON numbers decode as float64 when using interface{}. When the raw JSON
	// entity key is a floating-point integer (e.g. 3.0), entityKeyToString must
	// produce "3" so the entityMap lookup succeeds.
	cs := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "atomic",
				"text": " ",
				"entityRanges": []map[string]any{
					{"offset": 0, "length": 1, "key": float64(3)},
				},
			},
		},
		"entityMap": map[string]any{
			"3": map[string]any{"type": "DIVIDER", "data": map[string]any{}},
		},
	}
	csJSON, _ := json.Marshal(cs)
	ar := &types.WireArticleResult{ContentState: string(csJSON)}
	got := parsing.ExtractArticleText(ar)
	if got != "---" {
		t.Errorf("float64 key 3.0 should resolve to entityMap[\"3\"], got %q", got)
	}
}
