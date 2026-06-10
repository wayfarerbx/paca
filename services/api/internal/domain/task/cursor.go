package taskdom

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// TaskCursor holds the stable ordering fields for keyset-based pagination.
// Both the HTTP handler (encoding) and the postgres repository (decoding)
// must use this type so any change to the JSON tags is caught at compile time.
type TaskCursor struct {
	CreatedAt time.Time `json:"ca"`
	ID        string    `json:"id"`
}

// EncodeTaskCursor builds an opaque base64 cursor from a task's creation time and ID.
func EncodeTaskCursor(createdAt time.Time, id string) string {
	b, _ := json.Marshal(TaskCursor{CreatedAt: createdAt.UTC(), ID: id})
	return base64.URLEncoding.EncodeToString(b)
}

// DecodeTaskCursor parses a cursor token produced by EncodeTaskCursor.
func DecodeTaskCursor(s string) (*TaskCursor, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode cursor base64: %w", err)
	}
	var c TaskCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("decode cursor json: %w", err)
	}
	c.CreatedAt = c.CreatedAt.UTC()
	return &c, nil
}
