package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	githubdom "github.com/paca/api/internal/domain/github"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// -------------------------------------------------------------------------
// GORM models
// -------------------------------------------------------------------------

type githubIntegrationModel struct {
	ID             string    `gorm:"primarykey;type:uuid"`
	ProjectID      string    `gorm:"column:project_id;type:uuid;uniqueIndex"`
	AccessTokenEnc string    `gorm:"column:access_token_enc;not null"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (githubIntegrationModel) TableName() string { return "github_integrations" }

type githubRepositoryModel struct {
	ID               string    `gorm:"primarykey;type:uuid"`
	ProjectID        string    `gorm:"column:project_id;type:uuid"`
	IntegrationID    string    `gorm:"column:integration_id;type:uuid"`
	Owner            string    `gorm:"column:owner;not null"`
	RepoName         string    `gorm:"column:repo_name;not null"`
	FullName         string    `gorm:"column:full_name;not null"`
	WebhookID        int64     `gorm:"column:webhook_id"`
	WebhookSecretEnc string    `gorm:"column:webhook_secret_enc"`
	DefaultBranch    string    `gorm:"column:default_branch;not null;default:main"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (githubRepositoryModel) TableName() string { return "github_repositories" }

type githubPRModel struct {
	ID         string     `gorm:"primarykey;type:uuid"`
	ProjectID  string     `gorm:"column:project_id;type:uuid"`
	RepoID     string     `gorm:"column:repo_id;type:uuid"`
	PRNumber   int        `gorm:"column:pr_number;not null"`
	GitHubPRID int64      `gorm:"column:github_pr_id;not null"`
	Title      string     `gorm:"column:title;not null;default:''"`
	State      string     `gorm:"column:state;not null;default:open"`
	HTMLURL    string     `gorm:"column:html_url;not null;default:''"`
	HeadBranch string     `gorm:"column:head_branch;not null;default:''"`
	BaseBranch string     `gorm:"column:base_branch;not null;default:''"`
	Author     string     `gorm:"column:author;not null;default:''"`
	MergedAt   *time.Time `gorm:"column:merged_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at"`
}

func (githubPRModel) TableName() string { return "github_pull_requests" }

type githubTaskPRLinkModel struct {
	ID            string    `gorm:"primarykey;type:uuid"`
	TaskID        string    `gorm:"column:task_id;type:uuid"`
	PullRequestID string    `gorm:"column:pull_request_id;type:uuid"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (githubTaskPRLinkModel) TableName() string { return "github_task_pr_links" }

// -------------------------------------------------------------------------
// GitHubRepository (implements githubdom.Repository)
// -------------------------------------------------------------------------

// GitHubRepository is the PostgreSQL-backed implementation of
// githubdom.Repository.
type GitHubRepository struct {
	db *gorm.DB
}

// NewGitHubRepository creates a GitHubRepository backed by the given GORM DB.
func NewGitHubRepository(db *gorm.DB) *GitHubRepository {
	return &GitHubRepository{db: db}
}

// -------------------------------------------------------------------------
// IntegrationRepository
// -------------------------------------------------------------------------

func (r *GitHubRepository) FindIntegrationByProject(ctx context.Context, projectID uuid.UUID) (*githubdom.Integration, error) {
	var m githubIntegrationModel
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, githubdom.ErrIntegrationNotFound
	}
	if err != nil {
		return nil, err
	}
	return integrationFromModel(m), nil
}

func (r *GitHubRepository) UpsertIntegration(ctx context.Context, integration *githubdom.Integration) error {
	m := githubIntegrationModel{
		ID:             integration.ID.String(),
		ProjectID:      integration.ProjectID.String(),
		AccessTokenEnc: integration.AccessTokenEnc,
		CreatedAt:      integration.CreatedAt,
		UpdatedAt:      integration.UpdatedAt,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "project_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"access_token_enc", "updated_at"}),
		}).
		Create(&m).Error
}

func (r *GitHubRepository) DeleteIntegration(ctx context.Context, projectID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Delete(&githubIntegrationModel{}).Error
}

// -------------------------------------------------------------------------
// LinkedRepositoryRepository
// -------------------------------------------------------------------------

func (r *GitHubRepository) ListRepositoriesByProject(ctx context.Context, projectID uuid.UUID) ([]*githubdom.LinkedRepository, error) {
	var models []githubRepositoryModel
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]*githubdom.LinkedRepository, len(models))
	for i, m := range models {
		out[i] = repoFromModel(m)
	}
	return out, nil
}

func (r *GitHubRepository) FindRepositoryByID(ctx context.Context, repoID uuid.UUID) (*githubdom.LinkedRepository, error) {
	var m githubRepositoryModel
	err := r.db.WithContext(ctx).
		Where("id = ?", repoID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, githubdom.ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return repoFromModel(m), nil
}

func (r *GitHubRepository) FindRepositoryByFullName(ctx context.Context, fullName string) (*githubdom.LinkedRepository, error) {
	var m githubRepositoryModel
	err := r.db.WithContext(ctx).
		Where("full_name = ?", fullName).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, githubdom.ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return repoFromModel(m), nil
}

func (r *GitHubRepository) InsertRepository(ctx context.Context, repo *githubdom.LinkedRepository) error {
	m := githubRepositoryModel{
		ID:               repo.ID.String(),
		ProjectID:        repo.ProjectID.String(),
		IntegrationID:    repo.IntegrationID.String(),
		Owner:            repo.Owner,
		RepoName:         repo.RepoName,
		FullName:         repo.FullName,
		WebhookID:        repo.WebhookID,
		WebhookSecretEnc: repo.WebhookSecretEnc,
		DefaultBranch:    repo.DefaultBranch,
		CreatedAt:        repo.CreatedAt,
		UpdatedAt:        repo.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *GitHubRepository) DeleteRepositoryByID(ctx context.Context, repoID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", repoID.String()).
		Delete(&githubRepositoryModel{}).Error
}

// -------------------------------------------------------------------------
// PRRepository
// -------------------------------------------------------------------------

func (r *GitHubRepository) FindPRByRepoAndNumber(ctx context.Context, repoID uuid.UUID, prNumber int) (*githubdom.PullRequest, error) {
	var m githubPRModel
	err := r.db.WithContext(ctx).
		Where("repo_id = ? AND pr_number = ?", repoID.String(), prNumber).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, githubdom.ErrPRNotFound
	}
	if err != nil {
		return nil, err
	}
	return prFromModel(m), nil
}

func (r *GitHubRepository) ListPRsForTask(ctx context.Context, taskID uuid.UUID) ([]*githubdom.PullRequest, error) {
	var links []githubTaskPRLinkModel
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID.String()).
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []*githubdom.PullRequest{}, nil
	}

	prIDs := make([]string, len(links))
	for i, l := range links {
		prIDs[i] = l.PullRequestID
	}

	var prs []githubPRModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", prIDs).
		Find(&prs).Error; err != nil {
		return nil, err
	}

	result := make([]*githubdom.PullRequest, len(prs))
	for i, p := range prs {
		result[i] = prFromModel(p)
	}
	return result, nil
}

func (r *GitHubRepository) UpsertPR(ctx context.Context, pr *githubdom.PullRequest) error {
	m := githubPRModel{
		ID:         pr.ID.String(),
		ProjectID:  pr.ProjectID.String(),
		RepoID:     pr.RepoID.String(),
		PRNumber:   pr.PRNumber,
		GitHubPRID: pr.GitHubPRID,
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
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "repo_id"}, {Name: "pr_number"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"title", "state", "html_url", "head_branch", "base_branch",
				"author", "merged_at", "updated_at",
			}),
		}).
		Create(&m).Error
}

// -------------------------------------------------------------------------
// TaskPRLinkRepository
// -------------------------------------------------------------------------

func (r *GitHubRepository) LinkPRToTask(ctx context.Context, link *githubdom.TaskPRLink) error {
	m := githubTaskPRLinkModel{
		ID:            link.ID.String(),
		TaskID:        link.TaskID.String(),
		PullRequestID: link.PullRequestID.String(),
		CreatedAt:     link.CreatedAt,
	}
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&m)
	if res.Error != nil {
		return res.Error
	}
	// If nothing was inserted (duplicate), signal already-linked.
	if res.RowsAffected == 0 {
		return githubdom.ErrPRAlreadyLinked
	}
	return nil
}

func (r *GitHubRepository) UnlinkPRFromTask(ctx context.Context, taskID, prID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("task_id = ? AND pull_request_id = ?", taskID.String(), prID.String()).
		Delete(&githubTaskPRLinkModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return githubdom.ErrPRLinkNotFound
	}
	return nil
}

// -------------------------------------------------------------------------
// Model ↔ domain converters
// -------------------------------------------------------------------------

func integrationFromModel(m githubIntegrationModel) *githubdom.Integration {
	id, _ := uuid.Parse(m.ID)
	projectID, _ := uuid.Parse(m.ProjectID)
	return &githubdom.Integration{
		ID:             id,
		ProjectID:      projectID,
		AccessTokenEnc: m.AccessTokenEnc,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func repoFromModel(m githubRepositoryModel) *githubdom.LinkedRepository {
	id, _ := uuid.Parse(m.ID)
	projectID, _ := uuid.Parse(m.ProjectID)
	integrationID, _ := uuid.Parse(m.IntegrationID)
	return &githubdom.LinkedRepository{
		ID:               id,
		ProjectID:        projectID,
		IntegrationID:    integrationID,
		Owner:            m.Owner,
		RepoName:         m.RepoName,
		FullName:         m.FullName,
		WebhookID:        m.WebhookID,
		WebhookSecretEnc: m.WebhookSecretEnc,
		DefaultBranch:    m.DefaultBranch,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func prFromModel(m githubPRModel) *githubdom.PullRequest {
	id, _ := uuid.Parse(m.ID)
	projectID, _ := uuid.Parse(m.ProjectID)
	repoID, _ := uuid.Parse(m.RepoID)
	return &githubdom.PullRequest{
		ID:         id,
		ProjectID:  projectID,
		RepoID:     repoID,
		PRNumber:   m.PRNumber,
		GitHubPRID: m.GitHubPRID,
		Title:      m.Title,
		State:      m.State,
		HTMLURL:    m.HTMLURL,
		HeadBranch: m.HeadBranch,
		BaseBranch: m.BaseBranch,
		Author:     m.Author,
		MergedAt:   m.MergedAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
