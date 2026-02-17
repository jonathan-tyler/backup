from __future__ import annotations

import shutil
import tempfile
from pathlib import Path

from ..core.backup_config import BackupConfig
from ..core.protocols import ClockProtocol, ResticClientProtocol
from .base import Command


class RestoreSmokeCommand(Command):
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
        temp_root = Path(tempfile.mkdtemp())
        restore_dir = temp_root / "restore"
        restore_dir.mkdir(parents=True, exist_ok=True)

        print(f"Running restore smoke test at {self._clock.now_iso()}")
        print(f"Temporary restore path: {restore_dir}")

        try:
            snapshots = self._restic.run(["snapshots"], capture_output=True, check=False)
            if snapshots.returncode != 0:
                print("No snapshots found. Run backup first.")
                return 1

            restore_args = [
                "restore",
                "latest",
                "--target",
                str(restore_dir),
            ]
            for source_path in self._config.source_paths:
                restore_args.extend(["--include", str(source_path)])
            restore_args.append("--verify")

            self._restic.run(
                restore_args
            )

            if not restore_dir.is_dir():
                print("Restore failed: target directory not created.")
                return 1

            restored_files = sum(1 for p in restore_dir.rglob("*") if p.is_file())
            print(f"Restore smoke test complete. Restored files: {restored_files}")
            return 0
        finally:
            shutil.rmtree(temp_root, ignore_errors=True)
