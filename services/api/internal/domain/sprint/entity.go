// Package sprintdom defines the sprint aggregate and its domain contracts.
package sprintdom

import (
	"time"

	"github.com/google/uuid"
)

// SprintStatus describes the lifecycle state of a Sprint.
type SprintStatus string

// SprintStatus constants for planned, active, and completed sprints.
const (
	SprintStatusPlanned   SprintStatus = "planned"
	SprintStatusActive    SprintStatus = "active"
	SprintStatusCompleted SprintStatus = "completed"
)

// ValidSprintStatuses is the set of allowed sprint status values.
var ValidSprintStatuses = map[SprintStatus]bool{
	SprintStatusPlanned:   true,
	SprintStatusActive:    true,
	SprintStatusCompleted: true,
}

// Sprint is a time-boxed iteration containing a set of tasks.
type Sprint struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	StartDate *time.Time
	EndDate   *time.Time
	Goal      *string
	Status    SprintStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ViewType is the layout variant of a sprint view.
type ViewType string

// View type constants.
const (
	ViewTypeTable   ViewType = "table"
	ViewTypeBoard   ViewType = "board"
	ViewTypeRoadmap ViewType = "roadmap"
)

// ValidViewTypes is the set of allowed view_type values.
var ValidViewTypes = map[ViewType]bool{
	ViewTypeTable:   true,
	ViewTypeBoard:   true,
	ViewTypeRoadmap: true,
}

// ViewConfig holds the display settings for a sprint view.
// All fields are optional; when empty the client applies defaults.
type ViewConfig struct {
	Fields    []string `json:"fields,omitempty"`
	ColumnBy  string   `json:"column_by,omitempty"`
	Swimlanes string   `json:"swimlanes,omitempty"`
	SortBy    string   `json:"sort_by,omitempty"`
	FieldSum  string   `json:"field_sum,omitempty"`
	SliceBy   string   `json:"slice_by,omitempty"`
}

// SprintView is a named, persisted view configuration for a sprint or
// product-backlog integration.
// SprintID is nil for product-backlog views; ProjectID is always set.
type SprintView struct {
	ID        uuid.UUID
	SprintID  *uuid.UUID
	ProjectID uuid.UUID
	Name      string
	ViewType  ViewType
	Config    ViewConfig
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ViewTaskPosition records the manual ordering of a task within a specific
// view.  Only used when SprintView.Config.SortBy == "manual".
type ViewTaskPosition struct {
	ID       uuid.UUID
	ViewID   uuid.UUID
	TaskID   uuid.UUID
	Position int
	GroupKey *string
}
