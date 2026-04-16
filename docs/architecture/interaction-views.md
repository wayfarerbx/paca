# Interaction Views Model

This document explains how sprint, product-backlog, and timeline views work after the consolidation to a single view system.

## Why this exists

Paca used to treat sprint, backlog, and timeline views as separate API families. That made the code harder to evolve and forced the web app to carry route-specific logic in many places.

The current design keeps the UI concepts distinct, but models them through one shared view resource and one shared task-list resource.

## Core model

A saved view always belongs to exactly one project and one of these contexts:

- `sprint`
- `backlog`
- `timeline`

Sprint views also carry a `sprint_id`. Backlog and timeline views are project-level views with no sprint assignment.

## Unified view API

Use one resource for all view CRUD:

```text
GET    /api/v1/projects/:projectId/views?context=sprint&sprint_id=:sprintId
GET    /api/v1/projects/:projectId/views?context=backlog
GET    /api/v1/projects/:projectId/views?context=timeline
POST   /api/v1/projects/:projectId/views?...same query pattern...
PATCH  /api/v1/projects/:projectId/views/:viewId
DELETE /api/v1/projects/:projectId/views/:viewId
PUT    /api/v1/projects/:projectId/views/positions?...same query pattern...
```

Manual task ordering is also shared:

```text
GET /api/v1/projects/:projectId/views/:viewId/task-positions
PUT /api/v1/projects/:projectId/views/:viewId/task-positions/:taskId
PUT /api/v1/projects/:projectId/views/:viewId/task-positions
```

## Unified task-list API

All interaction pages should fetch tasks through the same endpoint:

```text
GET /api/v1/projects/:projectId/tasks
```

Supported query parameters:

- `sprint_id=<uuid>` for one sprint
- `sprint_ids=<uuid,uuid>` for multi-sprint saved views
- `sprint_id=null` when a caller explicitly wants only unscheduled backlog items
- `status_id=<uuid>` or `status_ids=<uuid,uuid>`
- `assignee_id=<uuid>` or `assignee_ids=<uuid,uuid>`
- `task_type_ids=<uuid,uuid>`
- `parent_task_id=<uuid>`

### Context conventions

- **Sprint page**: seeded with the current sprint in `config.filters.sprint_ids`
- **Backlog page**: seeded with all sprints and groups by `column_by = sprint`
- **Timeline page**: seeded with Epic-only `task_type_ids`

Manual ordering metadata is fetched separately from the shared view task-positions endpoint.

## Saved view config

Each view stores both presentation settings and reusable filters.

```json
{
  "fields": ["title", "status", "assignee"],
  "column_by": "status",
  "swimlanes": "assignee",
  "sort_by": "manual",
  "field_sum": "count",
  "slice_by": "none",
  "filters": {
    "sprint_ids": ["..."],
    "status_ids": ["..."],
    "assignee_ids": ["..."],
    "task_type_ids": ["..."]
  }
}
```

### Rule of thumb

- **Presentation** belongs in top-level config keys such as `column_by`, `sort_by`, and `slice_by`
- **Query constraints** belong in `config.filters`

That keeps the web app predictable: open a view, read its config, send the same saved filters to the single task-list API.

## Default seeded views

Typical defaults are:

- **Sprint**: Board and Table, both seeded with `column_by = status`, the current sprint selected, and non-system task types
- **Backlog**: Table, seeded with `column_by = sprint`, all sprints, and non-system task types
- **Timeline**: Roadmap, seeded with Epic-only task types across all sprints

Additional views can still be created by users per context.

## Web-app flow

For every interaction page:

1. load the views for the current context
2. pick the active view
3. read `activeView.config`
4. call the single task-list API with any saved `config.filters`
   - sprint scope now comes from `filters.sprint_ids` when present
   - page defaults are seeded by the API when projects and sprints are created
5. if the view uses manual sorting, load its task positions from the shared task-positions endpoint
6. render Board, Table, or Roadmap using the same task payload

## Maintenance guidance

When adding future behavior:

- prefer extending `ViewConfig` instead of adding new route families
- prefer filter-based task queries instead of context-specific list endpoints
- keep the dedicated view task-positions endpoint as the source of truth for manual ordering metadata
- keep timeline behavior driven by task types rather than hard-coded endpoint rules
