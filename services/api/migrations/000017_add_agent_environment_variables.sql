-- 000017_add_agent_environment_variables.sql
-- Adds per-agent secret environment variables, encrypted at rest, injected
-- into the agent's sandbox container at run time (see issues #240, #241).

BEGIN;

CREATE TABLE IF NOT EXISTS agent_environment_variables (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID        NOT NULL,
    key             TEXT        NOT NULL,
    encrypted_value TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agent_environment_variables_agent
        FOREIGN KEY (agent_id)
        REFERENCES agents(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_agent_environment_variables_key
    ON agent_environment_variables (agent_id, key);

COMMIT;
