package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// ExtractMedia converts wire media entities to normalized TweetMedia slices.
// Correction #31: previewUrl set for ANY media with sizes.small (not just video/GIF).
// Correction #32: dimensions use sizes.large first, sizes.medium fallback.
func ExtractMedia(entities *types.WireMediaEntities) []types.TweetMedia {
	if entities == nil {
		return nil
	}
	out := make([]types.TweetMedia, 0, len(entities.Media))
	for _, m := range entities.Media {
		tm := types.TweetMedia{
			Type: m.Type,
			URL:  m.MediaURLHttps,
		}

		// Dimensions: large → medium fallback.
		if m.Sizes.Large != nil {
			tm.Width = m.Sizes.Large.W
			tm.Height = m.Sizes.Large.H
		} else if m.Sizes.Medium != nil {
			tm.Width = m.Sizes.Medium.W
			tm.Height = m.Sizes.Medium.H
		}

		// PreviewURL for ANY media with sizes.small (correction #31).
		if m.Sizes.Small != nil {
			tm.PreviewURL = m.MediaURLHttps
		}

		// Video URL from variants.
		if m.VideoInfo != nil {
			tm.DurationMs = m.VideoInfo.DurationMillis
			tm.VideoURL = bestVideoVariant(m.VideoInfo.Variants)
		}

		out = append(out, tm)
	}
	return out
}

// bestVideoVariant selects the highest-bitrate mp4 variant URL.
func bestVideoVariant(variants []types.WireVideoVariant) string {
	var best string
	var bestBitrate int
	for _, v := range variants {
		if v.ContentType != "video/mp4" {
			continue
		}
		br := 0
		if v.Bitrate != nil {
			br = *v.Bitrate
		}
		if br >= bestBitrate {
			bestBitrate = br
			best = v.URL
		}
	}
	return best
}
