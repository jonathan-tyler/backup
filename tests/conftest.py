from __future__ import annotations

from pathlib import Path

import pytest

from scripts.core.backup_config import BackupConfig


class FixedClock:
    def __init__(self) -> None:
        self._timestamp = "20260216-010203"
        self._iso = "2026-02-16T01:02:03Z"

    def now_iso(self) -> str:
        return self._iso

    def timestamp(self) -> str:
        return self._timestamp


@pytest.fixture
def fixed_clock() -> FixedClock:
    return FixedClock()


@pytest.fixture
def sample_config(tmp_path: Path) -> BackupConfig:
    project_root = tmp_path / "project"
    excludes_dir = project_root / "config" / "excludes"
    includes_dir = project_root / "config" / "includes"
    excludes_dir.mkdir(parents=True, exist_ok=True)
    includes_dir.mkdir(parents=True, exist_ok=True)
    (excludes_dir / "common.exclude").write_text("", encoding="utf-8")
    (excludes_dir / "notes-repos.exclude").write_text("", encoding="utf-8")

    source_notes = tmp_path / "notes"
    source_repos = tmp_path / "repos"
    source_paths = [source_notes, source_repos]
    source_include_file = includes_dir / "notes-repos.include"
    exclude_common_file = excludes_dir / "common.exclude"
    exclude_set_file = excludes_dir / "notes-repos.exclude"
    report_dir = tmp_path / "reports"
    source_notes.mkdir(parents=True, exist_ok=True)
    source_repos.mkdir(parents=True, exist_ok=True)
    source_include_file.write_text(
        "\n".join(str(path) for path in source_paths) + "\n",
        encoding="utf-8",
    )
    report_dir.mkdir(parents=True, exist_ok=True)

    return BackupConfig(
        project_root=project_root,
        env_file=tmp_path / "backup.env",
        restic_repository=str(tmp_path / "restic-repo"),
        restic_password_command="printf unit-test-password",
        source_include_file=source_include_file,
        source_paths=source_paths,
        exclude_common_file=exclude_common_file,
        exclude_set_file=exclude_set_file,
        report_dir=report_dir,
        large_file_threshold="2G",
        hotspot_threshold=3,
        keep_daily=30,
        keep_weekly=12,
        keep_monthly=12,
    )
