// Package postgres — GORM implementation of taskdom.Repository.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	taskdom "github.com/Paca-AI/api/internal/domain/task"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- GORM models ------------------------------------------------------------

type taskTypeRecord struct {
	ID          string  `gorm:"primarykey;type:uuid"`
	ProjectID   string  `gorm:"type:uuid;not null;column:project_id"`
	Name        string  `gorm:"not null"`
	Icon        *string `gorm:"type:text"`
	Color       *string `gorm:"type:text"`
	Description *string `gorm:"type:text"`
	IsDefault   bool    `gorm:"not null;default:false;column:is_default"`
	IsSystem    bool    `gorm:"not null;default:false;column:is_system"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (taskTypeRecord) TableName() string { return "task_types" }

type taskStatusRecord struct {
	ID        string  `gorm:"primarykey;type:uuid"`
	ProjectID string  `gorm:"type:uuid;not null;column:project_id"`
	Name      string  `gorm:"not null"`
	Color     *string `gorm:"type:text"`
	Position  int     `gorm:"not null;default:0"`
	Category  string  `gorm:"not null"`
	IsDefault bool    `gorm:"not null;default:false;column:is_default"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (taskStatusRecord) TableName() string { return "task_statuses" }

type taskRecord struct {
	ID           string          `gorm:"primarykey;type:uuid"`
	ProjectID    string          `gorm:"type:uuid;not null;column:project_id"`
	TaskNumber   int64           `gorm:"not null;default:0;column:task_number"`
	TaskTypeID   *string         `gorm:"type:uuid;column:task_type_id"`
	StatusID     *string         `gorm:"type:uuid;column:status_id"`
	SprintID     *string         `gorm:"type:uuid;column:sprint_id"`
	ParentTaskID *string         `gorm:"type:uuid;column:parent_task_id"`
	Title        string          `gorm:"not null"`
	Description  json.RawMessage `gorm:"type:jsonb"`
	Importance   int             `gorm:"not null;default:0"`
	StoryPoints  *int            `gorm:"column:story_points"`
	AssigneeID   *string         `gorm:"type:uuid;column:assignee_id"`
	ReporterID   *string         `gorm:"type:uuid;column:reporter_id"`
	CustomFields []byte          `gorm:"type:jsonb;not null;column:custom_fields"`
	StartDate    *time.Time      `gorm:"type:date;column:start_date"`
	DueDate      *time.Time      `gorm:"type:date;column:due_date"`
	Tags         []byte          `gorm:"type:jsonb;not null;column:tags"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (taskRecord) TableName() string { return "tasks" }

// taskCounterRecord mirrors the task_counters table used for atomic
// per-project task number generation.
type taskCounterRecord struct {
	ProjectID string `gorm:"primarykey;type:uuid;column:project_id"`
	LastValue int64  `gorm:"not null;default:0;column:last_value"`
}

func (taskCounterRecord) TableName() string { return "task_counters" }

// --- Repository struct -------------------------------------------------------

// TaskRepository is the GORM implementation of taskdom.Repository.
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository returns a new TaskRepository.
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// --- Task Types -------------------------------------------------------------

// ListTaskTypes returns all task types for a project.
func (r *TaskRepository) ListTaskTypes(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskType, error) {
	var records []taskTypeRecord
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Order("name ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("task type repo: list: %w", err)
	}
	out := make([]*taskdom.TaskType, 0, len(records))
	for i := range records {
		out = append(out, toTaskTypeEntity(&records[i]))
	}
	return out, nil
}

// FindTaskTypeByID returns the task type with the given ID.
func (r *TaskRepository) FindTaskTypeByID(ctx context.Context, id uuid.UUID) (*taskdom.TaskType, error) {
	var rec taskTypeRecord
	err := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, taskdom.ErrTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("task type repo: find by id: %w", err)
	}
	return toTaskTypeEntity(&rec), nil
}

// CreateTaskType persists a new task type.
func (r *TaskRepository) CreateTaskType(ctx context.Context, t *taskdom.TaskType) error {
	rec := &taskTypeRecord{
		ID:          t.ID.String(),
		ProjectID:   t.ProjectID.String(),
		Name:        t.Name,
		Icon:        t.Icon,
		Color:       t.Color,
		Description: t.Description,
		IsDefault:   t.IsDefault,
		IsSystem:    t.IsSystem,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		return fmt.Errorf("task type repo: create: %w", err)
	}
	return nil
}

// UpdateTaskType persists changes to an existing task type.
func (r *TaskRepository) UpdateTaskType(ctx context.Context, t *taskdom.TaskType) error {
	updates := map[string]any{
		"name":        t.Name,
		"icon":        t.Icon,
		"color":       t.Color,
		"description": t.Description,
		"updated_at":  t.UpdatedAt,
	}
	res := r.db.WithContext(ctx).Model(&taskTypeRecord{}).Where("id = ?", t.ID.String()).Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("task type repo: update: %w", res.Error)
	}
	return nil
}

// DeleteTaskType removes a task type by ID.
func (r *TaskRepository) DeleteTaskType(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&taskTypeRecord{}, "id = ?", id.String())
	if res.Error != nil {
		return fmt.Errorf("task type repo: delete: %w", res.Error)
	}
	return nil
}

// SetDefaultTaskType atomically marks typeID as the project's default task type,
// clearing is_default on all other types in the same project. The operation is
// expressed as a single SQL statement so concurrent calls cannot produce multiple
// defaults. The partial unique index uq_task_types_one_default (project_id WHERE
// is_default = true) further enforces this invariant at the DB level.
func (r *TaskRepository) SetDefaultTaskType(ctx context.Context, projectID, typeID uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE task_types
		SET is_default = (id = ?), updated_at = NOW()
		WHERE project_id = ?
		  AND EXISTS (SELECT 1 FROM task_types WHERE id = ? AND project_id = ?)`,
		typeID.String(), projectID.String(), typeID.String(), projectID.String(),
	)
	if res.Error != nil {
		return fmt.Errorf("task type repo: set default: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return taskdom.ErrTypeNotFound
	}
	return nil
}

// --- Task Statuses ---------------------------------------------------------

// ListTaskStatuses returns all task statuses for a project ordered by position.
func (r *TaskRepository) ListTaskStatuses(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskStatus, error) {
	var records []taskStatusRecord
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Order("position ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("task status repo: list: %w", err)
	}
	out := make([]*taskdom.TaskStatus, 0, len(records))
	for i := range records {
		out = append(out, toTaskStatusEntity(&records[i]))
	}
	return out, nil
}

// FindTaskStatusByID returns the task status with the given ID.
func (r *TaskRepository) FindTaskStatusByID(ctx context.Context, id uuid.UUID) (*taskdom.TaskStatus, error) {
	var rec taskStatusRecord
	err := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, taskdom.ErrStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("task status repo: find by id: %w", err)
	}
	return toTaskStatusEntity(&rec), nil
}

