"""Valkey stream consumer worker loop."""

from __future__ import annotations

import asyncio
import logging

from .agent.executor import run_conversation
from .config import settings
from .core.registry import stop_events
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
    """Dispatch a stop control message to the running conversation."""
    cid = msg.conversation_id

    if msg.control_type == "agent.stop":
        stop_event = stop_events.get(cid)
        if stop_event is not None:
            logger.info("Stopping conversation %s via stream control message", cid)
            stop_event.set()
        else:
            logger.warning(
                "Received stop for conversation %s but no active run found on this replica", cid
            )
    else:
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
