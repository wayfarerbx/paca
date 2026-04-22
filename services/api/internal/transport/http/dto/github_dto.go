package dto

import (
	"time"

	"github.com/google/uuid"
	githubdom "github.com/paca/api/internal/domain/github"
)

// --- GitHub Integration DTOs -----------------------------------------------

// SetGitHubTokenRequest is the body for PUT /projects/:projectId/github/token.
type SetGitHubTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// GitHubIntegrationResponse is the public representation of a GitHub integration.
// The actual PAT is never returned.
type GitHubIntegrationResponse struct {
	ProjectID uuid.UUID `json:"project_id"`
	Connected bool      `json:"connected"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GitHubIntegrationFromEntity maps a domain Integration to a response DTO.
func GitHubIntegrationFromEntity(i *githubdom.Integration) GitHubIntegrationResponse {
	return GitHubIntegrationResponse{
		ProjectID: i.ProjectID,
		Connected: true,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
	}
}

// --- GitHub Repository DTOs ------------------------------------------------

// LinkGitHubRepositoryRequest is the body for PUT /projects/:projectId/github/repository.
type LinkGitHubRepositoryRequest struct {
	Owner    string `json:"owner" binding:"required"`
	RepoName string `json:"repo_name" binding:"required"`
}

// GitHubRepositoryResponse is the public representation of a linked repository.
type GitHubRepositoryResponse struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	Owner         string    `json:"owner"`
	RepoName      string    `json:"repo_name"`
	FullName      string    `json:"full_name"`
	DefaultBranch string    `json:"default_branch"`
	WebhookActive bool      `json:"webhook_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GitHubRepositoryFromEntity maps a domain LinkedRepository to a response DTO.
func GitHubRepositoryFromEntity(r *githubdom.LinkedRepository) GitHubRepositoryResponse {
	return GitHubRepositoryResponse{
		ID:            r.ID,
		ProjectID:     r.ProjectID,
		Owner:         r.Owner,
		RepoName:      r.RepoName,
		FullName:      r.FullName,
		DefaultBranch: r.DefaultBranch,
		WebhookActive: r.WebhookID != 0,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

// RepoInfoResponse is returned in the list-repositories endpoint.
type RepoInfoResponse struct {
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	RepoName      string `json:"repo_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

// RepoInfoFromDomain maps a domain RepoInfo to a response DTO.
func RepoInfoFromDomain(r githubdom.RepoInfo) RepoInfoResponse {
	return RepoInfoResponse{
		FullName:      r.FullName,
		Owner:         r.Owner,
		RepoName:      r.Name,
		DefaultBranch: r.DefaultBranch,
		Private:       r.Private,
	}
}

// --- GitHub Pull Request DTOs ----------------------------------------------

// LinkPRRequest is the body for POST /projects/:projectId/tasks/:taskId/github/pull-requests.
type LinkPRRequest struct {
	RepoID   uuid.UUID `json:"repo_id" binding:"required"`
	PRNumber int       `json:"pr_number" binding:"required,min=1"`
}

// PullRequestResponse is the public representation of a cached pull request.
type PullRequestResponse struct {
	ID         uuid.UUID  `json:"id"`
	ProjectID  uuid.UUID  `json:"project_id"`
	PRNumber   int        `json:"pr_number"`
	Title      string     `json:"title"`
	State      string     `json:"state"`
	HTMLURL    string     `json:"html_url"`
	HeadBranch string     `json:"head_branch"`
	BaseBranch string     `json:"base_branch"`
	Author     string     `json:"author"`
	MergedAt   *time.Time `json:"merged_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PullRequestFromEntity maps a domain PullRequest to a response DTO.
func PullRequestFromEntity(pr *githubdom.PullRequest) PullRequestResponse {
	return PullRequestResponse{
		ID:         pr.ID,
		ProjectID:  pr.ProjectID,
		PRNumber:   pr.PRNumber,
		Title:      pr.Title,
		State:      pr.State,
		HTMLURL:    pr.HTMLURL,
		HeadBranch: pr.HeadBranch,
		BaseBranch: pr.BaseBranch,
		Author:     pr.Author,
		MergedAt:   pr.MergedAt,
		CreatedAt:  pr.CreatedAt,
		UpdatedAt:  pr.UpdatedAt,
	}
}

// --- GitHub Branch DTOs ----------------------------------------------------

// CreateBranchRequest is the body for POST /projects/:projectId/tasks/:taskId/github/branches.
type CreateBranchRequest struct {
	RepoID       uuid.UUID `json:"repo_id" binding:"required"`
	BranchName   string    `json:"branch_name" binding:"required"`
	SourceBranch string    `json:"source_branch"` // optional; defaults to repo default branch
}

// CreateBranchResponse is returned after a branch is created.
type CreateBranchResponse struct {
	BranchName string `json:"branch_name"`
}
