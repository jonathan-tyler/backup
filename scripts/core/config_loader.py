from __future__ import annotations

import os
from pathlib import Path

from .backup_config import BackupConfig


class ConfigLoader:
    def __init__(self, project_root: Path) -> None:
        self._project_root = project_root

    @property
    def default_env_file(self) -> Path:
        return self._project_root / "config" / "backup.env"

    @property
    def example_env_file(self) -> Path:
        return self._project_root / "config" / "backup.env.example"

    def load(self, env_path: str | None = None) -> BackupConfig:
        env_file = Path(env_path).expanduser() if env_path else self.default_env_file
        if not env_file.is_file():
            raise SystemExit(
                f"Missing env file: {env_file}\n"
                f"Create it from: {self.example_env_file}"
            )

        env_values = self._parse_env_file(env_file)
        self._validate_required(env_values)

        report_dir = Path(env_values["REPORT_DIR"]).expanduser()
        report_dir.mkdir(parents=True, exist_ok=True)

        return BackupConfig(
            project_root=self._project_root,
            env_file=env_file,
            restic_repository=env_values["RESTIC_REPOSITORY"],
            restic_password_command=env_values["RESTIC_PASSWORD_COMMAND"],
            source_notes=Path(env_values["SOURCE_NOTES"]).expanduser(),
            source_repos=Path(env_values["SOURCE_REPOS"]).expanduser(),
            report_dir=report_dir,
            large_file_threshold=env_values.get("LARGE_FILE_THRESHOLD", "2G"),
            hotspot_threshold=int(env_values.get("HOTSPOT_THRESHOLD", "5000")),
            keep_daily=int(env_values.get("KEEP_DAILY", "30")),
            keep_weekly=int(env_values.get("KEEP_WEEKLY", "12")),
            keep_monthly=int(env_values.get("KEEP_MONTHLY", "12")),
        )

    def _parse_env_file(self, env_file: Path) -> dict[str, str]:
        values: dict[str, str] = {}
        for raw_line in env_file.read_text(encoding="utf-8").splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" not in line:
                continue
            key, value = line.split("=", 1)
            cleaned = value.strip().strip('"').strip("'")
            values[key.strip()] = os.path.expandvars(cleaned)
        return values

    def _validate_required(self, env_values: dict[str, str]) -> None:
        required = [
            "RESTIC_REPOSITORY",
            "RESTIC_PASSWORD_COMMAND",
            "SOURCE_NOTES",
            "SOURCE_REPOS",
            "REPORT_DIR",
        ]
        missing = [name for name in required if not env_values.get(name)]
        if missing:
            missing_str = ", ".join(missing)
            raise SystemExit(f"Missing required config values: {missing_str}")
