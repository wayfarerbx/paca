package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	githubdom "github.com/paca/api/internal/domain/github"
	"github.com/paca/api/internal/platform/authz"
	jwttoken "github.com/paca/api/internal/platform/token"
	authsvc "github.com/paca/api/internal/service/auth"
	"github.com/paca/api/internal/transport/http/handler"
	"github.com/paca/api/internal/transport/http/router"
)

// ---------------------------------------------------------------------------
// Fake GitHub service
// ---------------------------------------------------------------------------

type fakeGitHubService struct {
	mu           sync.RWMutex
	integrations map[uuid.UUID]*githubdom.Integration
	repos        map[uuid.UUID]*githubdom.LinkedRepository // keyed by repo ID
	prs          map[uuid.UUID]*githubdom.PullRequest
	links        map[string]bool // "taskID:prID"
	returnErr    error           // when non-nil, every call returns this error
}

func newFakeGitHubService() *fakeGitHubService {
	return &fakeGitHubService{
		integrations: make(map[uuid.UUID]*githubdom.Integration),
		repos:        make(map[uuid.UUID]*githubdom.LinkedRepository),
		prs:          make(map[uuid.UUID]*githubdom.PullRequest),
		links:        make(map[string]bool),
	}
}

func (s *fakeGitHubService) SetToken(_ context.Context, projectID uuid.UUID, _ string) (*githubdom.Integration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	now := time.Now()
	intg := &githubdom.Integration{ID: uuid.New(), ProjectID: projectID, CreatedAt: now, UpdatedAt: now}
	s.integrations[projectID] = intg
	return intg, nil
}

func (s *fakeGitHubService) GetIntegration(_ context.Context, projectID uuid.UUID) (*githubdom.Integration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	intg, ok := s.integrations[projectID]
	if !ok {
		return nil, githubdom.ErrIntegrationNotFound
	}
	return intg, nil
}

func (s *fakeGitHubService) DeleteIntegration(_ context.Context, projectID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return s.returnErr
	}
	delete(s.integrations, projectID)
	// Remove all repos belonging to this project.
	for repoID, r := range s.repos {
		if r.ProjectID == projectID {
			delete(s.repos, repoID)
		}
	}
	return nil
}

func (s *fakeGitHubService) ListRepositories(_ context.Context, _ uuid.UUID) ([]githubdom.RepoInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	return []githubdom.RepoInfo{
		{FullName: "owner/repo", Owner: "owner", Name: "repo", DefaultBranch: "main"},
	}, nil
}

func (s *fakeGitHubService) LinkRepository(_ context.Context, projectID uuid.UUID, owner, repoName string) (*githubdom.LinkedRepository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	now := time.Now()
	repo := &githubdom.LinkedRepository{
		ID:            uuid.New(),
		ProjectID:     projectID,
		IntegrationID: uuid.New(),
		Owner:         owner,
		RepoName:      repoName,
		FullName:      owner + "/" + repoName,
		DefaultBranch: "main",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.repos[repo.ID] = repo
	return repo, nil
}

func (s *fakeGitHubService) ListLinkedRepositories(_ context.Context, projectID uuid.UUID) ([]*githubdom.LinkedRepository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	var out []*githubdom.LinkedRepository
	for _, r := range s.repos {
		if r.ProjectID == projectID {
			out = append(out, r)
		}
	}
	if out == nil {
		out = []*githubdom.LinkedRepository{}
	}
	return out, nil
}

func (s *fakeGitHubService) UnlinkRepository(_ context.Context, _ uuid.UUID, repoID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return s.returnErr
	}
	if _, ok := s.repos[repoID]; !ok {
		return githubdom.ErrRepositoryNotFound
	}
	delete(s.repos, repoID)
	return nil
}

