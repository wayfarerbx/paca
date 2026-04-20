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

// viewContextFromQuery reads the ?context query param (sprint | backlog | timeline).
// Returns an error for unknown values. Defaults to "sprint" when omitted.
func viewContextFromQuery(c *gin.Context) (sprintdom.ViewContext, error) {
	raw := c.DefaultQuery("context", string(sprintdom.ViewContextSprint))
	vc := sprintdom.ViewContext(raw)
	switch vc {
	case sprintdom.ViewContextSprint, sprintdom.ViewContextBacklog, sprintdom.ViewContextTimeline:
		return vc, nil
	default:
		return "", apierr.New(apierr.CodeBadRequest, "invalid context: must be sprint, backlog, or timeline")
	}
}

// parseQueryUUID reads a named UUID from the request query string.
func parseQueryUUID(c *gin.Context, param string) (uuid.UUID, error) {
	raw := c.Query(param)
	if raw == "" {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, param+" is required")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid "+param)
	}
	return id, nil
}

// ListViews handles GET /projects/:projectId/views?context=sprint|backlog|timeline.
// Sprint context additionally requires ?sprint_id=<uuid>.
func (h *ViewHandler) ListViews(c *gin.Context) {
	viewCtx, err := viewContextFromQuery(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var views []*sprintdom.SprintView
	if viewCtx == sprintdom.ViewContextSprint {
		var sprintID uuid.UUID
		sprintID, err = parseQueryUUID(c, "sprint_id")
		if err != nil {
			presenter.Error(c, err)
			return
		}
		views, err = h.svc.ListViews(c.Request.Context(), sprintID)
	} else {
		views, err = h.svc.ListProjectViews(c.Request.Context(), projectID, viewCtx)
	}
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

// GetView handles GET /projects/:projectId/views/:viewId.
func (h *ViewHandler) GetView(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	v, err := h.svc.GetView(c.Request.Context(), projectID, viewID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.ViewFromEntity(v))
}

// CreateView handles POST /projects/:projectId/views?context=sprint|backlog|timeline.
// Sprint context additionally requires ?sprint_id=<uuid>.
func (h *ViewHandler) CreateView(c *gin.Context) {
	viewCtx, err := viewContextFromQuery(c)
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

	var input sprintdom.CreateViewInput
	if viewCtx == sprintdom.ViewContextSprint {
		sprintID, err := parseQueryUUID(c, "sprint_id")
		if err != nil {
			presenter.Error(c, err)
			return
		}
		input = req.ToCreateInput(sprintID, projectID)
	} else {
		input = req.ToCreateProjectViewInput(projectID, viewCtx)
	}

	v, err := h.svc.CreateView(c.Request.Context(), input)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.ViewFromEntity(v))
}

// UpdateView handles PATCH /sprints/:sprintId/views/:viewId.
func (h *ViewHandler) UpdateView(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateViewRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	v, err := h.svc.UpdateView(c.Request.Context(), projectID, viewID, req.ToUpdateInput())
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.ViewFromEntity(v))
}

// DeleteView handles DELETE /sprints/:sprintId/views/:viewId.
func (h *ViewHandler) DeleteView(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteView(c.Request.Context(), projectID, viewID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// ListTaskPositions handles GET /views/:viewId/task-positions.
func (h *ViewHandler) ListTaskPositions(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	positions, err := h.svc.ListTaskPositions(c.Request.Context(), projectID, viewID)
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
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
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

	if err := h.svc.MoveTask(c.Request.Context(), projectID, viewID, sprintdom.MoveTaskInput{
		TaskID:   taskID,
		Position: req.Position,
		GroupKey: req.GroupKey,
	}); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// BulkMoveTasks handles PUT /views/:viewId/task-positions.
// Upserts the manual positions of multiple tasks in a view within a single
// database transaction.
func (h *ViewHandler) BulkMoveTasks(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	viewID, err := parseViewID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.BulkMoveTasksRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	items := make([]sprintdom.MoveTaskInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, sprintdom.MoveTaskInput{
			TaskID:   item.TaskID,
			Position: item.Position,
			GroupKey: item.GroupKey,
		})
	}

	if err := h.svc.BulkMoveTasks(c.Request.Context(), projectID, viewID, items); err != nil {
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

// ReorderViews handles PUT /projects/:projectId/views/positions?context=sprint|backlog|timeline.
// Sprint context additionally requires ?sprint_id=<uuid>.
func (h *ViewHandler) ReorderViews(c *gin.Context) {
	viewCtx, err := viewContextFromQuery(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.ReorderViewsRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if viewCtx == sprintdom.ViewContextSprint {
		var sprintID uuid.UUID
		sprintID, err = parseQueryUUID(c, "sprint_id")
		if err != nil {
			presenter.Error(c, err)
			return
		}
		err = h.svc.ReorderViews(c.Request.Context(), sprintID, req.ViewIDs)
	} else {
		err = h.svc.ReorderProjectViews(c.Request.Context(), projectID, viewCtx, req.ViewIDs)
	}
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}
