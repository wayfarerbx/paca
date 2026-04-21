# Database Schema

Interactive diagram: [https://dbdiagram.io/d/Paca-69c212ae78c6c4bc7a4fc190](https://dbdiagram.io/d/Paca-69c212ae78c6c4bc7a4fc190)

> **Note:** The DBML diagram above may lag behind the latest migrations. The authoritative source is `services/api/migrations/`. The schema below reflects the current migration state.

## Current Migration State

| File | Purpose |
|---|---|
| `000001_init.sql` | Full consolidated schema: `global_roles`, `users`, projects, project roles/members, task configuration (`task_types`, `task_statuses`), `sprints`, `sprint_views`, `view_task_positions`, `custom_field_definitions`, `tasks`, `task_attachments`, `task_checklists`, `task_checklist_items`, `bdd_scenarios`, `task_activities`, `doc_folders` (hierarchical folders with `parent_id` self-reference, `position`, `created_by`), `documents` (BlockNote `content` JSONB, `folder_id`, `position`, soft-delete via `deleted_at`, `created_by`/`updated_by` referencing `project_members`), `doc_snapshots` (point-in-time content copies for diff/history, `snapshot_number` auto-incremented per document via a trigger), `doc_activities` (audit log + comments), `notifications` (task-assignment and @mention notifications with `recipient_user_id`, `actor_member_id`, `type`, `read_at`), and seed data. |

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
  name varchar
  description text
  settings jsonb
  created_by uuid
  created_at timestamp
}

Table project_roles {
  id uuid [primary key]
  project_id uuid
  role_name varchar
  permissions jsonb
}

Table project_members {
  id uuid [primary key]
  project_id uuid
  user_id uuid
  project_role_id uuid
  created_at timestamp [not null]
  deleted_at timestamp [null, note: 'Soft-delete timestamp. Non-null rows are excluded from active-member queries. Re-adding a removed member restores the existing row (sets deleted_at = NULL) rather than inserting a new one.']

  indexes {
    (project_id, user_id) [unique, note: 'Partial unique index: WHERE deleted_at IS NULL']
  }
}

// --- TASK CONFIGURATION ---
Table task_types {
  id uuid [primary key]
  project_id uuid
  name varchar
  icon varchar
  color varchar
  description text
  is_default boolean [not null, default: false, note: 'True for the single default type seeded at project creation (Task). Only one type per project should have is_default = true.']
  is_system boolean [not null, default: false, note: 'True for system-managed types (Epic, Subtask). System types are seeded at project creation and cannot be created, edited, or deleted by users. They are displayed in a read-only section on the Task Types settings page and are excluded from inline task creation type pickers unless explicitly supported.']
}

Table task_statuses {
  id uuid [primary key]
  project_id uuid
  name varchar
  color varchar
  position integer
  category varchar // backlog, refinement, ready, todo, inprogress, done
}

// --- TASKS ---
Table tasks {
  id uuid [primary key]
  project_id uuid
  task_number bigint [not null, default: 0, note: 'Project-scoped sequential ID (1, 2, 3, …) assigned at creation and never reused. Unique per project via uq_tasks_project_task_number constraint.']
  task_type_id uuid
  status_id uuid
  sprint_id uuid
  parent_task_id uuid [null]
  title varchar
  description jsonb [null, note: 'BlockNote rich-text document stored as a JSON array of block objects. null means no description. Each block object follows the BlockNote schema: { id, type, props, content, children }.']
  importance integer [not null, default: 0, note: 'unsigned; higher = more important']
  assignee_id uuid
  reporter_id uuid
  custom_fields jsonb
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
  project_id uuid
  name varchar
  start_date date
  end_date date
  goal text
  status varchar [note: 'planned | active | completed. Multiple sprints per project may be active simultaneously.']
}

