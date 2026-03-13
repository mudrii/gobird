package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func TestMapUser_Nil(t *testing.T) {
	if got := parsing.MapUser(nil); got != nil {
		t.Errorf("want nil for nil input, got %+v", got)
	}
}

func TestMapUser_Unavailable(t *testing.T) {
	raw := &types.WireRawUser{TypeName: "UserUnavailable"}
	if got := parsing.MapUser(raw); got != nil {
		t.Errorf("want nil for UserUnavailable, got %+v", got)
	}
}

func TestMapUser_Basic(t *testing.T) {
	raw := &types.WireRawUser{
		TypeName:       "User",
		RestID:         "42",
		IsBlueVerified: false,
		Legacy: &types.WireUserLegacy{
			ScreenName:     "testuser",
			Name:           "Test User",
			Description:    "A test account",
			FollowersCount: 100,
			FriendsCount:   50,
		},
	}
	u := parsing.MapUser(raw)
	if u == nil {
		t.Fatal("want non-nil user")
	}
	if u.ID != "42" {
		t.Errorf("want ID 42, got %q", u.ID)
	}
	if u.Username != "testuser" {
		t.Errorf("want Username testuser, got %q", u.Username)
	}
	if u.Name != "Test User" {
		t.Errorf("want Name 'Test User', got %q", u.Name)
	}
	if u.Description != "A test account" {
		t.Errorf("want Description 'A test account', got %q", u.Description)
	}
	if u.FollowersCount != 100 {
		t.Errorf("want FollowersCount 100, got %d", u.FollowersCount)
	}
	if u.FollowingCount != 50 {
		t.Errorf("want FollowingCount 50, got %d", u.FollowingCount)
	}
}

func TestMapUser_IsBlueVerified_TopLevel(t *testing.T) {
	raw := &types.WireRawUser{
		TypeName:       "User",
		RestID:         "99",
		IsBlueVerified: true,
		Legacy:         &types.WireUserLegacy{ScreenName: "bluecheck"},
	}
	u := parsing.MapUser(raw)
	if u == nil {
		t.Fatal("want non-nil user")
	}
	if !u.IsBlueVerified {
		t.Error("IsBlueVerified should be true from top-level field")
	}
}

func TestMapUser_NoLegacy(t *testing.T) {
	raw := &types.WireRawUser{
		TypeName: "User",
		RestID:   "77",
	}
	u := parsing.MapUser(raw)
	if u == nil {
		t.Fatal("want non-nil user even without Legacy")
	}
	if u.ID != "77" {
		t.Errorf("want ID 77, got %q", u.ID)
	}
	if u.Username != "" {
		t.Errorf("want empty Username without Legacy, got %q", u.Username)
	}
}

func TestMapUser_WithVisibilityResults(t *testing.T) {
	inner := &types.WireRawUser{
		TypeName:       "User",
		RestID:         "55",
		IsBlueVerified: true,
		Legacy: &types.WireUserLegacy{
			ScreenName: "inneruser",
			Name:       "Inner User",
		},
	}
	outer := &types.WireRawUser{
		TypeName: "UserWithVisibilityResults",
		User:     inner,
	}
	u := parsing.MapUser(outer)
	if u == nil {
		t.Fatal("want non-nil user from visibility wrapper")
	}
	if u.ID != "55" {
		t.Errorf("want inner user ID 55, got %q", u.ID)
	}
	if u.Username != "inneruser" {
		t.Errorf("want inneruser, got %q", u.Username)
	}
	if !u.IsBlueVerified {
		t.Error("IsBlueVerified should be true from inner user")
	}
}

func TestMapUser_ProfileImageURL(t *testing.T) {
	raw := &types.WireRawUser{
		TypeName: "User",
		RestID:   "11",
		Legacy: &types.WireUserLegacy{
			ScreenName:           "imguser",
			ProfileImageURLHTTPS: "https://pbs.twimg.com/profile_images/img.jpg",
		},
	}
	u := parsing.MapUser(raw)
	if u.ProfileImageURL != "https://pbs.twimg.com/profile_images/img.jpg" {
		t.Errorf("want profile image URL, got %q", u.ProfileImageURL)
	}
}

func TestParseUsersFromInstructions_Basic(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{
					Content: types.WireContent{
						ItemContent: &types.WireItemContent{
							UserResult: &types.WireUserResult{
								Result: &types.WireRawUser{
									TypeName: "User",
									RestID:   "101",
									Legacy:   &types.WireUserLegacy{ScreenName: "alice"},
								},
							},
						},
					},
				},
				{
					Content: types.WireContent{
						ItemContent: &types.WireItemContent{
							UserResult: &types.WireUserResult{
								Result: &types.WireRawUser{
									TypeName: "User",
									RestID:   "102",
									Legacy:   &types.WireUserLegacy{ScreenName: "bob"},
								},
							},
						},
					},
				},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 2 {
		t.Fatalf("want 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("want alice, got %q", users[0].Username)
	}
	if users[1].Username != "bob" {
		t.Errorf("want bob, got %q", users[1].Username)
	}
}

func TestParseUsersFromInstructions_Deduplication(t *testing.T) {
	sameUser := &types.WireRawUser{
		TypeName: "User",
		RestID:   "200",
		Legacy:   &types.WireUserLegacy{ScreenName: "dupuser"},
	}
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{ItemContent: &types.WireItemContent{UserResult: &types.WireUserResult{Result: sameUser}}}},
				{Content: types.WireContent{ItemContent: &types.WireItemContent{UserResult: &types.WireUserResult{Result: sameUser}}}},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 1 {
		t.Errorf("want 1 user after deduplication, got %d", len(users))
	}
}

func TestParseUsersFromInstructions_Empty(t *testing.T) {
	users := parsing.ParseUsersFromInstructions(nil)
	if len(users) != 0 {
		t.Errorf("want 0 users for nil instructions, got %d", len(users))
	}
}

func TestParseUsersFromInstructions_SkipsNilUserResult(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{
					Content: types.WireContent{
						ItemContent: &types.WireItemContent{
							UserResult: nil,
						},
					},
				},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 0 {
		t.Errorf("want 0 users for nil UserResult, got %d", len(users))
	}
}

func TestParseUsersFromInstructions_SkipsUnavailableUser(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{
					Content: types.WireContent{
						ItemContent: &types.WireItemContent{
							UserResult: &types.WireUserResult{
								Result: &types.WireRawUser{TypeName: "UserUnavailable"},
							},
						},
					},
				},
			},
		},
	}
	users := parsing.ParseUsersFromInstructions(instructions)
	if len(users) != 0 {
		t.Errorf("want 0 users for unavailable user, got %d", len(users))
	}
}
