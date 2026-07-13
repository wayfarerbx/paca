"""Repository management tools for the agent (list and clone repositories)."""

from __future__ import annotations

import re
import shlex
from collections.abc import Sequence
from urllib.parse import quote as urlquote
from urllib.parse import urlparse

import httpx
from openhands.sdk import Action, Observation, TextContent, ToolDefinition
from openhands.sdk.tool import Tool, ToolExecutor, register_tool
from openhands.tools.terminal import TerminalAction, TerminalExecutor
from pydantic import Field


def _scrub_token(text: str, token: str) -> str:
    """Remove an auth token from git command output to prevent accidental logging."""
    if not token:
        return text
    scrubbed = text.replace(token, "***")
    # Also scrub the percent-encoded form (token embedded in URL).
    quoted_token = urlquote(token, safe="")
    if quoted_token != token:
        scrubbed = scrubbed.replace(quoted_token, "***")
    # Scrub the full credential pattern that git may echo in error messages.
    scrubbed = re.sub(r"x-access-token:[^@]+@", "x-access-token:***@", scrubbed)
    return scrubbed


# NOTE: do NOT import `from ..config import settings` here.
# This module is loaded inside the sandbox container by the remote agent server
# (via importlib.import_module) and the container has no access to our service
# config (database URL, internal API key, etc.).  All Paca-API coordinates are
# passed explicitly through Tool params instead.


# ─── List Repositories ────────────────────────────────────────────────────────


class ListRepositoriesAction(Action):
    """Action to list all available repositories from all repository plugins."""


class ListRepositoriesObservation(Observation):
    """Observation containing list of available repositories."""

    repositories: list[dict] = Field(default_factory=list)
    count: int = 0
    error: str = ""

    @property
    def to_llm_content(self) -> Sequence[TextContent]:
        if self.error:
            return [TextContent(text=f"Failed to list repositories: {self.error}")]
        if not self.count:
            return [
                TextContent(
                    text="No repositories found. Make sure a repository is linked to this project."
                )
            ]

        lines = [f"Found {self.count} available repository(ies):", ""]
        for i, repo in enumerate(self.repositories, 1):
            lines += [
                f"{i}. {repo['full_name']}",
                f"   Plugin: {repo['plugin_name']}",
                f"   Repository ID: {repo['repo_id']}",
                f"   Owner: {repo['owner']}",
                f"   Repository: {repo['repo_name']}",
                f"   Clone URL: {repo['clone_url']}",
                "",
            ]
        lines.append("To clone a repository, use the clone_repository tool with:")
        lines.append(f"  - plugin_id: The plugin ID (e.g., '{self.repositories[0]['plugin_id']}')")
        lines.append(f"  - repo_id: The repository ID (e.g., '{self.repositories[0]['repo_id']}')")
        lines.append("  - target_dir: Target directory (default: /workspace/repo)")
        return [TextContent(text="\n".join(lines))]


class ListRepositoriesExecutor(ToolExecutor[ListRepositoriesAction, ListRepositoriesObservation]):
    def __init__(
        self,
        project_id: str,
        repo_plugin_ids: list[str],
        api_base_url: str,
        api_key: str,
    ) -> None:
        self.project_id = project_id
        self.repo_plugin_ids = repo_plugin_ids
        self.api_base_url = api_base_url
        self.api_key = api_key

    def __call__(
        self, action: ListRepositoriesAction, conversation=None
    ) -> ListRepositoriesObservation:
        all_repos: list[dict] = []
        errors: list[str] = []
        for plugin_id in self.repo_plugin_ids:
            try:
                url = (
                    f"{self.api_base_url}/api/v1/plugins/{plugin_id}"
                    f"/projects/{self.project_id}/repositories"
                )
                response = httpx.get(url, headers={"X-API-Key": self.api_key}, timeout=10)
                response.raise_for_status()
                items = response.json().get("data", [])
                if not isinstance(items, list):
                    items = []
                for repo in items:
                    all_repos.append(
                        {
                            "plugin_id": plugin_id,
                            "plugin_name": plugin_id.split(".")[-1].title(),
                            "repo_id": repo["id"],
                            "full_name": repo["full_name"],
                            "owner": repo["owner"],
                            "repo_name": repo["repo_name"],
                            "clone_url": repo["clone_url"],
                        }
                    )
            except Exception as exc:
                errors.append(f"{plugin_id}: {exc}")
        if errors and not all_repos:
            return ListRepositoriesObservation(error="; ".join(errors))
        return ListRepositoriesObservation(repositories=all_repos, count=len(all_repos))