Table sprint_views {
  id uuid [primary key]
  sprint_id uuid [null, note: 'null for project-level views (backlog, timeline); set for sprint views']
  project_id uuid [not null, ref: > projects.id]
  name varchar
  view_type varchar [not null, note: 'Layout: table | board | roadmap']
  view_context varchar [not null, note: 'Interaction discriminator: sprint | backlog | timeline. sprint rows always have sprint_id set; backlog and timeline rows have sprint_id = null.']
  position integer [not null, default: 0, note: 'Zero-based tab order within the interaction; lower = further left in the tab bar. Updated on drag-to-reorder.']
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
  ''']
  created_at timestamp
  updated_at timestamp
}

Table view_task_positions {
  id uuid [primary key]
  view_id uuid
  task_id uuid
  position integer [not null, note: 'Zero-based index within its group_key; lower = higher in list']
  group_key varchar [null, note: 'Value of the column_by field for this task (e.g. status name, assignee id) or swimlane key.  null = ungrouped.']

  indexes {
    (view_id, task_id) [unique]
  }
}

// --- FEATURES & UTILITIES ---
Table bdd_scenarios {
  id uuid [primary key]
  task_id uuid
  title varchar
  given text
  when text
  then text
  created_at timestamp
}

Table time_logs {
  id uuid [primary key]
  task_id uuid
  member_id uuid
  duration_minutes integer
  logged_date date
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
  project_id uuid
  name varchar
  layout jsonb
}

Table task_attachments {
  id uuid [primary key]
  task_id uuid
  file_name text [not null]
  file_size bigint [not null]
  mime_type text [not null]
  storage_url text [not null]
  uploaded_by uuid [null]
  created_at timestamp
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

// --- TASK CHECKLISTS ---
Table task_checklists {
  id         uuid [primary key]
  task_id    uuid [not null, ref: > tasks.id]
  title      varchar [not null]
  position   integer [not null, default: 0, note: 'Zero-based order among checklists on the same task']
  created_by uuid [null, ref: > project_members.id]
  created_at timestamp
  updated_at timestamp
}

Table task_checklist_items {
  id           uuid [primary key]
  checklist_id uuid [not null, ref: > task_checklists.id]
  title        text [not null]
  is_checked   boolean [not null, default: false]
  checked_by   uuid [null, ref: > project_members.id, note: 'Who checked this item']
  checked_at   timestamp [null]
  assignee_id  uuid [null, ref: > project_members.id, note: 'Optional per-item owner']
  due_date     date [null]
  position     integer [not null, default: 0, note: 'Zero-based order within the checklist']
  created_by   uuid [null, ref: > project_members.id]
  created_at   timestamp
  updated_at   timestamp
}

// --- RELATIONSHIPS ---
Ref: projects.id < project_members.project_id
Ref: users.id < project_members.user_id
Ref: project_roles.id < project_members.project_role_id
Ref: projects.id < project_roles.project_id

Ref: projects.id < task_types.project_id
Ref: projects.id < task_statuses.project_id
Ref: task_types.id < tasks.task_type_id
Ref: task_statuses.id < tasks.status_id

Ref: projects.id < tasks.project_id
Ref: projects.id < sprints.project_id
Ref: sprints.id < tasks.sprint_id
Ref: tasks.id < tasks.parent_task_id
Ref: tasks.id < bdd_scenarios.task_id
Ref: tasks.id < time_logs.task_id
Ref: tasks.id < task_activities.task_id
Ref: projects.id < documents.project_id
Ref: projects.id < dashboards.project_id
Ref: projects.id < doc_folders.project_id
Ref: doc_folders.id < doc_folders.parent_id
Ref: doc_folders.id < documents.folder_id
Ref: documents.id < doc_snapshots.document_id
Ref: documents.id < doc_activities.document_id
Ref: project_members.id < doc_folders.created_by
Ref: project_members.id < documents.created_by
Ref: project_members.id < documents.updated_by
Ref: project_members.id < doc_snapshots.created_by
Ref: project_members.id < doc_activities.actor_id

Ref: users.id < projects.created_by
Ref: project_members.id < time_logs.member_id
Ref: project_members.id < task_activities.member_id
Ref: project_members.id < tasks.assignee_id
Ref: project_members.id < tasks.reporter_id
Ref: tasks.id < task_attachments.task_id
Ref: project_members.id < task_attachments.uploaded_by
Ref: sprints.id < sprint_views.sprint_id
Ref: sprint_views.id < view_task_positions.view_id
Ref: tasks.id < view_task_positions.task_id

Ref: users.id < notifications.recipient_user_id
Ref: project_members.id < notifications.actor_member_id
Ref: tasks.id < notifications.task_id
Ref: projects.id < notifications.project_id
```
