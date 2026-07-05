-- 000018_add_automation_workflows.sql
-- Introduces automation workflows: a project-scoped, draft/active/archived
-- dependency graph over existing tasks. Nodes just wrap a task (position on
-- the canvas). Edges are plain dependency links (source node -> target node)
-- with no per-edge configuration. Status->assignee rules live on the
-- WORKFLOW (not the node) — one shared list per workflow, applied uniformly
-- to whichever task/node currently has that status — and are reused by both
-- automation events. The workflow also has a status-transition chain
-- (workflow_status_transitions): for each project status, an optional
-- "next status." The workflow's done status is DERIVED from this chain —
-- whichever status has no next status configured — rather than being a
-- separate per-node field (see docs/architecture/automation-workflows.md
-- for full semantics).

BEGIN;

CREATE TABLE IF NOT EXISTS workflows (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'active', 'archived')),
    created_by  UUID REFERENCES project_members(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_workflows_project ON workflows (project_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS workflow_nodes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id     UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    pos_x           DOUBLE PRECISION NOT NULL DEFAULT 0,
    pos_y           DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workflow_nodes_task UNIQUE (workflow_id, task_id)
);

-- Superseded by workflow_status_transitions below (done status moved from a
-- per-node field to a workflow-level derived value) — dropped defensively
-- in case this migration already ran against this database before that
-- change.
ALTER TABLE workflow_nodes DROP COLUMN IF EXISTS done_status_id;

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow ON workflow_nodes (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_task ON workflow_nodes (task_id);

CREATE TABLE IF NOT EXISTS workflow_status_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id         UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status_id           UUID NOT NULL REFERENCES task_statuses(id) ON DELETE CASCADE,
    assignee_member_id  UUID NOT NULL REFERENCES project_members(id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workflow_status_rules UNIQUE (workflow_id, status_id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_status_rules_workflow ON workflow_status_rules (workflow_id);

-- workflow_status_transitions is the "status workflow": for each project
-- status, an optional "next status" to move a task to once work at that
-- status is done. This lets an AI-agent assignee be told exactly what
-- status to set next instead of guessing. next_status_id = NULL marks that
-- status as terminal ("done") for this workflow; the workflow's single done
-- status is DERIVED as whichever status has no next status configured (see
-- workflowdom.DeriveDoneStatusID) — activation requires exactly one such
-- entry. Auto-seeded at workflow-creation time from the project's task
-- statuses ordered by position, chained sequentially.
CREATE TABLE IF NOT EXISTS workflow_status_transitions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id      UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status_id        UUID NOT NULL REFERENCES task_statuses(id) ON DELETE CASCADE,
    next_status_id   UUID REFERENCES task_statuses(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workflow_status_transitions UNIQUE (workflow_id, status_id),
    CONSTRAINT no_self_transition CHECK (next_status_id IS NULL OR next_status_id <> status_id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_status_transitions_workflow ON workflow_status_transitions (workflow_id);

CREATE TABLE IF NOT EXISTS workflow_edges (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id      UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    source_node_id   UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
    target_node_id   UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workflow_edges UNIQUE (source_node_id, target_node_id),
    CONSTRAINT no_self_edge CHECK (source_node_id <> target_node_id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_edges_workflow ON workflow_edges (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_edges_source ON workflow_edges (source_node_id);
CREATE INDEX IF NOT EXISTS idx_workflow_edges_target ON workflow_edges (target_node_id);

-- Grant the new workflows.read/workflows.write permissions to existing
-- project_roles rows for the built-in role names:
--   - PROJECT_OWNER / PROJECT_MANAGER / PROJECT_MEMBER / PROJECT_VIEWER are
--     the global role *templates* (project_id IS NULL) from
--     authz.DefaultProjectRoles() — kept in sync here defensively, though
--     bootstrap's seedDefaultProjectRoleTemplates already re-syncs them
--     on every startup.
--   - Admin / Editor / Viewer are the actual per-project roles seeded by
--     projectsvc.CreateProject for every project; that codepath has its own
--     hardcoded permission maps (not sourced from DefaultProjectRoles()) and
--     is never re-synced after project creation, so those rows need this
--     backfill too.
--
-- The JSONB `||` merge only adds/overwrites the listed keys, so any other
-- permission a project admin already customised on these rows is preserved.
-- Role renames after creation mean this name-based match isn't 100%
-- precise, but it's the same signal the application code itself uses to
-- identify these roles (see RoleName == "Admin" in projectsvc.CreateProject).

UPDATE project_roles
SET permissions = permissions || '{"workflows.*": true}'::jsonb,
    updated_at = NOW()
WHERE role_name IN ('PROJECT_OWNER', 'PROJECT_MANAGER', 'Admin');

UPDATE project_roles
SET permissions = permissions || '{"workflows.read": true, "workflows.write": true}'::jsonb,
    updated_at = NOW()
WHERE role_name IN ('PROJECT_MEMBER', 'Editor');

UPDATE project_roles
SET permissions = permissions || '{"workflows.read": true}'::jsonb,
    updated_at = NOW()
WHERE role_name IN ('PROJECT_VIEWER', 'Viewer');

COMMIT;
