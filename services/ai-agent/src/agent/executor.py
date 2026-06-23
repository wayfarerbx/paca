"""Agent conversation executor — orchestrates LLM, skills, MCP, and repo tools."""

from __future__ import annotations

import asyncio
import json
import logging
import threading
import time

import httpx
from openhands.sdk import Agent, AgentContext, Conversation
from openhands.sdk.conversation.state import ConversationExecutionStatus
from openhands.sdk.conversation.visualizer import ConversationVisualizerBase
from openhands.tools import get_default_tools

from ..config import settings
from ..core import streams as stream_store
from ..core.registry import active_conversations, stop_events
from ..core.streams import TriggerMessage
from ..models.agent import AgentConfig
from ..repositories import conversation_repository
from .builder import build_llm, build_mcp_config, build_skills
from .docker_workspace import docker_sandbox
from .prompt import build_initial_prompt
from .repo_tools import make_repository_tool_specs

logger = logging.getLogger(__name__)

_DONE_STATUSES = frozenset(
    {
        ConversationExecutionStatus.FINISHED,
        ConversationExecutionStatus.ERROR,
        ConversationExecutionStatus.STUCK,
    }
)

_ERROR_STATUSES = frozenset(
    {
        ConversationExecutionStatus.ERROR,
        ConversationExecutionStatus.STUCK,
    }
)


def _get_conversation_error_detail(conversation) -> str | None:
    """Extract the error detail from a conversation's ConversationErrorEvent, if any."""
    try:
        from openhands.sdk.event.conversation_error import ConversationErrorEvent

        events = conversation.state.events
        for event in reversed(list(events)):
            if isinstance(event, ConversationErrorEvent):
                code = (event.code or "").strip()
                detail = (event.detail or "").strip()
                if code and detail:
                    return f"{code}: {detail}"
                return detail or code or None
    except Exception:
        pass
    return None


def _wait_for_done_or_stop(
    conversation,
    stop_event: threading.Event,
    poll_interval: float = 2.0,
    timeout: float = 3600.0,
) -> tuple[bool, bool]:
    """Poll the remote conversation until it finishes or a stop is signaled.

    Returns (stopped, errored):
      stopped  — True if the loop exited because a stop was requested.
      errored  — True if the conversation ended with ERROR or STUCK status.
    """
    start = time.monotonic()
    while True:
        # stop_event.wait() blocks for up to poll_interval seconds, returning
        # True immediately if the event is already set.
        if stop_event.wait(timeout=poll_interval):
            try:
                conversation.pause()
            except Exception as exc:
                logger.warning("Failed to pause conversation on stop request: %s", exc)
            return True, False

        if time.monotonic() - start > timeout:
            logger.warning("Conversation polling timed out after %.0f seconds", timeout)
            return False, False

        try:
            status = conversation.state.execution_status
            if status in _DONE_STATUSES:
                errored = status in _ERROR_STATUSES
                if errored:
                    detail = _get_conversation_error_detail(conversation)
                    logger.error(
                        "Conversation ended with status %s — %s",
                        status.value,
                        detail or "no detail available",
                    )
                return False, errored
        except Exception as exc:
            logger.debug("Failed to read conversation execution status: %s", exc)


# ─── Custom visualizer ────────────────────────────────────────────────────────


class _QuietVisualizer(ConversationVisualizerBase):
    """No-op visualizer that silently accepts all event types.

    The SDK's DefaultConversationVisualizer emits a WARNING for every event
    type absent from its EVENT_VISUALIZATION_CONFIG (e.g. StreamingDeltaEvent).
    Since this service handles all events via callbacks / token_callbacks, we
    replace the default visualizer with this no-op to eliminate the noise.
    """

    def on_event(self, event) -> None:  # noqa: ANN001
        pass  # all event processing is done in the conversation callbacks


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


# ─── Shared event index ───────────────────────────────────────────────────────


