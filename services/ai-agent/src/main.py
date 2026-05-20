"""FastAPI application factory."""
from __future__ import annotations

import asyncio
import logging
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI

from .config import settings
from .db import close_pool
from .routes.conversations import router as conversations_router
from .routes.health import router as health_router
from .routes.llm import router as llm_router
from .streams import close_client
from .worker import run_worker, stop_worker

logging.basicConfig(level=settings.log_level.upper())
logger = logging.getLogger(__name__)

_worker_task: asyncio.Task | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global _worker_task
    logger.info("AI-agent service starting up")
    _worker_task = asyncio.create_task(run_worker())
    try:
        yield
    finally:
        logger.info("AI-agent service shutting down")
        stop_worker()
        if _worker_task:
            _worker_task.cancel()
            try:
                await _worker_task
            except asyncio.CancelledError:
                pass
        await close_pool()
        await close_client()


app = FastAPI(title="Paca AI-Agent Service", lifespan=lifespan)

app.include_router(health_router)
app.include_router(llm_router)
app.include_router(conversations_router)


if __name__ == "__main__":
    uvicorn.run("src.main:app", host="0.0.0.0", port=settings.port, log_level=settings.log_level.lower())
