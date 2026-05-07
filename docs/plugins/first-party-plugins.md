# First-Party Plugins Migration Plan

This document describes how four features that currently live in the core codebase will be extracted into first-party plugins as the proof-of-concept for the plugin system. Completing these migrations validates the extension point contracts and the SDK before the system is opened to third-party developers.

## Plugins to Migrate

| Plugin ID | Current Location (Backend) | Current Location (Frontend) | Extension Points |
|---|---|---|---|
| `com.paca.bdd` | `internal/domain/task` (BDDScenario entity, service methods) | `task-detail/bdd-scenarios-section.tsx` | `task.detail.section`, `project.settings.tab` |
| `com.paca.checklist` | `internal/domain/task` (task_checklists, task_checklist_items tables) | `task-detail/checklist-section.tsx`, `task-detail/checklists-section.tsx` | `task.detail.section` |
| `com.paca.github` | `internal/domain/github` (full domain package) | `settings/GitHubSettings.tsx`, `task-detail/branches-section.tsx`, `task-detail/pull-requests-section.tsx` | `task.detail.section`, `project.settings.tab`, `sidebar.project.section` |
| `com.paca.time-tracking` | Not yet implemented (new feature) | Not yet implemented | `task.detail.section`, `project.settings.tab`, `sidebar.project.section` |

---

## Migration Strategy

The migration follows a **strangler-fig pattern**: the plugin system infrastructure is built first, then each feature is moved behind it one at a time while keeping API compatibility for existing clients (MCP server, e2e tests). Once all four plugins are stable, the core code is removed.

### Phase 1 — Build the Plugin Infrastructure

Before any feature migration:

1. Implement the host-side plugin runtime (`services/api/internal/plugin/`).
2. Implement the frontend extension point registry and `<ExtensionPoint>` primitive (`apps/web/src/lib/plugins/`).
3. Publish initial versions of `@paca-ai/plugin-sdk-react` and `github.com/Paca-AI/plugin-sdk-go`.
4. Add the `plugins` and `plugin_extension_settings` tables via a core migration.
5. Implement `GET /api/v1/plugins` (list enabled plugins with their manifests).
6. Implement `PATCH /api/v1/admin/plugin-extension-settings` (super admin sets system-wide extension point ordering/visibility).

### Phase 2 — Migrate One Plugin at a Time

Each plugin migration follows these steps:

1. **Create the plugin directory** under `plugins/first-party/{name}/`.
2. **Write the backend WASM plugin** using the Go SDK.
3. **Write the frontend micro-frontend** using the TypeScript SDK.
4. **Write the `plugin.json` manifest.**
5. **Write plugin-owned DB migrations** (move the tables from the core schema into the plugin schema).
6. **Remove the feature from core** (domain entities, service methods, router routes, handler methods, DTOs).
7. **Replace the hardcoded UI component** with `<ExtensionPoint id="..." />`.
8. **Update e2e tests** to remain green (API paths change from `/api/v1/projects/:id/tasks/:id/bdd-scenarios` to `/api/v1/plugins/com.paca.bdd/projects/:id/tasks/:id/bdd-scenarios`).

### Phase 3 — Validate and Open to Third Parties

After all four first-party migrations are complete:

1. Review the SDK APIs against the experience of writing real plugins.
2. Publish SDK packages to npm and pkg.go.dev.
3. Write the external developer guide.
4. Announce the plugin system.

---

## Plugin: BDD Scenarios (`com.paca.bdd`)

### What It Does

Lets teams write Given/When/Then acceptance criteria attached to a task. Each task can have multiple named BDD scenarios.

### Current Backend Surface

- Entity: `taskdom.BDDScenario` in `internal/domain/task/entity.go`
- Service methods on `taskdom.Service`: `ListBDDScenarios`, `CreateBDDScenario`, `GetBDDScenario`, `UpdateBDDScenario`, `DeleteBDDScenario`
- Repository: BDD scenario CRUD in `internal/repository/task/`
- Handler methods on `TaskHandler`: `ListBDDScenarios`, `CreateBDDScenario`, `GetBDDScenario`, `UpdateBDDScenario`, `DeleteBDDScenario`
- Routes: `GET|POST /projects/:id/tasks/:taskId/bdd-scenarios`, `GET|PATCH|DELETE /projects/:id/tasks/:taskId/bdd-scenarios/:scenarioId`
- Activity events: `task.bdd_scenario.created`, `task.bdd_scenario.updated`, `task.bdd_scenario.deleted`
- DB table: `bdd_scenarios`

### Current Frontend Surface

- `apps/web/src/components/projects/interactions/task-detail/bdd-scenarios-section.tsx`
- API client calls: `GET|POST|PATCH|DELETE /projects/:id/tasks/:taskId/bdd-scenarios`

### Plugin Extension Points

