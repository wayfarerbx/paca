"""Conversation lifecycle status — mirrors the DB CHECK constraint on
agent_conversations.status (see services/api/migrations/000008_add_ai_agents.sql).
"""

from __future__ import annotations

from enum import StrEnum


class ConversationStatus(StrEnum):
    QUEUED = "queued"
    RUNNING = "running"
    PAUSED = "paused"
    FINISHED = "finished"
    FAILED = "failed"
    STOPPED = "stopped"

    def is_terminal(self) -> bool:
        """Whether this status ends the conversation's lifecycle.

        Terminal statuses cannot be resumed — a new conversation must be
        created instead (see agent_service.go's SendChatMessage, which only
        reuses a conversation when its status is PAUSED).
        """
        return self in (
            ConversationStatus.FINISHED,
            ConversationStatus.FAILED,
            ConversationStatus.STOPPED,
        )
