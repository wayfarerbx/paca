"""Internal REST endpoints for conversation control (proxied by services/api)."""
from __future__ import annotations

import asyncio
import logging
from uuid import UUID

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from ..db import get_pool, update_conversation_status

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/conversations")

# In-memory registry of active Conversation objects keyed by conversation_id str.
# Multi-replica note: in a multi-replica deployment, pair this with a Valkey key
# (paca:agent:active:{conversation_id}) to route control requests to the owning replica.
active_conversations: dict[str, object] = {}


class MessageRequest(BaseModel):
    message: str


@router.get("/{id}")
async def get_conversation(id: UUID):
    pool = await get_pool()
    row = await pool.fetchrow(
        "SELECT id, agent_id, project_id, status, trigger_type, created_at, updated_at FROM agent_conversations WHERE id = $1",
        str(id),
    )
    if row is None:
        raise HTTPException(status_code=404, detail="Conversation not found")
    return dict(row)


@router.get("/{id}/events")
async def get_conversation_events(id: UUID, offset: int = 0, limit: int = 100):
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT id, conversation_id, event_type, event_source, event_index, payload, created_at
        FROM agent_conversation_events
        WHERE conversation_id = $1
        ORDER BY event_index ASC
        LIMIT $2 OFFSET $3
        """,
        str(id),
        limit,
        offset,
    )
    return {"events": [dict(r) for r in rows]}


@router.post("/{id}/pause")
async def pause_conversation(id: UUID):
    conv = active_conversations.get(str(id))
    if conv is None:
        raise HTTPException(status_code=404, detail="No active conversation")
    if hasattr(conv, "pause"):
        conv.pause()  # type: ignore[union-attr]
    await update_conversation_status(str(id), "paused")
    return {"status": "paused"}


@router.post("/{id}/resume")
async def resume_conversation(id: UUID):
    conv = active_conversations.get(str(id))
    if conv is None:
        raise HTTPException(status_code=404, detail="No active or paused conversation on this replica")
    if hasattr(conv, "run"):
        asyncio.create_task(asyncio.to_thread(conv.run))  # type: ignore[union-attr]
    await update_conversation_status(str(id), "running")
    return {"status": "running"}


@router.post("/{id}/stop")
async def stop_conversation(id: UUID):
    conv = active_conversations.pop(str(id), None)
    if conv is not None and hasattr(conv, "close"):
        conv.close()  # type: ignore[union-attr]
    await update_conversation_status(str(id), "stopped")
    return {"status": "stopped"}


@router.post("/{id}/message")
async def send_message(id: UUID, body: MessageRequest):
    conv = active_conversations.get(str(id))
    if conv is None:
        raise HTTPException(status_code=404, detail="No active conversation")
    if hasattr(conv, "send_message"):
        conv.send_message(body.message)  # type: ignore[union-attr]
    return {"status": "message_sent"}
