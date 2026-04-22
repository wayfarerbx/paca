package githubdom

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the combined persistence contract for the GitHub integration
// aggregate.  A single concrete type satisfies all sub-interfaces.
type Repository interface {
	IntegrationRepository
	LinkedRepositoryRepository
	PRRepository
	TaskPRLinkRepository
}

// IntegrationRepository manages per-project GitHub PAT storage.
type IntegrationRepository interface {
	// FindIntegrationByProject returns the integration for a project, or
	// ErrIntegrationNotFound when none exists.
	FindIntegrationByProject(ctx context.Context, projectID uuid.UUID) (*Integration, error)

	// UpsertIntegration creates or replaces the integration for a project.
	UpsertIntegration(ctx context.Context, integration *Integration) error

	// DeleteIntegration removes the integration for a project, cascading to
	// the linked repository and any cached PRs.
	DeleteIntegration(ctx context.Context, projectID uuid.UUID) error
}

// LinkedRepositoryRepository manages the GitHub repositories linked to a project.
// A project may have multiple linked repositories.
type LinkedRepositoryRepository interface {
	// ListRepositoriesByProject returns all linked repositories for a project.
	// Returns an empty slice when none are linked.
	ListRepositoriesByProject(ctx context.Context, projectID uuid.UUID) ([]*LinkedRepository, error)

	// FindRepositoryByID returns a specific linked repository by its ID.
	// Returns ErrRepositoryNotFound when not found.
	FindRepositoryByID(ctx context.Context, repoID uuid.UUID) (*LinkedRepository, error)

	// FindRepositoryByFullName looks up a repository by its "owner/repo" name.
	// Used during webhook dispatch to route events to the correct project.
	FindRepositoryByFullName(ctx context.Context, fullName string) (*LinkedRepository, error)

	// InsertRepository adds a new linked repository for a project.
	// Returns an error when the same repository (by full_name) is already linked to the project.
	InsertRepository(ctx context.Context, repo *LinkedRepository) error

	// DeleteRepositoryByID removes a specific linked repository by its ID.
	DeleteRepositoryByID(ctx context.Context, repoID uuid.UUID) error
}

// PRRepository manages cached GitHub pull requests.
type PRRepository interface {
	// FindPRByRepoAndNumber looks up a cached PR by its repo and PR number.
	// Returns ErrPRNotFound when the PR is not in the local cache.
	FindPRByRepoAndNumber(ctx context.Context, repoID uuid.UUID, prNumber int) (*PullRequest, error)

	// ListPRsForTask returns all pull requests linked to the given task.
	ListPRsForTask(ctx context.Context, taskID uuid.UUID) ([]*PullRequest, error)

	// UpsertPR creates or updates a cached pull request.
	UpsertPR(ctx context.Context, pr *PullRequest) error
}

// TaskPRLinkRepository manages task ↔ pull-request associations.
type TaskPRLinkRepository interface {
	// LinkPRToTask creates a task-PR association.
	// Returns ErrPRAlreadyLinked when the link already exists.
	LinkPRToTask(ctx context.Context, link *TaskPRLink) error

	// UnlinkPRFromTask removes a task-PR association.
	// Returns ErrPRLinkNotFound when the link does not exist.
	UnlinkPRFromTask(ctx context.Context, taskID, prID uuid.UUID) error
}
