package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/apierr"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	taskdom "github.com/paca/api/internal/domain/task"
	"github.com/paca/api/internal/transport/http/dto"
	"github.com/paca/api/internal/transport/http/middleware"
	"github.com/paca/api/internal/transport/http/presenter"
)

// TaskHandler handles task management endpoints.
type TaskHandler struct {
	svc     taskdom.Service
	viewSvc sprintdom.ViewService
}

// NewTaskHandler returns a TaskHandler wired to the task service and the view
// service.
func NewTaskHandler(svc taskdom.Service, viewSvc sprintdom.ViewService) *TaskHandler {
	return &TaskHandler{svc: svc, viewSvc: viewSvc}
}

// --- Task Types -------------------------------------------------------------

// ListTaskTypes handles GET /projects/:projectId/task-types.
func (h *TaskHandler) ListTaskTypes(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	types, err := h.svc.ListTaskTypes(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.TaskTypeResponse, 0, len(types))
	for _, t := range types {
		resp = append(resp, dto.TaskTypeFromEntity(t))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// CreateTaskType handles POST /projects/:projectId/task-types.
func (h *TaskHandler) CreateTaskType(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateTaskTypeRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	t, err := h.svc.CreateTaskType(c.Request.Context(), taskdom.CreateTaskTypeInput{
		ProjectID:   projectID,
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Description: req.Description,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.TaskTypeFromEntity(t))
}

// UpdateTaskType handles PATCH /projects/:projectId/task-types/:typeId.
func (h *TaskHandler) UpdateTaskType(c *gin.Context) {
	typeID, err := parseTaskTypeID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateTaskTypeRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	t, err := h.svc.UpdateTaskType(c.Request.Context(), typeID, taskdom.UpdateTaskTypeInput{
		Name:        req.Name,
		Icon:        req.Icon,
		Color:       req.Color,
		Description: req.Description,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskTypeFromEntity(t))
}

// DeleteTaskType handles DELETE /projects/:projectId/task-types/:typeId.
func (h *TaskHandler) DeleteTaskType(c *gin.Context) {
	typeID, err := parseTaskTypeID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteTaskType(c.Request.Context(), typeID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "task type deleted"})
}

// SetDefaultTaskType handles PUT /projects/:projectId/task-types/:typeId/set-default.
func (h *TaskHandler) SetDefaultTaskType(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	typeID, err := parseTaskTypeID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	t, err := h.svc.SetDefaultTaskType(c.Request.Context(), projectID, typeID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskTypeFromEntity(t))
}

// --- Task Statuses ----------------------------------------------------------

// ListTaskStatuses handles GET /projects/:projectId/task-statuses.
func (h *TaskHandler) ListTaskStatuses(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	statuses, err := h.svc.ListTaskStatuses(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.TaskStatusResponse, 0, len(statuses))
	for _, s := range statuses {
		resp = append(resp, dto.TaskStatusFromEntity(s))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// CreateTaskStatus handles POST /projects/:projectId/task-statuses.
func (h *TaskHandler) CreateTaskStatus(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateTaskStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	s, err := h.svc.CreateTaskStatus(c.Request.Context(), taskdom.CreateTaskStatusInput{
		ProjectID: projectID,
		Name:      req.Name,
		Color:     req.Color,
		Position:  req.Position,
		Category:  req.Category,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.TaskStatusFromEntity(s))
}

// UpdateTaskStatus handles PATCH /projects/:projectId/task-statuses/:statusId.
func (h *TaskHandler) UpdateTaskStatus(c *gin.Context) {
	statusID, err := parseTaskStatusID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateTaskStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	s, err := h.svc.UpdateTaskStatus(c.Request.Context(), statusID, taskdom.UpdateTaskStatusInput{
		Name:     req.Name,
		Color:    req.Color,
		Position: req.Position,
		Category: req.Category,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskStatusFromEntity(s))
}

// DeleteTaskStatus handles DELETE /projects/:projectId/task-statuses/:statusId.
func (h *TaskHandler) DeleteTaskStatus(c *gin.Context) {
	statusID, err := parseTaskStatusID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteTaskStatus(c.Request.Context(), statusID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "task status deleted"})
}

// --- Tasks ------------------------------------------------------------------

// ListTasks handles GET /projects/:projectId/tasks.
// Supported filter query params:
//   - sprint_id=<uuid>|null or sprint_ids=<uuid,uuid>
//   - status_id=<uuid> or status_ids=<uuid,uuid>
//   - assignee_id=<uuid> or assignee_ids=<uuid,uuid>
//   - task_type_ids=<uuid,uuid>
//   - parent_task_id=<uuid>
func (h *TaskHandler) ListTasks(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	page, pageSize := pagingParams(c)
	filter := taskdom.TaskFilter{}

	if raw := c.Query("sprint_id"); raw != "" {
		if strings.EqualFold(strings.TrimSpace(raw), "null") {
			filter.BacklogOnly = true
		} else {
			id, err := uuid.Parse(raw)
			if err != nil {
				presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid sprint_id"))
				return
			}
			filter.SprintID = &id
		}
	}
	if ids, err := parseQueryUUIDs(c.Query("sprint_ids")); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid sprint_ids"))
		return
	} else if len(ids) > 0 {
		filter.SprintIDs = ids
		filter.BacklogOnly = false
		filter.SprintID = nil
	}
	if raw := c.Query("status_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid status_id"))
			return
		}
		filter.StatusID = &id
	}
	if ids, err := parseQueryUUIDs(c.Query("status_ids")); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid status_ids"))
		return
	} else if len(ids) > 0 {
		filter.StatusIDs = ids
	}
	if raw := c.Query("assignee_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid assignee_id"))
			return
		}
		filter.AssigneeID = &id
	}
	if ids, err := parseQueryUUIDs(c.Query("assignee_ids")); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid assignee_ids"))
		return
	} else if len(ids) > 0 {
		filter.AssigneeIDs = ids
	}
	if ids, err := parseQueryUUIDs(c.Query("task_type_ids")); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid task_type_ids"))
		return
	} else if len(ids) > 0 {
		filter.TaskTypeIDs = ids
	}
	if raw := c.Query("parent_task_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid parent_task_id"))
			return
		}
		filter.ParentTaskID = &id
	}

	var posMap map[uuid.UUID]*sprintdom.ViewTaskPosition
	if raw := c.Query("view_id"); raw != "" {
		viewID, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid view_id"))
			return
		}
		positions, err := h.viewSvc.ListTaskPositions(c.Request.Context(), viewID)
		if err != nil {
			presenter.Error(c, err)
			return
		}
		posMap = make(map[uuid.UUID]*sprintdom.ViewTaskPosition, len(positions))
		for _, p := range positions {
			cp := p
			posMap[p.TaskID] = cp
		}
	}

	tasks, total, err := h.svc.ListTasks(c.Request.Context(), projectID, filter, page, pageSize)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		r := dto.TaskFromEntity(t)
		if pos, ok := posMap[t.ID]; ok {
			r.ViewPosition = &pos.Position
			r.ViewGroupKey = pos.GroupKey
		}
		resp = append(resp, r)
	}
	presenter.OK(c, gin.H{"items": resp, "total": total, "page": page, "page_size": pageSize})
}

