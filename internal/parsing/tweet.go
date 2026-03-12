package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// UnwrapTweetResult unwraps TweetWithVisibilityResults to get the inner tweet.
// Checks result.tweet truthy — no __typename check. Correction §unwrap.
func UnwrapTweetResult(raw *types.WireRawTweet) *types.WireRawTweet {
	if raw == nil {
		return nil
	}
	if raw.Tweet != nil {
		return raw.Tweet
	}
	return raw
}

// MapTweetResult converts a raw wire tweet into a normalized TweetData.
func MapTweetResult(raw *types.WireRawTweet) *types.TweetData {
	if raw == nil {
		return nil
	}
	raw = UnwrapTweetResult(raw)

	td := &types.TweetData{
		ID:             raw.RestID,
		IsBlueVerified: raw.IsBlueVerified, // top-level field, NOT inside legacy (correction #8)
	}

	if raw.Legacy != nil {
		td.Text = ExtractTweetText(raw)
		td.CreatedAt = raw.Legacy.CreatedAt
		td.ConversationID = raw.Legacy.ConversationIDStr
		td.InReplyToStatusID = raw.Legacy.InReplyToStatusIDStr
		td.ReplyCount = raw.Legacy.ReplyCount
		td.RetweetCount = raw.Legacy.RetweetCount
		td.LikeCount = raw.Legacy.FavoriteCount
		td.AuthorID = raw.Legacy.UserIDStr
		td.Media = ExtractMedia(raw.Legacy.ExtendedEntities)
	}

	if raw.Core != nil {
		if u := raw.Core.UserResults.Result; u != nil && u.Legacy != nil {
			td.Author = types.TweetAuthor{
				Username: u.Legacy.ScreenName,
				Name:     u.Legacy.Name,
			}
		}
	}

	if raw.QuotedResult != nil && raw.QuotedResult.Result != nil {
		quoted := MapTweetResult(raw.QuotedResult.Result)
		td.QuotedTweet = quoted
	}

	if raw.Article != nil && raw.Article.ArticleResults.Result != nil {
		ar := raw.Article.ArticleResults.Result
		td.Article = &types.TweetArticle{
			Title:       ar.Title,
			PreviewText: ar.PreviewText,
		}
	}

	return td
}

// ExtractTweetText returns the tweet text in priority order:
//  1. Article title + preview text (if article present)
//  2. Note tweet text (long-form)
//  3. Legacy full_text
func ExtractTweetText(raw *types.WireRawTweet) string {
	if raw == nil {
		return ""
	}
	// 1. Article.
	if raw.Article != nil && raw.Article.ArticleResults.Result != nil {
		ar := raw.Article.ArticleResults.Result
		if ar.Title != "" {
			return ar.Title
		}
	}
	// 2. Note tweet.
	if raw.NoteTweet != nil && raw.NoteTweet.NoteTweetResults.Result != nil {
		if t := raw.NoteTweet.NoteTweetResults.Result.Text; t != "" {
			return t
		}
	}
	// 3. Legacy full_text.
	if raw.Legacy != nil {
		return raw.Legacy.FullText
	}
	return ""
}
