---
name: paca
description: Interact with Paca project management using MCP tools. Use when tracking tasks, writing docs, planning sprints, managing work items, creating bugs or features, viewing the board, or handling any project-management request involving Paca. Routes to specialized skills for complex workflows like epics, sprint planning, and task execution.
---

You have Paca MCP tools. Handle the request by using those tools directly — never create local files for tasks, docs, or to-do lists.

This is your default operating procedure for every conversation — it always applies, regardless of how you were invoked (task assignment, a comment mention, direct chat, or a description-write request).

---

## Step 0 — Invoke a specialized skill for anything beyond one obvious action

If the request matches one of these, you **must invoke** the specialized skill — actually read its full content via the skill mechanism — before doing the underlying work. Don't reconstruct what it probably says from memory and skip straight to the MCP calls:

| If the user wants to... | Invoke |
|---|---|
| Turn requirements into a full epic with child stories | `paca-epic` |
| Clarify or improve a vague task or spec | `paca-clarify` |
| Break a task into smaller sub-tasks | `paca-breakdown` |
| Plan a sprint from the backlog | `paca-sprint` |
| Estimate story points for tasks | `paca-estimate` |
| Set priorities across the backlog | `paca-prioritize` |
| Execute a task end-to-end | `paca-do` |
| Test or verify a task | `paca-test` |
| Write or update documentation | `paca-doc` |
| Automate a process — auto-assignment, status chaining, task dependencies | `paca-workflow` |

A user can also force one of these directly by typing `/<skill-name>` (e.g. `/paca-do #42`) in a chat message or comment.

**Skip invocation only for a single, unambiguous action that Step 1-4 below fully covers on its own** — e.g. closing a task with no other change, adding a plain comment, checking status/what's in the sprint, fetching a task's details. If the request takes judgment, multiple steps, or produces new content (a plan, an estimate, a breakdown, a draft, a graph), it is not "simple" — invoke the matching skill even if you're confident you already know what it says.

## Step 0.5 — When a task is in scope, let its status guide you

Whenever your invocation is tied to a specific task (an assignment, a comment on a task, or a description-write request), load it first (`get_task` / `get_task_by_number`) and let its **current status** — not just the request wording — help you pick the right specialized skill to invoke:

| Task status | Likely next step |
|---|---|
| No acceptance criteria, or description is thin | `paca-clarify` |
| Backlog / not yet sized or split | `paca-breakdown` (if large) or `paca-estimate` (if right-sized) |
| To do / ready, sprint not yet planned | `paca-sprint` |
| In progress | `paca-do` |
| In review / awaiting QA | `paca-test` |
| Done, but the feature has no linked documentation | `paca-doc` |

This table picks *which* skill; it doesn't excuse you from invoking one (see Step 0) unless the request is itself one of the trivial actions listed there. If the requester's message explicitly asks for something narrower than the status implies (e.g. "just estimate this" on an in-progress task), honor that instead — but still invoke the skill that matches the explicit ask (`paca-estimate`), rather than doing it ad hoc.

## Step 1 — Scan for a task reference

Scan the user's message for any of these patterns, wherever they appear:

| Pattern | Example | How to resolve |
|---|---|---|
| `#<number>` or number in task context | `#42`, `close #7`, `task 42 is done` | `get_task_by_number(projectId, 42)` |
| `PREFIX-<number>` | `ABC-42`, `PAC-7` | `list_projects` → match `task_id_prefix` → `get_task_by_number` |
| Paca URL | `http://…/projects/{id}/tasks/{id}` | parse both IDs → `get_task(projectId, taskId)` |
| UUID | `550e8400-e29b-41d4-a716-446655440000` | `get_task(projectId, uuid)` |

If a reference is found, fetch that task first, then apply the action the user is asking for.

## Step 2 — Infer the action

| What the user wants | Tools to use |
|---|---|
| Track work — bug, feature, to-do, ticket, chore | `create_task` / `update_task` / `list_tasks` |
| Write content — guide, spec, design, BDD, SDD, notes | `write_doc` |
| See status — board, sprint, what's in progress | `list_sprints` + `list_tasks` |
| Plan an iteration — sprint, milestone | `create_sprint` / `update_sprint` |
| Comment or annotate an existing task | `add_task_comment` |
| Close / complete work | `update_task` (set to done status) |
| Break work into pieces | `create_task` × N, each referencing the parent |
| Write or update documentation | `write_doc` (path decides create vs. update) |

## Step 3 — Get the project if needed

If no project is in context, call `list_projects` and pick the most relevant one based on the message. Only ask the user if it is genuinely ambiguous between two equally plausible candidates.

## Step 4 — Act and confirm

Execute the tool call(s), then report back: task/doc number, title, and any relevant ID or link.

## Asking for more information

Whenever you get stuck without enough information to proceed — regardless of how you were invoked — **do not just say so in your conversational reply and stop there**. The requester almost never reopens the agent conversation log, and even in direct chat, they may never come back to that exact session. Whichever surface applies, add a comment, then still say something brief in the conversational reply too (e.g. "Asked for clarification — see the comment on #42") — but the comment is what actually reaches the person.

**For a task** (assignment, task comment, description-write, or a task referenced in chat):
1. Call `add_task_comment` with your question(s) on the task itself.
2. Include a literal `@username` in the comment text (their login username — letters, digits, `.`, `_`, `-` only — not their display name, and not a `#`-style task/doc reference). This is parsed as a real mention and sends an in-app notification. Find the right username with `list_project_members`, matching against whatever name/context you already have (e.g. the commenter's name from `list_task_activities`, or the task's assignee).
3. Stop there — don't guess and proceed past a genuine blocker. Once they reply (typically by commenting back and mentioning you), a new conversation will pick up from where you left off; check `list_task_activities` for that history.

**For a document** (doc comment, or a doc referenced in chat, with no task involved):
- `add_doc_comment` exists, but **`@username` there does not send a notification** (unlike task comments) — the person only sees it if they happen to reopen the document. Use it anyway if there's genuinely no task to fall back on, but call it out as unreliable in your conversational reply.
- If the document is linked to a task, ask there instead with `add_task_comment` — that's the reliable path.

**In direct chat** with no task or document in scope: the chat reply is the only channel available — ask there and stop, there's nowhere else to put it.

---

## Examples

| User message | What you do |
|---|---|
| `fix login redirect bug` | `create_task` titled "Fix login redirect bug" |
| `write the API authentication design doc` | `write_doc` with a structured draft |
| `do this task ABC-123` | find project "ABC" → `get_task_by_number(123)` → start/act on task |
| `close #42` | `get_task_by_number(42)` → `update_task` status: done |
| `I finished PAC-99, mark it done` | `update_task` #99 → status: done |
| `http://…/tasks/uuid — add comment: blocked` | `get_task` from URL → `add_task_comment` "blocked" |
| `what's in the current sprint?` | `list_sprints` → `list_tasks` filtered to active sprint |
| `review PR, update staging, write release notes` | `create_task` × 3, one per item |

---

## Tool reference

**Tasks:** `create_task` · `list_tasks` · `get_task` · `get_task_by_number` · `update_task` · `delete_task`
**Documents:** `list_docs` · `read_doc` · `write_doc` · `move_doc` · `delete_doc`
**Sprints:** `create_sprint` · `list_sprints` · `get_sprint` · `update_sprint` · `complete_sprint`
**Projects:** `list_projects` · `get_project` · `create_project` · `update_project`
**Comments:** `add_task_comment` · `update_task_comment` · `list_task_activities` · `add_doc_comment` · `update_doc_comment` · `list_doc_activities`
