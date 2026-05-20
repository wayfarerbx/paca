-- 000008_add_ai_agents.sql
-- Adds AI Agent support: agent types, agents, skills, MCP servers,
-- conversations, conversation events, chat sessions, and modifications
-- to project_members to support agent members.

BEGIN;

-- -------------------------------------------------------------------------
-- AGENT TYPES
-- Reusable templates that pre-fill LLM, skills, and system prompt.
-- Global built-ins have project_id = NULL; project-scoped types have project_id set.
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agent_types (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id            UUID,
    name                  TEXT        NOT NULL,
    description           TEXT        NOT NULL DEFAULT '',
    slug                  TEXT        NOT NULL,
    default_llm_provider  TEXT        NOT NULL,
    default_llm_model     TEXT        NOT NULL,
    default_system_prompt TEXT        NOT NULL DEFAULT '',
    is_builtin            BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_types_project
        FOREIGN KEY (project_id)
        REFERENCES projects(id)
        ON DELETE CASCADE
);

-- Global built-in types: slug unique where project_id IS NULL
CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_types_global_slug
    ON agent_types (slug)
    WHERE project_id IS NULL;

-- Project-scoped types: (project_id, slug) unique
CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_types_project_slug
    ON agent_types (project_id, slug)
    WHERE project_id IS NOT NULL;