// CreateTaskStatus persists a new task status.
func (r *TaskRepository) CreateTaskStatus(ctx context.Context, s *taskdom.TaskStatus) error {
	rec := &taskStatusRecord{
		ID:        s.ID.String(),
		ProjectID: s.ProjectID.String(),
		Name:      s.Name,
		Color:     s.Color,
		Position:  s.Position,
		Category:  string(s.Category),
		IsDefault: s.IsDefault,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		return fmt.Errorf("task status repo: create: %w", err)
	}
	return nil
}

// UpdateTaskStatus persists changes to an existing task status.
func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, s *taskdom.TaskStatus) error {
	updates := map[string]any{
		"name":       s.Name,
		"color":      s.Color,
		"category":   string(s.Category),
		"updated_at": s.UpdatedAt,
	}
	res := r.db.WithContext(ctx).Model(&taskStatusRecord{}).Where("id = ?", s.ID.String()).Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("task status repo: update: %w", res.Error)
	}
	return nil
}

// DeleteTaskStatus removes a task status by ID.
func (r *TaskRepository) DeleteTaskStatus(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&taskStatusRecord{}, "id = ?", id.String())
	if res.Error != nil {
		return fmt.Errorf("task status repo: delete: %w", res.Error)
	}
	return nil
}

// SetDefaultTaskStatus atomically marks statusID as the project's default task
// status, clearing is_default on all other statuses in the same project. The
// operation is expressed as a single SQL statement so concurrent calls cannot
// produce multiple defaults. The partial unique index
// uq_task_statuses_one_default (project_id WHERE is_default = true) further
// enforces this invariant at the DB level.
func (r *TaskRepository) SetDefaultTaskStatus(ctx context.Context, projectID, statusID uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE task_statuses
		SET is_default = (id = ?), updated_at = NOW()
		WHERE project_id = ?
		  AND EXISTS (SELECT 1 FROM task_statuses WHERE id = ? AND project_id = ?)`,
		statusID.String(), projectID.String(), statusID.String(), projectID.String(),
	)
	if res.Error != nil {
		return fmt.Errorf("task status repo: set default: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return taskdom.ErrStatusNotFound
	}
	return nil
}

// FindDefaultTaskType returns the project's default task type, or nil if none is set.
func (r *TaskRepository) FindDefaultTaskType(ctx context.Context, projectID uuid.UUID) (*taskdom.TaskType, error) {
	var rec taskTypeRecord
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND is_default = true", projectID.String()).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("task type repo: find default: %w", err)
	}
	return toTaskTypeEntity(&rec), nil
}

// FindDefaultTaskStatus returns the project's default task status, or nil if none is set.
func (r *TaskRepository) FindDefaultTaskStatus(ctx context.Context, projectID uuid.UUID) (*taskdom.TaskStatus, error) {
	var rec taskStatusRecord
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND is_default = true", projectID.String()).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("task status repo: find default: %w", err)
	}
	return toTaskStatusEntity(&rec), nil
}

// --- Tasks ------------------------------------------------------------------

// ListTasks returns a page of tasks for a project with optional filters.
func (r *TaskRepository) ListTasks(ctx context.Context, projectID uuid.UUID, filter taskdom.TaskFilter, offset, limit int) ([]*taskdom.Task, int64, error) {
	q := r.db.WithContext(ctx).Model(&taskRecord{}).
		Where("project_id = ?", projectID.String())

	switch {
	case filter.ParentTaskID != nil:
		q = q.Where("parent_task_id = ?", filter.ParentTaskID.String())
	case len(filter.SprintIDs) > 0:
		q = q.Where("sprint_id IN ?", uuidSliceToStrSlice(filter.SprintIDs))
	case filter.BacklogOnly:
		q = q.Where("sprint_id IS NULL")
	case filter.SprintID != nil:
		q = q.Where("sprint_id = ?", filter.SprintID.String())
	}
	if len(filter.StatusIDs) > 0 {
		q = q.Where("status_id IN ?", uuidSliceToStrSlice(filter.StatusIDs))
	} else if filter.StatusID != nil {
		q = q.Where("status_id = ?", filter.StatusID.String())
	}
	if len(filter.AssigneeIDs) > 0 {
		q = q.Where("assignee_id IN ?", uuidSliceToStrSlice(filter.AssigneeIDs))
	} else if filter.AssigneeID != nil {
		q = q.Where("assignee_id = ?", filter.AssigneeID.String())
	}
	if len(filter.TaskTypeIDs) > 0 {
		q = q.Where("task_type_id IN ?", uuidSliceToStrSlice(filter.TaskTypeIDs))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("task repo: list count: %w", err)
	}

	var records []taskRecord
	if err := q.Order("created_at ASC").
		Offset(offset).Limit(limit).
		Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("task repo: list: %w", err)
	}

	tasks := make([]*taskdom.Task, 0, len(records))
	for i := range records {
		t, err := toTaskEntity(&records[i])
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, nil
}

// FindTaskByID returns the task with the given ID (non-deleted).
func (r *TaskRepository) FindTaskByID(ctx context.Context, id uuid.UUID) (*taskdom.Task, error) {
	var rec taskRecord
	err := r.db.WithContext(ctx).
		Where("id = ?", id.String()).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, taskdom.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("task repo: find by id: %w", err)
	}
	return toTaskEntity(&rec)
}

// FindTaskByNumber returns the task with the given project-scoped task number
// (non-deleted).
func (r *TaskRepository) FindTaskByNumber(ctx context.Context, projectID uuid.UUID, taskNumber int64) (*taskdom.Task, error) {
	var rec taskRecord
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND task_number = ?", projectID.String(), taskNumber).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, taskdom.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("task repo: find by number: %w", err)
	}
	return toTaskEntity(&rec)
}

// CreateTask persists a new task, assigning the next per-project task_number
// atomically via INSERT … ON CONFLICT DO UPDATE on task_counters.
func (r *TaskRepository) CreateTask(ctx context.Context, t *taskdom.Task) error {
	cf, err := json.Marshal(t.CustomFields)
	if err != nil {
		return fmt.Errorf("task repo: marshal custom_fields: %w", err)
	}
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("task repo: marshal tags: %w", err)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Atomically increment the per-project counter and retrieve its new value.
		var counter taskCounterRecord
		if err := tx.Raw(`
			INSERT INTO task_counters (project_id, last_value)
			VALUES (?, 1)
			ON CONFLICT (project_id) DO UPDATE
			  SET last_value = task_counters.last_value + 1
			RETURNING last_value`,
			t.ProjectID.String(),
		).Scan(&counter).Error; err != nil {
			return fmt.Errorf("task repo: increment counter: %w", err)
		}
		t.TaskNumber = counter.LastValue

		rec := &taskRecord{
			ID:           t.ID.String(),
			ProjectID:    t.ProjectID.String(),
			TaskNumber:   t.TaskNumber,
			TaskTypeID:   uuidPtrToStrPtr(t.TaskTypeID),
			StatusID:     uuidPtrToStrPtr(t.StatusID),
			SprintID:     uuidPtrToStrPtr(t.SprintID),
			ParentTaskID: uuidPtrToStrPtr(t.ParentTaskID),
			Title:        t.Title,
			Description:  t.Description,
			Importance:   t.Importance,
			StoryPoints:  t.StoryPoints,
			AssigneeID:   uuidPtrToStrPtr(t.AssigneeID),
			ReporterID:   uuidPtrToStrPtr(t.ReporterID),
			CustomFields: cf,
			StartDate:    t.StartDate,
			DueDate:      t.DueDate,
			Tags:         tagsJSON,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		}
		if err := tx.Create(rec).Error; err != nil {
			return fmt.Errorf("task repo: create: %w", err)
		}
		return nil
	})
}

// UpdateTask persists changes to an existing task.
func (r *TaskRepository) UpdateTask(ctx context.Context, t *taskdom.Task) error {
	cf, err := json.Marshal(t.CustomFields)
	if err != nil {
		return fmt.Errorf("task repo: marshal custom_fields: %w", err)
	}
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("task repo: marshal tags: %w", err)
	}
	updates := map[string]any{
		"task_type_id":   uuidPtrToStrPtr(t.TaskTypeID),
		"status_id":      uuidPtrToStrPtr(t.StatusID),
		"sprint_id":      uuidPtrToStrPtr(t.SprintID),
		"parent_task_id": uuidPtrToStrPtr(t.ParentTaskID),
		"title":          t.Title,
		"description":    t.Description,
		"importance":     t.Importance,
		"story_points":   t.StoryPoints,
		"assignee_id":    uuidPtrToStrPtr(t.AssigneeID),
		"reporter_id":    uuidPtrToStrPtr(t.ReporterID),
		"custom_fields":  cf,
		"start_date":     t.StartDate,
		"due_date":       t.DueDate,
		"tags":           tagsJSON,
		"updated_at":     t.UpdatedAt,
	}
	res := r.db.WithContext(ctx).Model(&taskRecord{}).
		Where("id = ?", t.ID.String()).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("task repo: update: %w", res.Error)
	}
	return nil
}

// DeleteTask soft-deletes a task by setting deleted_at.
func (r *TaskRepository) DeleteTask(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ?", id.String()).
		Delete(&taskRecord{})
	if res.Error != nil {
		return fmt.Errorf("task repo: delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return taskdom.ErrTaskNotFound
	}
	return nil
}

// BulkMoveSprintTasks reassigns all non-done tasks that belong to sourceSprintID
// to targetSprintID (nil = backlog) in a single UPDATE statement.
func (r *TaskRepository) BulkMoveSprintTasks(ctx context.Context, projectID, sourceSprintID uuid.UUID, targetSprintID *uuid.UUID) error {
	doneStatusSubquery := r.db.Model(&taskStatusRecord{}).
		Select("id").
		Where("project_id = ? AND category = ?", projectID.String(), string(taskdom.StatusCategoryDone))

	res := r.db.WithContext(ctx).Model(&taskRecord{}).
		Where("project_id = ? AND sprint_id = ?", projectID.String(), sourceSprintID.String()).
		Where("status_id IS NULL OR status_id NOT IN (?)", doneStatusSubquery).
		Updates(map[string]any{
			"sprint_id":  uuidPtrToStrPtr(targetSprintID),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return fmt.Errorf("task repo: bulk move sprint tasks: %w", res.Error)
	}
	return nil
}

// --- Entity converters ------------------------------------------------------

func toTaskTypeEntity(r *taskTypeRecord) *taskdom.TaskType {
	id, _ := uuid.Parse(r.ID)
	pid, _ := uuid.Parse(r.ProjectID)
	return &taskdom.TaskType{
		ID:          id,
		ProjectID:   pid,
		Name:        r.Name,
		Icon:        r.Icon,
		Color:       r.Color,
		Description: r.Description,
		IsDefault:   r.IsDefault,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toTaskStatusEntity(r *taskStatusRecord) *taskdom.TaskStatus {
	id, _ := uuid.Parse(r.ID)
	pid, _ := uuid.Parse(r.ProjectID)
	return &taskdom.TaskStatus{
		ID:        id,
		ProjectID: pid,
		Name:      r.Name,
		Color:     r.Color,
		Position:  r.Position,
		Category:  taskdom.StatusCategory(r.Category),
		IsDefault: r.IsDefault,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func toTaskEntity(r *taskRecord) (*taskdom.Task, error) {
	id, _ := uuid.Parse(r.ID)
	pid, _ := uuid.Parse(r.ProjectID)

	var cf map[string]any
	if len(r.CustomFields) > 0 {
		if err := json.Unmarshal(r.CustomFields, &cf); err != nil {
			return nil, fmt.Errorf("task repo: unmarshal custom_fields: %w", err)
		}
	}
	if cf == nil {
		cf = map[string]any{}
	}

	var tags []string
	if len(r.Tags) > 0 {
		if err := json.Unmarshal(r.Tags, &tags); err != nil {
			return nil, fmt.Errorf("task repo: unmarshal tags: %w", err)
		}
	}
	if tags == nil {
		tags = []string{}
	}

	var deletedAt *time.Time
	if r.DeletedAt.Valid {
		deletedAt = &r.DeletedAt.Time
	}

	return &taskdom.Task{
		ID:           id,
		ProjectID:    pid,
		TaskNumber:   r.TaskNumber,
		TaskTypeID:   strPtrToUUIDPtr(r.TaskTypeID),
		StatusID:     strPtrToUUIDPtr(r.StatusID),
		SprintID:     strPtrToUUIDPtr(r.SprintID),
		ParentTaskID: strPtrToUUIDPtr(r.ParentTaskID),
		Title:        r.Title,
		Description:  r.Description,
		Importance:   r.Importance,
		StoryPoints:  r.StoryPoints,
		AssigneeID:   strPtrToUUIDPtr(r.AssigneeID),
		ReporterID:   strPtrToUUIDPtr(r.ReporterID),
		CustomFields: cf,
		StartDate:    r.StartDate,
		DueDate:      r.DueDate,
		Tags:         tags,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		DeletedAt:    deletedAt,
	}, nil
}

// --- Custom Field Definitions -----------------------------------------------

type customFieldDefinitionRecord struct {
	ID          string `gorm:"primarykey;type:uuid"`
	ProjectID   string `gorm:"type:uuid;not null;column:project_id"`
	FieldKey    string `gorm:"not null;column:field_key"`
	DisplayName string `gorm:"not null;column:display_name"`
	FieldType   string `gorm:"not null;column:field_type"`
	Options     []byte `gorm:"type:jsonb;column:options"`
	IsRequired  bool   `gorm:"not null;default:false;column:is_required"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (customFieldDefinitionRecord) TableName() string { return "custom_field_definitions" }

// ListCustomFieldDefinitions returns all custom field definitions for a
// project ordered by display_name.
func (r *TaskRepository) ListCustomFieldDefinitions(ctx context.Context, projectID uuid.UUID) ([]*taskdom.CustomFieldDefinition, error) {
	var records []customFieldDefinitionRecord
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Order("display_name ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("custom field repo: list: %w", err)
	}
	out := make([]*taskdom.CustomFieldDefinition, 0, len(records))
	for i := range records {
		f, err := toCustomFieldEntity(&records[i])
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// FindCustomFieldDefinitionByID returns the custom field definition with the
// given ID.
func (r *TaskRepository) FindCustomFieldDefinitionByID(ctx context.Context, id uuid.UUID) (*taskdom.CustomFieldDefinition, error) {
	var rec customFieldDefinitionRecord
	err := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, taskdom.ErrCustomFieldNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("custom field repo: find by id: %w", err)
	}
	return toCustomFieldEntity(&rec)
}

// CreateCustomFieldDefinition persists a new custom field definition.
func (r *TaskRepository) CreateCustomFieldDefinition(ctx context.Context, f *taskdom.CustomFieldDefinition) error {
	opts, err := marshalOptions(f.Options)
	if err != nil {
		return err
	}
	rec := &customFieldDefinitionRecord{
		ID:          f.ID.String(),
		ProjectID:   f.ProjectID.String(),
		FieldKey:    f.FieldKey,
		DisplayName: f.DisplayName,
		FieldType:   string(f.FieldType),
		Options:     opts,
		IsRequired:  f.IsRequired,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		if isUniqueViolation(err) {
			return taskdom.ErrCustomFieldKeyTaken
		}
		return fmt.Errorf("custom field repo: create: %w", err)
	}
	return nil
}

// UpdateCustomFieldDefinition persists changes to an existing custom field
// definition.
func (r *TaskRepository) UpdateCustomFieldDefinition(ctx context.Context, f *taskdom.CustomFieldDefinition) error {
	opts, err := marshalOptions(f.Options)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"display_name": f.DisplayName,
		"field_type":   string(f.FieldType),
		"options":      opts,
		"is_required":  f.IsRequired,
		"updated_at":   f.UpdatedAt,
	}
	res := r.db.WithContext(ctx).Model(&customFieldDefinitionRecord{}).
		Where("id = ?", f.ID.String()).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("custom field repo: update: %w", res.Error)
	}
	return nil
}