- `task.detail.section` — renders the BDD panel below the task description.
- `project.settings.tab` — (optional v2) lets admins configure BDD defaults or templates.

### Migration Steps

1. Create `plugins/first-party/bdd/`.
2. Write `backend/main.go` implementing CRUD routes and a `task.deleted` event handler (cascade delete orphan scenarios).
3. Write `backend/migrations/0001_create_bdd_scenarios.sql` matching the current `bdd_scenarios` table DDL.
4. Write `frontend/src/TaskDetailSection.tsx` (extracted from `bdd-scenarios-section.tsx` with SDK API calls).
5. Write `plugin.json`.
6. Remove `BDDScenario` entity, methods, routes, and handlers from core.
7. Replace `<BDDScenariosSection />` in `task-detail/index.tsx` with `<ExtensionPoint id="task.detail.section" ... />`.

### API Path Change

| Before | After |
|---|---|
| `GET /api/v1/projects/:pid/tasks/:tid/bdd-scenarios` | `GET /api/v1/plugins/com.paca.bdd/projects/:pid/tasks/:tid/bdd-scenarios` |

---

## Plugin: Checklist (`com.paca.checklist`)

### What It Does

Adds one or more named checklists to a task, each containing ordered checkable items with optional assignees and due dates.

### Current Backend Surface

- Activity events: `task.checklist.created/updated/deleted`, `task.checklist_item.created/updated/deleted/checked/unchecked`
- DB tables: `task_checklists`, `task_checklist_items` (defined in `000001_init.sql`)
- Handler methods: `ListChecklists`, `CreateChecklist`, `UpdateChecklist`, `DeleteChecklist`, `CreateChecklistItem`, `UpdateChecklistItem`, `DeleteChecklistItem`, `CheckChecklistItem`, `UncheckChecklistItem`

### Current Frontend Surface

- `apps/web/src/components/projects/interactions/task-detail/checklist-section.tsx`
- `apps/web/src/components/projects/interactions/task-detail/checklists-section.tsx`
- Referenced via `<ChecklistsSection canEdit={canEdit} />` in `task-detail/index.tsx`

### Plugin Extension Points

- `task.detail.section` — renders the checklists panel.

### Migration Steps

1. Create `plugins/first-party/checklist/`.
2. Write the backend with routes for checklist CRUD and item CRUD.
3. Write `backend/migrations/0001_create_task_checklists.sql`.
4. Extract frontend components into the plugin, using `sdk.api` for all HTTP calls.
5. Remove checklist domain from core.
6. Replace `<ChecklistsSection />` with `<ExtensionPoint id="task.detail.section" ... />` (it will aggregate all registered `task.detail.section` plugins).

### API Path Change

| Before | After |
|---|---|
| `GET /api/v1/projects/:pid/tasks/:tid/checklists` | `GET /api/v1/plugins/com.paca.checklist/projects/:pid/tasks/:tid/checklists` |

---

## Plugin: GitHub Integration (`com.paca.github`)

### What It Does

- Stores an encrypted GitHub PAT per project.
- Links a GitHub repository to a project.
- Registers a webhook on the linked repo to sync pull requests.
- Lets users link pull requests to tasks.
- Shows branches and pull requests in the task detail panel.
- Shows GitHub settings in the project settings page.

### Current Backend Surface

- Full domain package: `internal/domain/github/`
- Handler: `internal/transport/http/handler/github_handler.go`
- Routes: `GET|POST|DELETE /projects/:id/github/integration`, `GET|POST|DELETE /projects/:id/github/repository`, `GET /projects/:id/github/pull-requests`, `POST|DELETE /projects/:id/tasks/:taskId/github/pr-links/:prId`
- External outbound HTTP calls to the GitHub REST API (must be declared in `plugin.json` as a network permission)

### Current Frontend Surface

- `apps/web/src/components/projects/settings/GitHubSettings.tsx`
- `apps/web/src/components/projects/interactions/task-detail/branches-section.tsx`
- `apps/web/src/components/projects/interactions/task-detail/pull-requests-section.tsx`

### Plugin Extension Points

- `task.detail.section` — branches and pull requests panels.
- `project.settings.tab` — GitHub configuration (PAT, linked repository).
- `sidebar.project.section` — (optional) quick link to open PRs for the project.

### Special Considerations

- The GitHub plugin makes **outbound HTTP calls** to `api.github.com`. This requires a network host function (`paca.http_fetch`) or an allowlisted outbound proxy function — the only plugin that needs it. The manifest must declare `"network:fetch:api.github.com"` permission.
- The encrypted PAT is stored in the plugin's storage (not raw plugin_data SQL) using the host's encryption key via a `paca.secrets_encrypt` / `paca.secrets_decrypt` host function pair.
- The webhook endpoint (`POST /webhooks/github`) must remain accessible at a stable URL. After migration it becomes a core-level webhook dispatcher that forwards payloads to the `com.paca.github` plugin's event handler.