// GetTask handles GET /projects/:projectId/tasks/:taskId.
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	t, err := h.svc.GetTask(c.Request.Context(), taskID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskFromEntity(t))
}

// GetTaskByNumber handles GET /projects/:projectId/tasks/by-number/:taskNumber.
// It looks up a task by its project-scoped sequential number.
func (h *TaskHandler) GetTaskByNumber(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var taskNumber int64
	if _, err := fmt.Sscanf(c.Param("taskNumber"), "%d", &taskNumber); err != nil || taskNumber < 1 {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid task number"))
		return
	}
	t, err := h.svc.GetTaskByNumber(c.Request.Context(), projectID, taskNumber)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskFromEntity(t))
}

// CreateTask handles POST /projects/:projectId/tasks.
func (h *TaskHandler) CreateTask(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateTaskRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	t, err := h.svc.CreateTask(c.Request.Context(), taskdom.CreateTaskInput{
		ProjectID:    projectID,
		TaskTypeID:   req.TaskTypeID,
		StatusID:     req.StatusID,
		SprintID:     req.SprintID,
		ParentTaskID: req.ParentTaskID,
		Title:        req.Title,
		Description:  req.Description,
		Importance:   req.Importance,
		AssigneeID:   req.AssigneeID,
		ReporterID:   req.ReporterID,
		CustomFields: req.CustomFields,
		StartDate:    req.StartDate,
		DueDate:      req.DueDate,
		Tags:         req.Tags,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.TaskFromEntity(t))
}

// UpdateTask handles PATCH /projects/:projectId/tasks/:taskId.
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateTaskRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	t, err := h.svc.UpdateTask(c.Request.Context(), taskID, taskdom.UpdateTaskInput{
		TaskTypeID:   req.TaskTypeID.Ptr(),
		StatusID:     req.StatusID.Ptr(),
		SprintID:     req.SprintID.Ptr(),
		ParentTaskID: req.ParentTaskID.Ptr(),
		Title:        req.Title,
		Description:  req.Description.Ptr(),
		Importance:   req.Importance,
		AssigneeID:   req.AssigneeID.Ptr(),
		ReporterID:   req.ReporterID.Ptr(),
		CustomFields: req.CustomFields,
		StartDate:    req.StartDate.Ptr(),
		DueDate:      req.DueDate.Ptr(),
		Tags:         req.Tags,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskFromEntity(t))
}

// DeleteTask handles DELETE /projects/:projectId/tasks/:taskId.
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteTask(c.Request.Context(), taskID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "task deleted"})
}

