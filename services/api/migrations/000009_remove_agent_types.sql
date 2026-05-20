-- 000009_remove_agent_types.sql
-- Removes the agent_types table and agent_type_id column from agents.
-- Agent configuration (LLM, system prompt, skills) is managed directly on each
-- agent; preset defaults are provided by the frontend only.

BEGIN;

-- Drop FK constraint first
ALTER TABLE agents DROP CONSTRAINT IF EXISTS fk_agents_type;

-- Drop the agent_type_id column from agents
ALTER TABLE agents DROP COLUMN IF EXISTS agent_type_id;

-- Drop the agent_types table (and its indexes)
DROP INDEX IF EXISTS uq_agent_types_global_slug;
DROP INDEX IF EXISTS uq_agent_types_project_slug;
DROP TABLE IF EXISTS agent_types;

COMMIT;
