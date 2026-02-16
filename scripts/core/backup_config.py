from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path


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
