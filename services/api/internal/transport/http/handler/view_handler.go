package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/apierr"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	"github.com/paca/api/internal/transport/http/dto"
	"github.com/paca/api/internal/transport/http/middleware"
	"github.com/paca/api/internal/transport/http/presenter"
)

// ViewHandler handles sprint-view and task-position endpoints.
type ViewHandler struct {
	svc sprintdom.ViewService
}

// NewViewHandler returns a ViewHandler wired to the view service.
func NewViewHandler(svc sprintdom.ViewService) *ViewHandler {
	return &ViewHandler{svc: svc}
}

// ListViews handles GET /sprints/:sprintId/views.
func (h *ViewHandler) ListViews(c *gin.Context) {
	sprintID, err := parseSprintID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	views, err := h.svc.ListViews(c.Request.Context(), sprintID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.ViewResponse, 0, len(views))
	for _, v := range views {
		resp = append(resp, dto.ViewFromEntity(v))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// GetView handles GET /sprints/:sprintId/views/:viewId.
func (h *ViewHandler) GetView(c *gin.Context) {
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	v, err := h.svc.GetView(c.Request.Context(), viewID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.ViewFromEntity(v))
}

// CreateView handles POST /sprints/:sprintId/views.
func (h *ViewHandler) CreateView(c *gin.Context) {
	sprintID, err := parseSprintID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateViewRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	v, err := h.svc.CreateView(c.Request.Context(), req.ToCreateInput(sprintID, projectID))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.ViewFromEntity(v))
}

// ListBacklogViews handles GET /projects/:projectId/product-backlog/views.
func (h *ViewHandler) ListBacklogViews(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	views, err := h.svc.ListBacklogViews(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.ViewResponse, 0, len(views))
	for _, v := range views {
		resp = append(resp, dto.ViewFromEntity(v))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// CreateBacklogView handles POST /projects/:projectId/product-backlog/views.
func (h *ViewHandler) CreateBacklogView(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateViewRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	v, err := h.svc.CreateView(c.Request.Context(), req.ToCreateBacklogInput(projectID))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.ViewFromEntity(v))
}

// UpdateView handles PATCH /sprints/:sprintId/views/:viewId.
func (h *ViewHandler) UpdateView(c *gin.Context) {
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateViewRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	v, err := h.svc.UpdateView(c.Request.Context(), viewID, req.ToUpdateInput())
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.ViewFromEntity(v))
}

// DeleteView handles DELETE /sprints/:sprintId/views/:viewId.
func (h *ViewHandler) DeleteView(c *gin.Context) {
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteView(c.Request.Context(), viewID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// ListTaskPositions handles GET /views/:viewId/task-positions.
func (h *ViewHandler) ListTaskPositions(c *gin.Context) {
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	positions, err := h.svc.ListTaskPositions(c.Request.Context(), viewID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.TaskPositionResponse, 0, len(positions))
	for _, p := range positions {
		resp = append(resp, dto.TaskPositionFromEntity(p))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// MoveTask handles PUT /views/:viewId/task-positions/:taskId.
func (h *ViewHandler) MoveTask(c *gin.Context) {
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	taskID, err := parseTaskIDParam(c, "taskId")
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.MoveTaskRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := h.svc.MoveTask(c.Request.Context(), viewID, sprintdom.MoveTaskInput{
		TaskID:   taskID,
		Position: req.Position,
		GroupKey: req.GroupKey,
	}); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// parseViewID extracts and validates the :viewId path parameter.
func parseViewID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("viewId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid view id")
	}
	return id, nil
}

// parseTaskIDParam extracts and validates a task UUID from the named path parameter.
func parseTaskIDParam(c *gin.Context, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid task id")
	}
	return id, nil
}
