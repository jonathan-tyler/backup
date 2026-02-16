from __future__ import annotations

import os
from pathlib import Path

import pytest

from scripts.core.config_loader import ConfigLoader


REQUIRED_ENV = """
RESTIC_REPOSITORY={repo}
RESTIC_PASSWORD_COMMAND=printf test-password
SOURCE_NOTES={notes}
SOURCE_REPOS={repos}
REPORT_DIR={report}
""".strip()


def test_load_reads_required_and_defaults(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    env_file = tmp_path / "backup.env"
    env_file.write_text(
        REQUIRED_ENV.format(
            repo=tmp_path / "repo",
            notes=tmp_path / "notes",
            repos=tmp_path / "repos",
            report=tmp_path / "reports",
        ),
        encoding="utf-8",
    )

    loader = ConfigLoader(project_root)
    config = loader.load(str(env_file))

    assert config.restic_repository == str(tmp_path / "repo")
    assert config.source_notes == tmp_path / "notes"
    assert config.source_repos == tmp_path / "repos"
    assert config.report_dir == tmp_path / "reports"
    assert config.large_file_threshold == "2G"
    assert config.hotspot_threshold == 5000
    assert config.keep_daily == 30
    assert config.keep_weekly == 12
    assert config.keep_monthly == 12
    assert config.report_dir.is_dir()


def test_load_expands_env_and_quotes(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    monkeypatch.setenv("TEST_BACKUP_ROOT", str(tmp_path / "dynamic"))

    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "RESTIC_REPOSITORY='${TEST_BACKUP_ROOT}/repo'",
                'RESTIC_PASSWORD_COMMAND="printf secret"',
                "SOURCE_NOTES=${TEST_BACKUP_ROOT}/notes",
                "SOURCE_REPOS=${TEST_BACKUP_ROOT}/repos",
                "REPORT_DIR=${TEST_BACKUP_ROOT}/reports",
                "LARGE_FILE_THRESHOLD=512M",
                "HOTSPOT_THRESHOLD=10",
                "KEEP_DAILY=5",
                "KEEP_WEEKLY=2",
                "KEEP_MONTHLY=1",
            ]
        ),
        encoding="utf-8",
    )

    config = ConfigLoader(project_root).load(str(env_file))

    root = Path(os.environ["TEST_BACKUP_ROOT"])
    assert config.restic_repository == str(root / "repo")
    assert config.source_notes == root / "notes"
    assert config.source_repos == root / "repos"
    assert config.report_dir == root / "reports"
    assert config.large_file_threshold == "512M"
    assert config.hotspot_threshold == 10
    assert config.keep_daily == 5
    assert config.keep_weekly == 2
    assert config.keep_monthly == 1


def test_load_raises_for_missing_env_file(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)

    loader = ConfigLoader(project_root)

    with pytest.raises(SystemExit, match="Missing env file"):
        loader.load(str(tmp_path / "missing.env"))


def test_load_raises_for_missing_required_values(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "RESTIC_REPOSITORY=/tmp/repo\nSOURCE_NOTES=/tmp/notes\n",
        encoding="utf-8",
    )

    with pytest.raises(SystemExit, match="Missing required config values"):
        ConfigLoader(project_root).load(str(env_file))


def test_parse_ignores_comments_and_invalid_lines(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)

    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "# comment",
                "",
                "NOT_A_VAR_LINE",
                "RESTIC_REPOSITORY=/tmp/repo",
                "RESTIC_PASSWORD_COMMAND=printf pass",
                "SOURCE_NOTES=/tmp/notes",
                "SOURCE_REPOS=/tmp/repos",
                "REPORT_DIR=/tmp/reports",
            ]
        ),
        encoding="utf-8",
    )

    config = ConfigLoader(project_root).load(str(env_file))
    assert config.restic_repository == "/tmp/repo"


def test_load_uses_default_env_file_when_omitted(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    config_dir = project_root / "config"
    config_dir.mkdir(parents=True)

    env_file = config_dir / "backup.env"
    env_file.write_text(
        REQUIRED_ENV.format(
            repo=tmp_path / "repo",
            notes=tmp_path / "notes",
            repos=tmp_path / "repos",
            report=tmp_path / "reports",
        ),
        encoding="utf-8",
    )

    config = ConfigLoader(project_root).load()

    assert config.env_file == env_file
