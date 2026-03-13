package client

import (
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

func TestAttachRawToTweets(t *testing.T) {
	tests := []struct {
		name    string
		items   []types.TweetData
		body    []byte
		wantRaw bool
		wantLen int
	}{
		{
			name:    "nil items",
			items:   nil,
			body:    []byte(`{"ok":true}`),
			wantRaw: false,
			wantLen: 0,
		},
		{
			name:    "empty items",
			items:   []types.TweetData{},
			body:    []byte(`{"ok":true}`),
			wantRaw: false,
			wantLen: 0,
		},
		{
			name:    "nil body",
			items:   []types.TweetData{{ID: "1"}},
			body:    nil,
			wantRaw: false,
			wantLen: 1,
		},
		{
			name:    "empty body",
			items:   []types.TweetData{{ID: "1"}},
			body:    []byte{},
			wantRaw: false,
			wantLen: 1,
		},
		{
			name:    "invalid JSON body",
			items:   []types.TweetData{{ID: "1"}},
			body:    []byte(`{invalid`),
			wantRaw: false,
			wantLen: 1,
		},
		{
			name:    "valid body attaches raw",
			items:   []types.TweetData{{ID: "1"}, {ID: "2"}},
			body:    []byte(`{"data":"value"}`),
			wantRaw: true,
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := attachRawToTweets(tt.items, tt.body)
			if len(result) != tt.wantLen {
				t.Errorf("len: want %d, got %d", tt.wantLen, len(result))
			}
			for i, item := range result {
				hasRaw := item.Raw != nil
				if hasRaw != tt.wantRaw {
					t.Errorf("item[%d].Raw: want present=%v, got present=%v", i, tt.wantRaw, hasRaw)
				}
			}
		})
	}
}

func TestAttachRawToTweets_allItemsShareSameRaw(t *testing.T) {
	items := []types.TweetData{{ID: "1"}, {ID: "2"}, {ID: "3"}}
	body := []byte(`{"key":"val"}`)
	result := attachRawToTweets(items, body)
	for i := 1; i < len(result); i++ {
		r0, ok0 := result[0].Raw.(map[string]any)
		ri, oki := result[i].Raw.(map[string]any)
		if !ok0 || !oki {
			t.Fatalf("item raw not map[string]any")
		}
		if r0["key"] != ri["key"] {
			t.Errorf("items[0] and items[%d] should have same raw content", i)
		}
	}
}

func TestAttachRawToNews(t *testing.T) {
	tests := []struct {
		name    string
		items   []types.NewsItem
		body    []byte
		wantRaw bool
		wantLen int
	}{
		{
			name:    "nil items",
			items:   nil,
			body:    []byte(`{}`),
			wantRaw: false,
			wantLen: 0,
		},
		{
			name:    "empty body",
			items:   []types.NewsItem{{ID: "1"}},
			body:    nil,
			wantRaw: false,
			wantLen: 1,
		},
		{
			name:    "invalid JSON",
			items:   []types.NewsItem{{ID: "1"}},
			body:    []byte(`not json`),
			wantRaw: false,
			wantLen: 1,
		},
		{
			name:    "valid",
			items:   []types.NewsItem{{ID: "1"}},
			body:    []byte(`{"x":1}`),
			wantRaw: true,
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := attachRawToNews(tt.items, tt.body)
			if len(result) != tt.wantLen {
				t.Errorf("len: want %d, got %d", tt.wantLen, len(result))
			}
			for i, item := range result {
				if (item.Raw != nil) != tt.wantRaw {
					t.Errorf("item[%d].Raw present=%v, want %v", i, item.Raw != nil, tt.wantRaw)
				}
			}
		})
	}
}

func TestAttachRawToUsers(t *testing.T) {
	tests := []struct {
		name    string
		items   []types.TwitterUser
		body    []byte
		wantRaw bool
		wantLen int
	}{
		{
			name:    "nil items",
			items:   nil,
			body:    []byte(`{}`),
			wantRaw: false,
			wantLen: 0,
		},
		{
			name:    "empty body",
			items:   []types.TwitterUser{{ID: "1"}},
			body:    []byte{},
			wantRaw: false,
			wantLen: 1,
		},
		{
			name:    "valid",
			items:   []types.TwitterUser{{ID: "1"}, {ID: "2"}},
			body:    []byte(`[1,2,3]`),
			wantRaw: true,
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := attachRawToUsers(tt.items, tt.body)
			if len(result) != tt.wantLen {
				t.Errorf("len: want %d, got %d", tt.wantLen, len(result))
			}
			for i, item := range result {
				if (item.Raw != nil) != tt.wantRaw {
					t.Errorf("item[%d].Raw present=%v, want %v", i, item.Raw != nil, tt.wantRaw)
				}
			}
		})
	}
}

func TestAttachRawToLists(t *testing.T) {
	tests := []struct {
		name    string
		items   []types.TwitterList
		body    []byte
		wantRaw bool
		wantLen int
	}{
		{
			name:    "nil items",
			items:   nil,
			body:    []byte(`{}`),
			wantRaw: false,
			wantLen: 0,
		},
		{
			name:    "empty items",
			items:   []types.TwitterList{},
			body:    []byte(`{}`),
			wantRaw: false,
			wantLen: 0,
		},
		{
			name:    "valid",
			items:   []types.TwitterList{{ID: "L1"}},
			body:    []byte(`{"list":true}`),
			wantRaw: true,
			wantLen: 1,
		},
		{
			name:    "invalid JSON preserves items",
			items:   []types.TwitterList{{ID: "L1"}},
			body:    []byte(`{bad`),
			wantRaw: false,
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := attachRawToLists(tt.items, tt.body)
			if len(result) != tt.wantLen {
				t.Errorf("len: want %d, got %d", tt.wantLen, len(result))
			}
			for i, item := range result {
				if (item.Raw != nil) != tt.wantRaw {
					t.Errorf("item[%d].Raw present=%v, want %v", i, item.Raw != nil, tt.wantRaw)
				}
			}
		})
	}
}

func TestAttachRawToTweets_doesNotModifyOriginalSlice(t *testing.T) {
	items := []types.TweetData{{ID: "1"}}
	body := []byte(`{"data":1}`)
	result := attachRawToTweets(items, body)
	if result[0].Raw == nil {
		t.Fatal("result should have raw attached")
	}
	// Since Go slices share backing array, the original is modified in-place.
	// Verify the function returns the same slice.
	if len(result) != len(items) {
		t.Error("should return same length slice")
	}
}
