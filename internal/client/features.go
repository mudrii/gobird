package client

import (
	"encoding/json"
	"os"
	"sync"
)

type featureOverrideConfig struct {
	Global map[string]any            `json:"global"`
	Sets   map[string]map[string]any `json:"sets"`
}

var (
	featureOverridesOnce sync.Once
	featureOverrides     featureOverrideConfig
)

func cloneFeatures(base map[string]any) map[string]any {
	cloned := make(map[string]any, len(base))
	for k, v := range base {
		cloned[k] = v
	}
	return cloned
}

func loadFeatureOverrides() featureOverrideConfig {
	featureOverridesOnce.Do(func() {
		var payload []byte
		if inline := os.Getenv("BIRD_FEATURES_JSON"); inline != "" {
			payload = []byte(inline)
		} else if path := os.Getenv("BIRD_FEATURES_PATH"); path != "" {
			if b, err := os.ReadFile(path); err == nil {
				payload = b
			}
		}
		if len(payload) == 0 {
			return
		}
		if err := json.Unmarshal(payload, &featureOverrides); err != nil {
			featureOverrides = featureOverrideConfig{}
		}
	})
	return featureOverrides
}

// buildArticleFeatures returns the feature map for article operations.
// This is the base feature set from which most other sets are derived.
func buildArticleFeatures() map[string]any {
	return applyFeatureOverrides("article", map[string]any{
		"rweb_video_screen_enabled":                                               true,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"responsive_web_profile_redirect_enabled":                                 true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      false,
		"responsive_web_grok_annotations_enabled":                                 false,
		"responsive_web_jetfuel_frame":                                            true,
		"post_ctas_fetch_enabled":                                                 true,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_grok_imagine_annotation_enabled":                          true,
		"responsive_web_grok_community_note_auto_translation_is_enabled":          false,
		"responsive_web_enhance_cards_enabled":                                    false,
	})
}

// buildTweetDetailFeatures returns features for TweetDetail.
// Spreads buildArticleFeatures() plus 3 additional flags.
func buildTweetDetailFeatures() map[string]any {
	f := cloneFeatures(buildArticleFeatures())
	f["responsive_web_twitter_article_plain_text_enabled"] = true
	f["responsive_web_twitter_article_seed_tweet_detail_enabled"] = true
	f["responsive_web_twitter_article_seed_tweet_summary_enabled"] = true
	return applyFeatureOverrides("tweetDetail", f)
}

// buildSearchFeatures returns features for SearchTimeline.
// Same as article plus rweb_video_timestamps_enabled.
func buildSearchFeatures() map[string]any {
	f := cloneFeatures(buildArticleFeatures())
	f["rweb_video_timestamps_enabled"] = true
	return applyFeatureOverrides("search", f)
}

// buildTweetCreateFeatures returns features for CreateTweet.
// Notable: responsive_web_profile_redirect_enabled=false.
func buildTweetCreateFeatures() map[string]any {
	f := cloneFeatures(buildArticleFeatures())
	f["responsive_web_profile_redirect_enabled"] = false
	return applyFeatureOverrides("tweetCreate", f)
}

// buildTimelineFeatures returns features for timeline operations
// (Bookmarks, HomeTimeline, etc.). Spreads buildSearchFeatures() plus extra flags.
func buildTimelineFeatures() map[string]any {
	f := cloneFeatures(buildSearchFeatures())
	f["blue_business_profile_image_shape_enabled"] = true
	f["responsive_web_text_conversations_enabled"] = false
	f["tweetypie_unmention_optimization_enabled"] = true
	f["vibe_api_enabled"] = true
	f["responsive_web_twitter_blue_verified_badge_is_enabled"] = true
	f["interactive_text_enabled"] = true
	f["longform_notetweets_richtext_consumption_enabled"] = true
	f["responsive_web_media_download_video_enabled"] = false
	return applyFeatureOverrides("timeline", f)
}

// buildBookmarksFeatures returns features for Bookmarks and BookmarkFolderTimeline.
func buildBookmarksFeatures() map[string]any {
	f := cloneFeatures(buildTimelineFeatures())
	f["graphql_timeline_v2_bookmark_timeline"] = true
	return applyFeatureOverrides("bookmarks", f)
}

// buildLikesFeatures returns features for Likes.
func buildLikesFeatures() map[string]any {
	return applyFeatureOverrides("likes", cloneFeatures(buildTimelineFeatures()))
}

// buildHomeTimelineFeatures returns features for HomeTimeline/HomeLatestTimeline.
func buildHomeTimelineFeatures() map[string]any {
	return applyFeatureOverrides("homeTimeline", cloneFeatures(buildTimelineFeatures()))
}

