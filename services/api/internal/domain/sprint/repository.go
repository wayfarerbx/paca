package sprintdom

import (
	"context"

	"github.com/google/uuid"
)

// SprintRepository defines persistence operations for sprints.
type SprintRepository interface {
	ListSprints(ctx context.Context, projectID uuid.UUID) ([]*Sprint, error)
	FindSprintByID(ctx context.Context, id uuid.UUID) (*Sprint, error)
	CreateSprint(ctx context.Context, s *Sprint) error
	UpdateSprint(ctx context.Context, s *Sprint) error
	DeleteSprint(ctx context.Context, id uuid.UUID) error
}

// ViewRepository defines persistence operations for sprint views and manual
// task ordering within those views.
type ViewRepository interface {
	ListViews(ctx context.Context, sprintID uuid.UUID) ([]*SprintView, error)
	ListBacklogViews(ctx context.Context, projectID uuid.UUID) ([]*SprintView, error)
	FindViewByID(ctx context.Context, id uuid.UUID) (*SprintView, error)
	CreateView(ctx context.Context, v *SprintView) error
	UpdateView(ctx context.Context, v *SprintView) error
	DeleteView(ctx context.Context, id uuid.UUID) error

	// CountViews returns the number of views belonging to a sprint.  Used to
	// guard deletion of the last remaining view.
	CountViews(ctx context.Context, sprintID uuid.UUID) (int, error)

	// CountBacklogViews returns the number of product-backlog views for a project.
	CountBacklogViews(ctx context.Context, projectID uuid.UUID) (int, error)

	// UpsertTaskPosition stores or updates the manual position of a task
	// within a view.
	UpsertTaskPosition(ctx context.Context, pos *ViewTaskPosition) error

	// ListTaskPositions returns all manual positions for a view, ordered by
	// position ASC.
	ListTaskPositions(ctx context.Context, viewID uuid.UUID) ([]*ViewTaskPosition, error)
}
