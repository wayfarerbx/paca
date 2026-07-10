# AI Agent — Database Schema

This document describes the new tables and modifications required to support AI Agents in Paca.

## Migration

File: `000008_add_ai_agents.sql`

---

## New Tables

### `agent_types`

Defines reusable agent type templates. Built-in types are seeded at startup; users can create project-scoped custom types.

```dbml
Table agent_types {
  id uuid [primary key]
  project_id uuid [null, ref: > projects.id, note: 'null = built-in global type; non-null = project-scoped custom type']
  name varchar [not null]
  description text [not null, default: '']
  slug varchar [not null, note: 'Machine-readable identifier: po-assistant | ba | developer | manual-tester | custom']
  default_llm_provider varchar [not null, note: 'LiteLLM provider prefix, e.g. anthropic, openai, azure']
  default_llm_model varchar [not null, note: 'LiteLLM model name, e.g. claude-sonnet-4-6']
  default_system_prompt text [not null, default: '']
  is_builtin boolean [not null, default: false]
  created_at timestamp [not null]
  updated_at timestamp [not null]

  indexes {
    (project_id, slug) [unique, note: 'Partial unique: WHERE project_id IS NOT NULL']
    (slug) [unique, note: 'Partial unique: WHERE project_id IS NULL (global built-ins)']
  }
}
```

### `agents`

Represents an AI agent belonging to a project.

```dbml
Table agents {
  id uuid [primary key]
  project_id uuid [not null, ref: > projects.id]
  agent_type_id uuid [not null, ref: > agent_types.id]
  name varchar [not null, note: 'Display name shown in the project member list']
  handle varchar [not null, note: '@mention handle, unique per project, e.g. "dev-bot"']
  avatar_url varchar [null]

  // LLM Configuration
  llm_provider varchar [not null, note: 'LiteLLM provider prefix, e.g. anthropic, openai']
  llm_model varchar [not null, note: 'LiteLLM model name']
  llm_api_key_secret_ref varchar [not null, note: 'Reference to an entry in the secrets store (not the key itself)']
  llm_base_url varchar [null, note: 'Optional custom base URL (e.g. for Azure or local LLMs)']

  // Agent Behaviour
  system_prompt text [not null, default: '']

  // Capabilities
  max_iterations integer [not null, default: 50, note: 'Hard cap on agent reasoning steps per conversation']
  timeout_minutes integer [not null, default: 30, note: 'Wall-clock timeout for a single conversation']

  created_by uuid [ref: > users.id]
  created_at timestamp [not null]
  updated_at timestamp [not null]
  deleted_at timestamp [null]

  indexes {
    (project_id, handle) [unique, note: 'Partial unique: WHERE deleted_at IS NULL']
  }
}
```

### `agent_mcp_servers`

Custom MCP server configurations attached to an agent. Each row is one entry in the `mcpServers` map.

```dbml
Table agent_mcp_servers {
  id uuid [primary key]
  agent_id uuid [not null, ref: > agents.id]
  server_name varchar [not null, note: 'Key in mcpServers map, e.g. "fetch", "repomix"']
  transport varchar [not null, note: 'stdio | sse | http']
  command varchar [null, note: 'Binary to execute for stdio transport, e.g. uvx']
  args jsonb [not null, default: '[]', note: 'Array of arguments for stdio transport']
  url varchar [null, note: 'Server URL for sse/http transport']
  env jsonb [not null, default: '{}', note: 'Extra environment variables injected into the MCP server process']
  is_enabled boolean [not null, default: true]
  created_at timestamp [not null]
  updated_at timestamp [not null]

  indexes {
    (agent_id, server_name) [unique]
  }
}
```

### `agent_skills`

Skills associated with an agent. Skills are stored as the full `SKILL.md` content in the database and materialized into the container workspace at conversation start.

```dbml
Table agent_skills {
  id uuid [primary key]
  agent_id uuid [not null, ref: > agents.id]
  skill_name varchar [not null, note: 'Unique skill identifier for this agent']
  skill_source varchar [not null, note: 'inline | marketplace | github_url']
  skill_content text [not null, note: 'Full SKILL.md content for inline skills']
  source_url varchar [null, note: 'GitHub URL or marketplace ID for non-inline skills']
  triggers jsonb [not null, default: '[]', note: 'Array of keyword trigger strings']
  is_enabled boolean [not null, default: true]
  created_at timestamp [not null]
  updated_at timestamp [not null]

  indexes {
    (agent_id, skill_name) [unique]
  }
}
```

### `agent_conversations`

Tracks each OpenHands conversation session. One row per trigger invocation.

