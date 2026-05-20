"""Fetches agent configuration from the PostgreSQL database."""
from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field
from typing import Any

from .db import get_pool

logger = logging.getLogger(__name__)


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
    llm_provider: str
    llm_model: str
    llm_api_key_secret_ref: str
    llm_base_url: str | None
    max_iterations: int
    can_clone_repos: bool
    mcp_servers: list[AgentMCPServerRow] = field(default_factory=list)
    skills: list[AgentSkillRow] = field(default_factory=list)


async def load_agent_config(agent_id: str) -> AgentConfig | None:
    """Load full agent configuration from the DB."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        SELECT
            a.id,
            a.project_id,
            a.system_prompt,
            a.llm_provider,
            a.llm_model,
            a.llm_api_key_secret AS llm_api_key_secret_ref,
            a.llm_base_url,
            a.max_iterations,
            a.can_clone_repos
        FROM agents a
        WHERE a.id = $1
        """,
        agent_id,
    )
    if row is None:
        return None

    mcp_rows = await pool.fetch(
        """
        SELECT server_name, transport, url, command, args, env, is_enabled
        FROM agent_mcp_servers
        WHERE agent_id = $1
        """,
        agent_id,
    )
    skill_rows = await pool.fetch(
        """
        SELECT skill_name, skill_content, triggers, is_enabled
        FROM agent_skills
        WHERE agent_id = $1
        """,
        agent_id,
    )

    mcp_servers = [
        AgentMCPServerRow(
            server_name=r["server_name"],
            transport=r["transport"],
            url=r["url"],
            command=r["command"],
            args=json.loads(r["args"]) if r["args"] else [],
            env=json.loads(r["env"]) if r["env"] else {},
            is_enabled=r["is_enabled"],
        )
        for r in mcp_rows
    ]
    skills = [
        AgentSkillRow(
            skill_name=r["skill_name"],
            skill_content=r["skill_content"],
            triggers=json.loads(r["triggers"]) if r["triggers"] else [],
            is_enabled=r["is_enabled"],
        )
        for r in skill_rows
    ]

    return AgentConfig(
        agent_id=str(row["id"]),
        project_id=str(row["project_id"]),
        system_prompt=row["system_prompt"],
        llm_provider=row["llm_provider"],
        llm_model=row["llm_model"],
        llm_api_key_secret_ref=row["llm_api_key_secret_ref"],
        llm_base_url=row["llm_base_url"],
        max_iterations=row["max_iterations"],
        can_clone_repos=row["can_clone_repos"],
        mcp_servers=mcp_servers,
        skills=skills,
    )
