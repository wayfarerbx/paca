// Package githubdom defines the GitHub integration aggregate and its domain
// contracts.  Each project may have one GitHub integration (a stored encrypted
// PAT) and one linked repository.  Pull requests are cached and can be linked
// to individual tasks.
package githubdom

import (
	"time"

	"github.com/google/uuid"
)

// Integration stores a project's GitHub personal access token (encrypted at
// rest).  Each project has at most one integration.
type Integration struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	AccessTokenEnc string // AES-256-GCM encrypted PAT (base64-encoded)
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// LinkedRepository represents a GitHub repository that is linked to a project.
// Exactly one webhook is registered on the repository when it is linked so
// that push, pull_request, and check_run events are delivered to this service.
type LinkedRepository struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	IntegrationID    uuid.UUID
	Owner            string // GitHub user/org login
	RepoName         string // bare repo name
	FullName         string // "owner/repo"
	WebhookID        int64  // GitHub-assigned webhook ID; 0 until the hook is created
	WebhookSecretEnc string // encrypted HMAC secret used to validate webhook payloads
	DefaultBranch    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PullRequest is a locally cached copy of a GitHub pull request.
// It is created/updated on manual linking and on incoming webhook events.
type PullRequest struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	RepoID     uuid.UUID
	PRNumber   int
	GitHubPRID int64
	Title      string
	State      string // "open" | "closed" | "merged"
	HTMLURL    string
	HeadBranch string
	BaseBranch string
	Author     string
	MergedAt   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TaskPRLink is the join between a task and a cached pull request.
type TaskPRLink struct {
	ID            uuid.UUID
	TaskID        uuid.UUID
	PullRequestID uuid.UUID
	CreatedAt     time.Time
}

// RepoInfo is a slimmed-down view of a GitHub repository used when listing
// repositories available under a given PAT.
type RepoInfo struct {
	FullName      string
	Owner         string
	Name          string
	DefaultBranch string
	Private       bool
}
