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
	"gorm.io/gorm/clause"
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

// taskWithPositionRow is a flat struct for scanning the view_position LEFT JOIN
// result. It explicitly lists every column from taskRecord plus vtp_position so
// that GORM maps columns by name without the embedded-model relationship logic
// that fires when the embedded type has a TableName() method.
type taskWithPositionRow struct {
	ID           string          `gorm:"column:id"`
	ProjectID    string          `gorm:"column:project_id"`
	TaskNumber   int64           `gorm:"column:task_number"`
	TaskTypeID   *string         `gorm:"column:task_type_id"`
	StatusID     *string         `gorm:"column:status_id"`
	SprintID     *string         `gorm:"column:sprint_id"`
	ParentTaskID *string         `gorm:"column:parent_task_id"`
	Title        string          `gorm:"column:title"`
	Description  json.RawMessage `gorm:"column:description"`
	Importance   int             `gorm:"column:importance"`
	StoryPoints  *int            `gorm:"column:story_points"`
	AssigneeID   *string         `gorm:"column:assignee_id"`
	ReporterID   *string         `gorm:"column:reporter_id"`
	CustomFields []byte          `gorm:"column:custom_fields"`
	StartDate    *time.Time      `gorm:"column:start_date"`
	DueDate      *time.Time      `gorm:"column:due_date"`
	Tags         []byte          `gorm:"column:tags"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt  `gorm:"column:deleted_at"`
	VTPPosition  *float64        `gorm:"column:vtp_position"`
}

func (r *taskWithPositionRow) asTaskRecord() taskRecord {
	return taskRecord{
		ID:           r.ID,
		ProjectID:    r.ProjectID,
		TaskNumber:   r.TaskNumber,
		TaskTypeID:   r.TaskTypeID,
		StatusID:     r.StatusID,
		SprintID:     r.SprintID,
		ParentTaskID: r.ParentTaskID,
		Title:        r.Title,
		Description:  r.Description,
		Importance:   r.Importance,
		StoryPoints:  r.StoryPoints,
		AssigneeID:   r.AssigneeID,
		ReporterID:   r.ReporterID,
		CustomFields: r.CustomFields,
		StartDate:    r.StartDate,
		DueDate:      r.DueDate,
		Tags:         r.Tags,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		DeletedAt:    r.DeletedAt,
	}
}

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

// applyTaskFilter adds WHERE predicates for all TaskFilter fields except
// CursorAfter (which is handled separately by applyCursorWhere).
// It is shared by ListTasks, CountTasks, and SumTaskField so that adding a new
// filter dimension only requires a single change.
func applyTaskFilter(q *gorm.DB, filter taskdom.TaskFilter) *gorm.DB {
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

	switch {
	case filter.AssigneeNull && len(filter.AssigneeIDs) > 0:
		q = q.Where("assignee_id IS NULL OR assignee_id IN ?", uuidSliceToStrSlice(filter.AssigneeIDs))
	case filter.AssigneeNull:
		q = q.Where("assignee_id IS NULL")
	case len(filter.AssigneeIDs) > 0:
		q = q.Where("assignee_id IN ?", uuidSliceToStrSlice(filter.AssigneeIDs))
	case filter.AssigneeID != nil:
		q = q.Where("assignee_id = ?", filter.AssigneeID.String())
	}

	switch {
	case filter.TaskTypeNull:
		q = q.Where("task_type_id IS NULL")
	case len(filter.TaskTypeIDs) > 0:
		q = q.Where("task_type_id IN ?", uuidSliceToStrSlice(filter.TaskTypeIDs))
	}

	return q
}

// applyTaskSort adds the SELECT extension, JOIN (when needed), and ORDER BY
// clause to q based on the sort configuration.
// The secondary sort on (created_at ASC, id ASC) guarantees stable pagination.
func applyTaskSort(q *gorm.DB, sort taskdom.TaskSort) *gorm.DB {
	switch sort.By {
	case "view_position":
		// LEFT JOIN view_task_positions so the position value is available for
		// ORDER BY and cursor pagination. Tasks without a row sort last (NULLS
		// LAST), with created_at as the tiebreaker.
		if sort.ViewID != nil {
			q = q.
				Select("tasks.*, vtp.position AS vtp_position").
				Joins("LEFT JOIN view_task_positions vtp ON vtp.task_id = tasks.id AND vtp.view_id = ?", sort.ViewID)
		}
		return q.Order("vtp.position ASC NULLS LAST, tasks.created_at ASC, tasks.id ASC")
	case "importance":
		return q.Order("importance DESC, created_at ASC, id ASC")
	case "title":
		return q.Order("title ASC, created_at ASC, id ASC")
	case "story_points":
		return q.Order("story_points DESC NULLS LAST, created_at ASC, id ASC")
	case "start_date":
		return q.Order("start_date ASC NULLS LAST, created_at ASC, id ASC")
	case "due_date":
		return q.Order("due_date ASC NULLS LAST, created_at ASC, id ASC")
	default:
		if sort.By != "" && sort.CFType != "" {
			return applyCFTaskSort(q, sort)
		}
		// sort.By == "created" (the public API key) falls here and uses created_at ASC.
		return q.Order("created_at ASC, id ASC")
	}
}

// applyCFTaskSort applies ORDER BY for a custom-field sort using JSONB expressions.
func applyCFTaskSort(q *gorm.DB, sort taskdom.TaskSort) *gorm.DB {
	switch sort.CFType {
	case "number":
		return q.Order(clause.Expr{
			SQL:  "(custom_fields->>?)::numeric ASC NULLS LAST, created_at ASC, id ASC",
			Vars: []interface{}{sort.By},
		})
	case "date":
		return q.Order(clause.Expr{
			SQL:  "(custom_fields->>?)::date ASC NULLS LAST, created_at ASC, id ASC",
			Vars: []interface{}{sort.By},
		})
	case "select":
		if len(sort.CFOpts) == 0 {
			return q.Order("created_at ASC, id ASC")
		}
		caseSQL, caseArgs := buildCFSelectCaseSQL(sort.By, sort.CFOpts)
		return q.Order(clause.Expr{
			SQL:  caseSQL + " ASC, created_at ASC, id ASC",
			Vars: caseArgs,
		})
	}
	return q.Order("created_at ASC, id ASC")
}

// buildCFSelectCaseSQL builds a parameterized CASE expression for select CF ordering.
// Returns the SQL fragment and the corresponding bound args.
func buildCFSelectCaseSQL(key string, opts []string) (string, []interface{}) {
	args := []interface{}{key}
	sql := "CASE custom_fields->>?"
	for i, opt := range opts {
		sql += fmt.Sprintf(" WHEN ? THEN %d", i)
		args = append(args, opt)
	}
	sql += " ELSE 9999 END"
	return sql, args
}

// applyCursorWhere adds the keyset-pagination WHERE predicate that skips rows
// already returned on the previous page.  The predicate is derived from the
// sort key stored inside the cursor so it correctly handles each sort order
// (including DESC, NULLS LAST, and custom JSONB field variants).
func applyCursorWhere(q *gorm.DB, cur *taskdom.TaskCursor, sort taskdom.TaskSort) *gorm.DB {
	ca := cur.CreatedAt.UTC()
	id := cur.ID

	switch cur.SortBy {
	case "view_position":
		// Positioned tasks (vtp.position IS NOT NULL) sort before unpositioned ones.
		// Use a full keyset predicate so equal positions are handled correctly:
		//   position strictly greater, OR same position with a later (created_at, id), OR unpositioned.
		// Cursor for an unpositioned task: only unpositioned tasks after (created_at, id).
		if cur.SortNumVal != nil {
			pos := *cur.SortNumVal
			return q.Where(
				"vtp.position > ? OR (vtp.position = ? AND (tasks.created_at, tasks.id) > (?, ?)) OR vtp.position IS NULL",
				pos, pos, ca, id,
			)
		}
		return q.Where("vtp.position IS NULL AND (tasks.created_at, tasks.id) > (?, ?)", ca, id)
	case "importance":
		imp := int64(*cur.SortNumVal)
		return q.Where(
			"importance < ? OR (importance = ? AND (created_at, id) > (?, ?))",
			imp, imp, ca, id,
		)
	case "title":
		t := *cur.SortStrVal
		return q.Where(
			"title > ? OR (title = ? AND (created_at, id) > (?, ?))",
			t, t, ca, id,
		)
	case "story_points":
		if cur.SortNumVal != nil {
			sp := int64(*cur.SortNumVal)
			return q.Where(
				"story_points < ? OR (story_points = ? AND (created_at, id) > (?, ?)) OR story_points IS NULL",
				sp, sp, ca, id,
			)
		}
		return q.Where("story_points IS NULL AND (created_at, id) > (?, ?)", ca, id)
	case "start_date":
		if cur.SortTimeVal != nil {
			d := *cur.SortTimeVal
			return q.Where(
				"start_date > ?::date OR (start_date = ?::date AND (created_at, id) > (?, ?)) OR start_date IS NULL",
				d, d, ca, id,
			)
		}
		return q.Where("start_date IS NULL AND (created_at, id) > (?, ?)", ca, id)
	case "due_date":
		if cur.SortTimeVal != nil {
			d := *cur.SortTimeVal
			return q.Where(
				"due_date > ?::date OR (due_date = ?::date AND (created_at, id) > (?, ?)) OR due_date IS NULL",
				d, d, ca, id,
			)
		}
		return q.Where("due_date IS NULL AND (created_at, id) > (?, ?)", ca, id)
	default:
		// Custom field cursor — use sort metadata to build keyset predicate.
		if cur.SortBy != "" && sort.CFType != "" {
			return applyCFCursorWhere(q, cur, sort, ca, id)
		}
		return q.Where("(created_at, id) > (?, ?)", ca, id)
	}
}

// applyCFCursorWhere builds the keyset WHERE predicate for custom-field sorts.
func applyCFCursorWhere(q *gorm.DB, cur *taskdom.TaskCursor, sort taskdom.TaskSort, ca interface{}, id string) *gorm.DB {
	switch sort.CFType {
	case "number":
		if cur.SortNumVal != nil {
			v := *cur.SortNumVal
			return q.Where(clause.Expr{
				SQL:  "(custom_fields->>?)::numeric > ? OR ((custom_fields->>?)::numeric = ? AND (created_at, id) > (?, ?)) OR custom_fields->>? IS NULL",
				Vars: []interface{}{sort.By, v, sort.By, v, ca, id, sort.By},
			})
		}
		return q.Where(clause.Expr{
			SQL:  "custom_fields->>? IS NULL AND (created_at, id) > (?, ?)",
			Vars: []interface{}{sort.By, ca, id},
		})
	case "date":
		if cur.SortTimeVal != nil {
			d := *cur.SortTimeVal
			return q.Where(clause.Expr{
				SQL:  "(custom_fields->>?)::date > ?::date OR ((custom_fields->>?)::date = ?::date AND (created_at, id) > (?, ?)) OR custom_fields->>? IS NULL",
				Vars: []interface{}{sort.By, d, sort.By, d, ca, id, sort.By},
			})
		}
		return q.Where(clause.Expr{
			SQL:  "custom_fields->>? IS NULL AND (created_at, id) > (?, ?)",
			Vars: []interface{}{sort.By, ca, id},
		})
	case "select":
		// Map cursor value to its option index (unknown → 9999).
		curIdx := 9999
		if cur.SortStrVal != nil {
			for i, opt := range sort.CFOpts {
				if opt == *cur.SortStrVal {
					curIdx = i
					break
				}
			}
		}
		caseSQL, caseArgs := buildCFSelectCaseSQL(sort.By, sort.CFOpts)
		// Build: (CASE ...) > curIdx OR ((CASE ...) = curIdx AND (created_at, id) > (ca, id))
		// The CASE expression appears twice, so caseArgs must be duplicated.
		allArgs := make([]interface{}, 0, len(caseArgs)*2+4)
		allArgs = append(allArgs, caseArgs...)
		allArgs = append(allArgs, curIdx)
		allArgs = append(allArgs, caseArgs...)
		allArgs = append(allArgs, curIdx, ca, id)
		return q.Where(clause.Expr{
			SQL:  fmt.Sprintf("(%s) > ? OR ((%s) = ? AND (created_at, id) > (?, ?))", caseSQL, caseSQL),
			Vars: allArgs,
		})
	}
	return q.Where("(created_at, id) > (?, ?)", ca, id)
}

// ListTasks returns a page of tasks with optional filter.
// When filter.CursorAfter is nil, returns from the beginning.
// When set, returns tasks strictly after the cursor position.
// hasMore is true when a next page exists beyond the returned slice.
func (r *TaskRepository) ListTasks(ctx context.Context, projectID uuid.UUID, filter taskdom.TaskFilter, limit int, sort taskdom.TaskSort) ([]*taskdom.Task, bool, error) {
	q := r.db.WithContext(ctx).Model(&taskRecord{}).
		Where("project_id = ?", projectID.String())
	q = applyTaskFilter(q, filter)

	if filter.CursorAfter != nil {
		cur, err := taskdom.DecodeTaskCursor(*filter.CursorAfter)
		if err != nil {
			return nil, false, fmt.Errorf("task repo: invalid cursor: %w", err)
		}
		q = applyCursorWhere(q, cur, sort)
	}

	// applyTaskSort adds the JOIN + SELECT extension for view_position and the
	// ORDER BY for all sort keys — must happen before Limit so the DB sorts the
	// full result set before pagination cuts it.
	sortedQ := applyTaskSort(q, sort)

	// view_position uses a flat scan struct (taskWithPositionRow) instead of
	// taskRecord because GORM v2 treats embedded structs with TableName() as
	// relationships and silently skips their fields during Scan.
	if sort.By == "view_position" && sort.ViewID != nil {
		var rows []taskWithPositionRow
		if err := sortedQ.Limit(limit + 1).Scan(&rows).Error; err != nil {
			return nil, false, fmt.Errorf("task repo: list (view_position): %w", err)
		}
		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}
		tasks := make([]*taskdom.Task, 0, len(rows))
		for i := range rows {
			rec := rows[i].asTaskRecord()
			t, err := toTaskEntity(&rec)
			if err != nil {
				return nil, false, err
			}
			t.ViewPosition = rows[i].VTPPosition
			tasks = append(tasks, t)
		}
		return tasks, hasMore, nil
	}

	var records []taskRecord
	if err := sortedQ.Limit(limit + 1).Find(&records).Error; err != nil {
		return nil, false, fmt.Errorf("task repo: list: %w", err)
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	tasks := make([]*taskdom.Task, 0, len(records))
	for i := range records {
		t, err := toTaskEntity(&records[i])
		if err != nil {
			return nil, false, err
		}
		tasks = append(tasks, t)
	}
	return tasks, hasMore, nil
}

// CountTasks returns the total number of tasks matching filter for a project,
// ignoring cursor-based pagination so the result reflects the true total.
func (r *TaskRepository) CountTasks(ctx context.Context, projectID uuid.UUID, filter taskdom.TaskFilter) (int64, error) {
	q := r.db.WithContext(ctx).Model(&taskRecord{}).
		Where("project_id = ?", projectID.String())
	q = applyTaskFilter(q, filter)

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("task repo: count: %w", err)
	}
	return count, nil
}

// SumTaskField sums a numeric task field across all tasks matching filter,
// ignoring cursor-based pagination so the result reflects the true total.
// fieldKey must be "story_points" or a custom field key (stored in the custom_fields JSONB column).
func (r *TaskRepository) SumTaskField(ctx context.Context, projectID uuid.UUID, filter taskdom.TaskFilter, fieldKey string) (float64, error) {
	q := r.db.WithContext(ctx).Model(&taskRecord{}).
		Where("project_id = ?", projectID.String())
	q = applyTaskFilter(q, filter)

	var sum float64
	var err error
	if fieldKey == "story_points" {
		err = q.Select("COALESCE(SUM(story_points), 0)").Scan(&sum).Error
	} else {
		err = q.Select("COALESCE(SUM((custom_fields->>?)::numeric), 0)", fieldKey).Scan(&sum).Error
	}
	if err != nil {
		return 0, fmt.Errorf("task repo: sum field %q: %w", fieldKey, err)
	}
	return sum, nil
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
