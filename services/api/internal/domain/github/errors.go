package githubdom

import "errors"

var (
	// ErrIntegrationNotFound is returned when the project has no GitHub
	// integration configured (no PAT stored).
	ErrIntegrationNotFound = errors.New("github: integration not found for project")

	// ErrRepositoryNotFound is returned when the project has no linked GitHub
	// repository.
	ErrRepositoryNotFound = errors.New("github: repository not linked for project")

	// ErrPRNotFound is returned when the requested pull request does not exist
	// in the local cache or on GitHub.
	ErrPRNotFound = errors.New("github: pull request not found")

	// ErrPRLinkNotFound is returned when a task-PR link does not exist.
	ErrPRLinkNotFound = errors.New("github: task pull-request link not found")

	// ErrPRAlreadyLinked is returned when the pull request is already linked
	// to the task.
	ErrPRAlreadyLinked = errors.New("github: pull request already linked to this task")

	// ErrInvalidToken is returned when the GitHub API rejects the supplied PAT.
	ErrInvalidToken = errors.New("github: invalid or expired personal access token")

	// ErrWebhookURLRequired is returned when the integration is configured but
	// the service has no public webhook URL set, preventing webhook creation.
	ErrWebhookURLRequired = errors.New("github: public webhook URL is not configured")

	// ErrRepoNotAccessible is returned when the GitHub API cannot access the
	// requested repository (e.g. 404 or insufficient PAT permissions).
	ErrRepoNotAccessible = errors.New("github: repository not found or not accessible")

	// ErrRepoAlreadyLinked is returned when the repository is already linked
	// to the project.
	ErrRepoAlreadyLinked = errors.New("github: repository is already linked to this project")

	// ErrWebhookCreationFailed is returned when creating a webhook on the
	// GitHub repository fails (e.g. insufficient permissions).
	ErrWebhookCreationFailed = errors.New("github: failed to create webhook on repository")

	// ErrWebhookURLNotPublic is returned when the configured webhook URL is not
	// publicly reachable by GitHub (for example, localhost in local development).
	ErrWebhookURLNotPublic = errors.New("github: webhook URL is not publicly reachable")
)
