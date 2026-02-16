#!/usr/bin/env python3

from __future__ import annotations

import argparse
import collections
import os
from pathlib import Path

try:
    from .common import load_config
    from .common import now_iso, parse_size_to_bytes, run_restic, snapshots_json, timestamp
except ImportError:
    from common import load_config
    from common import now_iso, parse_size_to_bytes, run_restic, snapshots_json, timestamp


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Generate backup report")
    parser.add_argument(
        "env_file",
        nargs="?",
        default=None,
        help="Path to env file (default: config/backup.env)",
    )
    return parser


def _scan_large_files(paths: list[Path], threshold_bytes: int, limit: int = 20) -> list[tuple[int, str]]:
    results: list[tuple[int, str]] = []
    for base_path in paths:
        if not base_path.exists():
            continue
        for root, _, files in os.walk(base_path):
            for name in files:
                full_path = Path(root) / name
                try:
                    size_bytes = full_path.stat().st_size
                except OSError:
                    continue
                if size_bytes > threshold_bytes:
                    results.append((size_bytes, str(full_path)))
    results.sort(key=lambda item: item[0], reverse=True)
    return results[:limit]


def _scan_hotspots(paths: list[Path], threshold: int, limit: int = 20) -> list[tuple[int, str]]:
    counts: collections.Counter[str] = collections.Counter()
    for base_path in paths:
        if not base_path.exists():
            continue
        for root, _, files in os.walk(base_path):
            file_count = len(files)
            if file_count > 0:
                counts[str(Path(root))] += file_count
    hotspots = [(count, directory) for directory, count in counts.items() if count > threshold]
    hotspots.sort(key=lambda item: item[0], reverse=True)
    return hotspots[:limit]


def _new_directories(diff_lines: list[str]) -> list[str]:
    return [line[2:] for line in diff_lines if line.startswith("+ ") and line.endswith("/")]


def run(env_file: str | None = None) -> int:
    config = load_config(env_file)
    report_file = config.report_dir / f"report-{timestamp()}.txt"
    diff_file = config.report_dir / f"diff-{timestamp()}.txt"

    snapshots = snapshots_json(config)
    lines: list[str] = []
    lines.append("Backup report")
    lines.append(f"Generated: {now_iso()}")
    lines.append(f"Repository: {config.restic_repository}")
    lines.append("")

    diff_lines: list[str] = []
    if len(snapshots) < 2:
        lines.append("Not enough snapshots for diff (need at least 2).")
    else:
        prev_snap = snapshots[-2]["id"]
        curr_snap = snapshots[-1]["id"]
        lines.append(f"Previous snapshot: {prev_snap}")
        lines.append(f"Current snapshot:  {curr_snap}")
        lines.append("")

        diff = run_restic(config, ["diff", prev_snap, curr_snap], capture_output=True)
        diff_lines = diff.stdout.splitlines()
        diff_file.write_text(diff.stdout, encoding="utf-8")

        added_count = sum(1 for item in diff_lines if item.startswith("+ "))
        removed_count = sum(1 for item in diff_lines if item.startswith("- "))
        changed_count = sum(1 for item in diff_lines if item.startswith(("M ", "U ", "T ")))

        lines.append("Changes summary")
        lines.append(f"- added:   {added_count}")
        lines.append(f"- changed: {changed_count}")
        lines.append(f"- removed: {removed_count}")
        lines.append("")

        lines.append("New directories")
        new_dirs = _new_directories(diff_lines)
        lines.extend(new_dirs if new_dirs else ["(none)"])
        lines.append("")
        lines.append(f"Diff log: {diff_file}")

    threshold_bytes = parse_size_to_bytes(config.large_file_threshold)
    large_files = _scan_large_files(
        [config.source_notes, config.source_repos],
        threshold_bytes,
    )

    lines.append("")
    lines.append(f"Large files over {config.large_file_threshold}")
    if large_files:
        for size_bytes, path in large_files:
            lines.append(f"{size_bytes} {path}")
    else:
        lines.append("(none)")

    hotspots = _scan_hotspots(
        [config.source_notes, config.source_repos],
        config.hotspot_threshold,
    )
    lines.append("")
    lines.append(f"Small-file hotspots over {config.hotspot_threshold} files")
    if hotspots:
        for count, directory in hotspots:
            lines.append(f"{count:>7} {directory}")
    else:
        lines.append("(none)")

    report_file.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"Report written: {report_file}")
    return 0


def main() -> int:
    args = build_parser().parse_args()
    return run(args.env_file)


if __name__ == "__main__":
    raise SystemExit(main())