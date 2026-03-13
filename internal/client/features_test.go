package client

import (
	"os"
	"sync"
	"testing"
)

func TestBuildSearchFeatures(t *testing.T) {
	f := buildSearchFeatures()
	if f == nil {
		t.Fatal("buildSearchFeatures returned nil")
	}
	if _, ok := f["rweb_video_timestamps_enabled"]; !ok {
		t.Error("buildSearchFeatures: missing rweb_video_timestamps_enabled")
	}
	if v, ok := f["rweb_video_timestamps_enabled"]; ok {
		if v != true {
			t.Errorf("rweb_video_timestamps_enabled: want true, got %v", v)
		}
	}
	if _, ok := f["responsive_web_graphql_timeline_navigation_enabled"]; !ok {
		t.Error("buildSearchFeatures: missing responsive_web_graphql_timeline_navigation_enabled")
	}
	if v := f["verified_phone_label_enabled"]; v != false {
		t.Errorf("verified_phone_label_enabled: want false, got %v", v)
	}
}

func TestBuildTweetDetailFeatures(t *testing.T) {
	f := buildTweetDetailFeatures()
	if f == nil {
		t.Fatal("buildTweetDetailFeatures returned nil")
	}
	keys := []string{
		"responsive_web_twitter_article_plain_text_enabled",
		"responsive_web_twitter_article_seed_tweet_detail_enabled",
		"responsive_web_twitter_article_seed_tweet_summary_enabled",
	}
	for _, k := range keys {
		if _, ok := f[k]; !ok {
			t.Errorf("buildTweetDetailFeatures: missing %q", k)
		}
	}
	// articles_rest_api_enabled is added by buildFetchTweetDetailFeatures, not buildTweetDetailFeatures.
	// Verify base article features are present.
	if _, ok := f["articles_preview_enabled"]; !ok {
		t.Error("buildTweetDetailFeatures: missing articles_preview_enabled")
	}
}

func TestBuildFetchTweetDetailFeatures_ContainsArticlesRestAPI(t *testing.T) {
	f := buildFetchTweetDetailFeatures()
	if v, ok := f["articles_rest_api_enabled"]; !ok || v != true {
		t.Errorf("buildFetchTweetDetailFeatures: articles_rest_api_enabled: want true, got %v (ok=%v)", v, ok)
	}
	if v, ok := f["rweb_video_timestamps_enabled"]; !ok || v != true {
		t.Errorf("buildFetchTweetDetailFeatures: rweb_video_timestamps_enabled: want true, got %v (ok=%v)", v, ok)
	}
}

func TestBuildTimelineFeatures(t *testing.T) {
	f := buildTimelineFeatures()
	if f == nil {
		t.Fatal("buildTimelineFeatures returned nil")
	}
	timelineKeys := []string{
		"blue_business_profile_image_shape_enabled",
		"tweetypie_unmention_optimization_enabled",
		"vibe_api_enabled",
		"responsive_web_twitter_blue_verified_badge_is_enabled",
		"interactive_text_enabled",
		"longform_notetweets_richtext_consumption_enabled",
	}
	for _, k := range timelineKeys {
		if _, ok := f[k]; !ok {
			t.Errorf("buildTimelineFeatures: missing %q", k)
		}
	}
}

func TestApplyFeatureOverrides_JSON(t *testing.T) {
	// Reset the once so the env var takes effect.
	featureOverridesOnce = sync.Once{}
	featureOverrides = featureOverrideConfig{}

	t.Setenv("BIRD_FEATURES_JSON", `{"global":{"rweb_video_screen_enabled":false},"sets":{}}`)
	defer func() {
		featureOverridesOnce = sync.Once{}
		featureOverrides = featureOverrideConfig{}
		os.Unsetenv("BIRD_FEATURES_JSON")
	}()

	base := map[string]any{
		"rweb_video_screen_enabled": true,
		"some_other_flag":           true,
	}
	result := applyFeatureOverrides("test", base)
	if v, ok := result["rweb_video_screen_enabled"]; !ok || v != false {
		t.Errorf("applyFeatureOverrides: rweb_video_screen_enabled: want false, got %v (ok=%v)", v, ok)
	}
	// Unmodified key should be unchanged.
	if v, ok := result["some_other_flag"]; !ok || v != true {
		t.Errorf("applyFeatureOverrides: some_other_flag: want true, got %v (ok=%v)", v, ok)
	}
}

