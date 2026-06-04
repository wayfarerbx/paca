"""Tests for TriggerMessage parsing from Valkey stream entries."""

import pytest

from src.core.streams import TriggerMessage


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
