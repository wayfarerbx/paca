package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	"github.com/paca/api/internal/platform/authz"
	jwttoken "github.com/paca/api/internal/platform/token"
	authsvc "github.com/paca/api/internal/service/auth"
	projectsvc "github.com/paca/api/internal/service/project"
	sprintsvc "github.com/paca/api/internal/service/sprint"
	tasksvc "github.com/paca/api/internal/service/task"
	usersvc "github.com/paca/api/internal/service/user"
	"github.com/paca/api/internal/transport/http/handler"
	"github.com/paca/api/internal/transport/http/router"
)

// ---------------------------------------------------------------------------
// Fake ViewRepository for integration tests
// ---------------------------------------------------------------------------

type fakeViewRepoIT struct {
	mu        sync.RWMutex
	views     map[uuid.UUID]*sprintdom.SprintView
	positions map[string]*sprintdom.ViewTaskPosition
}

func newFakeViewRepoIT() *fakeViewRepoIT {
	return &fakeViewRepoIT{
		views:     make(map[uuid.UUID]*sprintdom.SprintView),
		positions: make(map[string]*sprintdom.ViewTaskPosition),
	}
}

func viewPosKey(viewID, taskID uuid.UUID) string {
	return viewID.String() + ":" + taskID.String()
}

func (r *fakeViewRepoIT) ListViews(_ context.Context, sprintID uuid.UUID) ([]*sprintdom.SprintView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*sprintdom.SprintView
	for _, v := range r.views {
		if v.SprintID != nil && *v.SprintID == sprintID {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeViewRepoIT) ListBacklogViews(_ context.Context, projectID uuid.UUID) ([]*sprintdom.SprintView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*sprintdom.SprintView
	for _, v := range r.views {
		if v.SprintID == nil && v.ProjectID == projectID {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeViewRepoIT) FindViewByID(_ context.Context, id uuid.UUID) (*sprintdom.SprintView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.views[id]
	if !ok {
		return nil, sprintdom.ErrViewNotFound
	}
	cp := *v
	return &cp, nil
}

func (r *fakeViewRepoIT) CreateView(_ context.Context, v *sprintdom.SprintView) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *v
	r.views[v.ID] = &cp
	return nil
}

func (r *fakeViewRepoIT) UpdateView(_ context.Context, v *sprintdom.SprintView) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.views[v.ID]; !ok {
		return sprintdom.ErrViewNotFound
	}
	cp := *v
	r.views[v.ID] = &cp
	return nil
}

func (r *fakeViewRepoIT) DeleteView(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.views, id)
	return nil
}

func (r *fakeViewRepoIT) CountViews(_ context.Context, sprintID uuid.UUID) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, v := range r.views {
		if v.SprintID != nil && *v.SprintID == sprintID {
			count++
		}
	}
	return count, nil
}

