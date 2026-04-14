package dto

import (
	"time"

	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
)

// --- View DTOs --------------------------------------------------------------

// CreateViewRequest is the body for POST /sprints/:sprintId/views.
type CreateViewRequest struct {
	Name     string             `json:"name" binding:"required"`
	ViewType sprintdom.ViewType `json:"view_type"`
	Config   *ViewConfigDTO     `json:"config"`
	Position *float64           `json:"position"`
}

// UpdateViewRequest is the body for PATCH /sprints/:sprintId/views/:viewId.
type UpdateViewRequest struct {
	Name     *string             `json:"name"`
	ViewType *sprintdom.ViewType `json:"view_type"`
	Config   *ViewConfigDTO      `json:"config"`
	Position *float64            `json:"position"`
}

// ViewConfigDTO is the JSON representation of sprintdom.ViewConfig.
type ViewFiltersDTO struct {
	StatusIDs   []string `json:"status_ids,omitempty"`
	AssigneeIDs []string `json:"assignee_ids,omitempty"`
	TaskTypeIDs []string `json:"task_type_ids,omitempty"`
}

type ViewConfigDTO struct {
	Fields    []string        `json:"fields,omitempty"`
	ColumnBy  string          `json:"column_by,omitempty"`
	Swimlanes string          `json:"swimlanes,omitempty"`
	SortBy    string          `json:"sort_by,omitempty"`
	FieldSum  string          `json:"field_sum,omitempty"`
	SliceBy   string          `json:"slice_by,omitempty"`
	Filters   *ViewFiltersDTO `json:"filters,omitempty"`
}

