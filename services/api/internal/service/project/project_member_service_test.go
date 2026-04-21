package projectsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	projectdom "github.com/paca/api/internal/domain/project"
)

type memberServiceRepoMock struct {
	findByID         func(ctx context.Context, id uuid.UUID) (*projectdom.Project, error)
	findMember       func(ctx context.Context, projectID, userID uuid.UUID) (*projectdom.ProjectMember, error)
	findRoleByID     func(ctx context.Context, id uuid.UUID) (*projectdom.ProjectRole, error)
	updateMemberRole func(ctx context.Context, projectID, userID, roleID uuid.UUID) error
}

func (m *memberServiceRepoMock) List(context.Context, int, int) ([]*projectdom.Project, int64, error) {
	return nil, 0, nil
}

func (m *memberServiceRepoMock) ListAccessible(_ context.Context, _ uuid.UUID, _, _ int) ([]*projectdom.Project, int64, error) {
	return nil, 0, nil
}

func (m *memberServiceRepoMock) FindByID(ctx context.Context, id uuid.UUID) (*projectdom.Project, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, projectdom.ErrNotFound
}

func (m *memberServiceRepoMock) Create(context.Context, *projectdom.Project) error {
	return nil
}

func (m *memberServiceRepoMock) Update(context.Context, *projectdom.Project) error {
	return nil
}

func (m *memberServiceRepoMock) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (m *memberServiceRepoMock) ListMembers(context.Context, uuid.UUID) ([]*projectdom.ProjectMember, error) {
	return nil, nil
}

func (m *memberServiceRepoMock) FindMember(ctx context.Context, projectID, userID uuid.UUID) (*projectdom.ProjectMember, error) {
	if m.findMember != nil {
		return m.findMember(ctx, projectID, userID)
	}
	return nil, projectdom.ErrMemberNotFound
}

func (m *memberServiceRepoMock) FindMemberByUserProject(_ context.Context, _, _ uuid.UUID) (*projectdom.ProjectMember, error) {
	return nil, projectdom.ErrMemberNotFound
}

func (m *memberServiceRepoMock) AddMember(context.Context, *projectdom.ProjectMember) error {
	return nil
}

func (m *memberServiceRepoMock) UpdateMemberRole(ctx context.Context, projectID, userID, roleID uuid.UUID) error {
	if m.updateMemberRole != nil {
		return m.updateMemberRole(ctx, projectID, userID, roleID)
	}
	return nil
}

