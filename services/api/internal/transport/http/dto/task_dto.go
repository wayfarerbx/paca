package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	taskdom "github.com/paca/api/internal/domain/task"
)

// --- Optional JSON field types ----------------------------------------------
// These types distinguish three states in a JSON PATCH body:
//   - field absent  → Set=false (do not overwrite the stored value)
//   - field = null  → Set=true, Value=nil  (explicitly clear the stored value)
//   - field = value → Set=true, Value=non-nil (set the stored value)

// OptionalUUID is a JSON-decodable field for nullable UUID columns.
type OptionalUUID struct {
	Set   bool
	Value *uuid.UUID
}

// UnmarshalJSON implements json.Unmarshaler. It marks the field as Set and
// decodes the value, treating JSON null as a nil pointer.
func (o *OptionalUUID) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var id uuid.UUID
	if err := json.Unmarshal(data, &id); err != nil {
		return fmt.Errorf("invalid uuid value: %w", err)
	}
	o.Value = &id
	return nil
}

// Ptr returns a **uuid.UUID suitable for UpdateTaskInput:
// nil when the field was absent, &nil when explicitly null, &&id when set.
func (o OptionalUUID) Ptr() **uuid.UUID {
	if !o.Set {
		return nil
	}
	return &o.Value
}

// OptionalString is a JSON-decodable field for nullable string columns.
type OptionalString struct {
	Set   bool
	Value *string
}

// UnmarshalJSON implements json.Unmarshaler. It marks the field as Set and
// decodes the value, treating JSON null as a nil pointer.
func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid string value: %w", err)
	}
	o.Value = &s
	return nil
}

// Ptr returns a **string for use in patch inputs: nil when absent, &nil when
// explicitly null, or a pointer-to-pointer when set to a value.
func (o OptionalString) Ptr() **string {
	if !o.Set {
		return nil
	}
	return &o.Value
}

// OptionalTime is a JSON-decodable field for nullable time columns.
type OptionalTime struct {
	Set   bool
	Value *time.Time
}

// UnmarshalJSON implements json.Unmarshaler. It marks the field as Set and
// decodes the value, treating JSON null as a nil pointer.
func (o *OptionalTime) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("invalid time value: %w", err)
	}
	o.Value = &t
	return nil
}

// Ptr returns a **time.Time for use in patch inputs: nil when absent, &nil
// when explicitly null, or a pointer-to-pointer when set to a value.
func (o OptionalTime) Ptr() **time.Time {
	if !o.Set {
		return nil
	}
	return &o.Value
}

// OptionalJSON is a JSON-decodable field for nullable JSONB columns.
// It distinguishes between a field that was absent in the request body
// (Set=false) and one that was explicitly set to null or a JSON value.
type OptionalJSON struct {
	Set   bool
	Value json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler. It marks the field as Set and
// stores the raw JSON bytes, treating JSON null as a nil RawMessage.
func (o *OptionalJSON) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	o.Value = json.RawMessage(data)
	return nil
}

// Ptr returns a *json.RawMessage for use in patch inputs: nil when absent,
// &nil when explicitly null, or a pointer to the raw bytes when set.
func (o OptionalJSON) Ptr() *json.RawMessage {
	if !o.Set {
		return nil
	}
	return &o.Value
}

// --- Task Type DTOs ---------------------------------------------------------

