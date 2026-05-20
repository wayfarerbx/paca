# AI Agent — Repository Plugin Adapter

This document describes how AI agents securely access source code and create pull requests through the Paca repository plugin system.

## Design Goals

1. **Agents never store credentials** — all VCS tokens are ephemeral and fetched on demand.
2. **Credentials are never visible in agent output** — injected via OpenHands `SecretSource` which masks values in all logs and events.
3. **Plugin plugins remain the single source of VCS auth** — the GitHub plugin, GitLab plugin, etc., own token generation.
4. **The agent cannot read the raw token value** — the `SecretSource` pattern exposes the token only as an environment variable inside the container's process, not as a string in agent context.

---

## Protocol

```
services/ai-agent                          services/api (plugin adapter endpoint)
        │                                              │
        │  GET /internal/plugins/:id/repo-token        │
        │  Headers: X-Internal-Key: <shared-secret>    │
        │  Params: ?project_id=<uuid>&scopes=read,write│
        │ ─────────────────────────────────────────────►│
        │                                              │── invoke plugin's token provider
        │                                              │── GitHub: create installation token
        │                                              │── GitLab: create project access token
        │  200 OK                                      │
        │  { "token": "ghs_...", "expires_at": 1234 }  │
        │ ◄─────────────────────────────────────────────│
        │                                              │
        │  Inject into Conversation as SecretSource    │
        │  (re-fetches when within 60s of expiry)      │
```

### Internal Endpoint

`GET /internal/plugins/:pluginId/repo-token`

**Authorization:** `X-Internal-Key` header with a shared secret known only to Paca services. This endpoint is **not** exposed through the public API gateway.

**Query parameters:**

| Parameter | Description |
|---|---|
| `project_id` | UUID of the project. The plugin uses this to look up the linked repository. |
| `scopes` | Comma-separated: `read`, `write`. Agents that should not push code use `read` only. |

**Response:**
```json
{
  "token": "ghs_AbCdEfGhIjKlMnOpQrStUvWxYz",
  "expires_at": 1748649600,
  "clone_url": "https://github.com/org/my-repo.git",
  "default_branch": "main"
}
```

**Error responses:**

| Status | Meaning |
|---|---|
| `404` | Plugin not found or project has no linked repository |
| `403` | Plugin not authorized to generate tokens for this project |
| `502` | Upstream VCS API error |

---

## Implementation per Plugin

### GitHub Plugin

Uses the GitHub App installation token API:

```
POST https://api.github.com/app/installations/:installation_id/access_tokens
Body: { "repositories": ["repo-name"], "permissions": { "contents": "write", "pull_requests": "write" } }
```

Tokens expire after 60 minutes. The `SecretSource` auto-renews 60 seconds before expiry.

### GitLab Plugin

Uses GitLab project access tokens with `read_repository` + `write_repository` scopes, configured with a 1-hour expiry:

```
POST https://gitlab.com/api/v4/projects/:id/access_tokens
Body: { "name": "paca-agent-<conversation_id>", "scopes": ["read_repository", "write_repository"], "expires_at": "..." }
```

The token is revoked by the plugin adapter when the conversation finishes.

---

## Git Operations Inside the Container

The agent receives the token as the environment variable `GIT_TOKEN`. All git operations use this token embedded in the HTTPS URL. The agent never sees the token value in its reasoning — it is resolved by the shell at runtime.

**Recommended initial prompt fragment for coding agents:**

```
Clone the repository using:
  git clone https://x-access-token:$GIT_TOKEN@<clone_url> /workspace/repo

Work inside /workspace/repo. Create a feature branch named agent/<task-slug>
before making any changes.
```

---

## PR Creation Flow

When the agent signals completion:

1. `services/ai-agent` reads `branch_name` from the conversation finish action.
2. Calls the plugin adapter's PR creation endpoint:

```
POST /internal/plugins/:pluginId/pull-requests
Body:
{
  "project_id": "<uuid>",
  "head_branch": "agent/implement-oauth-login",
  "base_branch": "main",
  "title": "feat: implement OAuth login flow (PACA-42)",
  "body": "Agent-generated PR.\n\n## Changes\n...",
  "task_id": "<uuid>"
}
```

3. The plugin creates the PR and returns the URL.
4. `services/api` links the PR to the task and posts a comment with the PR URL.

### GitHub Plugin PR endpoint

```
POST https://api.github.com/repos/:owner/:repo/pulls
```

### GitLab Plugin PR endpoint

```
POST https://gitlab.com/api/v4/projects/:id/merge_requests
```

---

## Security Considerations

| Concern | Mitigation |
|---|---|
| Token leakage in logs | `SecretSource` masks all occurrences of the token value in OpenHands event output |
| Token used beyond conversation scope | Tokens have a maximum TTL (60 min for GitHub, configurable for GitLab) and are revoked on conversation end |
| Agent pushing to protected branches | PR creation enforces a separate branch; direct pushes to `main` are not permitted by the plugin adapter |
| SSRF via clone URL | Clone URL is fetched from the plugin (trusted), not from user input. The URL is validated to match a configured repository. |
| Container network access to internal services | Agent containers run on an isolated Docker network with no route to Paca services; the token is the only channel out |
