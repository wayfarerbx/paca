"""Agent conversation executor — orchestrates LLM, skills, MCP, and repo tools."""
from __future__ import annotations

import asyncio
import itertools
import logging
import uuid

import httpx
from openhands.sdk import Agent, AgentContext, Conversation
from openhands.tools import get_default_tools

from ..config import settings
from ..core import streams as stream_store
from ..core.streams import TriggerMessage
from ..models.agent import AgentConfig
from ..repositories import conversation_repository
from .builder import build_llm, build_mcp_config, build_skills
from .docker_workspace import docker_sandbox
from .prompt import build_initial_prompt
from .repo_tools import make_repository_tool_specs

logger = logging.getLogger(__name__)


# ─── Repository info helper ───────────────────────────────────────────────────


class RepoInfoSource:
    """Fetches linked repository list from the repository plugin."""

    def __init__(self, plugin_id: str, project_id: str) -> None:
        self.plugin_id = plugin_id
        self.project_id = project_id
        self.repositories: list[dict] = []
        self.clone_url: str | None = None

    def _refresh(self) -> None:
        url = (
            f"{settings.api_base_url}/api/v1/plugins/{self.plugin_id}"
            f"/projects/{self.project_id}/repositories"
        )
        response = httpx.get(url, headers={"X-API-Key": settings.paca_api_key}, timeout=10)
        response.raise_for_status()
        items = response.json().get("data", [])
        if not isinstance(items, list):
            items = []
        self.repositories = items
        self.clone_url = items[0].get("clone_url") if items else None

    def get_repositories(self) -> list[dict]:
        self._refresh()
        return self.repositories


def _gather_repo_sources(trigger: TriggerMessage) -> list[RepoInfoSource]:
    sources = []
    for plugin_id in trigger.repo_plugin_ids:
        source = RepoInfoSource(plugin_id, trigger.project_id)
        try:
            source._refresh()
            if source.repositories:
                sources.append(source)
        except Exception as exc:
            logger.warning("Failed to get repository info from plugin %s: %s", plugin_id, exc)
    return sources


# ─── Event callback ───────────────────────────────────────────────────────────


def _make_event_callback(trigger: TriggerMessage, loop: asyncio.AbstractEventLoop):
    """Return a synchronous callback invoked by the OpenHands SDK on each event."""
    idx = itertools.count()

    def callback(event) -> None:
        event_index = next(idx)
        payload = event.model_dump_json() if hasattr(event, "model_dump_json") else "{}"
        event_type = type(event).__name__
        event_source = getattr(event, "source", "agent")

        async def _persist():
            await conversation_repository.insert_conversation_event(
                conversation_id=trigger.conversation_id,
                event_type=event_type,
                event_source=str(event_source),
                event_index=event_index,
                payload=payload,
            )
            await stream_store.publish_event(
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

        future = asyncio.run_coroutine_threadsafe(_persist(), loop)
        try:
            future.result(timeout=10)
        except Exception as exc:
            logger.warning("Event persist failed for conversation %s: %s", trigger.conversation_id, exc)

    return callback


# ─── Main entry point ─────────────────────────────────────────────────────────


async def run_conversation(trigger: TriggerMessage, agent_config: AgentConfig) -> None:
    """Execute a single agent conversation end-to-end."""
    loop = asyncio.get_event_loop()
    logger.info("Starting conversation %s (agent=%s)", trigger.conversation_id, trigger.agent_id)
    await conversation_repository.update_conversation_status(trigger.conversation_id, "running")

    try:
        llm = build_llm(agent_config)
        skills = build_skills(agent_config.skills)
        mcp_config = build_mcp_config(agent_config.mcp_servers, agent_config.agent_id, trigger.project_id)

        system_suffix = agent_config.system_prompt or ""
        has_repos = len(trigger.repo_plugin_ids) > 0 and agent_config.can_clone_repos
        logger.info(
            "Conversation %s — repo_plugin_ids=%s can_clone_repos=%s has_repos=%s",
            trigger.conversation_id,
            trigger.repo_plugin_ids,
            agent_config.can_clone_repos,
            has_repos,
        )
        if has_repos:
            system_suffix += (
                "\n\n## IMPORTANT: Repository Access & Workflow\n"
                "This project has linked repositories. Follow these steps in order:\n\n"
                "1. Call list_repositories to see available repositories and get their IDs.\n"
                "2. Call clone_repository with the plugin_id and repo_id from step 1.\n"
                "   The repository will be cloned to /workspace/repo (the default target_dir).\n"
                "3. Create a new feature branch before making any changes:\n"
                "   git -C /workspace/repo checkout -b <branch-name>\n"
                "4. Make your code changes, then commit them:\n"
                "   git -C /workspace/repo add -A && git -C /workspace/repo commit -m '<message>'\n"
                "5. Call push_branch with the plugin_id, repo_id, and the branch name to publish the branch.\n"
                "6. Call create_pull_request with the plugin_id, repo_id, a descriptive title, "
                "the feature branch as head_branch, and the default branch as base_branch.\n\n"
                "Do NOT skip steps 5 and 6 — always push your branch and open a PR when finished."
            )

        agent_context = AgentContext(skills=skills, system_message_suffix=system_suffix)

        def _run_sync() -> None:
            with docker_sandbox(
                trigger.conversation_id,
                git_committer_name=agent_config.git_committer_name,
                git_committer_email=agent_config.git_committer_email,
            ) as workspace:

                agent_kwargs: dict = {"llm": llm, "agent_context": agent_context}
                if mcp_config.get("mcpServers"):
                    agent_kwargs["mcp_config"] = mcp_config

                if has_repos:
                    agent_kwargs["tools"] = get_default_tools() + make_repository_tool_specs(
                        trigger.project_id,
                        trigger.repo_plugin_ids,
                        trigger.task_id,
                        api_base_url=settings.api_base_url,
                        api_key=settings.paca_api_key,
                    )

                agent = Agent(**agent_kwargs)
                # RemoteWorkspace → RemoteConversation; persistence_dir is not
                # supported for remote conversations (state lives in the sandbox).
                conversation = Conversation(
                    agent=agent,
                    workspace=workspace,
                    conversation_id=uuid.UUID(trigger.conversation_id),
                    callbacks=[_make_event_callback(trigger, loop)],
                    max_iteration_per_run=agent_config.max_iterations,
                )

                all_repos_info = None
                if has_repos:
                    try:
                        repo_sources = _gather_repo_sources(trigger)
                        all_repos: list[dict] = []
                        for source in repo_sources:
                            for repo in source.repositories:
                                all_repos.append(
                                    {
                                        "plugin_id": source.plugin_id,
                                        "repo_id": repo["id"],
                                        "full_name": repo["full_name"],
                                        "owner": repo["owner"],
                                        "repo_name": repo["repo_name"],
                                        "clone_url": repo["clone_url"],
                                    }
                                )
                        if all_repos:
                            all_repos_info = all_repos
                    except Exception as exc:
                        logger.warning("Failed to gather repository info: %s", exc)

                conversation.send_message(build_initial_prompt(trigger, all_repos_info))
                conversation.run()

        await asyncio.get_event_loop().run_in_executor(None, _run_sync)
        await conversation_repository.update_conversation_status(trigger.conversation_id, "finished")

    except Exception as exc:
        logger.exception("Conversation %s failed: %s", trigger.conversation_id, exc)
        await conversation_repository.update_conversation_status(trigger.conversation_id, "failed")