func TestCloneFeatures(t *testing.T) {
	original := map[string]any{
		"key_a": true,
		"key_b": false,
	}
	cloned := cloneFeatures(original)

	// Modifying clone must not affect original.
	cloned["key_a"] = false
	cloned["new_key"] = "hello"

	if original["key_a"] != true {
		t.Error("cloneFeatures: modifying clone changed original key_a")
	}
	if _, ok := original["new_key"]; ok {
		t.Error("cloneFeatures: modifying clone added new_key to original")
	}
}

func TestLoadFeatureOverrides_InvalidJSON(t *testing.T) {
	featureOverridesOnce = sync.Once{}
	featureOverrides = featureOverrideConfig{}

	t.Setenv("BIRD_FEATURES_JSON", `{not valid json!!!`)
	defer func() {
		featureOverridesOnce = sync.Once{}
		featureOverrides = featureOverrideConfig{}
		os.Unsetenv("BIRD_FEATURES_JSON")
	}()

	// Should not panic; invalid JSON is silently ignored.
	cfg := loadFeatureOverrides()
	if cfg.Global != nil {
		t.Errorf("loadFeatureOverrides: invalid JSON should yield nil Global, got %v", cfg.Global)
	}
}

func TestBuildHomeTimelineFeatures(t *testing.T) {
	f := buildHomeTimelineFeatures()
	if f == nil {
		t.Fatal("buildHomeTimelineFeatures returned nil")
	}
	// HomeTimeline is derived from buildTimelineFeatures which adds these keys.
	timelineKeys := []string{
		"blue_business_profile_image_shape_enabled",
		"vibe_api_enabled",
		"responsive_web_twitter_blue_verified_badge_is_enabled",
		"interactive_text_enabled",
		"tweetypie_unmention_optimization_enabled",
		"longform_notetweets_richtext_consumption_enabled",
	}
	for _, k := range timelineKeys {
		if _, ok := f[k]; !ok {
			t.Errorf("buildHomeTimelineFeatures: missing key %q", k)
		}
	}
	// Must include base article features.
	if _, ok := f["articles_preview_enabled"]; !ok {
		t.Error("buildHomeTimelineFeatures: missing articles_preview_enabled")
	}
}

func TestBuildFollowingFeatures(t *testing.T) {
	f := buildFollowingFeatures()
	if f == nil {
		t.Fatal("buildFollowingFeatures returned nil")
	}
	// Spot-check distinctive values from the hardcoded map.
	checks := map[string]any{
		"rweb_video_screen_enabled":                          true,
		"profile_label_improvements_pcf_label_in_post_enabled": false,
		"premium_content_api_read_enabled":                   true,
		"tweet_awards_web_tipping_enabled":                   true,
		"responsive_web_grok_image_annotation_enabled":       false,
	}
	for k, want := range checks {
		got, ok := f[k]
		if !ok {
			t.Errorf("buildFollowingFeatures: missing key %q", k)
			continue
		}
		if got != want {
			t.Errorf("buildFollowingFeatures: %q want %v, got %v", k, want, got)
		}
	}
}

func TestBuildUserTweetsFeatures(t *testing.T) {
	f := buildUserTweetsFeatures()
	if f == nil {
		t.Fatal("buildUserTweetsFeatures returned nil")
	}
	// Correction #29: withV2Timeline must NOT be present.
	if _, ok := f["withV2Timeline"]; ok {
		t.Error("buildUserTweetsFeatures: must not contain withV2Timeline (correction #29)")
	}
	// Should include standard graphql timeline nav flag.
	if _, ok := f["responsive_web_graphql_timeline_navigation_enabled"]; !ok {
		t.Error("buildUserTweetsFeatures: missing responsive_web_graphql_timeline_navigation_enabled")
	}
}

func TestFeatureOverride_PathFile(t *testing.T) {
	featureOverridesOnce = sync.Once{}
	featureOverrides = featureOverrideConfig{}
	defer func() {
		featureOverridesOnce = sync.Once{}
		featureOverrides = featureOverrideConfig{}
	}()

	// Write a temp file with an override.
	tmp, err := os.CreateTemp("", "bird_features_*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	_, _ = tmp.WriteString(`{"global":{"verified_phone_label_enabled":true},"sets":{}}`)
	tmp.Close()

	t.Setenv("BIRD_FEATURES_PATH", tmp.Name())
	defer os.Unsetenv("BIRD_FEATURES_PATH")

	base := map[string]any{"verified_phone_label_enabled": false}
	result := applyFeatureOverrides("test", base)
	if v, ok := result["verified_phone_label_enabled"]; !ok || v != true {
		t.Errorf("BIRD_FEATURES_PATH override: want true, got %v (ok=%v)", v, ok)
	}
}
