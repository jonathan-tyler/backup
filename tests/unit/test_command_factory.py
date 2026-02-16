from __future__ import annotations

from collections.abc import Iterable
from pathlib import Path
import subprocess
from typing import cast

import pytest

from scripts.commands.backup_command import BackupCommand
from scripts.commands.factory import CommandFactory
from scripts.commands.forget_prune_command import ForgetPruneCommand
from scripts.commands.report_command import ReportCommand
from scripts.commands.restore_smoke_command import RestoreSmokeCommand
from scripts.core.backup_config import BackupConfig
from scripts.core.config_loader import ConfigLoader
from tests.conftest import FixedClock


class LoaderStub:
    def __init__(self, config: BackupConfig) -> None:
        self.config = config
        self.calls: list[str | None] = []

    def load(self, env_file: str | None) -> BackupConfig:
        self.calls.append(env_file)
        return self.config


class ResticStub:
    def run(
        self,
        args: Iterable[str],
        *,
        capture_output: bool = False,
        check: bool = True,
    ) -> subprocess.CompletedProcess[str]:
        raise NotImplementedError

    def ensure_initialized(self) -> None:
        raise NotImplementedError

    def snapshots_json(self) -> list[dict[str, object]]:
        raise NotImplementedError


def test_create_returns_expected_command_types(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    loader = LoaderStub(sample_config)

    factory = CommandFactory(
        Path("/unused"),
        config_loader=cast(ConfigLoader, loader),
        clock=fixed_clock,
        restic_client_factory=lambda _: ResticStub(),
    )

    backup_cmd = factory.create("backup", "env1")
    report_cmd = factory.create("report", "env2")
    prune_cmd = factory.create("forget-prune", "env3")
    restore_cmd = factory.create("restore-smoke", "env4")

    assert isinstance(backup_cmd, BackupCommand)
    assert isinstance(report_cmd, ReportCommand)
    assert isinstance(prune_cmd, ForgetPruneCommand)
    assert isinstance(restore_cmd, RestoreSmokeCommand)
    assert loader.calls == ["env1", "env2", "env3", "env4"]


def test_create_raises_for_unsupported_action(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    factory = CommandFactory(
        Path("/unused"),
        config_loader=cast(ConfigLoader, LoaderStub(sample_config)),
        clock=fixed_clock,
        restic_client_factory=lambda _: ResticStub(),
    )

    with pytest.raises(SystemExit, match="Unsupported action"):
        factory.create("unsupported", None)