func (r *fakeViewRepoIT) CountBacklogViews(_ context.Context, projectID uuid.UUID) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, v := range r.views {
		if v.SprintID == nil && v.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (r *fakeViewRepoIT) UpsertTaskPosition(_ context.Context, pos *sprintdom.ViewTaskPosition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *pos
	r.positions[viewPosKey(pos.ViewID, pos.TaskID)] = &cp
	return nil
}

func (r *fakeViewRepoIT) ListTaskPositions(_ context.Context, viewID uuid.UUID) ([]*sprintdom.ViewTaskPosition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*sprintdom.ViewTaskPosition
	for _, p := range r.positions {
		if p.ViewID == viewID {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Router builder
// ---------------------------------------------------------------------------

func buildViewTestRouter(viewRepo *fakeViewRepoIT, sprintRepo *fakeSprintRepoIT, store *projectPermStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	tm := jwttoken.New(testSecret, 15*time.Minute, 168*time.Hour)
	refreshStore := &fakeRefreshStore{}
	userRepo := newFakeUserRepo()
	authService := authsvc.New(userRepo, tm, refreshStore, 168*time.Hour, 24*time.Hour)
	userService := usersvc.New(userRepo)
	projectRepo := newFakeProjectRepo()
	taskRepo := newFakeTaskRepoIT()
	projectService := projectsvc.New(projectRepo, taskRepo)
	taskService := tasksvc.New(taskRepo)
	sprintService := sprintsvc.New(sprintRepo)
	viewService := sprintsvc.NewViewService(viewRepo)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	return router.New(router.Deps{
		TokenManager: tm,
		Authorizer:   authz.NewAuthorizer(store),
		Health:       handler.NewHealthHandler(),
		Auth:         handler.NewAuthHandler(authService, testCookieCfg),
		User:         handler.NewUserHandler(userService),
		GlobalRole:   handler.NewGlobalRoleHandler(&fakeGlobalRoleService{}),
		Project:      handler.NewProjectHandler(projectService, authz.NewAuthorizer(store)),
		Task:         handler.NewTaskHandler(taskService),
		Sprint:       handler.NewSprintHandler(sprintService, viewService),
		View:         handler.NewViewHandler(viewService),
		Log:          log,
	})
}

func issueViewToken(t *testing.T, subject string) string {
	t.Helper()
	tm := jwttoken.New(testSecret, 15*time.Minute, 168*time.Hour)
	tok, err := tm.IssueAccess(subject, "view-user", "USER", "fam-view", false)
	if err != nil {
		t.Fatalf("issue view token: %v", err)
	}
	return tok
}

// viewIDFrom decodes data.id from a handler JSON response.
func viewIDFrom(t *testing.T, body []byte) string {
	t.Helper()
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode view response: %v", err)
	}
	id, _ := env.Data["id"].(string)
	if id == "" {
		t.Fatalf("missing id in view response: %s", string(body))
	}
	return id
}

// viewListCount decodes data.items and returns its length.
func viewListCount(t *testing.T, body []byte) int {
	t.Helper()
	var env struct {
		Data struct {
			Items []any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode view list response: %v", err)
	}
	return len(env.Data.Items)
}

// ---------------------------------------------------------------------------
// Helper — seed a sprint into the fake repo
// ---------------------------------------------------------------------------

func seedSprintIT(t *testing.T, repo *fakeSprintRepoIT, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ctx := context.Background()
	if err := repo.CreateSprint(ctx, &sprintdom.Sprint{
		ID:        id,
		ProjectID: projectID,
		Name:      "Sprint IT",
		Status:    sprintdom.SprintStatusPlanned,
	}); err != nil {
		t.Fatalf("seed sprint: %v", err)
	}
	return id
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestIntegrationViews_CRUD(t *testing.T) {
	projectID := uuid.New()
	store := &projectPermStore{
		projectPerms: map[uuid.UUID][]authz.Permission{
			projectID: {
				authz.PermissionSprintsRead,
				authz.PermissionSprintsWrite,
				authz.PermissionTasksRead,
				authz.PermissionTasksWrite,
			},
		},
	}
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, store)
	tok := issueViewToken(t, uuid.NewString())
	sprintID := seedSprintIT(t, sprintRepo, projectID)

	base := fmt.Sprintf("/api/v1/projects/%s/sprints/%s/views", projectID, sprintID)

	// Create
	createW := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Board View",
		"view_type": "board",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create view: expected 201, got %d (%s)", createW.Code, createW.Body.String())
	}
	viewID := viewIDFrom(t, createW.Body.Bytes())

	// List
	listW := serve(r, authedJSONReq(t.Context(), http.MethodGet, base, tok, nil))
	if listW.Code != http.StatusOK {
		t.Fatalf("list views: expected 200, got %d (%s)", listW.Code, listW.Body.String())
	}
	if count := viewListCount(t, listW.Body.Bytes()); count != 1 {
		t.Errorf("expected 1 view, got %d", count)
	}

	// Get single
	getW := serve(r, authedJSONReq(t.Context(), http.MethodGet, base+"/"+viewID, tok, nil))
	if getW.Code != http.StatusOK {
		t.Fatalf("get view: expected 200, got %d (%s)", getW.Code, getW.Body.String())
	}

	// Update name
	patchW := serve(r, authedJSONReq(t.Context(), http.MethodPatch, base+"/"+viewID, tok, map[string]any{
		"name": "Renamed Board",
	}))
	if patchW.Code != http.StatusOK {
		t.Fatalf("update view: expected 200, got %d (%s)", patchW.Code, patchW.Body.String())
	}

	// Create a second view so deletion of first is allowed
	create2W := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Table View",
		"view_type": "table",
	}))
	if create2W.Code != http.StatusCreated {
		t.Fatalf("create second view: expected 201, got %d (%s)", create2W.Code, create2W.Body.String())
	}

	// Delete first view
	delW := serve(r, authedJSONReq(t.Context(), http.MethodDelete, base+"/"+viewID, tok, nil))
	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete view: expected 204, got %d (%s)", delW.Code, delW.Body.String())
	}

	// Verify removed
	getDeleted := serve(r, authedJSONReq(t.Context(), http.MethodGet, base+"/"+viewID, tok, nil))
	if getDeleted.Code != http.StatusNotFound {
		t.Fatalf("get deleted: expected 404, got %d", getDeleted.Code)
	}
	if code := decodeErrorCode(t, getDeleted); code != "VIEW_NOT_FOUND" {
		t.Errorf("expected VIEW_NOT_FOUND, got %q", code)
	}
}

