package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	taskdom "github.com/paca/api/internal/domain/task"
	"gorm.io/gorm"
)

// --- GORM model -------------------------------------------------------------

type taskActivityRecord struct {
	ID           string          `gorm:"primarykey;type:uuid"`
	TaskID       string          `gorm:"type:uuid;not null;column:task_id"`
	ActorID      *string         `gorm:"type:uuid;column:actor_id"`
	ActivityType string          `gorm:"not null;column:activity_type"`
	Content      json.RawMessage `gorm:"type:jsonb;not null;column:content"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `gorm:"column:deleted_at"`

	// Joined from the users table (populated by explicit SELECT with JOIN).
	ActorFullName *string `gorm:"->;column:actor_full_name"`
	ActorUsername *string `gorm:"->;column:actor_username"`
}

func (taskActivityRecord) TableName() string { return "task_activities" }

// --- Repository struct -------------------------------------------------------

// TaskActivityRepository is the GORM implementation of taskdom.ActivityRepository.
type TaskActivityRepository struct {
	db *gorm.DB
}

// NewTaskActivityRepository returns a new TaskActivityRepository backed by db.
func NewTaskActivityRepository(db *gorm.DB) *TaskActivityRepository {
	return &TaskActivityRepository{db: db}
}

// --- Mapping helpers --------------------------------------------------------

func activityFromRecord(r taskActivityRecord) *taskdom.Activity {
	a := &taskdom.Activity{
		ID:           uuid.MustParse(r.ID),
		TaskID:       uuid.MustParse(r.TaskID),
		ActivityType: taskdom.ActivityType(r.ActivityType),
		Content:      r.Content,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		DeletedAt:    r.DeletedAt,
	}
	if r.ActorID != nil {
		id := uuid.MustParse(*r.ActorID)
		a.ActorID = &id
	}
	if r.ActorFullName != nil {
		a.ActorName = *r.ActorFullName
	}
	if r.ActorUsername != nil {
		a.ActorUsername = *r.ActorUsername
	}
	return a
}

// --- CRUD -------------------------------------------------------------------

// listQuery returns a base query that LEFT JOINs users for actor name resolution.
func (r *TaskActivityRepository) listQuery() *gorm.DB {
	return r.db.Table("task_activities ta").
		Select("ta.*, u.full_name AS actor_full_name, u.username AS actor_username").
		Joins("LEFT JOIN users u ON u.id = ta.actor_id")
}

// ListActivities returns all non-deleted activities for a task, oldest first.
func (r *TaskActivityRepository) ListActivities(_ context.Context, taskID uuid.UUID) ([]*taskdom.Activity, error) {
	var records []taskActivityRecord
	err := r.listQuery().
		Where("ta.task_id = ? AND ta.deleted_at IS NULL", taskID.String()).
		Order("ta.created_at ASC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	out := make([]*taskdom.Activity, 0, len(records))
	for _, rec := range records {
		out = append(out, activityFromRecord(rec))
	}
	return out, nil
}

// FindActivityByID returns a single activity (including soft-deleted).
func (r *TaskActivityRepository) FindActivityByID(_ context.Context, id uuid.UUID) (*taskdom.Activity, error) {
	var rec taskActivityRecord
	err := r.listQuery().
		Where("ta.id = ?", id.String()).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, taskdom.ErrActivityNotFound
		}
		return nil, err
	}
	return activityFromRecord(rec), nil
}

// CreateActivity persists a new activity record.
func (r *TaskActivityRepository) CreateActivity(ctx context.Context, a *taskdom.Activity) error {
	content := a.Content
	if content == nil {
		content = json.RawMessage("{}")
	}
	rec := taskActivityRecord{
		ID:           a.ID.String(),
		TaskID:       a.TaskID.String(),
		ActivityType: string(a.ActivityType),
		Content:      content,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
	if a.ActorID != nil {
		s := a.ActorID.String()
		rec.ActorID = &s
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

// UpdateActivity updates the content and updated_at of an existing activity.
func (r *TaskActivityRepository) UpdateActivity(ctx context.Context, a *taskdom.Activity) error {
	return r.db.WithContext(ctx).
		Table("task_activities").
		Where("id = ?", a.ID.String()).
		Updates(map[string]any{
			"content":    a.Content,
			"updated_at": a.UpdatedAt,
		}).Error
}

// DeleteActivity soft-deletes an activity by setting deleted_at.
func (r *TaskActivityRepository) DeleteActivity(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Table("task_activities").
		Where("id = ?", id.String()).
		Update("deleted_at", now).Error
}
