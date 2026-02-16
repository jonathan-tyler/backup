from __future__ import annotations

from collections.abc import Callable
from pathlib import Path

from ..core.clock import Clock
from ..core.backup_config import BackupConfig
from ..core.config_loader import ConfigLoader
from ..core.file_scanner import FileScanner
from ..core.protocols import ClockProtocol, FileScannerProtocol, ResticClientProtocol
from ..core.restic_client import ResticClient
from .backup_command import BackupCommand
from .base import Command
from .forget_prune_command import ForgetPruneCommand
from .report_command import ReportCommand
from .restore_smoke_command import RestoreSmokeCommand


class CommandFactory:
    def __init__(
        self,
        project_root: Path,
        *,
        config_loader: ConfigLoader | None = None,
        clock: ClockProtocol | None = None,
        scanner: FileScannerProtocol | None = None,
        restic_client_factory: Callable[[BackupConfig], ResticClientProtocol] | None = None,
    ) -> None:
        self._config_loader = config_loader or ConfigLoader(project_root)
        self._clock = clock or Clock()
        self._scanner = scanner or FileScanner()
        self._restic_client_factory = restic_client_factory or ResticClient

    def create(self, action: str, env_file: str | None) -> Command:
        config = self._config_loader.load(env_file)
        restic = self._restic_client_factory(config)

        if action == "backup":
            return BackupCommand(config, restic, self._clock)
        if action == "report":
            return ReportCommand(config, restic, self._clock, self._scanner)
        if action == "forget-prune":
            return ForgetPruneCommand(config, restic, self._clock)
        if action == "restore-smoke":
            return RestoreSmokeCommand(config, restic, self._clock)
        raise SystemExit(f"Unsupported action: {action}")
