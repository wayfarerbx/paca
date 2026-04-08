// Package sprintsvc implements sprint view application services.
package sprintsvc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
)

// ViewService is the concrete implementation of sprintdom.ViewService.
type ViewService struct {
	repo sprintdom.ViewRepository
}

// NewViewService returns a configured ViewService.
func NewViewService(repo sprintdom.ViewRepository) *ViewService {
	return &ViewService{repo: repo}
}

// ListViews returns all views for a sprint.
func (s *ViewService) ListViews(ctx context.Context, sprintID uuid.UUID) ([]*sprintdom.SprintView, error) {
	return s.repo.ListViews(ctx, sprintID)
}

// ListBacklogViews returns all product-backlog views for a project.
func (s *ViewService) ListBacklogViews(ctx context.Context, projectID uuid.UUID) ([]*sprintdom.SprintView, error) {
	return s.repo.ListBacklogViews(ctx, projectID)
}

// GetView returns the view with the given ID.
func (s *ViewService) GetView(ctx context.Context, id uuid.UUID) (*sprintdom.SprintView, error) {
	return s.repo.FindViewByID(ctx, id)
}

// CreateView creates a new view for the given sprint.
func (s *ViewService) CreateView(ctx context.Context, in sprintdom.CreateViewInput) (*sprintdom.SprintView, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, sprintdom.ErrViewNameInvalid
	}

	vt := in.ViewType
	if vt == "" {
		vt = sprintdom.ViewTypeTable
	}
	if !sprintdom.ValidViewTypes[vt] {
		return nil, sprintdom.ErrViewTypeInvalid
	}

	now := time.Now()
	v := &sprintdom.SprintView{
		ID:        uuid.New(),
		SprintID:  in.SprintID,
		ProjectID: in.ProjectID,
		Name:      name,
		ViewType:  vt,
		Config:    in.Config,
		Position:  in.Position,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateView(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateView updates the mutable fields of an existing view.
func (s *ViewService) UpdateView(ctx context.Context, id uuid.UUID, in sprintdom.UpdateViewInput) (*sprintdom.SprintView, error) {
	v, err := s.repo.FindViewByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, sprintdom.ErrViewNameInvalid
		}
		v.Name = name
	}
	if in.ViewType != nil {
		if !sprintdom.ValidViewTypes[*in.ViewType] {
			return nil, sprintdom.ErrViewTypeInvalid
		}
		v.ViewType = *in.ViewType
	}
	if in.Config != nil {
		v.Config = *in.Config
	}
	if in.Position != nil {
		v.Position = *in.Position
	}
	v.UpdatedAt = time.Now()

	if err := s.repo.UpdateView(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DeleteView removes a view by ID.  Deletion of the last remaining view is
// rejected with ErrViewIsLastView.
func (s *ViewService) DeleteView(ctx context.Context, id uuid.UUID) error {
	v, err := s.repo.FindViewByID(ctx, id)
	if err != nil {
		return err
	}

	var count int
	if v.SprintID != nil {
		count, err = s.repo.CountViews(ctx, *v.SprintID)
	} else {
		count, err = s.repo.CountBacklogViews(ctx, v.ProjectID)
	}
	if err != nil {
		return err
	}
	if count <= 1 {
		return sprintdom.ErrViewIsLastView
	}

	return s.repo.DeleteView(ctx, id)
}

// MoveTask updates the manual position of a task within a view.
func (s *ViewService) MoveTask(ctx context.Context, viewID uuid.UUID, in sprintdom.MoveTaskInput) error {
	if _, err := s.repo.FindViewByID(ctx, viewID); err != nil {
		return err
	}
	pos := &sprintdom.ViewTaskPosition{
		ID:       uuid.New(),
		ViewID:   viewID,
		TaskID:   in.TaskID,
		Position: in.Position,
		GroupKey: in.GroupKey,
	}
	return s.repo.UpsertTaskPosition(ctx, pos)
}

// ListTaskPositions returns the manual ordering for all tasks in a view.
func (s *ViewService) ListTaskPositions(ctx context.Context, viewID uuid.UUID) ([]*sprintdom.ViewTaskPosition, error) {
	if _, err := s.repo.FindViewByID(ctx, viewID); err != nil {
		return nil, err
	}
	return s.repo.ListTaskPositions(ctx, viewID)
}