# ─── Clone Repository ─────────────────────────────────────────────────────────


class CloneRepositoryAction(Action):
    """Action to clone a specific repository."""

    plugin_id: str = Field(description="The plugin ID (e.g., plugin UUID)")
    repo_id: str = Field(description="The repository ID to clone (get this from list_repositories)")
    target_dir: str = Field(
        default="/workspace/repo",
        description="Target directory for cloning (absolute path)",
    )


class CloneRepositoryObservation(Observation):
    """Observation containing clone result."""

    success: bool = False
    message: str = ""
    cloned_path: str = ""
    branch: str = ""

    @property
    def to_llm_content(self) -> Sequence[TextContent]:
        if self.success:
            lines = [
                "Repository cloned successfully!",
                f"  Location: {self.cloned_path}",
                f"  Current branch: {self.branch}",
                "",
                "You can now work with the code in this repository.",
            ]
        else:
            lines = ["Failed to clone repository:", f"  {self.message}"]
        return [TextContent(text="\n".join(lines))]


class CloneRepositoryExecutor(ToolExecutor[CloneRepositoryAction, CloneRepositoryObservation]):
    def __init__(
        self,
        project_id: str,
        terminal: TerminalExecutor,
        api_base_url: str,
        api_key: str,
    ) -> None:
        self.project_id = project_id
        self.terminal = terminal
        self.api_base_url = api_base_url
        self.api_key = api_key

    def __call__(
        self, action: CloneRepositoryAction, conversation=None
    ) -> CloneRepositoryObservation:
        try:
            url = (
                f"{self.api_base_url}/api/v1/plugins/{action.plugin_id}"
                f"/projects/{self.project_id}/repositories/{action.repo_id}/clone-info"
            )
            response = httpx.get(url, headers={"X-API-Key": self.api_key}, timeout=10)
            response.raise_for_status()
            repo = response.json().get("data", {})
            if not repo.get("id"):
                return CloneRepositoryObservation(
                    success=False,
                    message=f"Repository {action.repo_id} not found in plugin {action.plugin_id}",
                )

            token = repo.get("token", "")
            parsed = urlparse(repo["clone_url"])
            host = parsed.hostname or parsed.netloc
            if parsed.port:
                host += f":{parsed.port}"

            # Embed the token directly in the clone URL (percent-encoded so
            # special characters don't break the URL or the shell command).
            clone_url = f"https://x-access-token:{urlquote(token, safe='')}@{host}{parsed.path}"

            target = shlex.quote(action.target_dir)
            self.terminal(TerminalAction(command=f"rm -rf {target}"))
            clone_cmd = f"git clone {shlex.quote(clone_url)} {target} 2>&1"
            result = self.terminal(TerminalAction(command=clone_cmd))
            if result.exit_code is not None and result.exit_code != 0:
                safe_msg = _scrub_token(result.text, token)
                return CloneRepositoryObservation(
                    success=False, message=f"git clone failed: {safe_msg}"
                )

            branch_result = self.terminal(
                TerminalAction(command=f"cd {target} && git branch --show-current 2>&1")
            )
            branch = branch_result.text.strip()
            return CloneRepositoryObservation(
                success=True,
                message=f"Cloned {repo['full_name']}",
                cloned_path=action.target_dir,
                branch=branch,
            )
        except Exception as exc:
            return CloneRepositoryObservation(success=False, message=str(exc))


# ─── Push Branch ─────────────────────────────────────────────────────────────


