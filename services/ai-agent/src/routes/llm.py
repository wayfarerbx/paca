"""LLM models endpoint — serves the pre-generated provider/model list."""

from __future__ import annotations

import json
from pathlib import Path

from fastapi import APIRouter, HTTPException

router = APIRouter(prefix="/llm")

_DATA_FILE = Path(__file__).parent.parent.parent / "data" / "llm_models.json"

# Loaded once on first request, then cached for the lifetime of the process.
_cache: dict | None = None


def _load() -> dict:
    global _cache
    if _cache is None:
        if not _DATA_FILE.exists():
            raise FileNotFoundError(
                f"Model list not found at {_DATA_FILE}. "
                "Run scripts/generate_llm_models.py to generate it."
            )
        _cache = json.loads(_DATA_FILE.read_text())
    return _cache


@router.get("/models")
async def list_llm_models() -> dict[str, dict]:
    """Return the pre-generated LLM model/provider list.

    Refresh the list by running scripts/generate_llm_models.py and restarting
    the service (or simply redeploying the updated data/llm_models.json).
    """
    try:
        return _load()
    except FileNotFoundError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
