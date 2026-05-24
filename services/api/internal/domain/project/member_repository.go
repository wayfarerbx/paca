package projectdom

import (
	"context"

	"github.com/google/uuid"
)

// MemberRepository defines persistence operations for project members.
type MemberRepository interface {
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]*ProjectMember, error)
	FindMember(ctx context.Context, projectID, userID uuid.UUID) (*ProjectMember, error)
	// FindMemberByAgent returns the active member record for an agent in a
	// project. Used to resolve an agent UUID to a member UUID.
	FindMemberByAgent(ctx context.Context, projectID, agentID uuid.UUID) (*ProjectMember, error)
	// FindMemberByUserProject returns the active member record for a user in a
	// project.  Used by background workers to resolve a user UUID to a member UUID.
	FindMemberByUserProject(ctx context.Context, userID, projectID uuid.UUID) (*ProjectMember, error)
	// FindMemberByID returns the active member record for the given
	// project_members.id.  Used to resolve an assignee member ID to a user ID.
	FindMemberByID(ctx context.Context, memberID uuid.UUID) (*ProjectMember, error)
	AddMember(ctx context.Context, m *ProjectMember) error
	UpdateMemberRole(ctx context.Context, projectID, userID, roleID uuid.UUID) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	// AddAgentMember inserts an agent as a project member with the given role.
	AddAgentMember(ctx context.Context, memberID, projectID, agentID, roleID uuid.UUID) error
	// RemoveAgentMember soft-deletes the agent's membership record.
	RemoveAgentMember(ctx context.Context, projectID, agentID uuid.UUID) error
}
