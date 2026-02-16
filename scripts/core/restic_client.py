from __future__ import annotations

import json
import os
import subprocess
from typing import Iterable
from typing import cast

from .backup_config import BackupConfig


class ResticClient:
    def __init__(self, config: BackupConfig) -> None:
        self._config = config

    def run(
        self,
        args: Iterable[str],
        *,
        capture_output: bool = False,
        check: bool = True,
    ) -> subprocess.CompletedProcess[str]:
        cmd = ["restic", *args]
        return subprocess.run(
            cmd,
            env=self._build_env(),
            text=True,
            capture_output=capture_output,
            check=check,
        )

    def ensure_initialized(self) -> None:
        result = self.run(["snapshots"], capture_output=True, check=False)
        if result.returncode == 0:
            return
        print(f"Initializing repository: {self._config.restic_repository}")
        self.run(["init"])

    def snapshots_json(self) -> list[dict[str, object]]:
        result = self.run(["snapshots", "--json"], capture_output=True)
        try:
            payload = json.loads(result.stdout)
        except json.JSONDecodeError as exc:
            raise SystemExit(f"Could not parse restic snapshots JSON: {exc}") from exc
        if isinstance(payload, list):
            return cast(list[dict[str, object]], payload)
        raise SystemExit("Unexpected snapshots payload format from restic")

    def _build_env(self) -> dict[str, str]:
        env = os.environ.copy()
        env["RESTIC_REPOSITORY"] = self._config.restic_repository
        env["RESTIC_PASSWORD_COMMAND"] = self._config.restic_password_command
        return env
