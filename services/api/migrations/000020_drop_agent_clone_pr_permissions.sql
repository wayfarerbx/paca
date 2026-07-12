-- 000020_drop_agent_clone_pr_permissions.sql
-- Removes the per-agent can_clone_repos and can_create_prs permission columns.
-- All agents may now clone linked repositories and create PRs when a GitHub
-- account is configured — these are runtime capabilities, not per-agent toggles.

BEGIN;

ALTER TABLE agents DROP COLUMN IF EXISTS can_clone_repos;
ALTER TABLE agents DROP COLUMN IF EXISTS can_create_prs;

COMMIT;
