from __future__ import annotations

import shutil
import uuid
from pathlib import Path
from typing import Any

import pytest

import scripts.commands.restore_smoke_command as restore_module
from scripts.commands.backup_command import BackupCommand
from scripts.commands.forget_prune_command import ForgetPruneCommand
from scripts.commands.report_command import ReportCommand
from scripts.commands.restore_smoke_command import RestoreSmokeCommand
from scripts.core.config_loader import ConfigLoader
from scripts.core.file_scanner import FileScanner
from scripts.core.restic_client import ResticClient


pytestmark = pytest.mark.integration


class IntegrationClock:
    def __init__(self) -> None:
        self._counter = 0

    def now_iso(self) -> str:
        return "2026-02-16T10:00:00Z"

    def timestamp(self) -> str:
        self._counter += 1
        return f"20260216-1000{self._counter:02d}"


@pytest.fixture
def require_restic() -> None:
    if shutil.which("restic") is None:
        pytest.fail(
            "restic must be installed in the dev container for integration tests")


def _write_env_file(path: Path, values: dict[str, str]) -> None:
    text = "\n".join(f"{key}={value}" for key, value in values.items()) + "\n"
    path.write_text(text, encoding="utf-8")


def _make_user_data_root() -> Path:
    root = Path.home() / ".pytest-integration" / str(uuid.uuid4())
    root.mkdir(parents=True, exist_ok=True)
    return root


def test_backup_report_and_prune_flow(tmp_path: Path, require_restic: None) -> None:
    project_root = Path(__file__).resolve().parents[2]
    data_root = _make_user_data_root()

    try:
        source_notes = data_root / "notes"
        source_repos = data_root / "repos"
        report_dir = data_root / "reports"
        repository_dir = data_root / "restic-repo"
        source_notes.mkdir(parents=True)
        source_repos.mkdir(parents=True)

        (source_notes / "note-1.txt").write_text("first", encoding="utf-8")
        (source_repos / "repo-file-1.txt").write_text("one", encoding="utf-8")

        env_file = data_root / "integration.env"
        _write_env_file(
            env_file,
            {
                "RESTIC_REPOSITORY": str(repository_dir),
                "RESTIC_PASSWORD_COMMAND": "printf integration-password",
                "SOURCE_NOTES": str(source_notes),
                "SOURCE_REPOS": str(source_repos),
                "REPORT_DIR": str(report_dir),
                "LARGE_FILE_THRESHOLD": "1",
                "HOTSPOT_THRESHOLD": "1",
                "KEEP_DAILY": "30",
                "KEEP_WEEKLY": "12",
                "KEEP_MONTHLY": "12",
            },
        )

        config = ConfigLoader(project_root).load(str(env_file))
        clock = IntegrationClock()
        restic = ResticClient(config)

        first_backup = BackupCommand(config, restic, clock).run()
        assert first_backup == 0

        (source_notes / "note-2.txt").write_text("second", encoding="utf-8")
        second_backup = BackupCommand(config, restic, clock).run()
        assert second_backup == 0

        report_result = ReportCommand(
            config, restic, clock, FileScanner()).run()
        assert report_result == 0

        prune_result = ForgetPruneCommand(config, restic, clock).run()
        assert prune_result == 0

        report_files = sorted(report_dir.glob("report-*.txt"))
        diff_files = sorted(report_dir.glob("diff-*.txt"))
        assert report_files
        assert diff_files
    finally:
        shutil.rmtree(data_root, ignore_errors=True)


def test_restore_smoke_restores_file_contents(
    tmp_path: Path,
    require_restic: None,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project_root = Path(__file__).resolve().parents[2]
    data_root = _make_user_data_root()

    try:
        source_notes = data_root / "notes"
        source_repos = data_root / "repos"
        report_dir = data_root / "reports"
        repository_dir = data_root / "restic-repo"
        source_notes.mkdir(parents=True)
        source_repos.mkdir(parents=True)

        note_file = source_notes / "restore-me.txt"
        note_file.write_text("restore-content", encoding="utf-8")

        env_file = data_root / "integration.env"
        _write_env_file(
            env_file,
            {
                "RESTIC_REPOSITORY": str(repository_dir),
                "RESTIC_PASSWORD_COMMAND": "printf integration-password",
                "SOURCE_NOTES": str(source_notes),
                "SOURCE_REPOS": str(source_repos),
                "REPORT_DIR": str(report_dir),
                "LARGE_FILE_THRESHOLD": "2G",
                "HOTSPOT_THRESHOLD": "5000",
                "KEEP_DAILY": "30",
                "KEEP_WEEKLY": "12",
                "KEEP_MONTHLY": "12",
            },
        )

        config = ConfigLoader(project_root).load(str(env_file))
        clock = IntegrationClock()
        restic = ResticClient(config)

        assert BackupCommand(config, restic, clock).run() == 0

        temp_root = data_root / "restore-temp"
        monkeypatch.setattr(restore_module.tempfile,
                            "mkdtemp", lambda: str(temp_root))

        def no_op_rmtree(path: str | Path, ignore_errors: bool = True, *args: Any, **kwargs: Any) -> None:
            _ = path, ignore_errors, args, kwargs

        monkeypatch.setattr(restore_module.shutil, "rmtree", no_op_rmtree)

        result = RestoreSmokeCommand(config, restic, clock).run()
        assert result == 0

        restore_dir = temp_root / "restore"
        restored_matches = list(restore_dir.rglob("restore-me.txt"))
        assert restored_matches, "Expected restored file to exist under restore target"
        restored_text = restored_matches[0].read_text(encoding="utf-8")
        assert restored_text == "restore-content"
    finally:
        shutil.rmtree(data_root, ignore_errors=True)
