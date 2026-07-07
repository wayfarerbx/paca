"""Agent conversation executor — orchestrates LLM, skills, MCP, and repo tools."""

from __future__ import annotations

import asyncio
import logging
import threading
import time

import httpx
from openhands.sdk import Agent, AgentContext, Conversation
from openhands.sdk.conversation.state import ConversationExecutionStatus
from openhands.sdk.conversation.visualizer import ConversationVisualizerBase
from openhands.sdk.skills import merge_skills_by_name
from openhands.tools import get_default_tools

from ..config import settings
from ..core import streams as stream_store
from ..core.registry import active_conversations, stop_events
from ..core.streams import TriggerMessage
from ..models.agent import AgentConfig
from ..repositories import conversation_repository
from . import trigger_skills
from .builder import build_llm, build_mcp_config, build_skills, load_default_skills
from .docker_workspace import docker_sandbox
from .prompt import build_initial_prompt
from .repo_tools import make_repository_tool_specs

logger = logging.getLogger(__name__)

_ERROR_STATUSES = frozenset(
    {
        ConversationExecutionStatus.ERROR,
        ConversationExecutionStatus.STUCK,
    }
)

# events.reconcile() re-walks the *entire* remote event history over REST
# (openhands.sdk.conversation.impl.remote_conversation.RemoteEventsList has no
# "since" cursor) — the SDK's own blocking run() only pays that cost once, at
# completion, rather than on every status poll. We still need it more often
# than that to bound the WS-dead-thread persistence gap (see
# _wait_for_done_or_stop), but not on every 2s tick.
_RECONCILE_INTERVAL = 10.0


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
    reconcile,
    poll_interval: float = 2.0,
    timeout: float = 3600.0,
) -> tuple[bool, bool]:
    """Poll the remote conversation until it finishes or a stop is signaled.

    Returns (stopped, errored):
      stopped  — True if the loop exited because a stop was requested.
      errored  — True if the conversation ended with ERROR or STUCK status.
    """
    start = time.monotonic()
    last_reconcile = start
    while True:
        # stop_event.wait() blocks for up to poll_interval seconds, returning
        # True immediately if the event is already set.
        if stop_event.wait(timeout=poll_interval):
            try:
                conversation.pause()
            except Exception as exc:
                logger.warning("Failed to pause conversation on stop request: %s", exc)
            # Catch up on anything the dead WS thread missed before we stop
            # watching this conversation entirely.
            reconcile(conversation)
            return True, False

        now = time.monotonic()
        if now - start > timeout:
            logger.warning("Conversation polling timed out after %.0f seconds", timeout)
            reconcile(conversation)
            return False, False

        try:
            # RemoteState caches execution_status from WebSocket events and only
            # ever re-fetches over REST if the cache is still empty (see
            # RemoteState._get_conversation_info). openhands-sdk's WS client
            # thread permanently stops reconnecting after any ConnectionClosed
            # (upstream bug, still present as of openhands-sdk 1.30.0 — see
            # OpenHands/software-agent-sdk#1532), which would otherwise freeze
            # this loop on a stale "running" status until the timeout above.
            # Force a live REST read every poll so a dead socket can't wedge us.
            conversation.state.refresh_from_server()
            status = conversation.state.execution_status

            # The same dead socket also stops new agent events from ever
            # reaching our persistence callback (it only fires from inside the
            # WS client thread). reconcile() is a full-history REST walk (see
            # _RECONCILE_INTERVAL), so throttle it independently of the
            # cheap poll_interval status check above.
            if now - last_reconcile >= _RECONCILE_INTERVAL:
                reconcile(conversation)
                last_reconcile = now

            if status.is_terminal():
                errored = status in _ERROR_STATUSES
                if errored:
                    detail = _get_conversation_error_detail(conversation)
                    logger.error(
                        "Conversation ended with status %s — %s",
                        status.value,
                        detail or "no detail available",
                    )
                # Final catch-up so the run never ends with an un-persisted
                # tail from the last reconcile_interval stretch.
                reconcile(conversation)
                return False, errored
        except Exception as exc:
            logger.debug("Failed to read conversation execution status: %s", exc)


# ─── Custom visualizer ────────────────────────────────────────────────────────


class _QuietVisualizer(ConversationVisualizerBase):
    """No-op visualizer that silently accepts all event types.

    The SDK's DefaultConversationVisualizer emits a WARNING for every event
    type absent from its EVENT_VISUALIZATION_CONFIG (e.g. StreamingDeltaEvent).
    Since this service handles all events via the `callbacks` list, we
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
    """Thread-safe monotonic counter for ordering persisted events."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._value = 0

    def next(self) -> int:
        with self._lock:
            v = self._value
            self._value += 1
            return v


class _SeenEvents:
    """Thread-safe set of SDK event ids already routed to persistence.

    Shared between the live WebSocket callback and the reconciliation
    fallback in `_wait_for_done_or_stop` so that an event is persisted
    exactly once no matter which path first observes it.
    """

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._ids: set[str] = set()

    def mark_new(self, event_id: str) -> bool:
        """Return True and record the id if it hasn't been seen before."""
        with self._lock:
            if event_id in self._ids:
                return False
            self._ids.add(event_id)
            return True


