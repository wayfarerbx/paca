"""Tests for docker_workspace utility functions (no Docker daemon required)."""

from unittest.mock import patch

import pytest

from src.agent.docker_workspace import (
    _acquire_port,
    _detect_platform,
    _docker_base_url,
    _openhands_home_host_path,
    _ports_in_use,
    _release_port,
    _use_openhands_home_volume,
)
from src.config import settings


@pytest.fixture(autouse=True)
def clean_port_pool():
    """Restore the port pool to empty before and after every test."""
    _ports_in_use.clear()
    yield
    _ports_in_use.clear()


# ─── _detect_platform ─────────────────────────────────────────────────────────


def test_x86_64_returns_amd64():
    with patch("src.agent.docker_workspace.platform_module.machine", return_value="x86_64"):
        assert _detect_platform() == "linux/amd64"


def test_aarch64_returns_arm64():
    with patch("src.agent.docker_workspace.platform_module.machine", return_value="aarch64"):
        assert _detect_platform() == "linux/arm64"


def test_armv7l_returns_arm64():
    with patch("src.agent.docker_workspace.platform_module.machine", return_value="armv7l"):
        assert _detect_platform() == "linux/arm64"


def test_unknown_arch_defaults_to_amd64():
    with patch("src.agent.docker_workspace.platform_module.machine", return_value="riscv64"):
        assert _detect_platform() == "linux/amd64"


# ─── Port pool ────────────────────────────────────────────────────────────────


def test_acquire_returns_first_available_port():
    port = _acquire_port()
    assert port == settings.port_pool_start


def test_acquired_port_is_tracked():
    port = _acquire_port()
    assert port in _ports_in_use


def test_second_acquire_returns_next_port():
    first = _acquire_port()
    second = _acquire_port()
    assert second == first + 1


def test_release_removes_port_from_pool():
    port = _acquire_port()
    _release_port(port)
    assert port not in _ports_in_use


def test_released_port_can_be_reacquired():
    port = _acquire_port()
    _release_port(port)
    reacquired = _acquire_port()
    assert reacquired == port


def test_exhausted_pool_raises_runtime_error():
    _ports_in_use.update(
        range(settings.port_pool_start, settings.port_pool_start + settings.port_pool_size)
    )
    with pytest.raises(RuntimeError, match="No ports available"):
        _acquire_port()


def test_windows_default_docker_socket_uses_npipe(monkeypatch):
    monkeypatch.setattr(settings, "docker_socket", "/var/run/docker.sock")
    with patch("src.agent.docker_workspace.platform_module.system", return_value="Windows"):
        assert _docker_base_url() == "npipe:////./pipe/dockerDesktopLinuxEngine"


def test_explicit_docker_url_is_preserved(monkeypatch):
    monkeypatch.setattr(settings, "docker_socket", "tcp://127.0.0.1:2375")
    assert _docker_base_url() == "tcp://127.0.0.1:2375"


def test_openhands_home_host_path_returns_existing_openhands_dir(tmp_path, monkeypatch):
    openhands_dir = tmp_path / ".openhands"
    openhands_dir.mkdir()
    monkeypatch.setattr("src.agent.docker_workspace.Path.home", lambda: tmp_path)

    assert _openhands_home_host_path() == str(openhands_dir)


def test_openhands_home_host_path_omits_missing_openhands_dir(tmp_path, monkeypatch):
    monkeypatch.setattr("src.agent.docker_workspace.Path.home", lambda: tmp_path)

    assert _openhands_home_host_path() is None


def test_windows_local_dev_uses_named_openhands_volume(monkeypatch):
    monkeypatch.setattr("src.agent.docker_workspace.platform_module.system", lambda: "Windows")
    monkeypatch.setattr("src.agent.docker_workspace._is_inside_docker", lambda: False)

    assert _use_openhands_home_volume() is True


def test_non_windows_uses_bind_mount_for_openhands_home(monkeypatch):
    monkeypatch.setattr("src.agent.docker_workspace.platform_module.system", lambda: "Linux")
    monkeypatch.setattr("src.agent.docker_workspace._is_inside_docker", lambda: False)

    assert _use_openhands_home_volume() is False
