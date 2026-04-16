// Package sprintsvc implements sprint application services.
package sprintsvc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	taskdom "github.com/paca/api/internal/domain/task"
)

// Service is the concrete implementation of sprintdom.SprintService.
type Service struct {
	repo     sprintdom.SprintRepository
	taskRepo taskdom.TaskRepository
}

// New returns a configured sprint service.
func New(repo sprintdom.SprintRepository, taskRepo taskdom.TaskRepository) *Service {
	return &Service{repo: repo, taskRepo: taskRepo}
}

// ListSprints returns all sprints for a project.
func (s *Service) ListSprints(ctx context.Context, projectID uuid.UUID) ([]*sprintdom.Sprint, error) {
	return s.repo.ListSprints(ctx, projectID)
}

// GetSprint returns the sprint with the given ID.
func (s *Service) GetSprint(ctx context.Context, id uuid.UUID) (*sprintdom.Sprint, error) {
	return s.repo.FindSprintByID(ctx, id)
}

// CreateSprint creates a new sprint for the given project.
func (s *Service) CreateSprint(ctx context.Context, in sprintdom.CreateSprintInput) (*sprintdom.Sprint, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, sprintdom.ErrSprintNameInvalid
	}

	status := in.Status
	if status == "" {
		status = sprintdom.SprintStatusPlanned
	}
	if !sprintdom.ValidSprintStatuses[status] {
		return nil, sprintdom.ErrSprintStatusInvalid
	}

	now := time.Now()
	sp := &sprintdom.Sprint{
		ID:        uuid.New(),
		ProjectID: in.ProjectID,
		Name:      name,
		StartDate: in.StartDate,
		EndDate:   in.EndDate,
		Goal:      in.Goal,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateSprint(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// UpdateSprint updates the mutable fields of an existing sprint.
func (s *Service) UpdateSprint(ctx context.Context, id uuid.UUID, in sprintdom.UpdateSprintInput) (*sprintdom.Sprint, error) {
	sp, err := s.repo.FindSprintByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name := strings.TrimSpace(in.Name); name != "" {
		sp.Name = name
	}
	sp.StartDate = in.StartDate
	sp.EndDate = in.EndDate
	sp.Goal = in.Goal
	if in.Status != nil {
		if !sprintdom.ValidSprintStatuses[*in.Status] {
			return nil, sprintdom.ErrSprintStatusInvalid
		}
		sp.Status = *in.Status
	}
	sp.UpdatedAt = time.Now()

	if err := s.repo.UpdateSprint(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// DeleteSprint removes a sprint by ID.
func (s *Service) DeleteSprint(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindSprintByID(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteSprint(ctx, id)
}

// CompleteSprint bulk-moves all non-done tasks out of the sprint and marks
// the sprint as completed in two sequential writes.  Tasks whose status
// has category "done" are left in place.
func (s *Service) CompleteSprint(ctx context.Context, id uuid.UUID, in sprintdom.CompleteSprintInput) (*sprintdom.Sprint, error) {
	sp, err := s.repo.FindSprintByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp.Status == sprintdom.SprintStatusCompleted {
		return nil, sprintdom.ErrSprintAlreadyComplete
	}

	// Move non-done tasks first so a subsequent failure leaves the sprint
	// in its original state (retrying the complete is then still possible).
	if err := s.taskRepo.BulkMoveSprintTasks(ctx, sp.ProjectID, id, in.MoveToSprintID); err != nil {
		return nil, err
	}

	sp.Status = sprintdom.SprintStatusCompleted
	sp.UpdatedAt = time.Now()
	if err := s.repo.UpdateSprint(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}
