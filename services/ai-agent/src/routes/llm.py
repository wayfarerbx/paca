"""LLM models endpoint — returns the verified model list from the OpenHands SDK."""
from __future__ import annotations

from fastapi import APIRouter

router = APIRouter(prefix="/llm")


@router.get("/models")
async def list_llm_models() -> dict[str, list[str]]:
    """Return verified LLM models grouped by provider.

    The list is sourced directly from the installed ``openhands-sdk`` package so
    it always reflects the version deployed alongside this service — no manual
    updates required.
    """
    from openhands.sdk.llm.utils.verified_models import VERIFIED_MODELS  # noqa: PLC0415

    return dict(VERIFIED_MODELS)
