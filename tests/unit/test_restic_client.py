from __future__ import annotations

import json
import subprocess
from collections.abc import Iterable
from typing import Any

import pytest

from scripts.core.backup_config import BackupConfig
from scripts.core.restic_client import ResticClient


def test_run_builds_command_and_env(
    sample_config: BackupConfig,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    captured: dict[str, Any] = {}

    def fake_run(cmd: list[str], **kwargs: Any) -> subprocess.CompletedProcess[str]:
        captured["cmd"] = cmd
        captured.update(kwargs)
        return subprocess.CompletedProcess(cmd, 0, stdout="", stderr="")

    monkeypatch.setattr(subprocess, "run", fake_run)

    client = ResticClient(sample_config)
    client.run(["snapshots"], capture_output=True)

    assert captured["cmd"] == ["restic", "snapshots"]
    assert captured["capture_output"] is True
    assert captured["check"] is True
    assert captured["text"] is True
    assert captured["env"]["RESTIC_REPOSITORY"] == sample_config.restic_repository
    assert captured["env"]["RESTIC_PASSWORD_COMMAND"] == sample_config.restic_password_command


def test_ensure_initialized_returns_when_repo_exists(sample_config: BackupConfig) -> None:
    class FakeResticClient(ResticClient):
        def __init__(self, config: BackupConfig) -> None:
            super().__init__(config)
            self.calls: list[list[str]] = []

        def run(
            self,
            args: Iterable[str],
            *,
            capture_output: bool = False,
            check: bool = True,
        ) -> subprocess.CompletedProcess[str]:
            self.calls.append(list(args))
            return subprocess.CompletedProcess(["restic", *args], 0, stdout="", stderr="")

    client = FakeResticClient(sample_config)
    client.ensure_initialized()

    assert client.calls == [["snapshots"]]


def test_ensure_initialized_calls_init_when_missing(sample_config: BackupConfig) -> None:
    class FakeResticClient(ResticClient):
        def __init__(self, config: BackupConfig) -> None:
            super().__init__(config)
            self.calls: list[list[str]] = []

        def run(
            self,
            args: Iterable[str],
            *,
            capture_output: bool = False,
            check: bool = True,
        ) -> subprocess.CompletedProcess[str]:
            current = list(args)
            self.calls.append(current)
            if current == ["snapshots"]:
                return subprocess.CompletedProcess(["restic", *args], 1, stdout="", stderr="")
            return subprocess.CompletedProcess(["restic", *args], 0, stdout="", stderr="")

    client = FakeResticClient(sample_config)
    client.ensure_initialized()

    assert client.calls == [["snapshots"], ["init"]]


def test_snapshots_json_returns_parsed_list(sample_config: BackupConfig) -> None:
    class FakeResticClient(ResticClient):
        def run(
            self,
            args: Iterable[str],
            *,
            capture_output: bool = False,
            check: bool = True,
        ) -> subprocess.CompletedProcess[str]:
            payload = json.dumps([{"id": "a"}, {"id": "b"}])
            return subprocess.CompletedProcess(["restic", *args], 0, stdout=payload, stderr="")

    snapshots = FakeResticClient(sample_config).snapshots_json()

    assert snapshots == [{"id": "a"}, {"id": "b"}]


def test_snapshots_json_raises_on_invalid_json(sample_config: BackupConfig) -> None:
    class FakeResticClient(ResticClient):
        def run(
            self,
            args: Iterable[str],
            *,
            capture_output: bool = False,
            check: bool = True,
        ) -> subprocess.CompletedProcess[str]:
            return subprocess.CompletedProcess(["restic", *args], 0, stdout="not-json", stderr="")

    with pytest.raises(SystemExit, match="Could not parse restic snapshots JSON"):
        FakeResticClient(sample_config).snapshots_json()


def test_snapshots_json_raises_on_non_list_payload(sample_config: BackupConfig) -> None:
    class FakeResticClient(ResticClient):
        def run(
            self,
            args: Iterable[str],
            *,
            capture_output: bool = False,
            check: bool = True,
        ) -> subprocess.CompletedProcess[str]:
            return subprocess.CompletedProcess(
                ["restic", *args],
                0,
                stdout=json.dumps({"id": "a"}),
                stderr="",
            )

    with pytest.raises(SystemExit, match="Unexpected snapshots payload format"):
        FakeResticClient(sample_config).snapshots_json()
