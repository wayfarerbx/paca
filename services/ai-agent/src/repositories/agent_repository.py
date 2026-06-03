"""Database access layer for agent configuration."""
from __future__ import annotations

import json
import logging

from ..core.db import get_pool
from ..models.agent import AgentConfig, AgentMCPServerRow, AgentSkillRow

logger = logging.getLogger(__name__)


async def load_agent_config(agent_id: str) -> AgentConfig | None:
    """Load full agent configuration (agent, MCP servers, skills) from the database."""
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
            a.can_clone_repos,
            a.git_committer_name,
            a.git_committer_email
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
        git_committer_name=row["git_committer_name"] or "paca-agent",
        git_committer_email=row["git_committer_email"] or "280579135+paca-agent@users.noreply.github.com",
        mcp_servers=mcp_servers,
        skills=skills,
    )
