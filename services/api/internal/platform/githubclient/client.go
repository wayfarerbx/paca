// Package githubclient provides a thin GitHub REST API v3 client that uses a
// personal access token for authentication.  It avoids pulling in the full
// google/go-github SDK to keep dependencies minimal.
package githubclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL        = "https://api.github.com"
	apiVersion     = "2022-11-28"
	defaultTimeout = 15 * time.Second
)

// Repository is a minimal GitHub repository representation.
type Repository struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Owner         Owner  `json:"owner"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

// Owner is a GitHub user or organisation login.
type Owner struct {
	Login string `json:"login"`
}

// PullRequest is a minimal representation of a GitHub pull request.
type PullRequest struct {
	ID       int64      `json:"id"`
	Number   int        `json:"number"`
	Title    string     `json:"title"`
	State    string     `json:"state"` // "open" | "closed"
	HTMLURL  string     `json:"html_url"`
	Head     PRRef      `json:"head"`
	Base     PRRef      `json:"base"`
	User     PRUser     `json:"user"`
	Merged   bool       `json:"merged"`
	MergedAt *time.Time `json:"merged_at"`
}

// PRRef carries the branch name in a PR head/base.
type PRRef struct {
	Ref string `json:"ref"`
}

// PRUser is the GitHub login of a PR author.
type PRUser struct {
	Login string `json:"login"`
}

// APIError carries a non-2xx HTTP status and the GitHub error message.
type APIError struct {
	StatusCode int
	Message    string
	Details    string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("github: API error %d: %s (%s)", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("github: API error %d: %s", e.StatusCode, e.Message)
}

// Client is a GitHub REST API v3 client authenticated via a PAT.
type Client struct {
	token      string
	base       string
	httpClient *http.Client
}

// New creates a new GitHub API Client with the given personal access token.
func New(token string) *Client {
	return &Client{
		token:      token,
		base:       baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// NewWithBase creates a Client that targets a custom base URL instead of the
// default https://api.github.com. Intended for use in tests only.
func NewWithBase(token, base string) *Client {
	return &Client{
		token:      token,
		base:       base,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// ValidateToken checks whether the token is accepted by the GitHub API.
// It calls GET /user to verify authentication.
func (c *Client) ValidateToken(ctx context.Context) error {
	var user struct {
		Login string `json:"login"`
	}
	return c.get(ctx, c.base+"/user", &user)
}

// ListRepositories returns all repositories accessible to the token owner
// (including organisation repositories where the owner is a collaborator).
// Results are paginated; all pages are fetched.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var all []Repository
	page := 1
	for {
		url := fmt.Sprintf("%s/user/repos?affiliation=owner,collaborator&per_page=100&page=%d", c.base, page)
		var batch []Repository
		if err := c.get(ctx, url, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// GetRepository fetches metadata for a specific repository.
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.base, owner, repo)
	var r Repository
	if err := c.get(ctx, url, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateWebhook registers a webhook on the given repository.
// events should be a list like ["push", "pull_request", "check_run"].
// Returns the GitHub webhook ID assigned by the API.
func (c *Client) CreateWebhook(ctx context.Context, owner, repo, webhookURL, secret string, events []string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", c.base, owner, repo)
	body := map[string]any{
		"name":   "web",
		"active": true,
		"events": events,
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       secret,
		},
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	if err := c.post(ctx, url, body, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

// DeleteWebhook removes a webhook from a repository.
// A 404 from GitHub (hook already gone) is silently ignored.
func (c *Client) DeleteWebhook(ctx context.Context, owner, repo string, hookID int64) error {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%d", c.base, owner, repo, hookID)
	return c.doDelete(ctx, url)
}

// GetPullRequest fetches a single pull request by number.
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, prNumber int) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.base, owner, repo, prNumber)
	var pr PullRequest
	if err := c.get(ctx, url, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// CreateBranch creates a new branch in the repository from an existing source branch.
func (c *Client) CreateBranch(ctx context.Context, owner, repo, newBranch, sourceBranch string) error {
	// Resolve the source branch tip SHA.
	refURL := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", c.base, owner, repo, sourceBranch)
	var refResp struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := c.get(ctx, refURL, &refResp); err != nil {
		return fmt.Errorf("resolve source branch %q: %w", sourceBranch, err)
	}

	// Create the ref pointing to the same SHA.
	createURL := fmt.Sprintf("%s/repos/%s/%s/git/refs", c.base, owner, repo)
	body := map[string]string{
		"ref": "refs/heads/" + newBranch,
		"sha": refResp.Object.SHA,
	}
	return c.post(ctx, createURL, body, nil)
}

// --- internal helpers -------------------------------------------------------

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
}

func (c *Client) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("githubclient: build request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("githubclient: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("githubclient: decode response body: %w", err)
		}
	}
	return nil
}

func (c *Client) post(ctx context.Context, url string, body, out any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("githubclient: encode request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("githubclient: build request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("githubclient: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("githubclient: decode response body: %w", err)
		}
	}
	return nil
}

func (c *Client) doDelete(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("githubclient: build request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("githubclient: execute request: %w", err)
	}
	defer resp.Body.Close()

	// Treat 404 as success: the webhook is already gone.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	return nil
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errBody struct {
		Message string `json:"message"`
		Errors  []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	_ = json.Unmarshal(body, &errBody)
	if errBody.Message == "" {
		errBody.Message = http.StatusText(resp.StatusCode)
	}

	details := make([]string, 0, len(errBody.Errors))
	for _, e := range errBody.Errors {
		msg := strings.TrimSpace(e.Message)
		if msg == "" {
			continue
		}
		if e.Field != "" {
			details = append(details, fmt.Sprintf("%s: %s", e.Field, msg))
			continue
		}
		details = append(details, msg)
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    errBody.Message,
		Details:    strings.Join(details, "; "),
	}
}
