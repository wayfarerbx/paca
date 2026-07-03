"""Database access layer for agent configuration."""

from __future__ import annotations

import base64
import json
import logging

from ..config import settings
from ..core.db import get_pool
from ..models.agent import AgentConfig, AgentMCPServerRow, AgentSkillRow

logger = logging.getLogger(__name__)


def _decrypt_secret(ciphertext: str, label: str = "LLM API key") -> str:
    """Decrypt an AES-256-GCM ciphertext produced by the Go API's secret.Encryptor.

    If ENCRYPTION_KEY is not configured the value is returned as-is (plaintext
    backward-compat mode).  Any decryption error returns an empty string so
    that a clear "missing API key" error surfaces at the LLM call rather than
    the misleading "token expired / incorrect" that results from forwarding the
    raw ciphertext to the provider. `label` is used only to identify the field
    being decrypted in log messages (e.g. "LLM API key", "environment variable").
    """
    if not ciphertext:
        return ciphertext
    if not settings.encryption_key:
        logger.warning(
            "ENCRYPTION_KEY is not set — %s will be used as-is from the "
            "database. If the api service encrypts secrets, set ENCRYPTION_KEY in "
            "the ai-agent environment to the same value.",
            label,
        )
        return ciphertext
    try:
        from cryptography.hazmat.primitives.ciphers.aead import AESGCM

        key = bytes.fromhex(settings.encryption_key)
        raw = base64.b64decode(ciphertext)
        # Go's GCM uses a 12-byte nonce; nonce is prepended to ciphertext+tag.
        nonce_size = 12
        nonce, ct_with_tag = raw[:nonce_size], raw[nonce_size:]
        plaintext = AESGCM(key).decrypt(nonce, ct_with_tag, None)
        return plaintext.decode()
    except Exception as exc:
        logger.error(
            "Failed to decrypt %s — the value will be treated as empty. Verify "
            "that ENCRYPTION_KEY in the ai-agent service matches the api "
            "service. Error: %s",
            label,
            exc,
        )
        return ""


async def load_agent_config(agent_id: str) -> AgentConfig | None:
    """Load full agent configuration (agent, MCP servers, skills) from the database."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        SELECT
            a.id,
            a.project_id,
            a.system_prompt,
            a.task_trigger_prompt,
            a.doc_comment_trigger_prompt,
            a.chat_trigger_prompt,
            a.description_write_trigger_prompt,
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
    env_var_rows = await pool.fetch(
        """
        SELECT key, encrypted_value
        FROM agent_environment_variables
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
    env_vars = {
        r["key"]: _decrypt_secret(r["encrypted_value"], label=f"environment variable {r['key']}")
        for r in env_var_rows
    }

    return AgentConfig(
        agent_id=str(row["id"]),
        project_id=str(row["project_id"]),
        system_prompt=row["system_prompt"],
        task_trigger_prompt=row["task_trigger_prompt"] or "",
        doc_comment_trigger_prompt=row["doc_comment_trigger_prompt"] or "",
        chat_trigger_prompt=row["chat_trigger_prompt"] or "",
        description_write_trigger_prompt=row["description_write_trigger_prompt"] or "",
        llm_provider=row["llm_provider"],
        llm_model=row["llm_model"],
        llm_api_key_secret_ref=_decrypt_secret(row["llm_api_key_secret_ref"] or ""),
        llm_base_url=row["llm_base_url"] or "",
        max_iterations=row["max_iterations"],
        can_clone_repos=row["can_clone_repos"],
        git_committer_name=row["git_committer_name"] or "paca-agent",
        git_committer_email=row["git_committer_email"]
        or "280579135+paca-agent@users.noreply.github.com",
        mcp_servers=mcp_servers,
        skills=skills,
        env_vars=env_vars,
    )
