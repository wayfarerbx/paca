package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ValidateBlockNoteContent checks that raw is either absent, JSON null, or a
// JSON array of BlockNote block objects (each a JSON object with a "type"
// string field). Task descriptions and document content are stored as
// BlockNote documents and rendered client-side with block-array assumptions
// (e.g. `.map()` over the blocks); accepting anything else (a bare string, a
// single object, a number, ...) would store data the web app cannot render
// and crashes on. Activity/doc comment content has its own, more permissive
// validation in the task/doc activity services (it also allows a legacy
// {"text": "..."} shape), so this validator is not used for comments.
func ValidateBlockNoteContent(raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}

	var blocks []map[string]any
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return fmt.Errorf("content must be a BlockNote document (a JSON array of blocks)")
	}
	for i, block := range blocks {
		if _, ok := block["type"].(string); !ok {
			return fmt.Errorf("content block at index %d is missing a \"type\" field", i)
		}
	}
	return nil
}