class PushBranchAction(Action):
    """Action to push a local branch to the remote repository."""

    plugin_id: str = Field(description="The plugin ID (from list_repositories)")
    repo_id: str = Field(description="The repository ID (from list_repositories)")
    branch_name: str = Field(
        default="",
        description=(
            "Name of the branch to push to the remote. "
            "Leave empty to use the currently checked-out branch."
        ),
    )
    repo_dir: str = Field(
        default="/workspace/repo",
        description="Local path to the cloned repository (default: /workspace/repo).",
    )


class PushBranchObservation(Observation):
    """Observation containing push result."""

    success: bool = False
    branch: str = ""
    message: str = ""

    @property
    def to_llm_content(self) -> Sequence[TextContent]:
        if self.success:
            return [
                TextContent(
                    text=(
                        f"Branch '{self.branch}' pushed to remote successfully.\n\n"
                        "**Next step (mandatory):** Create a pull/merge request now "
                        "so the changes can be reviewed. Call the repository plugin's "
                        "PR creation tool (e.g. `github_create_pull_request`) with "
                        "the branch you just pushed as `head_branch` and the repo's "
                        "default branch as `base_branch`. Do not consider the task "
                        "complete until a PR has been created."
                    )
                )
            ]
        return [TextContent(text=f"Failed to push branch: {self.message}")]


class PushBranchExecutor(ToolExecutor[PushBranchAction, PushBranchObservation]):
    def __init__(
        self,
        project_id: str,
        terminal: TerminalExecutor,
        api_base_url: str,
        api_key: str,
    ) -> None:
        self.project_id = project_id
        self.terminal = terminal
        self.api_base_url = api_base_url
        self.api_key = api_key

    def __call__(self, action: PushBranchAction, conversation=None) -> PushBranchObservation:
        try:
            # Fetch a fresh auth token for the push.
            ci_url = (
                f"{self.api_base_url}/api/v1/plugins/{action.plugin_id}"
                f"/projects/{self.project_id}/repositories/{action.repo_id}/clone-info"
            )
            resp = httpx.get(ci_url, headers={"X-API-Key": self.api_key}, timeout=10)
            resp.raise_for_status()
            ci = resp.json().get("data", {})
            token = ci.get("token", "")
            clone_url_str = ci.get("clone_url", "")
            if not clone_url_str:
                return PushBranchObservation(
                    success=False,
                    message="Could not determine clone URL for repository.",
                )

            # Resolve branch name.
            branch = action.branch_name.strip()
            if not branch:
                br = self.terminal(
                    TerminalAction(
                        command=f"git -C {shlex.quote(action.repo_dir)} branch --show-current 2>&1"
                    )
                )
                branch = br.text.strip()
            if not branch:
                return PushBranchObservation(
                    success=False,
                    message="Could not determine current branch. Specify branch_name explicitly.",
                )

            # Build an authenticated push URL (token is not exposed in logs).
            parsed = urlparse(clone_url_str)
            host = parsed.hostname or parsed.netloc
            if parsed.port:
                host += f":{parsed.port}"
            push_url = f"https://x-access-token:{urlquote(token, safe='')}@{host}{parsed.path}"

            result = self.terminal(
                TerminalAction(
                    command=(
                        f"git -C {shlex.quote(action.repo_dir)} push "
                        f"{shlex.quote(push_url)} HEAD:{shlex.quote(branch)} 2>&1"
                    )
                )
            )
            if result.exit_code is not None and result.exit_code != 0:
                safe_msg = _scrub_token(result.text, token)
                return PushBranchObservation(
                    success=False, branch=branch, message=f"git push failed: {safe_msg}"
                )
            return PushBranchObservation(success=True, branch=branch)
        except Exception as exc:
            return PushBranchObservation(success=False, message=str(exc))


# ─── Tool Definitions ─────────────────────────────────────────────────────────

_LIST_DESC = """\
List all available repositories from all repository plugins.

Use this when you need to see available repositories, choose which to clone, or
get repository IDs for use with clone_repository.
"""

