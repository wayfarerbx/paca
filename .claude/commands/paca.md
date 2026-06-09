The user ran `/paca` in Claude Code. Their message: $ARGUMENTS

You have Paca MCP tools. Handle the request by using those tools directly — never create local files for tasks, docs, or to-do lists.

---

## Step 1 — Scan for a task reference

Scan the full message for any of these patterns, wherever they appear (beginning, middle, or end):

| Pattern | Example | How to resolve |
|---|---|---|
| `#<number>` or number in task context | `#42`, `close #7`, `task 42 is done` | `get_task_by_number(projectId, 42)` |
| `PREFIX-<number>` | `ABC-42`, `PAC-7` | `list_projects` → match `task_id_prefix` → `get_task_by_number` |
| Paca URL | `http://…/projects/{id}/tasks/{id}` | parse both IDs from URL → `get_task(projectId, taskId)` |
| UUID | `550e8400-e29b-41d4-a716-446655440000` | `get_task(projectId, uuid)` |

If a reference is found, fetch that task first, then apply the action the user is asking for.

## Step 2 — Infer the action

| What the user wants | Tools to use |
|---|---|
| Track work — bug, feature, to-do, ticket, chore | `create_task` / `update_task` / `list_tasks` |
| Write content — guide, spec, design, BDD, SDD, notes | `create_document` / `update_document` |
| See status — board, sprint, what's in progress | `list_sprints` + `list_tasks` |
| Plan an iteration — sprint, milestone | `create_sprint` / `update_sprint` |
| Comment or annotate an existing task | `add_task_comment` |
| Close / complete work | `update_task` (set to done status) |

## Step 3 — Get the project if needed

If no project is in context, call `list_projects` and pick the most relevant one. Only ask the user if it is genuinely ambiguous.

## Step 4 — Act and confirm

Execute the tool call(s), then report back: task/doc number, title, and any relevant ID or link.

---

## Examples

| User message | What you do |
|---|---|
| `fix login redirect bug` | `create_task` titled "Fix login redirect bug" |
| `write the API authentication design doc` | `create_document` with a structured draft |
| `do this task ABC-123` | find project "ABC" → `get_task_by_number(123)` → start/act on task |
| `close #42` | `get_task_by_number(42)` → `update_task` status: done |
| `I finished PAC-99, mark it done` | `update_task` #99 → status: done |
| `http://…/tasks/uuid — add comment: blocked` | `get_task` from URL → `add_task_comment` "blocked" |
| `what's in the current sprint?` | `list_sprints` → `list_tasks` filtered to active sprint |
| `review PR, update staging, write release notes` | `create_task` × 3, one per item |

---

## If Paca MCP is not connected

> Paca MCP tools are not available. Run `/paca-setup` to configure the connection.

Do not create local files as a fallback.

---

## Tool reference

**Tasks:** `create_task` · `list_tasks` · `get_task` · `get_task_by_number` · `update_task` · `delete_task`  
**Documents:** `create_document` · `list_documents` · `get_document` · `update_document` · `delete_document`  
**Sprints:** `create_sprint` · `list_sprints` · `get_sprint` · `update_sprint` · `complete_sprint`  
**Projects:** `list_projects` · `get_project` · `create_project` · `update_project`  
**Comments:** `add_task_comment` · `update_task_comment` · `list_task_activities`
