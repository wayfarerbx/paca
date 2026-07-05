# Automation Workflows

This document explains the automation-workflow feature: a project-scoped
dependency graph over *existing* tasks that automatically hands work off
between members (human or AI agent) as tasks move through statuses.

## Why this exists

Before this feature, task assignment was always a manual action. Automation
workflows let a project define, once, "when this task is done, hand the next
one to this person" and "whenever this task's status becomes X, assign it to
Y" — without a human re-assigning things by hand every time.

## Core model

A workflow is a directed graph, plus two shared, workflow-level lookup
tables:

- **Node** — wraps one existing task. A task can be a node in zero, one, or
  many workflows.
- **Status rule** (on the workflow) — maps a status to a member: "when any
  task in this workflow reaches status X, assign it to member M." This is
  **one shared list per workflow**, not one list per node — a rule for status
  X applies uniformly no matter which node's task actually reaches X. The
  workflow can have any number of rules, one per status.
- **Status transition** (on the workflow) — the "status workflow," distinct
  from the task-dependency graph above: for each project status, an optional
  "next status" — what a task at that status should move to once work there
  is done. A status with no next status configured is **terminal**; the
  workflow's single **done status** is *derived* as whichever status is
  terminal (see [Done status resolution](#done-status-resolution)). This
  drives both the AND-join in event 2 and the hint given to an AI-agent
  assignee (see [below](#telling-an-assigned-agent-what-to-do-next)).
- **Edge** — a plain directed link `source node → target node`. It carries no
  configuration of its own; it only means "once source is done, re-evaluate
  target."

Both automation events reuse the *same* status-rule lookup — they just
evaluate it for different (task, status) pairs. There is no way to make an
edge itself change a task's status; only the status rules assign, and only
a human, an agent, or another automation-triggered status change ever
changes a task's actual status.

## The two events

1. **Status changed** — a task's status changes (by a human, an agent, or a
   cascade from event 2). Look up the workflow's status rule for the *new*
   status. If one exists and the assignee differs from the current
   assignee, reassign the task to it.
2. **Predecessor done** — a node's task reaches the workflow's **derived done
   status**. For every outgoing edge from that node to a target node, check
   whether *all* of the target's incoming edges now have a done source (see
   [AND-join](#and-join) below). If so, look up the workflow's status rule
   for the target task's **own current status** (unchanged) and, if one
   exists, reassign the target task to it.

Event 2 never changes a task's status — it only decides *whether* to apply
event 1's lookup to a downstream task, using that task's status as it
already stands. This is why the workflow should have a status rule
configured for whatever status a downstream task naturally sits in while
waiting (e.g. "Ready" or "To Do"), not just a terminal status — otherwise
there's nothing to reassign it to when it unlocks.

### Done status resolution

Unlike an earlier version of this feature, there is no per-node
`done_status_id` field and no implicit "project's single done-category
status" fallback. The workflow's done status is always **derived** from its
status-transition chain: it's whichever status has `next_status_id = NULL`
(see `workflowdom.DeriveDoneStatusID`). This must be exactly one status —
`Activate` (see [Lifecycle](#lifecycle)) rejects the workflow if the chain
has zero or more than one such entry.

A default chain is auto-generated when a workflow is created:
`CreateWorkflow` lists the project's task statuses ordered by board
`position` and chains them sequentially (status at position *N*'s next
status is the one at position *N+1*), leaving the last (highest-position)
status terminal. Users and agents can customize this afterward via
`SetStatusTransition`/`set_workflow_status_transition`.

`CreateWorkflow` also seeds a default status rule for every status, so a
brand-new workflow already hands work off somewhere instead of doing nothing
until manually configured. The default assignee is resolved as: the
workflow's creator, if they're a human user; otherwise (the creator is an AI
agent, or couldn't be resolved) the project's first human member — an agent
can't hand its own work off to itself, so the default has to be a real
person. If the project has no human members at all, no default rules are
seeded. Both auto-seed steps are best-effort — failures don't block workflow
creation, since the resulting empty-chain/no-rules draft is still perfectly
usable and fixable via the inline, always-visible editors.

### AND-join

If a target node has more than one incoming edge, event 2 only fires once
**every** predecessor has reached the workflow's derived done status — not
on the first one.

This is evaluated **statelessly**: there is no persisted "join progress"
counter. On every status-change event, the engine re-derives "has predecessor
P finished?" by reading P's task's *current* status live and comparing it to
the workflow's derived done status. If not all predecessors currently
qualify, nothing happens; the same check naturally re-runs (and can pass) the
next time a remaining predecessor also reaches the done status. This makes
the join idempotent and safe under at-least-once stream redelivery —
replaying the same event twice just recomputes the same booleans.

### Loop safety

The graph is validated as a DAG at edge-creation time (same reachability
check used for task parent/child cycles). Because event 2 only ever
propagates forward along edges, and the graph cannot contain a cycle, any
chain of cascades is guaranteed to terminate in at most *N* node-hops. An
idempotency check (only reassign when the assignee actually changes) stops
redundant re-fires even if the same event is processed more than once.

## Lifecycle

A workflow is always in one of three states:

| State      | Meaning                                                        |
|------------|-----------------------------------------------------------------|
| `draft`    | Freely editable (nodes, edges, status rules, status transitions). Engine ignores it. |
| `active`   | Engine evaluates it on every relevant task status change. Graph is locked — no node/edge/status-rule/status-transition mutations. |
| `archived` | Engine ignores it. Terminal-ish; can be reverted to draft.       |

Transitions: `draft → active` (`Activate`, validated — see below),
`active → archived` (`Archive`), `active|archived → draft` (`RevertToDraft`,
re-enables editing). Renaming/describing a workflow is allowed in any state.

**Activation validation**: at least one node; the graph is still a DAG
(defensive re-check); every node's task still exists in the project (hasn't
been deleted since the node was added); the workflow's status-transition
chain has exactly one derivable done status (see
[Done status resolution](#done-status-resolution)).

## Data model

```sql
workflows                    -- id, project_id, name, description, status, created_by
workflow_nodes                -- id, workflow_id, task_id, pos_x, pos_y
workflow_status_rules         -- id, workflow_id, status_id, assignee_member_id
workflow_status_transitions   -- id, workflow_id, status_id, next_status_id (nullable)
workflow_edges                -- id, workflow_id, source_node_id, target_node_id
```

See `services/api/migrations/000018_add_automation_workflows.sql` for the
full DDL. Key constraints:

- `workflow_nodes` has a unique `(workflow_id, task_id)` — a task appears at
  most once *per workflow*, but can belong to many different workflows.
- `workflow_edges` has `CHECK (source_node_id <> target_node_id)` (no
  self-loops) and a unique `(source_node_id, target_node_id)` (no duplicate
  edges).
- `workflow_status_rules` has a unique `(workflow_id, status_id)` — one rule
  per status per workflow (not per node); setting a rule for an
  already-configured status updates the existing row's assignee rather than
  creating a second one.
- `workflow_status_transitions` has a unique `(workflow_id, status_id)` and
  `CHECK (next_status_id IS NULL OR next_status_id <> status_id)` (no
  self-transitions); `next_status_id = NULL` marks that status as the
  workflow's terminal/done status.

## Execution engine

`internal/worker/workflow_consumer.go` (`WorkflowConsumer`) subscribes to the
same Valkey stream the task-activity pipeline already writes to,
`paca.task_activities` (`events.StreamTaskActivities`), under its own
consumer group `api.workflow_engine` — it is a sibling reader, not a special
case wired into the HTTP handler.

On every `task.updated` activity whose `FieldChange[]` includes a `status`
entry:

1. `ListActiveNodesByTaskID(taskID)` — nodes across *active* workflows only
   referencing this task. If none, ack and return (cheap no-op for the
   overwhelming majority of ordinary task updates).
2. Re-fetch the task fresh from the repository to get its authoritative
   current `StatusID` — the activity payload's `FieldChange` carries resolved
   status *names*, not IDs, so it can't be used directly.
3. For each matching node: apply event 1 (the workflow's rule for the task's
   new status), then check `isNodeDone` — whether the task's status equals
   the workflow's derived done status (`workflowdom.DeriveDoneStatusID` over
   `ListStatusTransitionsByWorkflow`). If so, walk the node's outgoing edges
   and apply event 2's AND-join check + reassignment to each qualifying
   target.

Reassignments go through the ordinary task service `UpdateTask`, so they get
the same validation and side effects as a human PATCH. The engine then:

- Records a `workflow.assigned` activity (`taskdom.ActivityTypeWorkflowAssigned`)
  with `{workflow_id, workflow_name, reason: "status_rule"|"predecessor_done",
  old_assignee, new_assignee}` so the activity feed can attribute the change
  to the workflow instead of a human actor (`ActorID` is left `nil`).
- Publishes to `events.StreamTaskAssignments` (mirroring what the task
  handler does for a human-initiated assignee change) so the existing
  `NotificationConsumer` still creates the in-app notification / triggers the
  agent conversation uniformly, regardless of whether the reassignment came
  from a human PATCH or the workflow engine.

### Telling an assigned agent what to do next

When the newly-assigned member is an AI agent, `NotificationConsumer` folds
the workflow's name and a **next-status name** into a note appended to the
agent's initial prompt via `TriggerTaskAssigned`'s trailing `note` parameter.
The next-status name is looked up from the workflow's status-transition
chain for the task's *current* status (not necessarily the workflow's
terminal done status) — e.g. if the agent is assigned when a task hits
"In Progress" and the chain says "In Progress" → "Review", the note reads
*"This task was automatically assigned to you by the automation workflow
'Release Pipeline'. When you finish your part, set the task status to
'Review' to continue the workflow."* If the task is already at the
terminal/done status (no configured next), the second sentence is omitted.

This is the fix for a problem the earlier per-node-done-status design had:
an agent assigned partway through a multi-handoff pipeline used to always be
told to set the status straight to the final done status, skipping whatever
intermediate stages and assignees the status rules were supposed to route
through. Now the agent is always told the *immediate* next status, one hop
at a time, matching how the status-rule handoffs actually work.

## API

All endpoints are under `/api/v1/projects/:projectId/workflows`. Read routes
require `workflows.read`; everything else requires `workflows.write`. Node,
edge, status-rule, and status-transition mutations additionally require the
workflow to be in `draft` state.

```
GET    /workflows                                  list (optional ?status=draft|active|archived)
POST   /workflows                                   create (starts in draft; auto-seeds a default status-transition chain)
GET    /workflows/:workflowId                        get full graph (workflow + nodes + edges + status rules + status transitions)
PATCH  /workflows/:workflowId                        rename / re-describe
DELETE /workflows/:workflowId                        soft-delete
POST   /workflows/:workflowId/activate               draft → active
POST   /workflows/:workflowId/archive                active → archived
POST   /workflows/:workflowId/revert-to-draft        active|archived → draft

POST   /workflows/:workflowId/nodes                                    add a task as a node
PATCH  /workflows/:workflowId/nodes/:nodeId                             move (pos_x/pos_y)
DELETE /workflows/:workflowId/nodes/:nodeId                             remove (cascades its edges)

POST   /workflows/:workflowId/status-rules                              create/update a status → assignee rule
DELETE /workflows/:workflowId/status-rules/:ruleId                      remove a rule

POST   /workflows/:workflowId/status-transitions                       create/update a status → next-status entry
DELETE /workflows/:workflowId/status-transitions/:transitionId         remove an entry

POST   /workflows/:workflowId/edges                                     link two nodes (runs the DAG/cycle check)
DELETE /workflows/:workflowId/edges/:edgeId                              remove a link
```

## Permissions

Two new permission keys, following the same `<domain>.read` / `<domain>.write`
convention as the rest of the project permission model:

- `workflows.read` — view workflows and their graphs.
- `workflows.write` — create/edit/activate/archive/delete workflows and their
  nodes, edges, status rules, and status transitions.

Granted by default to: `PROJECT_OWNER` / `PROJECT_MANAGER` (via
`workflows.*`), `PROJECT_MEMBER` (both keys), `PROJECT_VIEWER` (read only) —
see `authz.DefaultProjectRoles()`. The per-project roles actually seeded on
project creation (`Admin`/`Editor`/`Viewer`, in `projectsvc.CreateProject`)
carry the same grants. The tail of `000018_add_automation_workflows.sql`
backfills these two keys onto every `project_roles` row already using one of
these role names, so projects created before this feature shipped don't need
manual reconfiguration.

## AI agent integration

Workflow management is exposed to AI agents as ordinary MCP tools in the
existing Paca MCP server (`apps/mcp/src/tools/workflow-tools.ts`) — the same
mechanism used for every other Paca resource (tasks, sprints, docs, etc.),
not a special-cased sandbox tool or a separate server. Tool availability is
gated the same way every other tool in that server is: by the calling
agent's own `workflows.read` / `workflows.write` project permissions
(`apps/mcp/src/permissions.ts`), resolved at MCP-session startup — there is
no separate per-agent capability flag.

Tools: `list_workflows`, `get_workflow`, `create_workflow`,
`update_workflow`, `delete_workflow`, `activate_workflow`,
`archive_workflow`, `revert_workflow_to_draft`, `add_workflow_node`,
`remove_workflow_node`, `set_workflow_status_rule`,
`remove_workflow_status_rule`, `set_workflow_status_transition`,
`remove_workflow_status_transition`, `add_workflow_edge`,
`remove_workflow_edge`. Like other tools in this server, they take explicit
UUIDs (task/status/member IDs obtained via `list_tasks` / `list_task_statuses`
/ `list_project_members`) rather than resolving names themselves.
`create_workflow` auto-seeds the default status-transition chain the same
way the HTTP API does; `set_workflow_status_transition` customizes it
afterward.

## Frontend

The visual builder lives at `apps/web/src/routes/_authenticated/projects/$projectId/automation/`
— a list page and a canvas builder page (`@xyflow/react`), reachable from the
project sidebar's "Automation" entry. `apps/web/src/lib/workflow-api.ts` is
the API client.

The builder page layout is a persistent left sidebar (collapsible via a
toolbar toggle) next to the canvas — not a button-triggered overlay — since
the status-transition chain and status rules are the settings that actually
make automation work and shouldn't be hidden behind a click. The sidebar
stacks `workflow-status-transitions-panel.tsx` (the status-workflow chain
editor: next-status picker per project status, with a warning banner if the
chain doesn't have exactly one terminal/done status) above
`workflow-status-rules-panel.tsx` (the status→assignee rule list). See
`apps/web/src/components/projects/automation/` for these, the canvas, the
per-node panel (now just "remove from workflow" — done status is no longer
per-node), and the task-picker components.