// buildListsFeatures returns the hardcoded feature map for list operations.
func buildListsFeatures() map[string]any {
	return applyFeatureOverrides("lists", map[string]any{
		"rweb_video_screen_enabled":                                               true,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"responsive_web_profile_redirect_enabled":                                 true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      false,
		"responsive_web_grok_annotations_enabled":                                 false,
		"responsive_web_jetfuel_frame":                                            true,
		"post_ctas_fetch_enabled":                                                 true,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_grok_imagine_annotation_enabled":                          true,
		"responsive_web_grok_community_note_auto_translation_is_enabled":          false,
		"responsive_web_enhance_cards_enabled":                                    false,
		"blue_business_profile_image_shape_enabled":                               false,
		"vibe_api_enabled":                                                        false,
		"interactive_text_enabled":                                                false,
		"tweetypie_unmention_optimization_enabled":                                true,
		"responsive_web_text_conversations_enabled":                               false,
	})
}

// buildUserTweetsFeatures returns the hardcoded feature map for UserTweets.
func buildUserTweetsFeatures() map[string]any {
	return applyFeatureOverrides("userTweets", map[string]any{
		"rweb_video_screen_enabled":                                               false,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"responsive_web_profile_redirect_enabled":                                 false,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      true,
		"responsive_web_grok_annotations_enabled":                                 false,
		"responsive_web_jetfuel_frame":                                            true,
		"post_ctas_fetch_enabled":                                                 true,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           true,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_grok_imagine_annotation_enabled":                          true,
		"responsive_web_grok_community_note_auto_translation_is_enabled":          false,
		"responsive_web_enhance_cards_enabled":                                    false,
	})
}

// buildFollowingFeatures returns the feature map for Following/Followers.
func buildFollowingFeatures() map[string]any {
	return applyFeatureOverrides("following", map[string]any{
		"rweb_video_screen_enabled":                                               true,
		"profile_label_improvements_pcf_label_in_post_enabled":                    false,
		"responsive_web_profile_redirect_enabled":                                 true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        true,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      false,
		"responsive_web_grok_annotations_enabled":                                 false,
		"responsive_web_jetfuel_frame":                                            false,
		"post_ctas_fetch_enabled":                                                 true,
		"responsive_web_grok_share_attachment_enabled":                            false,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        true,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        false,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            false,
		"responsive_web_grok_imagine_annotation_enabled":                          false,
		"responsive_web_grok_community_note_auto_translation_is_enabled":          false,
		"responsive_web_enhance_cards_enabled":                                    false,
	})
}

// buildExploreFeatures returns the feature map for GenericTimelineById (news/trending).
func buildExploreFeatures() map[string]any {
	f := cloneFeatures(buildSearchFeatures())
	f["responsive_web_grok_analyze_button_fetch_trends_enabled"] = true
	f["responsive_web_grok_analyze_post_followups_enabled"] = true
	f["responsive_web_grok_annotations_enabled"] = true
	f["responsive_web_grok_show_grok_translated_post"] = true
	f["responsive_web_grok_community_note_auto_translation_is_enabled"] = true
	return applyFeatureOverrides("explore", f)
}

// buildArticleFieldToggles returns the field toggles for article operations.
func buildArticleFieldToggles() map[string]any {
	return map[string]any{
		"withPayments":                false,
		"withAuxiliaryUserLabels":     false,
		"withArticleRichContentState": true,
		"withArticlePlainText":        true,
		"withGrokAnalyze":             false,
		"withDisallowedReplyControls": false,
	}
}

// buildUserTweetsFieldToggles returns fieldToggles for UserTweets (correction #13).
func buildUserTweetsFieldToggles() map[string]any {
	return map[string]any{"withArticlePlainText": false}
}

// buildUserByScreenNameFieldToggles returns fieldToggles for UserByScreenName.
func buildUserByScreenNameFieldToggles() map[string]any {
	return map[string]any{"withAuxiliaryUserLabels": false}
}

// applyFeatureOverrides merges runtime feature overrides on top of the base map.
// Order: base → global overrides → set-specific overrides.
func applyFeatureOverrides(setName string, base map[string]any) map[string]any {
	merged := cloneFeatures(base)
	overrides := loadFeatureOverrides()
	for k, v := range overrides.Global {
		merged[k] = v
	}
	if setOverrides, ok := overrides.Sets[setName]; ok {
		for k, v := range setOverrides {
			merged[k] = v
		}
	}
	return merged
}
