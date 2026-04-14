package handler

import (
	"context"

	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	taskdom "github.com/paca/api/internal/domain/task"
)

// taskTypeLister is the minimal task-service surface needed to seed default
// view filters from the project's built-in task types.
type taskTypeLister interface {
	ListTaskTypes(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskType, error)
}

func loadTaskTypes(ctx context.Context, taskSvc taskTypeLister, projectID uuid.UUID) ([]*taskdom.TaskType, error) {
	if taskSvc == nil {
		return nil, nil
	}
	return taskSvc.ListTaskTypes(ctx, projectID)
}

func filterTaskTypeIDs(taskTypes []*taskdom.TaskType, include func(*taskdom.TaskType) bool) []string {
	ids := make([]string, 0, len(taskTypes))
	for _, taskType := range taskTypes {
		if taskType != nil && include(taskType) {
			ids = append(ids, taskType.ID.String())
		}
	}
	return ids
}

func defaultNonSystemTaskTypeIDs(taskTypes []*taskdom.TaskType) []string {
	return filterTaskTypeIDs(taskTypes, func(taskType *taskdom.TaskType) bool {
		return !taskType.IsSystem
	})
}

func defaultEpicTaskTypeIDs(taskTypes []*taskdom.TaskType) []string {
	return filterTaskTypeIDs(taskTypes, func(taskType *taskdom.TaskType) bool {
		return taskType.IsSystem && taskType.Name == "Epic"
	})
}

func newDefaultFilters(taskTypeIDs, sprintIDs []string) *sprintdom.ViewFilters {
	filters := &sprintdom.ViewFilters{}
	if len(taskTypeIDs) > 0 {
		filters.TaskTypeIDs = taskTypeIDs
	}
	if len(sprintIDs) > 0 {
		filters.SprintIDs = sprintIDs
	}
	if len(filters.TaskTypeIDs) == 0 && len(filters.SprintIDs) == 0 {
		return nil
	}
	return filters
}

func defaultProjectViewInputs(projectID uuid.UUID, taskTypes []*taskdom.TaskType) []sprintdom.CreateViewInput {
	nonSystemTaskTypeIDs := defaultNonSystemTaskTypeIDs(taskTypes)
	epicTaskTypeIDs := defaultEpicTaskTypeIDs(taskTypes)
	return []sprintdom.CreateViewInput{
		{
			ProjectID:   projectID,
			Name:        "Table",
			ViewType:    sprintdom.ViewTypeTable,
			Position:    0,
			ViewContext: sprintdom.ViewContextBacklog,
			Config: sprintdom.ViewConfig{
				ColumnBy: "sprint",
				Filters:  newDefaultFilters(nonSystemTaskTypeIDs, nil),
			},
		},
		{
			ProjectID:   projectID,
			Name:        "Roadmap",
			ViewType:    sprintdom.ViewTypeRoadmap,
			Position:    0,
			ViewContext: sprintdom.ViewContextTimeline,
			Config: sprintdom.ViewConfig{
				Filters: newDefaultFilters(epicTaskTypeIDs, nil),
			},
		},
	}
}

func defaultSprintViewInputs(projectID, sprintID uuid.UUID, taskTypes []*taskdom.TaskType) []sprintdom.CreateViewInput {
	nonSystemTaskTypeIDs := defaultNonSystemTaskTypeIDs(taskTypes)
	sprintIDs := []string{sprintID.String()}
	return []sprintdom.CreateViewInput{
		{
			SprintID:    &sprintID,
			ProjectID:   projectID,
			Name:        "Board",
			ViewType:    sprintdom.ViewTypeBoard,
			Position:    0,
			ViewContext: sprintdom.ViewContextSprint,
			Config: sprintdom.ViewConfig{
				ColumnBy: "status",
				Filters:  newDefaultFilters(nonSystemTaskTypeIDs, sprintIDs),
			},
		},
		{
			SprintID:    &sprintID,
			ProjectID:   projectID,
			Name:        "Table",
			ViewType:    sprintdom.ViewTypeTable,
			Position:    1,
			ViewContext: sprintdom.ViewContextSprint,
			Config: sprintdom.ViewConfig{
				ColumnBy: "status",
				Filters:  newDefaultFilters(nonSystemTaskTypeIDs, sprintIDs),
			},
		},
	}
}