-- -------------------------------------------------------------------------
-- AGENTS
-- AI agents belonging to a project.
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agents (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id            UUID        NOT NULL,
    agent_type_id         UUID        NOT NULL,
    name                  TEXT        NOT NULL,
    handle                TEXT        NOT NULL,
    avatar_url            TEXT,
    llm_provider          TEXT        NOT NULL,
    llm_model             TEXT        NOT NULL,
    llm_api_key_secret    TEXT        NOT NULL,
    llm_base_url          TEXT,
    system_prompt         TEXT        NOT NULL DEFAULT '',
    can_clone_repos       BOOLEAN     NOT NULL DEFAULT TRUE,
    can_create_prs        BOOLEAN     NOT NULL DEFAULT TRUE,
    max_iterations        INTEGER     NOT NULL DEFAULT 50,
    timeout_minutes       INTEGER     NOT NULL DEFAULT 30,
    created_by            UUID,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,
    CONSTRAINT fk_agents_project
        FOREIGN KEY (project_id)
        REFERENCES projects(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_agents_type
        FOREIGN KEY (agent_type_id)
        REFERENCES agent_types(id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_agents_created_by
        FOREIGN KEY (created_by)
        REFERENCES users(id)
        ON DELETE SET NULL
);

-- Unique handle per project (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_agents_project_handle
    ON agents (project_id, handle)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agents_project_id ON agents (project_id);
CREATE INDEX IF NOT EXISTS idx_agents_deleted_at ON agents (deleted_at) WHERE deleted_at IS NOT NULL;

-- -------------------------------------------------------------------------
-- AGENT MCP SERVERS
-- Custom MCP server configurations per agent.
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agent_mcp_servers (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id    UUID        NOT NULL,
    server_name TEXT        NOT NULL,
    transport   TEXT        NOT NULL CHECK (transport IN ('stdio', 'sse', 'http')),
    command     TEXT,
    args        JSONB       NOT NULL DEFAULT '[]'::jsonb,
    url         TEXT,
    env         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    is_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_mcp_servers_agent
        FOREIGN KEY (agent_id)
        REFERENCES agents(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_mcp_servers_name
    ON agent_mcp_servers (agent_id, server_name);

-- -------------------------------------------------------------------------
-- AGENT SKILLS
-- Skills stored as SKILL.md content; materialized into containers at runtime.
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agent_skills (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id      UUID        NOT NULL,
    skill_name    TEXT        NOT NULL,
    skill_source  TEXT        NOT NULL CHECK (skill_source IN ('inline', 'marketplace', 'github_url')),
    skill_content TEXT        NOT NULL DEFAULT '',
    source_url    TEXT,
    triggers      JSONB       NOT NULL DEFAULT '[]'::jsonb,
    is_enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_skills_agent
        FOREIGN KEY (agent_id)
        REFERENCES agents(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_skills_name
    ON agent_skills (agent_id, skill_name);

-- -------------------------------------------------------------------------
-- AGENT CHAT SESSIONS
-- Persistent chat sessions between a user and an agent.
-- Declared before agent_conversations because conversations reference it.
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agent_chat_sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID        NOT NULL,
    project_id      UUID        NOT NULL,
    member_id       UUID        NOT NULL,
    title           TEXT,
    last_message_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_chat_sessions_agent
        FOREIGN KEY (agent_id)
        REFERENCES agents(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_agent_chat_sessions_project
        FOREIGN KEY (project_id)
        REFERENCES projects(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_chat_sessions_agent_member
    ON agent_chat_sessions (agent_id, member_id);
CREATE INDEX IF NOT EXISTS idx_agent_chat_sessions_project_member
    ON agent_chat_sessions (project_id, member_id);

-- -------------------------------------------------------------------------
-- AGENT CONVERSATIONS
-- Tracks each OpenHands conversation session (one per trigger invocation).
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agent_conversations (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id                 UUID        NOT NULL,
    project_id               UUID        NOT NULL,
    trigger_type             TEXT        NOT NULL CHECK (trigger_type IN ('task_assigned', 'comment_mention', 'chat_message')),
    task_id                  UUID,
    comment_id               UUID,
    chat_session_id          UUID,
    triggered_by_member_id   UUID        NOT NULL,
    status                   TEXT        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'paused', 'finished', 'failed', 'stopped')),
    container_id             TEXT,
    host_port                INTEGER,
    iteration_count          INTEGER     NOT NULL DEFAULT 0,
    error_message            TEXT,
    repo_plugin_id           UUID,
    repo_clone_url           TEXT,
    branch_name              TEXT,
    pr_url                   TEXT,
    persistence_dir          TEXT,
    started_at               TIMESTAMPTZ,
    finished_at              TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_conversations_agent
        FOREIGN KEY (agent_id)
        REFERENCES agents(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_agent_conversations_project
        FOREIGN KEY (project_id)
        REFERENCES projects(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_agent_conversations_task
        FOREIGN KEY (task_id)
        REFERENCES tasks(id)
        ON DELETE SET NULL,
    CONSTRAINT fk_agent_conversations_chat_session
        FOREIGN KEY (chat_session_id)
        REFERENCES agent_chat_sessions(id)
        ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_conversations_agent_status
    ON agent_conversations (agent_id, status);
CREATE INDEX IF NOT EXISTS idx_agent_conversations_task_id
    ON agent_conversations (task_id) WHERE task_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_agent_conversations_chat_session
    ON agent_conversations (chat_session_id) WHERE chat_session_id IS NOT NULL;

-- -------------------------------------------------------------------------
-- AGENT CONVERSATION EVENTS
-- Individual events emitted by the OpenHands SDK during a conversation.
-- -------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS agent_conversation_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL,
    event_index     INTEGER     NOT NULL,
    event_type      TEXT        NOT NULL,
    event_source    TEXT        NOT NULL CHECK (event_source IN ('agent', 'user', 'system')),
    payload         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_conversation_events_conversation
        FOREIGN KEY (conversation_id)
        REFERENCES agent_conversations(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_conversation_events_index
    ON agent_conversation_events (conversation_id, event_index);
CREATE INDEX IF NOT EXISTS idx_agent_conversation_events_type
    ON agent_conversation_events (conversation_id, event_type);

-- -------------------------------------------------------------------------
-- MODIFY project_members: add member_type and agent_id
-- -------------------------------------------------------------------------

-- Make user_id nullable (agents don't have a user_id)
ALTER TABLE project_members
    ALTER COLUMN user_id DROP NOT NULL;

-- Add member type discriminator (default 'human' for existing rows)
ALTER TABLE project_members
    ADD COLUMN IF NOT EXISTS member_type TEXT NOT NULL DEFAULT 'human'
    CHECK (member_type IN ('human', 'agent'));

-- Add optional link to the agents table
ALTER TABLE project_members
    ADD COLUMN IF NOT EXISTS agent_id UUID
    REFERENCES agents(id)
    ON DELETE CASCADE;

-- Ensure data integrity: human members have user_id; agent members have agent_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'ck_pm_member_type_ref'
    ) THEN
        ALTER TABLE project_members
            ADD CONSTRAINT ck_pm_member_type_ref CHECK (
                (member_type = 'human' AND user_id IS NOT NULL AND agent_id IS NULL)
                OR
                (member_type = 'agent' AND agent_id IS NOT NULL AND user_id IS NULL)
            );
    END IF;
END $$;

-- Partial unique index for agent members
CREATE UNIQUE INDEX IF NOT EXISTS uq_pm_project_agent
    ON project_members (project_id, agent_id)
    WHERE deleted_at IS NULL AND member_type = 'agent';

-- -------------------------------------------------------------------------
-- SEED: built-in agent types
-- -------------------------------------------------------------------------

INSERT INTO agent_types (id, name, description, slug, default_llm_provider, default_llm_model, default_system_prompt, is_builtin, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'PO Assistant',      'Product Owner assistant for backlog grooming, acceptance criteria, and prioritization.', 'po-assistant',   'anthropic', 'claude-sonnet-4-5-20250929', '', TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'Business Analyst',  'Requirements analysis, gap analysis, process modelling, and functional specifications.', 'ba',             'anthropic', 'claude-sonnet-4-5-20250929', '', TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'Developer',         'Coding, code review, PR creation, and bug fixing.',                                     'developer',      'anthropic', 'claude-sonnet-4-5-20250929', '', TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'Manual Tester',     'Test case design, exploratory testing documentation, and defect analysis.',             'manual-tester',  'anthropic', 'claude-sonnet-4-5-20250929', '', TRUE, NOW(), NOW())
ON CONFLICT DO NOTHING;

COMMIT;
