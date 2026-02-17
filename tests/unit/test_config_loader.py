from __future__ import annotations

import os
from pathlib import Path

import pytest

from scripts.core.config_loader import ConfigLoader


REQUIRED_ENV = """
RESTIC_REPOSITORY={repo}
RESTIC_PASSWORD_COMMAND=printf test-password
SOURCE_INCLUDE_FILE={include_file}
EXCLUDE_COMMON_FILE={exclude_common_file}
EXCLUDE_SET_FILE={exclude_set_file}
REPORT_DIR={report}
""".strip()


def _write_include_file(path: Path, sources: list[Path | str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        "\n".join(str(source) for source in sources) + "\n",
        encoding="utf-8",
    )


def _write_exclude_file(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("", encoding="utf-8")


def test_load_reads_required_and_defaults(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    env_file = tmp_path / "backup.env"
    include_file = tmp_path / "sources.include"
    exclude_common_file = tmp_path / "common.exclude"
    exclude_set_file = tmp_path / "notes-repos.exclude"
    _write_include_file(include_file, [tmp_path / "notes", tmp_path / "repos"])
    _write_exclude_file(exclude_common_file)
    _write_exclude_file(exclude_set_file)
    env_file.write_text(
        REQUIRED_ENV.format(
            repo=tmp_path / "repo",
            include_file=include_file,
            exclude_common_file=exclude_common_file,
            exclude_set_file=exclude_set_file,
            report=tmp_path / "reports",
        ),
        encoding="utf-8",
    )

    loader = ConfigLoader(project_root)
    config = loader.load(str(env_file))

    assert config.restic_repository == str(tmp_path / "repo")
    assert config.source_include_file == include_file
    assert config.source_paths == [tmp_path / "notes", tmp_path / "repos"]
    assert config.exclude_common_file == exclude_common_file
    assert config.exclude_set_file == exclude_set_file
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
    include_file = tmp_path / "sources.include"
    exclude_common_file = tmp_path / "common.exclude"
    exclude_set_file = tmp_path / "notes-repos.exclude"
    _write_include_file(
        include_file,
        [
            "${TEST_BACKUP_ROOT}/notes",
            "${TEST_BACKUP_ROOT}/repos",
        ],
    )
    _write_exclude_file(exclude_common_file)
    _write_exclude_file(exclude_set_file)

    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "RESTIC_REPOSITORY='${TEST_BACKUP_ROOT}/repo'",
                'RESTIC_PASSWORD_COMMAND="printf secret"',
                f"SOURCE_INCLUDE_FILE={include_file}",
                f"EXCLUDE_COMMON_FILE={exclude_common_file}",
                f"EXCLUDE_SET_FILE={exclude_set_file}",
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
    assert config.source_include_file == include_file
    assert config.source_paths == [root / "notes", root / "repos"]
    assert config.exclude_common_file == exclude_common_file
    assert config.exclude_set_file == exclude_set_file
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
        "RESTIC_REPOSITORY=/tmp/repo\nREPORT_DIR=/tmp/reports\n",
        encoding="utf-8",
    )

    with pytest.raises(SystemExit, match="Missing required config values"):
        ConfigLoader(project_root).load(str(env_file))


def test_parse_ignores_comments_and_invalid_lines(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)

    env_file = tmp_path / "backup.env"
    include_file = tmp_path / "sources.include"
    exclude_common_file = tmp_path / "common.exclude"
    exclude_set_file = tmp_path / "notes-repos.exclude"
    _write_include_file(include_file, ["/tmp/notes", "/tmp/repos"])
    _write_exclude_file(exclude_common_file)
    _write_exclude_file(exclude_set_file)
    env_file.write_text(
        "\n".join(
            [
                "# comment",
                "",
                "NOT_A_VAR_LINE",
                "RESTIC_REPOSITORY=/tmp/repo",
                "RESTIC_PASSWORD_COMMAND=printf pass",
                f"SOURCE_INCLUDE_FILE={include_file}",
                f"EXCLUDE_COMMON_FILE={exclude_common_file}",
                f"EXCLUDE_SET_FILE={exclude_set_file}",
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

    include_file = tmp_path / "sources.include"
    exclude_common_file = tmp_path / "common.exclude"
    exclude_set_file = tmp_path / "notes-repos.exclude"
    _write_include_file(include_file, [tmp_path / "notes", tmp_path / "repos"])
    _write_exclude_file(exclude_common_file)
    _write_exclude_file(exclude_set_file)

    env_file = config_dir / "backup.env"
    env_file.write_text(
        REQUIRED_ENV.format(
            repo=tmp_path / "repo",
            include_file=include_file,
            exclude_common_file=exclude_common_file,
            exclude_set_file=exclude_set_file,
            report=tmp_path / "reports",
        ),
        encoding="utf-8",
    )

    config = ConfigLoader(project_root).load()

    assert config.env_file == env_file


def test_load_resolves_relative_source_include_file(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    config_dir = project_root / "config"
    config_dir.mkdir(parents=True)
    include_file = config_dir / "sources.include"
    exclude_common_file = config_dir / "common.exclude"
    exclude_set_file = config_dir / "notes-repos.exclude"
    _write_include_file(include_file, ["/tmp/notes", "/tmp/repos"])
    _write_exclude_file(exclude_common_file)
    _write_exclude_file(exclude_set_file)
    env_file = config_dir / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "RESTIC_REPOSITORY=/tmp/repo",
                "RESTIC_PASSWORD_COMMAND=printf pass",
                "SOURCE_INCLUDE_FILE=./sources.include",
                "EXCLUDE_COMMON_FILE=./common.exclude",
                "EXCLUDE_SET_FILE=./notes-repos.exclude",
                "REPORT_DIR=/tmp/reports",
            ]
        ),
        encoding="utf-8",
    )

    config = ConfigLoader(project_root).load(str(env_file))

    assert config.source_include_file == include_file
    assert config.exclude_common_file == exclude_common_file
    assert config.exclude_set_file == exclude_set_file


def test_load_raises_for_missing_source_include_file(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "RESTIC_REPOSITORY=/tmp/repo",
                "RESTIC_PASSWORD_COMMAND=printf pass",
                f"SOURCE_INCLUDE_FILE={tmp_path / 'missing.include'}",
                "REPORT_DIR=/tmp/reports",
            ]
        ),
        encoding="utf-8",
    )

    with pytest.raises(SystemExit, match="Missing source include file"):
        ConfigLoader(project_root).load(str(env_file))


def test_load_raises_for_empty_source_include_file(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    include_file = tmp_path / "sources.include"
    include_file.write_text("\n# comments only\n", encoding="utf-8")
    exclude_common_file = tmp_path / "common.exclude"
    exclude_set_file = tmp_path / "notes-repos.exclude"
    _write_exclude_file(exclude_common_file)
    _write_exclude_file(exclude_set_file)
    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "RESTIC_REPOSITORY=/tmp/repo",
                "RESTIC_PASSWORD_COMMAND=printf pass",
                f"SOURCE_INCLUDE_FILE={include_file}",
                f"EXCLUDE_COMMON_FILE={exclude_common_file}",
                f"EXCLUDE_SET_FILE={exclude_set_file}",
                "REPORT_DIR=/tmp/reports",
            ]
        ),
        encoding="utf-8",
    )

    with pytest.raises(SystemExit, match="No source paths found"):
        ConfigLoader(project_root).load(str(env_file))


def test_load_raises_for_missing_exclude_files(tmp_path: Path) -> None:
    project_root = tmp_path / "project"
    project_root.mkdir(parents=True)
    include_file = tmp_path / "sources.include"
    _write_include_file(include_file, ["/tmp/notes", "/tmp/repos"])
    env_file = tmp_path / "backup.env"
    env_file.write_text(
        "\n".join(
            [
                "RESTIC_REPOSITORY=/tmp/repo",
                "RESTIC_PASSWORD_COMMAND=printf pass",
                f"SOURCE_INCLUDE_FILE={include_file}",
                f"EXCLUDE_COMMON_FILE={tmp_path / 'missing-common.exclude'}",
                f"EXCLUDE_SET_FILE={tmp_path / 'missing-set.exclude'}",
                "REPORT_DIR=/tmp/reports",
            ]
        ),
        encoding="utf-8",
    )

    with pytest.raises(SystemExit, match="Missing exclude file"):
        ConfigLoader(project_root).load(str(env_file))
