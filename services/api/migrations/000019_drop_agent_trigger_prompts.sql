-- Drop the per-trigger system prompt columns on agents.
-- Superseded by the ai-agent service's default skill set + fixed
-- trigger-context skills (services/ai-agent/src/skills/, src/agent/trigger_skills.py) —
-- action-specific behavior is no longer stored as free-text prompts per agent.

ALTER TABLE agents
    DROP COLUMN IF EXISTS task_trigger_prompt,
    DROP COLUMN IF EXISTS doc_comment_trigger_prompt,
    DROP COLUMN IF EXISTS chat_trigger_prompt,
    DROP COLUMN IF EXISTS description_write_trigger_prompt;
