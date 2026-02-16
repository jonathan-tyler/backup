from __future__ import annotations

from ..core.backup_config import BackupConfig
from ..core.protocols import ClockProtocol, FileScannerProtocol, ResticClientProtocol
from ..core.size_parser import SizeParser
from .base import Command


class ReportCommand(Command):
    def __init__(
        self,
        config: BackupConfig,
        restic: ResticClientProtocol,
        clock: ClockProtocol,
        scanner: FileScannerProtocol,
    ) -> None:
        self._config = config
        self._restic = restic
        self._clock = clock
        self._scanner = scanner

    def run(self) -> int:
        report_file = self._config.report_dir / f"report-{self._clock.timestamp()}.txt"
        diff_file = self._config.report_dir / f"diff-{self._clock.timestamp()}.txt"

        snapshots = self._restic.snapshots_json()
        lines: list[str] = [
            "Backup report",
            f"Generated: {self._clock.now_iso()}",
            f"Repository: {self._config.restic_repository}",
            "",
        ]

        diff_lines: list[str] = []
        if len(snapshots) < 2:
            lines.append("Not enough snapshots for diff (need at least 2).")
        else:
            prev_snap = snapshots[-2]["id"]
            curr_snap = snapshots[-1]["id"]
            lines.extend(
                [
                    f"Previous snapshot: {prev_snap}",
                    f"Current snapshot:  {curr_snap}",
                    "",
                ]
            )

            diff = self._restic.run(["diff", prev_snap, curr_snap], capture_output=True)
            diff_lines = diff.stdout.splitlines()
            diff_file.write_text(diff.stdout, encoding="utf-8")

            added_count = sum(1 for item in diff_lines if item.startswith("+ "))
            removed_count = sum(1 for item in diff_lines if item.startswith("- "))
            changed_count = sum(
                1 for item in diff_lines if item.startswith(("M ", "U ", "T "))
            )

            lines.extend(
                [
                    "Changes summary",
                    f"- added:   {added_count}",
                    f"- changed: {changed_count}",
                    f"- removed: {removed_count}",
                    "",
                    "New directories",
                ]
            )
            lines.extend(self._new_directories(diff_lines) or ["(none)"])
            lines.extend(["", f"Diff log: {diff_file}"])

        threshold = SizeParser.parse_bytes(self._config.large_file_threshold)
        large_files = self._scanner.find_large_files(
            [self._config.source_notes, self._config.source_repos],
            threshold,
        )

        lines.extend(["", f"Large files over {self._config.large_file_threshold}"])
        if large_files:
            lines.extend(f"{size} {path}" for size, path in large_files)
        else:
            lines.append("(none)")

        hotspots = self._scanner.find_hotspots(
            [self._config.source_notes, self._config.source_repos],
            self._config.hotspot_threshold,
        )
        lines.extend(["", f"Small-file hotspots over {self._config.hotspot_threshold} files"])
        if hotspots:
            lines.extend(f"{count:>7} {directory}" for count, directory in hotspots)
        else:
            lines.append("(none)")

        report_file.write_text("\n".join(lines) + "\n", encoding="utf-8")
        print(f"Report written: {report_file}")
        return 0

    def _new_directories(self, diff_lines: list[str]) -> list[str]:
        return [line[2:] for line in diff_lines if line.startswith("+ ") and line.endswith("/")]
