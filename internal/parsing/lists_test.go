package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func TestMapList_Nil(t *testing.T) {
	if got := parsing.MapList(nil); got != nil {
		t.Errorf("MapList(nil): want nil, got %+v", got)
	}
}

func TestMapList_Basic(t *testing.T) {
	raw := &types.WireList{
		IDStr:           "list1",
		Name:            "My List",
		Description:     "A test list",
		MemberCount:     42,
		SubscriberCount: 7,
		Mode:            "public",
		CreatedAt:       "2024-01-01",
	}
	got := parsing.MapList(raw)
	if got == nil {
		t.Fatal("MapList: want non-nil result")
	}
	if got.ID != "list1" {
		t.Errorf("ID: want %q, got %q", "list1", got.ID)
	}
	if got.Name != "My List" {
		t.Errorf("Name: want %q, got %q", "My List", got.Name)
	}
	if got.Description != "A test list" {
		t.Errorf("Description: want %q, got %q", "A test list", got.Description)
	}
	if got.MemberCount != 42 {
		t.Errorf("MemberCount: want 42, got %d", got.MemberCount)
	}
	if got.SubscriberCount != 7 {
		t.Errorf("SubscriberCount: want 7, got %d", got.SubscriberCount)
	}
	if got.IsPrivate {
		t.Error("IsPrivate: want false for public list")
	}
	if got.CreatedAt != "2024-01-01" {
		t.Errorf("CreatedAt: want %q, got %q", "2024-01-01", got.CreatedAt)
	}
}

func TestMapList_Private(t *testing.T) {
	raw := &types.WireList{
		IDStr: "priv1",
		Mode:  "private",
	}
	got := parsing.MapList(raw)
	if got == nil {
		t.Fatal("MapList: want non-nil result")
	}
	if !got.IsPrivate {
		t.Error("IsPrivate: want true for private list")
	}
}

func TestMapList_WithOwner(t *testing.T) {
	raw := &types.WireList{
		IDStr: "list2",
		Name:  "Owned List",
		UserResults: types.WireUserResult{
			Result: &types.WireRawUser{
				RestID: "owner1",
				Legacy: &types.WireUserLegacy{
					ScreenName: "ownerhandle",
					Name:       "Owner Name",
				},
			},
		},
	}
	got := parsing.MapList(raw)
	if got == nil {
		t.Fatal("MapList: want non-nil")
	}
	if got.Owner == nil {
		t.Fatal("Owner: want non-nil")
	}
	if got.Owner.ID != "owner1" {
		t.Errorf("Owner.ID: want %q, got %q", "owner1", got.Owner.ID)
	}
	if got.Owner.Username != "ownerhandle" {
		t.Errorf("Owner.Username: want %q, got %q", "ownerhandle", got.Owner.Username)
	}
	if got.Owner.Name != "Owner Name" {
		t.Errorf("Owner.Name: want %q, got %q", "Owner Name", got.Owner.Name)
	}
}

func TestMapList_OwnerNilLegacy(t *testing.T) {
	raw := &types.WireList{
		IDStr: "list3",
		UserResults: types.WireUserResult{
			Result: &types.WireRawUser{
				RestID: "u2",
				Legacy: nil,
			},
		},
	}
	got := parsing.MapList(raw)
	if got == nil {
		t.Fatal("MapList: want non-nil")
	}
	if got.Owner != nil {
		t.Errorf("Owner: want nil when Legacy is nil, got %+v", got.Owner)
	}
}

func TestParseListsFromInstructions_ReturnsNil(t *testing.T) {
	result := parsing.ParseListsFromInstructions([]types.WireTimelineInstruction{
		{Entries: []types.WireEntry{*makeTweetEntry("1")}},
	})
	if result != nil {
		t.Errorf("ParseListsFromInstructions: want nil (placeholder), got %v", result)
	}
}
