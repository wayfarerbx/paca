package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/apierr"
	githubdom "github.com/paca/api/internal/domain/github"
	"github.com/paca/api/internal/transport/http/dto"
	"github.com/paca/api/internal/transport/http/presenter"
)

// GitHubHandler handles GitHub integration endpoints.
type GitHubHandler struct {
	svc githubdom.Service
}

// NewGitHubHandler creates a GitHubHandler wired to the given service.
func NewGitHubHandler(svc githubdom.Service) *GitHubHandler {
	return &GitHubHandler{svc: svc}
}

// --- Integration endpoints -------------------------------------------------

// GetIntegration handles GET /projects/:projectId/github.
func (h *GitHubHandler) GetIntegration(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	integration, err := h.svc.GetIntegration(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.GitHubIntegrationFromEntity(integration))
}

// SetToken handles PUT /projects/:projectId/github/token.
// Validates and stores (or replaces) the GitHub personal access token.
func (h *GitHubHandler) SetToken(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.SetGitHubTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}
	integration, err := h.svc.SetToken(c.Request.Context(), projectID, req.Token)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.GitHubIntegrationFromEntity(integration))
}

// DeleteToken handles DELETE /projects/:projectId/github/token.
// Removes the GitHub integration and unlinks any repository.
func (h *GitHubHandler) DeleteToken(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteIntegration(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// --- Repository endpoints --------------------------------------------------

// ListLinkedRepositories handles GET /projects/:projectId/github/linked-repositories.
func (h *GitHubHandler) ListLinkedRepositories(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	repos, err := h.svc.ListLinkedRepositories(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	items := make([]dto.GitHubRepositoryResponse, len(repos))
	for i, r := range repos {
		items[i] = dto.GitHubRepositoryFromEntity(r)
	}
	presenter.OK(c, items)
}

// ListRepositories handles GET /projects/:projectId/github/repositories.
// Returns all repositories accessible with the project's PAT.
func (h *GitHubHandler) ListRepositories(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	repos, err := h.svc.ListRepositories(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	items := make([]dto.RepoInfoResponse, len(repos))
	for i, r := range repos {
		items[i] = dto.RepoInfoFromDomain(r)
	}
	presenter.OK(c, items)
}

// LinkRepository handles PUT /projects/:projectId/github/repository.
// Links a repository and automatically creates a webhook on it.
func (h *GitHubHandler) LinkRepository(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.LinkGitHubRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}
	repo, err := h.svc.LinkRepository(c.Request.Context(), projectID, req.Owner, req.RepoName)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.GitHubRepositoryFromEntity(repo))
}

// UnlinkRepository handles DELETE /projects/:projectId/github/linked-repositories/:repoId.
// Removes a specific repository link and deletes its webhook.
func (h *GitHubHandler) UnlinkRepository(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	repoID, parseErr := uuid.Parse(c.Param("repoId"))
	if parseErr != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid repo id"))
		return
	}
	if err := h.svc.UnlinkRepository(c.Request.Context(), projectID, repoID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// --- Task PR endpoints -----------------------------------------------------

// ListTaskPRs handles GET /projects/:projectId/tasks/:taskId/github/pull-requests.
func (h *GitHubHandler) ListTaskPRs(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	prs, err := h.svc.ListTaskPRs(c.Request.Context(), taskID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	items := make([]dto.PullRequestResponse, len(prs))
	for i, pr := range prs {
		items[i] = dto.PullRequestFromEntity(pr)
	}
	presenter.OK(c, items)
}

// LinkPRToTask handles POST /projects/:projectId/tasks/:taskId/github/pull-requests.
// Fetches the PR from GitHub, caches it, and links it to the task.
func (h *GitHubHandler) LinkPRToTask(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.LinkPRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}
	pr, err := h.svc.LinkPRToTask(c.Request.Context(), projectID, taskID, req.RepoID, req.PRNumber)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.PullRequestFromEntity(pr))
}

// UnlinkPRFromTask handles DELETE /projects/:projectId/tasks/:taskId/github/pull-requests/:prId.
func (h *GitHubHandler) UnlinkPRFromTask(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	prID, parseErr := uuid.Parse(c.Param("prId"))
	if parseErr != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid pr id"))
		return
	}
	if err := h.svc.UnlinkPRFromTask(c.Request.Context(), taskID, prID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// --- Branch endpoint -------------------------------------------------------

// CreateBranch handles POST /projects/:projectId/tasks/:taskId/github/branches.
func (h *GitHubHandler) CreateBranch(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, err.Error()))
		return
	}
	branchName, err := h.svc.CreateBranch(c.Request.Context(), projectID, req.RepoID, req.BranchName, req.SourceBranch)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.CreateBranchResponse{BranchName: branchName})
}

// --- Webhook endpoint ------------------------------------------------------

// ReceiveWebhook handles POST /github/webhook.
// This endpoint is public — GitHub delivers events without a bearer token.
// The HMAC-SHA256 signature in X-Hub-Signature-256 is used for verification.
func (h *GitHubHandler) ReceiveWebhook(c *gin.Context) {
	event := c.GetHeader("X-GitHub-Event")
	signature := c.GetHeader("X-Hub-Signature-256")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	repoFullName := extractRepoFullName(body)
	if repoFullName == "" {
		// No repository in payload — accept silently.
		c.Status(http.StatusNoContent)
		return
	}

	// Always respond 204 so GitHub does not retry on application errors.
	_ = h.svc.HandleWebhookEvent(c.Request.Context(), repoFullName, event, signature, body)
	c.Status(http.StatusNoContent)
}

// extractRepoFullName extracts "repository.full_name" from a GitHub webhook payload.
func extractRepoFullName(payload []byte) string {
	var p struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return ""
	}
	return p.Repository.FullName
}
