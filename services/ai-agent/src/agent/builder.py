"""Builders for LLM, skill, and MCP configuration objects."""

from __future__ import annotations

import logging

from openhands.sdk import LLM
from openhands.sdk.context import KeywordTrigger, Skill
from pydantic import SecretStr

from ..config import settings
from ..models.agent import AgentConfig, AgentMCPServerRow, AgentSkillRow

logger = logging.getLogger(__name__)

# Providers that use an OpenAI-compatible REST API but are not natively
# recognised by LiteLLM.  A base URL is transparently injected for them.
_OPENAI_COMPAT_PROVIDER_URLS: dict[str, str] = {
    "glm": "https://open.bigmodel.cn/api/paas/v4/",
    "nvidia": "https://integrate.api.nvidia.com/v1",
    "qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
}


def build_llm(agent_config: AgentConfig) -> LLM:
    """Construct an OpenHands SDK LLM instance from agent configuration."""
    provider = agent_config.llm_provider
    llm_base_url = agent_config.llm_base_url

    if not llm_base_url and provider in _OPENAI_COMPAT_PROVIDER_URLS:
        llm_base_url = _OPENAI_COMPAT_PROVIDER_URLS[provider]

    # When a base_url is present the endpoint is OpenAI-compatible;
    # LiteLLM requires the "openai/" prefix in that case.
    model_str = (
        f"openai/{agent_config.llm_model}"
        if llm_base_url
        else f"{provider}/{agent_config.llm_model}"
    )

    key_val = agent_config.llm_api_key_secret_ref or ""
    logger.info(
        "LLM config — model=%s base_url=%s api_key_set=%s",
        model_str,
        llm_base_url or "(none)",
        bool(key_val),
    )

    llm_kwargs: dict = {
        "model": model_str,
        "api_key": SecretStr(key_val),
        "stream": True,
    }
    if llm_base_url:
        llm_kwargs["base_url"] = llm_base_url

    return LLM(**llm_kwargs)


def build_skills(db_skills: list[AgentSkillRow]) -> list[Skill]:
    """Convert DB skill rows into OpenHands SDK Skill objects."""
    result = []
    for row in db_skills:
        if not row.is_enabled:
            continue
        trigger = KeywordTrigger(keywords=row.triggers) if row.triggers else None
        content = row.skill_content or ""
        result.append(Skill(name=row.skill_name, content=content, trigger=trigger))
    return result


def build_mcp_config(
    db_servers: list[AgentMCPServerRow],
    agent_id: str,
    project_id: str,
) -> dict:
    """Build the MCP server configuration dict for the OpenHands SDK.

    User-configured servers come first; the built-in Paca MCP server is always
    appended last so it cannot be overridden by user entries.
    """
    servers: dict = {}

    for row in db_servers:
        if not row.is_enabled:
            continue
        if row.transport == "stdio":
            servers[row.server_name] = {
                "command": row.command,
                "args": row.args,
                "env": row.env,
            }
        else:
            servers[row.server_name] = {
                "url": row.url,
                **({"auth": "oauth"} if row.transport == "oauth" else {}),
            }

    if settings.paca_api_key:
        servers["paca"] = {
            "command": "npx",
            "args": ["-y", "@paca-ai/paca-mcp"],
            "env": {
                "PACA_API_KEY": settings.paca_api_key,
                "PACA_API_URL": settings.api_base_url,
                "PACA_GATEWAY_URL": settings.gateway_base_url,
                "PACA_AGENT_ID": agent_id,
                "PACA_PROJECT_ID": project_id,
            },
        }

    return {"mcpServers": servers}
