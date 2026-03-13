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