// --- helpers ----------------------------------------------------------------

func parseTaskTypeID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("typeId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid task type id")
	}
	return id, nil
}

func parseTaskStatusID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("statusId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid task status id")
	}
	return id, nil
}

func parseTaskID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid task id")
	}
	return id, nil
}

func parseQueryUUIDs(raw string) ([]uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseUUIDStrings(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// --- Custom Field Definitions -----------------------------------------------

// ListCustomFieldDefinitions handles GET /projects/:projectId/custom-fields.
func (h *TaskHandler) ListCustomFieldDefinitions(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	fields, err := h.svc.ListCustomFieldDefinitions(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.CustomFieldDefinitionResponse, 0, len(fields))
	for _, f := range fields {
		resp = append(resp, dto.CustomFieldDefinitionFromEntity(f))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// GetCustomFieldDefinition handles GET /projects/:projectId/custom-fields/:fieldId.
func (h *TaskHandler) GetCustomFieldDefinition(c *gin.Context) {
	fieldID, err := parseCustomFieldID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	f, err := h.svc.GetCustomFieldDefinition(c.Request.Context(), fieldID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.CustomFieldDefinitionFromEntity(f))
}

// CreateCustomFieldDefinition handles POST /projects/:projectId/custom-fields.
func (h *TaskHandler) CreateCustomFieldDefinition(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateCustomFieldDefinitionRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	f, err := h.svc.CreateCustomFieldDefinition(c.Request.Context(), taskdom.CreateCustomFieldDefinitionInput{
		ProjectID:   projectID,
		FieldKey:    req.FieldKey,
		DisplayName: req.DisplayName,
		FieldType:   req.FieldType,
		Options:     req.Options,
		IsRequired:  req.IsRequired,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.CustomFieldDefinitionFromEntity(f))
}

// UpdateCustomFieldDefinition handles PATCH /projects/:projectId/custom-fields/:fieldId.
func (h *TaskHandler) UpdateCustomFieldDefinition(c *gin.Context) {
	fieldID, err := parseCustomFieldID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateCustomFieldDefinitionRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	f, err := h.svc.UpdateCustomFieldDefinition(c.Request.Context(), fieldID, taskdom.UpdateCustomFieldDefinitionInput{
		DisplayName: req.DisplayName,
		FieldType:   req.FieldType,
		Options:     req.Options,
		IsRequired:  req.IsRequired,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.CustomFieldDefinitionFromEntity(f))
}

// DeleteCustomFieldDefinition handles DELETE /projects/:projectId/custom-fields/:fieldId.
func (h *TaskHandler) DeleteCustomFieldDefinition(c *gin.Context) {
	fieldID, err := parseCustomFieldID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteCustomFieldDefinition(c.Request.Context(), fieldID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "custom field deleted"})
}

func parseCustomFieldID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("fieldId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid custom field id")
	}
	return id, nil
}

// ListBacklogTasks handles GET /projects/:projectId/product-backlog.
// Returns a paginated list of tasks not assigned to any sprint (sprint_id IS NULL).
// This represents the product backlog — work that has been identified but not yet
// committed to a sprint, distinct from any sprint's own task list.
func (h *TaskHandler) ListBacklogTasks(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	page, pageSize := pagingParams(c)
	filter := taskdom.TaskFilter{BacklogOnly: true}
	if raw := c.Query("status_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			filter.StatusID = &id
		}
	}
	if raw := c.Query("assignee_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			filter.AssigneeID = &id
		}
	}

	var posMap map[uuid.UUID]*sprintdom.ViewTaskPosition
	if raw := c.Query("view_id"); raw != "" {
		viewID, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid view_id"))
			return
		}
		positions, err := h.viewSvc.ListTaskPositions(c.Request.Context(), viewID)
		if err != nil {
			presenter.Error(c, err)
			return
		}
		posMap = make(map[uuid.UUID]*sprintdom.ViewTaskPosition, len(positions))
		for _, p := range positions {
			cp := p
			posMap[p.TaskID] = cp
		}
	}

	tasks, total, err := h.svc.ListTasks(c.Request.Context(), projectID, filter, page, pageSize)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		r := dto.TaskFromEntity(t)
		if pos, ok := posMap[t.ID]; ok {
			r.ViewPosition = &pos.Position
			r.ViewGroupKey = pos.GroupKey
		}
		resp = append(resp, r)
	}
	presenter.OK(c, gin.H{"items": resp, "total": total, "page": page, "page_size": pageSize})
}