class _AtomicCounter:
    """Thread-safe monotonic counter shared across event and token callbacks."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._value = 0

    def next(self) -> int:
        with self._lock:
            v = self._value
            self._value += 1
            return v


class _TurnState:
    """Tracks whether _make_token_callback already persisted the agent's
    reply for the in-flight turn.

    openhands.sdk.conversation.impl.RemoteConversation accepts a
    token_callbacks kwarg but never wires it to anything — Conversation()
    always builds a RemoteConversation here since docker_sandbox() always
    yields a RemoteWorkspace, so on_token is never actually invoked.
    _make_event_callback consults this flag to fall back to persisting the
    SDK's own MessageEvent instead, so the reply is never silently dropped.
    See https://github.com/Paca-AI/paca/issues/211 and #214.
    """

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._token_persisted = False

    def mark_token_persisted(self) -> None:
        with self._lock:
            self._token_persisted = True

    def consume(self) -> bool:
        """Return True if the token callback persisted this turn, clearing
        the flag for the next one."""
        with self._lock:
            was_persisted = self._token_persisted
            self._token_persisted = False
            return was_persisted


# ─── Event callback ───────────────────────────────────────────────────────────


def _make_event_callback(
    trigger: TriggerMessage,
    loop: asyncio.AbstractEventLoop,
    counter: _AtomicCounter,
    turn_state: _TurnState,
):
    """Return a synchronous callback invoked by the OpenHands SDK on each complete event.

    Agent MessageEvents are normally skipped here in favor of
    _make_token_callback, which captures the LLM's text output directly from
    the streaming response for cleaner content without SDK wrapper fields —
    when on_token actually gets invoked (see _TurnState). All other events
    (actions, observations, system messages) are saved normally.
    """

    def callback(event) -> None:
        event_type = type(event).__name__
        event_source = str(getattr(event, "source", "agent"))

        # Events the frontend never renders — skip to avoid wasting event_index
        # slots and polluting the paginated events list.
        if event_type in {
            "StreamingDeltaEvent",  # raw streaming chunks — only the final message matters
            "ConversationStateUpdateEvent",  # internal iteration bookkeeping
            "SystemPromptEvent",  # system prompt echo — not shown to user
            "ConversationErrorEvent",  # SDK error signal — surfaced via conversation status
        }:
            return
        # Agent text responses are normally captured by token_callbacks with
        # richer streaming data; skip them here to avoid duplicate DB rows —
        # unless the token callback never actually persisted this turn (it
        # never fires at all for RemoteConversation — see _TurnState), in
        # which case fall through and persist this event instead of silently
        # dropping the reply.
        if event_type == "MessageEvent" and event_source == "agent" and turn_state.consume():
            return

        event_index = counter.next()
        payload = event.model_dump_json() if hasattr(event, "model_dump_json") else "{}"

        async def _persist():
            await conversation_repository.insert_conversation_event(
                conversation_id=trigger.conversation_id,
                event_type=event_type,
                event_source=event_source,
                event_index=event_index,
                payload=payload,
            )
            await stream_store.publish_event(
                {
                    "conversation_id": trigger.conversation_id,
                    "project_id": trigger.project_id,
                    "event_type": event_type,
                    "event_source": event_source,
                    "event_index": str(event_index),
                    "payload": payload,
                    "status": "running",
                }
            )
            await stream_store.publish_realtime(
                project_id=trigger.project_id,
                conversation_id=trigger.conversation_id,
                event_type=f"agent.{event_type.lower()}",
            )

        future = asyncio.run_coroutine_threadsafe(_persist(), loop)
        try:
            future.result(timeout=10)
        except Exception as exc:
            logger.warning(
                "Event persist failed for conversation %s: %s",
                trigger.conversation_id,
                exc,
            )

    return callback


# ─── Token (streaming) callback ───────────────────────────────────────────────


def _make_token_callback(
    trigger: TriggerMessage,
    loop: asyncio.AbstractEventLoop,
    counter: _AtomicCounter,
    turn_state: _TurnState,
):
    """Return a token callback that accumulates streaming LLM chunks into complete
    messages and persists each finished message as a MessageEvent.

    The OpenHands SDK calls this once per streaming chunk.  When finish_reason
    is set the accumulated content is flushed to the database and a realtime
    pub/sub notification is published so WebSocket clients update immediately.
    """
    lock = threading.Lock()
    parts_content: list[str] = []
    parts_reasoning: list[str] = []

    def on_token(stream) -> None:
        for choice in stream.choices:
            delta = choice.delta
            finish_reason = getattr(choice, "finish_reason", None)

            content_chunk = getattr(delta, "content", None) or ""
            reasoning_chunk = getattr(delta, "reasoning_content", None) or ""

            with lock:
                if content_chunk:
                    parts_content.append(content_chunk)
                if reasoning_chunk:
                    parts_reasoning.append(reasoning_chunk)

                if finish_reason:
                    full_content = "".join(parts_content)
                    full_reasoning = "".join(parts_reasoning)
                    parts_content.clear()
                    parts_reasoning.clear()

                    if not full_content and not full_reasoning:
                        return

                    event_index = counter.next()
                    payload_obj: dict = {"content": full_content, "source": "agent"}
                    if full_reasoning:
                        payload_obj["reasoning_content"] = full_reasoning
                    payload_str = json.dumps(payload_obj)

                    async def _persist(idx=event_index, p=payload_str):
                        await conversation_repository.insert_conversation_event(
                            conversation_id=trigger.conversation_id,
                            event_type="MessageEvent",
                            event_source="agent",
                            event_index=idx,
                            payload=p,
                        )
                        await stream_store.publish_event(
                            {
                                "conversation_id": trigger.conversation_id,
                                "project_id": trigger.project_id,
                                "event_type": "MessageEvent",
                                "event_source": "agent",
                                "event_index": str(idx),
                                "payload": p,
                                "status": "running",
                            }
                        )
                        await stream_store.publish_realtime(
                            project_id=trigger.project_id,
                            conversation_id=trigger.conversation_id,
                            event_type="agent.messageevent",
                        )

                    future = asyncio.run_coroutine_threadsafe(_persist(), loop)
                    try:
                        future.result(timeout=10)
                        turn_state.mark_token_persisted()
                    except Exception as exc:
                        logger.warning(
                            "Token callback persist failed for conversation %s: %s",
                            trigger.conversation_id,
                            exc,
                        )

    return on_token


def _build_project_context_suffix(project_id: str) -> str:
    return (
        f"\n\n## Current Project Context\n"
        f"You are working inside project `{project_id}`.\n"
        "**Always pass this value as `projectId` in every MCP tool call** "
        "that requires it — never ask the user for the project ID and "
        "never call `list_projects` to find it.\n"
    )


# ─── Main entry point ─────────────────────────────────────────────────────────


async def run_conversation(trigger: TriggerMessage, agent_config: AgentConfig) -> None:
    """Execute a single agent conversation end-to-end."""
    loop = asyncio.get_event_loop()
    counter = _AtomicCounter()
    turn_state = _TurnState()
    stop_event = threading.Event()
    logger.info("Starting conversation %s (agent=%s)", trigger.conversation_id, trigger.agent_id)
    await conversation_repository.update_conversation_status(trigger.conversation_id, "running")

    # Register the stop event so the worker can signal us via stream control messages.
    stop_events[trigger.conversation_id] = stop_event

    try:
        llm = build_llm(agent_config)
        skills = build_skills(agent_config.skills)
        mcp_config = build_mcp_config(
            agent_config.mcp_servers, agent_config.agent_id, trigger.project_id
        )

        system_suffix = agent_config.system_prompt or ""

        # Inject current project context so the AI never needs to ask for the
        # project ID or call list_projects to discover it.
        system_suffix += _build_project_context_suffix(trigger.project_id)

        # Documentation workflow — always read project docs first, always write to Paca.
        system_suffix += (
            "\n\n## IMPORTANT: Documentation Workflow\n"
            f"This project's documentation is managed in Paca"
            f" (project ID: `{trigger.project_id}`).\n\n"
            "**Before starting any task**, read the project documentation:\n"
            f"1. Call `list_docs` with `projectId='{trigger.project_id}'`"
            " to see the full documentation tree.\n"
            "2. Call `read_doc` on relevant documents to understand"
            " the project context before proceeding.\n\n"
            "**When writing documentation**, always use the Paca MCP tools"
            " — never create local markdown files:\n"
            "- Call `list_docs` to check whether a document already exists at the intended path.\n"
            "- If it exists: `write_doc` will update it automatically.\n"
            "- If it does not exist: `write_doc` will create it"
            " and any missing folders automatically.\n"
            "- Use paths like `'Architecture/API Design'` — folder structure is handled for you.\n"
        )

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
                "5. Call push_branch with the plugin_id, repo_id,"
                " and the branch name to publish the branch.\n"
                "6. Call create_pull_request with the plugin_id, repo_id, a descriptive title, "
                "the feature branch as head_branch, and the default branch as base_branch.\n\n"
                "Do NOT skip steps 5 and 6 — always push your branch and open a PR when finished."
            )

        # Append the trigger-specific prompt last so it takes precedence.
        # These prompts are stored on the agent and sourced from TRIGGER_PROMPTS
        # defined in the frontend (apps/web/src/lib/agent-api.ts).
        if trigger.trigger_type == "agent.task_assigned":
            if agent_config.task_trigger_prompt:
                system_suffix += "\n\n" + agent_config.task_trigger_prompt
        elif trigger.trigger_type == "agent.comment_mention":
            # task_id is set for task-comment mentions; absent for doc-comment mentions.
            if trigger.task_id:
                if agent_config.task_trigger_prompt:
                    system_suffix += "\n\n" + agent_config.task_trigger_prompt
            else:
                if agent_config.doc_comment_trigger_prompt:
                    system_suffix += "\n\n" + agent_config.doc_comment_trigger_prompt
        elif trigger.trigger_type == "agent.chat_message":
            if agent_config.chat_trigger_prompt:
                system_suffix += "\n\n" + agent_config.chat_trigger_prompt
        elif trigger.trigger_type == "agent.description_write":
            if agent_config.description_write_trigger_prompt:
                system_suffix += "\n\n" + agent_config.description_write_trigger_prompt

        agent_context = AgentContext(skills=skills, system_message_suffix=system_suffix)

        def _run_sync() -> bool:
            """Returns True if the run was stopped, False if it finished naturally."""
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
                # Do not pass conversation_id: the sandbox server assigns its own
                # UUID internally.  Passing our application ID causes the server
                # to return 400 on subsequent GET /api/conversations/{id} calls
                # (the sandbox only handles IDs it generated itself).  Event
                # callbacks persist to our DB using trigger.conversation_id
                # directly, so the mapping stays correct regardless.
                conversation = Conversation(
                    agent=agent,
                    workspace=workspace,
                    callbacks=[_make_event_callback(trigger, loop, counter, turn_state)],
                    token_callbacks=[_make_token_callback(trigger, loop, counter, turn_state)],
                    max_iteration_per_run=agent_config.max_iterations,
                    visualizer=_QuietVisualizer(),
                )

                # Register so the worker's stop handler can find this conversation.
                active_conversations[trigger.conversation_id] = conversation
                try:
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
                    # Use non-blocking run so our polling loop can be interrupted
                    # by a stop signal without waiting for the SDK timeout.
                    conversation.run(blocking=False)
                    return _wait_for_done_or_stop(conversation, stop_event)
                finally:
                    active_conversations.pop(trigger.conversation_id, None)
                    # Stop the SDK's WebSocket client thread so it does not
                    # linger after the sandbox container is stopped.  Without
                    # this, Docker may recycle the container IP before the
                    # thread exits, causing the thread to send requests for the
                    # old conversation ID to the new container and get 400s.
                    try:
                        conversation.close()
                    except Exception as exc:
                        logger.debug("Failed to close conversation: %s", exc)

        stopped, errored = await asyncio.get_event_loop().run_in_executor(None, _run_sync)
        if not stopped:
            if errored:
                await conversation_repository.update_conversation_status(
                    trigger.conversation_id, "failed"
                )
                await stream_store.publish_realtime(
                    project_id=trigger.project_id,
                    conversation_id=trigger.conversation_id,
                    event_type="agent.conversation.failed",
                )
            else:
                await conversation_repository.update_conversation_status(
                    trigger.conversation_id, "finished"
                )
                await stream_store.publish_realtime(
                    project_id=trigger.project_id,
                    conversation_id=trigger.conversation_id,
                    event_type="agent.conversation.finished",
                )

    except Exception as exc:
        if not stop_event.is_set():
            logger.exception("Conversation %s failed: %s", trigger.conversation_id, exc)
            await conversation_repository.update_conversation_status(
                trigger.conversation_id, "failed", error_message=str(exc)
            )
            await stream_store.publish_realtime(
                project_id=trigger.project_id,
                conversation_id=trigger.conversation_id,
                event_type="agent.conversation.failed",
            )

    finally:
        stop_events.pop(trigger.conversation_id, None)
