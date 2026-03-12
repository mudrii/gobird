package client

import "testing"

func TestFallbackQueryIDsCount(t *testing.T) {
	// Correction #71: exactly 29 entries.
	if got := len(FallbackQueryIDs); got != 29 {
		t.Errorf("FallbackQueryIDs: want 29 entries, got %d", got)
	}
}

func TestDefaultNewsTabsNoTrending(t *testing.T) {
	// Correction #46: default tabs must NOT include "trending".
	for _, tab := range DefaultNewsTabs {
		if tab == "trending" {
			t.Error("DefaultNewsTabs must not include 'trending'")
		}
	}
}

func TestDefaultNewsTabsCount(t *testing.T) {
	if got := len(DefaultNewsTabs); got != 4 {
		t.Errorf("DefaultNewsTabs: want 4, got %d", got)
	}
}

func TestUserByScreenNameHardcoded(t *testing.T) {
	ids, ok := PerOperationFallbackIDs["UserByScreenName"]
	if !ok {
		t.Fatal("UserByScreenName not in PerOperationFallbackIDs")
	}
	want := []string{"xc8f1g7BYqr6VTzTbvNlGw", "qW5u-DAuXpMEG0zA1F7UGQ", "sLVLhk0bGj3MVFEKTdax1w"}
	if len(ids) != len(want) {
		t.Fatalf("UserByScreenName: want %d IDs, got %d", len(want), len(ids))
	}
	for i, id := range want {
		if ids[i] != id {
			t.Errorf("UserByScreenName[%d]: want %q, got %q", i, id, ids[i])
		}
	}
}