// ListTimelineTasks handles GET /projects/:projectId/timeline.
// Returns a paginated list of Epic tasks for the project.
// Epics are tracked on the timeline regardless of sprint assignment.
func (h *TaskHandler) ListTimelineTasks(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	page, pageSize := pagingParams(c)
	filter := taskdom.TaskFilter{}
	if ids, err := parseQueryUUIDs(c.Query("task_type_ids")); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid task_type_ids"))
		return
	} else if len(ids) > 0 {
		filter.TaskTypeIDs = ids
	}
	if raw := c.Query("status_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			filter.StatusID = &id
		}
	}
	if raw := c.Query("assignee_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			filter.AssigneeID = &id
		}
	}

	var posMap map[uuid.UUID]*sprintdom.ViewTaskPosition
	if raw := c.Query("view_id"); raw != "" {
		viewID, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid view_id"))
			return
		}
		positions, err := h.viewSvc.ListTaskPositions(c.Request.Context(), viewID)
		if err != nil {
			presenter.Error(c, err)
			return
		}
		posMap = make(map[uuid.UUID]*sprintdom.ViewTaskPosition, len(positions))
		for _, p := range positions {
			cp := p
			posMap[p.TaskID] = cp
		}
	}

	tasks, total, err := h.svc.ListTasks(c.Request.Context(), projectID, filter, page, pageSize)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		r := dto.TaskFromEntity(t)
		if pos, ok := posMap[t.ID]; ok {
			r.ViewPosition = &pos.Position
			r.ViewGroupKey = pos.GroupKey
		}
		resp = append(resp, r)
	}
	presenter.OK(c, gin.H{"items": resp, "total": total, "page": page, "page_size": pageSize})
}
