package postgres

import (
	"context"
	"fmt"

	"github.com/Paca-AI/api/internal/platform/authz"
	"github.com/google/uuid"
)

// GetAgentProjectRoleName returns the role name for an agent member in a project.
func (s *AuthzPermissionStore) GetAgentProjectRoleName(ctx context.Context, agentID, projectID uuid.UUID) (string, error) {
	var row struct {
		RoleName string
	}
	result := s.db.WithContext(ctx).
		Table("project_roles pr").
		Select("pr.role_name").
		Joins("JOIN project_members pm ON pm.project_role_id = pr.id").
		Where("pm.agent_id = ? AND pm.project_id = ? AND pm.deleted_at IS NULL", agentID.String(), projectID.String()).
		Scan(&row)
	if result.Error != nil {
		return "", fmt.Errorf("authz store: get agent project role name: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("agent not found in project")
	}
	return row.RoleName, nil
}

// ListAgentProjectPermissions returns permissions from project role memberships for
// an agent in the given project.
func (s *AuthzPermissionStore) ListAgentProjectPermissions(ctx context.Context, agentID, projectID uuid.UUID) ([]authz.Permission, error) {
	var rows []struct {
		Permissions []byte
	}
	err := s.db.WithContext(ctx).
		Table("project_roles pr").
		Select("pr.permissions").
		Joins("JOIN project_members pm ON pm.project_role_id = pr.id").
		Where("pm.agent_id = ? AND pm.project_id = ? AND pm.deleted_at IS NULL", agentID.String(), projectID.String()).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("authz store: list agent project permissions: %w", err)
	}

	return collectPermissions(rows), nil
}
