// Package githubsvc implements the GitHub integration service.
// It validates PATs, manages repository linking with automatic webhook
// creation, caches pull requests, and processes incoming webhook events.
package githubsvc

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	githubdom "github.com/paca/api/internal/domain/github"
	"github.com/paca/api/internal/platform/githubclient"
	"github.com/paca/api/internal/platform/secret"
)

// Service implements githubdom.Service.
type Service struct {
	repo          githubdom.Repository
	enc           *secret.Encryptor
	webhookURL    string // public URL where GitHub will POST events
	clientBaseURL string // optional override for the GitHub API base URL (used in tests)
}

// New creates a new GitHub integration Service.
// webhookURL is the public URL for the webhook endpoint (e.g. "https://api.example.com/api/v1/github/webhook").
// It may be empty during development; webhook creation will return ErrWebhookURLRequired in that case.
func New(repo githubdom.Repository, enc *secret.Encryptor, webhookURL string) *Service {
	return &Service{repo: repo, enc: enc, webhookURL: webhookURL}
}

// WithClientBaseURL sets a custom GitHub API base URL. Intended for tests only.
func (s *Service) WithClientBaseURL(base string) *Service {
	s.clientBaseURL = base
	return s
}

// newClient creates a GitHub API client for the given token, using the
// configured base URL (defaults to https://api.github.com).
func (s *Service) newClient(token string) *githubclient.Client {
	if s.clientBaseURL != "" {
		return githubclient.NewWithBase(token, s.clientBaseURL)
	}
	return githubclient.New(token)
}

// SetToken validates the supplied PAT against the GitHub API and stores it
// encrypted for the given project.
func (s *Service) SetToken(ctx context.Context, projectID uuid.UUID, plainToken string) (*githubdom.Integration, error) {
	// Validate the token by hitting the GitHub /user endpoint.
	ghClient := s.newClient(plainToken)
	if err := ghClient.ValidateToken(ctx); err != nil {
		var apiErr *githubclient.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
			return nil, githubdom.ErrInvalidToken
		}
		return nil, fmt.Errorf("github: validate token: %w", err)
	}

	encrypted, err := s.enc.Encrypt(plainToken)
	if err != nil {
		return nil, fmt.Errorf("github: encrypt token: %w", err)
	}

	now := time.Now().UTC()
	integration := &githubdom.Integration{
		ID:             uuid.New(),
		ProjectID:      projectID,
		AccessTokenEnc: encrypted,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.UpsertIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("github: persist integration: %w", err)
	}
	// Return a copy without the encrypted value exposed.
	out := *integration
	out.AccessTokenEnc = ""
	return &out, nil
}

// GetIntegration returns the GitHub integration for a project.
func (s *Service) GetIntegration(ctx context.Context, projectID uuid.UUID) (*githubdom.Integration, error) {
	integration, err := s.repo.FindIntegrationByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	// Strip the encrypted token from the returned value.
	out := *integration
	out.AccessTokenEnc = ""
	return &out, nil
}

// DeleteIntegration removes the PAT and all linked repositories (including their webhooks).
func (s *Service) DeleteIntegration(ctx context.Context, projectID uuid.UUID) error {
	// Best-effort: delete all repository webhooks first.
	repos, err := s.repo.ListRepositoriesByProject(ctx, projectID)
	if err == nil {
		for _, repo := range repos {
			_ = s.deleteWebhook(ctx, projectID, repo)
		}
	}
	return s.repo.DeleteIntegration(ctx, projectID)
}

// ListRepositories returns all GitHub repositories accessible with the project's PAT.
func (s *Service) ListRepositories(ctx context.Context, projectID uuid.UUID) ([]githubdom.RepoInfo, error) {
	token, err := s.decryptToken(ctx, projectID)
	if err != nil {
		return nil, err
	}

	ghClient := s.newClient(token)
	repos, err := ghClient.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("github: list repositories: %w", err)
	}

	infos := make([]githubdom.RepoInfo, len(repos))
	for i, r := range repos {
		infos[i] = githubdom.RepoInfo{
			FullName:      r.FullName,
			Owner:         r.Owner.Login,
			Name:          r.Name,
			DefaultBranch: r.DefaultBranch,
			Private:       r.Private,
		}
	}
	return infos, nil
}

