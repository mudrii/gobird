package cli

import (
	"os"
	"testing"

	"github.com/mudrii/gobird/internal/config"
)

// writeTempFile creates a temp file with the given bytes and returns its path.
func writeTempFile(t *testing.T, data []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "testmime-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return f.Name()
}

func TestDetectMime_JPEG(t *testing.T) {
	// JPEG magic bytes: FF D8 FF
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	path := writeTempFile(t, data)
	mime, err := detectMime(path)
	if err != nil {
		t.Fatalf("detectMime JPEG: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("detectMime JPEG = %q, want image/jpeg", mime)
	}
}

func TestDetectMime_PNG(t *testing.T) {
	// PNG magic bytes: 89 50 4E 47 0D 0A 1A 0A
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	path := writeTempFile(t, data)
	mime, err := detectMime(path)
	if err != nil {
		t.Fatalf("detectMime PNG: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("detectMime PNG = %q, want image/png", mime)
	}
}

func TestDetectMime_MP4(t *testing.T) {
	// http.DetectContentType recognises the "mp41" brand as video/mp4.
	data := make([]byte, 32)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x20
	copy(data[4:8], []byte("ftyp"))
	copy(data[8:12], []byte("mp41"))
	path := writeTempFile(t, data)
	mime, err := detectMime(path)
	if err != nil {
		t.Fatalf("detectMime MP4: %v", err)
	}
	if mime != "video/mp4" {
		t.Errorf("detectMime MP4 = %q, want video/mp4", mime)
	}
}

func TestDetectMime_Unknown(t *testing.T) {
	// Empty / unknown data falls back to application/octet-stream
	data := []byte{0x00, 0x00, 0x00, 0x00}
	path := writeTempFile(t, data)
	mime, err := detectMime(path)
	if err != nil {
		t.Fatalf("detectMime unknown: %v", err)
	}
	if mime != "application/octet-stream" {
		t.Errorf("detectMime unknown = %q, want application/octet-stream", mime)
	}
}

func TestResolveQuoteDepth_Default(t *testing.T) {
	// When quoteDepth flag is -1 (unset) and config has no setting, default is 1.
	old := globalFlags.quoteDepth
	globalFlags.quoteDepth = -1
	defer func() { globalFlags.quoteDepth = old }()

	cfg := &config.Config{}
	got := resolveQuoteDepth(cfg)
	if got != 1 {
		t.Errorf("resolveQuoteDepth default = %d, want 1", got)
	}
}

func TestResolveQuoteDepth_Explicit(t *testing.T) {
	// When quoteDepth flag is >= 0, that value is used.
	old := globalFlags.quoteDepth
	globalFlags.quoteDepth = 2
	defer func() { globalFlags.quoteDepth = old }()

	cfg := &config.Config{}
	got := resolveQuoteDepth(cfg)
	if got != 2 {
		t.Errorf("resolveQuoteDepth explicit = %d, want 2", got)
	}
}
