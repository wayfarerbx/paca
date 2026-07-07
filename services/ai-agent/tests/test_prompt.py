"""Tests for the initial prompt builder."""

from src.agent.prompt import build_initial_prompt
from src.core.streams import TriggerMessage


def _trigger(**kwargs) -> TriggerMessage:
    defaults = dict(
        stream_id="1-1",
        trigger_type="agent.task_assigned",
        conversation_id="conv-123",
        agent_id="agent-456",
        project_id="proj-789",
        task_id=None,
        comment_id=None,
        chat_session_id=None,
        message="Implement feature X",
        actor_member_id="member-001",
        repo_plugin_ids=[],
    )
    return TriggerMessage(**{**defaults, **kwargs})


def test_message_included():
    result = build_initial_prompt(_trigger(message="Do something"))
    assert "Do something" in result


def test_action_type_task_assigned():
    result = build_initial_prompt(_trigger(trigger_type="agent.task_assigned"))
    assert "Action type: Task assignment" in result


def test_action_type_task_comment_mention():
    result = build_initial_prompt(
        _trigger(trigger_type="agent.comment_mention", task_id="task-1")
    )
    assert "Action type: Task comment mention" in result


def test_action_type_doc_comment_mention():
    result = build_initial_prompt(
        _trigger(trigger_type="agent.comment_mention", task_id=None)
    )
    assert "Action type: Document comment mention" in result


def test_action_type_chat_message():
    result = build_initial_prompt(_trigger(trigger_type="agent.chat_message"))
    assert "Action type: Direct chat message" in result


def test_action_type_description_write():
    result = build_initial_prompt(_trigger(trigger_type="agent.description_write"))
    assert "Action type: Write task description" in result


def test_task_id_included_when_set():
    result = build_initial_prompt(_trigger(task_id="task-42"))
    assert "task-42" in result


def test_task_id_absent_when_none():
    result = build_initial_prompt(_trigger(task_id=None))
    assert "Task ID" not in result


def test_comment_id_included_when_set():
    result = build_initial_prompt(_trigger(comment_id="cmt-7"))
    assert "cmt-7" in result


def test_no_repo_section_without_repos():
    result = build_initial_prompt(_trigger(), all_repos=None)
    assert "Repository" not in result


def test_single_repo_inline_clone_instructions():
    repo = {
        "plugin_id": "plugin-1",
        "repo_id": "repo-99",
        "full_name": "org/myrepo",
        "clone_url": "https://github.com/org/myrepo.git",
    }
    result = build_initial_prompt(_trigger(), all_repos=[repo])
    assert "plugin-1" in result
    assert "repo-99" in result
    assert "org/myrepo" in result


def test_multiple_repos_list_repositories_hint():
    repos = [
        {"plugin_id": "p1", "repo_id": "r1", "full_name": "org/a", "clone_url": ""},
        {"plugin_id": "p2", "repo_id": "r2", "full_name": "org/b", "clone_url": ""},
    ]
    result = build_initial_prompt(_trigger(), all_repos=repos)
    assert "list_repositories" in result
