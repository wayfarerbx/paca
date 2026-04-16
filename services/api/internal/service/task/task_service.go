// Package tasksvc implements task management application services.
package tasksvc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	taskdom "github.com/paca/api/internal/domain/task"
)

var reservedSystemTypeNames = map[string]bool{
	"Epic":    true,
	"Subtask": true,
}

// Service is the concrete implementation of taskdom.Service.
type Service struct {
	repo taskdom.Repository
}

// New returns a configured task service.
func New(repo taskdom.Repository) *Service {
	return &Service{repo: repo}
}

// --- Task Types -------------------------------------------------------------

// ListTaskTypes returns all task types for a project.
func (s *Service) ListTaskTypes(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskType, error) {
	return s.repo.ListTaskTypes(ctx, projectID)
}

// GetTaskType returns the task type with the given ID.
func (s *Service) GetTaskType(ctx context.Context, id uuid.UUID) (*taskdom.TaskType, error) {
	return s.repo.FindTaskTypeByID(ctx, id)
}

// CreateTaskType creates a new task type for the given project.
func (s *Service) CreateTaskType(ctx context.Context, in taskdom.CreateTaskTypeInput) (*taskdom.TaskType, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, taskdom.ErrTypeNameInvalid
	}
	if reservedSystemTypeNames[name] {
		return nil, taskdom.ErrTypeNameReserved
	}

	now := time.Now()
	t := &taskdom.TaskType{
		ID:          uuid.New(),
		ProjectID:   in.ProjectID,
		Name:        name,
		Icon:        in.Icon,
		Color:       in.Color,
		Description: in.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateTaskType(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// UpdateTaskType updates the mutable fields of an existing task type.
func (s *Service) UpdateTaskType(ctx context.Context, id uuid.UUID, in taskdom.UpdateTaskTypeInput) (*taskdom.TaskType, error) {
	t, err := s.repo.FindTaskTypeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.IsSystem {
		return nil, taskdom.ErrTypeIsSystem
	}

	if name := strings.TrimSpace(in.Name); name != "" {
		t.Name = name
	}
	t.Icon = in.Icon
	t.Color = in.Color
	t.Description = in.Description
	t.UpdatedAt = time.Now()

	if err := s.repo.UpdateTaskType(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTaskType removes a task type by ID.
func (s *Service) DeleteTaskType(ctx context.Context, id uuid.UUID) error {
	t, err := s.repo.FindTaskTypeByID(ctx, id)
	if err != nil {
		return err
	}
	if t.IsSystem {
		return taskdom.ErrTypeIsSystem
	}
	return s.repo.DeleteTaskType(ctx, id)
}

// SetDefaultTaskType marks typeID as the project's default task type,
// clearing the flag on all other types in the same project.
func (s *Service) SetDefaultTaskType(ctx context.Context, projectID, typeID uuid.UUID) (*taskdom.TaskType, error) {
	if err := s.repo.SetDefaultTaskType(ctx, projectID, typeID); err != nil {
		return nil, err
	}
	return s.repo.FindTaskTypeByID(ctx, typeID)
}

// --- Task Statuses ----------------------------------------------------------

// ListTaskStatuses returns all task statuses for a project.
func (s *Service) ListTaskStatuses(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskStatus, error) {
	return s.repo.ListTaskStatuses(ctx, projectID)
}

// GetTaskStatus returns the task status with the given ID.
func (s *Service) GetTaskStatus(ctx context.Context, id uuid.UUID) (*taskdom.TaskStatus, error) {
	return s.repo.FindTaskStatusByID(ctx, id)
}

// CreateTaskStatus creates a new task status for the given project.
func (s *Service) CreateTaskStatus(ctx context.Context, in taskdom.CreateTaskStatusInput) (*taskdom.TaskStatus, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, taskdom.ErrStatusNameInvalid
	}
	if !taskdom.ValidStatusCategories[in.Category] {
		return nil, taskdom.ErrStatusCategoryInvalid
	}

	now := time.Now()
	st := &taskdom.TaskStatus{
		ID:        uuid.New(),
		ProjectID: in.ProjectID,
		Name:      name,
		Color:     in.Color,
		Position:  in.Position,
		Category:  in.Category,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateTaskStatus(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// UpdateTaskStatus updates the mutable fields of an existing task status.
func (s *Service) UpdateTaskStatus(ctx context.Context, id uuid.UUID, in taskdom.UpdateTaskStatusInput) (*taskdom.TaskStatus, error) {
	st, err := s.repo.FindTaskStatusByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name := strings.TrimSpace(in.Name); name != "" {
		st.Name = name
	}
	st.Color = in.Color
	if in.Position != nil {
		st.Position = *in.Position
	}
	if in.Category != nil {
		if !taskdom.ValidStatusCategories[*in.Category] {
			return nil, taskdom.ErrStatusCategoryInvalid
		}
		st.Category = *in.Category
	}
	st.UpdatedAt = time.Now()

	if err := s.repo.UpdateTaskStatus(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// DeleteTaskStatus removes a task status by ID.
func (s *Service) DeleteTaskStatus(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindTaskStatusByID(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteTaskStatus(ctx, id)
}

// --- Tasks ------------------------------------------------------------------

// ListTasks returns a page of tasks for a project with optional filters.
func (s *Service) ListTasks(ctx context.Context, projectID uuid.UUID, filter taskdom.TaskFilter, page, pageSize int) ([]*taskdom.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.ListTasks(ctx, projectID, filter, offset, pageSize)
}

// GetTask returns the task with the given ID.
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*taskdom.Task, error) {
	return s.repo.FindTaskByID(ctx, id)
}

// GetTaskByNumber returns the task with the given project-scoped sequential number.
func (s *Service) GetTaskByNumber(ctx context.Context, projectID uuid.UUID, taskNumber int64) (*taskdom.Task, error) {
	return s.repo.FindTaskByNumber(ctx, projectID, taskNumber)
}

// CreateTask creates a new task.
func (s *Service) CreateTask(ctx context.Context, in taskdom.CreateTaskInput) (*taskdom.Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, taskdom.ErrTaskTitleInvalid
	}

	cf := in.CustomFields
	if cf == nil {
		cf = map[string]any{}
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	now := time.Now()
	t := &taskdom.Task{
		ID:           uuid.New(),
		ProjectID:    in.ProjectID,
		TaskTypeID:   in.TaskTypeID,
		StatusID:     in.StatusID,
		SprintID:     in.SprintID,
		ParentTaskID: in.ParentTaskID,
		Title:        title,
		Description:  in.Description,
		Importance:   in.Importance,
		AssigneeID:   in.AssigneeID,
		ReporterID:   in.ReporterID,
		CustomFields: cf,
		StartDate:    in.StartDate,
		DueDate:      in.DueDate,
		Tags:         tags,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateTask(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// UpdateTask updates the mutable fields of an existing task.
func (s *Service) UpdateTask(ctx context.Context, id uuid.UUID, in taskdom.UpdateTaskInput) (*taskdom.Task, error) {
	t, err := s.repo.FindTaskByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if title := strings.TrimSpace(in.Title); title != "" {
		t.Title = title
	}
	if in.TaskTypeID != nil {
		t.TaskTypeID = *in.TaskTypeID
	}
	if in.StatusID != nil {
		t.StatusID = *in.StatusID
	}
	if in.SprintID != nil {
		t.SprintID = *in.SprintID
	}
	if in.ParentTaskID != nil {
		t.ParentTaskID = *in.ParentTaskID
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if in.Importance != nil {
		t.Importance = *in.Importance
	}
	if in.AssigneeID != nil {
		t.AssigneeID = *in.AssigneeID
	}
	if in.ReporterID != nil {
		t.ReporterID = *in.ReporterID
	}
	if in.CustomFields != nil {
		t.CustomFields = *in.CustomFields
	}
	if in.StartDate != nil {
		t.StartDate = *in.StartDate
	}
	if in.DueDate != nil {
		t.DueDate = *in.DueDate
	}
	if in.Tags != nil {
		t.Tags = *in.Tags
	}
	t.UpdatedAt = time.Now()

	if err := s.repo.UpdateTask(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTask soft-deletes a task by ID.
func (s *Service) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTask(ctx, id)
}

// --- Custom Field Definitions -----------------------------------------------

// ListCustomFieldDefinitions returns all custom field definitions for a project.
func (s *Service) ListCustomFieldDefinitions(ctx context.Context, projectID uuid.UUID) ([]*taskdom.CustomFieldDefinition, error) {
	return s.repo.ListCustomFieldDefinitions(ctx, projectID)
}

// GetCustomFieldDefinition returns the custom field definition with the given ID.
func (s *Service) GetCustomFieldDefinition(ctx context.Context, id uuid.UUID) (*taskdom.CustomFieldDefinition, error) {
	return s.repo.FindCustomFieldDefinitionByID(ctx, id)
}

// CreateCustomFieldDefinition creates a new custom field definition.
func (s *Service) CreateCustomFieldDefinition(ctx context.Context, in taskdom.CreateCustomFieldDefinitionInput) (*taskdom.CustomFieldDefinition, error) {
	fieldKey := strings.TrimSpace(in.FieldKey)
	if fieldKey == "" {
		return nil, taskdom.ErrCustomFieldKeyInvalid
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		return nil, taskdom.ErrCustomFieldNameInvalid
	}
	if !taskdom.ValidFieldTypes[in.FieldType] {
		return nil, taskdom.ErrCustomFieldTypeInvalid
	}

	opts := in.Options
	if opts == nil {
		opts = []string{}
	}

	now := time.Now()
	f := &taskdom.CustomFieldDefinition{
		ID:          uuid.New(),
		ProjectID:   in.ProjectID,
		FieldKey:    fieldKey,
		DisplayName: displayName,
		FieldType:   in.FieldType,
		Options:     opts,
		IsRequired:  in.IsRequired,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateCustomFieldDefinition(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// UpdateCustomFieldDefinition updates the mutable fields of a custom field
// definition. The field_key is immutable after creation.
func (s *Service) UpdateCustomFieldDefinition(ctx context.Context, id uuid.UUID, in taskdom.UpdateCustomFieldDefinitionInput) (*taskdom.CustomFieldDefinition, error) {
	f, err := s.repo.FindCustomFieldDefinitionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if displayName := strings.TrimSpace(in.DisplayName); displayName != "" {
		f.DisplayName = displayName
	}
	if in.FieldType != nil {
		if !taskdom.ValidFieldTypes[*in.FieldType] {
			return nil, taskdom.ErrCustomFieldTypeInvalid
		}
		f.FieldType = *in.FieldType
	}
	if in.Options != nil {
		f.Options = in.Options
	}
	if in.IsRequired != nil {
		f.IsRequired = *in.IsRequired
	}
	f.UpdatedAt = time.Now()

	if err := s.repo.UpdateCustomFieldDefinition(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// DeleteCustomFieldDefinition removes a custom field definition by ID.
func (s *Service) DeleteCustomFieldDefinition(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindCustomFieldDefinitionByID(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteCustomFieldDefinition(ctx, id)
}

// --- BDD Scenarios ---------------------------------------------------------

// ListBDDScenarios returns all BDD scenarios for the given task.
func (s *Service) ListBDDScenarios(ctx context.Context, taskID uuid.UUID) ([]*taskdom.BDDScenario, error) {
	return s.repo.ListBDDScenarios(ctx, taskID)
}

// GetBDDScenario returns the BDD scenario with the given ID.
func (s *Service) GetBDDScenario(ctx context.Context, id uuid.UUID) (*taskdom.BDDScenario, error) {
	return s.repo.FindBDDScenarioByID(ctx, id)
}

// CreateBDDScenario creates a new BDD scenario for the given task.
func (s *Service) CreateBDDScenario(ctx context.Context, in taskdom.CreateBDDScenarioInput) (*taskdom.BDDScenario, error) {
	// Verify that the parent task exists.
	if _, err := s.repo.FindTaskByID(ctx, in.TaskID); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, taskdom.ErrBDDScenarioTitleInvalid
	}

	now := time.Now()
	scenario := &taskdom.BDDScenario{
		ID:        uuid.New(),
		TaskID:    in.TaskID,
		Title:     title,
		Given:     in.Given,
		When:      in.When,
		Then:      in.Then,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateBDDScenario(ctx, scenario); err != nil {
		return nil, err
	}
	return scenario, nil
}

// UpdateBDDScenario applies partial updates to an existing BDD scenario.
func (s *Service) UpdateBDDScenario(ctx context.Context, id uuid.UUID, in taskdom.UpdateBDDScenarioInput) (*taskdom.BDDScenario, error) {
	scenario, err := s.repo.FindBDDScenarioByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		scenario.Title = *in.Title
	}
	if in.Given != nil {
		scenario.Given = *in.Given
	}
	if in.When != nil {
		scenario.When = *in.When
	}
	if in.Then != nil {
		scenario.Then = *in.Then
	}
	scenario.UpdatedAt = time.Now()

	if err := s.repo.UpdateBDDScenario(ctx, scenario); err != nil {
		return nil, err
	}
	return scenario, nil
}

// DeleteBDDScenario removes a BDD scenario by ID.
func (s *Service) DeleteBDDScenario(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindBDDScenarioByID(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteBDDScenario(ctx, id)
}
