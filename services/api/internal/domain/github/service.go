package githubdom

import (
	"context"

	"github.com/google/uuid"
)

// Service defines the GitHub integration service contract.
type Service interface {
	// SetToken validates the supplied personal access token against the GitHub
	// API and, if accepted, stores it encrypted for the given project.
	// Calling SetToken again replaces the existing token.
	SetToken(ctx context.Context, projectID uuid.UUID, plainToken string) (*Integration, error)

	// GetIntegration returns the GitHub integration for a project.
	// Returns ErrIntegrationNotFound when the project has no integration.
	GetIntegration(ctx context.Context, projectID uuid.UUID) (*Integration, error)

	// DeleteIntegration removes the GitHub integration (PAT) for a project and
	// unlinks any associated repository (including deleting its webhook).
	DeleteIntegration(ctx context.Context, projectID uuid.UUID) error

	// ListRepositories returns all GitHub repositories accessible with the
	// project's stored PAT.
	// Returns ErrIntegrationNotFound when no PAT has been configured.
	ListRepositories(ctx context.Context, projectID uuid.UUID) ([]RepoInfo, error)

	// LinkRepository links a specific GitHub repository to a project.
	// A webhook is automatically created on the repository so that push,
	// pull_request, and check_run events are delivered to this service.
	// A project may have multiple linked repositories.
	// Returns ErrIntegrationNotFound when no PAT has been configured.
	// Returns ErrWebhookURLRequired when the service has no public URL set.
	LinkRepository(ctx context.Context, projectID uuid.UUID, owner, repoName string) (*LinkedRepository, error)

	// ListLinkedRepositories returns all repositories linked to a project.
	// Returns an empty slice when none are linked.
	ListLinkedRepositories(ctx context.Context, projectID uuid.UUID) ([]*LinkedRepository, error)

	// UnlinkRepository removes a specific repository link and deletes its webhook.
	// Returns ErrRepositoryNotFound when the repository is not linked.
	UnlinkRepository(ctx context.Context, projectID, repoID uuid.UUID) error

	// ListTaskPRs returns all pull requests linked to a task.
	ListTaskPRs(ctx context.Context, taskID uuid.UUID) ([]*PullRequest, error)

	// LinkPRToTask fetches the pull request with the given number from GitHub,
	// caches it locally, and creates a task-PR link.
	// repoID identifies which linked repository to use.
	// Returns ErrIntegrationNotFound / ErrRepositoryNotFound when the project
	// is not fully configured.
	// Returns ErrPRAlreadyLinked when the link already exists.
	LinkPRToTask(ctx context.Context, projectID, taskID, repoID uuid.UUID, prNumber int) (*PullRequest, error)

	// UnlinkPRFromTask removes the link between a task and a pull request.
	// Returns ErrPRLinkNotFound when the link does not exist.
	UnlinkPRFromTask(ctx context.Context, taskID, prID uuid.UUID) error

	// CreateBranch creates a new git branch in the linked repository identified by repoID.
	// branchName is the desired branch name; it must be non-empty.
	// sourceBranch is the branch to branch off; if empty, the repository's
	// default branch is used.
	// Returns the full branch name on success.
	CreateBranch(ctx context.Context, projectID, repoID uuid.UUID, branchName, sourceBranch string) (string, error)

	// HandleWebhookEvent processes an incoming GitHub webhook event.
	// repoFullName is the "owner/repo" value from the event payload.
	// event is the X-GitHub-Event header value.
	// signature is the X-Hub-Signature-256 header value (used for HMAC verification).
	// payload is the raw request body.
	HandleWebhookEvent(ctx context.Context, repoFullName, event, signature string, payload []byte) error
}
