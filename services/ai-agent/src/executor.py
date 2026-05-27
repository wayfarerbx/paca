"""Agent conversation executor using the OpenHands SDK."""
from __future__ import annotations

import asyncio
import itertools
import logging
import os
import uuid
from pathlib import Path

import httpx
from openhands.sdk import LLM, Agent, AgentContext, Conversation, LocalWorkspace
from openhands.sdk.context import KeywordTrigger, Skill
from openhands.sdk.secret import SecretSource
from pydantic import SecretStr

from .config import settings
from .repo_adapter import AgentConfig, AgentSkillRow, AgentMCPServerRow
from .streams import TriggerMessage, publish_event
from . import db

logger = logging.getLogger(__name__)


class RepoTokenSecretSource(SecretSource):
    """Dynamically fetches a short-lived VCS token from the repository plugin."""

    def __init__(self, plugin_id: str, project_id: str) -> None:
        self.plugin_id = plugin_id
        self.project_id = project_id
        self._cached_token: str | None = None
        self._expires_at: float = 0

    def get_value(self) -> str:
        import time

        if self._cached_token and time.time() < self._expires_at - 60:
            return self._cached_token
        response = httpx.get(
            f"{settings.api_base_url}/internal/plugins/{self.plugin_id}/repo-token",
            headers={"X-Internal-Key": settings.internal_api_key},
            params={"project_id": self.project_id},
            timeout=10,
        )
        response.raise_for_status()
        data = response.json()
        self._cached_token = data["token"]
        self._expires_at = float(data["expires_at"])
        assert self._cached_token is not None
        return self._cached_token


def _build_skills(db_skills: list[AgentSkillRow]) -> list[Skill]:
    result = []
    for row in db_skills:
        if not row.is_enabled:
            continue
        trigger = KeywordTrigger(keywords=row.triggers) if row.triggers else None
        content = row.skill_content or ""
        result.append(Skill(name=row.skill_name, content=content, trigger=trigger))
    return result


def _build_mcp_config(db_servers: list[AgentMCPServerRow], agent_id: str) -> dict:
    servers: dict = {}

    # User-configured MCP servers from DB
    for row in db_servers:
        if not row.is_enabled:
            continue
        if row.transport == "stdio":
            servers[row.server_name] = {
                "command": row.command,
                "args": row.args,
                "env": row.env,
            }
        else:
            servers[row.server_name] = {
                "url": row.url,
                **({"auth": "oauth"} if row.transport == "oauth" else {}),
            }

    # Built-in Paca MCP — always injected last so it cannot be overridden or
    # removed by user-configured DB entries.  Requires PACA_API_KEY to be set.
    if settings.paca_api_key:
        servers["paca"] = {
            "command": "npx",
            "args": ["-y", "@paca-ai/paca-mcp"],
            "env": {
                "PACA_API_KEY": settings.paca_api_key,
                "PACA_API_URL": settings.api_base_url,
                "PACA_AGENT_ID": agent_id,
            },
        }

    return {"mcpServers": servers}


def _build_initial_prompt(trigger: TriggerMessage) -> str:
    lines = [trigger.message]
    if trigger.task_id:
        lines.append(f"\nTask ID: {trigger.task_id}")
    if trigger.comment_id:
        lines.append(f"Comment ID: {trigger.comment_id}")
    if trigger.chat_session_id:
        lines.append(f"Chat Session ID: {trigger.chat_session_id}")
    return "\n".join(lines)


def _make_event_callback(trigger: TriggerMessage, loop: asyncio.AbstractEventLoop):
    """Returns a synchronous callback invoked by the OpenHands SDK on each event."""
    idx = itertools.count()

    def callback(event) -> None:
        event_index = next(idx)
        payload = event.model_dump_json() if hasattr(event, "model_dump_json") else "{}"
        event_type = type(event).__name__
        event_source = getattr(event, "source", "agent")

        async def _persist():
            await db.insert_conversation_event(
                conversation_id=trigger.conversation_id,
                event_type=event_type,
                event_source=str(event_source),
                event_index=event_index,
                payload=payload,
            )
            await publish_event(
                {
                    "conversation_id": trigger.conversation_id,
                    "project_id": trigger.project_id,
                    "event_type": event_type,
                    "event_source": str(event_source),
                    "event_index": str(event_index),
                    "payload": payload,
                    "status": "running",
                }
            )

        # Schedule onto the main event loop (which owns the asyncpg pool)
        future = asyncio.run_coroutine_threadsafe(_persist(), loop)
        future.result()  # block executor thread until persisted

    return callback


