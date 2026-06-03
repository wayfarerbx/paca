"""Docker container sandbox for isolated OpenHands agent execution."""
from __future__ import annotations

import logging
import os
import platform as platform_module
import secrets
import socket
import threading
import time
from contextlib import contextmanager
from pathlib import Path
from typing import Iterator

import docker
import httpx
from docker.models.containers import Container

from openhands.sdk import RemoteWorkspace

from ..config import settings

logger = logging.getLogger(__name__)

# ─── Port pool ────────────────────────────────────────────────────────────────

_port_lock = threading.Lock()
_ports_in_use: set[int] = set()


def _acquire_port() -> int:
    with _port_lock:
        for p in range(settings.port_pool_start, settings.port_pool_start + settings.port_pool_size):
            if p not in _ports_in_use:
                _ports_in_use.add(p)
                return p
    raise RuntimeError(
        f"No ports available in the agent port pool "
        f"({settings.port_pool_start}–{settings.port_pool_start + settings.port_pool_size - 1}). "
        "Consider increasing PORT_POOL_SIZE."
    )


def _release_port(port: int) -> None:
    with _port_lock:
        _ports_in_use.discard(port)


# ─── Helpers ──────────────────────────────────────────────────────────────────


def _detect_platform() -> str:
    machine = platform_module.machine().lower()
    if "arm" in machine or "aarch64" in machine:
        return "linux/arm64"
    return "linux/amd64"


def _is_inside_docker() -> bool:
    return os.path.exists("/.dockerenv")


def _get_current_networks(client: docker.DockerClient) -> list[str]:
    """Return the Docker networks the current container is attached to."""
    try:
        hostname = socket.gethostname()
        info = client.containers.get(hostname)
        return list(info.attrs["NetworkSettings"]["Networks"].keys())
    except Exception as exc:
        logger.debug("Could not detect current container networks: %s", exc)
        return []


def _get_app_host_path(client: docker.DockerClient) -> str | None:
    """Return the host-side absolute path of the ai-agent application directory.

    Inside Docker: read it from the container's bind-mount table (the compose
    file mounts the source tree at /app).

    Outside Docker (local dev): derive it from the location of the ``src``
    package on disk.
    """
    if _is_inside_docker():
        return _find_host_path_for(client, "/app")
    # Local dev: walk up from this file's location to the project root
    # (…/services/ai-agent/src/agent/docker_workspace.py → …/services/ai-agent)
    try:
        return str(Path(__file__).resolve().parent.parent.parent)
    except Exception:
        return None


def _find_host_path_for(client: docker.DockerClient, container_path: str) -> str | None:
    """Return the host-side source of a bind mount in the current container.

    Used to re-share a directory (e.g. /mcp) that is already bind-mounted into
    this container into the sibling sandbox containers we spawn.  The Docker
    socket gives access to the *host* daemon, so volumes must be expressed as
    host paths — not paths that only exist inside this container.
    """
    try:
        hostname = socket.gethostname()
        info = client.containers.get(hostname)
        for mount in info.attrs.get("Mounts", []):
            if mount.get("Type") == "bind" and mount.get("Destination") == container_path:
                return mount["Source"]
    except Exception as exc:
        logger.debug("Could not inspect mounts for %s: %s", container_path, exc)
    return None


def _wait_for_ready(host: str, timeout: float = 120.0) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            resp = httpx.get(f"{host}/health", timeout=2.0)
            if resp.status_code < 500:
                return
        except Exception:
            pass
        time.sleep(1.0)
    raise TimeoutError(f"Agent server at {host} not ready after {timeout}s")


# ─── Context manager ──────────────────────────────────────────────────────────


