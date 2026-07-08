"""Valkey stream consumer worker loop."""

from __future__ import annotations

import asyncio
import logging
import time

from .agent.executor import run_conversation, teardown_paused_chat_sandbox
from .config import settings
from .core.registry import chat_sandboxes, pause_events, stop_events
from .core.streams import (
    ControlMessage,
    TriggerMessage,
    ack_trigger,
    ensure_consumer_group,
    read_triggers,
)
from .repositories.agent_repository import load_agent_config

logger = logging.getLogger(__name__)

_shutdown_event: asyncio.Event | None = None


async def _handle_control(msg: ControlMessage) -> None:
    """Dispatch a stop/pause/heartbeat control message to the running conversation."""
    cid = msg.conversation_id

    if msg.control_type == "agent.stop":
        # Full stop, unchanged from today: interrupt the in-flight turn (if
        # any) and tear the sandbox down for good.
        stop_event = stop_events.get(cid)
        if stop_event is not None:
            logger.info("Stopping conversation %s via stream control message", cid)
            stop_event.set()
            return
        # No in-flight run to signal — this is either a chat conversation
        # paused between turns (explicit stop, or the idle reaper firing
        # early via a control message) or a stop for a conversation this
        # replica never owned.
        if await teardown_paused_chat_sandbox(cid):
            logger.info("Stopped paused chat sandbox for conversation %s", cid)
        else:
            logger.warning(
                "Received stop for conversation %s but no active run or paused "
                "sandbox found on this replica",
                cid,
            )
        return

    if msg.control_type == "agent.pause":
        # Interrupt-only: pause the in-flight turn (if any); the sandbox is
        # never touched here. No-op (not a teardown) if nothing is running —
        # this is the assistant-UI stop button.
        pause_event = pause_events.get(cid)
        if pause_event is not None:
            logger.info("Pausing conversation %s via stream control message", cid)
            pause_event.set()
        else:
            logger.info(
                "Received pause for conversation %s but no active run found on this "
                "replica — no-op",
                cid,
            )
        return

    if msg.control_type == "agent.heartbeat":
        entry = chat_sandboxes.get(cid)
        # Cross-check project_id (cheap, in-memory) so a heartbeat can't keep
        # alive a sandbox from a conversation this caller wasn't scoped to.
        if entry is not None and entry.project_id == msg.project_id:
            entry.last_active_at = time.monotonic()
        return

    logger.warning("Unknown control type %r for conversation %s", msg.control_type, cid)


async def _process_trigger(msg: TriggerMessage | ControlMessage) -> None:
    if isinstance(msg, ControlMessage):
        await _handle_control(msg)
        return

    agent_config = await load_agent_config(msg.agent_id)
    if agent_config is None:
        logger.warning("Agent %s not found; dropping trigger %s", msg.agent_id, msg.stream_id)
        return

    await run_conversation(msg, agent_config)


async def run_worker() -> None:
    """Main worker loop — reads from the trigger stream and dispatches conversations."""
    global _shutdown_event
    _shutdown_event = asyncio.Event()

    await ensure_consumer_group()
    logger.info("AI-agent worker started (concurrency=%d)", settings.worker_concurrency)

    semaphore = asyncio.Semaphore(settings.worker_concurrency)
    tasks: set[asyncio.Task] = set()

    while not _shutdown_event.is_set():
        messages = await read_triggers(count=settings.worker_concurrency)
        for msg in messages:
            await semaphore.acquire()

            async def _guarded(m=msg):
                try:
                    await _process_trigger(m)
                    await ack_trigger(m.stream_id)
                except Exception as exc:
                    logger.exception("Unhandled error processing trigger %s: %s", m.stream_id, exc)
                finally:
                    semaphore.release()

            task = asyncio.create_task(_guarded())
            tasks.add(task)
            task.add_done_callback(tasks.discard)

    # Drain pending tasks on shutdown
    if tasks:
        await asyncio.gather(*tasks, return_exceptions=True)


def stop_worker() -> None:
    if _shutdown_event is not None:
        _shutdown_event.set()
