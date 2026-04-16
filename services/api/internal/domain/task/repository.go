package taskdom

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the combined persistence contract for the task aggregate.
type Repository interface {
	TaskTypeRepository
	TaskStatusRepository
	TaskRepository
	CustomFieldDefinitionRepository
}

// TaskTypeRepository defines persistence operations for task types.
type TaskTypeRepository interface {
	ListTaskTypes(ctx context.Context, projectID uuid.UUID) ([]*TaskType, error)
	FindTaskTypeByID(ctx context.Context, id uuid.UUID) (*TaskType, error)
	CreateTaskType(ctx context.Context, t *TaskType) error
	UpdateTaskType(ctx context.Context, t *TaskType) error
	DeleteTaskType(ctx context.Context, id uuid.UUID) error
	// SetDefaultTaskType atomically clears is_default for every type in the
	// project and then marks the given type as the default.
	SetDefaultTaskType(ctx context.Context, projectID, typeID uuid.UUID) error
}

// TaskStatusRepository defines persistence operations for task statuses.
type TaskStatusRepository interface {
	ListTaskStatuses(ctx context.Context, projectID uuid.UUID) ([]*TaskStatus, error)
	FindTaskStatusByID(ctx context.Context, id uuid.UUID) (*TaskStatus, error)
	CreateTaskStatus(ctx context.Context, s *TaskStatus) error
	UpdateTaskStatus(ctx context.Context, s *TaskStatus) error
	DeleteTaskStatus(ctx context.Context, id uuid.UUID) error
}

// TaskRepository defines persistence operations for tasks.
type TaskRepository interface {
	ListTasks(ctx context.Context, projectID uuid.UUID, filter TaskFilter, offset, limit int) ([]*Task, int64, error)
	FindTaskByID(ctx context.Context, id uuid.UUID) (*Task, error)
	FindTaskByNumber(ctx context.Context, projectID uuid.UUID, taskNumber int64) (*Task, error)
	CreateTask(ctx context.Context, t *Task) error
	UpdateTask(ctx context.Context, t *Task) error
	DeleteTask(ctx context.Context, id uuid.UUID) error
	// BulkMoveSprintTasks reassigns all non-done tasks in sourceSprintID to
	// targetSprintID. A nil targetSprintID moves tasks to the backlog (sprint_id = NULL).
	// Tasks whose status has category "done" are not moved.
	BulkMoveSprintTasks(ctx context.Context, projectID, sourceSprintID uuid.UUID, targetSprintID *uuid.UUID) error
}

// TaskFilter carries optional criteria for listing tasks.
type TaskFilter struct {
	SprintID     *uuid.UUID  // single-value compat; ignored when SprintIDs is non-empty
	SprintIDs    []uuid.UUID // multi-value; takes priority over SprintID and BacklogOnly
	StatusID     *uuid.UUID  // single-value compat; ignored when StatusIDs is non-empty
	StatusIDs    []uuid.UUID // multi-value; takes priority over StatusID
	AssigneeID   *uuid.UUID  // single-value compat; ignored when AssigneeIDs is non-empty
	AssigneeIDs  []uuid.UUID // multi-value; takes priority over AssigneeID
	ParentTaskID *uuid.UUID  // non-nil → only subtasks of this parent
	TaskTypeIDs  []uuid.UUID // multi-value; when non-empty, only tasks of these types
	BacklogOnly  bool        // true → only tasks where sprint_id IS NULL
}

// CustomFieldDefinitionRepository defines persistence operations for custom
// field definitions.
type CustomFieldDefinitionRepository interface {
	ListCustomFieldDefinitions(ctx context.Context, projectID uuid.UUID) ([]*CustomFieldDefinition, error)
	FindCustomFieldDefinitionByID(ctx context.Context, id uuid.UUID) (*CustomFieldDefinition, error)
	CreateCustomFieldDefinition(ctx context.Context, f *CustomFieldDefinition) error
	UpdateCustomFieldDefinition(ctx context.Context, f *CustomFieldDefinition) error
	DeleteCustomFieldDefinition(ctx context.Context, id uuid.UUID) error
}