// LinkRepository links a GitHub repository to a project and creates a webhook.
func (s *Service) LinkRepository(ctx context.Context, projectID uuid.UUID, owner, repoName string) (*githubdom.LinkedRepository, error) {
	if s.webhookURL == "" {
		return nil, githubdom.ErrWebhookURLRequired
	}

	integration, err := s.repo.FindIntegrationByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	token, err := s.enc.Decrypt(integration.AccessTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("github: decrypt token: %w", err)
	}

	ghClient := s.newClient(token)

	// Fetch repo metadata to confirm access and get the default branch.
	ghRepo, err := ghClient.GetRepository(ctx, owner, repoName)
	if err != nil {
		var apiErr *githubclient.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 403 || apiErr.StatusCode == 404) {
			return nil, githubdom.ErrRepoNotAccessible
		}
		return nil, fmt.Errorf("github: get repository: %w", err)
	}

	// If the same repository is already linked, reject the request.
	existing, err := s.repo.ListRepositoriesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("github: list repositories: %w", err)
	}
	for _, r := range existing {
		if r.FullName == ghRepo.FullName {
			return nil, githubdom.ErrRepoAlreadyLinked
		}
	}

	// Generate a random webhook secret.
	webhookSecret, err := generateWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("github: generate webhook secret: %w", err)
	}

	webhookID, err := ghClient.CreateWebhook(ctx, owner, repoName, s.webhookURL, webhookSecret,
		[]string{"push", "pull_request", "check_run"})
	if err != nil {
		var apiErr *githubclient.APIError
		if errors.As(err, &apiErr) && isWebhookURLNotPublic(apiErr) {
			return nil, githubdom.ErrWebhookURLNotPublic
		}
		return nil, githubdom.ErrWebhookCreationFailed
	}

	encSecret, err := s.enc.Encrypt(webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("github: encrypt webhook secret: %w", err)
	}

	now := time.Now().UTC()
	linked := &githubdom.LinkedRepository{
		ID:               uuid.New(),
		ProjectID:        projectID,
		IntegrationID:    integration.ID,
		Owner:            ghRepo.Owner.Login,
		RepoName:         ghRepo.Name,
		FullName:         ghRepo.FullName,
		WebhookID:        webhookID,
		WebhookSecretEnc: encSecret,
		DefaultBranch:    ghRepo.DefaultBranch,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.InsertRepository(ctx, linked); err != nil {
		return nil, fmt.Errorf("github: persist linked repository: %w", err)
	}
	return linked, nil
}

func isWebhookURLNotPublic(apiErr *githubclient.APIError) bool {
	if apiErr == nil {
		return false
	}
	if apiErr.StatusCode != 422 {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(apiErr.Message + " " + apiErr.Details))
	return strings.Contains(msg, "isn't reachable over the public internet") ||
		strings.Contains(msg, "not publicly reachable") ||
		strings.Contains(msg, "localhost")
}

// ListLinkedRepositories returns all linked repositories for a project.
func (s *Service) ListLinkedRepositories(ctx context.Context, projectID uuid.UUID) ([]*githubdom.LinkedRepository, error) {
	return s.repo.ListRepositoriesByProject(ctx, projectID)
}

// UnlinkRepository removes a specific repository link and deletes its webhook.
func (s *Service) UnlinkRepository(ctx context.Context, projectID, repoID uuid.UUID) error {
	repo, err := s.repo.FindRepositoryByID(ctx, repoID)
	if err != nil {
		return err
	}
	_ = s.deleteWebhook(ctx, projectID, repo)
	return s.repo.DeleteRepositoryByID(ctx, repoID)
}

// ListTaskPRs returns all pull requests linked to a task.
func (s *Service) ListTaskPRs(ctx context.Context, taskID uuid.UUID) ([]*githubdom.PullRequest, error) {
	return s.repo.ListPRsForTask(ctx, taskID)
}

// LinkPRToTask fetches the PR from GitHub, caches it, and links it to the task.
func (s *Service) LinkPRToTask(ctx context.Context, projectID, taskID, repoID uuid.UUID, prNumber int) (*githubdom.PullRequest, error) {
	token, err := s.decryptToken(ctx, projectID)
	if err != nil {
		return nil, err
	}

	linked, err := s.repo.FindRepositoryByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	ghClient := s.newClient(token)
	ghPR, err := ghClient.GetPullRequest(ctx, linked.Owner, linked.RepoName, prNumber)
	if err != nil {
		var apiErr *githubclient.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil, githubdom.ErrPRNotFound
		}
		return nil, fmt.Errorf("github: fetch pull request: %w", err)
	}

	pr := ghPRToDomain(ghPR, projectID, linked.ID)

	// Try to find existing cached PR to preserve its ID.
	existing, findErr := s.repo.FindPRByRepoAndNumber(ctx, linked.ID, prNumber)
	if findErr == nil {
		pr.ID = existing.ID
		pr.CreatedAt = existing.CreatedAt
	}

	if err := s.repo.UpsertPR(ctx, pr); err != nil {
		return nil, fmt.Errorf("github: cache pull request: %w", err)
	}

	link := &githubdom.TaskPRLink{
		ID:            uuid.New(),
		TaskID:        taskID,
		PullRequestID: pr.ID,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repo.LinkPRToTask(ctx, link); err != nil {
		return nil, err
	}
	return pr, nil
}

// UnlinkPRFromTask removes the task-PR association.
func (s *Service) UnlinkPRFromTask(ctx context.Context, taskID, prID uuid.UUID) error {
	return s.repo.UnlinkPRFromTask(ctx, taskID, prID)
}

