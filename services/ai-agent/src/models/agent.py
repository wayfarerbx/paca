"""Domain model dataclasses for the AI agent service."""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class AgentMCPServerRow:
    server_name: str
    transport: str
    url: str | None
    command: str | None
    args: list[str]
    env: dict[str, str]
    is_enabled: bool


@dataclass
class AgentSkillRow:
    skill_name: str
    skill_content: str | None
    triggers: list[str]
    is_enabled: bool


@dataclass
class AgentConfig:
    agent_id: str
    project_id: str
    system_prompt: str | None
    task_trigger_prompt: str
    doc_comment_trigger_prompt: str
    chat_trigger_prompt: str
    description_write_trigger_prompt: str
    llm_provider: str
    llm_model: str
    llm_api_key_secret_ref: str
    llm_base_url: str
    max_iterations: int
    can_clone_repos: bool
    git_committer_name: str = "paca-agent"
    git_committer_email: str = "280579135+paca-agent@users.noreply.github.com"
    mcp_servers: list[AgentMCPServerRow] = field(default_factory=list)
    skills: list[AgentSkillRow] = field(default_factory=list)
    env_vars: dict[str, str] = field(default_factory=dict)
