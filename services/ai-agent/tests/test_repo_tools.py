"""Tests for repo_tools: token scrubbing and observation formatters."""

from urllib.parse import quote as urlquote

from src.agent.repo_tools import (
    CloneRepositoryObservation,
    ListRepositoriesObservation,
    _scrub_token,
)

# ─── _scrub_token ─────────────────────────────────────────────────────────────


def test_scrub_plain_token():
    result = _scrub_token("fatal: auth failed mytoken123", "mytoken123")
    assert "mytoken123" not in result
    assert "***" in result


def test_scrub_url_encoded_token():
    token = "tok@n!special"
    encoded = urlquote(token, safe="")
    text = f"https://x-access-token:{encoded}@github.com/org/repo.git"
    result = _scrub_token(text, token)
    assert token not in result
    assert encoded not in result
    assert "***" in result


def test_scrub_git_credential_pattern():
    text = "fatal: https://x-access-token:abc123@github.com/org/repo.git"
    result = _scrub_token(text, "different-token")
    assert "abc123" not in result
    assert "x-access-token:***@" in result


def test_empty_token_is_passthrough():
    text = "some git output"
    assert _scrub_token(text, "") == text


def test_absent_token_does_not_alter_text():
    assert _scrub_token("clean output", "not-present") == "clean output"


def test_scrub_both_plain_and_encoded_forms():
    token = "p@ss!"
    encoded = urlquote(token, safe="")
    text = f"plain={token} encoded={encoded}"
    result = _scrub_token(text, token)
    assert token not in result
    assert encoded not in result


# ─── ListRepositoriesObservation.to_llm_content ───────────────────────────────


def test_list_repos_error_shows_message():
    obs = ListRepositoriesObservation(error="connection refused")
    text = obs.to_llm_content[0].text
    assert "connection refused" in text
    assert "Failed" in text


def test_list_repos_empty_no_repos_found():
    obs = ListRepositoriesObservation(count=0)
    text = obs.to_llm_content[0].text
    assert "No repositories found" in text


def test_list_repos_populated_shows_all_fields():
    repos = [
        {
            "plugin_id": "p1",
            "plugin_name": "Github",
            "repo_id": "r1",
            "full_name": "org/myrepo",
            "owner": "org",
            "repo_name": "myrepo",
            "clone_url": "https://github.com/org/myrepo.git",
        }
    ]
    obs = ListRepositoriesObservation(repositories=repos, count=1)
    text = obs.to_llm_content[0].text
    assert "org/myrepo" in text
    assert "p1" in text
    assert "r1" in text
    assert "clone_repository" in text


def test_list_repos_multiple_entries_all_shown():
    repos = [
        {
            "plugin_id": "p1",
            "plugin_name": "Github",
            "repo_id": "r1",
            "full_name": "org/repo1",
            "owner": "org",
            "repo_name": "repo1",
            "clone_url": "https://github.com/org/repo1.git",
        },
        {
            "plugin_id": "p2",
            "plugin_name": "Gitlab",
            "repo_id": "r2",
            "full_name": "org/repo2",
            "owner": "org",
            "repo_name": "repo2",
            "clone_url": "https://gitlab.com/org/repo2.git",
        },
    ]
    obs = ListRepositoriesObservation(repositories=repos, count=2)
    text = obs.to_llm_content[0].text
    assert "org/repo1" in text
    assert "org/repo2" in text


# ─── CloneRepositoryObservation.to_llm_content ────────────────────────────────


def test_clone_success_shows_path_and_branch():
    obs = CloneRepositoryObservation(
        success=True,
        cloned_path="/workspace/repo",
        branch="main",
    )
    text = obs.to_llm_content[0].text
    assert "successfully" in text
    assert "/workspace/repo" in text
    assert "main" in text


def test_clone_failure_shows_message():
    obs = CloneRepositoryObservation(success=False, message="authentication failed")
    text = obs.to_llm_content[0].text
    assert "Failed" in text
    assert "authentication failed" in text


def test_clone_failure_empty_message():
    obs = CloneRepositoryObservation(success=False, message="")
    text = obs.to_llm_content[0].text
    assert "Failed" in text
