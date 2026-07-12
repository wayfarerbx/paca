"""Tests for worker._handle_control's stop/pause/heartbeat dispatch."""

import threading
from unittest.mock import AsyncMock

import pytest

import src.worker as worker
from src.core.registry import ChatSandboxState, chat_sandboxes, pause_events, stop_events
from src.core.streams import ControlMessage


@pytest.fixture(autouse=True)
def _clear_registry():
    """All three registries are module-level dicts shared across tests."""
    stop_events.clear()
    pause_events.clear()
    chat_sandboxes.clear()
    yield
    stop_events.clear()
    pause_events.clear()
    chat_sandboxes.clear()


def _control(control_type: str, conversation_id: str = "conv-1", project_id: str = "proj-1"):
    return ControlMessage(
        stream_id="1-1",
        control_type=control_type,
        conversation_id=conversation_id,
        project_id=project_id,
    )


def _fake_sandbox_state(project_id: str = "proj-1") -> ChatSandboxState:
    return ChatSandboxState(
        handle=object(),  # type: ignore[arg-type]
        sdk_conversation_id="sdk-conv-xyz",
        project_id=project_id,
        last_active_at=0.0,
    )


# ─── agent.stop (full teardown, unchanged from before) ─────────────────────────


async def test_stop_sets_stop_event_when_run_in_flight(monkeypatch):
    teardown_mock = AsyncMock(return_value=True)
    monkeypatch.setattr(worker, "teardown_paused_chat_sandbox", teardown_mock)
    stop_event = threading.Event()
    stop_events["conv-1"] = stop_event

    await worker._handle_control(_control("agent.stop"))

    assert stop_event.is_set()
    teardown_mock.assert_not_called()


async def test_stop_tears_down_paused_sandbox_when_not_in_flight(monkeypatch):
    teardown_mock = AsyncMock(return_value=True)
    monkeypatch.setattr(worker, "teardown_paused_chat_sandbox", teardown_mock)

    await worker._handle_control(_control("agent.stop"))

    teardown_mock.assert_awaited_once_with("conv-1")


# ─── agent.pause (interrupt-only) ──────────────────────────────────────────────


async def test_pause_sets_pause_event_when_run_in_flight():
    pause_event = threading.Event()
    pause_events["conv-1"] = pause_event

    await worker._handle_control(_control("agent.pause"))

    assert pause_event.is_set()


async def test_pause_does_not_fall_through_to_teardown_when_not_in_flight(monkeypatch):
    """The lightweight pause must be a no-op when nothing is running — unlike
    the old overloaded stop behavior, it must never tear down a paused
    sandbox."""
    # worker.py does `from .agent.executor import ... teardown_paused_chat_sandbox`,
    # so the name to patch is the one bound in worker's own namespace.
    teardown_mock = AsyncMock(return_value=True)
    monkeypatch.setattr(worker, "teardown_paused_chat_sandbox", teardown_mock)
    chat_sandboxes["conv-1"] = _fake_sandbox_state()

    await worker._handle_control(_control("agent.pause"))

    teardown_mock.assert_not_called()
    assert "conv-1" in chat_sandboxes


# ─── agent.heartbeat ────────────────────────────────────────────────────────────


async def test_heartbeat_refreshes_last_active_at():
    chat_sandboxes["conv-1"] = _fake_sandbox_state(project_id="proj-1")

    await worker._handle_control(_control("agent.heartbeat", project_id="proj-1"))

    assert chat_sandboxes["conv-1"].last_active_at > 0.0


async def test_heartbeat_ignored_when_project_id_mismatches():
    chat_sandboxes["conv-1"] = _fake_sandbox_state(project_id="proj-1")

    await worker._handle_control(_control("agent.heartbeat", project_id="proj-OTHER"))

    assert chat_sandboxes["conv-1"].last_active_at == 0.0


async def test_heartbeat_no_op_when_sandbox_absent():
    # Should not raise even though no sandbox is registered for this replica.
    await worker._handle_control(_control("agent.heartbeat"))


# ─── unknown control type ──────────────────────────────────────────────────────


async def test_unknown_control_type_logs_and_does_not_raise(caplog):
    await worker._handle_control(_control("agent.something_else"))
    assert "Unknown control type" in caplog.text