func TestIntegrationViews_DeleteLastViewRejected(t *testing.T) {
	projectID := uuid.New()
	store := &projectPermStore{
		projectPerms: map[uuid.UUID][]authz.Permission{
			projectID: {authz.PermissionSprintsRead, authz.PermissionSprintsWrite},
		},
	}
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, store)
	tok := issueViewToken(t, uuid.NewString())
	sprintID := seedSprintIT(t, sprintRepo, projectID)
	base := fmt.Sprintf("/api/v1/projects/%s/sprints/%s/views", projectID, sprintID)

	createW := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Only View",
		"view_type": "table",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create view: expected 201, got %d", createW.Code)
	}
	viewID := viewIDFrom(t, createW.Body.Bytes())

	delW := serve(r, authedJSONReq(t.Context(), http.MethodDelete, base+"/"+viewID, tok, nil))
	if delW.Code != http.StatusConflict {
		t.Fatalf("delete last view: expected 409, got %d (%s)", delW.Code, delW.Body.String())
	}
	if code := decodeErrorCode(t, delW); code != "VIEW_IS_LAST_VIEW" {
		t.Errorf("expected VIEW_IS_LAST_VIEW, got %q", code)
	}
}

func TestIntegrationViews_CreateInvalidTypeReturns400(t *testing.T) {
	projectID := uuid.New()
	store := &projectPermStore{
		projectPerms: map[uuid.UUID][]authz.Permission{
			projectID: {authz.PermissionSprintsRead, authz.PermissionSprintsWrite},
		},
	}
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, store)
	tok := issueViewToken(t, uuid.NewString())
	sprintID := seedSprintIT(t, sprintRepo, projectID)
	base := fmt.Sprintf("/api/v1/projects/%s/sprints/%s/views", projectID, sprintID)

	w := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Bad",
		"view_type": "gantt",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if code := decodeErrorCode(t, w); code != "VIEW_TYPE_INVALID" {
		t.Errorf("expected VIEW_TYPE_INVALID, got %q", code)
	}
}

