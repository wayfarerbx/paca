"""Initial prompt builder for agent conversations."""

from __future__ import annotations

from ..core.streams import TriggerMessage

_ACTION_TYPE_LABELS = {
    "agent.task_assigned": "Task assignment",
    "agent.chat_message": "Direct chat message",
    "agent.description_write": "Write task description",
}


def _action_type_label(trigger: TriggerMessage) -> str:
    if trigger.trigger_type == "agent.comment_mention":
        # task_id is set for task-comment mentions; absent for doc-comment mentions.
        return "Task comment mention" if trigger.task_id else "Document comment mention"
    return _ACTION_TYPE_LABELS.get(trigger.trigger_type, trigger.trigger_type)


def build_initial_prompt(trigger: TriggerMessage, all_repos: list[dict] | None = None) -> str:
    """Construct the first message sent to the agent."""
    lines = [f"Action type: {_action_type_label(trigger)}", "", trigger.message]
    if trigger.task_id:
        lines.append(f"\nTask ID: {trigger.task_id}")
    if trigger.comment_id:
        lines.append(f"\nComment ID: {trigger.comment_id}")
    if trigger.chat_session_id:
        lines.append(f"\nChat Session ID: {trigger.chat_session_id}")

    if all_repos:
        lines.append("\n## Repository Setup Required")
        lines.append(
            f"This project has {len(all_repos)} linked"
            f" repositor{'y' if len(all_repos) == 1 else 'ies'}."
            " You MUST clone it before working on any code."
        )
        if len(all_repos) == 1:
            repo = all_repos[0]
            lines.append(
                f"\nClone the repository now by calling clone_repository with:"
                f"\n  plugin_id='{repo['plugin_id']}'"
                f"\n  repo_id='{repo['repo_id']}'"
                f"\n  (target_dir defaults to /workspace/repo)"
            )
            lines.append(f"\nRepository: {repo['full_name']}")
        else:
            lines.append(
                "\nCall list_repositories to get the available"
                " repositories, then clone the one you need."
            )

    return "\n".join(lines)