async def run_conversation(trigger: TriggerMessage, agent_config: AgentConfig) -> None:
    """Execute a single agent conversation."""
    loop = asyncio.get_event_loop()

    logger.info("Starting conversation %s (agent=%s)", trigger.conversation_id, trigger.agent_id)
    await db.update_conversation_status(trigger.conversation_id, "running")

    # Providers that are in the OpenHands SDK verified list but are NOT
    # recognised by LiteLLM natively.  They use an OpenAI-compatible REST API
    # at a known base URL so we transparently inject that URL when the user
    # hasn't configured one explicitly.
    _OPENAI_COMPAT_PROVIDER_URLS: dict[str, str] = {
        "glm": "https://open.bigmodel.cn/api/paas/v4/",
        "nvidia": "https://integrate.api.nvidia.com/v1",
        "qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    }

    try:
        provider = agent_config.llm_provider
        llm_base_url = agent_config.llm_base_url

        # Auto-supply base_url for known OpenAI-compatible providers that
        # LiteLLM doesn't natively recognise, unless the user overrode it.
        if not llm_base_url and provider in _OPENAI_COMPAT_PROVIDER_URLS:
            llm_base_url = _OPENAI_COMPAT_PROVIDER_URLS[provider]

        # When a base_url is in use the endpoint is OpenAI-compatible;
        # LiteLLM requires the "openai/" prefix in that case.
        if llm_base_url:
            model_str = f"openai/{agent_config.llm_model}"
        else:
            model_str = f"{provider}/{agent_config.llm_model}"

        key_val = agent_config.llm_api_key_secret_ref or ""
        key_preview = (key_val[:8] + "...") if len(key_val) > 8 else ("<empty>" if not key_val else key_val)
        logger.info(
            "LLM config — model=%s base_url=%s api_key_prefix=%s",
            model_str, llm_base_url or "(none)", key_preview,
        )

        llm_kwargs: dict = {
            "model": model_str,
            "api_key": SecretStr(key_val),
        }
        if llm_base_url:
            llm_kwargs["base_url"] = llm_base_url

        llm = LLM(**llm_kwargs)

        skills = _build_skills(agent_config.skills)
        mcp_config = _build_mcp_config(agent_config.mcp_servers, agent_config.agent_id)

        agent_context = AgentContext(
            skills=skills,
            system_message_suffix=agent_config.system_prompt or "",
        )

        agent_kwargs: dict = {"llm": llm, "agent_context": agent_context}
        if mcp_config.get("mcpServers"):
            agent_kwargs["mcp_config"] = mcp_config

        agent = Agent(**agent_kwargs)

        persistence_dir = os.path.join(
            settings.conversation_persistence_root,
            agent_config.agent_id,
        )
        Path(persistence_dir).mkdir(parents=True, exist_ok=True)

        def _run_sync() -> None:
            with LocalWorkspace(working_dir=persistence_dir) as workspace:
                if trigger.repo_plugin_id and agent_config.can_clone_repos:
                    token_source = RepoTokenSecretSource(
                        trigger.repo_plugin_id, trigger.project_id
                    )
                    workspace.execute_command(
                        "git clone https://x-access-token:$GIT_TOKEN@placeholder /workspace/repo"
                    )

                conversation = Conversation(
                    agent=agent,
                    workspace=workspace,
                    conversation_id=uuid.UUID(trigger.conversation_id),
                    persistence_dir=persistence_dir,
                    callbacks=[_make_event_callback(trigger, loop)],
                    max_iteration_per_run=agent_config.max_iterations,
                )

                if trigger.repo_plugin_id:
                    token_source = RepoTokenSecretSource(
                        trigger.repo_plugin_id, trigger.project_id
                    )
                    conversation.update_secrets({"GIT_TOKEN": token_source})

                conversation.send_message(_build_initial_prompt(trigger))
                conversation.run()

        await asyncio.get_event_loop().run_in_executor(None, _run_sync)
        await db.update_conversation_status(trigger.conversation_id, "finished")

    except Exception as exc:
        logger.exception("Conversation %s failed: %s", trigger.conversation_id, exc)
        await db.update_conversation_status(trigger.conversation_id, "failed")
        await publish_event(
            {
                "conversation_id": trigger.conversation_id,
                "project_id": trigger.project_id,
                "event_type": "AgentErrorEvent",
                "event_source": "system",
                "event_index": "-1",
                "payload": f'{{"error": "{exc}"}}',
                "status": "failed",
            }
        )
