package taskdom

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service is the combined task management service contract.
type Service interface {
	TaskTypeService
	TaskStatusService
	TaskService
	CustomFieldDefinitionService
}

// --- Task Type Service -----------------------------------------------------

// TaskTypeService defines task-type use cases.
type TaskTypeService interface {
	ListTaskTypes(ctx context.Context, projectID uuid.UUID) ([]*TaskType, error)
	GetTaskType(ctx context.Context, id uuid.UUID) (*TaskType, error)
	CreateTaskType(ctx context.Context, in CreateTaskTypeInput) (*TaskType, error)
	UpdateTaskType(ctx context.Context, id uuid.UUID, in UpdateTaskTypeInput) (*TaskType, error)
	DeleteTaskType(ctx context.Context, id uuid.UUID) error
}

// CreateTaskTypeInput carries fields required to create a task type.
type CreateTaskTypeInput struct {
	ProjectID   uuid.UUID
	Name        string
	Icon        *string
	Color       *string
	Description *string
}

// UpdateTaskTypeInput carries mutable task-type fields.
type UpdateTaskTypeInput struct {
	Name        string
	Icon        *string
	Color       *string
	Description *string
}

// --- Task Status Service ---------------------------------------------------

// TaskStatusService defines task-status use cases.
type TaskStatusService interface {
	ListTaskStatuses(ctx context.Context, projectID uuid.UUID) ([]*TaskStatus, error)
	GetTaskStatus(ctx context.Context, id uuid.UUID) (*TaskStatus, error)
	CreateTaskStatus(ctx context.Context, in CreateTaskStatusInput) (*TaskStatus, error)
	UpdateTaskStatus(ctx context.Context, id uuid.UUID, in UpdateTaskStatusInput) (*TaskStatus, error)
	DeleteTaskStatus(ctx context.Context, id uuid.UUID) error
}

// CreateTaskStatusInput carries fields required to create a task status.
type CreateTaskStatusInput struct {
	ProjectID uuid.UUID
	Name      string
	Color     *string
	Position  int
	Category  StatusCategory
}

// UpdateTaskStatusInput carries mutable task-status fields.
type UpdateTaskStatusInput struct {
	Name     string
	Color    *string
	Position *int
	Category *StatusCategory
}

// --- Task Service ----------------------------------------------------------

// TaskService defines task use cases.
type TaskService interface {
	ListTasks(ctx context.Context, projectID uuid.UUID, filter TaskFilter, page, pageSize int) ([]*Task, int64, error)
	GetTask(ctx context.Context, id uuid.UUID) (*Task, error)
	CreateTask(ctx context.Context, in CreateTaskInput) (*Task, error)
	UpdateTask(ctx context.Context, id uuid.UUID, in UpdateTaskInput) (*Task, error)
	DeleteTask(ctx context.Context, id uuid.UUID) error
}

// CreateTaskInput carries fields required to create a task.
type CreateTaskInput struct {
	ProjectID    uuid.UUID
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
}

// UpdateTaskInput carries mutable task fields for a PATCH operation.
// Title is applied when non-empty.
// For nullable reference fields, the double-pointer encodes three states:
//   - nil outer pointer  → field was absent in the request; do NOT overwrite
//   - non-nil outer pointer, inner pointer nil  → explicitly set to null (clear)
//   - non-nil outer pointer, inner pointer non-nil  → set to the given value
//
// For slice/map fields (Tags, CustomFields), a nil pointer means the field was
// absent and should not be overwritten; a non-nil pointer (even to an empty
// slice/map) means the field was explicitly set and replaces the stored value.
type UpdateTaskInput struct {
	TaskTypeID   **uuid.UUID
	StatusID     **uuid.UUID
	SprintID     **uuid.UUID
	ParentTaskID **uuid.UUID
	Title        string
	Description  **string
	Importance   *int
	AssigneeID   **uuid.UUID
	ReporterID   **uuid.UUID
	CustomFields *map[string]any
	StartDate    **time.Time
	DueDate      **time.Time
	Tags         *[]string
}

// --- Custom Field Definition Service --------------------------------------

// CustomFieldDefinitionService defines custom field definition use cases.
type CustomFieldDefinitionService interface {
	ListCustomFieldDefinitions(ctx context.Context, projectID uuid.UUID) ([]*CustomFieldDefinition, error)
	GetCustomFieldDefinition(ctx context.Context, id uuid.UUID) (*CustomFieldDefinition, error)
	CreateCustomFieldDefinition(ctx context.Context, in CreateCustomFieldDefinitionInput) (*CustomFieldDefinition, error)
	UpdateCustomFieldDefinition(ctx context.Context, id uuid.UUID, in UpdateCustomFieldDefinitionInput) (*CustomFieldDefinition, error)
	DeleteCustomFieldDefinition(ctx context.Context, id uuid.UUID) error
}

// CreateCustomFieldDefinitionInput carries fields required to create a custom
// field definition.
type CreateCustomFieldDefinitionInput struct {
	ProjectID   uuid.UUID
	FieldKey    string
	DisplayName string
	FieldType   FieldType
	Options     []string
	IsRequired  bool
}

// UpdateCustomFieldDefinitionInput carries mutable custom field definition
// fields.
type UpdateCustomFieldDefinitionInput struct {
	DisplayName string
	FieldType   *FieldType
	Options     []string
	IsRequired  *bool
}