@contextmanager
def docker_sandbox(
    conversation_id: str,
    git_committer_name: str = "paca-agent",
    git_committer_email: str = "280579135+paca-agent@users.noreply.github.com",
) -> Iterator[RemoteWorkspace]:
    """Spin up an isolated agent-server container and yield a RemoteWorkspace.

    Inside Docker (production/dev-compose)
    ───────────────────────────────────────
    The sandbox container joins the same Docker network as the ai-agent service.
    This is what makes the `api` and `gateway` hostnames resolvable inside the
    sandbox — MCP servers (including the built-in Paca MCP) call those services
    directly, so they must be reachable from within the container.

    The MCP build directory (/mcp) is re-shared into the sandbox using the
    host-side bind-mount path, so `node /mcp/build/index.js` works inside the
    container without baking the file into the agent-server image.

    Outside Docker (local dev)
    ──────────────────────────
    The container port is mapped to a host port from the pool and accessed via
    localhost.  MCP volume sharing is attempted on a best-effort basis.
    """
    client = docker.DockerClient(base_url=f"unix://{settings.docker_socket}")
    container: Container | None = None
    host_port: int | None = None

    try:
        # ── Volumes ───────────────────────────────────────────────────────────
        volumes: dict = {}

        # Share the MCP build directory so `node /mcp/build/index.js` works.
        mcp_host_path = _find_host_path_for(client, "/mcp")
        if mcp_host_path:
            volumes[mcp_host_path] = {"bind": "/mcp", "mode": "ro"}
            logger.debug("Sharing MCP build into sandbox from host path: %s", mcp_host_path)
        else:
            logger.warning(
                "Could not detect host path for /mcp — the built-in Paca MCP "
                "server may not be available inside the agent sandbox."
            )

        # Share the ai-agent source tree so the remote server can import
        # src.agent.repo_tools to register the custom repository tools.
        app_host_path = _get_app_host_path(client)
        if app_host_path:
            volumes[app_host_path] = {"bind": "/app", "mode": "ro"}
            logger.debug("Sharing app source into sandbox from host path: %s", app_host_path)
        else:
            logger.warning(
                "Could not detect host path for /app — custom repository tools "
                "(list_repositories, clone_repository, …) will not be available."
            )

        # ── Environment ───────────────────────────────────────────────────────
        environment: dict = {
            # OH_SECRET_KEY is required by the agent server to encrypt persisted
            # secrets.  Each container is ephemeral so a fresh key per run is fine.
            "OH_SECRET_KEY": secrets.token_hex(32),
            # Suppress the SDK startup banner from container logs.
            "OPENHANDS_SUPPRESS_BANNER": "1",
            # Git committer identity — applied to every git command in the container.
            "GIT_AUTHOR_NAME": git_committer_name,
            "GIT_AUTHOR_EMAIL": git_committer_email,
            "GIT_COMMITTER_NAME": git_committer_name,
            "GIT_COMMITTER_EMAIL": git_committer_email,
        }
        if app_host_path:
            # Add the mounted app directory to Python's module search path so
            # `importlib.import_module("src.agent.repo_tools")` succeeds inside
            # the container.
            environment["OH_EXTRA_PYTHON_PATH"] = "/app"

        run_kwargs: dict = {
            "image": settings.agent_server_image,
            "detach": True,
            "platform": _detect_platform(),
            # Automatically deleted when stopped — no manual cleanup needed.
            "remove": True,
            "labels": {
                "paca.conversation_id": conversation_id,
                "paca.managed": "true",
            },
            "environment": environment,
        }
        if volumes:
            run_kwargs["volumes"] = volumes

        if _is_inside_docker():
            # Join the same network as this container so the sandbox can reach
            # `api` and `gateway` by their Docker Compose service hostnames.
            networks = _get_current_networks(client)
            network = networks[0] if networks else "bridge"
            run_kwargs["network"] = network

            logger.info(
                "Starting agent sandbox: conversation=%s network=%s image=%s",
                conversation_id,
                network,
                settings.agent_server_image,
            )
            container = client.containers.run(**run_kwargs)
            container.reload()
            container_ip = (
                container.attrs["NetworkSettings"]["Networks"][network]["IPAddress"]
            )
            host = f"http://{container_ip}:{settings.agent_server_container_port}"
        else:
            # Local dev: map to a pooled host port and access via localhost.
            host_port = _acquire_port()
            run_kwargs["ports"] = {
                f"{settings.agent_server_container_port}/tcp": host_port
            }

            logger.info(
                "Starting agent sandbox: conversation=%s host_port=%d image=%s",
                conversation_id,
                host_port,
                settings.agent_server_image,
            )
            container = client.containers.run(**run_kwargs)
            host = f"http://localhost:{host_port}"

        _wait_for_ready(host)
        logger.info("Agent sandbox ready: conversation=%s host=%s", conversation_id, host)

        yield RemoteWorkspace(host=host, working_dir="/workspace")

    finally:
        if container is not None:
            try:
                container.stop(timeout=30)
                logger.info("Agent sandbox stopped: conversation=%s", conversation_id)
            except Exception as exc:
                logger.warning(
                    "Failed to stop agent container for conversation=%s: %s",
                    conversation_id,
                    exc,
                )
        if host_port is not None:
            _release_port(host_port)
        client.close()