func TestIntegrationViews_TaskPositions(t *testing.T) {
	projectID := uuid.New()
	store := &projectPermStore{
		projectPerms: map[uuid.UUID][]authz.Permission{
			projectID: {
				authz.PermissionSprintsRead,
				authz.PermissionSprintsWrite,
				authz.PermissionTasksRead,
				authz.PermissionTasksWrite,
			},
		},
	}
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, store)
	tok := issueViewToken(t, uuid.NewString())
	sprintID := seedSprintIT(t, sprintRepo, projectID)
	base := fmt.Sprintf("/api/v1/projects/%s/sprints/%s/views", projectID, sprintID)

	// Create view
	createW := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Table",
		"view_type": "table",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create view: expected 201, got %d", createW.Code)
	}
	viewID := viewIDFrom(t, createW.Body.Bytes())

	taskID := uuid.NewString()
	posURL := fmt.Sprintf("%s/%s/task-positions/%s", base, viewID, taskID)

	// Move task
	moveW := serve(r, authedJSONReq(t.Context(), http.MethodPut, posURL, tok, map[string]any{
		"position":  5,
		"group_key": "in-progress",
	}))
	if moveW.Code != http.StatusNoContent {
		t.Fatalf("move task: expected 204, got %d (%s)", moveW.Code, moveW.Body.String())
	}

	// List positions
	listPosURL := fmt.Sprintf("%s/%s/task-positions", base, viewID)
	listW := serve(r, authedJSONReq(t.Context(), http.MethodGet, listPosURL, tok, nil))
	if listW.Code != http.StatusOK {
		t.Fatalf("list positions: expected 200, got %d (%s)", listW.Code, listW.Body.String())
	}

	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(listW.Body).Decode(&env); err != nil {
		t.Fatalf("decode positions: %v", err)
	}
	if len(env.Data.Items) != 1 {
		t.Fatalf("expected 1 position, got %d", len(env.Data.Items))
	}
	pos := env.Data.Items[0]
	if pos["task_id"] != taskID {
		t.Errorf("task_id mismatch: %v", pos["task_id"])
	}
}

func TestIntegrationViews_AuthzGuard(t *testing.T) {
	projectID := uuid.New()
	// No permissions at all
	store := &projectPermStore{}
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, store)
	tok := issueViewToken(t, uuid.NewString())
	sprintID := uuid.New()
	base := fmt.Sprintf("/api/v1/projects/%s/sprints/%s/views", projectID, sprintID)

	w := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Unauthorized",
		"view_type": "table",
	}))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Product-backlog view integration tests
// ---------------------------------------------------------------------------

func newBacklogViewPerms(projectID uuid.UUID) *projectPermStore {
	return &projectPermStore{
		projectPerms: map[uuid.UUID][]authz.Permission{
			projectID: {
				authz.PermissionSprintsRead,
				authz.PermissionSprintsWrite,
				authz.PermissionTasksRead,
				authz.PermissionTasksWrite,
			},
		},
	}
}