# ─── Event callback ───────────────────────────────────────────────────────────


def _persist_event(
    trigger: TriggerMessage,
    loop: asyncio.AbstractEventLoop,
    counter: _AtomicCounter,
    event,
) -> None:
    """Filter, index, and persist a single SDK event to Postgres + Valkey.

    Shared by the live WebSocket callback and the reconciliation fallback (see
    `_make_reconciler`) so both paths apply identical filtering and write to
    the same destinations, regardless of which one first observes an event.
    """
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


def _make_event_callback(
    trigger: TriggerMessage,
    loop: asyncio.AbstractEventLoop,
    counter: _AtomicCounter,
    seen: _SeenEvents,
):
    """Return a synchronous callback invoked by the OpenHands SDK on each complete event."""

    def callback(event) -> None:
        event_id = getattr(event, "id", None)
        if event_id is not None and not seen.mark_new(event_id):
            return
        _persist_event(trigger, loop, counter, event)

    return callback


def _make_reconciler(
    trigger: TriggerMessage,
    loop: asyncio.AbstractEventLoop,
    counter: _AtomicCounter,
    seen: _SeenEvents,
):
    """Return a function that catches up on events the dead WS thread missed.

    openhands-sdk's WS client thread permanently stops reconnecting after any
    ConnectionClosed (see the comment in `_wait_for_done_or_stop`), so the
    composed event callback silently stops firing entirely — even though the
    agent keeps running in its sandbox and producing events. `events.reconcile()`
    re-fetches the full event history over REST and merges it straight into
    the SDK's local cache, bypassing callbacks, so anything new it pulls in
    has to be routed through `_persist_event` by hand here.
    """

    def reconcile(conversation) -> None:
        try:
            conversation.state.events.reconcile()
            for event in list(conversation.state.events):
                event_id = getattr(event, "id", None)
                if event_id is None or not seen.mark_new(event_id):
                    continue
                _persist_event(trigger, loop, counter, event)
        except Exception as exc:
            logger.debug(
                "Event reconciliation failed for conversation %s: %s",
                trigger.conversation_id,
                exc,
            )

    return reconcile


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
    seen_events = _SeenEvents()
    stop_event = threading.Event()
    logger.info("Starting conversation %s (agent=%s)", trigger.conversation_id, trigger.agent_id)
    await conversation_repository.update_conversation_status(trigger.conversation_id, "running")

    # Register the stop event so the worker can signal us via stream control messages.
    stop_events[trigger.conversation_id] = stop_event

    try:
        llm = build_llm(agent_config)
        # User-configured skills win on a name collision with a default.
        skills = merge_skills_by_name(build_skills(agent_config.skills), load_default_skills())
        trigger_skills.append_trigger_skill(
            skills, trigger.trigger_type, trigger.task_id, trigger.conversation_id
        )
        mcp_config = build_mcp_config(
            agent_config.mcp_servers, agent_config.agent_id, trigger.project_id
        )

        system_suffix = agent_config.system_prompt or ""

        # Inject current project context so the AI never needs to ask for the
        # project ID or call list_projects to discover it.
        system_suffix += _build_project_context_suffix(trigger.project_id)

        # Documentation lives in Paca — never create local files, and always
        # read it before acting (it holds architecture/conventions/prior
        # decisions no skill-specific instruction should let you skip).
        # Detailed write-back workflow is covered by the always-active
        # `paca` skill and the specialized `paca-doc` skill.
        system_suffix += (
            "\n\n## Documentation\n"
            "This project's documentation is managed in Paca — use "
            "`list_docs` / `read_doc` / `write_doc`. Never create local "
            "markdown files. **Read the relevant documentation before doing "
            "anything else** — it defines the project's architecture, "
            "conventions, and prior decisions.\n"
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
                "\n\n## Repository Access\n"
                "This project has linked repositories: `list_repositories` → "
                "`clone_repository` (clones to /workspace/repo). Create a "
                "feature branch before changing anything, commit your work, "
                "then call `push_branch` — do NOT skip this step. PR "
                "creation, review, and comments are handled by the "
                "repository plugin's own tools.\n"
            )

        agent_context = AgentContext(skills=skills, system_message_suffix=system_suffix)

        def _run_sync() -> bool:
            """Returns True if the run was stopped, False if it finished naturally."""
            with docker_sandbox(
                trigger.conversation_id,
                git_committer_name=agent_config.git_committer_name,
                git_committer_email=agent_config.git_committer_email,
                env=agent_config.env_vars,
            ) as workspace:
                agent_kwargs: dict = {"llm": llm, "agent_context": agent_context}
                if mcp_config.get("mcpServers"):
                    agent_kwargs["mcp_config"] = mcp_config

                if has_repos:
                    agent_kwargs["tools"] = get_default_tools() + make_repository_tool_specs(
                        trigger.project_id,
                        trigger.repo_plugin_ids,
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
                    callbacks=[_make_event_callback(trigger, loop, counter, seen_events)],
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
                    reconcile = _make_reconciler(trigger, loop, counter, seen_events)
                    return _wait_for_done_or_stop(conversation, stop_event, reconcile)
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
