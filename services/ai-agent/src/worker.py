"""Valkey stream consumer worker loop."""
from __future__ import annotations

import asyncio
import logging

from .config import settings
from .executor import run_conversation
from .repo_adapter import load_agent_config
from .streams import ack_trigger, ensure_consumer_group, read_triggers

logger = logging.getLogger(__name__)

_running = True


async def _process_trigger(msg) -> None:
    agent_config = await load_agent_config(msg.agent_id)
    if agent_config is None:
        logger.warning("Agent %s not found; dropping trigger %s", msg.agent_id, msg.stream_id)
        return

    await run_conversation(msg, agent_config)


async def run_worker() -> None:
    """Main worker loop — reads from the trigger stream and dispatches conversations."""
    await ensure_consumer_group()
    logger.info("AI-agent worker started (concurrency=%d)", settings.worker_concurrency)

    semaphore = asyncio.Semaphore(settings.worker_concurrency)
    tasks: set[asyncio.Task] = set()

    while _running:
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
    global _running
    _running = False
