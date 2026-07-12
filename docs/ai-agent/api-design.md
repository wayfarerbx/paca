# AI Agent — REST API Design

This document describes the public REST endpoints added to `services/api` for AI Agent management.

All endpoints follow the existing Paca API conventions: JWT authentication, project-scoped authorization, and standard error envelope `{"error": "..."}`.

---

## Agent Management

### `GET /api/v1/projects/:projectId/agents`

List all agents in a project.

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Dev Bot",
    "handle": "dev-bot",
    "avatar_url": null,
    "agent_type": {
      "id": "uuid",
      "name": "Developer",
      "slug": "developer"
    },
    "llm_provider": "anthropic",
    "llm_model": "claude-sonnet-4-6",
    "max_iterations": 50,
    "timeout_minutes": 30,
    "member_id": "uuid",
    "created_at": "2026-05-19T00:00:00Z"
  }
]
```

---

### `POST /api/v1/projects/:projectId/agents`

Create a new agent. This also creates the corresponding `project_members` row with `member_type = 'agent'`.

**Request body:**
```json
{
  "name": "Dev Bot",
  "handle": "dev-bot",
  "agent_type_id": "uuid",
  "llm_provider": "anthropic",
  "llm_model": "claude-sonnet-4-6",
  "llm_api_key": "sk-ant-...",
  "llm_base_url": null,
  "system_prompt": "You are a senior software engineer...",
  "max_iterations": 50,
  "timeout_minutes": 30,
  "project_role_id": "uuid"
}
```

The `llm_api_key` is stored in the secrets store (e.g., encrypted column or external vault); only a reference is kept in the `agents` table.

**Response:** `201 Created` with the created agent object.

---

### `GET /api/v1/projects/:projectId/agents/:agentId`

Get a single agent, including its MCP servers and skills.

**Response:**
```json
{
  "id": "uuid",
  "name": "Dev Bot",
  "handle": "dev-bot",
  "system_prompt": "...",
  "mcp_servers": [
    {
      "id": "uuid",
      "server_name": "fetch",
      "transport": "stdio",
      "command": "uvx",
      "args": ["mcp-server-fetch"],
      "is_enabled": true
    }
  ],
  "skills": [
    {
      "id": "uuid",
      "skill_name": "developer",
      "skill_source": "inline",
      "triggers": ["implement", "fix", "refactor"],
      "is_enabled": true
    }
  ]
}
```

---

### `PATCH /api/v1/projects/:projectId/agents/:agentId`

Update agent configuration (name, handle, LLM, system prompt, limits).

---

### `DELETE /api/v1/projects/:projectId/agents/:agentId`

Soft-delete the agent and its corresponding project member. Stops any running conversations for this agent.

---

## Agent MCP Servers

### `POST /api/v1/projects/:projectId/agents/:agentId/mcp-servers`

Add an MCP server to an agent.

**Request body:**
```json
{
  "server_name": "repomix",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "repomix@1.4.2", "--mcp"],
  "env": {}
}
```

---

### `PATCH /api/v1/projects/:projectId/agents/:agentId/mcp-servers/:serverId`

Update or enable/disable an MCP server.

---

### `DELETE /api/v1/projects/:projectId/agents/:agentId/mcp-servers/:serverId`

Remove an MCP server from an agent.

---

## Agent Skills

### `POST /api/v1/projects/:projectId/agents/:agentId/skills`

Add a skill to an agent.

**Request body (inline):**
```json
{
  "skill_name": "code-review-guide",
  "skill_source": "inline",
  "skill_content": "---\nname: code-review-guide\ndescription: Project code review guidelines.\n---\n\n# Code Review Guidelines\n...",
  "triggers": ["review", "pr"]
}
```

**Request body (GitHub URL):**
```json
{
  "skill_name": "github-workflow",
  "skill_source": "github_url",
  "source_url": "https://github.com/OpenHands/extensions/blob/main/github/SKILL.md",
  "triggers": ["github", "git"]
}
```

---

### `PATCH /api/v1/projects/:projectId/agents/:agentId/skills/:skillId`

Update or enable/disable a skill.

---

### `DELETE /api/v1/projects/:projectId/agents/:agentId/skills/:skillId`

Remove a skill.

---

## Agent Types

### `GET /api/v1/agent-types`

List built-in and project-scoped agent types.

**Query params:** `?project_id=<uuid>` — include project-scoped types.

---

### `POST /api/v1/projects/:projectId/agent-types`

Create a custom agent type for a project.

---

## Conversations

### `GET /api/v1/projects/:projectId/agents/:agentId/conversations`

List all conversations for an agent. Sorted by `created_at DESC`.

**Query params:** `?status=running&task_id=<uuid>&limit=20&offset=0`

**Response:**
```json
{
  "conversations": [
    {
      "id": "uuid",
      "trigger_type": "task_assigned",
      "task_id": "uuid",
      "task_title": "Implement OAuth login",
      "status": "running",
      "iteration_count": 12,
      "branch_name": "agent/implement-oauth-login",
      "pr_url": null,
      "started_at": "2026-05-19T10:00:00Z",
      "finished_at": null
    }
  ],
  "total": 1
}
```

---

### `GET /api/v1/projects/:projectId/conversations/:conversationId`

Get full conversation detail including events.

**Query params:** `?include_events=true&event_limit=100&event_offset=0`

---

### `GET /api/v1/projects/:projectId/conversations/:conversationId/events`

Paginated list of conversation events.

**Response:**
```json
{
  "events": [
    {
      "id": "uuid",
      "event_index": 0,
      "event_type": "MessageAction",
      "event_source": "user",
      "payload": { "message": "Implement the OAuth login flow..." },
      "created_at": "2026-05-19T10:00:01Z"
    },
    {
      "id": "uuid",
      "event_index": 1,
      "event_type": "CmdRunAction",
      "event_source": "agent",
      "payload": { "command": "ls -la /workspace/repo/src" },
      "created_at": "2026-05-19T10:00:03Z"
    }
  ],
  "total": 45,
  "has_more": true
}
```

---

### `POST /api/v1/projects/:projectId/conversations/:conversationId/pause`

Pause a running conversation.

**Response:** `200 OK` `{"status": "paused"}`

---

### `POST /api/v1/projects/:projectId/conversations/:conversationId/resume`

Resume a paused conversation.

**Response:** `200 OK` `{"status": "running"}`

---

### `POST /api/v1/projects/:projectId/conversations/:conversationId/stop`

Permanently stop a conversation and destroy its container.

**Response:** `200 OK` `{"status": "stopped"}`

---

### `POST /api/v1/projects/:projectId/conversations/:conversationId/message`

Send an additional message to an active conversation (running or paused).

**Request body:**
```json
{
  "message": "Actually, please also add tests for the new endpoint."
}
```

---

## Agent Chat Sessions

### `GET /api/v1/projects/:projectId/agents/:agentId/chat-sessions`

List chat sessions for a member+agent pair.

---

### `POST /api/v1/projects/:projectId/agents/:agentId/chat-sessions`

Start a new chat session with an agent.

**Request body:**
```json
{
  "message": "Can you help me write the acceptance criteria for PACA-42?"
}
```

**Response:** `201 Created` with the new session and the queued conversation.

---

### `POST /api/v1/projects/:projectId/chat-sessions/:sessionId/messages`

Send a follow-up message in an existing chat session.

---

### `GET /api/v1/projects/:projectId/chat-sessions/:sessionId/messages`

List all messages in a chat session (human turns + agent replies).

---

## Real-time Events (Socket.IO)

The `services/realtime` service emits the following events to project rooms when conversation state changes:

| Event | Payload | When |
|---|---|---|
| `agent:conversation:started` | `{ conversation_id, agent_id, trigger_type }` | Conversation begins |
| `agent:conversation:event` | `{ conversation_id, event_type, event_source, event_index, payload }` | Each SDK event |
| `agent:conversation:status` | `{ conversation_id, status }` | Status change (paused, resumed, finished, stopped, failed) |
| `agent:conversation:pr_created` | `{ conversation_id, task_id, pr_url, branch_name }` | Agent created a PR |
| `agent:chat:reply` | `{ session_id, message, conversation_id }` | Agent replied in chat |
