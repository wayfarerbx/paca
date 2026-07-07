"""Tests for deterministic per-trigger-type context (src/agent/trigger_skills.py)."""

from openhands.sdk import AgentContext
from openhands.sdk.context import Skill

from src.agent.trigger_skills import append_trigger_skill, get_trigger_skill

# ─── get_trigger_skill ────────────────────────────────────────────────────────


def test_task_assigned_returns_trigger_task_assigned():
    skill = get_trigger_skill("agent.task_assigned", task_id=None)
    assert skill is not None
    assert skill.name == "paca-trigger-task-assigned"
    assert skill.trigger is None


def test_task_comment_mention_returns_trigger_task_assigned():
    skill = get_trigger_skill("agent.comment_mention", task_id="task-1")
    assert skill is not None
    assert skill.name == "paca-trigger-task-assigned"


def test_doc_comment_mention_returns_trigger_doc_comment():
    skill = get_trigger_skill("agent.comment_mention", task_id=None)
    assert skill is not None
    assert skill.name == "paca-trigger-doc-comment"


def test_chat_message_returns_trigger_chat():
    skill = get_trigger_skill("agent.chat_message", task_id=None)
    assert skill is not None
    assert skill.name == "paca-trigger-chat"


def test_description_write_returns_trigger_description_write():
    skill = get_trigger_skill("agent.description_write", task_id=None)
    assert skill is not None
    assert skill.name == "paca-trigger-description-write"


def test_unknown_trigger_type_returns_none():
    assert get_trigger_skill("agent.something_else", task_id=None) is None


def test_all_trigger_skills_are_always_active():
    """`trigger=None` is load-bearing: these are chosen deterministically by
    trigger type, not something the model should discover via <available_skills>."""
    for trigger_type, task_id in [
        ("agent.task_assigned", None),
        ("agent.comment_mention", "task-1"),
        ("agent.comment_mention", None),
        ("agent.chat_message", None),
        ("agent.description_write", None),
    ]:
        skill = get_trigger_skill(trigger_type, task_id)
        assert skill.trigger is None


def test_all_trigger_skills_are_paca_prefixed():
    """The paca- prefix groups these with the rest of the default skill set
    (paca, paca-do, paca-doc, ...) so the reserved-name check on the API side
    (agent_service.go) and a human skimming an agent's skill list can both
    recognize them as internal at a glance."""
    for trigger_type, task_id in [
        ("agent.task_assigned", None),
        ("agent.comment_mention", "task-1"),
        ("agent.comment_mention", None),
        ("agent.chat_message", None),
        ("agent.description_write", None),
    ]:
        skill = get_trigger_skill(trigger_type, task_id)
        assert skill.name.startswith("paca-")


# ─── append_trigger_skill ─────────────────────────────────────────────────────


def test_append_trigger_skill_appends_when_no_collision():
    skills: list[Skill] = [Skill(name="paca", content="...", trigger=None)]
    append_trigger_skill(skills, "agent.chat_message", None, "conv-1")
    assert [s.name for s in skills] == ["paca", "paca-trigger-chat"]


def test_append_trigger_skill_is_noop_for_unknown_trigger_type():
    skills: list[Skill] = [Skill(name="paca", content="...", trigger=None)]
    append_trigger_skill(skills, "agent.something_else", None, "conv-1")
    assert [s.name for s in skills] == ["paca"]


def test_append_trigger_skill_skips_on_name_collision():
    """Regression test: a user-configured skill can legally be named
    'paca-trigger-chat' (nothing stops it client- or server-side beyond the
    API's reserved-name check, which is a separate, defense-in-depth layer —
    see agent_service.go). If it collides with this conversation's trigger
    skill, the user's skill must win, not crash the conversation."""
    user_skill = Skill(name="paca-trigger-chat", content="user content", trigger=None)
    skills: list[Skill] = [user_skill]

    append_trigger_skill(skills, "agent.chat_message", None, "conv-1")

    assert len(skills) == 1
    assert skills[0] is user_skill
    assert skills[0].content == "user content"


def test_append_trigger_skill_collision_does_not_crash_agent_context():
    """End-to-end confirmation of the fix: constructing AgentContext with the
    result of append_trigger_skill must never raise, even on a collision.
    (AgentContext raises ValueError on any duplicate skill name — this is the
    exact crash this function exists to prevent.)"""
    skills: list[Skill] = [Skill(name="paca-trigger-chat", content="user content", trigger=None)]
    append_trigger_skill(skills, "agent.chat_message", None, "conv-1")

    AgentContext(skills=skills, system_message_suffix="")  # must not raise
