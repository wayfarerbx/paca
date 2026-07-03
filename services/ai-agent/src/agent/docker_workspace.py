"""Docker container sandbox for isolated OpenHands agent execution."""

from __future__ import annotations

import io
import logging
import os
import platform as platform_module
import secrets
import socket
import tarfile
import threading
import time
from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

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
        for p in range(
            settings.port_pool_start, settings.port_pool_start + settings.port_pool_size
        ):
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


def _docker_base_url() -> str:
    docker_socket = settings.docker_socket.strip()
    if docker_socket.startswith(("unix://", "npipe://", "tcp://", "http://", "https://")):
        return docker_socket
    if platform_module.system().lower() == "windows":
        if docker_socket in {"/var/run/docker.sock", ""}:
            return "npipe:////./pipe/dockerDesktopLinuxEngine"
        if docker_socket.startswith(("\\\\.\\pipe\\", "//./pipe/")):
            return "npipe:////./pipe/" + docker_socket.replace("\\", "/").rsplit("/", 1)[-1]
    return f"unix://{docker_socket}"


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

    Used to re-share a directory that is already bind-mounted into this
    container into the sibling sandbox containers we spawn.  The Docker socket
    gives access to the *host* daemon, so volumes must be expressed as host
    paths — not paths that only exist inside this container.
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


# ─── Repo tools injection ─────────────────────────────────────────────────────

_REPO_TOOLS_DEST = "/tmp/paca_tools"
_OPENHANDS_HOME_DEST = "/home/openhands/.openhands"
_OPENHANDS_HOME_VOLUME = "paca-openhands-home"
_REPO_TOOLS_FILES = [
    ("src/__init__.py", "paca_tools/src/__init__.py"),
    ("src/agent/__init__.py", "paca_tools/src/agent/__init__.py"),
    ("src/agent/repo_tools.py", "paca_tools/src/agent/repo_tools.py"),
]


def _openhands_home_host_path() -> str | None:
    openhands_dir = Path.home() / ".openhands"
    if not openhands_dir.exists():
        return None
    return str(openhands_dir)


def _use_openhands_home_volume() -> bool:
    return platform_module.system().lower() == "windows" and not _is_inside_docker()


def _sandbox_bind_dir_for(path: str) -> str | None:
    if not path.startswith("/"):
        return None
    path_parts = Path(path).parts
    if len(path_parts) < 2:
        return None
    return "/" + path_parts[1]


def _tar_directory_contents(source: Path) -> bytes:
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tar:
        for child in source.rglob("*"):
            tar.add(str(child), arcname=str(child.relative_to(source)))
    buf.seek(0)
    return buf.getvalue()


def _ensure_openhands_home_volume(client: docker.DockerClient, host_path: str) -> str:
    try:
        client.volumes.get(_OPENHANDS_HOME_VOLUME)
    except docker.errors.NotFound:
        client.volumes.create(name=_OPENHANDS_HOME_VOLUME)

    sync_container = None
    try:
        sync_container = client.containers.create(
            image=settings.agent_server_image,
            command=["-c", "mkdir -p /home/openhands/.openhands && sleep 60"],
            entrypoint="sh",
            user="root",
            detach=True,
            volumes={
                _OPENHANDS_HOME_VOLUME: {
                    "bind": _OPENHANDS_HOME_DEST,
                    "mode": "rw",
                }
            },
        )
        sync_container.start()
        sync_container.put_archive(
            _OPENHANDS_HOME_DEST,
            _tar_directory_contents(Path(host_path)),
        )
        exit_code, output = sync_container.exec_run(
            f"chown -R 10001:10001 {_OPENHANDS_HOME_DEST}"
        )
        if exit_code != 0:
            raise RuntimeError(
                "Failed to prepare OpenHands home volume: "
                + output.decode(errors="replace")
            )
    finally:
        if sync_container is not None:
            sync_container.remove(force=True)

    return _OPENHANDS_HOME_VOLUME