func TestIntegrationBacklogViews_CRUD(t *testing.T) {
	projectID := uuid.New()
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, newBacklogViewPerms(projectID))
	tok := issueViewToken(t, uuid.NewString())

	base := fmt.Sprintf("/api/v1/projects/%s/product-backlog/views", projectID)

	// Create first backlog view
	createW := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Backlog Board",
		"view_type": "board",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create backlog view: expected 201, got %d (%s)", createW.Code, createW.Body.String())
	}
	viewID := viewIDFrom(t, createW.Body.Bytes())

	// Verify sprint_id is absent / null
	var createResp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(createW.Body.Bytes(), &createResp)
	if sid, ok := createResp.Data["sprint_id"]; ok && sid != nil {
		t.Errorf("expected sprint_id to be absent/null for backlog view, got %v", sid)
	}
	if pid, _ := createResp.Data["project_id"].(string); pid != projectID.String() {
		t.Errorf("expected project_id=%s, got %q", projectID, pid)
	}

	// List — should see the created view
	listW := serve(r, authedJSONReq(t.Context(), http.MethodGet, base, tok, nil))
	if listW.Code != http.StatusOK {
		t.Fatalf("list backlog views: expected 200, got %d (%s)", listW.Code, listW.Body.String())
	}
	if count := viewListCount(t, listW.Body.Bytes()); count != 1 {
		t.Errorf("expected 1 backlog view, got %d", count)
	}

	// Get single
	getW := serve(r, authedJSONReq(t.Context(), http.MethodGet, base+"/"+viewID, tok, nil))
	if getW.Code != http.StatusOK {
		t.Fatalf("get backlog view: expected 200, got %d (%s)", getW.Code, getW.Body.String())
	}
	var getResp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(getW.Body.Bytes(), &getResp)
	if getResp.Data["id"] != viewID {
		t.Errorf("get view id mismatch: want %s, got %v", viewID, getResp.Data["id"])
	}

	// Update name
	patchW := serve(r, authedJSONReq(t.Context(), http.MethodPatch, base+"/"+viewID, tok, map[string]any{
		"name": "Renamed Backlog Board",
	}))
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch backlog view: expected 200, got %d (%s)", patchW.Code, patchW.Body.String())
	}
	var patchResp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(patchW.Body.Bytes(), &patchResp)
	if patchResp.Data["name"] != "Renamed Backlog Board" {
		t.Errorf("expected name=Renamed Backlog Board, got %v", patchResp.Data["name"])
	}

	// Create second view so we can delete the first
	create2W := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Backlog Table",
		"view_type": "table",
	}))
	if create2W.Code != http.StatusCreated {
		t.Fatalf("create second backlog view: expected 201, got %d", create2W.Code)
	}

	// Delete first view
	delW := serve(r, authedJSONReq(t.Context(), http.MethodDelete, base+"/"+viewID, tok, nil))
	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete backlog view: expected 204, got %d (%s)", delW.Code, delW.Body.String())
	}

	// Verify removed
	getDeletedW := serve(r, authedJSONReq(t.Context(), http.MethodGet, base+"/"+viewID, tok, nil))
	if getDeletedW.Code != http.StatusNotFound {
		t.Fatalf("deleted backlog view: expected 404, got %d", getDeletedW.Code)
	}
}

func TestIntegrationBacklogViews_DeleteLastViewRejected(t *testing.T) {
	projectID := uuid.New()
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, newBacklogViewPerms(projectID))
	tok := issueViewToken(t, uuid.NewString())
	base := fmt.Sprintf("/api/v1/projects/%s/product-backlog/views", projectID)

	createW := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Only Backlog View",
		"view_type": "table",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create backlog view: expected 201, got %d", createW.Code)
	}
	viewID := viewIDFrom(t, createW.Body.Bytes())

	delW := serve(r, authedJSONReq(t.Context(), http.MethodDelete, base+"/"+viewID, tok, nil))
	if delW.Code != http.StatusConflict {
		t.Fatalf("delete last backlog view: expected 409, got %d (%s)", delW.Code, delW.Body.String())
	}
	if code := decodeErrorCode(t, delW); code != "VIEW_IS_LAST_VIEW" {
		t.Errorf("expected VIEW_IS_LAST_VIEW, got %q", code)
	}
}