// ViewResponse is the public representation of a sprint view.
type ViewResponse struct {
	ID        uuid.UUID          `json:"id"`
	SprintID  *uuid.UUID         `json:"sprint_id,omitempty"`
	ProjectID uuid.UUID          `json:"project_id"`
	Name      string             `json:"name"`
	ViewType  sprintdom.ViewType `json:"view_type"`
	Config    ViewConfigDTO      `json:"config"`
	Position  float64            `json:"position"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// ViewFromEntity maps a domain SprintView to a ViewResponse DTO.
func ViewFromEntity(v *sprintdom.SprintView) ViewResponse {
	return ViewResponse{
		ID:        v.ID,
		SprintID:  v.SprintID,
		ProjectID: v.ProjectID,
		Name:      v.Name,
		ViewType:  v.ViewType,
		Config: ViewConfigDTO{
			Fields:    v.Config.Fields,
			ColumnBy:  v.Config.ColumnBy,
			Swimlanes: v.Config.Swimlanes,
			SortBy:    v.Config.SortBy,
			FieldSum:  v.Config.FieldSum,
			SliceBy:   v.Config.SliceBy,
			Filters: func() *ViewFiltersDTO {
				if v.Config.Filters == nil {
					return nil
				}
				return &ViewFiltersDTO{
					StatusIDs:   v.Config.Filters.StatusIDs,
					AssigneeIDs: v.Config.Filters.AssigneeIDs,
					TaskTypeIDs: v.Config.Filters.TaskTypeIDs,
				}
			}(),
		},
		Position:  v.Position,
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}

// toViewConfig maps a nullable ViewConfigDTO pointer to a domain ViewConfig.
func toViewConfig(d *ViewConfigDTO) sprintdom.ViewConfig {
	if d == nil {
		return sprintdom.ViewConfig{}
	}
	return sprintdom.ViewConfig{
		Fields:    d.Fields,
		ColumnBy:  d.ColumnBy,
		Swimlanes: d.Swimlanes,
		SortBy:    d.SortBy,
		FieldSum:  d.FieldSum,
		SliceBy:   d.SliceBy,
		Filters: func() *sprintdom.ViewFilters {
			if d.Filters == nil {
				return nil
			}
			return &sprintdom.ViewFilters{
				StatusIDs:   d.Filters.StatusIDs,
				AssigneeIDs: d.Filters.AssigneeIDs,
				TaskTypeIDs: d.Filters.TaskTypeIDs,
			}
		}(),
	}
}

// toViewConfigPtr maps a nullable ViewConfigDTO pointer to a domain ViewConfig pointer.
func toViewConfigPtr(d *ViewConfigDTO) *sprintdom.ViewConfig {
	if d == nil {
		return nil
	}
	cfg := toViewConfig(d)
	return &cfg
}

// --- Task-position DTOs -----------------------------------------------------

// MoveTaskRequest is the body for PUT /views/:viewId/task-positions/:taskId.
type MoveTaskRequest struct {
	Position float64 `json:"position" binding:"min=0"`
	GroupKey *string `json:"group_key"`
}

// BulkMoveTaskItem is a single entry in a BulkMoveTasksRequest.
type BulkMoveTaskItem struct {
	TaskID   uuid.UUID `json:"task_id" binding:"required"`
	Position float64   `json:"position" binding:"min=0"`
	GroupKey *string   `json:"group_key"`
}

// BulkMoveTasksRequest is the body for PUT /views/:viewId/task-positions.
type BulkMoveTasksRequest struct {
	Items []BulkMoveTaskItem `json:"items" binding:"required,min=1"`
}

// TaskPositionResponse is the public representation of a ViewTaskPosition.
type TaskPositionResponse struct {
	ViewID   uuid.UUID `json:"view_id"`
	TaskID   uuid.UUID `json:"task_id"`
	Position float64   `json:"position"`
	GroupKey *string   `json:"group_key,omitempty"`
}

// TaskPositionFromEntity maps a domain ViewTaskPosition to a DTO.
func TaskPositionFromEntity(p *sprintdom.ViewTaskPosition) TaskPositionResponse {
	return TaskPositionResponse{
		ViewID:   p.ViewID,
		TaskID:   p.TaskID,
		Position: p.Position,
		GroupKey: p.GroupKey,
	}
}

// ToCreateInput builds the domain input for a sprint-scoped view.
func (r CreateViewRequest) ToCreateInput(sprintID uuid.UUID, projectID uuid.UUID) sprintdom.CreateViewInput {
	pos := 0.0
	if r.Position != nil {
		pos = *r.Position
	}
	return sprintdom.CreateViewInput{
		SprintID:    &sprintID,
		ProjectID:   projectID,
		Name:        r.Name,
		ViewType:    r.ViewType,
		Config:      toViewConfig(r.Config),
		Position:    pos,
		ViewContext: sprintdom.ViewContextSprint,
	}
}

// ToCreateProjectViewInput builds the domain input for a project-level view
// (backlog or timeline).  viewCtx must be ViewContextBacklog or ViewContextTimeline.
func (r CreateViewRequest) ToCreateProjectViewInput(projectID uuid.UUID, viewCtx sprintdom.ViewContext) sprintdom.CreateViewInput {
	pos := 0.0
	if r.Position != nil {
		pos = *r.Position
	}
	return sprintdom.CreateViewInput{
		SprintID:    nil,
		ProjectID:   projectID,
		Name:        r.Name,
		ViewType:    r.ViewType,
		Config:      toViewConfig(r.Config),
		Position:    pos,
		ViewContext: viewCtx,
	}
}

// ToUpdateInput builds the domain input from request data.
func (r UpdateViewRequest) ToUpdateInput() sprintdom.UpdateViewInput {
	return sprintdom.UpdateViewInput{
		Name:     r.Name,
		ViewType: r.ViewType,
		Config:   toViewConfigPtr(r.Config),
		Position: r.Position,
	}
}

// --- Reorder DTOs -----------------------------------------------------------

// ReorderViewsRequest is the body for PUT /views/positions.
// ViewIDs must list every view for the interaction in the desired tab order.
type ReorderViewsRequest struct {
	ViewIDs []uuid.UUID `json:"view_ids" binding:"required,min=1"`
}
