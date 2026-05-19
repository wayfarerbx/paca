# Database Schema

Interactive diagram: [https://dbdiagram.io/d/Paca-69c212ae78c6c4bc7a4fc190](https://dbdiagram.io/d/Paca-69c212ae78c6c4bc7a4fc190)

> **Note:** The DBML diagram above may lag behind the latest migrations. The authoritative source is `services/api/migrations/`. The schema below reflects the current migration state.

## Current Migration State

| File | Purpose |
|---|---|
| `000001_init.sql` | Full consolidated schema: `global_roles`, `users`, projects (with `task_id_prefix` for human-readable task IDs), project roles/members, task configuration (`task_types`, `task_statuses`), `sprints`, `sprint_views` (with `view_context` field), `view_task_positions`, `custom_field_definitions`, `task_counters` (tracks per-project sequential task numbers), `tasks` (with `story_points`), `files` (central file-metadata registry), `task_attachments` (now links to `files` table), `task_activities`, `doc_folders` (hierarchical folders with `parent_id` self-reference, `position`, `created_by`), `documents` (BlockNote `content` JSONB, `folder_id`, `position`, soft-delete via `deleted_at`, `created_by`/`updated_by` referencing `project_members`), `doc_snapshots` (point-in-time content copies for diff/history, `snapshot_number` auto-incremented per document via a trigger), `doc_activities` (audit log + comments), `notifications` (task-assignment and @mention notifications with `recipient_user_id`, `actor_member_id`, `type`, `read_at`), `api_keys` (user API keys for programmatic authentication), `plugins` (core plugin registry), `plugin_extension_settings` (system-wide plugin extension point configuration), and seed data. Note: `task_checklists` and `task_checklist_items` tables were originally here but have been migrated to the `com.paca.checklist` plugin via migration 000005. |
| `000002_add_story_points.sql` | Adds `story_points` INTEGER column to the tasks table (nullable, >= 0). |
| `000003_add_project_is_public.sql` | Adds `is_public` BOOLEAN flag to projects table for anonymous read access. |
| `000004_add_plugins.sql` | Adds `plugins` and `plugin_extension_settings` tables for the plugin system. |
| `000005_migrate_checklists_to_plugin.sql` | Removes legacy `task_checklists` and `task_checklist_items` tables from public schema (migrated to com.paca.checklist plugin). |
| `000006_add_plugin_view_type.sql` | Extends sprint_views.view_type CHECK constraint to allow 'plugin' as a valid view type. |
| `000007_remove_github_tables.sql` | Removes GitHub integration tables (`github_integrations`, `github_repositories`, `github_pull_requests`, `github_task_pr_links`, `github_task_branches`) as they have been migrated to plugins. |

## Schema (DBML)

```dbml
// --- USER & GLOBAL ROLE MANAGEMENT ---
Table users {
  id uuid [primary key]
  username varchar [unique, not null]
  password_hash varchar [not null]
  full_name varchar
  role_id uuid [ref: > global_roles.id, not null]
  must_change_password boolean [not null, default: false]
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp [null]
}

Table global_roles {
  id uuid [primary key]
  name varchar [unique, not null]
  permissions jsonb [not null]
  created_at timestamp
  updated_at timestamp
}

// --- PROJECT & TEAM MANAGEMENT ---
Table projects {
  id uuid [primary key]
  name varchar [not null]
  description text [not null, default: '']
  task_id_prefix varchar [not null, default: '', note: 'Short uppercase alphanumeric tag prepended to task_number to form human-readable task ID, e.g. "PACA" → "PACA-1"']
  settings jsonb [not null, default: '{}']
  is_public boolean [not null, default: false, note: 'Allows anonymous read access when true']
  created_by uuid [ref: > users.id]
  created_at timestamp
}

Table project_roles {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  role_name varchar
  permissions jsonb
}

Table project_members {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  user_id uuid [ref: > users.id]
  project_role_id uuid [ref: > project_roles.id]
  created_at timestamp [not null]
  deleted_at timestamp [null, note: 'Soft-delete timestamp. Non-null rows are excluded from active-member queries. Re-adding a removed member restores the existing row (sets deleted_at = NULL) rather than inserting a new one.']

  indexes {
    (project_id, user_id) [unique, note: 'Partial unique index: WHERE deleted_at IS NULL']
  }
}

// --- TASK CONFIGURATION ---
Table task_types {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  name varchar
  icon varchar
  color varchar
  description text
  is_default boolean [not null, default: false, note: 'True for the single default type seeded at project creation (Task). Only one type per project should have is_default = true.']
  is_system boolean [not null, default: false, note: 'True for system-managed types (Epic, Subtask). System types are seeded at project creation and cannot be created, edited, or deleted by users. They are displayed in a read-only section on the Task Types settings page and are excluded from inline task creation type pickers unless explicitly supported.']
}

Table task_statuses {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  name varchar
  color varchar
  position integer
  category varchar [note: 'backlog | refinement | ready | todo | inprogress | done']
  is_default boolean [not null, default: false, note: 'True for the single default status seeded at project creation. Only one status per project should have is_default = true.']
}

// --- TASK COUNTERS ---
Table task_counters {
  project_id uuid [primary key, ref: > projects.id, note: 'Tracks the per-project sequential task number so that every task within a project gets a human-readable, monotonically increasing identifier.']
  last_value bigint [not null, default: 0, note: 'The last task number assigned to a task in this project']
}

// --- TASKS ---
Table tasks {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  task_number bigint [not null, default: 0, note: 'Project-scoped sequential ID (1, 2, 3, …) assigned at creation and never reused. Unique per project via uq_tasks_project_task_number constraint.']
  task_type_id uuid [ref: > task_types.id]
  status_id uuid [ref: > task_statuses.id]
  sprint_id uuid [ref: > sprints.id]
  parent_task_id uuid [null, ref: > tasks.id]
  title varchar [not null]
  description jsonb [null, note: 'BlockNote rich-text document stored as a JSON array of block objects. null means no description. Each block object follows the BlockNote schema: { id, type, props, content, children }.']
  importance integer [not null, default: 0, note: 'unsigned; higher = more important']
  story_points integer [null, note: 'Story point estimate; must be >= 0 if set']
  assignee_id uuid [ref: > project_members.id]
  reporter_id uuid [ref: > project_members.id]
  custom_fields jsonb [not null, default: '{}']
  start_date date [null]
  due_date date [null]
  tags jsonb [not null, default: '[]']
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp [null, note: 'Soft-delete timestamp. Non-null rows are excluded from normal queries.']
}

Table custom_field_definitions {
  id uuid [primary key]
  project_id uuid [not null, ref: > projects.id]
  field_key varchar [not null, note: 'Unique per project; immutable after creation']
  display_name varchar [not null]
  field_type varchar [not null, note: 'text | number | date | select | multi_select | boolean | url']
  options jsonb [null, note: 'Ordered list of option labels for select / multi_select types']
  is_required boolean [not null, default: false]
  created_at timestamp
  updated_at timestamp

  indexes {
    (project_id, field_key) [unique]
  }
}

// --- SPRINTS & VIEWS ---
Table sprints {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  name varchar
  start_date date
  end_date date
  goal text
  status varchar [note: 'planned | active | completed. Multiple sprints per project may be active simultaneously.']
}

Table sprint_views {
  id uuid [primary key]
  sprint_id uuid [null, ref: > sprints.id, note: 'null for project-level views (backlog, timeline); set for sprint views']
  project_id uuid [not null, ref: > projects.id]
  name varchar [not null]
  view_type varchar [not null, note: 'Layout: table | board | roadmap | plugin']
  view_context varchar [not null, note: 'Interaction discriminator: sprint | backlog | timeline. sprint rows always have sprint_id set; backlog and timeline rows have sprint_id = null.']
  position double [not null, default: 0, note: 'Zero-based tab order within the interaction; lower = further left in the tab bar. Updated on drag-to-reorder.']
  config jsonb [note: '''
    View display settings.  All keys are optional; unset keys fall back to
    per-project or system defaults.

    fields      array<string>  Ordered list of visible column names.
                               e.g. ["title","assignees","status","importance"]
    column_by   string         Field used to group board columns or table
                               groups.  e.g. "status" (default for board/sprint
                               views), "sprint" (default for product-backlog
                               Table view — groups tasks into sprint columns
                               plus an "Unassigned" column for tasks with no
                               sprint).
    swimlanes   string|null    Field used to create horizontal swimlane bands
                               across the view.  null = no swimlanes.
    sort_by     string         "manual" = user-defined drag order stored in
                               view_task_positions.  Any other value is a
                               field name used for automatic sort.
                               e.g. "importance", "created_at", "manual".
    field_sum   string         Aggregate shown in group/column headings.
                               "count" (default) = number of tasks.  Can be
                               any numeric custom field key.
    slice_by    string|null    Additional filter dimension applied to the
                               visible task set.  null = no slice.
    For plugin views: plugin_id, plugin_component are stored here.
  ''']
  created_at timestamp
  updated_at timestamp
}

Table view_task_positions {
  id uuid [primary key]
  view_id uuid [ref: > sprint_views.id]
  task_id uuid [ref: > tasks.id]
  position double [not null, default: 0, note: 'Zero-based index within its group_key; lower = higher in list']
  group_key varchar [null, note: 'Value of the column_by field for this task (e.g. status name, assignee id) or swimlane key.  null = ungrouped.']

  indexes {
    (view_id, task_id) [unique]
  }
}

// --- FEATURES & UTILITIES ---
// Note: task_checklists and task_checklist_items are owned by the com.paca.checklist plugin.

Table time_logs {
  id uuid [primary key]
  task_id uuid [ref: > tasks.id]
  member_id uuid [ref: > project_members.id]
  duration_minutes integer
  logged_date date
}

// --- FILES ---
Table files {
  id uuid [primary key]
  storage_key text [unique, not null, note: 'Key in the object-store (S3-compatible)']
  bucket text [not null, note: 'S3 bucket name']
  file_name text [not null]
  content_type text [not null, default: 'application/octet-stream']
  file_size bigint [not null, default: 0]
  upload_status text [not null, default: 'pending', note: 'pending | uploaded | failed']
  multipart_upload_id text [null, note: 'Non-null only while a multipart upload is in progress']
  uploaded_by uuid [ref: > users.id]
  created_at timestamp
  updated_at timestamp
}

Table task_attachments {
  id uuid [primary key]
  task_id uuid [not null, ref: > tasks.id]
  file_id uuid [not null, ref: > files.id]
  created_by uuid [ref: > users.id]
  created_at timestamp [not null]

  indexes {
    (task_id, file_id) [unique]
  }
}

// --- DOCUMENTATION ---
Table doc_folders {
  id         uuid [primary key]
  project_id uuid [not null, ref: > projects.id]
  parent_id  uuid [null, ref: > doc_folders.id, note: 'null = root; self-reference for nested folders']
  name       varchar [not null]
  position   integer [not null, default: 0, note: 'Zero-based order among siblings']
  created_by uuid [null, ref: > project_members.id]
  created_at timestamp
  updated_at timestamp
}

Table documents {
  id         uuid [primary key]
  project_id uuid [not null, ref: > projects.id]
  folder_id  uuid [null, ref: > doc_folders.id, note: 'null = root (no folder)']
  title      varchar [not null, default: 'Untitled']
  content    jsonb [null, note: 'BlockNote rich-text document stored as a JSON array of block objects. null means no content. Each block follows the BlockNote schema: { id, type, props, content, children }.']
  position   integer [not null, default: 0, note: 'Zero-based order within the same folder/root']
  created_by uuid [null, ref: > project_members.id]
  updated_by uuid [null, ref: > project_members.id]
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp [null, note: 'Soft-delete timestamp']
}

Table doc_snapshots {
  id              uuid [primary key]
  document_id     uuid [not null, ref: > documents.id]
  title           varchar [not null, note: 'Title at the time of the snapshot']
  content         jsonb [null, note: 'BlockNote content at the time of the snapshot']
  snapshot_number bigint [not null, default: 0, note: 'Monotonically increasing per document; set by trigger']
  created_by      uuid [null, ref: > project_members.id]
  created_at      timestamp
}

Table doc_activities {
  id            uuid [primary key]
  document_id   uuid [not null, ref: > documents.id]
  actor_id      uuid [null, ref: > project_members.id, note: 'NULL for system events or if the member was removed']
  activity_type varchar [not null, note: 'doc.created | doc.updated | doc.deleted | doc.moved | doc.folder.created | doc.folder.updated | doc.folder.deleted | comment']
  content       jsonb [not null, default: '{}', note: 'For doc.updated: [{field, old, new}]. For comment: {text}. For doc.moved: {from_folder_id, to_folder_id}.']
  created_at    timestamp
  updated_at    timestamp
  deleted_at    timestamp [null, note: 'Soft-delete for comments']
}

Table dashboards {
  id uuid [primary key]
  project_id uuid [ref: > projects.id]
  name varchar
  layout jsonb
}

Table task_activities {
  id uuid [primary key]
  task_id uuid [not null, ref: > tasks.id]
  actor_id uuid [null, ref: > project_members.id, note: 'References project_members(id). Resolved from the authenticated user UUID by the ActivityConsumer at consume-time using the task project_id. NULL for system events or if the member was removed before the stream message was processed.']
  activity_type varchar [not null]
  content jsonb [not null, default: '{}']
  created_at timestamp
  updated_at timestamp
  deleted_at timestamp [null, note: 'Soft-delete for comments']
}

// --- PLUGINS ---
Table plugins {
  id uuid [primary key]
  name text [unique, not null, note: 'reverse-DNS id, e.g. "com.paca.checklist"']
  version text [not null, default: '0.0.0', note: 'semver, e.g. "1.0.0"']
  manifest jsonb [not null, default: '{}', note: 'Full plugin.json contents (routes, extension points, event subscriptions, etc.)']
  enabled boolean [not null, default: true]
  installed_at timestamp
  updated_at timestamp
}

Table plugin_extension_settings {
  id uuid [primary key]
  plugin_id uuid [not null, ref: > plugins.id]
  extension_point text [not null, note: 'Extension point identifier, e.g. "task.detail.section"']
  settings jsonb [not null, default: '{}', note: 'System-wide ordering and visibility settings: {order, hidden}']
  updated_at timestamp

  indexes {
    (plugin_id, extension_point) [unique]
  }
}

// --- NOTIFICATIONS ---
Table notifications {
  id                uuid [primary key]
  recipient_user_id uuid [not null, ref: > users.id, note: 'The user who receives the notification']
  actor_member_id   uuid [null, ref: > project_members.id, note: 'The project member who triggered the notification']
  type              varchar [not null, note: 'assigned | mentioned']
  task_id           uuid [null, ref: > tasks.id]
  project_id        uuid [not null, ref: > projects.id]
  read_at           timestamp [null, note: 'When the notification was marked as read']
  created_at        timestamp
}

// --- NOTIFICATIONS ---
Table api_keys {
  id uuid [primary key]
  user_id uuid [not null, ref: > users.id]
  name text [not null]
  key_prefix text [not null, note: 'First 8 hex characters of the raw key, for display only']
  key_hash text [not null, unique, note: 'SHA-256 hex digest of the raw key used for lookup/validation']
  last_used_at timestamp [null]
  expires_at timestamp [null]
  created_at timestamp
  revoked_at timestamp [null]
}
```
