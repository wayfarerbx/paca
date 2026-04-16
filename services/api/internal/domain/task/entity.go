// Package taskdom defines the task aggregate and its domain contracts.
package taskdom

import (
	"time"

	"github.com/google/uuid"
)

// StatusCategory describes which phase of the workflow a TaskStatus belongs to.
type StatusCategory string

// StatusCategory constants for task workflow phases.
const (
	StatusCategoryBacklog    StatusCategory = "backlog"
	StatusCategoryRefinement StatusCategory = "refinement"
	StatusCategoryReady      StatusCategory = "ready"
	StatusCategoryTodo       StatusCategory = "todo"
	StatusCategoryInProgress StatusCategory = "inprogress"
	StatusCategoryDone       StatusCategory = "done"
)

// ValidStatusCategories is the set of allowed status category values.
var ValidStatusCategories = map[StatusCategory]bool{
	StatusCategoryBacklog:    true,
	StatusCategoryRefinement: true,
	StatusCategoryReady:      true,
	StatusCategoryTodo:       true,
	StatusCategoryInProgress: true,
	StatusCategoryDone:       true,
}

// TaskType categorises tasks within a project.
type TaskType struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Icon        *string
	Color       *string
	Description *string
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskStatus represents a column in the project's status workflow.
type TaskStatus struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	Color     *string
	Position  int
	Category  StatusCategory
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FieldType describes the kind of value a CustomFieldDefinition accepts.
type FieldType string

// FieldType constants.
const (
	FieldTypeText        FieldType = "text"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiSelect FieldType = "multi_select"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeURL         FieldType = "url"
)

// ValidFieldTypes is the set of allowed field type values.
var ValidFieldTypes = map[FieldType]bool{
	FieldTypeText:        true,
	FieldTypeNumber:      true,
	FieldTypeDate:        true,
	FieldTypeSelect:      true,
	FieldTypeMultiSelect: true,
	FieldTypeBoolean:     true,
	FieldTypeURL:         true,
}

// CustomFieldDefinition is a project-level schema entry that describes one
// extra field that can be stored in Task.CustomFields under FieldKey.
type CustomFieldDefinition struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	FieldKey    string
	DisplayName string
	FieldType   FieldType
	Options     []string // populated for select / multi_select types
	IsRequired  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Task is the core work item aggregate.
type Task struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	TaskNumber   int64
	TaskTypeID   *uuid.UUID
	StatusID     *uuid.UUID
	SprintID     *uuid.UUID
	ParentTaskID *uuid.UUID
	Title        string
	Description  *string
	Importance   int
	AssigneeID   *uuid.UUID
	ReporterID   *uuid.UUID
	CustomFields map[string]any
	StartDate    *time.Time
	DueDate      *time.Time
	Tags         []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
