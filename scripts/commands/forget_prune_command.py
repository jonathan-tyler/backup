from __future__ import annotations

from ..core.backup_config import BackupConfig
from ..core.protocols import ClockProtocol, ResticClientProtocol
from .base import Command


class ForgetPruneCommand(Command):
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
        print(f"Running forget/prune at {self._clock.now_iso()}")
        print(
            "Policy: "
            f"daily={self._config.keep_daily} "
            f"weekly={self._config.keep_weekly} "
            f"monthly={self._config.keep_monthly}"
        )

        self._restic.run(
            [
                "forget",
                "--prune",
                "--keep-daily",
                str(self._config.keep_daily),
                "--keep-weekly",
                str(self._config.keep_weekly),
                "--keep-monthly",
                str(self._config.keep_monthly),
            ]
        )

        print(f"Forget/prune completed at {self._clock.now_iso()}")
        return 0
