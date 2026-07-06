"""Tests for TriggerMessage and ControlMessage parsing from Valkey stream entries."""

import pytest

from src.core.streams import ControlMessage, TriggerMessage


def _fields(**kwargs) -> dict[str, str]:
    base = {
        "trigger_type": "task_assigned",
        "conversation_id": "conv-1",
        "agent_id": "agent-1",
        "project_id": "proj-1",
        "message": "Do the thing",
        "actor_member_id": "member-1",
    }
    return {**base, **kwargs}


def test_full_fields_parsed_correctly():
    fields = _fields(
        task_id="task-99",
        comment_id="",
        chat_session_id="",
        repo_plugin_ids="plugin-a,plugin-b",
    )
    msg = TriggerMessage.from_stream_entry("1-1", fields)

    assert msg.stream_id == "1-1"
    assert msg.trigger_type == "task_assigned"
    assert msg.conversation_id == "conv-1"
    assert msg.agent_id == "agent-1"
    assert msg.project_id == "proj-1"
    assert msg.task_id == "task-99"
    assert msg.comment_id is None  # empty string → None
    assert msg.chat_session_id is None  # empty string → None
    assert msg.message == "Do the thing"
    assert msg.actor_member_id == "member-1"
    assert msg.repo_plugin_ids == ["plugin-a", "plugin-b"]


def test_missing_optional_fields_default_to_none_and_empty():
    msg = TriggerMessage.from_stream_entry("2-0", _fields())

    assert msg.task_id is None
    assert msg.comment_id is None
    assert msg.chat_session_id is None
    assert msg.message == "Do the thing"
    assert msg.repo_plugin_ids == []


def test_missing_actor_member_id_defaults_to_none():
    """Workflow-triggered assignments have no human actor, so the API omits
    this field entirely rather than sending an empty string."""
    fields = _fields()
    del fields["actor_member_id"]
    msg = TriggerMessage.from_stream_entry("2-1", fields)
    assert msg.actor_member_id is None


def test_empty_repo_plugin_ids_string_becomes_empty_list():
    msg = TriggerMessage.from_stream_entry("3-0", _fields(repo_plugin_ids=""))
    assert msg.repo_plugin_ids == []


def test_single_repo_plugin_id():
    msg = TriggerMessage.from_stream_entry("4-0", _fields(repo_plugin_ids="only-plugin"))
    assert msg.repo_plugin_ids == ["only-plugin"]


def test_multiple_repo_plugin_ids():
    msg = TriggerMessage.from_stream_entry("5-0", _fields(repo_plugin_ids="p1,p2,p3"))
    assert msg.repo_plugin_ids == ["p1", "p2", "p3"]


def test_missing_required_field_raises_key_error():
    fields = _fields()
    del fields["conversation_id"]
    with pytest.raises(KeyError):
        TriggerMessage.from_stream_entry("6-0", fields)


def test_comment_trigger_fields():
    fields = _fields(
        trigger_type="comment_mention",
        comment_id="cmt-42",
        task_id="",
    )
    msg = TriggerMessage.from_stream_entry("7-0", fields)

    assert msg.trigger_type == "comment_mention"
    assert msg.comment_id == "cmt-42"
    assert msg.task_id is None


# ─── ControlMessage tests ─────────────────────────────────────────────────────


def _control_fields(control_type: str = "agent.stop") -> dict[str, str]:
    return {
        "type": control_type,
        "conversation_id": "conv-99",
        "project_id": "proj-1",
    }


def test_control_stop_parsed():
    msg = ControlMessage.from_stream_entry("10-0", _control_fields("agent.stop"))
    assert msg.stream_id == "10-0"
    assert msg.control_type == "agent.stop"
    assert msg.conversation_id == "conv-99"
    assert msg.project_id == "proj-1"


def test_control_missing_conversation_id_raises():
    fields = _control_fields()
    del fields["conversation_id"]
    with pytest.raises(KeyError):
        ControlMessage.from_stream_entry("13-0", fields)
