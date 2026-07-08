"""Internal REST endpoints for conversation control (proxied by services/api)."""

from __future__ import annotations

import logging
import secrets
from uuid import UUID

from fastapi import APIRouter, Depends, Header, HTTPException
from pydantic import BaseModel

from ..config import settings
from ..core.db import get_pool
from ..core.registry import active_conversations
from ..models.conversation_status import ConversationStatus
from ..repositories.conversation_repository import update_conversation_status

logger = logging.getLogger(__name__)


def _require_internal_key(x_internal_token: str = Header(default="")) -> None:
    """Dependency that rejects requests missing the correct internal API token."""
    if not secrets.compare_digest(x_internal_token, settings.internal_api_key):
        raise HTTPException(status_code=401, detail="Unauthorized")


router = APIRouter(prefix="/conversations", dependencies=[Depends(_require_internal_key)])


class MessageRequest(BaseModel):
    message: str


@router.get("/{id}")
async def get_conversation(id: UUID):
    pool = await get_pool()
    row = await pool.fetchrow(
        (
            "SELECT id, agent_id, project_id, status,"
            " trigger_type, created_at, updated_at"
            " FROM agent_conversations WHERE id = $1"
        ),
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


@router.post("/{id}/stop")
async def stop_conversation(id: UUID):
    conv = active_conversations.pop(str(id), None)
    if conv is not None and hasattr(conv, "close"):
        conv.close()  # type: ignore[union-attr]
    await update_conversation_status(str(id), ConversationStatus.STOPPED)
    return {"status": ConversationStatus.STOPPED.value}


@router.post("/{id}/message")
async def send_message(id: UUID, body: MessageRequest):
    conv = active_conversations.get(str(id))
    if conv is None:
        raise HTTPException(status_code=404, detail="No active conversation")
    if hasattr(conv, "send_message"):
        conv.send_message(body.message)  # type: ignore[union-attr]
    return {"status": "message_sent"}