def _copy_repo_tools_to_container(container: Container) -> bool:
    """Copy src.agent.repo_tools into /tmp/paca_tools in the sandbox.

    Used in production where /app is baked into the image instead of
    bind-mounted from the host.  After this call, the agent server can
    run importlib.import_module("src.agent.repo_tools") when
    OH_EXTRA_PYTHON_PATH=/tmp/paca_tools is set.

    Returns True on success, False if any required source file is missing.
    """
    src_root = Path("/app")
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tar:
        for dir_name in ["paca_tools", "paca_tools/src", "paca_tools/src/agent"]:
            info = tarfile.TarInfo(name=dir_name)
            info.type = tarfile.DIRTYPE
            info.mode = 0o755
            tar.addfile(info)
        for src_rel, dest_rel in _REPO_TOOLS_FILES:
            path = src_root / src_rel
            if not path.exists():
                logger.warning("Cannot copy repo_tools into sandbox: missing %s", path)
                return False
            tar.add(str(path), arcname=dest_rel)
    buf.seek(0)
    container.put_archive("/tmp", buf.getvalue())
    logger.debug("Copied repo_tools into sandbox container at %s", _REPO_TOOLS_DEST)
    return True


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

    Outside Docker (local dev)
    ──────────────────────────
    The container port is mapped to a host port from the pool and accessed via
    localhost.
    """
    client = docker.DockerClient(base_url=_docker_base_url())
    container: Container | None = None
    host_port: int | None = None

    try:
        # ── Volumes ───────────────────────────────────────────────────────────
        volumes: dict = {}

        # Share the ai-agent source tree so the remote server can import
        # src.agent.repo_tools to register the custom repository tools.
        app_host_path = _get_app_host_path(client)
        if app_host_path:
            volumes[app_host_path] = {"bind": "/app", "mode": "ro"}
            logger.debug("Sharing app source into sandbox from host path: %s", app_host_path)
        else:
            # Production: /app is baked into the image (not bind-mounted).
            # We will copy the necessary files into the container after it starts.
            logger.debug(
                "No host bind-mount for /app detected — "
                "will inject repo_tools into the sandbox container directly."
            )

        openhands_host_path = _openhands_home_host_path()
        if openhands_host_path:
            if _use_openhands_home_volume():
                openhands_volume = _ensure_openhands_home_volume(client, openhands_host_path)
                volumes[openhands_volume] = {"bind": _OPENHANDS_HOME_DEST, "mode": "rw"}
                logger.debug("Sharing OpenHands home directory into sandbox via Docker volume.")
            else:
                volumes[openhands_host_path] = {"bind": _OPENHANDS_HOME_DEST, "mode": "rw"}
                logger.debug("Sharing OpenHands home directory into sandbox.")

        # When DEV_MCP_PATH is set, the MCP server is a local file on a bind-
        # mounted volume (e.g. /mcp/build/index.js).  The OpenHands runtime
        # runs the MCP subprocess *inside* the sandbox container, so the same
        # volume must be forwarded there.  We resolve the host-side source of
        # the bind mount so Docker can re-attach it to the sibling container.
        if settings.dev_mcp_path and _is_inside_docker():
            path_parts = Path(settings.dev_mcp_path).parts
            if len(path_parts) < 2:
                logger.warning(
                    "DEV_MCP_PATH=%s has no parent directory component; "
                    "skipping sandbox volume injection.",
                    settings.dev_mcp_path,
                )
                path_parts = ()
            mcp_bind_dir = ("/" + path_parts[1]) if path_parts else None
            mcp_host_path = _find_host_path_for(client, mcp_bind_dir) if mcp_bind_dir else None
            if mcp_host_path:
                volumes[mcp_host_path] = {"bind": mcp_bind_dir, "mode": "ro"}
                logger.debug(
                    "Sharing MCP source into sandbox: host=%s → container=%s",
                    mcp_host_path,
                    mcp_bind_dir,
                )
            else:
                logger.warning(
                    "DEV_MCP_PATH=%s is set but %s is not a recognised bind-mount "
                    "on this container; the MCP server will likely fail to start "
                    "inside the sandbox.",
                    settings.dev_mcp_path,
                    mcp_bind_dir,
                )

        # ── Environment ───────────────────────────────────────────────────────
        if settings.dev_mcp_path and not _is_inside_docker():
            mcp_bind_dir = _sandbox_bind_dir_for(settings.dev_mcp_path)
            if mcp_bind_dir and settings.dev_mcp_host_path:
                mcp_host_path = str(Path(settings.dev_mcp_host_path).resolve())
                volumes[mcp_host_path] = {"bind": mcp_bind_dir, "mode": "ro"}
                logger.debug(
                    "Sharing MCP source into sandbox: host=%s -> container=%s",
                    mcp_host_path,
                    mcp_bind_dir,
                )
            else:
                logger.warning(
                    "DEV_MCP_PATH=%s is set outside Docker, but DEV_MCP_HOST_PATH is empty "
                    "or DEV_MCP_PATH is not an absolute container path; the MCP server will "
                    "likely fail to start inside the sandbox.",
                    settings.dev_mcp_path,
                )

        if settings.local_repos_host_path:
            local_repos_host_path = str(Path(settings.local_repos_host_path).resolve())
            volumes[local_repos_host_path] = {
                "bind": settings.local_repos_path,
                "mode": "rw",
            }
            logger.debug(
                "Sharing local repositories into sandbox: host=%s -> container=%s",
                local_repos_host_path,
                settings.local_repos_path,
            )

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
            # Add the app source to Python's module search path so the agent
            # server can run importlib.import_module("src.agent.repo_tools").
            # Value depends on whether we share via bind-mount or file injection.
            "OH_EXTRA_PYTHON_PATH": "/app" if app_host_path else _REPO_TOOLS_DEST,
        }

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
            container_ip = container.attrs["NetworkSettings"]["Networks"][network]["IPAddress"]
            host = f"http://{container_ip}:{settings.agent_server_container_port}"
        else:
            # Local dev: map to a pooled host port and access via localhost.
            host_port = _acquire_port()
            run_kwargs["ports"] = {f"{settings.agent_server_container_port}/tcp": host_port}

            logger.info(
                "Starting agent sandbox: conversation=%s host_port=%d image=%s",
                conversation_id,
                host_port,
                settings.agent_server_image,
            )
            container = client.containers.run(**run_kwargs)
            host = f"http://localhost:{host_port}"

        if not app_host_path:
            # Production path: inject repo_tools directly into the container
            # filesystem before the server finishes starting up.
            try:
                _copy_repo_tools_to_container(container)
            except Exception as exc:
                logger.warning(
                    "Failed to inject repo_tools into sandbox — "
                    "repository tools (list_repositories, clone_repository, …) "
                    "will not be available: %s",
                    exc,
                )

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