// CreateTaskTypeRequest is the body for POST /projects/:projectId/task-types.
type CreateTaskTypeRequest struct {
	Name        string  `json:"name" binding:"required"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

// UpdateTaskTypeRequest is the body for PATCH /projects/:projectId/task-types/:typeId.
type UpdateTaskTypeRequest struct {
	Name        string  `json:"name"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

// TaskTypeResponse is the public representation of a task type.
type TaskTypeResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Icon        *string   `json:"icon,omitempty"`
	Color       *string   `json:"color,omitempty"`
	Description *string   `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskTypeFromEntity maps a domain TaskType to a TaskTypeResponse DTO.
func TaskTypeFromEntity(t *taskdom.TaskType) TaskTypeResponse {
	return TaskTypeResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Name:        t.Name,
		Icon:        t.Icon,
		Color:       t.Color,
		Description: t.Description,
		IsDefault:   t.IsDefault,
		IsSystem:    t.IsSystem,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// --- Task Status DTOs -------------------------------------------------------

// CreateTaskStatusRequest is the body for POST /projects/:projectId/task-statuses.
type CreateTaskStatusRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Color    *string                `json:"color"`
	Position int                    `json:"position"`
	Category taskdom.StatusCategory `json:"category" binding:"required"`
}

// UpdateTaskStatusRequest is the body for PATCH /projects/:projectId/task-statuses/:statusId.
type UpdateTaskStatusRequest struct {
	Name     string                  `json:"name"`
	Color    *string                 `json:"color"`
	Position *int                    `json:"position"`
	Category *taskdom.StatusCategory `json:"category"`
}

// TaskStatusResponse is the public representation of a task status.
type TaskStatusResponse struct {
	ID        uuid.UUID              `json:"id"`
	ProjectID uuid.UUID              `json:"project_id"`
	Name      string                 `json:"name"`
	Color     *string                `json:"color,omitempty"`
	Position  int                    `json:"position"`
	Category  taskdom.StatusCategory `json:"category"`
	IsDefault bool                   `json:"is_default"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TaskStatusFromEntity maps a domain TaskStatus to a TaskStatusResponse DTO.
func TaskStatusFromEntity(s *taskdom.TaskStatus) TaskStatusResponse {
	return TaskStatusResponse{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Name:      s.Name,
		Color:     s.Color,
		Position:  s.Position,
		Category:  s.Category,
		IsDefault: s.IsDefault,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// --- Task DTOs --------------------------------------------------------------

// CreateTaskRequest is the body for POST /projects/:projectId/tasks.
type CreateTaskRequest struct {
	Title        string           `json:"title"`
	TaskTypeID   *uuid.UUID       `json:"task_type_id"`
	StatusID     *uuid.UUID       `json:"status_id"`
	SprintID     *uuid.UUID       `json:"sprint_id"`
	ParentTaskID *uuid.UUID       `json:"parent_task_id"`
	Description  *json.RawMessage `json:"description"`
	Importance   int              `json:"importance"`
	AssigneeID   *uuid.UUID       `json:"assignee_id"`
	ReporterID   *uuid.UUID       `json:"reporter_id"`
	CustomFields map[string]any   `json:"custom_fields"`
	StartDate    *time.Time       `json:"start_date"`
	DueDate      *time.Time       `json:"due_date"`
	Tags         []string         `json:"tags"`
}

// NormalizedDescription returns the description as a json.RawMessage suitable
// for CreateTaskInput. Both a missing field (nil pointer) and an explicit JSON
// null value are normalized to nil (SQL NULL).
func (r CreateTaskRequest) NormalizedDescription() json.RawMessage {
	if r.Description == nil || string(*r.Description) == "null" {
		return nil
	}
	return *r.Description
}

// UpdateTaskRequest is the body for PATCH /projects/:projectId/tasks/:taskId.
// Only fields present in the JSON body are applied; absent fields are left unchanged.
// For Tags and CustomFields, a nil pointer means absent (don't update); a non-nil
// pointer (even to an empty slice/map) means the field was explicitly set.
type UpdateTaskRequest struct {
	Title        string          `json:"title"`
	TaskTypeID   OptionalUUID    `json:"task_type_id"`
	StatusID     OptionalUUID    `json:"status_id"`
	SprintID     OptionalUUID    `json:"sprint_id"`
	ParentTaskID OptionalUUID    `json:"parent_task_id"`
	Description  OptionalJSON    `json:"description"`
	Importance   *int            `json:"importance"`
	AssigneeID   OptionalUUID    `json:"assignee_id"`
	ReporterID   OptionalUUID    `json:"reporter_id"`
	CustomFields *map[string]any `json:"custom_fields"`
	StartDate    OptionalTime    `json:"start_date"`
	DueDate      OptionalTime    `json:"due_date"`
	Tags         *[]string       `json:"tags"`
}

// TaskResponse is the public representation of a task.
// ViewPosition and ViewGroupKey are only populated when the caller supplies a
// valid view_id query parameter on list endpoints; they reflect the task's
// manual position within that view.
type TaskResponse struct {
	ID           uuid.UUID       `json:"id"`
	ProjectID    uuid.UUID       `json:"project_id"`
	TaskNumber   int64           `json:"task_number"`
	Title        string          `json:"title"`
	TaskTypeID   *uuid.UUID      `json:"task_type_id,omitempty"`
	StatusID     *uuid.UUID      `json:"status_id,omitempty"`
	SprintID     *uuid.UUID      `json:"sprint_id,omitempty"`
	ParentTaskID *uuid.UUID      `json:"parent_task_id,omitempty"`
	Description  json.RawMessage `json:"description,omitempty"`
	Importance   int             `json:"importance"`
	AssigneeID   *uuid.UUID      `json:"assignee_id,omitempty"`
	ReporterID   *uuid.UUID      `json:"reporter_id,omitempty"`
	CustomFields map[string]any  `json:"custom_fields"`
	StartDate    *time.Time      `json:"start_date,omitempty"`
	DueDate      *time.Time      `json:"due_date,omitempty"`
	Tags         []string        `json:"tags"`
	ViewPosition *float64        `json:"view_position,omitempty"`
	ViewGroupKey *string         `json:"view_group_key,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// TaskFromEntity maps a domain Task to a TaskResponse DTO.
func TaskFromEntity(t *taskdom.Task) TaskResponse {
	cf := t.CustomFields
	if cf == nil {
		cf = map[string]any{}
	}
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	return TaskResponse{
		ID:           t.ID,
		ProjectID:    t.ProjectID,
		TaskNumber:   t.TaskNumber,
		Title:        t.Title,
		TaskTypeID:   t.TaskTypeID,
		StatusID:     t.StatusID,
		SprintID:     t.SprintID,
		ParentTaskID: t.ParentTaskID,
		Description:  t.Description,
		Importance:   t.Importance,
		AssigneeID:   t.AssigneeID,
		ReporterID:   t.ReporterID,
		CustomFields: cf,
		StartDate:    t.StartDate,
		DueDate:      t.DueDate,
		Tags:         tags,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// --- Custom Field Definition DTOs ------------------------------------------

// CreateCustomFieldDefinitionRequest is the body for
// POST /projects/:projectId/custom-fields.
type CreateCustomFieldDefinitionRequest struct {
	FieldKey    string            `json:"field_key" binding:"required"`
	DisplayName string            `json:"display_name" binding:"required"`
	FieldType   taskdom.FieldType `json:"field_type" binding:"required"`
	Options     []string          `json:"options"`
	IsRequired  bool              `json:"is_required"`
}

// UpdateCustomFieldDefinitionRequest is the body for
// PATCH /projects/:projectId/custom-fields/:fieldId.
type UpdateCustomFieldDefinitionRequest struct {
	DisplayName string             `json:"display_name"`
	FieldType   *taskdom.FieldType `json:"field_type"`
	Options     []string           `json:"options"`
	IsRequired  *bool              `json:"is_required"`
}

// CustomFieldDefinitionResponse is the public representation of a custom
// field definition.
type CustomFieldDefinitionResponse struct {
	ID          uuid.UUID         `json:"id"`
	ProjectID   uuid.UUID         `json:"project_id"`
	FieldKey    string            `json:"field_key"`
	DisplayName string            `json:"display_name"`
	FieldType   taskdom.FieldType `json:"field_type"`
	Options     []string          `json:"options"`
	IsRequired  bool              `json:"is_required"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// CustomFieldDefinitionFromEntity maps a domain CustomFieldDefinition to a DTO.
func CustomFieldDefinitionFromEntity(f *taskdom.CustomFieldDefinition) CustomFieldDefinitionResponse {
	opts := f.Options
	if opts == nil {
		opts = []string{}
	}
	return CustomFieldDefinitionResponse{
		ID:          f.ID,
		ProjectID:   f.ProjectID,
		FieldKey:    f.FieldKey,
		DisplayName: f.DisplayName,
		FieldType:   f.FieldType,
		Options:     opts,
		IsRequired:  f.IsRequired,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
}

// --- BDD Scenarios ----------------------------------------------------------

// CreateBDDScenarioRequest is the body for
// POST /projects/:projectId/tasks/:taskId/bdd-scenarios.
type CreateBDDScenarioRequest struct {
	Title string `json:"title"    binding:"required"`
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

// UpdateBDDScenarioRequest is the body for
// PATCH /projects/:projectId/tasks/:taskId/bdd-scenarios/:scenarioId.
type UpdateBDDScenarioRequest struct {
	Title *string `json:"title"`
	Given *string `json:"given"`
	When  *string `json:"when"`
	Then  *string `json:"then"`
}

// BDDScenarioResponse is the public representation of a BDD scenario.
type BDDScenarioResponse struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	Title     string    `json:"title"`
	Given     string    `json:"given"`
	When      string    `json:"when"`
	Then      string    `json:"then"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BDDScenarioFromEntity maps a domain BDDScenario to a DTO.
func BDDScenarioFromEntity(s *taskdom.BDDScenario) BDDScenarioResponse {
	return BDDScenarioResponse{
		ID:        s.ID,
		TaskID:    s.TaskID,
		Title:     s.Title,
		Given:     s.Given,
		When:      s.When,
		Then:      s.Then,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// --- Activity / Comment DTOs -----------------------------------------------

// ActivityResponse is the public representation of a task activity entry.
type ActivityResponse struct {
	ID            uuid.UUID            `json:"id"`
	TaskID        uuid.UUID            `json:"task_id"`
	ActorID       *uuid.UUID           `json:"actor_id,omitempty"`
	ActorName     string               `json:"actor_name"`
	ActorUsername string               `json:"actor_username"`
	ActivityType  taskdom.ActivityType `json:"activity_type"`
	Content       json.RawMessage      `json:"content"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// ActivityFromEntity maps a domain Activity to an ActivityResponse DTO.
func ActivityFromEntity(a *taskdom.Activity) ActivityResponse {
	content := a.Content
	if len(content) == 0 {
		content = json.RawMessage("{}")
	}
	return ActivityResponse{
		ID:            a.ID,
		TaskID:        a.TaskID,
		ActorID:       a.ActorID,
		ActorName:     a.ActorName,
		ActorUsername: a.ActorUsername,
		ActivityType:  a.ActivityType,
		Content:       content,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

// AddCommentRequest is the body for
// POST /projects/:projectId/tasks/:taskId/activities/comments.
type AddCommentRequest struct {
	Text string `json:"text" binding:"required"`
}

// UpdateCommentRequest is the body for
// PATCH /projects/:projectId/tasks/:taskId/activities/comments/:commentId.
type UpdateCommentRequest struct {
	Text string `json:"text" binding:"required"`
}
