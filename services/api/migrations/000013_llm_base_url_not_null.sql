-- Backfill NULL llm_base_url rows and make the column NOT NULL.
-- Previously the column was TEXT (nullable); now it is required in the domain layer.

UPDATE agents SET llm_base_url = '' WHERE llm_base_url IS NULL;

ALTER TABLE agents ALTER COLUMN llm_base_url SET NOT NULL;
ALTER TABLE agents ALTER COLUMN llm_base_url SET DEFAULT '';
