---
name: paca-do
description: Execute a Paca task end-to-end — reading context and acceptance criteria, doing the work (code, writing, research, review), updating task status, and commenting results. Use when asked to start, implement, complete, or work on a specific Paca task. Reads project docs first to understand the codebase and tech stack before acting.
triggers:
  - /paca-do
---

You are executing a task from Paca — reading it, understanding context, doing the work, and updating the record. Use Paca MCP tools throughout — never create local files for task records or documentation.

**If no task is specified**, call `list_tasks` filtered to in-progress tasks in the current sprint. Show them and ask which to work on.

---

## Step 1 — Load task and project context

1. Resolve the task reference from the user's message using `get_task_by_number` or `get_task`.
2. **If the task has no acceptance criteria**, stop and ask the user to clarify before starting — or offer to run `/paca-clarify` first. Starting work without a clear "done" condition wastes effort.
3. Call `list_docs` and search for documents relevant to this task — architecture, design specs, BDD scenarios, API references, integration guides. Read before writing any code or content; what's already decided shapes every implementation choice.
4. Call `list_task_activities` to read prior comments and implementation notes — someone may have already investigated this.
5. Note the acceptance criteria from the task description. These are your exit criteria.

## Step 2 — Mark in progress

1. Call `list_task_statuses` to find the "in progress" status.
2. Call `update_task` to set the status. (No confirmation needed — this is a lightweight, reversible status change.)
3. Call `add_task_comment`: "Starting work on this task."

## Step 3 — Do the work

Execute based on the task type:

- **Code task**: find the relevant source files, read existing tests to understand the expected behavior, implement the change, run the test suite. If you need to understand what "in scope" looks like, the BDD scenarios you read in Step 1 are authoritative.
- **Writing task**: draft the content in the response, or create/update a Paca document via `write_doc`. Never write to a local file.
- **Research / investigation task**: investigate, write findings as a comment via `add_task_comment` or as a Paca doc, then update the task description with the conclusions.
- **Review task**: analyse the artifact (PR, document, design), post a structured review as `add_task_comment`.

If you discover a blocker or a genuine sub-task that wasn't anticipated, create it in Paca with `create_task` (reference the parent: `Blocked by #<parent>`). Don't silently skip or work around it.

### Code tasks with a linked repository — push and create a PR (MANDATORY)

If this task involves code changes AND the project has a linked repository, completing the work means publishing it — changes left only in the sandbox are NOT done. Follow ALL steps:

1. **Clone and branch first** (before editing):
   - `list_repositories` → note the `plugin_id` and `repo_id`.
   - `clone_repository` (clones to /workspace/repo).
   - Create and switch to a feature branch: `git checkout -b feat/<task-number>-<short-description>`.

2. **Make your changes** in the cloned repo. Verify with tests if available.

3. **Commit:**
   ```
   git add -A
   git commit -m '<type>: <concise message>'
   ```

4. **Push the branch:** Call `push_branch` with the `plugin_id` and `repo_id` from step 1. Do NOT skip this — without a push, the remote has no branch for the PR.

5. **Create a pull request:** Call the repository plugin's PR tool (e.g. `github_create_pull_request` for GitHub repos). This is required — a pushed branch without a PR is unfinished work. Pass:
   - `projectId`, `taskId`, `repoId`
   - `title` — a clear commit-style title
   - `head_branch` — your feature branch name
   - `base_branch` — the repository's default branch (e.g. `main`)
   - `body` — a summary of what changed and why, referencing the task

6. **Report the PR URL** in the task comment (Step 4).

If any step fails (push rejected, token error, etc.), report it in a task `add_task_comment` rather than silently moving on.

## Step 4 — Update and close

1. Call `add_task_comment` with a completion summary: what was done, what changed, any known caveats or follow-up needed.
2. If any project documentation was affected (README, architecture doc, API reference), update the relevant Paca document with `write_doc`. Never write new docs as local files.
3. Call `update_task` to set the status to done (or the next stage — e.g. "review" — if your workflow has one).

**What's next:** Consider running `/paca-test #<number>` to verify the implementation against acceptance criteria.

Report back: task number, title, summary of what was done, and any new tasks or docs created. If you pushed code and opened a PR, include the PR URL.

---

## Tool reference

**Tasks:** `get_task` · `get_task_by_number` · `update_task` · `create_task` · `list_task_statuses`
**Comments:** `add_task_comment` · `list_task_activities`
**Documents:** `list_docs` · `read_doc` · `write_doc`
**Projects:** `list_projects`
**Repositories (built-in tools):** `list_repositories` · `clone_repository` · `push_branch`
**GitHub PRs/branches (MCP plugin):** `github_create_pull_request` · `github_list_task_prs` · `github_create_branch` · `github_list_task_branches`