// CreateBranch creates a new branch in the linked repository identified by repoID.
func (s *Service) CreateBranch(ctx context.Context, projectID, repoID uuid.UUID, branchName, sourceBranch string) (string, error) {
	token, err := s.decryptToken(ctx, projectID)
	if err != nil {
		return "", err
	}

	linked, err := s.repo.FindRepositoryByID(ctx, repoID)
	if err != nil {
		return "", err
	}

	if sourceBranch == "" {
		sourceBranch = linked.DefaultBranch
	}

	ghClient := s.newClient(token)
	if err := ghClient.CreateBranch(ctx, linked.Owner, linked.RepoName, branchName, sourceBranch); err != nil {
		return "", fmt.Errorf("github: create branch: %w", err)
	}
	return branchName, nil
}

// HandleWebhookEvent processes an incoming GitHub webhook event.
func (s *Service) HandleWebhookEvent(ctx context.Context, repoFullName, event, signature string, payload []byte) error {
	// Look up the repository to find its webhook secret.
	linked, err := s.repo.FindRepositoryByFullName(ctx, repoFullName)
	if err != nil {
		// Unknown repo — silently ignore.
		return nil
	}

	if linked.WebhookSecretEnc != "" {
		webhookSecret, err := s.enc.Decrypt(linked.WebhookSecretEnc)
		if err != nil {
			return fmt.Errorf("github: decrypt webhook secret: %w", err)
		}
		if !verifyHMAC(payload, webhookSecret, signature) {
			return fmt.Errorf("github: webhook signature mismatch")
		}
	}

	switch event {
	case "pull_request":
		return s.handlePREvent(ctx, linked, payload)
	case "push", "check_run":
		// These can be handled in the future for CI status / branch activity.
		return nil
	default:
		return nil
	}
}

// -------------------------------------------------------------------------
// internal helpers
// -------------------------------------------------------------------------

// decryptToken looks up and decrypts the PAT for a project.
func (s *Service) decryptToken(ctx context.Context, projectID uuid.UUID) (string, error) {
	integration, err := s.repo.FindIntegrationByProject(ctx, projectID)
	if err != nil {
		return "", err
	}
	token, err := s.enc.Decrypt(integration.AccessTokenEnc)
	if err != nil {
		return "", fmt.Errorf("github: decrypt token: %w", err)
	}
	return token, nil
}

// deleteWebhook looks up the PAT and calls deleteWebhookWithToken.
func (s *Service) deleteWebhook(ctx context.Context, projectID uuid.UUID, repo *githubdom.LinkedRepository) error {
	if repo.WebhookID == 0 {
		return nil
	}
	token, err := s.decryptToken(ctx, projectID)
	if err != nil {
		return err
	}
	return s.deleteWebhookWithToken(ctx, token, repo)
}

func (s *Service) deleteWebhookWithToken(_ context.Context, token string, repo *githubdom.LinkedRepository) error {
	if repo.WebhookID == 0 {
		return nil
	}
	ghClient := s.newClient(token)
	return ghClient.DeleteWebhook(context.Background(), repo.Owner, repo.RepoName, repo.WebhookID)
}

// handlePREvent upserts cached PR data from a pull_request webhook event.
func (s *Service) handlePREvent(ctx context.Context, linked *githubdom.LinkedRepository, payload []byte) error {
	var event struct {
		PullRequest githubclient.PullRequest `json:"pull_request"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("github: parse pull_request event: %w", err)
	}

	pr := ghPRToDomain(&event.PullRequest, linked.ProjectID, linked.ID)

	// Preserve existing cache entry ID if it exists.
	if existing, err := s.repo.FindPRByRepoAndNumber(ctx, linked.ID, pr.PRNumber); err == nil {
		pr.ID = existing.ID
		pr.CreatedAt = existing.CreatedAt
	}

	return s.repo.UpsertPR(ctx, pr)
}

// ghPRToDomain converts a GitHub API pull request to the domain model.
func ghPRToDomain(ghPR *githubclient.PullRequest, projectID, repoID uuid.UUID) *githubdom.PullRequest {
	state := ghPR.State
	if ghPR.Merged {
		state = "merged"
	}
	now := time.Now().UTC()
	return &githubdom.PullRequest{
		ID:         uuid.New(),
		ProjectID:  projectID,
		RepoID:     repoID,
		PRNumber:   ghPR.Number,
		GitHubPRID: ghPR.ID,
		Title:      ghPR.Title,
		State:      state,
		HTMLURL:    ghPR.HTMLURL,
		HeadBranch: ghPR.Head.Ref,
		BaseBranch: ghPR.Base.Ref,
		Author:     ghPR.User.Login,
		MergedAt:   ghPR.MergedAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// generateWebhookSecret returns a cryptographically random 32-byte hex string.
func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// verifyHMAC checks the GitHub HMAC-SHA256 webhook signature.
// signature has the form "sha256=<hex>".
func verifyHMAC(payload []byte, secret, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hmac.Equal(mac.Sum(nil), expected)
}