_CLONE_DESC = """\
Clone a repository into your workspace so you can read and modify the code.

You MUST call list_repositories first to get the plugin_id and repo_id.

Parameters:
- plugin_id: The plugin UUID from list_repositories
- repo_id: The repository UUID from list_repositories
- target_dir: Where to clone (default: /workspace/repo — use this default)

After cloning, the full repository is available at target_dir and you can
read, edit, and run code using your other tools.
"""

_PUSH_BRANCH_DESC = """\
Push the current local branch to the remote repository.

Call this after you have committed all your changes and are ready to publish
the branch. You MUST have cloned the repository first with clone_repository.

Parameters:
- plugin_id: The plugin UUID (from list_repositories)
- repo_id: The repository UUID (from list_repositories)
- branch_name: Branch name to push (leave empty to use the currently checked-out branch)
- repo_dir: Path to the local repository (default: /workspace/repo)

**Mandatory next step:** After a successful push, you MUST create a pull/merge
request so the changes can be reviewed and merged. Call the repository plugin's
PR tool (e.g. `github_create_pull_request` for GitHub repos) with projectId,
taskId, repoId, title, head_branch (this branch), and base_branch (the repo's
default branch). A pushed branch without a PR is unfinished work.
"""


class ListRepositoriesTool(ToolDefinition[ListRepositoriesAction, ListRepositoriesObservation]):
    @classmethod
    def create(
        cls,
        conv_state=None,
        *,
        project_id: str,
        repo_plugin_ids: list[str],
        api_base_url: str,
        api_key: str,
    ) -> Sequence[ToolDefinition]:
        return [
            cls(
                description=_LIST_DESC,
                action_type=ListRepositoriesAction,
                executor=ListRepositoriesExecutor(
                    project_id, repo_plugin_ids, api_base_url, api_key
                ),
            )
        ]


class CloneRepositoryTool(ToolDefinition[CloneRepositoryAction, CloneRepositoryObservation]):
    @classmethod
    def create(
        cls,
        conv_state=None,
        *,
        project_id: str,
        api_base_url: str,
        api_key: str,
    ) -> Sequence[ToolDefinition]:
        working_dir = conv_state.workspace.working_dir if conv_state else "/tmp"
        terminal = TerminalExecutor(working_dir=working_dir)
        return [
            cls(
                description=_CLONE_DESC,
                action_type=CloneRepositoryAction,
                observation_type=CloneRepositoryObservation,
                executor=CloneRepositoryExecutor(project_id, terminal, api_base_url, api_key),
            )
        ]


class PushBranchTool(ToolDefinition[PushBranchAction, PushBranchObservation]):
    @classmethod
    def create(
        cls,
        conv_state=None,
        *,
        project_id: str,
        api_base_url: str,
        api_key: str,
    ) -> Sequence[ToolDefinition]:
        working_dir = conv_state.workspace.working_dir if conv_state else "/tmp"
        terminal = TerminalExecutor(working_dir=working_dir)
        return [
            cls(
                description=_PUSH_BRANCH_DESC,
                action_type=PushBranchAction,
                observation_type=PushBranchObservation,
                executor=PushBranchExecutor(project_id, terminal, api_base_url, api_key),
            )
        ]


# Register tool classes so Agent can resolve them via Tool(name=..., params={...})
register_tool("list_repositories", ListRepositoriesTool)
register_tool("clone_repository", CloneRepositoryTool)
register_tool("push_branch", PushBranchTool)


def make_repository_tool_specs(
    project_id: str,
    repo_plugin_ids: list[str],
    *,
    api_base_url: str,
    api_key: str,
) -> list[Tool]:
    """Return Tool specs (name references) for Agent instantiation.

    ``api_base_url`` and ``api_key`` are forwarded into every tool's params so
    the executors can call the Paca API from inside the sandbox container
    without importing our service settings.
    """
    common = {"api_base_url": api_base_url, "api_key": api_key}
    return [
        Tool(
            name="list_repositories",
            params={
                "project_id": project_id,
                "repo_plugin_ids": repo_plugin_ids,
                **common,
            },
        ),
        Tool(name="clone_repository", params={"project_id": project_id, **common}),
        Tool(name="push_branch", params={"project_id": project_id, **common}),
    ]
