from __future__ import annotations

from collections.abc import Iterable
import subprocess
from typing import TypedDict

import pytest

from scripts.commands.backup_command import BackupCommand
from scripts.core.backup_config import BackupConfig
from tests.conftest import FixedClock


class RunCall(TypedDict):
    args: list[str]
    capture_output: bool
    check: bool


class ResticStub:
    def __init__(self, returncode: int = 0, stderr: str = "") -> None:
        self.returncode = returncode
        self.stderr = stderr
        self.ensure_called = 0
        self.run_calls: list[RunCall] = []

    def ensure_initialized(self) -> None:
        self.ensure_called += 1

    def run(
        self,
        args: Iterable[str],
        *,
        capture_output: bool = False,
        check: bool = True,
    ) -> subprocess.CompletedProcess[str]:
        args_list = list(args)
        self.run_calls.append(
            {
                "args": args_list,
                "capture_output": capture_output,
                "check": check,
            }
        )
        return subprocess.CompletedProcess(
            ["restic", *args_list],
            self.returncode,
            stdout="backup stdout\n",
            stderr=self.stderr,
        )

    def snapshots_json(self) -> list[dict[str, object]]:
        raise NotImplementedError


def test_backup_command_runs_and_writes_log(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    restic = ResticStub(returncode=0)
    command = BackupCommand(sample_config, restic, fixed_clock)

    result = command.run()

    assert result == 0
    assert restic.ensure_called == 1
    assert restic.run_calls
    first_args = restic.run_calls[0]["args"]
    assert first_args[:3] == [
        "backup",
        "--files-from",
        str(sample_config.source_include_file),
    ]

    log_file = sample_config.report_dir / "backup-20260216-010203.log"
    assert log_file.is_file()
    contents = log_file.read_text(encoding="utf-8")
    assert "Starting backup" in contents
    assert "backup stdout" in contents
    assert "Backup completed" in contents


def test_backup_command_raises_on_failed_restic(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    restic = ResticStub(returncode=3)
    command = BackupCommand(sample_config, restic, fixed_clock)

    with pytest.raises(SystemExit) as exc:
        command.run()

    assert exc.value.code == 3


def test_backup_command_logs_stderr_output(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    restic = ResticStub(returncode=0, stderr="backup stderr\n")
    command = BackupCommand(sample_config, restic, fixed_clock)

    result = command.run()

    assert result == 0
    log_file = sample_config.report_dir / "backup-20260216-010203.log"
    contents = log_file.read_text(encoding="utf-8")
    assert "backup stderr" in contents
