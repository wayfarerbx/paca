# Database Schema

Interactive diagram: [https://dbdiagram.io/d/Paca-69c212ae78c6c4bc7a4fc190](https://dbdiagram.io/d/Paca-69c212ae78c6c4bc7a4fc190)

> **Note:** The DBML diagram above may lag behind the latest migrations. The authoritative source is `services/api/migrations/`. The schema below reflects the current migration state.

## Current Migration State

| File | Purpose |
|---|---|
| `000001_init.sql` | Full consolidated schema: `global_roles`, `users`, projects, project roles/members, task configuration (`task_types`, `task_statuses`), `sprints`, `sprint_views` (with `view_type`, `config`, `position`), `view_task_positions` (manual task order), `custom_field_definitions`, `tasks` (with `start_date`, `due_date`, `tags`), `task_attachments`, `task_checklists`, `task_checklist_items`, seed data. On project creation the seed inserts three user-manageable task types (Bug, Story, Task — where Task is `is_default = true`) and two system task types (Epic, Subtask — where `is_system = true`). The API now seeds one backlog Table view with `config.column_by = "sprint"` and non-system task types, plus one timeline Roadmap view filtered to Epics; sprint creation seeds Board and Table views scoped to that sprint. |
| `000002_add_view_context.sql` | Adds `view_context TEXT NOT NULL` to `sprint_views` with a `CHECK` constraint (`'sprint'\|'backlog'\|'timeline'`). Replaces the previous `is_timeline` boolean approach. Back-fills existing sprint rows to `'sprint'` and project-level (backlog) rows to `'backlog'`. Adds a partial index `idx_sprint_views_context` on `(project_id, view_context) WHERE sprint_id IS NULL` to speed up project-level view lookups. |

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

  indexes {
    (project_id, user_id) [unique]
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
  task_type_id uuid
  status_id uuid
  sprint_id uuid
  parent_task_id uuid [null]
  title varchar
  description text
  importance integer [not null, default: 0, note: 'unsigned; higher = more important']
  assignee_id uuid
  reporter_id uuid
  custom_fields jsonb
  start_date date [null]
  due_date date [null]
  tags jsonb [not null, default: '[]']
  created_at timestamp
  updated_at timestamp
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
  view_context varchar [not null, note: 'Integration discriminator: sprint | backlog | timeline. sprint rows always have sprint_id set; backlog and timeline rows have sprint_id = null.']
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

Table documents {
  id uuid [primary key]
  project_id uuid
  title varchar
  content text
  created_by uuid
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
  task_id uuid
  member_id uuid
  activity_type varchar
  content text
  created_at timestamp
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

Ref: users.id < projects.created_by
Ref: project_members.id < documents.created_by
Ref: project_members.id < time_logs.member_id
Ref: project_members.id < task_activities.member_id
Ref: project_members.id < tasks.assignee_id
Ref: project_members.id < tasks.reporter_id
Ref: tasks.id < task_attachments.task_id
Ref: project_members.id < task_attachments.uploaded_by
Ref: sprints.id < sprint_views.sprint_id
Ref: sprint_views.id < view_task_positions.view_id
Ref: tasks.id < view_task_positions.task_id
Ref: tasks.id < task_checklists.task_id
Ref: project_members.id < task_checklists.created_by
Ref: task_checklists.id < task_checklist_items.checklist_id
Ref: project_members.id < task_checklist_items.checked_by
Ref: project_members.id < task_checklist_items.assignee_id
Ref: project_members.id < task_checklist_items.created_by
```
