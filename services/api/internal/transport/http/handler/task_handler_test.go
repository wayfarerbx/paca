package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	taskdom "github.com/paca/api/internal/domain/task"
	"github.com/paca/api/internal/transport/http/handler"
	"github.com/paca/api/internal/transport/http/middleware"
)

// ---------------------------------------------------------------------------
// Fake task service
// ---------------------------------------------------------------------------

type fakeTaskSvc struct {
	mu            sync.RWMutex
	tasks         map[uuid.UUID]*taskdom.Task
	types         map[uuid.UUID]*taskdom.TaskType
	bddScenarios  map[uuid.UUID]*taskdom.BDDScenario
	lastProjectID uuid.UUID
	lastFilter    taskdom.TaskFilter
}

func newFakeTaskSvc() *fakeTaskSvc {
	return &fakeTaskSvc{
		tasks:        make(map[uuid.UUID]*taskdom.Task),
		types:        make(map[uuid.UUID]*taskdom.TaskType),
		bddScenarios: make(map[uuid.UUID]*taskdom.BDDScenario),
	}
}

// -- TaskTypeService --

func (f *fakeTaskSvc) ListTaskTypes(_ context.Context, _ uuid.UUID) ([]*taskdom.TaskType, error) {
	return nil, nil
}

func (f *fakeTaskSvc) GetTaskType(_ context.Context, _ uuid.UUID) (*taskdom.TaskType, error) {
	return nil, taskdom.ErrTypeNotFound
}

func (f *fakeTaskSvc) CreateTaskType(_ context.Context, in taskdom.CreateTaskTypeInput) (*taskdom.TaskType, error) {
	if in.Name == "" {
		return nil, taskdom.ErrTypeNameInvalid
	}
	now := time.Now()
	t := &taskdom.TaskType{ID: uuid.New(), ProjectID: in.ProjectID, Name: in.Name, CreatedAt: now, UpdatedAt: now}
	f.mu.Lock()
	f.types[t.ID] = t
	f.mu.Unlock()
	return t, nil
}

func (f *fakeTaskSvc) UpdateTaskType(_ context.Context, _, _ uuid.UUID, _ taskdom.UpdateTaskTypeInput) (*taskdom.TaskType, error) {
	return nil, taskdom.ErrTypeNotFound
}

func (f *fakeTaskSvc) DeleteTaskType(_ context.Context, _, _ uuid.UUID) error { return nil }

func (f *fakeTaskSvc) SetDefaultTaskType(_ context.Context, _, _ uuid.UUID) (*taskdom.TaskType, error) {
	return &taskdom.TaskType{}, nil
}

// -- TaskStatusService --

func (f *fakeTaskSvc) ListTaskStatuses(_ context.Context, _ uuid.UUID) ([]*taskdom.TaskStatus, error) {
	return nil, nil
}

func (f *fakeTaskSvc) GetTaskStatus(_ context.Context, _ uuid.UUID) (*taskdom.TaskStatus, error) {
	return nil, taskdom.ErrStatusNotFound
}

func (f *fakeTaskSvc) CreateTaskStatus(_ context.Context, in taskdom.CreateTaskStatusInput) (*taskdom.TaskStatus, error) {
	if !taskdom.ValidStatusCategories[in.Category] {
		return nil, taskdom.ErrStatusCategoryInvalid
	}
	now := time.Now()
	s := &taskdom.TaskStatus{ID: uuid.New(), ProjectID: in.ProjectID, Name: in.Name, Category: in.Category, CreatedAt: now, UpdatedAt: now}
	return s, nil
}

func (f *fakeTaskSvc) UpdateTaskStatus(_ context.Context, _, _ uuid.UUID, _ taskdom.UpdateTaskStatusInput) (*taskdom.TaskStatus, error) {
	return nil, taskdom.ErrStatusNotFound
}

func (f *fakeTaskSvc) DeleteTaskStatus(_ context.Context, _, _ uuid.UUID) error { return nil }

// -- TaskService --

func (f *fakeTaskSvc) ListTasks(_ context.Context, projectID uuid.UUID, filter taskdom.TaskFilter, _, _ int) ([]*taskdom.Task, int64, error) {
	f.mu.Lock()
	f.lastProjectID = projectID
	f.lastFilter = filter
	f.mu.Unlock()
	return nil, 0, nil
}

