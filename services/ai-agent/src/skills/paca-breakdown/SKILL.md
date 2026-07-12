---
name: paca-breakdown
description: Break a large Paca task or epic into smaller, actionable sub-tasks with dependency ordering. Use when decomposing work that is too large to estimate or execute in a single session, when creating an implementation plan, or when a task needs to be split into vertical slices before sprint planning.
triggers:
  - /paca-breakdown
---

You are breaking a task or epic into smaller, actionable sub-tasks in Paca. Use Paca MCP tools throughout — never create local files.

**If no task is specified**, call `list_tasks` filtered to the current sprint or backlog and show the user the largest or most complex unbroken tasks as candidates.

---

## Step 1 — Load project context

1. Resolve the task reference from the user's message using `get_task_by_number` or `get_task`.
2. Call `list_docs` and read documents that explain the technical landscape — architecture, design decisions, BDD scenarios, API specs. Understanding the tech stack and boundaries is essential to find the right split points.
3. Call `list_task_types` and `list_task_statuses` to know available types and statuses.
4. Call `list_tasks` to see what already exists, so you don't propose sub-tasks that duplicate open work.

## Step 2 — Analyse the task

Read the task title, description, and acceptance criteria. Identify:
- Technical layers involved (frontend, backend, database, infra, tests, docs)
- External dependencies or blockers to flag
- Natural vertical slices (end-to-end thin features) vs. horizontal layers — **prefer vertical slices**
- Any parts that are risky or uncertain and should be isolated as their own sub-task (unknown = own task)

## Step 3 — Propose the breakdown

Present a numbered list of proposed sub-tasks. For each:
- **Title** — clear, imperative ("Add rate-limiting middleware", not "Rate limiting")
- **Size** — fits in 1–2 days
- **Done condition** — one sentence on what "done" looks like
- **Depends on** — note any sub-tasks that must be completed first (e.g. "Depends on #3 above")

Ask the user to confirm, adjust, or remove items before creating anything. This is especially important if the list has more than 6 items.

## Step 4 — Create sub-tasks

For each confirmed sub-task, call `create_task`:
- Reference the parent in the description: `Part of #<parent-number>`
- Include the done condition as the acceptance criteria
- Note any dependencies: `Blocked by #<sibling-number>` if one sub-task must precede another
- Assign to the same sprint as the parent if one exists (check with `list_sprints`)

**What's next:** Consider running `/paca-estimate` on the newly created sub-tasks to add story points.

Report back: parent task number, list of created sub-task numbers, titles, and any dependency chain noted.

---

## Tool reference

**Tasks:** `get_task` · `get_task_by_number` · `list_tasks` · `create_task` · `update_task` · `list_task_types` · `list_task_statuses`
**Documents:** `list_docs` · `read_doc`
**Sprints:** `list_sprints`
**Projects:** `list_projects`
