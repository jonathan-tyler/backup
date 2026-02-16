from __future__ import annotations

from collections.abc import Iterable, Mapping, Sequence
from pathlib import Path
import subprocess

from scripts.commands.report_command import ReportCommand
from scripts.core.backup_config import BackupConfig
from tests.conftest import FixedClock


class ResticStub:
    def __init__(self, snapshots: Sequence[Mapping[str, object]], diff_output: str = "") -> None:
        self._snapshots = [dict(snapshot) for snapshot in snapshots]
        self._diff_output = diff_output
        self.run_calls: list[list[str]] = []

    def snapshots_json(self) -> list[dict[str, object]]:
        return self._snapshots

    def run(
        self,
        args: Iterable[str],
        *,
        capture_output: bool = False,
        check: bool = True,
    ) -> subprocess.CompletedProcess[str]:
        current = list(args)
        self.run_calls.append(current)
        return subprocess.CompletedProcess(
            ["restic", *current],
            0,
            stdout=self._diff_output,
            stderr="",
        )

    def ensure_initialized(self) -> None:
        raise NotImplementedError


class ScannerStub:
    def __init__(
        self,
        large_files: list[tuple[int, str]] | None = None,
        hotspots: list[tuple[int, str]] | None = None,
    ) -> None:
        self.large_calls: list[tuple[list[str], int]] = []
        self.hotspot_calls: list[tuple[list[str], int]] = []
        self._large_files = large_files if large_files is not None else [
            (1000, "/tmp/a.bin")]
        self._hotspots = hotspots if hotspots is not None else [
            (15, "/tmp/hotspot")]

    def find_large_files(
        self,
        paths: list[Path],
        threshold_bytes: int,
        limit: int = 20,
    ) -> list[tuple[int, str]]:
        self.large_calls.append(
            ([str(path) for path in paths], threshold_bytes))
        return self._large_files

    def find_hotspots(
        self,
        paths: list[Path],
        threshold: int,
        limit: int = 20,
    ) -> list[tuple[int, str]]:
        self.hotspot_calls.append(([str(path) for path in paths], threshold))
        return self._hotspots


def test_report_command_generates_report_and_diff(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    snapshots = [{"id": "snap-1"}, {"id": "snap-2"}]
    diff_output = "+ /new-dir/\n+ /new-dir/file.txt\n- /old-file\nM /changed.txt\n"
    restic = ResticStub(snapshots, diff_output=diff_output)
    scanner = ScannerStub()

    command = ReportCommand(sample_config, restic, fixed_clock, scanner)
    result = command.run()

    assert result == 0
    assert restic.run_calls == [["diff", "snap-1", "snap-2"]]

    report_file = sample_config.report_dir / "report-20260216-010203.txt"
    diff_file = sample_config.report_dir / "diff-20260216-010203.txt"
    assert report_file.is_file()
    assert diff_file.is_file()

    report_text = report_file.read_text(encoding="utf-8")
    assert "Changes summary" in report_text
    assert "- added:   2" in report_text
    assert "- changed: 1" in report_text
    assert "- removed: 1" in report_text
    assert "/new-dir/" in report_text
    assert "Large files over 2G" in report_text
    assert "1000 /tmp/a.bin" in report_text
    assert "Small-file hotspots over 3 files" in report_text
    assert "15 /tmp/hotspot" in report_text


def test_report_command_handles_single_snapshot(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    restic = ResticStub(snapshots=[{"id": "only-one"}], diff_output="")
    scanner = ScannerStub()
    command = ReportCommand(sample_config, restic, fixed_clock, scanner)

    result = command.run()

    assert result == 0
    assert restic.run_calls == []
    report_file = sample_config.report_dir / "report-20260216-010203.txt"
    text = report_file.read_text(encoding="utf-8")
    assert "Not enough snapshots for diff" in text


def test_report_command_writes_none_when_no_dir_or_scanner_results(
    sample_config: BackupConfig,
    fixed_clock: FixedClock,
) -> None:
    snapshots = [{"id": "snap-1"}, {"id": "snap-2"}]
    diff_output = "+ /new-file.txt\n"
    restic = ResticStub(snapshots, diff_output=diff_output)
    scanner = ScannerStub(large_files=[], hotspots=[])

    result = ReportCommand(sample_config, restic, fixed_clock, scanner).run()

    assert result == 0
    report_file = sample_config.report_dir / "report-20260216-010203.txt"
    text = report_file.read_text(encoding="utf-8")
    assert "New directories" in text
    assert "(none)" in text
    assert "Large files over 2G" in text
    assert "Small-file hotspots over 3 files" in text
