package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Paca-AI/api/internal/apierr"
	sprintdom "github.com/Paca-AI/api/internal/domain/sprint"
	taskdom "github.com/Paca-AI/api/internal/domain/task"
	"github.com/Paca-AI/api/internal/events"
	"github.com/Paca-AI/api/internal/platform/messaging"
	"github.com/Paca-AI/api/internal/transport/http/dto"
	"github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// TaskHandler handles task management endpoints.
type TaskHandler struct {
	svc         taskdom.Service
	viewSvc     sprintdom.ViewService
	activitySvc taskdom.ActivityService
	publisher   *messaging.Publisher
}

// NewTaskHandler returns a TaskHandler wired to the task service, view service,
// and activity service.
func NewTaskHandler(svc taskdom.Service, viewSvc sprintdom.ViewService, activitySvc taskdom.ActivityService, opts ...TaskHandlerOption) *TaskHandler {
	h := &TaskHandler{svc: svc, viewSvc: viewSvc, activitySvc: activitySvc}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// TaskHandlerOption is a functional option for TaskHandler.
type TaskHandlerOption func(*TaskHandler)

// WithTaskPublisher attaches a Valkey publisher used to enqueue assignment
// events for the NotificationConsumer worker.
func WithTaskPublisher(p *messaging.Publisher) TaskHandlerOption {
	return func(h *TaskHandler) {
		h.publisher = p
	}
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

	var req dto.UpdateTaskTypeRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	t, err := h.svc.UpdateTaskType(c.Request.Context(), projectID, typeID, taskdom.UpdateTaskTypeInput{
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
	if err := h.svc.DeleteTaskType(c.Request.Context(), projectID, typeID); err != nil {
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
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	statusID, err := parseTaskStatusID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateTaskStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	s, err := h.svc.UpdateTaskStatus(c.Request.Context(), projectID, statusID, taskdom.UpdateTaskStatusInput{
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
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	statusID, err := parseTaskStatusID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteTaskStatus(c.Request.Context(), projectID, statusID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "task status deleted"})
}

// SetDefaultTaskStatus handles PUT /projects/:projectId/task-statuses/:statusId/set-default.
func (h *TaskHandler) SetDefaultTaskStatus(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	statusID, err := parseTaskStatusID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	s, err := h.svc.SetDefaultTaskStatus(c.Request.Context(), projectID, statusID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.TaskStatusFromEntity(s))
}

// --- Tasks ------------------------------------------------------------------

// parseTaskSort resolves the sort_by query parameter into a TaskSort.
// For custom field sort keys the field definition is looked up via the service
// (ListCustomFieldDefinitions is cached, so this is cheap).
// Note: the public "created" key maps to the DB column created_at in applyTaskSort.
func parseTaskSort(ctx context.Context, svc taskdom.Service, projectID uuid.UUID, sortByRaw string) taskdom.TaskSort {
	sortBy := strings.TrimSpace(sortByRaw)
	switch sortBy {
	case "importance", "title", "story_points", "start_date", "due_date", "created":
		return taskdom.TaskSort{By: sortBy}
	case "", "manual":
		return taskdom.TaskSort{}
	default:
		cfs, err := svc.ListCustomFieldDefinitions(ctx, projectID)
		if err != nil {
			return taskdom.TaskSort{}
		}
		for _, cf := range cfs {
			if cf.FieldKey == sortBy {
				return taskdom.TaskSort{
					By:     sortBy,
					CFType: string(cf.FieldType),
					CFOpts: cf.Options,
				}
			}
		}
		return taskdom.TaskSort{}
	}
}

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

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
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
		if strings.EqualFold(strings.TrimSpace(raw), "null") {
			filter.AssigneeNull = true
		} else {
			id, err := uuid.Parse(raw)
			if err != nil {
				presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid assignee_id"))
				return
			}
			filter.AssigneeID = &id
		}
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
	if raw := c.Query("task_type_id"); raw != "" {
		if strings.EqualFold(strings.TrimSpace(raw), "null") {
			filter.TaskTypeNull = true
		} else {
			id, err := uuid.Parse(raw)
			if err != nil {
				presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid task_type_id"))
				return
			}
			filter.TaskTypeIDs = []uuid.UUID{id}
		}
	}
	if raw := c.Query("parent_task_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid parent_task_id"))
			return
		}
		filter.ParentTaskID = &id
	}
	if cursorRaw := c.Query("cursor"); cursorRaw != "" {
		filter.CursorAfter = &cursorRaw
	}
	sort := parseTaskSort(c.Request.Context(), h.svc, projectID, c.Query("sort_by"))

	var posMap map[uuid.UUID]*sprintdom.ViewTaskPosition
	if raw := c.Query("view_id"); raw != "" {
		viewID, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid view_id"))
			return
		}
		positions, err := h.viewSvc.ListTaskPositions(c.Request.Context(), projectID, viewID)
		if err != nil {
			presenter.Error(c, err)
			return
		}
		posMap = make(map[uuid.UUID]*sprintdom.ViewTaskPosition, len(positions))
		for _, p := range positions {
			cp := p
			posMap[p.TaskID] = cp
		}
		// When no explicit sort_by is requested (manual sort), order by the saved
		// view positions so the first page reflects the user's manual order.
		if sort.By == "" {
			sort.By = "view_position"
			sort.ViewID = &viewID
		}
	}

	// Count without cursor so the total reflects all matching tasks, not just the current page.
	aggFilter := filter
	aggFilter.CursorAfter = nil
	// Optionally sum a numeric field across all matching tasks.
	// Returns null in the response when sum_field is absent or "count".
	sumField := strings.TrimSpace(c.Query("sum_field"))

	var (
		tasks      []*taskdom.Task
		hasMore    bool
		totalCount int64
		fieldSumV  float64
	)
	g, gctx := errgroup.WithContext(c.Request.Context())
	g.Go(func() error {
		var err error
		tasks, hasMore, err = h.svc.ListTasks(gctx, projectID, filter, pageSize, sort)
		return err
	})
	g.Go(func() error {
		var err error
		totalCount, err = h.svc.CountTasks(gctx, projectID, aggFilter)
		return err
	})
	if sumField != "" && sumField != "count" {
		g.Go(func() error {
			var err error
			fieldSumV, err = h.svc.SumTaskField(gctx, projectID, aggFilter, sumField)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		presenter.Error(c, err)
		return
	}

	var fieldSum *float64
	if sumField != "" && sumField != "count" {
		fieldSum = &fieldSumV
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

	var nextCursor *string
	if hasMore && len(tasks) > 0 {
		last := tasks[len(tasks)-1]
		s := taskdom.EncodeTaskCursor(last, sort)
		nextCursor = &s
	}
	presenter.OK(c, gin.H{
		"items":       resp,
		"page_size":   pageSize,
		"next_cursor": nextCursor,
		"total_count": totalCount,
		"field_sum":   fieldSum,
	})
}

// GetTask handles GET /projects/:projectId/tasks/:taskId.
func (h *TaskHandler) GetTask(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	t, err := h.svc.GetTask(c.Request.Context(), projectID, taskID)
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
		Description:  req.NormalizedDescription(),
		Importance:   req.Importance,
		StoryPoints:  req.StoryPoints,
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

	// Record creation activity (best-effort).
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok {
		agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
		var agentIDPtr *uuid.UUID
		if agentID != uuid.Nil {
			agentIDPtr = &agentID
		}
		content, _ := json.Marshal(map[string]any{"title": t.Title})
		_ = h.activitySvc.RecordActivity(c.Request.Context(), taskdom.RecordActivityInput{
			TaskID:       t.ID,
			ProjectID:    projectID,
			ActorID:      &actorID,
			ActorAgentID: agentIDPtr,
			ActivityType: taskdom.ActivityTypeTaskCreated,
			Content:      content,
		})

		// Enqueue an assignment event so the NotificationConsumer can create
		// the in-app notification asynchronously (best-effort).
		if h.publisher != nil && req.AssigneeID != nil {
			_ = h.publisher.Append(c.Request.Context(), events.StreamTaskAssignments, "task.assigned", map[string]any{
				"task_id":                t.ID,
				"project_id":             projectID,
				"new_assignee_member_id": req.AssigneeID.String(),
				"actor_user_id":          actorID.String(),
			})
		}
	}

	presenter.Created(c, dto.TaskFromEntity(t))
}

// UpdateTask handles PATCH /projects/:projectId/tasks/:taskId.
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateTaskRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	// Fetch old state before mutating so we can record before/after values.
	oldTask, _ := h.svc.GetTask(c.Request.Context(), projectID, taskID)

	t, err := h.svc.UpdateTask(c.Request.Context(), projectID, taskID, taskdom.UpdateTaskInput{
		TaskTypeID:   req.TaskTypeID.Ptr(),
		StatusID:     req.StatusID.Ptr(),
		SprintID:     req.SprintID.Ptr(),
		ParentTaskID: req.ParentTaskID.Ptr(),
		Title:        req.Title,
		Description:  req.Description.Ptr(),
		Importance:   req.Importance,
		StoryPoints:  req.StoryPoints.Ptr(),
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

	// Record update activity (best-effort).
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok && oldTask != nil {
		agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
		var agentIDPtr *uuid.UUID
		if agentID != uuid.Nil {
			agentIDPtr = &agentID
		}
		changes := h.taskChangedFields(c.Request.Context(), oldTask, req)
		if len(changes) > 0 {
			content, _ := json.Marshal(map[string]any{"changes": changes})
			_ = h.activitySvc.RecordActivity(c.Request.Context(), taskdom.RecordActivityInput{
				TaskID:       taskID,
				ProjectID:    projectID,
				ActorID:      &actorID,
				ActorAgentID: agentIDPtr,
				ActivityType: taskdom.ActivityTypeTaskUpdated,
				Content:      content,
			})
		}

		// Enqueue an assignment event when the assignee changed so the
		// NotificationConsumer can create the in-app notification asynchronously.
		if h.publisher != nil && req.AssigneeID.Set && req.AssigneeID.Value != nil {
			oldAssignee := uuidPtrToStr(oldTask.AssigneeID)
			newAssignee := req.AssigneeID.Value.String()
			if oldAssignee != newAssignee {
				_ = h.publisher.Append(c.Request.Context(), events.StreamTaskAssignments, "task.assigned", map[string]any{
					"task_id":                taskID,
					"project_id":             projectID,
					"new_assignee_member_id": req.AssigneeID.Value.String(),
					"old_assignee_member_id": oldAssignee,
					"actor_user_id":          actorID.String(),
				})
			}
		}
	}

	presenter.OK(c, dto.TaskFromEntity(t))
}

// taskChangedFields compares the old task snapshot against the patch request
// and returns a FieldChange for each field that actually changed.  Old and New
// values are populated so activity consumers can render before/after messages.
// Status and type names are resolved via the service; for ID-only references
// (assignee, reporter, sprint, parent) the UUID string is stored.
func (h *TaskHandler) taskChangedFields(ctx context.Context, old *taskdom.Task, req dto.UpdateTaskRequest) []taskdom.FieldChange {
	var changes []taskdom.FieldChange

	if req.Title != "" && req.Title != old.Title {
		changes = append(changes, taskdom.FieldChange{Field: "title", Old: old.Title, New: req.Title})
	}

	if req.StatusID.Set {
		oldName := h.resolveStatusName(ctx, old.StatusID)
		newName := h.resolveStatusName(ctx, req.StatusID.Value)
		if fmt.Sprint(oldName) != fmt.Sprint(newName) {
			changes = append(changes, taskdom.FieldChange{Field: "status", Old: oldName, New: newName})
		}
	}

	if req.TaskTypeID.Set {
		oldName := h.resolveTaskTypeName(ctx, old.TaskTypeID)
		newName := h.resolveTaskTypeName(ctx, req.TaskTypeID.Value)
		if fmt.Sprint(oldName) != fmt.Sprint(newName) {
			changes = append(changes, taskdom.FieldChange{Field: "task_type", Old: oldName, New: newName})
		}
	}

	if req.Importance != nil && *req.Importance != old.Importance {
		changes = append(changes, taskdom.FieldChange{Field: "importance", Old: old.Importance, New: *req.Importance})
	}

	if req.StoryPoints.Set {
		oldVal := intPtrToStr(old.StoryPoints)
		newVal := intPtrToStr(req.StoryPoints.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "story_points", Old: old.StoryPoints, New: req.StoryPoints.Value})
		}
	}

	if req.AssigneeID.Set {
		oldVal := uuidPtrToStr(old.AssigneeID)
		newVal := uuidPtrToStr(req.AssigneeID.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "assignee", Old: oldVal, New: newVal})
		}
	}

	if req.ReporterID.Set {
		oldVal := uuidPtrToStr(old.ReporterID)
		newVal := uuidPtrToStr(req.ReporterID.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "reporter", Old: oldVal, New: newVal})
		}
	}

	if req.SprintID.Set {
		oldVal := uuidPtrToStr(old.SprintID)
		newVal := uuidPtrToStr(req.SprintID.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "sprint", Old: oldVal, New: newVal})
		}
	}

	if req.ParentTaskID.Set {
		oldVal := uuidPtrToStr(old.ParentTaskID)
		newVal := uuidPtrToStr(req.ParentTaskID.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "parent_task", Old: oldVal, New: newVal})
		}
	}

	if req.StartDate.Set {
		oldVal := timePtrToStr(old.StartDate)
		newVal := timePtrToStr(req.StartDate.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "start_date", Old: oldVal, New: newVal})
		}
	}

	if req.DueDate.Set {
		oldVal := timePtrToStr(old.DueDate)
		newVal := timePtrToStr(req.DueDate.Value)
		if oldVal != newVal {
			changes = append(changes, taskdom.FieldChange{Field: "due_date", Old: oldVal, New: newVal})
		}
	}

	if req.Description.Set && string(req.Description.Value) != string(old.Description) {
		changes = append(changes, taskdom.FieldChange{Field: "description", Old: old.Description, New: req.Description.Value})
	}

	if req.Tags != nil {
		changes = append(changes, taskdom.FieldChange{Field: "tags", Old: old.Tags, New: req.Tags})
	}

	if req.CustomFields != nil {
		oldJSON, oldErr := json.Marshal(old.CustomFields)
		newJSON, newErr := json.Marshal(*req.CustomFields)
		if oldErr != nil || newErr != nil || string(oldJSON) != string(newJSON) {
			changes = append(changes, taskdom.FieldChange{Field: "custom_fields"})
		}
	}

	return changes
}

