# AI Agent — Default Skill Set

Every agent automatically gets Paca's default skill set, in addition to whatever skills the user configures for that agent (`agent_skills` table, edited from the agent's Skills tab). Defaults live under `services/ai-agent/src/skills/` and are bundled into the `ai-agent` Docker image by the existing `COPY src/ ./src/` line in its Dockerfile. They are loaded once per worker process (`builder.load_default_skills()`, cached) and merged with the agent's own skills at conversation start (`merge_skills_by_name`, in `executor.run_conversation`) — a user-configured skill wins on a name collision with a default.

## Two formats, two behaviors

- **`paca.md`** — a plain Markdown file (not named `SKILL.md`), Paca's baseline operating procedure. It has no `triggers:` frontmatter, so the OpenHands SDK treats it as *legacy format with `trigger=None`*: its full content is always injected into every conversation's system prompt, regardless of how the agent was invoked. It routes to the specialized skills below and includes a task-status → skill routing table (e.g. in-progress → `paca-do`, in-review → `paca-test`) so the model picks the right one once it reads the task via the Paca MCP tool.
- **`paca-do/`, `paca-clarify/`, `paca-breakdown/`, `paca-doc/`, `paca-epic/`, `paca-estimate/`, `paca-prioritize/`, `paca-sprint/`, `paca-test/`, `paca-workflow/`** — each a `<name>/SKILL.md` directory (the AgentSkills standard). These are *model-selectable*: listed by name and description in `<available_skills>`, full content read on demand (progressive disclosure). Each also declares `triggers: ["/<name>"]`, so typing e.g. `/paca-do #42` in a chat message or task comment auto-injects that skill's content via the SDK's own keyword matching — no custom slash-command parser needed.

`paca-setup` (from the repo-root `/skills/paca-setup`, used to wire the Paca MCP server into a Claude Code session) is intentionally **not** ported — the in-product agent always has its MCP server auto-configured (`builder.build_mcp_config`), so there's nothing to set up.

## Action-type context, not free-text prompts

The old per-agent `task_trigger_prompt` / `doc_comment_trigger_prompt` / `chat_trigger_prompt` / `description_write_trigger_prompt` columns are gone. The equivalent content now lives as fixed constants in `services/ai-agent/src/agent/trigger_skills.py`, and `executor.run_conversation` picks exactly one — based on the trigger type of the current conversation — and appends it to the skill list (via `trigger_skills.append_trigger_skill`) as an always-active (`trigger=None`) `Skill` named `paca-trigger-task-assigned` / `paca-trigger-doc-comment` / `paca-trigger-chat` / `paca-trigger-description-write`. This is deterministic scaffolding for the current conversation, not something users edit or the model discovers on its own — the API rejects any user-created skill using one of these four names (`reservedSkillNames` in `services/api`'s `agent_service.go`), and `append_trigger_skill` also skips (rather than crashes) if one somehow already exists, since `AgentContext` hard-errors on any duplicate skill name.

`prompt.build_initial_prompt` also prepends a plain-English "Action type: …" line to the first message, so the model has explicit context (task assignment vs. comment vs. chat vs. description-write) to reason about alongside whatever task status it reads via MCP.

## Adding or changing a default skill

Add or edit a `SKILL.md` (or `paca.md`) under `services/ai-agent/src/skills/`. No code change is needed — `load_default_skills()` re-scans the directory on the next process start. Avoid the `paca-trigger-*` prefix reserved for the fixed trigger-context skills in `trigger_skills.py` (the API also rejects it for user-created skills, in `agent_service.go`); `AgentContext` raises a hard error on any duplicate skill name.
