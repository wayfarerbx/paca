# Task Activity & Comments API

## Overview

Every task has an **activity log** — a unified, time-ordered stream of
system-generated change events and user-authored comments. Both are stored in
the same `task_activities` table and returned through the same list endpoint so
the UI can render them in a single chronological feed.

---

## Database Schema

```sql
CREATE TABLE task_activities (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
    activity_type TEXT        NOT NULL,
    content       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ  -- soft-delete for comments
);
```

Indexes: `(task_id, created_at)` for efficient chronological listing.

---

## Activity Types

| `activity_type`                | Trigger                                   |
|-------------------------------|-------------------------------------------|
| `task.created`                | Task created                              |
| `task.updated`                | Task fields changed                       |
| `task.deleted`                | Task soft-deleted                         |
| `task.attachment.added`       | Attachment linked                         |
| `task.attachment.removed`     | Attachment unlinked                       |
| `task.bdd_scenario.created`   | BDD scenario added                        |
| `task.bdd_scenario.updated`   | BDD scenario edited                       |
| `task.bdd_scenario.deleted`   | BDD scenario removed                      |
| `task.checklist.created`      | Checklist group created                   |
| `task.checklist.updated`      | Checklist group renamed                   |
| `task.checklist.deleted`      | Checklist group removed                   |
| `task.checklist_item.created` | Checklist item added                      |
| `task.checklist_item.updated` | Checklist item edited or checked          |
| `task.checklist_item.deleted` | Checklist item removed                    |
| `comment`                     | User comment (user-created)               |

---

## Content Shapes (JSONB)

### `task.created`
```json
{ "title": "Implement login", "task_number": 42 }
```

### `task.updated`
```json
{
  "changes": [
    { "field": "title",     "old": "Old title", "new": "New title" },
    { "field": "status_id", "old": "uuid-a",    "new": "uuid-b" }
  ]
}
```

Tracked fields: `title`, `status_id`, `assignee_id`, `reporter_id`,
`task_type_id`, `sprint_id`, `parent_task_id`, `importance`,
`start_date`, `due_date`.

### `task.deleted`
```json
{ "title": "Old task title", "task_number": 42 }
```

### `task.attachment.added` / `task.attachment.removed`
```json
{ "file_name": "screenshot.png", "file_size": 102400 }
```

### `task.bdd_scenario.created` / `task.bdd_scenario.deleted`
```json
{ "title": "User can log in" }
```

### `task.bdd_scenario.updated`
```json
{ "title": "User can log in", "changes": ["given", "when", "then"] }
```

### `task.checklist.created` / `task.checklist.deleted`
```json
{ "title": "Acceptance Criteria" }
```

### `task.checklist_item.created` / `task.checklist_item.deleted`
```json
{ "text": "AC item text" }
```

### `task.checklist_item.updated`
```json
{ "text": "AC item", "changes": [{"field":"is_checked","old":false,"new":true}] }
```

### `comment`
```json
{ "text": "Looks good. Ready to review." }
```

---

## API Endpoints

All endpoints are under `/api/v1/projects/:projectId/tasks/:taskId`.
Authentication is required. Project `tasks.read` permission is required for
reading; `tasks.write` is required for posting/editing/deleting comments.

### List Activities

```
GET /api/v1/projects/:projectId/tasks/:taskId/activities
```

Returns all activities and comments for a task, sorted oldest → newest.
Soft-deleted comments are excluded.

**Response:**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "uuid",
        "task_id": "uuid",
        "actor_id": "uuid",
        "actor_name": "Jane Doe",
        "actor_username": "jane",
        "activity_type": "task.created",
        "content": { "title": "Login screen" },
        "created_at": "2026-04-01T10:00:00Z",
        "updated_at": "2026-04-01T10:00:00Z"
      }
    ]
  }
}
```

### Post a Comment

```
POST /api/v1/projects/:projectId/tasks/:taskId/activities/comments
```

**Request body:**
```json
{ "text": "This looks good to me." }
```

**Response:** `201 Created` — the created activity entry.

### Edit a Comment

```
PATCH /api/v1/projects/:projectId/tasks/:taskId/activities/comments/:commentId
```

Only the original author may edit their comment.

**Request body:**
```json
{ "text": "Updated comment text." }
```

**Response:** `200 OK` — the updated activity entry.

### Delete a Comment

```
DELETE /api/v1/projects/:projectId/tasks/:taskId/activities/comments/:commentId
```

Only the original author may delete their comment. Performs a soft-delete
(sets `deleted_at`); the entry is excluded from normal list responses.

**Response:** `204 No Content`

---

## Valkey Event Stream

Every activity record that is created also publishes to the Valkey stream
`paca.analytics` (durable, for downstream consumers) and, for comment events,
to the `paca.events` Pub/Sub channel (for real-time fan-out via
`services/realtime`).

### Stream Entry Format

```
Stream: paca.analytics
Fields:
  type    = "task.created" | "task.updated" | ... | "comment"
  payload = <JSON string of the event payload below>
```

### Event Payloads

#### `task.created`, `task.updated`, `task.deleted`
```json
{
  "task_id":      "uuid",
  "project_id":   "uuid",
  "actor_id":     "uuid",
  "activity_id":  "uuid",
  "changes":      [...],  // only for task.updated
  "task_number":  42,     // only for task.created / task.deleted
  "title":        "..."   // only for task.created / task.deleted
}
```

#### `comment`
```json
{
  "task_id":     "uuid",
  "project_id":  "uuid",
  "actor_id":    "uuid",
  "activity_id": "uuid",
  "text":        "Comment text"
}
```

---

## Stream Consumer

A background goroutine (`internal/platform/messaging/consumer.go`) reads from
`paca.analytics` using `XREADGROUP`. Currently it logs all task-related events
and is the extension point for:

- Email / push notifications
- Webhook delivery
- Analytics aggregation

Consumer group name: `paca-api-workers`  
Consumer name: `paca-api-{hostname}`

---

## Actor Resolution

The authenticated user's `user_id` (from JWT `sub` claim) is embedded into the
Go `context.Context` by the `Authn` middleware via `middleware.WithActorID`.
Service methods read it with `middleware.ActorIDFromContext(ctx)`.

---

## Permission Model

| Action              | Required permission      |
|---------------------|--------------------------|
| List activities     | `tasks.read` (project)   |
| Post comment        | `tasks.write` (project)  |
| Edit own comment    | `tasks.write` (project)  |
| Delete own comment  | `tasks.write` (project)  |
