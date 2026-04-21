package projectdom

import (
	"context"

	"github.com/google/uuid"
)

// MemberRepository defines persistence operations for project members.
type MemberRepository interface {
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]*ProjectMember, error)
	FindMember(ctx context.Context, projectID, userID uuid.UUID) (*ProjectMember, error)
	// FindMemberByUserProject returns the active member record for a user in a
	// project.  Used by background workers to resolve a user UUID to a member UUID.
	FindMemberByUserProject(ctx context.Context, userID, projectID uuid.UUID) (*ProjectMember, error)
	// FindMemberByID returns the active member record for the given
	// project_members.id.  Used to resolve an assignee member ID to a user ID.
	FindMemberByID(ctx context.Context, memberID uuid.UUID) (*ProjectMember, error)
	AddMember(ctx context.Context, m *ProjectMember) error
	UpdateMemberRole(ctx context.Context, projectID, userID, roleID uuid.UUID) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
}
