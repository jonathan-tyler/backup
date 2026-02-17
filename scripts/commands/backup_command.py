from __future__ import annotations

import sys

from ..core.backup_config import BackupConfig
from ..core.protocols import ClockProtocol, ResticClientProtocol
from .base import Command


class BackupCommand(Command):
    def __init__(
        self,
        config: BackupConfig,
        restic: ResticClientProtocol,
        clock: ClockProtocol,
    ) -> None:
        self._config = config
        self._restic = restic
        self._clock = clock

    def run(self) -> int:
        self._restic.ensure_initialized()

        log_file = self._config.report_dir / f"backup-{self._clock.timestamp()}.log"

        print(f"Starting backup at {self._clock.now_iso()}")
        print(f"Repo: {self._config.restic_repository}")
        print(f"Sources include file: {self._config.source_include_file}")

        args = [
            "backup",
            "--files-from",
            str(self._config.source_include_file),
            "--exclude-file",
            str(self._config.exclude_common_file),
            "--exclude-file",
            str(self._config.exclude_set_file),
            "--tag",
            "hot",
            "--tag",
            "notes-repos",
            "--json",
        ]

        with log_file.open("w", encoding="utf-8") as handle:
            handle.write(f"Starting backup at {self._clock.now_iso()}\n")
            handle.write(f"Repo: {self._config.restic_repository}\n")
            handle.write(
                f"Sources include file: {self._config.source_include_file}\n")
            process = self._restic.run(args, capture_output=True, check=False)
            if process.stdout:
                print(process.stdout, end="")
                handle.write(process.stdout)
            if process.stderr:
                print(process.stderr, end="", file=sys.stderr)
                handle.write(process.stderr)
            if process.returncode != 0:
                raise SystemExit(process.returncode)
            handle.write(f"Backup completed at {self._clock.now_iso()}\n")

        print(f"Backup completed at {self._clock.now_iso()}")
        print(f"Log written: {log_file}")
        return 0
