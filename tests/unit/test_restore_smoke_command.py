from __future__ import annotations

from collections.abc import Iterable
import subprocess
from pathlib import Path
from typing import Any

import pytest

import scripts.commands.restore_smoke_command as restore_module
from scripts.commands.restore_smoke_command import RestoreSmokeCommand
from scripts.core.backup_config import BackupConfig
from tests.conftest import FixedClock


class ResticStub:
    def __init__(self, snapshots_returncode: int = 0) -> None:
        self.snapshots_returncode = snapshots_returncode
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
            return subprocess.CompletedProcess(
                ["restic", *current],
                self.snapshots_returncode,
                "",
                "",
            )
        if current[:2] == ["restore", "latest"]:
            target = Path(current[current.index("--target") + 1])
            restored = target / "restored.txt"
            restored.parent.mkdir(parents=True, exist_ok=True)
            restored.write_text("ok", encoding="utf-8")
            return subprocess.CompletedProcess(["restic", *current], 0, "", "")
        return subprocess.CompletedProcess(["restic", *current], 0, "", "")

    def ensure_initialized(self) -> None:
        raise NotImplementedError

    def snapshots_json(self) -> list[dict[str, object]]:
        raise NotImplementedError


def test_restore_smoke_returns_1_when_no_snapshots(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    restic = ResticStub(snapshots_returncode=1)
    command = RestoreSmokeCommand(sample_config, restic, fixed_clock)

    result = command.run()

    assert result == 1
    assert restic.calls == [["snapshots"]]


def test_restore_smoke_runs_restore_with_include(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    restic = ResticStub(snapshots_returncode=0)
    command = RestoreSmokeCommand(sample_config, restic, fixed_clock)

    result = command.run()

    assert result == 0
    assert restic.calls[0] == ["snapshots"]
    restore_args = restic.calls[1]
    assert restore_args[:2] == ["restore", "latest"]
    assert "--target" in restore_args
    include_paths = [
        restore_args[index + 1]
        for index, value in enumerate(restore_args)
        if value == "--include"
    ]
    assert include_paths == [str(path) for path in sample_config.source_paths]


def test_restore_smoke_cleans_temp_dir(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    temp_root = sample_config.report_dir / "temp-root"
    observed: dict[str, object] = {}

    monkeypatch.setattr(restore_module.tempfile,
                        "mkdtemp", lambda: str(temp_root))

    def fake_rmtree(path: str | Path, ignore_errors: bool = True, *args: Any, **kwargs: Any) -> None:
        _ = args, kwargs
        observed["path"] = Path(path)
        observed["ignore_errors"] = ignore_errors

    monkeypatch.setattr(restore_module.shutil, "rmtree", fake_rmtree)

    restic = ResticStub(snapshots_returncode=1)
    RestoreSmokeCommand(sample_config, restic, fixed_clock).run()

    assert observed["path"] == temp_root
    assert observed["ignore_errors"] is True


def test_restore_smoke_returns_1_when_target_not_dir(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    class NoWriteResticStub(ResticStub):
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
                return subprocess.CompletedProcess(["restic", *current], 0, "", "")
            return subprocess.CompletedProcess(["restic", *current], 0, "", "")

    restic = NoWriteResticStub(snapshots_returncode=0)

    original_is_dir = Path.is_dir

    def patched_is_dir(path: Path) -> bool:
        if path.name == "restore":
            return False
        return original_is_dir(path)

    monkeypatch.setattr(Path, "is_dir", patched_is_dir)

    result = RestoreSmokeCommand(sample_config, restic, fixed_clock).run()

    assert result == 1
