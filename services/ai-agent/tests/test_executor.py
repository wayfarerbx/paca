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
    _make_reconciler,
    _SeenEvents,
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


# ─── _make_event_callback ─────────────────────────────────────────────────────
#
# These run a real asyncio loop on a background thread because the callbacks
# call asyncio.run_coroutine_threadsafe(...).result(...) — exactly how they're
# driven in production (the SDK invokes them synchronously from the executor
# thread, while the main event loop runs separately).


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
    relies on the class name, `.id`, `.source`, and `.model_dump_json()`."""

    def __init__(self, source: str = "agent", id: str | None = None) -> None:
        self.source = source
        self.id = id

    def model_dump_json(self) -> str:
        return json.dumps({"source": self.source})


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


def test_event_callback_persists_agent_reply(bg_loop, mock_persistence):
    """Regression coverage for https://github.com/Paca-AI/paca/issues/211 and
    #214: openhands-sdk's RemoteConversation never wires up token_callbacks
    (confirmed by reading openhands.sdk.conversation.impl.remote_conversation
    — the real constructor has no such parameter, it's swallowed by a
    catch-all **kwargs), so the event callback is the only path that ever
    persists the agent's reply."""
    trigger = _trigger()
    event_callback = _make_event_callback(trigger, bg_loop, _AtomicCounter(), _SeenEvents())

    event_callback(MessageEvent(source="agent"))

    mock_persistence.assert_awaited_once()
    kwargs = mock_persistence.call_args.kwargs
    assert kwargs["event_type"] == "MessageEvent"
    assert kwargs["event_source"] == "agent"
    assert kwargs["conversation_id"] == "conv-1"


def test_event_callback_skips_non_message_noise_events(bg_loop, mock_persistence):
    """Internal bookkeeping events must be skipped."""
    trigger = _trigger()
    event_callback = _make_event_callback(trigger, bg_loop, _AtomicCounter(), _SeenEvents())

    noise = type("ConversationStateUpdateEvent", (), {"source": "agent"})()
    event_callback(noise)

    mock_persistence.assert_not_awaited()


# ─── _make_reconciler ─────────────────────────────────────────────────────────


class _FakeEventsList:
    """Stand-in for RemoteEventsList — reconcile() + iteration only."""

    def __init__(self, events: list) -> None:
        self._events = events
        self.reconcile_calls = 0

    def reconcile(self) -> int:
        self.reconcile_calls += 1
        return 0

    def __iter__(self):
        return iter(self._events)


class _FakeConversation:
    def __init__(self, events: list) -> None:
        self.state = type("State", (), {"events": _FakeEventsList(events)})()


def test_reconciler_persists_events_missed_by_dead_ws_thread(bg_loop, mock_persistence):
    """Simulates the openhands-sdk WS-thread-died scenario: events the agent
    produced never reached the live callback (nothing called it), but they
    are visible over REST via events.reconcile(). The reconciler must push
    those through the normal persistence path exactly once."""
    trigger = _trigger()
    seen = _SeenEvents()
    reconcile = _make_reconciler(trigger, bg_loop, _AtomicCounter(), seen)

    conversation = _FakeConversation(
        [MessageEvent(source="agent", id="evt-1"), MessageEvent(source="agent", id="evt-2")]
    )

    reconcile(conversation)

    assert mock_persistence.await_count == 2
    persisted_ids = {call.kwargs["event_index"] for call in mock_persistence.call_args_list}
    assert len(persisted_ids) == 2  # each event got its own monotonic index


def test_reconciler_does_not_reprocess_events_already_seen(bg_loop, mock_persistence):
    """An event already delivered via the live WebSocket callback (and marked
    seen there) must not be persisted again just because reconcile() also
    surfaces it from the full REST history."""
    trigger = _trigger()
    seen = _SeenEvents()
    event_callback = _make_event_callback(trigger, bg_loop, _AtomicCounter(), seen)
    reconcile = _make_reconciler(trigger, bg_loop, _AtomicCounter(), seen)

    live_event = MessageEvent(source="agent", id="evt-live")
    event_callback(live_event)
    mock_persistence.assert_awaited_once()
    mock_persistence.reset_mock()

    conversation = _FakeConversation([live_event, MessageEvent(source="agent", id="evt-new")])
    reconcile(conversation)

    mock_persistence.assert_awaited_once()
    assert mock_persistence.call_args.kwargs["conversation_id"] == "conv-1"
