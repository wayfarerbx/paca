"""Builders for LLM, skill, and MCP configuration objects."""

from __future__ import annotations

import logging
import platform
from urllib.parse import urlsplit, urlunsplit

from openhands.sdk import LLM
from openhands.sdk.context import KeywordTrigger, Skill
from openhands.sdk.llm.auth.openai import (
    CODEX_API_ENDPOINT,
    OpenAISubscriptionAuth,
    _extract_chatgpt_account_id,
)
from pydantic import SecretStr

from .. import llm_catalog
from ..config import settings
from ..models.agent import AgentConfig, AgentMCPServerRow, AgentSkillRow

logger = logging.getLogger(__name__)


def _uses_openai_subscription(provider: str, base_url: str | None) -> bool:
    return provider.strip().lower() == "chatgpt" and not base_url


def _create_openai_subscription_llm(model: str) -> LLM:
    """Create a ChatGPT subscription LLM without OpenHands' stale model whitelist."""
    auth = OpenAISubscriptionAuth()
    credentials = auth.refresh_if_needed_sync()
    if credentials is None:
        raise ValueError("OpenAI subscription login is required")

    account_id = _extract_chatgpt_account_id(credentials.access_token)
    extra_headers = {
        "originator": "codex_cli_rs",
        "OpenAI-Beta": "responses=experimental",
        "User-Agent": f"openhands-sdk ({platform.system()}; {platform.machine()})",
    }
    if account_id:
        extra_headers["chatgpt-account-id"] = account_id

    llm = LLM(
        model=f"openai/{model.removeprefix('openai/')}",
        base_url=CODEX_API_ENDPOINT.rsplit("/", 1)[0],
        api_key=None,
        extra_headers=extra_headers,
        litellm_extra_body={"store": False},
        temperature=None,
        max_output_tokens=None,
        stream=True,
        auth_type="subscription",
        subscription_vendor="openai",
    )
    llm._is_subscription = True
    llm.max_output_tokens = None
    llm._effective_max_output_tokens = None
    llm.temperature = None
    llm.auth_type = "subscription"
    llm.subscription_vendor = "openai"
    llm._subscription_credential_store = auth._credential_store
    llm._subscription_credentials = credentials
    return llm


def _sandbox_reachable_url(url: str) -> str:
    parsed = urlsplit(url)
    if parsed.hostname not in {"127.0.0.1", "localhost"}:
        return url

    netloc = "host.docker.internal"
    if parsed.port is not None:
        netloc = f"{netloc}:{parsed.port}"
    return urlunsplit((parsed.scheme, netloc, parsed.path, parsed.query, parsed.fragment))


def build_llm(agent_config: AgentConfig) -> LLM:
    """Construct an OpenHands SDK LLM instance from agent configuration."""
    provider = agent_config.llm_provider
    llm_base_url = agent_config.llm_base_url or None

    if _uses_openai_subscription(provider, llm_base_url):
        logger.info(
            "LLM config — model=%s auth_type=subscription vendor=openai",
            agent_config.llm_model,
        )
        return _create_openai_subscription_llm(agent_config.llm_model)

    # For providers outside our own catalog (freeform "Custom…" entries), route
    # through the OpenAI-compatible client by prefixing with "openai/". Checking
    # against our catalog rather than litellm.provider_list matters: litellm
    # reserves names like "custom"/"vllm"/"huggingface" for its own non-chat,
    # non-OpenAI-compatible handlers, and a user typing one of those by
    # coincidence would otherwise get silently misrouted into them.
    if llm_base_url and provider not in llm_catalog.load():
        model_str = (
            agent_config.llm_model
            if agent_config.llm_model.startswith("openai/")
            else f"openai/{agent_config.llm_model}"
        )
    else:
        model_str = f"{provider}/{agent_config.llm_model}"

    key_val = agent_config.llm_api_key_secret_ref or ""
    logger.info(
        "LLM config — model=%s base_url=%s api_key_set=%s",
        model_str,
        llm_base_url or "(none)",
        bool(key_val),
    )
    return LLM(
        model=model_str,
        api_key=SecretStr(key_val),
        base_url=llm_base_url,
        stream=True,
    )


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
        paca_api_url = _sandbox_reachable_url(settings.api_base_url)
        paca_gateway_url = _sandbox_reachable_url(settings.gateway_base_url)
        if settings.dev_mcp_path:
            command, args = "node", [settings.dev_mcp_path]
        else:
            command, args = "npx", ["-y", "@paca-ai/paca-mcp"]
        servers["paca"] = {
            "command": command,
            "args": args,
            "env": {
                "PACA_API_KEY": settings.paca_api_key,
                "PACA_API_URL": paca_api_url,
                "PACA_GATEWAY_URL": paca_gateway_url,
                "PACA_AGENT_ID": agent_id,
                "PACA_PROJECT_ID": project_id,
            },
        }

    return {"mcpServers": servers}
