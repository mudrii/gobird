package parsing

import (
	"encoding/json"
	"strconv"

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
			return text
		}
	}
	return result.PreviewText
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
	result := ""
	for i, block := range cs.Blocks {
		if i > 0 {
			result += "\n"
		}
		if block.Type == "atomic" {
			result += renderAtomicBlock(block, cs.EntityMap)
		} else {
			result += block.Text
		}
	}
	return result
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
