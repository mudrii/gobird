package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func intPtr(n int) *int { return &n }

func TestExtractMediaPreviewURLForPhoto(t *testing.T) {
	// Correction #31: previewUrl set for ANY media with sizes.small, not just video.
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "photo",
				MediaURLHttps: "https://example.com/photo.jpg",
				Sizes: types.WireMediaSizes{
					Small: &types.WireMediaSize{W: 100, H: 100},
					Large: &types.WireMediaSize{W: 800, H: 600},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media, got %d", len(media))
	}
	if media[0].PreviewURL == "" {
		t.Error("PreviewURL should be set for photo with sizes.small")
	}
}

func TestExtractMediaDimensions(t *testing.T) {
	// Correction #32: large first, medium fallback.
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "photo",
				MediaURLHttps: "https://example.com/a.jpg",
				Sizes: types.WireMediaSizes{
					Large:  &types.WireMediaSize{W: 1200, H: 900},
					Medium: &types.WireMediaSize{W: 600, H: 450},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].Width != 1200 || media[0].Height != 900 {
		t.Errorf("want 1200x900, got %dx%d", media[0].Width, media[0].Height)
	}

	// No large → medium.
	entities.Media[0].Sizes.Large = nil
	media = parsing.ExtractMedia(entities)
	if media[0].Width != 600 || media[0].Height != 450 {
		t.Errorf("want 600x450, got %dx%d", media[0].Width, media[0].Height)
	}
}
