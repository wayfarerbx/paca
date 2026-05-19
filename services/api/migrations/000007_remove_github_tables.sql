-- 000007_remove_github_tables.sql
-- Removes GitHub integration tables as they have been migrated to plugins.
-- DROP … CASCADE removes dependent indexes and FK constraints automatically.
-- IF EXISTS makes this idempotent if already applied.

BEGIN;

DROP TABLE IF EXISTS public.github_task_branches CASCADE;
DROP TABLE IF EXISTS public.github_task_pr_links CASCADE;
DROP TABLE IF EXISTS public.github_pull_requests CASCADE;
DROP TABLE IF EXISTS public.github_repositories CASCADE;
DROP TABLE IF EXISTS public.github_integrations CASCADE;

COMMIT;