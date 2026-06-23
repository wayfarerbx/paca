"""Tests for executor helpers."""

import asyncio
import json
import threading
from unittest.mock import AsyncMock

import pytest

from src.agent.executor import (
    _AtomicCounter,
    _build_project_context_suffix,
    _make_event_callback,
    _make_token_callback,
    _TurnState,
)
from src.core import streams as stream_store
from src.core.streams import TriggerMessage
from src.repositories import conversation_repository


def test_project_id_appears_in_suffix():
    result = _build_project_context_suffix("proj-123")
    assert "proj-123" in result


def test_suffix_instructs_agent_to_pass_project_id():
    result = _build_project_context_suffix("proj-abc")
    assert "projectId" in result


def test_suffix_discourages_list_projects_call():
    result = _build_project_context_suffix("proj-abc")
    assert "list_projects" in result


def test_different_project_ids_produce_different_suffixes():
    assert _build_project_context_suffix("proj-1") != _build_project_context_suffix("proj-2")


def test_turn_state_defaults_to_not_persisted():
    """The event callback should fall back to persisting the raw MessageEvent
    when the token callback never fired for this turn — which is always, for
    RemoteConversation (see _TurnState's docstring in executor.py)."""
    state = _TurnState()
    assert state.consume() is False


def test_turn_state_consume_reflects_token_persist():
    state = _TurnState()
    state.mark_token_persisted()
    assert state.consume() is True


def test_turn_state_consume_clears_flag_for_next_turn():
    state = _TurnState()
    state.mark_token_persisted()
    assert state.consume() is True
    # A second turn with no streaming output must not inherit the previous
    # turn's flag — otherwise its reply would be silently dropped too.
    assert state.consume() is False


# ─── _make_event_callback / _make_token_callback integration ─────────────────
#
# These exercise the real closures together rather than just _TurnState in
# isolation. They run a real asyncio loop on a background thread because the
# callbacks call asyncio.run_coroutine_threadsafe(...).result(...) — exactly
# how they're driven in production (the SDK invokes them synchronously from
# the executor thread, while the main event loop runs separately).


@pytest.fixture
def bg_loop():
    loop = asyncio.new_event_loop()
    thread = threading.Thread(target=loop.run_forever, daemon=True)
    thread.start()
    yield loop
    loop.call_soon_threadsafe(loop.stop)
    thread.join(timeout=5)
    loop.close()


@pytest.fixture
def mock_persistence(monkeypatch):
    insert_mock = AsyncMock()
    monkeypatch.setattr(conversation_repository, "insert_conversation_event", insert_mock)
    monkeypatch.setattr(stream_store, "publish_event", AsyncMock())
    monkeypatch.setattr(stream_store, "publish_realtime", AsyncMock())
    return insert_mock


class MessageEvent:
    """Stand-in for openhands.sdk.event.MessageEvent — the callback only
    relies on the class name, `.source`, and `.model_dump_json()`."""

    def __init__(self, source: str = "agent") -> None:
        self.source = source

    def model_dump_json(self) -> str:
        return json.dumps({"source": self.source})


class _FakeChoice:
    def __init__(self, content: str = "", finish_reason: str | None = None) -> None:
        self.delta = type("Delta", (), {"content": content, "reasoning_content": ""})()
        self.finish_reason = finish_reason


class _FakeStreamChunk:
    def __init__(self, *choices: _FakeChoice) -> None:
        self.choices = choices


def _trigger(conversation_id: str = "conv-1") -> TriggerMessage:
    return TriggerMessage(
        stream_id="1-1",
        trigger_type="chat_message",
        conversation_id=conversation_id,
        agent_id="agent-1",
        project_id="proj-1",
        task_id=None,
        comment_id=None,
        chat_session_id=None,
        message="hi",
        actor_member_id="member-1",
        repo_plugin_ids=[],
    )


def test_event_callback_persists_agent_reply_when_token_callback_never_fires(
    bg_loop, mock_persistence
):
    """Mirrors production under openhands-sdk's RemoteConversation: on_token
    is never invoked at all (token_callbacks isn't wired through for remote
    workspaces — see openhands.sdk.conversation.impl.remote_conversation).
    The event callback must persist the agent's reply itself rather than
    silently dropping it."""
    trigger = _trigger()
    turn_state = _TurnState()
    event_callback = _make_event_callback(trigger, bg_loop, _AtomicCounter(), turn_state)

    # on_token is intentionally never called here.
    event_callback(MessageEvent(source="agent"))

    mock_persistence.assert_awaited_once()
    kwargs = mock_persistence.call_args.kwargs
    assert kwargs["event_type"] == "MessageEvent"
    assert kwargs["event_source"] == "agent"
    assert kwargs["conversation_id"] == "conv-1"


def test_event_callback_skips_duplicate_when_token_callback_already_persisted(
    bg_loop, mock_persistence
):
    """When the token callback *does* manage to flush a finished message,
    the event callback must not also persist the SDK's own MessageEvent —
    that would duplicate the reply in the conversation."""
    trigger = _trigger()
    turn_state = _TurnState()
    on_token = _make_token_callback(trigger, bg_loop, _AtomicCounter(), turn_state)
    event_callback = _make_event_callback(trigger, bg_loop, _AtomicCounter(), turn_state)

    on_token(_FakeStreamChunk(_FakeChoice(content="Hello", finish_reason="stop")))
    mock_persistence.assert_awaited_once()
    mock_persistence.reset_mock()

    event_callback(MessageEvent(source="agent"))
    mock_persistence.assert_not_awaited()


def test_event_callback_still_skips_non_message_noise_events(bg_loop, mock_persistence):
    """Unrelated to the fallback logic: internal bookkeeping events must
    still be skipped regardless of turn_state."""
    trigger = _trigger()
    event_callback = _make_event_callback(trigger, bg_loop, _AtomicCounter(), _TurnState())

    noise = type("ConversationStateUpdateEvent", (), {"source": "agent"})()
    event_callback(noise)

    mock_persistence.assert_not_awaited()
