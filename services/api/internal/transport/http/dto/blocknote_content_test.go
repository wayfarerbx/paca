package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/Paca-AI/api/internal/transport/http/dto"
)

func TestValidateBlockNoteContent_AcceptsAbsentOrNull(t *testing.T) {
	if err := dto.ValidateBlockNoteContent(nil); err != nil {
		t.Fatalf("expected nil raw to be accepted, got %v", err)
	}
	if err := dto.ValidateBlockNoteContent(json.RawMessage("null")); err != nil {
		t.Fatalf("expected JSON null to be accepted, got %v", err)
	}
}

func TestValidateBlockNoteContent_AcceptsValidBlockArray(t *testing.T) {
	raw := json.RawMessage(`[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]`)
	if err := dto.ValidateBlockNoteContent(raw); err != nil {
		t.Fatalf("expected valid block array to be accepted, got %v", err)
	}
}

func TestValidateBlockNoteContent_AcceptsEmptyArray(t *testing.T) {
	if err := dto.ValidateBlockNoteContent(json.RawMessage(`[]`)); err != nil {
		t.Fatalf("expected empty array to be accepted, got %v", err)
	}
}

func TestValidateBlockNoteContent_RejectsPlainString(t *testing.T) {
	// This is the exact shape reported in GitHub issue #233: a plain string
	// sent as the description, which crashes the web app's block renderers.
	if err := dto.ValidateBlockNoteContent(json.RawMessage(`"just a plain string"`)); err == nil {
		t.Fatal("expected a plain string to be rejected")
	}
}

func TestValidateBlockNoteContent_RejectsObject(t *testing.T) {
	if err := dto.ValidateBlockNoteContent(json.RawMessage(`{"type":"paragraph"}`)); err == nil {
		t.Fatal("expected a bare object (not wrapped in an array) to be rejected")
	}
}

func TestValidateBlockNoteContent_RejectsNumber(t *testing.T) {
	if err := dto.ValidateBlockNoteContent(json.RawMessage(`42`)); err == nil {
		t.Fatal("expected a number to be rejected")
	}
}

func TestValidateBlockNoteContent_RejectsBlockMissingType(t *testing.T) {
	if err := dto.ValidateBlockNoteContent(json.RawMessage(`[{"content":[]}]`)); err == nil {
		t.Fatal("expected a block without a type field to be rejected")
	}
}
