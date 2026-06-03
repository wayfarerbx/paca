-- Add git committer identity columns to agents so each agent can commit
-- under a configurable name and email (defaults to the shared paca-agent bot).
ALTER TABLE agents
    ADD COLUMN IF NOT EXISTS git_committer_name  TEXT NOT NULL DEFAULT 'paca-agent',
    ADD COLUMN IF NOT EXISTS git_committer_email TEXT NOT NULL DEFAULT '280579135+paca-agent@users.noreply.github.com';
