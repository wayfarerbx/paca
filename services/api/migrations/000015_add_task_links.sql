-- 000015_add_task_links.sql
-- Introduces a task_links join table that stores directional relationships
-- between tasks (blocks/relates_to/duplicates).  Inverse relationships
-- (is_blocked_by, is_duplicated_by) are computed at read time.

BEGIN;

CREATE TABLE IF NOT EXISTS task_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    target_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    link_type      VARCHAR(50) NOT NULL
                       CHECK (link_type IN ('blocks', 'relates_to', 'duplicates')),
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_task_links UNIQUE (source_task_id, target_task_id, link_type),
    CONSTRAINT no_self_link    CHECK  (source_task_id <> target_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_links_source ON task_links (source_task_id);
CREATE INDEX IF NOT EXISTS idx_task_links_target ON task_links (target_task_id);

COMMIT;
