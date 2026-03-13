package parsing

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/mudrii/gobird/internal/types"
)

// ExtractArticleText extracts the text content of an article tweet.
// Priority: content_state (Draft.js) → plain_text fallback.
func ExtractArticleText(result *types.WireArticleResult) string {
	if result == nil {
		return ""
	}
	if result.ContentState != "" {
		if text := renderContentState(result.ContentState); text != "" {
			if result.Title != "" && text != result.Title {
				return result.Title + "\n\n" + text
			}
			return text
		}
	}
	switch {
	case result.Title != "" && result.PreviewText != "" && result.PreviewText != result.Title:
		return result.Title + "\n\n" + result.PreviewText
	case result.PreviewText != "":
		return result.PreviewText
	default:
		return result.Title
	}
}

// draftJSContentState is the shape of a Draft.js content state JSON blob.
type draftJSContentState struct {
	Blocks    []draftBlock           `json:"blocks"`
	EntityMap map[string]draftEntity `json:"entityMap"`
}

type draftBlock struct {
	Type         string             `json:"type"`
	Text         string             `json:"text"`
	EntityRanges []draftEntityRange `json:"entityRanges"`
}

type draftEntityRange struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
	// Key can be int (number) or string in the wire format.
	Key json.RawMessage `json:"key"`
}

type draftEntity struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// renderContentState parses a Draft.js JSON content state and returns plain text.
func renderContentState(contentStateJSON string) string {
	var cs draftJSContentState
	if err := json.Unmarshal([]byte(contentStateJSON), &cs); err != nil {
		return ""
	}
	var blocks []string
	for _, block := range cs.Blocks {
		var rendered string
		if block.Type == "atomic" {
			rendered = renderAtomicBlock(block, cs.EntityMap)
		} else {
			rendered = renderBlockText(block, cs.EntityMap)
		}
		if rendered == "" {
			continue
		}
		blocks = append(blocks, rendered)
	}
	return strings.Join(blocks, "\n\n")
}

// renderAtomicBlock renders an atomic Draft.js block using its entity map entry.
func renderAtomicBlock(block draftBlock, entityMap map[string]draftEntity) string {
	if len(block.EntityRanges) == 0 {
		return block.Text
	}
	keyStr := entityKeyToString(block.EntityRanges[0].Key)
	entity, ok := entityMap[keyStr]
	if !ok {
		return block.Text
	}
	switch entity.Type {
	case "LINK":
		if url, ok := entity.Data["url"].(string); ok {
			return url
		}
	case "IMAGE", "MEDIA":
		if src, ok := entity.Data["src"].(string); ok {
			return src
		}
	case "DIVIDER":
		return "---"
	case "TWEET":
		if url, ok := entity.Data["url"].(string); ok {
			return url
		}
		if id, ok := entity.Data["id"].(string); ok {
			return "https://twitter.com/i/status/" + id
		}
	case "MARKDOWN":
		if src, ok := entity.Data["src"].(string); ok {
			return src
		}
		if content, ok := entity.Data["content"].(string); ok {
			return content
		}
	}
	return block.Text
}

// renderBlockText renders a non-atomic block's text with inline entity ranges applied.
func renderBlockText(block draftBlock, entityMap map[string]draftEntity) string {
	if len(block.EntityRanges) == 0 {
		return block.Text
	}
	runes := []rune(block.Text)
	// For each entity range that is a LINK, we could annotate the text,
	// but for plain-text output we just return the raw text as-is.
	// If any range covers the full text and is a LINK, return the URL instead.
	for _, er := range block.EntityRanges {
		// Guard against malformed ranges that extend beyond the text.
		if er.Offset < 0 || er.Length <= 0 || er.Offset+er.Length > len(runes) {
			continue
		}
		keyStr := entityKeyToString(er.Key)
		entity, ok := entityMap[keyStr]
		if !ok {
			continue
		}
		if entity.Type == "LINK" && er.Offset == 0 && er.Length == len(runes) {
			if url, ok := entity.Data["url"].(string); ok {
				return url
			}
		}
	}
	return block.Text
}

// entityKeyToString converts a raw JSON key (int or string) to a map-lookup string.
func entityKeyToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try integer first.
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return strconv.Itoa(n)
	}
	// Try string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}