// resolveStatusName looks up a status by ID and returns its name, falling back
// to the UUID string if the lookup fails.  Returns nil for a nil ID.
func (h *TaskHandler) resolveStatusName(ctx context.Context, id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	if s, err := h.svc.GetTaskStatus(ctx, *id); err == nil {
		return s.Name
	}
	return id.String()
}

// resolveTaskTypeName looks up a task type by ID and returns its name, falling
// back to the UUID string if the lookup fails.  Returns nil for a nil ID.
func (h *TaskHandler) resolveTaskTypeName(ctx context.Context, id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	if t, err := h.svc.GetTaskType(ctx, *id); err == nil {
		return t.Name
	}
	return id.String()
}

// uuidPtrToStr converts a *uuid.UUID to a string (empty string for nil).
func uuidPtrToStr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// timePtrToStr formats a *time.Time as a date string (empty string for nil).
func timePtrToStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// intPtrToStr converts a *int to a string (empty string for nil).
func intPtrToStr(n *int) string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf("%d", *n)
}

// DeleteTask handles DELETE /projects/:projectId/tasks/:taskId.
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteTask(c.Request.Context(), projectID, taskID); err != nil {
		presenter.Error(c, err)
		return
	}

	// Record deletion activity (best-effort).
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok {
		agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
		var agentIDPtr *uuid.UUID
		if agentID != uuid.Nil {
			agentIDPtr = &agentID
		}
		_ = h.activitySvc.RecordActivity(c.Request.Context(), taskdom.RecordActivityInput{
			TaskID:       taskID,
			ProjectID:    projectID,
			ActorID:      &actorID,
			ActorAgentID: agentIDPtr,
			ActivityType: taskdom.ActivityTypeTaskDeleted,
			Content:      json.RawMessage(`{}`),
		})
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
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	fieldID, err := parseCustomFieldID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	f, err := h.svc.GetCustomFieldDefinition(c.Request.Context(), projectID, fieldID)
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
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	fieldID, err := parseCustomFieldID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateCustomFieldDefinitionRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	f, err := h.svc.UpdateCustomFieldDefinition(c.Request.Context(), projectID, fieldID, taskdom.UpdateCustomFieldDefinitionInput{
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
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	fieldID, err := parseCustomFieldID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteCustomFieldDefinition(c.Request.Context(), projectID, fieldID); err != nil {
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

// --- Activities / Comments --------------------------------------------------

// parseCommentID parses the :commentId path parameter.
func parseCommentID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid comment id")
	}
	return id, nil
}

// ListTaskActivities handles GET /projects/:projectId/tasks/:taskId/activities.
func (h *TaskHandler) ListTaskActivities(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	activities, err := h.activitySvc.ListActivities(c.Request.Context(), taskID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.ActivityResponse, 0, len(activities))
	for _, a := range activities {
		resp = append(resp, dto.ActivityFromEntity(a))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// AddComment handles POST /projects/:projectId/tasks/:taskId/activities/comments.
func (h *TaskHandler) AddComment(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	actorID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	var req dto.AddCommentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
	var agentIDPtr *uuid.UUID
	if agentID != uuid.Nil {
		agentIDPtr = &agentID
	}

	a, err := h.activitySvc.AddComment(c.Request.Context(), taskdom.AddCommentInput{
		TaskID:    taskID,
		ProjectID: projectID,
		ActorID:   actorID,
		AgentID:   agentIDPtr,
		Content:   req.Content,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.ActivityFromEntity(a))
}

// UpdateComment handles PATCH /projects/:projectId/tasks/:taskId/activities/comments/:commentId.
func (h *TaskHandler) UpdateComment(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	commentID, err := parseCommentID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	actorID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	var req dto.UpdateCommentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
	var agentIDPtr *uuid.UUID
	if agentID != uuid.Nil {
		agentIDPtr = &agentID
	}

	a, err := h.activitySvc.UpdateComment(c.Request.Context(), commentID, projectID, actorID, agentIDPtr, req.Content)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.ActivityFromEntity(a))
}

// DeleteComment handles DELETE /projects/:projectId/tasks/:taskId/activities/comments/:commentId.
func (h *TaskHandler) DeleteComment(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	commentID, err := parseCommentID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	actorID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
	var agentIDPtr *uuid.UUID
	if agentID != uuid.Nil {
		agentIDPtr = &agentID
	}

	if err := h.activitySvc.DeleteComment(c.Request.Context(), commentID, projectID, actorID, agentIDPtr); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}
