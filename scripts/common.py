#!/usr/bin/env python3

from __future__ import annotations

import json
import os
import re
import subprocess
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable


@dataclass
class BackupConfig:
    project_root: Path
    env_file: Path
    restic_repository: str
    restic_password_command: str
    source_notes: Path
    source_repos: Path
    report_dir: Path
    large_file_threshold: str
    hotspot_threshold: int
    keep_daily: int
    keep_weekly: int
    keep_monthly: int


def now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def timestamp() -> str:
    return datetime.now().strftime("%Y%m%d-%H%M%S")


def project_root() -> Path:
    return Path(__file__).resolve().parent.parent


def default_env_file() -> Path:
    return project_root() / "config" / "backup.env"


def _parse_env_file(env_file: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw_line in env_file.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        expanded = os.path.expandvars(value)
        values[key] = expanded
    return values


def load_config(env_path: str | None = None) -> BackupConfig:
    env_file = Path(env_path).expanduser() if env_path else default_env_file()
    if not env_file.is_file():
        raise SystemExit(
            f"Missing env file: {env_file}\n"
            f"Create it from: {project_root() / 'config' / 'backup.env.example'}"
        )

    env_values = _parse_env_file(env_file)

    required = [
        "RESTIC_REPOSITORY",
        "RESTIC_PASSWORD_COMMAND",
        "SOURCE_NOTES",
        "SOURCE_REPOS",
        "REPORT_DIR",
    ]
    missing = [item for item in required if not env_values.get(item)]
    if missing:
        missing_str = ", ".join(missing)
        raise SystemExit(f"Missing required config values: {missing_str}")

    report_dir = Path(env_values["REPORT_DIR"]).expanduser()
    report_dir.mkdir(parents=True, exist_ok=True)

    return BackupConfig(
        project_root=project_root(),
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


def _restic_env(config: BackupConfig) -> dict[str, str]:
    env = os.environ.copy()
    env["RESTIC_REPOSITORY"] = config.restic_repository
    env["RESTIC_PASSWORD_COMMAND"] = config.restic_password_command
    return env


def run_restic(
    config: BackupConfig,
    args: Iterable[str],
    *,
    capture_output: bool = False,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    cmd = ["restic", *args]
    return subprocess.run(
        cmd,
        env=_restic_env(config),
        text=True,
        capture_output=capture_output,
        check=check,
    )


def ensure_repo_initialized(config: BackupConfig) -> None:
    result = run_restic(config, ["snapshots"], capture_output=True, check=False)
    if result.returncode == 0:
        return
    print(f"Initializing repository: {config.restic_repository}")
    run_restic(config, ["init"])


def snapshots_json(config: BackupConfig) -> list[dict]:
    result = run_restic(config, ["snapshots", "--json"], capture_output=True)
    try:
        payload = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Could not parse restic snapshots JSON: {exc}") from exc
    if isinstance(payload, list):
        return payload
    raise SystemExit("Unexpected snapshots payload format from restic")


_SIZE_PATTERN = re.compile(r"^(\d+)([KMGTP]?)(B)?$", re.IGNORECASE)


def parse_size_to_bytes(size_value: str) -> int:
    match = _SIZE_PATTERN.match(size_value.strip())
    if not match:
        raise SystemExit(f"Invalid size format: {size_value}")
    number = int(match.group(1))
    suffix = match.group(2).upper()
    scale = {
        "": 1,
        "K": 1024,
        "M": 1024**2,
        "G": 1024**3,
        "T": 1024**4,
        "P": 1024**5,
    }[suffix]
    return number * scale