func TestIntegrationBacklogViews_SprintViewsNotIncluded(t *testing.T) {
	projectID := uuid.New()
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, newBacklogViewPerms(projectID))
	tok := issueViewToken(t, uuid.NewString())
	sprintID := seedSprintIT(t, sprintRepo, projectID)

	backlogBase := fmt.Sprintf("/api/v1/projects/%s/product-backlog/views", projectID)
	sprintBase := fmt.Sprintf("/api/v1/projects/%s/sprints/%s/views", projectID, sprintID)

	// Create a sprint view and a backlog view
	w1 := serve(r, authedJSONReq(t.Context(), http.MethodPost, sprintBase, tok, map[string]any{
		"name":      "Sprint View",
		"view_type": "board",
	}))
	if w1.Code != http.StatusCreated {
		t.Fatalf("create sprint view: expected 201, got %d", w1.Code)
	}
	w2 := serve(r, authedJSONReq(t.Context(), http.MethodPost, backlogBase, tok, map[string]any{
		"name":      "Backlog View",
		"view_type": "table",
	}))
	if w2.Code != http.StatusCreated {
		t.Fatalf("create backlog view: expected 201, got %d", w2.Code)
	}

	// Backlog list must contain exactly 1 item (the backlog view)
	listW := serve(r, authedJSONReq(t.Context(), http.MethodGet, backlogBase, tok, nil))
	if listW.Code != http.StatusOK {
		t.Fatalf("list backlog views: expected 200, got %d", listW.Code)
	}
	if count := viewListCount(t, listW.Body.Bytes()); count != 1 {
		t.Errorf("expected 1 backlog view, got %d", count)
	}

	// Sprint list must also contain exactly 1 item (the sprint view)
	sprintListW := serve(r, authedJSONReq(t.Context(), http.MethodGet, sprintBase, tok, nil))
	if sprintListW.Code != http.StatusOK {
		t.Fatalf("list sprint views: expected 200, got %d", sprintListW.Code)
	}
	if count := viewListCount(t, sprintListW.Body.Bytes()); count != 1 {
		t.Errorf("expected 1 sprint view, got %d", count)
	}
}

func TestIntegrationBacklogViews_TaskPositions(t *testing.T) {
	projectID := uuid.New()
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, newBacklogViewPerms(projectID))
	tok := issueViewToken(t, uuid.NewString())
	base := fmt.Sprintf("/api/v1/projects/%s/product-backlog/views", projectID)

	createW := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Backlog Table",
		"view_type": "table",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create backlog view: expected 201, got %d", createW.Code)
	}
	viewID := viewIDFrom(t, createW.Body.Bytes())
	taskID := uuid.NewString()

	// Move task
	moveW := serve(r, authedJSONReq(t.Context(), http.MethodPut,
		fmt.Sprintf("%s/%s/task-positions/%s", base, viewID, taskID), tok,
		map[string]any{"position": 3, "group_key": "todo"},
	))
	if moveW.Code != http.StatusNoContent {
		t.Fatalf("move task: expected 204, got %d (%s)", moveW.Code, moveW.Body.String())
	}

	// List positions
	listW := serve(r, authedJSONReq(t.Context(), http.MethodGet,
		fmt.Sprintf("%s/%s/task-positions", base, viewID), tok, nil,
	))
	if listW.Code != http.StatusOK {
		t.Fatalf("list positions: expected 200, got %d (%s)", listW.Code, listW.Body.String())
	}
	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(listW.Body).Decode(&env); err != nil {
		t.Fatalf("decode positions: %v", err)
	}
	if len(env.Data.Items) != 1 {
		t.Fatalf("expected 1 position, got %d", len(env.Data.Items))
	}
	if env.Data.Items[0]["task_id"] != taskID {
		t.Errorf("task_id mismatch: %v", env.Data.Items[0]["task_id"])
	}
}

func TestIntegrationBacklogViews_AuthzGuard(t *testing.T) {
	projectID := uuid.New()
	// No permissions at all
	store := &projectPermStore{}
	sprintRepo := newFakeSprintRepoIT()
	viewRepo := newFakeViewRepoIT()
	r := buildViewTestRouter(viewRepo, sprintRepo, store)
	tok := issueViewToken(t, uuid.NewString())
	base := fmt.Sprintf("/api/v1/projects/%s/product-backlog/views", projectID)

	w := serve(r, authedJSONReq(t.Context(), http.MethodPost, base, tok, map[string]any{
		"name":      "Unauthorized",
		"view_type": "table",
	}))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
