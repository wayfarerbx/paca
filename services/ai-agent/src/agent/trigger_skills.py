"""Deterministic per-trigger-type context, delivered as an always-active Skill.

Replaces the old task_trigger_prompt / doc_comment_trigger_prompt /
chat_trigger_prompt / description_write_trigger_prompt agent columns: the
content is now fixed (not user-editable) and delivered through the same
Skill mechanism as the rest of the agent's context, instead of raw string
concatenation into the system prompt.
"""

from __future__ import annotations

import logging

from openhands.sdk.context import Skill

logger = logging.getLogger(__name__)

_TASK_PROMPT = (
    "## Current invocation: task assignment or task comment\n\n"
    "You were triggered by a task assignment or a task comment @mention.\n\n"
    "### Mandatory: invoke a skill BEFORE any other action\n\n"
    "Before doing any work, making any status change, or calling any other "
    'tool, you MUST invoke one specialized skill using `invoke_skill(name="...")`. '
    "This is non-negotiable — the specialized skills contain step-by-step "
    "instructions you must follow, and reconstructing them from memory leads "
    "to incomplete or incorrect results.\n\n"
    "**Procedure:**\n"
    "1. Load the task via the Paca MCP tool (`get_task` / `get_task_by_number`).\n"
    "2. Based on the task's status, pick the right skill from `paca`'s "
    "Step 0.5 status-routing table and **invoke it immediately**:\n"
    '   - `invoke_skill(name="paca-do")` — execute/implement a ready task\n'
    '   - `invoke_skill(name="paca-clarify")` — task lacks acceptance criteria\n'
    '   - `invoke_skill(name="paca-breakdown")` — task is too large\n'
    '   - `invoke_skill(name="paca-estimate")` — size/estimate the task\n'
    '   - `invoke_skill(name="paca-sprint")` — plan into a sprint\n'
    '   - `invoke_skill(name="paca-test")` — verify/test a completed task\n'
    '   - `invoke_skill(name="paca-doc")` — write documentation\n'
    '   - `invoke_skill(name="paca-workflow")` — automate a process\n'
    "3. Follow the skill's full instructions step by step.\n\n"
    "**The ONLY exception:** A single trivial action with zero judgment — "
    'closing a task when someone explicitly said "close this", or adding '
    'a plain comment like "noted". If the request involves any implementation, '
    "planning, analysis, breakdown, estimation, testing, or documentation, "
    "you MUST invoke a skill first.\n\n"
    "If the comment is a simple question (not a work request), just answer "
    "it directly.\n\n"
    "If you get stuck without enough information, don't just say so in this "
    "conversation — the requester rarely reads it. Follow `paca`'s 'Asking "
    "for more information' guidance: add a task comment with an `@username` "
    "mention instead."
)

_DOC_COMMENT_PROMPT = (
    "## Current invocation: documentation comment\n\n"
    "You were triggered by a documentation comment @mention. There is no "
    "task and no status to route on here, so:\n\n"
    "1. Read the document that triggered the mention (`read_doc`) plus any "
    "related docs (`list_docs`) needed for context. Check `list_doc_activities` "
    "for prior discussion.\n"
    "2. If the request is to write or update documentation, use the "
    "`paca-doc` skill. Otherwise, just answer or act directly.\n"
    "3. Reply with `add_doc_comment`.\n\n"
    "If you need more information: `@username` in a doc comment does NOT "
    "send a notification (unlike task comments) — the person only sees it "
    "if they reopen this document. If this doc is linked to a task, ask "
    "there instead with `add_task_comment`, which does notify. Otherwise, "
    "`add_doc_comment` is the best you have."
)

_CHAT_PROMPT = (
    "## Current invocation: direct chat\n\n"
    "You were triggered by direct chat. Reply directly in the conversation, "
    "but that alone is not reliable — the user may not return to this exact "
    "session to see it or reply. So:\n\n"
    "1. If a task is referenced, load it, use `paca`'s task-status routing "
    "table to pick the specialized skill that fits, and invoke it (unless "
    "the request is one of the trivial single-action exceptions in `paca`'s "
    "Step 0); if a document is referenced and the request is to write or "
    "update it, invoke `paca-doc`.\n"
    "2. Whenever you need more information AND a task or document is in "
    "scope, don't just ask in the chat reply — also add a comment there "
    "with an `@username` mention (`add_task_comment` for a task; for a doc, "
    "prefer a linked task's comment if one exists, since doc-comment "
    "mentions don't notify). That's what actually reaches the user if they "
    "don't come back to this chat.\n"
    "3. If there's no task or document in scope, the chat reply is the only "
    "channel available — ask there and stop.\n"
    "4. Otherwise, do the work and reply directly in the conversation."
)

_DESCRIPTION_WRITE_PROMPT = (
    "## Current invocation: write task description\n\n"
    "You were triggered to write a description for one specific task. "
    "There is no one to ask, so act without waiting for input:\n\n"
    "1. Read the task's title, type, and any existing description.\n"
    "2. Read relevant project documentation for context.\n"
    "3. Write a clear, concise description with explicit acceptance "
    "criteria, then save it with `update_task` — don't hand this off to "
    "`paca-clarify`, which waits for answers before writing anything.\n"
    "4. Add a comment confirming the description was written and noting "
    "any assumptions made."
)


def get_trigger_skill(trigger_type: str, task_id: str | None) -> Skill | None:
    """Return the fixed, always-active Skill for this conversation's trigger.

    `trigger=None` is intentional: this is deterministic scaffolding chosen
    by the trigger type, not something the model should discover or invoke
    on its own via <available_skills>.
    """
    if trigger_type == "task_assigned":
        return Skill(name="paca-trigger-task-assigned", content=_TASK_PROMPT, trigger=None)
    if trigger_type == "comment_mention":
        # task_id is set for task-comment mentions; absent for doc-comment mentions.
        if task_id:
            return Skill(name="paca-trigger-task-assigned", content=_TASK_PROMPT, trigger=None)
        return Skill(name="paca-trigger-doc-comment", content=_DOC_COMMENT_PROMPT, trigger=None)
    if trigger_type == "chat_message":
        return Skill(name="paca-trigger-chat", content=_CHAT_PROMPT, trigger=None)
    if trigger_type == "description_write":
        return Skill(
            name="paca-trigger-description-write",
            content=_DESCRIPTION_WRITE_PROMPT,
            trigger=None,
        )
    return None


def append_trigger_skill(
    skills: list[Skill], trigger_type: str, task_id: str | None, conversation_id: str
) -> None:
    """Resolve this conversation's trigger skill and append it to `skills`, in place.

    The API rejects new agent skills named after a reserved trigger skill
    (see reservedSkillNames in services/api's agent_service.go), but that's
    enforced at skill-creation time — it can't see conversation state, so it
    can't stop a collision from happening. Guard here too, as the last line
    of defense: `AgentContext` hard-errors on ANY duplicate skill name, and a
    collision would otherwise fail every conversation of this trigger type
    for the agent. Skip (with a warning) rather than crash — an existing
    user-configured skill should win, same as it already does against a
    same-named *default* skill in `merge_skills_by_name`.
    """
    trigger_skill = get_trigger_skill(trigger_type, task_id)
    if trigger_skill is None:
        return
    if any(s.name == trigger_skill.name for s in skills):
        logger.warning(
            "Conversation %s: skipping trigger skill %r — a configured skill "
            "already uses this reserved name",
            conversation_id,
            trigger_skill.name,
        )
        return
    skills.append(trigger_skill)