func (m *memberServiceRepoMock) RemoveMember(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (m *memberServiceRepoMock) ListRoles(context.Context, uuid.UUID) ([]*projectdom.ProjectRole, error) {
	return nil, nil
}

func (m *memberServiceRepoMock) FindRoleByID(ctx context.Context, id uuid.UUID) (*projectdom.ProjectRole, error) {
	if m.findRoleByID != nil {
		return m.findRoleByID(ctx, id)
	}
	return nil, projectdom.ErrRoleNotFound
}

func (m *memberServiceRepoMock) FindRoleByName(context.Context, uuid.UUID, string) (*projectdom.ProjectRole, error) {
	return nil, projectdom.ErrRoleNotFound
}

func (m *memberServiceRepoMock) CreateRole(context.Context, *projectdom.ProjectRole) error {
	return nil
}

func (m *memberServiceRepoMock) UpdateRole(context.Context, *projectdom.ProjectRole) error {
	return nil
}

func (m *memberServiceRepoMock) DeleteRole(context.Context, uuid.UUID) error {
	return nil
}

func (m *memberServiceRepoMock) CountMembersWithRole(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *memberServiceRepoMock) FindMemberByID(_ context.Context, _ uuid.UUID) (*projectdom.ProjectMember, error) {
	return nil, projectdom.ErrMemberNotFound
}

var _ projectdom.Repository = (*memberServiceRepoMock)(nil)

func TestUpdateMemberRole_Success(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	oldRoleID := uuid.New()
	newRoleID := uuid.New()
	findMemberCalls := 0

	repo := &memberServiceRepoMock{
		findByID: func(_ context.Context, id uuid.UUID) (*projectdom.Project, error) {
			return &projectdom.Project{ID: id}, nil
		},
		findMember: func(_ context.Context, pid, uid uuid.UUID) (*projectdom.ProjectMember, error) {
			findMemberCalls++
			roleID := oldRoleID
			if findMemberCalls > 1 {
				roleID = newRoleID
			}
			return &projectdom.ProjectMember{
				ID:            uuid.New(),
				ProjectID:     pid,
				UserID:        uid,
				ProjectRoleID: roleID,
			}, nil
		},
		findRoleByID: func(_ context.Context, id uuid.UUID) (*projectdom.ProjectRole, error) {
			return &projectdom.ProjectRole{ID: id, ProjectID: &projectID}, nil
		},
		updateMemberRole: func(_ context.Context, pid, uid, rid uuid.UUID) error {
			if pid != projectID || uid != userID || rid != newRoleID {
				t.Fatalf("unexpected ids in update call")
			}
			return nil
		},
	}

	svc := New(repo, nil)
	member, err := svc.UpdateMemberRole(context.Background(), projectID, userID, projectdom.UpdateMemberRoleInput{
		ProjectRoleID: newRoleID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if member.ProjectRoleID != newRoleID {
		t.Fatalf("expected role id %s, got %s", newRoleID, member.ProjectRoleID)
	}
	if findMemberCalls != 2 {
		t.Fatalf("expected find member to be called twice, got %d", findMemberCalls)
	}
}

func TestUpdateMemberRole_ProjectNotFound(t *testing.T) {
	repo := &memberServiceRepoMock{
		findByID: func(context.Context, uuid.UUID) (*projectdom.Project, error) {
			return nil, projectdom.ErrNotFound
		},
	}

	svc := New(repo, nil)
	_, err := svc.UpdateMemberRole(context.Background(), uuid.New(), uuid.New(), projectdom.UpdateMemberRoleInput{
		ProjectRoleID: uuid.New(),
	})
	if !errors.Is(err, projectdom.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateMemberRole_RoleFromDifferentProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	repo := &memberServiceRepoMock{
		findByID: func(_ context.Context, id uuid.UUID) (*projectdom.Project, error) {
			return &projectdom.Project{ID: id}, nil
		},
		findMember: func(_ context.Context, pid, uid uuid.UUID) (*projectdom.ProjectMember, error) {
			return &projectdom.ProjectMember{ProjectID: pid, UserID: uid}, nil
		},
		findRoleByID: func(_ context.Context, id uuid.UUID) (*projectdom.ProjectRole, error) {
			return &projectdom.ProjectRole{ID: id, ProjectID: &otherProjectID}, nil
		},
		updateMemberRole: func(_ context.Context, _, _, _ uuid.UUID) error {
			t.Fatal("update should not be called for role from different project")
			return nil
		},
	}

	svc := New(repo, nil)
	_, err := svc.UpdateMemberRole(context.Background(), projectID, uuid.New(), projectdom.UpdateMemberRoleInput{
		ProjectRoleID: uuid.New(),
	})
	if !errors.Is(err, projectdom.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestUpdateMemberRole_UpdateError(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	repo := &memberServiceRepoMock{
		findByID: func(_ context.Context, id uuid.UUID) (*projectdom.Project, error) {
			return &projectdom.Project{ID: id}, nil
		},
		findMember: func(_ context.Context, pid, uid uuid.UUID) (*projectdom.ProjectMember, error) {
			return &projectdom.ProjectMember{ProjectID: pid, UserID: uid}, nil
		},
		findRoleByID: func(_ context.Context, id uuid.UUID) (*projectdom.ProjectRole, error) {
			return &projectdom.ProjectRole{ID: id, ProjectID: &projectID}, nil
		},
		updateMemberRole: func(_ context.Context, _, _, _ uuid.UUID) error {
			return projectdom.ErrMemberNotFound
		},
	}

	svc := New(repo, nil)
	_, err := svc.UpdateMemberRole(context.Background(), projectID, userID, projectdom.UpdateMemberRoleInput{
		ProjectRoleID: uuid.New(),
	})
	if !errors.Is(err, projectdom.ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetMyProjectPermissions
// ---------------------------------------------------------------------------

func TestGetMyProjectPermissions_Success(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	wantPerms := map[string]any{"tasks.read": true, "tasks.write": true, "sprints.read": true}

	repo := &memberServiceRepoMock{
		findMember: func(_ context.Context, pid, uid uuid.UUID) (*projectdom.ProjectMember, error) {
			if pid != projectID || uid != userID {
				t.Errorf("unexpected findMember args: pid=%v uid=%v", pid, uid)
			}
			return &projectdom.ProjectMember{
				ID:            uuid.New(),
				ProjectID:     projectID,
				UserID:        userID,
				ProjectRoleID: roleID,
			}, nil
		},
		findRoleByID: func(_ context.Context, id uuid.UUID) (*projectdom.ProjectRole, error) {
			if id != roleID {
				t.Errorf("unexpected findRoleByID arg: %v", id)
			}
			return &projectdom.ProjectRole{
				ID:          roleID,
				ProjectID:   &projectID,
				Permissions: wantPerms,
			}, nil
		},
	}

	svc := New(repo, nil)
	got, err := svc.GetMyProjectPermissions(context.Background(), projectID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(wantPerms) {
		t.Fatalf("expected %d permissions, got %d", len(wantPerms), len(got))
	}
	for k, v := range wantPerms {
		if got[k] != v {
			t.Errorf("permission %q: expected %v, got %v", k, v, got[k])
		}
	}
}

func TestGetMyProjectPermissions_MemberNotFound(t *testing.T) {
	repo := &memberServiceRepoMock{
		findMember: func(_ context.Context, _, _ uuid.UUID) (*projectdom.ProjectMember, error) {
			return nil, projectdom.ErrMemberNotFound
		},
	}

	svc := New(repo, nil)
	_, err := svc.GetMyProjectPermissions(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, projectdom.ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

func TestGetMyProjectPermissions_RoleNotFound(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	repo := &memberServiceRepoMock{
		findMember: func(_ context.Context, _, _ uuid.UUID) (*projectdom.ProjectMember, error) {
			return &projectdom.ProjectMember{
				ID:            uuid.New(),
				ProjectID:     projectID,
				UserID:        userID,
				ProjectRoleID: roleID,
			}, nil
		},
		findRoleByID: func(_ context.Context, _ uuid.UUID) (*projectdom.ProjectRole, error) {
			return nil, projectdom.ErrRoleNotFound
		},
	}

	svc := New(repo, nil)
	_, err := svc.GetMyProjectPermissions(context.Background(), projectID, userID)
	if !errors.Is(err, projectdom.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGetMyProjectPermissions_NilPermissions(t *testing.T) {
	projectID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	repo := &memberServiceRepoMock{
		findMember: func(_ context.Context, _, _ uuid.UUID) (*projectdom.ProjectMember, error) {
			return &projectdom.ProjectMember{
				ID:            uuid.New(),
				ProjectID:     projectID,
				UserID:        userID,
				ProjectRoleID: roleID,
			}, nil
		},
		findRoleByID: func(_ context.Context, _ uuid.UUID) (*projectdom.ProjectRole, error) {
			return &projectdom.ProjectRole{
				ID:          roleID,
				ProjectID:   &projectID,
				Permissions: nil, // nil permissions
			}, nil
		},
	}

	svc := New(repo, nil)
	got, err := svc.GetMyProjectPermissions(context.Background(), projectID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil map, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}
