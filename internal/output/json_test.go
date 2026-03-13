package output_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
)

func TestToJSON_ValidStruct(t *testing.T) {
	v := map[string]string{"key": "value"}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Errorf("ToJSON produced invalid JSON: %s", b)
	}
	if !strings.Contains(string(b), "\"key\"") {
		t.Errorf("ToJSON output missing key: %s", b)
	}
	if !strings.Contains(string(b), "\"value\"") {
		t.Errorf("ToJSON output missing value: %s", b)
	}
}

func TestToJSON_Indented(t *testing.T) {
	v := map[string]int{"count": 3}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "\n") || !strings.Contains(s, "  ") {
		t.Errorf("ToJSON output does not appear indented: %s", s)
	}
}

func TestToJSON_WithOmitempty(t *testing.T) {
	type row struct {
		Name  string `json:"name"`
		Score int    `json:"score,omitempty"`
	}
	v := row{Name: "alice"}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "score") {
		t.Errorf("omitempty zero field should be absent: %s", s)
	}
}

func TestPrintJSON_WritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	v := map[string]bool{"ok": true}
	if err := output.PrintJSON(&buf, v); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "\"ok\"") {
		t.Errorf("PrintJSON output missing expected key: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("PrintJSON output should end with newline: %q", got)
	}
}

func TestPrintJSON_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintJSON(&buf, []string{}); err != nil {
		t.Fatalf("PrintJSON empty slice: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "[]" {
		t.Errorf("expected '[]', got %q", got)
	}
}

func TestPrintJSON_NilValue(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintJSON(&buf, nil); err != nil {
		t.Fatalf("PrintJSON nil: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "null" {
		t.Errorf("expected 'null', got %q", got)
	}
}

// errorWriter is an io.Writer that always returns an error.
type errorWriter struct{}

func (errorWriter) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("simulated write error")
}

func TestPrintJSON_PropagatesWriteError(t *testing.T) {
	err := output.PrintJSON(errorWriter{}, map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected error when writer fails, got nil")
	}
	if !strings.Contains(err.Error(), "simulated write error") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestToJSON_NilInput(t *testing.T) {
	b, err := output.ToJSON(nil)
	if err != nil {
		t.Fatalf("ToJSON(nil): %v", err)
	}
	if string(b) != "null" {
		t.Errorf("expected 'null', got %q", string(b))
	}
}

// ---------------------------------------------------------------------------
// Additional JSON output edge cases
// ---------------------------------------------------------------------------

func TestToJSON_TweetDataWithRawField(t *testing.T) {
	tw := types.TweetData{
		ID:   "raw1",
		Text: "with raw",
		Author: types.TweetAuthor{
			Username: "user",
			Name:     "User",
		},
		Raw: map[string]any{"original": "data", "nested": map[string]any{"key": "val"}},
	}
	b, err := output.ToJSON(tw)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Errorf("invalid JSON output: %s", b)
	}
	s := string(b)
	if !strings.Contains(s, `"_raw"`) {
		t.Errorf("Raw field should appear as _raw: %s", s)
	}
	if !strings.Contains(s, `"original"`) {
		t.Errorf("Raw data content should be present: %s", s)
	}
}

func TestToJSON_TweetDataWithoutRaw(t *testing.T) {
	tw := types.TweetData{
		ID:   "noraw1",
		Text: "no raw",
		Author: types.TweetAuthor{
			Username: "user",
			Name:     "User",
		},
	}
	b, err := output.ToJSON(tw)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "_raw") {
		t.Errorf("_raw should be omitted when Raw is nil: %s", s)
	}
}

func TestToJSON_SpecialCharacters(t *testing.T) {
	v := map[string]string{
		"text":    "hello <world> & \"friends\"",
		"unicode": "\u00e9\u00e0\u00fc",
	}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Errorf("invalid JSON with special chars: %s", b)
	}
}

func TestToJSON_NestedStruct(t *testing.T) {
	tw := types.TweetData{
		ID:   "nested1",
		Text: "outer",
		Author: types.TweetAuthor{
			Username: "outer_user",
			Name:     "Outer",
		},
		QuotedTweet: &types.TweetData{
			ID:   "nested2",
			Text: "inner",
			Author: types.TweetAuthor{
				Username: "inner_user",
				Name:     "Inner",
			},
		},
		Media: []types.TweetMedia{
			{Type: "photo", URL: "https://example.com/img.jpg"},
		},
		Article: &types.TweetArticle{Title: "Art Title"},
	}
	b, err := output.ToJSON(tw)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Errorf("invalid JSON: %s", b)
	}
	var roundtrip types.TweetData
	if err := json.Unmarshal(b, &roundtrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if roundtrip.QuotedTweet == nil || roundtrip.QuotedTweet.ID != "nested2" {
		t.Error("nested quoted tweet should survive JSON round-trip")
	}
}

func TestToJSON_LargeSlice(t *testing.T) {
	items := make([]types.TweetData, 100)
	for i := range items {
		items[i] = types.TweetData{
			ID:     fmt.Sprintf("tweet-%d", i),
			Text:   fmt.Sprintf("text %d", i),
			Author: types.TweetAuthor{Username: "u", Name: "U"},
		}
	}
	b, err := output.ToJSON(items)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Error("large slice should produce valid JSON")
	}
	var parsed []types.TweetData
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(parsed) != 100 {
		t.Errorf("want 100 items, got %d", len(parsed))
	}
}

func TestPrintJSON_TweetData(t *testing.T) {
	tw := types.TweetData{
		ID:   "pj1",
		Text: "print json test",
		Author: types.TweetAuthor{
			Username: "pjuser",
			Name:     "PJ User",
		},
	}
	var buf bytes.Buffer
	if err := output.PrintJSON(&buf, tw); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	got := buf.String()
	if !json.Valid([]byte(strings.TrimSpace(got))) {
		t.Errorf("PrintJSON output is not valid JSON: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("PrintJSON output should end with newline")
	}
}

func TestToJSON_EmptyMap(t *testing.T) {
	b, err := output.ToJSON(map[string]any{})
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if strings.TrimSpace(string(b)) != "{}" {
		t.Errorf("empty map should produce {}, got %q", string(b))
	}
}

func TestToJSON_TwitterUserWithRaw(t *testing.T) {
	u := types.TwitterUser{
		ID:       "u1",
		Username: "rawuser",
		Name:     "Raw User",
		Raw:      map[string]any{"extra": "field"},
	}
	b, err := output.ToJSON(u)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"_raw"`) {
		t.Errorf("TwitterUser Raw field should appear as _raw: %s", s)
	}
}

func TestToJSON_NewsItemWithRaw(t *testing.T) {
	n := types.NewsItem{
		ID:       "n1",
		Headline: "Raw News",
		Raw:      []any{"raw", "data"},
	}
	b, err := output.ToJSON(n)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"_raw"`) {
		t.Errorf("NewsItem Raw field should appear as _raw: %s", s)
	}
}
