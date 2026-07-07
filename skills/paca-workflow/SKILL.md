---
name: paca-workflow
description: Plan, design, and build an automation workflow in Paca — a dependency graph over existing tasks that auto-assigns work as statuses change and unlocks downstream tasks once their predecessors finish. Use when asked to automate a process, set up auto-assignment rules, chain task statuses, wire up dependencies so work hands off automatically, or edit/activate/archive an existing automation workflow.
compatibility: Requires Paca MCP server. Run /paca-setup if Paca tools are not available.
---

You are designing and building an automation workflow in Paca — a dependency graph over existing tasks, plus two workflow-level lookup tables (status→assignee rules, and a status-transition chain). Use Paca MCP tools throughout — never create local files.

**If no workflow or goal is specified**, call `get_workflow` (without `workflowId`) to list existing workflows, and ask what process should be automated.

---

## Step 1 — Understand the domain model before touching any tool

A workflow has four parts, all addressed by `taskId` / `statusId` / `memberId` — you never need an internal node/edge/rule/transition ID to build or edit one:

- **Nodes** — existing tasks wrapped into the graph, each with a required canvas position (`posX`, `posY`). There is no auto-placement; you decide both before calling.
- **Edges** — `sourceTaskId → targetTaskId` dependency links. Once the source task reaches the workflow's done status, the target task is re-evaluated against the status rules using **its own current status** — no status is force-changed on the target, only its assignment. A target with multiple incoming edges waits for ALL predecessors to reach done.
- **Status rules** — one shared table for the whole workflow: whenever ANY task in it changes to `statusId`, auto-assign `assigneeMemberId`. Creating a workflow seeds one default rule per project status automatically (assigned to you, or the project's first human member if you're the agent, since an agent can't hand work to itself) — these are valid starting points, not placeholders to delete. Change one with `statusRules.set` (upserts by `statusId`); don't remove-then-recreate.
- **Status transitions** (the "status workflow") — for each status, what status comes next once work there is done. The status with no configured "next" is the workflow's done status — this is what edges watch for, and what tells an AI-agent assignee exactly what status to set next. A default chain is auto-generated from the project's statuses ordered by board position (chained sequentially); override entries via `statusTransitions`.

**Lifecycle:** `draft` (freely editable, ignored by the automation engine) → `active` (engine runs it; requires ≥1 node and exactly one status with no next configured) → `archived` (terminal — can never revert; delete or build a new one instead).

## Step 2 — Gather everything before calling create_workflow or update_workflow

1. Resolve the project (`list_projects` if not already known).
2. Call `list_task_statuses` and `list_project_members` — you'll need statusIds and memberIds for rules/transitions.
3. Identify the tasks to automate via `list_tasks` (or create them first with `create_task` — nodes wrap **existing** tasks only, never invent a task inline here).
4. Work out layout on paper before calling anything:
   - Tasks with no predecessor go in the top row (`posY = 0`); each task depending on another sits in a strictly **lower** row (larger `posY`) than everything it depends on — this must agree with the edges you declare.
   - Tasks that are parallel/independent at the same stage share the same `posY` but get different `posX` values.
   - Space lanes ≥300px apart on X or ≥200px apart on Y — e.g. `posY = 0, 200, 400` for a 3-stage pipeline, `posX = 0, 300, 600` for 3 parallel lanes at one stage. The tool checks this after the call and returns a warning naming any pair still too close — treat that as a required fix, not a suggestion.
5. If editing an existing workflow, call `get_workflow` with its `workflowId` first to see current nodes/edges/positions, and lay out any new nodes past them rather than guessing or reusing a position.

## Step 3 — Build it

- **New workflow**: call `create_workflow` with `name` and, ideally, the whole graph (`nodes`, `edges`, and any `statusRules`/`statusTransitions` overrides) in one call — it starts in `draft`. Pass `activate: true` once the graph is complete, or activate later via `update_workflow`.
- **Existing workflow**: call `update_workflow`, addressing everything by taskId/statusId — `nodes: {set, remove}`, `statusRules: {set, remove}`, `statusTransitions: {set, remove}`, `edges: {add, remove}`. Put every node you're touching in **one** `nodes.set` array in a single call — never one call per node, even for pure repositioning.
- You do not need to manage the draft/active lock yourself: if you call `update_workflow` with graph edits and omit `status`, an active workflow is automatically reverted to draft, edited, and re-activated within that same call. Only pass `status` explicitly when you want a different end state than what it already had (e.g. `"draft"` to stay in draft for review, `"archived"` to retire it permanently).
- Each node/rule/transition/edge is applied independently — one bad entry (e.g. an edge that would create a cycle) doesn't block the rest. Check the response for any failed items and fix them with a follow-up call.

## Step 4 — Verify and confirm

1. Call `get_workflow` with the `workflowId` to review the final graph.
2. If activation was the goal, confirm the response says the workflow is now `active` (it requires ≥1 node and exactly one status with no next status).
3. Report back: workflow name and ID, node/edge count, the done status, and who gets auto-assigned at each stage.

**Archiving is permanent** — confirm with the user before setting `status: "archived"` rather than assuming that's what "pause" or "turn off" means; they may just want `"draft"`.

---

## Examples

| User message | What you do |
|---|---|
| `automate the bug-fix process: triage → fix → review → done, assign review to Alice` | `create_workflow` with 4 nodes chained top-to-bottom, edges linking each stage in order, a status rule assigning the review status to Alice |
| `these 3 tasks can run in parallel, then #9 depends on all of them` | 3 nodes at the same `posY` (different `posX`) plus 1 node in the row below; 3 edges from each parallel task into #9 |
| `pause the deploy workflow` | Ask whether they mean temporarily (`status: "draft"`) or permanently (`status: "archived"`, irreversible) before acting |
| `reposition these nodes so they don't overlap` | `get_workflow` to see current positions → one `update_workflow` call with every affected node in a single `nodes.set` array |

---

## If Paca MCP is not connected

> Paca MCP tools are not available. Run `/paca-setup` to configure the connection.

---

## Tool reference

**Workflows:** `get_workflow` (also lists all workflows when `workflowId` is omitted) · `create_workflow` · `update_workflow` · `delete_workflow`
**Supporting lookups:** `list_tasks` · `list_task_statuses` · `list_project_members` · `list_projects`