// DeleteCustomFieldDefinition removes a custom field definition by ID.
func (r *TaskRepository) DeleteCustomFieldDefinition(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&customFieldDefinitionRecord{}, "id = ?", id.String())
	if res.Error != nil {
		return fmt.Errorf("custom field repo: delete: %w", res.Error)
	}
	return nil
}

func toCustomFieldEntity(r *customFieldDefinitionRecord) (*taskdom.CustomFieldDefinition, error) {
	id, _ := uuid.Parse(r.ID)
	pid, _ := uuid.Parse(r.ProjectID)

	var opts []string
	if len(r.Options) > 0 {
		if err := json.Unmarshal(r.Options, &opts); err != nil {
			return nil, fmt.Errorf("custom field repo: unmarshal options: %w", err)
		}
	}
	if opts == nil {
		opts = []string{}
	}

	return &taskdom.CustomFieldDefinition{
		ID:          id,
		ProjectID:   pid,
		FieldKey:    r.FieldKey,
		DisplayName: r.DisplayName,
		FieldType:   taskdom.FieldType(r.FieldType),
		Options:     opts,
		IsRequired:  r.IsRequired,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}, nil
}

func marshalOptions(opts []string) ([]byte, error) {
	if opts == nil {
		opts = []string{}
	}
	b, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("custom field repo: marshal options: %w", err)
	}
	return b, nil
}

// --- helpers ----------------------------------------------------------------

func uuidSliceToStrSlice(ids []uuid.UUID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.String()
	}
	return s
}

func uuidPtrToStrPtr(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

func strPtrToUUIDPtr(s *string) *uuid.UUID {
	if s == nil {
		return nil
	}
	u, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &u
}