func (f *fakeTaskSvc) GetTask(_ context.Context, _, id uuid.UUID) (*taskdom.Task, error) {
	f.mu.RLock()
	t, ok := f.tasks[id]
	f.mu.RUnlock()
	if !ok {
		return nil, taskdom.ErrTaskNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTaskSvc) GetTaskByNumber(_ context.Context, projectID uuid.UUID, taskNumber int64) (*taskdom.Task, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, t := range f.tasks {
		if t.ProjectID == projectID && t.TaskNumber == taskNumber {
			cp := *t
			return &cp, nil
		}
	}
	return nil, taskdom.ErrTaskNotFound
}

func (f *fakeTaskSvc) CreateTask(_ context.Context, in taskdom.CreateTaskInput) (*taskdom.Task, error) {
	if in.Title == "" {
		return nil, taskdom.ErrTaskTitleInvalid
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	taskNum := int64(len(f.tasks) + 1)
	now := time.Now()
	t := &taskdom.Task{
		ID:           uuid.New(),
		ProjectID:    in.ProjectID,
		TaskNumber:   taskNum,
		Title:        in.Title,
		SprintID:     in.SprintID,
		StatusID:     in.StatusID,
		CustomFields: map[string]any{},
		Tags:         []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.tasks[t.ID] = t
	return t, nil
}

func (f *fakeTaskSvc) UpdateTask(_ context.Context, _, id uuid.UUID, in taskdom.UpdateTaskInput) (*taskdom.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, taskdom.ErrTaskNotFound
	}
	if in.StatusID != nil {
		t.StatusID = *in.StatusID
	}
	if in.SprintID != nil {
		t.SprintID = *in.SprintID
	}
	if in.TaskTypeID != nil {
		t.TaskTypeID = *in.TaskTypeID
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTaskSvc) DeleteTask(_ context.Context, _, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.tasks[id]; !ok {
		return taskdom.ErrTaskNotFound
	}
	delete(f.tasks, id)
	return nil
}

// -- CustomFieldDefinitionService --

func (f *fakeTaskSvc) ListCustomFieldDefinitions(_ context.Context, _ uuid.UUID) ([]*taskdom.CustomFieldDefinition, error) {
	return nil, nil
}

func (f *fakeTaskSvc) GetCustomFieldDefinition(_ context.Context, _, _ uuid.UUID) (*taskdom.CustomFieldDefinition, error) {
	return nil, taskdom.ErrCustomFieldNotFound
}

func (f *fakeTaskSvc) CreateCustomFieldDefinition(_ context.Context, _ taskdom.CreateCustomFieldDefinitionInput) (*taskdom.CustomFieldDefinition, error) {
	return nil, nil
}

func (f *fakeTaskSvc) UpdateCustomFieldDefinition(_ context.Context, _, _ uuid.UUID, _ taskdom.UpdateCustomFieldDefinitionInput) (*taskdom.CustomFieldDefinition, error) {
	return nil, taskdom.ErrCustomFieldNotFound
}

func (f *fakeTaskSvc) DeleteCustomFieldDefinition(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// -- BDDScenarioService --

func (f *fakeTaskSvc) ListBDDScenarios(_ context.Context, _, taskID uuid.UUID) ([]*taskdom.BDDScenario, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []*taskdom.BDDScenario
	for _, s := range f.bddScenarios {
		if s.TaskID == taskID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeTaskSvc) GetBDDScenario(_ context.Context, _, _, id uuid.UUID) (*taskdom.BDDScenario, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	s, ok := f.bddScenarios[id]
	if !ok {
		return nil, taskdom.ErrBDDScenarioNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeTaskSvc) CreateBDDScenario(_ context.Context, in taskdom.CreateBDDScenarioInput) (*taskdom.BDDScenario, error) {
	if in.Title == "" {
		return nil, taskdom.ErrBDDScenarioTitleInvalid
	}
	now := time.Now()
	s := &taskdom.BDDScenario{
		ID:     uuid.New(),
		TaskID: in.TaskID,
		Title:  in.Title,
		Given:  in.Given,
		When:   in.When,
		Then:   in.Then,

		CreatedAt: now,
		UpdatedAt: now,
	}
	f.mu.Lock()
	f.bddScenarios[s.ID] = s
	f.mu.Unlock()
	return s, nil
}

func (f *fakeTaskSvc) UpdateBDDScenario(_ context.Context, _, _, id uuid.UUID, in taskdom.UpdateBDDScenarioInput) (*taskdom.BDDScenario, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.bddScenarios[id]
	if !ok {
		return nil, taskdom.ErrBDDScenarioNotFound
	}
	if in.Title != nil {
		s.Title = *in.Title
	}
	if in.Given != nil {
		s.Given = *in.Given
	}
	if in.When != nil {
		s.When = *in.When
	}
	if in.Then != nil {
		s.Then = *in.Then
	}
	cp := *s
	return &cp, nil
}

func (f *fakeTaskSvc) DeleteBDDScenario(_ context.Context, _, _, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.bddScenarios[id]; !ok {
		return taskdom.ErrBDDScenarioNotFound
	}
	delete(f.bddScenarios, id)
	return nil
}

// ---------------------------------------------------------------------------
// Fake activity service
// ---------------------------------------------------------------------------

type fakeActivitySvc struct {
	mu         sync.RWMutex
	activities map[uuid.UUID]*taskdom.Activity
}

func newFakeActivitySvc() *fakeActivitySvc {
	return &fakeActivitySvc{activities: make(map[uuid.UUID]*taskdom.Activity)}
}

func (f *fakeActivitySvc) RecordActivity(_ context.Context, in taskdom.RecordActivityInput) error {
	a := &taskdom.Activity{
		ID:           uuid.New(),
		TaskID:       in.TaskID,
		ActorID:      in.ActorID,
		ActivityType: in.ActivityType,
		Content:      in.Content,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	f.mu.Lock()
	f.activities[a.ID] = a
	f.mu.Unlock()
	return nil
}

func (f *fakeActivitySvc) ListActivities(_ context.Context, taskID uuid.UUID) ([]*taskdom.Activity, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out []*taskdom.Activity
	for _, a := range f.activities {
		if a.TaskID == taskID && a.DeletedAt == nil {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeActivitySvc) AddComment(_ context.Context, in taskdom.AddCommentInput) (*taskdom.Activity, error) {
	if in.Text == "" {
		return nil, taskdom.ErrCommentTextInvalid
	}
	now := time.Now()
	// In the handler test fake, use the ActorID directly as the member UUID
	// (mimicking what fakeActivityMemberRepo does in integration tests).
	a := &taskdom.Activity{
		ID:           uuid.New(),
		TaskID:       in.TaskID,
		ActorID:      &in.ActorID,
		ActivityType: taskdom.ActivityTypeComment,
		Content:      []byte(`{"text":"` + in.Text + `"}`),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	f.mu.Lock()
	f.activities[a.ID] = a
	f.mu.Unlock()
	return a, nil
}

func (f *fakeActivitySvc) UpdateComment(_ context.Context, id uuid.UUID, _ uuid.UUID, actorID uuid.UUID, text string) (*taskdom.Activity, error) {
	if text == "" {
		return nil, taskdom.ErrCommentTextInvalid
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.activities[id]
	if !ok {
		return nil, taskdom.ErrActivityNotFound
	}
	if a.ActivityType != taskdom.ActivityTypeComment {
		return nil, taskdom.ErrActivityNotAComment
	}
	if a.ActorID == nil || *a.ActorID != actorID {
		return nil, taskdom.ErrActivityForbidden
	}
	a.Content = []byte(`{"text":"` + text + `"}`)
	a.UpdatedAt = time.Now()
	cp := *a
	return &cp, nil
}

func (f *fakeActivitySvc) DeleteComment(_ context.Context, id uuid.UUID, _ uuid.UUID, actorID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.activities[id]
	if !ok {
		return taskdom.ErrActivityNotFound
	}
	if a.ActivityType != taskdom.ActivityTypeComment {
		return taskdom.ErrActivityNotAComment
	}
	if a.ActorID == nil || *a.ActorID != actorID {
		return taskdom.ErrActivityForbidden
	}
	now := time.Now()
	a.DeletedAt = &now
	return nil
}

type fakeViewSvcTask struct{}

func (f *fakeViewSvcTask) ListViews(_ context.Context, _ uuid.UUID) ([]*sprintdom.SprintView, error) {
	return nil, nil
}

func (f *fakeViewSvcTask) ListProjectViews(_ context.Context, _ uuid.UUID, _ sprintdom.ViewContext) ([]*sprintdom.SprintView, error) {
	return nil, nil
}

func (f *fakeViewSvcTask) GetView(_ context.Context, _, _ uuid.UUID) (*sprintdom.SprintView, error) {
	return nil, sprintdom.ErrViewNotFound
}

func (f *fakeViewSvcTask) CreateView(_ context.Context, _ sprintdom.CreateViewInput) (*sprintdom.SprintView, error) {
	return nil, nil
}

func (f *fakeViewSvcTask) UpdateView(_ context.Context, _, _ uuid.UUID, _ sprintdom.UpdateViewInput) (*sprintdom.SprintView, error) {
	return nil, sprintdom.ErrViewNotFound
}

func (f *fakeViewSvcTask) DeleteView(_ context.Context, _, _ uuid.UUID) error { return nil }

func (f *fakeViewSvcTask) MoveTask(_ context.Context, _, _ uuid.UUID, _ sprintdom.MoveTaskInput) error {
	return nil
}

func (f *fakeViewSvcTask) BulkMoveTasks(_ context.Context, _, _ uuid.UUID, _ []sprintdom.MoveTaskInput) error {
	return nil
}

func (f *fakeViewSvcTask) ListTaskPositions(_ context.Context, _, _ uuid.UUID) ([]*sprintdom.ViewTaskPosition, error) {
	return nil, nil
}

func (f *fakeViewSvcTask) ReorderViews(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (f *fakeViewSvcTask) ReorderProjectViews(_ context.Context, _ uuid.UUID, _ sprintdom.ViewContext, _ []uuid.UUID) error {
	return nil
}

// ---------------------------------------------------------------------------
// Router helper
// ---------------------------------------------------------------------------

func buildTaskHandlerRouter(svc *fakeTaskSvc) *gin.Engine {
	return buildTaskHandlerRouterWithActivity(svc, newFakeActivitySvc())
}

func buildTaskHandlerRouterWithActivity(svc *fakeTaskSvc, actSvc *fakeActivitySvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := handler.NewTaskHandler(svc, &fakeViewSvcTask{}, actSvc)
	r := gin.New()
	projectGroup := r.Group("/projects/:projectId")
	projectGroup.GET("/task-types", h.ListTaskTypes)
	projectGroup.POST("/task-types", h.CreateTaskType)
	projectGroup.PATCH("/task-types/:typeId", h.UpdateTaskType)
	projectGroup.DELETE("/task-types/:typeId", h.DeleteTaskType)
	projectGroup.GET("/tasks", h.ListTasks)
	projectGroup.POST("/tasks", h.CreateTask)
	projectGroup.GET("/tasks/by-number/:taskNumber", h.GetTaskByNumber)
	projectGroup.GET("/tasks/:taskId", h.GetTask)
	projectGroup.PATCH("/tasks/:taskId", h.UpdateTask)
	projectGroup.DELETE("/tasks/:taskId", h.DeleteTask)
	// BDD scenarios
	projectGroup.GET("/tasks/:taskId/bdd-scenarios", h.ListBDDScenarios)
	projectGroup.POST("/tasks/:taskId/bdd-scenarios", h.CreateBDDScenario)
	projectGroup.GET("/tasks/:taskId/bdd-scenarios/:scenarioId", h.GetBDDScenario)
	projectGroup.PATCH("/tasks/:taskId/bdd-scenarios/:scenarioId", h.UpdateBDDScenario)
	projectGroup.DELETE("/tasks/:taskId/bdd-scenarios/:scenarioId", h.DeleteBDDScenario)
	// Activities / comments
	projectGroup.GET("/tasks/:taskId/activities", h.ListTaskActivities)
	projectGroup.POST("/tasks/:taskId/activities/comments", h.AddComment)
	projectGroup.PATCH("/tasks/:taskId/activities/comments/:commentId", h.UpdateComment)
	projectGroup.DELETE("/tasks/:taskId/activities/comments/:commentId", h.DeleteComment)
	return r
}

func doTaskRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeTaskID(t *testing.T, body []byte) string {
	t.Helper()
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	id, _ := env.Data["id"].(string)
	if id == "" {
		t.Fatalf("missing id in response: %s", body)
	}
	return id
}

func decodeTaskField(t *testing.T, body []byte, field string) any {
	t.Helper()
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return env.Data[field]
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestTaskHandler_ListTasks_UsesUnifiedFilters(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)

	projectID := uuid.New()
	sprintID := uuid.New()
	statusID := uuid.New()
	assigneeID := uuid.New()
	taskType1 := uuid.New()
	taskType2 := uuid.New()

	path := fmt.Sprintf(
		"/projects/%s/tasks?sprint_id=%s&status_ids=%s&assignee_ids=%s&task_type_ids=%s,%s",
		projectID,
		sprintID,
		statusID,
		assigneeID,
		taskType1,
		taskType2,
	)
	w := doTaskRequest(r, http.MethodGet, path, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.lastProjectID != projectID {
		t.Fatalf("expected project id %s, got %s", projectID, svc.lastProjectID)
	}
	if svc.lastFilter.SprintID == nil || *svc.lastFilter.SprintID != sprintID {
		t.Fatalf("expected sprint filter %s, got %+v", sprintID, svc.lastFilter.SprintID)
	}
	if len(svc.lastFilter.StatusIDs) != 1 || svc.lastFilter.StatusIDs[0] != statusID {
		t.Fatalf("unexpected status ids: %+v", svc.lastFilter.StatusIDs)
	}
	if len(svc.lastFilter.AssigneeIDs) != 1 || svc.lastFilter.AssigneeIDs[0] != assigneeID {
		t.Fatalf("unexpected assignee ids: %+v", svc.lastFilter.AssigneeIDs)
	}
	if len(svc.lastFilter.TaskTypeIDs) != 2 || svc.lastFilter.TaskTypeIDs[0] != taskType1 || svc.lastFilter.TaskTypeIDs[1] != taskType2 {
		t.Fatalf("unexpected task type ids: %+v", svc.lastFilter.TaskTypeIDs)
	}
}

func TestTaskHandler_ListTasks_SprintIDsTakePriority(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	sprintID1 := uuid.New()
	sprintID2 := uuid.New()

	w := doTaskRequest(r, http.MethodGet, fmt.Sprintf("/projects/%s/tasks?sprint_id=null&sprint_ids=%s,%s", projectID, sprintID1, sprintID2), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.lastFilter.BacklogOnly {
		t.Fatal("expected BacklogOnly=false when sprint_ids are provided")
	}
	if len(svc.lastFilter.SprintIDs) != 2 || svc.lastFilter.SprintIDs[0] != sprintID1 || svc.lastFilter.SprintIDs[1] != sprintID2 {
		t.Fatalf("unexpected sprint ids: %+v", svc.lastFilter.SprintIDs)
	}
}

func TestTaskHandler_ListTasks_SprintNullMeansBacklog(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	w := doTaskRequest(r, http.MethodGet, fmt.Sprintf("/projects/%s/tasks?sprint_id=null", projectID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !svc.lastFilter.BacklogOnly {
		t.Fatal("expected BacklogOnly=true when sprint_id=null")
	}
}

func TestTaskHandler_ListTasks_TaskTypeIDsDriveFiltering(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskTypeID := uuid.New()

	w := doTaskRequest(r, http.MethodGet, fmt.Sprintf("/projects/%s/tasks?task_type_ids=%s", projectID, taskTypeID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(svc.lastFilter.TaskTypeIDs) != 1 || svc.lastFilter.TaskTypeIDs[0] != taskTypeID {
		t.Fatalf("unexpected task type ids: %+v", svc.lastFilter.TaskTypeIDs)
	}
}

func TestTaskHandler_CreateTask_Returns201(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	w := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": "New Task"},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	id := decodeTaskID(t, w.Body.Bytes())
	if id == "" {
		t.Error("expected non-empty id in response")
	}
}

func TestTaskHandler_CreateTask_EmptyTitleReturns400(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	w := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": ""},
	)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_GetTask_Returns200(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	// Create first
	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": "Test Task"},
	)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: got %d", createW.Code)
	}
	taskID := decodeTaskID(t, createW.Body.Bytes())

	// Get by ID
	getW := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s", projectID, taskID),
		nil,
	)
	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", getW.Code, getW.Body.String())
	}
	gotID := decodeTaskID(t, getW.Body.Bytes())
	if gotID != taskID {
		t.Errorf("expected id %s, got %s", taskID, gotID)
	}
}

func TestTaskHandler_GetTask_NotFoundReturns404(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	w := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s", projectID, uuid.New()),
		nil,
	)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestTaskHandler_UpdateTask_StatusOnlyPreservesSprintID verifies that a PATCH
// with only status_id does not clear other fields (the partial-update bug fix).
func TestTaskHandler_UpdateTask_StatusOnlyPreservesSprintID(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	sprintID := uuid.New()
	statusID := uuid.New()

	// Create a task assigned to a sprint.
	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{
			"title":     "Sprint Task",
			"sprint_id": sprintID.String(),
		},
	)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: got %d", createW.Code)
	}
	taskID := decodeTaskID(t, createW.Body.Bytes())

	// PATCH with only status_id — sprint_id must not be cleared.
	patchW := doTaskRequest(r, http.MethodPatch,
		fmt.Sprintf("/projects/%s/tasks/%s", projectID, taskID),
		map[string]any{"status_id": statusID.String()},
	)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", patchW.Code, patchW.Body.String())
	}
	gotSprintID := decodeTaskField(t, patchW.Body.Bytes(), "sprint_id")
	if gotSprintID != sprintID.String() {
		t.Errorf("expected sprint_id %s to be preserved, got %v", sprintID, gotSprintID)
	}
}

// TestTaskHandler_UpdateTask_NullSprintIDClearsField verifies that sending
// sprint_id=null explicitly removes the sprint assignment.
func TestTaskHandler_UpdateTask_NullSprintIDClearsField(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	sprintID := uuid.New()

	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": "Sprint Task", "sprint_id": sprintID.String()},
	)
	taskID := decodeTaskID(t, createW.Body.Bytes())

	// Send sprint_id: null explicitly.
	body := []byte(`{"sprint_id": null}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		fmt.Sprintf("/projects/%s/tasks/%s", projectID, taskID),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	gotSprintID := decodeTaskField(t, w.Body.Bytes(), "sprint_id")
	if gotSprintID != nil {
		t.Errorf("expected sprint_id=nil after explicit null, got %v", gotSprintID)
	}
}

func TestTaskHandler_DeleteTask_Returns200(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": "Delete Me"},
	)
	taskID := decodeTaskID(t, createW.Body.Bytes())

	delW := doTaskRequest(r, http.MethodDelete,
		fmt.Sprintf("/projects/%s/tasks/%s", projectID, taskID),
		nil,
	)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", delW.Code, delW.Body.String())
	}
}

func TestTaskHandler_DeleteTask_NotFoundReturns404(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	w := doTaskRequest(r, http.MethodDelete,
		fmt.Sprintf("/projects/%s/tasks/%s", projectID, uuid.New()),
		nil,
	)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_CreateTaskType_Returns201(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	w := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/task-types", projectID),
		map[string]any{"name": "Bug"},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_InvalidTaskID_Returns400(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	w := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/not-a-uuid", projectID),
		nil,
	)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid task id, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Task Number handler tests
// ---------------------------------------------------------------------------

func TestTaskHandler_CreateTask_ResponseContainsTaskNumber(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	w := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": "Numbered Task"},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	taskNum := decodeTaskField(t, w.Body.Bytes(), "task_number")
	if taskNum == nil {
		t.Fatal("expected task_number in response")
	}
	if taskNum.(float64) < 1 {
		t.Errorf("expected task_number >= 1, got %v", taskNum)
	}
}

func TestTaskHandler_GetTaskByNumber_Returns200(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	// Create task first to get a task number.
	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks", projectID),
		map[string]any{"title": "Task by number"},
	)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: got %d", createW.Code)
	}
	taskNum := decodeTaskField(t, createW.Body.Bytes(), "task_number").(float64)
	originalID := decodeTaskID(t, createW.Body.Bytes())

	// Look up by task number.
	getW := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/by-number/%d", projectID, int64(taskNum)),
		nil,
	)
	if getW.Code != http.StatusOK {
		t.Fatalf("get by number: expected 200, got %d: %s", getW.Code, getW.Body.String())
	}
	gotID := decodeTaskID(t, getW.Body.Bytes())
	if gotID != originalID {
		t.Errorf("expected id %s, got %s", originalID, gotID)
	}
}

func TestTaskHandler_GetTaskByNumber_NotFoundReturns404(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	w := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/by-number/9999", projectID),
		nil,
	)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_GetTaskByNumber_InvalidNumberReturns400(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()

	w := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/by-number/not-a-number", projectID),
		nil,
	)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid task number, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// BDD Scenario handler tests
// ---------------------------------------------------------------------------

func TestBDDScenarioHandler_CreateAndList(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	// Pre-seed a task so the fake service's GetTask can resolve it.
	task := &taskdom.Task{ID: taskID, ProjectID: projectID, Title: "feat", Tags: []string{}, CustomFields: map[string]any{}}
	svc.mu.Lock()
	svc.tasks[taskID] = task
	svc.mu.Unlock()

	// Create a BDD scenario.
	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios", projectID, taskID),
		map[string]any{"title": "User login", "given": "the user is on login page", "when": "they submit valid credentials", "then": "they are redirected"},
	)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	// List BDD scenarios.
	listW := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios", projectID, taskID),
		nil,
	)
	if listW.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", listW.Code, listW.Body.String())
	}
	var listEnv struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listEnv); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listEnv.Data.Items) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(listEnv.Data.Items))
	}
}

func TestBDDScenarioHandler_CreateMissingTitle(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	// Pre-seed a task so the ownership check passes (GetTask must resolve it).
	task := &taskdom.Task{ID: taskID, ProjectID: projectID, Title: "feat", Tags: []string{}, CustomFields: map[string]any{}}
	svc.mu.Lock()
	svc.tasks[taskID] = task
	svc.mu.Unlock()

	w := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios", projectID, taskID),
		map[string]any{"given": "some context"},
	)
	// binding:"required" on Title → 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing title, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBDDScenarioHandler_GetNotFound(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	w := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios/%s", projectID, taskID, uuid.New()),
		nil,
	)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBDDScenarioHandler_UpdateAndGet(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	task := &taskdom.Task{ID: taskID, ProjectID: projectID, Title: "feat", Tags: []string{}, CustomFields: map[string]any{}}
	svc.mu.Lock()
	svc.tasks[taskID] = task
	svc.mu.Unlock()

	// Create.
	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios", projectID, taskID),
		map[string]any{"title": "Original"},
	)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", createW.Code)
	}
	var createEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &createEnv); err != nil {
		t.Fatalf("json.Unmarshal createEnv: %v", err)
	}
	scenarioID := createEnv.Data.ID

	// Patch title.
	newTitle := "Updated title"
	patchW := doTaskRequest(r, http.MethodPatch,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios/%s", projectID, taskID, scenarioID),
		map[string]any{"title": newTitle},
	)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", patchW.Code, patchW.Body.String())
	}

	// Get and verify updated title.
	getW := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios/%s", projectID, taskID, scenarioID),
		nil,
	)
	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getW.Code)
	}
	var getEnv struct {
		Data struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &getEnv); err != nil {
		t.Fatalf("json.Unmarshal getEnv: %v", err)
	}
	if getEnv.Data.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, getEnv.Data.Title)
	}
}

func TestBDDScenarioHandler_Delete(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	task := &taskdom.Task{ID: taskID, ProjectID: projectID, Title: "feat", Tags: []string{}, CustomFields: map[string]any{}}
	svc.mu.Lock()
	svc.tasks[taskID] = task
	svc.mu.Unlock()

	// Create.
	createW := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios", projectID, taskID),
		map[string]any{"title": "To delete"},
	)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", createW.Code)
	}
	var createEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &createEnv); err != nil {
		t.Fatalf("json.Unmarshal createEnv: %v", err)
	}
	scenarioID := createEnv.Data.ID

	// Delete.
	delW := doTaskRequest(r, http.MethodDelete,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios/%s", projectID, taskID, scenarioID),
		nil,
	)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	// List should now be empty.
	listW := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios", projectID, taskID),
		nil,
	)
	var listEnv struct {
		Data struct {
			Items []any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listEnv); err != nil {
		t.Fatalf("json.Unmarshal listEnv: %v", err)
	}
	if len(listEnv.Data.Items) != 0 {
		t.Errorf("expected 0 items after delete, got %d", len(listEnv.Data.Items))
	}
}

func TestBDDScenarioHandler_DeleteNotFound(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	w := doTaskRequest(r, http.MethodDelete,
		fmt.Sprintf("/projects/%s/tasks/%s/bdd-scenarios/%s", projectID, taskID, uuid.New()),
		nil,
	)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Helpers for actor-aware requests
// ---------------------------------------------------------------------------

func doTaskRequestWithActor(r *gin.Engine, method, path string, body any, actorID uuid.UUID) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequestWithContext(middleware.WithActorID(context.Background(), actorID), method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Activity handler tests
// ---------------------------------------------------------------------------

func TestActivityHandler_ListEmpty(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	w := doTaskRequest(r, http.MethodGet,
		fmt.Sprintf("/projects/%s/tasks/%s/activities", projectID, taskID),
		nil,
	)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivityHandler_AddComment(t *testing.T) {
	svc := newFakeTaskSvc()
	actSvc := newFakeActivitySvc()
	r := buildTaskHandlerRouterWithActivity(svc, actSvc)
	projectID := uuid.New()
	taskID := uuid.New()
	actorID := uuid.New()

	w := doTaskRequestWithActor(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments", projectID, taskID),
		map[string]any{"text": "hello world"},
		actorID,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivityHandler_AddComment_NoActor(t *testing.T) {
	svc := newFakeTaskSvc()
	r := buildTaskHandlerRouter(svc)
	projectID := uuid.New()
	taskID := uuid.New()

	w := doTaskRequest(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments", projectID, taskID),
		map[string]any{"text": "hello world"},
	)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivityHandler_AddComment_EmptyText(t *testing.T) {
	svc := newFakeTaskSvc()
	actSvc := newFakeActivitySvc()
	r := buildTaskHandlerRouterWithActivity(svc, actSvc)
	projectID := uuid.New()
	taskID := uuid.New()
	actorID := uuid.New()

	w := doTaskRequestWithActor(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments", projectID, taskID),
		map[string]any{"text": ""},
		actorID,
	)
	// Empty text fails binding (required) or service validation
	if w.Code == http.StatusCreated {
		t.Fatalf("expected error, got 201")
	}
}

func TestActivityHandler_UpdateAndDeleteComment(t *testing.T) {
	svc := newFakeTaskSvc()
	actSvc := newFakeActivitySvc()
	r := buildTaskHandlerRouterWithActivity(svc, actSvc)
	projectID := uuid.New()
	taskID := uuid.New()
	actorID := uuid.New()

	// Add a comment first
	w := doTaskRequestWithActor(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments", projectID, taskID),
		map[string]any{"text": "original"},
		actorID,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("add comment: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	commentID := created.Data.ID

	// Update the comment
	w = doTaskRequestWithActor(r, http.MethodPatch,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments/%s", projectID, taskID, commentID),
		map[string]any{"text": "updated"},
		actorID,
	)
	if w.Code != http.StatusOK {
		t.Fatalf("update comment: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete the comment
	w = doTaskRequestWithActor(r, http.MethodDelete,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments/%s", projectID, taskID, commentID),
		nil,
		actorID,
	)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete comment: expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivityHandler_UpdateComment_Forbidden(t *testing.T) {
	svc := newFakeTaskSvc()
	actSvc := newFakeActivitySvc()
	r := buildTaskHandlerRouterWithActivity(svc, actSvc)
	projectID := uuid.New()
	taskID := uuid.New()
	actorID := uuid.New()
	otherActor := uuid.New()

	// Add comment as actorID
	w := doTaskRequestWithActor(r, http.MethodPost,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments", projectID, taskID),
		map[string]any{"text": "mine"},
		actorID,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("add comment: expected 201, got %d", w.Code)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Try to update as different actor
	w = doTaskRequestWithActor(r, http.MethodPatch,
		fmt.Sprintf("/projects/%s/tasks/%s/activities/comments/%s", projectID, taskID, created.Data.ID),
		map[string]any{"text": "hacked"},
		otherActor,
	)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
