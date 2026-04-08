// Package postgres — GORM implementation of sprintdom.ViewRepository.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- GORM models ------------------------------------------------------------

type sprintViewRecord struct {
	ID        string  `gorm:"primarykey;type:uuid"`
	SprintID  *string `gorm:"type:uuid;column:sprint_id"`
	ProjectID string  `gorm:"type:uuid;not null;column:project_id"`
	Name      string  `gorm:"not null"`
	ViewType  string  `gorm:"not null;column:view_type;default:table"`
	Config    []byte  `gorm:"type:jsonb;not null;column:config"`
	Position  int     `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (sprintViewRecord) TableName() string { return "sprint_views" }

type viewTaskPositionRecord struct {
	ID       string  `gorm:"primarykey;type:uuid"`
	ViewID   string  `gorm:"type:uuid;not null;column:view_id"`
	TaskID   string  `gorm:"type:uuid;not null;column:task_id"`
	Position int     `gorm:"not null;default:0"`
	GroupKey *string `gorm:"type:text;column:group_key"`
}

func (viewTaskPositionRecord) TableName() string { return "view_task_positions" }

// --- Repository struct -------------------------------------------------------

// ViewRepository is the GORM implementation of sprintdom.ViewRepository.
type ViewRepository struct {
	db *gorm.DB
}

// NewViewRepository returns a new ViewRepository.
func NewViewRepository(db *gorm.DB) *ViewRepository {
	return &ViewRepository{db: db}
}

// --- View methods -----------------------------------------------------------

// ListViews returns all views for a sprint ordered by position.
func (r *ViewRepository) ListViews(ctx context.Context, sprintID uuid.UUID) ([]*sprintdom.SprintView, error) {
	var records []sprintViewRecord
	if err := r.db.WithContext(ctx).
		Where("sprint_id = ?", sprintID.String()).
		Order("position ASC, created_at ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("view repo: list: %w", err)
	}
	out := make([]*sprintdom.SprintView, 0, len(records))
	for i := range records {
		v, err := toViewEntity(&records[i])
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// FindViewByID returns the view with the given ID.
func (r *ViewRepository) FindViewByID(ctx context.Context, id uuid.UUID) (*sprintdom.SprintView, error) {
	var rec sprintViewRecord
	err := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, sprintdom.ErrViewNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("view repo: find by id: %w", err)
	}
	return toViewEntity(&rec)
}

// ListBacklogViews returns all product-backlog views for a project ordered by position.
func (r *ViewRepository) ListBacklogViews(ctx context.Context, projectID uuid.UUID) ([]*sprintdom.SprintView, error) {
	var records []sprintViewRecord
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND sprint_id IS NULL", projectID.String()).
		Order("position ASC, created_at ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("view repo: list backlog: %w", err)
	}
	out := make([]*sprintdom.SprintView, 0, len(records))
	for i := range records {
		v, err := toViewEntity(&records[i])
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// CreateView persists a new sprint view.
func (r *ViewRepository) CreateView(ctx context.Context, v *sprintdom.SprintView) error {
	configBytes, err := json.Marshal(v.Config)
	if err != nil {
		return fmt.Errorf("view repo: marshal config: %w", err)
	}
	var sprintIDStr *string
	if v.SprintID != nil {
		s := v.SprintID.String()
		sprintIDStr = &s
	}
	rec := &sprintViewRecord{
		ID:        v.ID.String(),
		SprintID:  sprintIDStr,
		ProjectID: v.ProjectID.String(),
		Name:      v.Name,
		ViewType:  string(v.ViewType),
		Config:    configBytes,
		Position:  v.Position,
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		return fmt.Errorf("view repo: create: %w", err)
	}
	return nil
}

// UpdateView persists changes to an existing sprint view.
func (r *ViewRepository) UpdateView(ctx context.Context, v *sprintdom.SprintView) error {
	configBytes, err := json.Marshal(v.Config)
	if err != nil {
		return fmt.Errorf("view repo: marshal config: %w", err)
	}
	updates := map[string]any{
		"name":       v.Name,
		"view_type":  string(v.ViewType),
		"config":     configBytes,
		"position":   v.Position,
		"updated_at": v.UpdatedAt,
	}
	res := r.db.WithContext(ctx).Model(&sprintViewRecord{}).Where("id = ?", v.ID.String()).Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("view repo: update: %w", res.Error)
	}
	return nil
}

// DeleteView removes a sprint view by ID.
func (r *ViewRepository) DeleteView(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&sprintViewRecord{}, "id = ?", id.String())
	if res.Error != nil {
		return fmt.Errorf("view repo: delete: %w", res.Error)
	}
	return nil
}

// CountViews returns the number of views for a sprint.
func (r *ViewRepository) CountViews(ctx context.Context, sprintID uuid.UUID) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&sprintViewRecord{}).
		Where("sprint_id = ?", sprintID.String()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("view repo: count: %w", err)
	}
	return int(count), nil
}

// CountBacklogViews returns the number of product-backlog views for a project.
func (r *ViewRepository) CountBacklogViews(ctx context.Context, projectID uuid.UUID) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&sprintViewRecord{}).
		Where("project_id = ? AND sprint_id IS NULL", projectID.String()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("view repo: count backlog: %w", err)
	}
	return int(count), nil
}

// --- Task position methods --------------------------------------------------

// UpsertTaskPosition stores or updates the manual position of a task in a view.
func (r *ViewRepository) UpsertTaskPosition(ctx context.Context, pos *sprintdom.ViewTaskPosition) error {
	rec := &viewTaskPositionRecord{
		ID:       pos.ID.String(),
		ViewID:   pos.ViewID.String(),
		TaskID:   pos.TaskID.String(),
		Position: pos.Position,
		GroupKey: pos.GroupKey,
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "view_id"}, {Name: "task_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"position", "group_key"}),
	}).Create(rec)
	if result.Error != nil {
		return fmt.Errorf("view repo: upsert task position: %w", result.Error)
	}
	return nil
}

// ListTaskPositions returns all manual positions for a view ordered by position ASC.
func (r *ViewRepository) ListTaskPositions(ctx context.Context, viewID uuid.UUID) ([]*sprintdom.ViewTaskPosition, error) {
	var records []viewTaskPositionRecord
	if err := r.db.WithContext(ctx).
		Where("view_id = ?", viewID.String()).
		Order("position ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("view repo: list task positions: %w", err)
	}
	out := make([]*sprintdom.ViewTaskPosition, 0, len(records))
	for i := range records {
		out = append(out, toTaskPositionEntity(&records[i]))
	}
	return out, nil
}

// --- Entity converters ------------------------------------------------------

func toViewEntity(r *sprintViewRecord) (*sprintdom.SprintView, error) {
	id, _ := uuid.Parse(r.ID)
	pid, _ := uuid.Parse(r.ProjectID)
	var sid *uuid.UUID
	if r.SprintID != nil {
		parsed, _ := uuid.Parse(*r.SprintID)
		sid = &parsed
	}
	var cfg sprintdom.ViewConfig
	if len(r.Config) > 0 {
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			return nil, fmt.Errorf("view repo: unmarshal config: %w", err)
		}
	}
	return &sprintdom.SprintView{
		ID:        id,
		SprintID:  sid,
		ProjectID: pid,
		Name:      r.Name,
		ViewType:  sprintdom.ViewType(r.ViewType),
		Config:    cfg,
		Position:  r.Position,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

func toTaskPositionEntity(r *viewTaskPositionRecord) *sprintdom.ViewTaskPosition {
	id, _ := uuid.Parse(r.ID)
	vid, _ := uuid.Parse(r.ViewID)
	tid, _ := uuid.Parse(r.TaskID)
	return &sprintdom.ViewTaskPosition{
		ID:       id,
		ViewID:   vid,
		TaskID:   tid,
		Position: r.Position,
		GroupKey: r.GroupKey,
	}
}
