package sprintdom

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SprintService defines sprint use cases.
type SprintService interface {
	ListSprints(ctx context.Context, projectID uuid.UUID) ([]*Sprint, error)
	GetSprint(ctx context.Context, id uuid.UUID) (*Sprint, error)
	CreateSprint(ctx context.Context, in CreateSprintInput) (*Sprint, error)
	UpdateSprint(ctx context.Context, id uuid.UUID, in UpdateSprintInput) (*Sprint, error)
	DeleteSprint(ctx context.Context, id uuid.UUID) error
}

// CreateSprintInput carries fields required to create a sprint.
type CreateSprintInput struct {
	ProjectID uuid.UUID
	Name      string
	StartDate *time.Time
	EndDate   *time.Time
	Goal      *string
	Status    SprintStatus
}

// UpdateSprintInput carries mutable sprint fields.
type UpdateSprintInput struct {
	Name      string
	StartDate *time.Time
	EndDate   *time.Time
	Goal      *string
	Status    *SprintStatus
}

// ViewService defines use cases for sprint views and manual task ordering.
type ViewService interface {
	ListViews(ctx context.Context, sprintID uuid.UUID) ([]*SprintView, error)
	ListBacklogViews(ctx context.Context, projectID uuid.UUID) ([]*SprintView, error)
	GetView(ctx context.Context, id uuid.UUID) (*SprintView, error)
	CreateView(ctx context.Context, in CreateViewInput) (*SprintView, error)
	UpdateView(ctx context.Context, id uuid.UUID, in UpdateViewInput) (*SprintView, error)
	DeleteView(ctx context.Context, id uuid.UUID) error

	// MoveTask updates the manual position of a task within a view.
	MoveTask(ctx context.Context, viewID uuid.UUID, in MoveTaskInput) error

	// ListTaskPositions returns the manual ordering for all tasks in a view.
	ListTaskPositions(ctx context.Context, viewID uuid.UUID) ([]*ViewTaskPosition, error)
}

// CreateViewInput carries fields required to create a sprint view.
// SprintID is nil for product-backlog views; ProjectID is always required.
type CreateViewInput struct {
	SprintID  *uuid.UUID
	ProjectID uuid.UUID
	Name      string
	ViewType  ViewType
	Config    ViewConfig
	Position  int
}

// UpdateViewInput carries mutable view fields.
type UpdateViewInput struct {
	Name     *string
	ViewType *ViewType
	Config   *ViewConfig
	Position *int
}

// MoveTaskInput requests a change to a task's manual position in a view.
type MoveTaskInput struct {
	TaskID   uuid.UUID
	Position int
	GroupKey *string
}
