package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	"github.com/paca/api/internal/transport/http/handler"
)

// ---------------------------------------------------------------------------
// Minimal fakes
// ---------------------------------------------------------------------------

type fakeSprintSvcH struct {
	created []*sprintdom.Sprint
}

func (f *fakeSprintSvcH) ListSprints(_ context.Context, _ uuid.UUID) ([]*sprintdom.Sprint, error) {
	return nil, nil
}
func (f *fakeSprintSvcH) GetSprint(_ context.Context, _ uuid.UUID) (*sprintdom.Sprint, error) {
	return nil, sprintdom.ErrSprintNotFound
}
func (f *fakeSprintSvcH) CreateSprint(_ context.Context, in sprintdom.CreateSprintInput) (*sprintdom.Sprint, error) {
	now := time.Now()
	sp := &sprintdom.Sprint{
		ID:        uuid.New(),
		ProjectID: in.ProjectID,
		Name:      in.Name,
		Status:    sprintdom.SprintStatusPlanned,
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.created = append(f.created, sp)
	return sp, nil
}
func (f *fakeSprintSvcH) UpdateSprint(_ context.Context, _ uuid.UUID, _ sprintdom.UpdateSprintInput) (*sprintdom.Sprint, error) {
	return nil, sprintdom.ErrSprintNotFound
}
func (f *fakeSprintSvcH) DeleteSprint(_ context.Context, _ uuid.UUID) error {
	return nil
}

type fakeViewSvcH struct {
	mu      sync.Mutex
	created []*sprintdom.SprintView
}

func (f *fakeViewSvcH) ListViews(_ context.Context, _ uuid.UUID) ([]*sprintdom.SprintView, error) {
	return nil, nil
}
func (f *fakeViewSvcH) ListBacklogViews(_ context.Context, _ uuid.UUID) ([]*sprintdom.SprintView, error) {
	return nil, nil
}
func (f *fakeViewSvcH) GetView(_ context.Context, _ uuid.UUID) (*sprintdom.SprintView, error) {
	return nil, sprintdom.ErrViewNotFound
}
func (f *fakeViewSvcH) CreateView(_ context.Context, in sprintdom.CreateViewInput) (*sprintdom.SprintView, error) {
	now := time.Now()
	v := &sprintdom.SprintView{
		ID:        uuid.New(),
		SprintID:  in.SprintID,
		ProjectID: in.ProjectID,
		Name:      in.Name,
		ViewType:  in.ViewType,
		Position:  in.Position,
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.mu.Lock()
	f.created = append(f.created, v)
	f.mu.Unlock()
	return v, nil
}
func (f *fakeViewSvcH) UpdateView(_ context.Context, _ uuid.UUID, _ sprintdom.UpdateViewInput) (*sprintdom.SprintView, error) {
	return nil, sprintdom.ErrViewNotFound
}
func (f *fakeViewSvcH) DeleteView(_ context.Context, _ uuid.UUID) error { return nil }
func (f *fakeViewSvcH) MoveTask(_ context.Context, _ uuid.UUID, _ sprintdom.MoveTaskInput) error {
	return nil
}
func (f *fakeViewSvcH) ListTaskPositions(_ context.Context, _ uuid.UUID) ([]*sprintdom.ViewTaskPosition, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateSprint_SeedsDefaultViews(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sprintSvc := &fakeSprintSvcH{}
	viewSvc := &fakeViewSvcH{}

	h := handler.NewSprintHandler(sprintSvc, viewSvc)

	r := gin.New()
	r.POST("/projects/:projectId/sprints", h.CreateSprint)

	body, _ := json.Marshal(map[string]any{"name": "Sprint 1"})
	projectID := uuid.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/projects/"+projectID.String()+"/sprints", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if len(viewSvc.created) != 2 {
		t.Fatalf("expected 2 default views, got %d", len(viewSvc.created))
	}

	viewTypes := map[sprintdom.ViewType]bool{}
	for _, v := range viewSvc.created {
		viewTypes[v.ViewType] = true
	}
	if !viewTypes[sprintdom.ViewTypeBoard] {
		t.Error("expected a board view to be created")
	}
	if !viewTypes[sprintdom.ViewTypeTable] {
		t.Error("expected a table view to be created")
	}

	// Both views must belong to the newly created sprint.
	sprintID := sprintSvc.created[0].ID
	for _, v := range viewSvc.created {
		if v.SprintID == nil || *v.SprintID != sprintID {
			t.Errorf("view %s has wrong sprint ID: want %s, got %v", v.ID, sprintID, v.SprintID)
		}
	}
}
