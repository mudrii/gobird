package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func intPtr(n int) *int { return &n }

func TestExtractMedia_Empty(t *testing.T) {
	got := parsing.ExtractMedia(nil)
	if got != nil {
		t.Errorf("want nil for nil entities, got %v", got)
	}
}

func TestExtractMedia_EmptySlice(t *testing.T) {
	entities := &types.WireMediaEntities{Media: []types.WireMedia{}}
	got := parsing.ExtractMedia(entities)
	if len(got) != 0 {
		t.Errorf("want empty slice, got %d items", len(got))
	}
}

func TestExtractMedia_Photo(t *testing.T) {
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
	m := media[0]
	if m.Type != "photo" {
		t.Errorf("want type photo, got %q", m.Type)
	}
	if m.URL != "https://example.com/photo.jpg" {
		t.Errorf("want URL https://example.com/photo.jpg, got %q", m.URL)
	}
	if m.Width != 800 || m.Height != 600 {
		t.Errorf("want 800x600, got %dx%d", m.Width, m.Height)
	}
}

func TestExtractMedia_PreviewURL_HasSmallSuffix(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "photo",
				MediaURLHttps: "https://example.com/photo.jpg",
				Sizes: types.WireMediaSizes{
					Small: &types.WireMediaSize{W: 100, H: 100},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media, got %d", len(media))
	}
	want := "https://example.com/photo.jpg:small"
	if media[0].PreviewURL != want {
		t.Errorf("want PreviewURL %q, got %q", want, media[0].PreviewURL)
	}
}

func TestExtractMedia_NoSmallSize_NoPreviewURL(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "photo",
				MediaURLHttps: "https://example.com/photo.jpg",
				Sizes: types.WireMediaSizes{
					Large: &types.WireMediaSize{W: 800, H: 600},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].PreviewURL != "" {
		t.Errorf("want empty PreviewURL when no Small size, got %q", media[0].PreviewURL)
	}
}

func TestExtractMedia_Video(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				Sizes: types.WireMediaSizes{
					Small: &types.WireMediaSize{W: 320, H: 180},
					Large: &types.WireMediaSize{W: 1280, H: 720},
				},
				VideoInfo: &types.WireVideoInfo{
					DurationMillis: intPtr(30000),
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/low.mp4", Bitrate: intPtr(500000)},
						{ContentType: "video/mp4", URL: "https://example.com/high.mp4", Bitrate: intPtr(2000000)},
						{ContentType: "application/x-mpegURL", URL: "https://example.com/stream.m3u8"},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media, got %d", len(media))
	}
	m := media[0]
	if m.Type != "video" {
		t.Errorf("want type video, got %q", m.Type)
	}
	if m.VideoURL != "https://example.com/high.mp4" {
		t.Errorf("want highest bitrate mp4 URL, got %q", m.VideoURL)
	}
	if m.DurationMs == nil || *m.DurationMs != 30000 {
		t.Error("want DurationMs = 30000")
	}
}

func TestExtractMedia_AnimatedGif(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "animated_gif",
				MediaURLHttps: "https://example.com/gif.mp4",
				Sizes: types.WireMediaSizes{
					Small: &types.WireMediaSize{W: 200, H: 200},
				},
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/gif.mp4", Bitrate: intPtr(0)},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 1 {
		t.Fatalf("want 1 media, got %d", len(media))
	}
	if media[0].Type != "animated_gif" {
		t.Errorf("want type animated_gif, got %q", media[0].Type)
	}
	if media[0].VideoURL != "https://example.com/gif.mp4" {
		t.Errorf("want gif video URL, got %q", media[0].VideoURL)
	}
}

func TestExtractMedia_VideoSelectsHighestBitrate(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/v1.mp4", Bitrate: intPtr(1000000)},
						{ContentType: "video/mp4", URL: "https://example.com/v2.mp4", Bitrate: intPtr(3000000)},
						{ContentType: "video/mp4", URL: "https://example.com/v3.mp4", Bitrate: intPtr(2000000)},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].VideoURL != "https://example.com/v2.mp4" {
		t.Errorf("want highest bitrate mp4, got %q", media[0].VideoURL)
	}
}

func TestExtractMedia_VideoSkipsNonMp4(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "application/x-mpegURL", URL: "https://example.com/stream.m3u8"},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].VideoURL != "" {
		t.Errorf("want empty VideoURL for non-mp4 only variants, got %q", media[0].VideoURL)
	}
}

func TestExtractMedia_DimensionsFallback(t *testing.T) {
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
		t.Errorf("want 1200x900 from Large, got %dx%d", media[0].Width, media[0].Height)
	}

	entities.Media[0].Sizes.Large = nil
	media = parsing.ExtractMedia(entities)
	if media[0].Width != 600 || media[0].Height != 450 {
		t.Errorf("want 600x450 from Medium fallback, got %dx%d", media[0].Width, media[0].Height)
	}
}

func TestExtractMedia_MultipleItems(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{Type: "photo", MediaURLHttps: "https://example.com/1.jpg"},
			{Type: "photo", MediaURLHttps: "https://example.com/2.jpg"},
			{Type: "photo", MediaURLHttps: "https://example.com/3.jpg"},
		},
	}
	media := parsing.ExtractMedia(entities)
	if len(media) != 3 {
		t.Errorf("want 3 media items, got %d", len(media))
	}
}

func TestBestVideoVariant_EqualBitratesKeepsFirst(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/first.mp4", Bitrate: intPtr(2000000)},
						{ContentType: "video/mp4", URL: "https://example.com/second.mp4", Bitrate: intPtr(2000000)},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].VideoURL != "https://example.com/first.mp4" {
		t.Errorf("want first URL when bitrates are equal, got %q", media[0].VideoURL)
	}
}

func TestBestVideoVariant_AllNilBitrates(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/first.mp4"},
						{ContentType: "video/mp4", URL: "https://example.com/second.mp4"},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].VideoURL != "https://example.com/first.mp4" {
		t.Errorf("want first URL when all bitrates are nil, got %q", media[0].VideoURL)
	}
}

func TestBestVideoVariant_NilVsNonNilBitrate(t *testing.T) {
	entities := &types.WireMediaEntities{
		Media: []types.WireMedia{
			{
				Type:          "video",
				MediaURLHttps: "https://example.com/thumb.jpg",
				VideoInfo: &types.WireVideoInfo{
					Variants: []types.WireVideoVariant{
						{ContentType: "video/mp4", URL: "https://example.com/nil-bitrate.mp4"},
						{ContentType: "video/mp4", URL: "https://example.com/has-bitrate.mp4", Bitrate: intPtr(1000)},
					},
				},
			},
		},
	}
	media := parsing.ExtractMedia(entities)
	if media[0].VideoURL != "https://example.com/has-bitrate.mp4" {
		t.Errorf("want variant with non-nil bitrate, got %q", media[0].VideoURL)
	}
}