func (s *fakeGitHubService) ListTaskPRs(_ context.Context, taskID uuid.UUID) ([]*githubdom.PullRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	var out []*githubdom.PullRequest
	for k := range s.links {
		tid := k[:36]
		if tid == taskID.String() {
			prID, _ := uuid.Parse(k[37:])
			if p, ok := s.prs[prID]; ok {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

func (s *fakeGitHubService) LinkPRToTask(_ context.Context, projectID, taskID, _ uuid.UUID, prNumber int) (*githubdom.PullRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return nil, s.returnErr
	}
	now := time.Now()
	pr := &githubdom.PullRequest{
		ID:        uuid.New(),
		ProjectID: projectID,
		PRNumber:  prNumber,
		Title:     "test PR",
		State:     "open",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.prs[pr.ID] = pr
	key := taskID.String() + ":" + pr.ID.String()
	if s.links[key] {
		return nil, githubdom.ErrPRAlreadyLinked
	}
	s.links[key] = true
	return pr, nil
}

func (s *fakeGitHubService) UnlinkPRFromTask(_ context.Context, taskID, prID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return s.returnErr
	}
	key := taskID.String() + ":" + prID.String()
	if !s.links[key] {
		return githubdom.ErrPRLinkNotFound
	}
	delete(s.links, key)
	return nil
}

func (s *fakeGitHubService) CreateBranch(_ context.Context, _, _ uuid.UUID, branchName, _ string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.returnErr != nil {
		return "", s.returnErr
	}
	return branchName, nil
}

func (s *fakeGitHubService) HandleWebhookEvent(_ context.Context, _, _, _ string, _ []byte) error {
	return s.returnErr
}

// ---------------------------------------------------------------------------
// Router builder
// ---------------------------------------------------------------------------

func buildGitHubTestRouter(ghSvc githubdom.Service, permStore *projectPermStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	tm := jwttoken.New(testSecret, 15*time.Minute, 168*time.Hour)
	refreshStore := &fakeRefreshStore{}
	userRepo := newFakeUserRepo()
	authService := authsvc.New(userRepo, tm, refreshStore, 168*time.Hour, 24*time.Hour)
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	return router.New(router.Deps{
		TokenManager: tm,
		Authorizer:   authz.NewAuthorizer(permStore),
		Health:       handler.NewHealthHandler(),
		Auth:         handler.NewAuthHandler(authService, testCookieCfg),
		User:         handler.NewUserHandler(nil),
		GitHub:       handler.NewGitHubHandler(ghSvc),
		Log:          log,
	})
}

// ghWriteStore returns a permission store granting global projects.write + tasks.write.
func ghWriteStore() *projectPermStore {
	return &projectPermStore{
		globalPerms: []authz.Permission{
			authz.PermissionProjectsRead,
			authz.PermissionProjectsWrite,
			authz.PermissionTasksRead,
			authz.PermissionTasksWrite,
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGitHub_SetToken_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())

	projectID := uuid.New()
	body, _ := json.Marshal(map[string]string{"token": "ghp_fake"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/github/token",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitHub_SetToken_InvalidToken(t *testing.T) {
	svc := newFakeGitHubService()
	svc.returnErr = githubdom.ErrInvalidToken
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())

	projectID := uuid.New()
	body, _ := json.Marshal(map[string]string{"token": "ghp_bad"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/github/token",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if code := decodeErrorCode(t, w); code != "GITHUB_INVALID_TOKEN" {
		t.Errorf("expected GITHUB_INVALID_TOKEN, got %q", code)
	}
}

func TestGitHub_GetIntegration_NotFound(t *testing.T) {
	svc := newFakeGitHubService() // no integrations seeded
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())

	projectID := uuid.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if code := decodeErrorCode(t, w); code != "GITHUB_INTEGRATION_NOT_FOUND" {
		t.Errorf("expected GITHUB_INTEGRATION_NOT_FOUND, got %q", code)
	}
}

func TestGitHub_SetAndGetIntegration(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())

	projectID := uuid.New()

	// Set token first.
	body, _ := json.Marshal(map[string]string{"token": "ghp_real"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut,
		"/api/v1/projects/"+projectID.String()+"/github/token",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("set token: expected 200, got %d", w.Code)
	}

	// Get integration.
	req2 := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get integration: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var env map[string]any
	if err := json.NewDecoder(w2.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, _ := env["data"].(map[string]any)
	if data["project_id"] != projectID.String() {
		t.Errorf("expected project_id %q, got %v", projectID, data["project_id"])
	}
}

func TestGitHub_DeleteToken_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()

	// Seed integration.
	svc.integrations[projectID] = &githubdom.Integration{ID: uuid.New(), ProjectID: projectID}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete,
		"/api/v1/projects/"+projectID.String()+"/github/token", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitHub_ListRepositories_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	svc.integrations[projectID] = &githubdom.Integration{ID: uuid.New(), ProjectID: projectID}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github/repositories", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env map[string]any
	json.NewDecoder(w.Body).Decode(&env) //nolint:errcheck
	items, _ := env["data"].([]any)
	if len(items) == 0 {
		t.Error("expected non-empty repository list")
	}
}

func TestGitHub_LinkAndGetRepository(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	svc.integrations[projectID] = &githubdom.Integration{ID: uuid.New(), ProjectID: projectID}

	// Link.
	body, _ := json.Marshal(map[string]string{"owner": "myorg", "repo_name": "myrepo"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/github/linked-repositories",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("link repo: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List linked repositories.
	req2 := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github/linked-repositories", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list repos: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var env map[string]any
	json.NewDecoder(w2.Body).Decode(&env) //nolint:errcheck
	items, _ := env["data"].([]any)
	if len(items) == 0 {
		t.Error("expected at least one linked repository")
	}
	data, _ := items[0].(map[string]any)
	if data["full_name"] != "myorg/myrepo" {
		t.Errorf("expected full_name myorg/myrepo, got %v", data["full_name"])
	}
}

func TestGitHub_ListLinkedRepositories_Empty(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github/linked-repositories", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var env map[string]any
	json.NewDecoder(w.Body).Decode(&env) //nolint:errcheck
	items, _ := env["data"].([]any)
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestGitHub_UnlinkRepository_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	repoID := uuid.New()
	svc.repos[repoID] = &githubdom.LinkedRepository{ID: repoID, ProjectID: projectID, FullName: "o/r"}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete,
		"/api/v1/projects/"+projectID.String()+"/github/linked-repositories/"+repoID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitHub_LinkPRToTask_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	taskID := uuid.New()

	body, _ := json.Marshal(map[string]any{"repo_id": uuid.New().String(), "pr_number": 7})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/github/pull-requests",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitHub_ListTaskPRs_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	taskID := uuid.New()

	// Pre-link a PR.
	prID := uuid.New()
	pr := &githubdom.PullRequest{ID: prID, ProjectID: projectID, PRNumber: 42, State: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc.prs[pr.ID] = pr
	svc.links[taskID.String()+":"+prID.String()] = true

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/github/pull-requests",
		nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env map[string]any
	json.NewDecoder(w.Body).Decode(&env) //nolint:errcheck
	items, _ := env["data"].([]any)
	if len(items) != 1 {
		t.Errorf("expected 1 PR, got %d", len(items))
	}
}

func TestGitHub_UnlinkPR_NotFound(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	taskID := uuid.New()
	prID := uuid.New()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete,
		"/api/v1/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/github/pull-requests/"+prID.String(),
		nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if code := decodeErrorCode(t, w); code != "GITHUB_PR_LINK_NOT_FOUND" {
		t.Errorf("expected GITHUB_PR_LINK_NOT_FOUND, got %q", code)
	}
}

func TestGitHub_CreateBranch_OK(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())
	tok := issueProjectToken(t, uuid.NewString())
	projectID := uuid.New()
	taskID := uuid.New()

	body, _ := json.Marshal(map[string]string{"repo_id": uuid.New().String(), "branch_name": "feature/test-branch"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/github/branches",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var env map[string]any
	json.NewDecoder(w.Body).Decode(&env) //nolint:errcheck
	data, _ := env["data"].(map[string]any)
	if data["branch_name"] != "feature/test-branch" {
		t.Errorf("expected branch_name feature/test-branch, got %v", data["branch_name"])
	}
}

func TestGitHub_Webhook_NoContent(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())

	payload := `{"repository":{"full_name":"owner/repo"},"action":"opened"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/v1/github/webhook",
		bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalidsignature")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Webhook always responds 204 regardless of verification outcome.
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitHub_Webhook_EmptyPayload(t *testing.T) {
	svc := newFakeGitHubService()
	r := buildGitHubTestRouter(svc, ghWriteStore())

	// Payload with no repository field — should still respond 204.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/v1/github/webhook",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "ping")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