```dbml
Table agent_conversations {
  id uuid [primary key, note: 'Also used as the OpenHands conversation_id for SDK persistence']
  agent_id uuid [not null, ref: > agents.id]
  project_id uuid [not null, ref: > projects.id]

  // Trigger context
  trigger_type varchar [not null, note: 'task_assigned | comment_mention | chat_message']
  task_id uuid [null, ref: > tasks.id]
  comment_id uuid [null, note: 'task_activities row id for the triggering comment']
  chat_session_id uuid [null, ref: > agent_chat_sessions.id]
  triggered_by_member_id uuid [not null, ref: > project_members.id]

  // Execution state
  status varchar [not null, default: 'queued', note: 'queued | running | paused | finished | failed | stopped']
  container_id varchar [null, note: 'Docker container ID while running']
  host_port integer [null, note: 'Mapped port for the agent-server HTTP endpoint while running']
  iteration_count integer [not null, default: 0]
  error_message text [null]

  // Repository context
  repo_plugin_id uuid [null, note: 'Plugin instance that provided the clone URL']
  repo_clone_url varchar [null]
  branch_name varchar [null, note: 'Branch created/used by the agent']
  pr_url varchar [null, note: 'PR URL if agent created a pull request']

  // Persistence
  persistence_dir varchar [null, note: 'Path inside the ai-agent service filesystem where conversation state is persisted']

  started_at timestamp [null]
  finished_at timestamp [null]
  created_at timestamp [not null]
  updated_at timestamp [not null]
}
```

### `agent_conversation_events`

Individual events emitted by the OpenHands SDK during a conversation. Used for history and real-time monitoring.

```dbml
Table agent_conversation_events {
  id uuid [primary key]
  conversation_id uuid [not null, ref: > agent_conversations.id]
  event_index integer [not null, note: 'Sequential index within the conversation (0-based)']
  event_type varchar [not null, note: 'OpenHands SDK event type: MessageAction | CmdRunAction | FileEditAction | AgentFinishAction | CmdOutputObservation | etc.']
  event_source varchar [not null, note: 'agent | user | system']
  payload jsonb [not null, note: 'Full OpenHands event JSON payload']
  created_at timestamp [not null]

  indexes {
    (conversation_id, event_index) [unique]
  }
}
```

### `agent_chat_sessions`

Persistent chat sessions between a user and an agent. Each session accumulates messages and can be resumed.

```dbml
Table agent_chat_sessions {
  id uuid [primary key]
  agent_id uuid [not null, ref: > agents.id]
  project_id uuid [not null, ref: > projects.id]
  member_id uuid [not null, ref: > project_members.id, note: 'The human member chatting with the agent']
  title varchar [null, note: 'Auto-generated or user-set session title']
  last_message_at timestamp [null]
  created_at timestamp [not null]
  updated_at timestamp [not null]
}
```

---

## Modified Tables

### `project_members` — add `member_type` and `agent_id`

```sql
-- Add member_type discriminator
ALTER TABLE project_members
  ADD COLUMN member_type VARCHAR NOT NULL DEFAULT 'human'
  CHECK (member_type IN ('human', 'agent'));

-- Optional link to agents table (NULL for human members)
ALTER TABLE project_members
  ADD COLUMN agent_id UUID NULL REFERENCES agents(id);

-- Constraint: human members must have user_id, agent members must have agent_id
ALTER TABLE project_members
  ADD CONSTRAINT ck_pm_member_type_ref
  CHECK (
    (member_type = 'human' AND user_id IS NOT NULL AND agent_id IS NULL)
    OR
    (member_type = 'agent' AND agent_id IS NOT NULL AND user_id IS NULL)
  );

-- Ensure unique index covers both types
-- Existing (project_id, user_id) unique index remains for human members
-- New partial index for agent members:
CREATE UNIQUE INDEX uq_pm_project_agent
  ON project_members (project_id, agent_id)
  WHERE deleted_at IS NULL AND member_type = 'agent';
```

---

## Schema Relationships (DBML)

```dbml
// Agents belong to a project and are exposed as project members
Ref: agents.project_id > projects.id
Ref: project_members.agent_id > agents.id [note: 'null for human members']

// Agent configuration
Ref: agent_mcp_servers.agent_id > agents.id [delete: cascade]
Ref: agent_skills.agent_id > agents.id [delete: cascade]

// Conversations reference the agent, project, task, and triggering member
Ref: agent_conversations.agent_id > agents.id
Ref: agent_conversations.project_id > projects.id
Ref: agent_conversations.task_id > tasks.id
Ref: agent_conversations.triggered_by_member_id > project_members.id
Ref: agent_conversations.chat_session_id > agent_chat_sessions.id

// Events belong to a conversation
Ref: agent_conversation_events.conversation_id > agent_conversations.id [delete: cascade]

// Chat sessions tie an agent to a human member
Ref: agent_chat_sessions.agent_id > agents.id
Ref: agent_chat_sessions.member_id > project_members.id
```

---

## Index Strategy

| Table | Key indexes |
|---|---|
| `agents` | `(project_id, handle)` partial unique (WHERE deleted_at IS NULL) |
| `agent_conversations` | `(agent_id, status)`, `(task_id)`, `(chat_session_id)` |
| `agent_conversation_events` | `(conversation_id, event_index)` unique, `(conversation_id, event_type)` |
| `agent_chat_sessions` | `(agent_id, member_id)`, `(project_id, member_id)` |