### Migration Steps

1. Create `plugins/first-party/github/`.
2. Add `paca.http_fetch` and `paca.secrets_encrypt/decrypt` host functions (GitHub is the only v1 consumer).
3. Move `internal/domain/github/` business logic into the plugin backend.
4. Keep the webhook dispatcher in core (`POST /webhooks/github`) but route the payload as a `com.paca.github.webhook` event via the plugin event system.
5. Extract frontend components.
6. Remove `internal/domain/github/`, `github_handler.go`, and GitHub routes from core.

### API Path Change

| Before | After |
|---|---|
| `GET /api/v1/projects/:pid/github/integration` | `GET /api/v1/plugins/com.paca.github/projects/:pid/integration` |
| `GET /api/v1/projects/:pid/github/pull-requests` | `GET /api/v1/plugins/com.paca.github/projects/:pid/pull-requests` |

---

## Plugin: Time Tracking (`com.paca.time-tracking`)

### What It Does (New Feature)

Time tracking is not yet implemented in core. This is a net-new feature that will be built entirely as a plugin, demonstrating that plugins can introduce brand-new capabilities without any core involvement.

### Feature Scope

- Members can log time entries against a task (duration, description, logged-at date).
- Members can view and edit their own entries; project admins can view and delete all entries.
- A per-task "Time Spent" summary is shown in the task detail panel.
- A per-project time log report is shown in a project settings tab.
- The project sidebar section shows a link to the time log report.

### Extension Points

- `task.detail.section` — "Time Tracking" panel showing logged entries and an "Add Entry" form.
- `project.settings.tab` — full project time log report with filter and export.
- `sidebar.project.section` — "Time Log" navigation link.

### Backend Schema (new, plugin-owned)

```sql
-- plugins/first-party/time-tracking/backend/migrations/0001_create_time_entries.sql
CREATE TABLE time_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID NOT NULL,
    project_id  UUID NOT NULL,
    user_id     UUID NOT NULL,
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    description TEXT,
    logged_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON time_entries (task_id);
CREATE INDEX ON time_entries (project_id, user_id);
```

### Routes

| Method | Path | Description |
|---|---|---|
| `GET` | `/tasks/:taskId/time-entries` | List entries for a task |
| `POST` | `/tasks/:taskId/time-entries` | Log a new entry |
| `PATCH` | `/tasks/:taskId/time-entries/:entryId` | Edit an entry (own entries only, unless admin) |
| `DELETE` | `/tasks/:taskId/time-entries/:entryId` | Delete an entry |
| `GET` | `/time-report` | Project-wide time report (aggregated by member, date range) |

---

## Database Migration Strategy

### Moving Tables from Core Schema to Plugin Schema

When a feature is migrated, its tables move from the public core schema into the `plugin_data_{pluginId}` schema owned by the plugin. The migration sequence is:

1. **Core migration** (new file in `services/api/migrations/`):
   - Rename the table to the plugin schema (e.g., `ALTER TABLE bdd_scenarios SET SCHEMA plugin_data_com_paca_bdd`).
   - Drop the related foreign-key columns from `tasks` if any.
   - Remove activity log entries referencing the old activity types (or remap them).
2. **Plugin installation** runs its own `0001_*.sql` which expects the table to already exist (for existing installations) or creates it fresh (for new installations).

For zero-downtime, the rename migration and the plugin installation are coordinated as a single deploy step.

### Activity Log

The `task_activities` table records activity types like `task.bdd_scenario.created`. After migration, plugins emit these as plugin-namespaced events (`com.paca.bdd.scenario_created`) via `paca.event_emit`. The activity log display in the task detail panel will be updated to render plugin activity events using a plugin-provided renderer component registered at the `activity.item` extension point (v2 consideration).

---

## Impact on MCP Server (`apps/mcp`)

The MCP server in `apps/mcp` wraps core API routes. After migration, BDD and checklist tool calls will target the new plugin API paths. The MCP tool implementations must be updated to call `/api/v1/plugins/com.paca.bdd/...` instead of `/api/v1/projects/.../bdd-scenarios`.

This is a **breaking change for MCP tool paths**. A minor version bump of the MCP server is required.

---

## Impact on E2E Tests (`apps/e2e`)

E2E tests cover BDD scenario CRUD in `tests/projects/bdd-scenarios.spec.ts` (and the BDD feature file). After migration:
- API call paths in page objects must be updated.
- The plugin must be enabled in the e2e seed data.
- The test environment config must include the plugin bundle URL (can point to a local build).

---

## Rollout Order

1. `com.paca.checklist` — simplest, no external dependencies, no complex schema.
2. `com.paca.bdd` — similar complexity, validates the pattern end-to-end.
3. `com.paca.time-tracking` — net-new, validates that plugins can introduce new features.
4. `com.paca.github` — most complex due to webhooks, external HTTP, and encryption.
